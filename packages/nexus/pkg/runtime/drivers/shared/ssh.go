package shared

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
)

func isSocketPath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSocket != 0
}

func daemonSSHAuthSock() string {
	if sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")); isSocketPath(sock) {
		return sock
	}
	if out, err := exec.Command("launchctl", "getenv", "SSH_AUTH_SOCK").Output(); err == nil {
		if sock := strings.TrimSpace(string(out)); isSocketPath(sock) {
			return sock
		}
	}
	return ""
}

func withSSHAuthSockEnv(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	sock := daemonSSHAuthSock()
	if sock == "" {
		return
	}
	cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK="+sock)
}

// limaHome returns the Lima home directory, respecting $LIMA_HOME.
func limaHome() (string, error) {
	if d := os.Getenv("LIMA_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".lima"), nil
}

// limaSSHConfigPath returns ~/.lima/INSTANCE/ssh.config (Lima-generated).
func limaSSHConfigPath(instanceName string) (string, error) {
	dir, err := limaHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, instanceName, "ssh.config"), nil
}

// nexusSSHConfigPath returns a nexus-managed SSH config for instanceName.
// It rewrites Lima's "Host lima-INSTANCE" to plain "Host INSTANCE" so all
// ssh invocations use the bare instance name (e.g. "nexus") rather than
// "lima-nexus".  The file is regenerated each call so port changes after a
// Lima restart are always picked up.
func nexusSSHConfigPath(instanceName string) (string, error) {
	limaCfg, err := limaSSHConfigPath(instanceName)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(limaCfg)
	if err != nil {
		return "", fmt.Errorf("lima ssh.config not found for %q: %w", instanceName, err)
	}

	// Replace "Host lima-INSTANCE" → "Host INSTANCE" (keeps the same block).
	rewritten := strings.ReplaceAll(string(raw), "Host lima-"+instanceName, "Host "+instanceName)

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".nexus", "ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	out := filepath.Join(dir, instanceName+".ssh.config")
	if err := os.WriteFile(out, []byte(rewritten), 0o600); err != nil {
		return "", err
	}
	return out, nil
}

// DirectSSHInteractiveArgs returns the ssh(1) argument slice that opens an
// interactive login shell inside the Lima instance, starting in workdir.
//
// Equivalent to what `limactl shell --reconnect --workdir WORKDIR INSTANCE`
// does internally, but without the limactl wrapper overhead or log noise.
func DirectSSHInteractiveArgs(instanceName, workdir, shell string) ([]string, error) {
	cfgPath, err := nexusSSHConfigPath(instanceName)
	if err != nil {
		return nil, err
	}

	sh := strings.TrimSpace(shell)
	if sh == "" {
		sh = "bash"
	}

	ensureSSHAgent := strings.Join([]string{
		`if [ -z "${SSH_AUTH_SOCK:-}" ]; then`,
		`  export SSH_AUTH_SOCK="$HOME/.ssh/nexus-agent.sock";`,
		`fi;`,
		`if [ ! -S "${SSH_AUTH_SOCK:-}" ]; then`,
		`  mkdir -p "$HOME/.ssh"; chmod 700 "$HOME/.ssh" 2>/dev/null || true;`,
		`  rm -f "$SSH_AUTH_SOCK"; ssh-agent -a "$SSH_AUTH_SOCK" >/tmp/nexus-ssh-agent.log 2>&1 || true;`,
		`fi;`,
		`if [ -S "${SSH_AUTH_SOCK:-}" ] && command -v ssh-add >/dev/null 2>&1; then`,
		`  ssh-add -q "$HOME/.ssh/id_ed25519" "$HOME/.ssh/id_rsa" "$HOME/.ssh/id_ecdsa" >/dev/null 2>&1 || true;`,
		`fi`,
	}, " ")
	interactiveInnerCmd := "exec " + sh + " -i"
	wd := strings.TrimSpace(workdir)
	if wd != "" {
		// Make the working directory a launch-time invariant rather than a shell
		// startup side effect. Run the requested shell once as a login shell so it
		// can initialize the environment, then exec a non-login interactive shell
		// from the requested cwd. If cd fails, SSH exits instead of silently
		// landing in ~.
		interactiveInnerCmd = "cd " + ShellQuote(wd) + " && " + interactiveInnerCmd
	}

	// Ensure new shell processes pick up docker group privileges immediately
	// when docker is installed and accessible through the docker group.
	innerCmd := ensureSSHAgent + "; if command -v docker >/dev/null 2>&1 && ! docker info >/dev/null 2>&1 && getent group docker >/dev/null 2>&1; then exec sg docker -c " + ShellQuote(interactiveInnerCmd) + "; fi; " + interactiveInnerCmd
	fullCmd := sh + " -l -c " + ShellQuote(innerCmd)
	return []string{
		"-F", cfgPath,
		"-o", "LogLevel=ERROR",
		"-o", "ForwardAgent=yes",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		"-A",
		"-t", // force remote PTY allocation; required when a command is given
		instanceName,
		fullCmd,
	}, nil
}

