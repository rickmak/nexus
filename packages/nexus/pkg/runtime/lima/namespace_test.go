package lima

import (
	"strings"
	"testing"
)

func TestNamespaceWrapCommand(t *testing.T) {
	inner := "bash -i"
	workspaceID := "ws-abc123"
	wrapped := namespaceWrapCommand(inner, workspaceID)

	if !strings.Contains(wrapped, "unshare --mount") {
		t.Error("wrapped command must use unshare --mount")
	}
	if !strings.Contains(wrapped, "mount --bind") {
		t.Error("wrapped command must bind-mount workspace subdir to /workspace")
	}
	if !strings.Contains(wrapped, "/workspace") {
		t.Error("wrapped command must reference /workspace")
	}
	if !strings.Contains(wrapped, inner) {
		t.Errorf("wrapped command must exec inner command %q", inner)
	}
}
