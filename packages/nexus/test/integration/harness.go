//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// DriverConfig identifies a driver configuration under test.
type DriverConfig struct {
	Backend    string // "firecracker", "lima", "process"
	Mode       string // "dedicated", "pool", "process"
	SkipUnless func(t *testing.T)
}

// AllDrivers enumerates all 5 driver configurations with their hardware guards.
var AllDrivers = []DriverConfig{
	{
		Backend: "firecracker",
		Mode:    "dedicated",
		SkipUnless: func(t *testing.T) {
			t.Helper()
			if _, err := os.Stat("/dev/kvm"); err != nil {
				t.Skip("requires KVM (/dev/kvm not present)")
			}
		},
	},
	{
		Backend: "firecracker",
		Mode:    "pool",
		SkipUnless: func(t *testing.T) {
			t.Helper()
			if _, err := os.Stat("/dev/kvm"); err != nil {
				t.Skip("requires KVM (/dev/kvm not present)")
			}
		},
	},
	{
		Backend: "lima",
		Mode:    "dedicated",
		SkipUnless: func(t *testing.T) {
			t.Helper()
			if !isRunningOnMacOS() {
				t.Skip("requires macOS")
			}
			out, _ := exec.Command("sysctl", "-n", "kern.hv_support").Output()
			if strings.TrimSpace(string(out)) != "1" {
				t.Skip("requires nested virtualization (kern.hv_support=1)")
			}
		},
	},
	{
		Backend: "lima",
		Mode:    "pool",
		SkipUnless: func(t *testing.T) {
			t.Helper()
			if !isRunningOnMacOS() {
				t.Skip("requires macOS")
			}
		},
	},
	{
		Backend: "process",
		Mode:    "process",
		SkipUnless: func(t *testing.T) {
			t.Helper()
			// process/sandbox runs everywhere, never skip
		},
	},
}

// WorkspaceHandle is a reference to a created workspace for test use.
type WorkspaceHandle struct {
	ID      string
	Backend string
	Mode    string
}

// CreateWorkspace creates a workspace using the nexus CLI and returns a handle.
func CreateWorkspace(t *testing.T, cfg DriverConfig, projectRoot string) WorkspaceHandle {
	t.Helper()
	cmd := exec.Command("nexus", "workspace", "create",
		"--backend", cfg.Backend,
		"--mode", cfg.Mode,
		"--project-root", projectRoot,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create workspace: %v\n%s", err, out)
	}
	id := parseWorkspaceID(string(out))
	t.Cleanup(func() { DestroyWorkspace(t, id) })
	return WorkspaceHandle{ID: id, Backend: cfg.Backend, Mode: cfg.Mode}
}

// ExecInWorkspace runs a shell command inside the workspace and returns stdout.
func ExecInWorkspace(t *testing.T, ws WorkspaceHandle, shellCmd string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "nexus", "exec", ws.ID, "--", "sh", "-c", shellCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("exec in workspace %s: %v\n%s", ws.ID, err, out)
	}
	return string(out)
}

// DestroyWorkspace destroys a workspace by ID, ignoring errors (used in cleanup).
func DestroyWorkspace(t *testing.T, id string) {
	t.Helper()
	exec.Command("nexus", "workspace", "destroy", id).Run() //nolint:errcheck
}

// ForkWorkspace forks a workspace and returns the child handle.
func ForkWorkspace(t *testing.T, parent WorkspaceHandle) WorkspaceHandle {
	t.Helper()
	cmd := exec.Command("nexus", "workspace", "fork", parent.ID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fork workspace %s: %v\n%s", parent.ID, err, out)
	}
	childID := parseWorkspaceID(string(out))
	t.Cleanup(func() { DestroyWorkspace(t, childID) })
	return WorkspaceHandle{ID: childID, Backend: parent.Backend, Mode: parent.Mode}
}

func parseWorkspaceID(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if id := strings.TrimPrefix(line, "workspace-id: "); id != line {
			return strings.TrimSpace(id)
		}
	}
	return strings.TrimSpace(lines[len(lines)-1])
}

func isRunningOnMacOS() bool {
	return os.Getenv("RUNNER_OS") == "macOS" || os.Getenv("GOOS") == "darwin"
}