// DirectSSHScriptArgs returns the ssh(1) argument slice that runs a
// non-interactive shell script inside the Lima instance.
//
// Equivalent to `limactl shell INSTANCE -- sh -lc SCRIPT`.
func DirectSSHScriptArgs(instanceName, script string) ([]string, error) {
	cfgPath, err := nexusSSHConfigPath(instanceName)
	if err != nil {
		return nil, err
	}

	return []string{
		"-F", cfgPath,
		"-o", "LogLevel=ERROR",
		"-o", "ControlMaster=no",
		"-o", "ControlPath=none",
		instanceName,
		"--",
		"sh", "-lc", script,
	}, nil
}

// DirectSSHScript runs script inside instanceName via direct SSH and returns
// combined stdout+stderr. Drop-in replacement for LimactlShellScript.
func DirectSSHScript(ctx context.Context, instanceName, script string) ([]byte, error) {
	args, err := DirectSSHScriptArgs(instanceName, script)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "ssh", args...)
	withSSHAuthSockEnv(cmd)
	return cmd.CombinedOutput()
}

// DirectSSHScriptWithInput runs script inside instanceName and streams input to
// the remote process stdin. Useful for large payload delivery without embedding
// payload bytes into SSH argv.
func DirectSSHScriptWithInput(ctx context.Context, instanceName, script, input string) ([]byte, error) {
	args, err := DirectSSHScriptArgs(instanceName, script)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "ssh", args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	return cmd.CombinedOutput()
}

// TryDirectSSHScript runs script in the first candidate Lima instance whose
// ssh.config exists and the SSH command succeeds.  Use this wherever a single
// instance name is not guaranteed (e.g. after a daemon restart where the
// stored instance name may be a legacy alias).
func TryDirectSSHScript(ctx context.Context, candidates []string, script string) ([]byte, error) {
	var lastErr error
	for _, candidate := range candidates {
		out, err := DirectSSHScript(ctx, candidate, script)
		if err == nil {
			return out, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return nil, fmt.Errorf("no lima instance candidates available")
	}
	return nil, lastErr
}

// TryDirectSSHShellPTY attempts to start an interactive PTY shell in each
// Lima instance candidate via direct SSH (no limactl wrapper).
// It is a drop-in replacement for TrySSHShellPTY.
func TryDirectSSHShellPTY(ctx context.Context, opt TrySSHPTYOptions) (*exec.Cmd, *os.File, error) {
	launchShell := NormalizeLaunchShell(opt.LaunchShell)
	workdir := strings.TrimSpace(opt.Workdir)
	var lastErr error

	for _, candidate := range opt.Candidates {
		if opt.BeforeEachCandidate != nil {
			if err := opt.BeforeEachCandidate(ctx, candidate); err != nil {
				lastErr = err
				continue
			}
		}

		args, err := DirectSSHInteractiveArgs(candidate, workdir, launchShell)
		if err != nil {
			lastErr = err
			continue
		}

		cmd := exec.CommandContext(ctx, "ssh", args...)
		withSSHAuthSockEnv(cmd)
		ptmx, ptyErr := opt.PtyStart(cmd, &pty.Winsize{Rows: 30, Cols: 120})
		if ptyErr == nil {
			return cmd, ptmx, nil
		}
		lastErr = ptyErr
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no lima instance candidates available")
	}
	prefix := strings.TrimSpace(opt.ErrPrefix)
	if prefix != "" {
		return nil, nil, fmt.Errorf("%s: %w", prefix, lastErr)
	}
	return nil, nil, fmt.Errorf("lima ssh shell start failed: %w", lastErr)
}
