package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectSSHInteractiveArgsUsesExplicitCdWrapper(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("LIMA_HOME", filepath.Join(tmp, ".lima"))

	limaDir := filepath.Join(tmp, ".lima", "nexus")
	if err := os.MkdirAll(limaDir, 0o755); err != nil {
		t.Fatalf("mkdir lima dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(limaDir, "ssh.config"), []byte("Host lima-nexus\n  HostName 127.0.0.1\n"), 0o600); err != nil {
		t.Fatalf("write ssh.config: %v", err)
	}

	args, err := DirectSSHInteractiveArgs("nexus", "/nexus/ws/ws-123", "bash")
	if err != nil {
		t.Fatalf("DirectSSHInteractiveArgs: %v", err)
	}
	if len(args) < 7 {
		t.Fatalf("expected ssh args to include command wrapper, got %d: %v", len(args), args)
	}
	commandArg := args[len(args)-1]
	if strings.Contains(commandArg, "PROMPT_COMMAND") {
		t.Fatalf("expected explicit cd wrapper, got legacy PROMPT_COMMAND command: %q", commandArg)
	}
	if !strings.Contains(commandArg, "/nexus/ws/ws-123") || !strings.Contains(commandArg, "exec bash -i") {
		t.Fatalf("expected explicit cd wrapper in ssh command, got %q", commandArg)
	}
	hasNoMux := false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-o" && args[i+1] == "ControlMaster=no" {
			hasNoMux = true
			break
		}
	}
	if !hasNoMux {
		t.Fatalf("expected ssh args to disable multiplexing, got %v", args)
	}
}
