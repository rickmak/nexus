// Package daemonclient provides helpers for locating, checking liveness of,
// and auto-starting the nexus workspace daemon from the CLI.
package daemonclient

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var daemonProcessCommandLineFn = daemonProcessCommandLine
var daemonProcessStartedAtFn = daemonProcessStartedAt

// RunDir returns the platform directory used for daemon runtime files
// (PID file, token file, log file).
// It respects $XDG_RUNTIME_DIR if set; otherwise uses ~/.config/nexus/run.
func RunDir() (string, error) {
	if d := os.Getenv("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "nexus"), nil
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "nexus", "run"), nil
}

// defaultWorkspaceDir returns the default directory for workspace storage.
// It respects $XDG_DATA_HOME if set; otherwise falls back to ~/.nexus/workspaces.
func defaultWorkspaceDir() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "nexus", "workspaces")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "workspaces")
}

// DefaultDataDir returns the absolute path for daemon persistent data
// ($XDG_DATA_HOME/nexus or ~/.local/share/nexus), matching config.DefaultConfig.
func DefaultDataDir() (string, error) {
	if d := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); d != "" {
		return filepath.Join(d, "nexus"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("daemon data dir: %w", err)
	}
	return filepath.Join(home, ".local", "share", "nexus"), nil
}

// TokenPath returns the legacy path of the per-user daemon token file (runtime dir).
func TokenPath() (string, error) {
	d, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "token"), nil
}

// ReadDaemonToken reads the secret the workspace daemon uses for JWT auth.
// It prefers $XDG_DATA_HOME/nexus/token (or ~/.local/share/nexus/token), then
// falls back to the legacy TokenPath location for transitional compatibility.
func ReadDaemonToken() (string, error) {
	dataDir, err := DefaultDataDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "token")
	data, err := os.ReadFile(path)
	if err == nil {
		tok := strings.TrimSpace(string(data))
		if tok != "" {
			return tok, nil
		}
	}
	legacy, legErr := TokenPath()
	if legErr != nil {
		if err != nil {
			return "", fmt.Errorf("read daemon token: %w", err)
		}
		return "", fmt.Errorf("read daemon token: %w", legErr)
	}
	data, err = os.ReadFile(legacy)
	if err != nil {
		return "", fmt.Errorf("read daemon token: %w", err)
	}
	tok := strings.TrimSpace(string(data))
	if tok == "" {
		return "", fmt.Errorf("read daemon token: empty file")
	}
	return tok, nil
}

