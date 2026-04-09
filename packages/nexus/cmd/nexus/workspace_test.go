package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestPrintWorkspaceUsageIncludesStart(t *testing.T) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	os.Stderr = w

	printWorkspaceUsage()

	_ = w.Close()
	os.Stderr = oldStderr

	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read usage output: %v", err)
	}

	got := string(buf)
	if !strings.Contains(got, "start <id>") {
		t.Fatalf("expected usage to include start <id>, got %q", got)
	}
}

func TestRunWorkspaceStartCommandCallsWorkspaceStartRPC(t *testing.T) {
	origEnsure := ensureDaemonFn
	origRPC := daemonRPCFn
	t.Cleanup(func() {
		ensureDaemonFn = origEnsure
		daemonRPCFn = origRPC
	})

	calledMethod := ""
	calledID := ""

	ensureDaemonFn = func() (*websocket.Conn, error) {
		return nil, nil
	}
	daemonRPCFn = func(_ *websocket.Conn, method string, params interface{}, out interface{}) error {
		calledMethod = method
		payload, ok := params.(map[string]any)
		if !ok {
			t.Fatalf("expected map params, got %T", params)
		}
		calledID, _ = payload["id"].(string)
		return nil
	}

	runWorkspaceStartCommand([]string{"ws-123"})

	if calledMethod != "workspace.start" {
		t.Fatalf("expected workspace.start method, got %q", calledMethod)
	}
	if calledID != "ws-123" {
		t.Fatalf("expected workspace id ws-123, got %q", calledID)
	}
}
