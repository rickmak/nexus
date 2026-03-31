package lxc

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls []call
	err   error
}

type call struct {
	dir  string
	cmd  string
	args []string
}

func (f *fakeRunner) Run(ctx context.Context, dir string, cmd string, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{dir: dir, cmd: cmd, args: args})
	return f.err
}

func (f *fakeRunner) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeRunner) Called() (dir, cmd string, args []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return "", "", nil
	}
	c := f.calls[len(f.calls)-1]
	return c.dir, c.cmd, c.args
}

func (f *fakeRunner) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}

func TestLXCDriver_Backend(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)
	if d.Backend() != "lxc" {
		t.Errorf("expected backend 'lxc', got %q", d.Backend())
	}
}

func TestLXCDriver_StartCallsLxcStart(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	_ = d.Start(context.Background(), "ws-1")

	dir, cmd, args := runner.Called()
	if dir != "/projects/ws-1" {
		t.Errorf("expected dir /projects/ws-1, got %q", dir)
	}
	if cmd != "lxc" {
		t.Errorf("expected cmd 'lxc', got %q", cmd)
	}
	if len(args) != 2 || args[0] != "start" || args[1] != "ws-1" {
		t.Errorf("expected args [start ws-1], got %v", args)
	}
}

func TestLXCDriver_StopCallsLxcStop(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	_ = d.Stop(context.Background(), "ws-1")

	dir, cmd, args := runner.Called()
	if dir != "/projects/ws-1" {
		t.Errorf("expected dir /projects/ws-1, got %q", dir)
	}
	if cmd != "lxc" {
		t.Errorf("expected cmd 'lxc', got %q", cmd)
	}
	if len(args) != 2 || args[0] != "stop" || args[1] != "ws-1" {
		t.Errorf("expected args [stop ws-1], got %v", args)
	}
}

func TestLXCDriver_CreateStoresProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	err := d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = d.Start(context.Background(), "ws-1")
	dir, _, _ := runner.Called()
	if dir != "/projects/ws-1" {
		t.Errorf("expected dir /projects/ws-1, got %q", dir)
	}
}

func TestLXCDriver_CreateRequiresProjectRoot(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	err := d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "",
	})
	if err == nil {
		t.Fatal("expected error when project root is empty")
	}
}

func TestLXCDriver_RestoreCallsLxcStart(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	_ = d.Restore(context.Background(), "ws-1")

	dir, cmd, args := runner.Called()
	if dir != "/projects/ws-1" {
		t.Errorf("expected dir /projects/ws-1, got %q", dir)
	}
	if cmd != "lxc" {
		t.Errorf("expected cmd 'lxc', got %q", cmd)
	}
	if len(args) != 2 || args[0] != "start" || args[1] != "ws-1" {
		t.Errorf("expected args [start ws-1], got %v", args)
	}
}

func TestLXCDriver_DestroyCallsLxcStop(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	_ = d.Destroy(context.Background(), "ws-1")

	dir, cmd, args := runner.Called()
	if dir != "/projects/ws-1" {
		t.Errorf("expected dir /projects/ws-1, got %q", dir)
	}
	if cmd != "lxc" {
		t.Errorf("expected cmd 'lxc', got %q", cmd)
	}
	if len(args) != 2 || args[0] != "stop" || args[1] != "ws-1" {
		t.Errorf("expected args [stop ws-1], got %v", args)
	}
}

func TestLXCDriver_DestroyWithoutCreateIsNoOp(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Destroy(context.Background(), "ws-1")

	if len(runner.calls) != 0 {
		t.Errorf("expected no calls, got %v", runner.calls)
	}
}

func TestLXCDriver_StartWithoutCreateIsNoOp(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Start(context.Background(), "ws-1")

	if len(runner.calls) != 0 {
		t.Errorf("expected no calls, got %v", runner.calls)
	}
}

func TestLXCDriver_StartErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("lxc start failed"))

	err := d.Start(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Start, got nil")
	}
}

func TestLXCDriver_StopErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("lxc stop failed"))

	err := d.Stop(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Stop, got nil")
	}
}

func TestLXCDriver_RestoreErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("lxc start failed"))

	err := d.Restore(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Restore, got nil")
	}
}

func TestLXCDriver_DestroyErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("lxc stop failed"))

	err := d.Destroy(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Destroy, got nil")
	}
}

func TestLXCDriver_StopWithoutCreateReturnsError(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	err := d.Stop(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error when stopping non-existent workspace")
	}
}

func TestLXCDriver_RestoreWithoutCreateReturnsError(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	err := d.Restore(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error when restoring non-existent workspace")
	}
}

func TestLXCDriver_DestroyPreventsSubsequentStart(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	_ = d.Destroy(context.Background(), "ws-1")

	err := d.Start(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error when starting destroyed workspace")
	}
}