// IsRunning returns true if the daemon is accepting requests on the given port.
func IsRunning(port int) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// EnsureRunning starts the daemon in the background if it is not already
// listening on port. It locates the daemon binary next to the current
// executable (or via $PATH), writes a PID file to RunDir(), and polls
// /healthz for up to 5 seconds before returning.
//
// tokenForDaemon, if non-empty, is passed as --token (e.g. NEXUS_DAEMON_TOKEN).
// If empty, the daemon loads or creates the token in its data directory.
// workspaceDir is passed as --workspace-dir to the daemon (empty uses default).
func EnsureRunning(port int, workspaceDir string, tokenForDaemon string) error {
	daemonBin, err := resolveDaemonBin()
	if err != nil {
		return fmt.Errorf("daemonclient: cannot find nexus-daemon binary: %w", err)
	}

	if IsRunning(port) {
		restart, err := shouldRestartRunningDaemon(port, daemonBin)
		if err != nil {
			return fmt.Errorf("daemonclient: inspect running daemon: %w", err)
		}
		if !restart {
			return nil
		}
		if stopErr := stopRunningDaemon(port); stopErr != nil {
			return fmt.Errorf("daemonclient: restart daemon: %w", stopErr)
		}
	}

	runDir, err := RunDir()
	if err != nil {
		return fmt.Errorf("daemonclient: run dir: %w", err)
	}
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return fmt.Errorf("daemonclient: create run dir: %w", err)
	}

	logPath := filepath.Join(runDir, "daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		logFile = io.Discard.(*os.File) // best-effort; fall back to /dev/null below
		logFile, _ = os.Open(os.DevNull)
	}

	if workspaceDir == "" {
		workspaceDir = defaultWorkspaceDir()
	}

	dataDir, err := DefaultDataDir()
	if err != nil {
		return fmt.Errorf("daemonclient: data dir: %w", err)
	}

	args := []string{
		"--port", strconv.Itoa(port),
		"--data-dir", dataDir,
		"--workspace-dir", workspaceDir,
	}
	if tokenForDaemon != "" {
		args = append(args, "--token", tokenForDaemon)
	}
	cmd := exec.Command(daemonBin, args...)
	// Detach from the calling process: new session, no controlling terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("daemonclient: start daemon: %w", err)
	}

	_ = os.WriteFile(pidFilePath(runDir, port), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	_ = os.WriteFile(filepath.Join(runDir, "daemon.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)

	// Detach from our process table so we don't wait for it.
	_ = cmd.Process.Release()

	if err := pollHealthz(port, 5*time.Second); err != nil {
		return err
	}
	if tokenForDaemon == "" {
		if err := pollDaemonToken(5 * time.Second); err != nil {
			return err
		}
	}
	return nil
}

func pollDaemonToken(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 50 * time.Millisecond
	for time.Now().Before(deadline) {
		if _, err := ReadDaemonToken(); err == nil {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("daemonclient: token file not available within %s", timeout)
}

// resolveDaemonBin finds the nexus-daemon binary. It first looks next to the
// current executable (co-installed), then falls back to $PATH.
func resolveDaemonBin() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "nexus-daemon")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	// Fall back to PATH.
	path, err := exec.LookPath("nexus-daemon")
	if err != nil {
		return "", fmt.Errorf("nexus-daemon not found next to %s or in $PATH", exe)
	}
	return path, nil
}

// pollHealthz polls /healthz until it returns 200 or the deadline passes.
func pollHealthz(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 100 * time.Millisecond
	client := &http.Client{Timeout: 400 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/healthz", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(interval)
		if interval < 500*time.Millisecond {
			interval += 100 * time.Millisecond
		}
	}
	return fmt.Errorf("daemonclient: daemon did not become ready within %s", timeout)
}

func shouldRestartRunningDaemon(port int, daemonBin string) (bool, error) {
	pid, err := readRunningDaemonPID(port)
	if err != nil {
		return false, err
	}
	if pid <= 0 {
		return false, nil
	}

	commandLine, err := daemonProcessCommandLineFn(pid)
	if err == nil {
		binName := filepath.Base(strings.TrimSpace(daemonBin))
		if binName != "" && !strings.Contains(commandLine, binName) {
			return true, nil
		}
	}

	binInfo, err := os.Stat(daemonBin)
	if err != nil {
		return false, nil
	}
	startedAt, err := daemonProcessStartedAtFn(pid)
	if err != nil {
		return true, nil
	}
	return binInfo.ModTime().After(startedAt.Add(time.Second)), nil
}

func stopRunningDaemon(port int) error {
	pid, err := readRunningDaemonPID(port)
	if err != nil {
		return err
	}
	if pid <= 0 {
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil && err != os.ErrProcessDone {
		return err
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsRunning(port) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = proc.Signal(syscall.SIGKILL)

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !IsRunning(port) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("existing daemon (pid %d) did not exit", pid)
}

func readRunningDaemonPID(port int) (int, error) {
	runDir, err := RunDir()
	if err != nil {
		return 0, err
	}
	pidData, err := os.ReadFile(pidFilePath(runDir, port))
	if err == nil {
		return strconv.Atoi(strings.TrimSpace(string(pidData)))
	}
	if !os.IsNotExist(err) {
		return 0, err
	}
	pidData, err = os.ReadFile(filepath.Join(runDir, "daemon.pid"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(pidData)))
}

func pidFilePath(runDir string, port int) string {
	return filepath.Join(runDir, fmt.Sprintf("daemon-%d.pid", port))
}

func daemonProcessCommandLine(pid int) (string, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func daemonProcessStartedAt(pid int) (time.Time, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=").Output()
	if err != nil {
		return time.Time{}, err
	}
	startedAtRaw := strings.TrimSpace(string(out))
	if startedAtRaw == "" {
		return time.Time{}, fmt.Errorf("empty process start time")
	}
	startedAt, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", startedAtRaw, time.Local)
	if err != nil {
		startedAt, err = time.ParseInLocation("Mon Jan _2 15:04:05 2006", startedAtRaw, time.Local)
		if err != nil {
			return time.Time{}, err
		}
	}
	return startedAt, nil
}
