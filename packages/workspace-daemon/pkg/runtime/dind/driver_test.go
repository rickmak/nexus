package dind

import (
	"context"
	"errors"
	"testing"

	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime"
)

type fakeRunner struct {
	calls []call
	err   error
}

type call struct {
	dir  string
	cmd  string
	args []string
}

func (f *fakeRunner) Run(ctx context.Context, dir string, cmd string, args ...string) error {
	f.calls = append(f.calls, call{dir: dir, cmd: cmd, args: args})
	return f.err
}

func (f *fakeRunner) SetError(err error) {
	f.err = err
}

func (f *fakeRunner) Called() (dir, cmd string, args []string) {
	if len(f.calls) == 0 {
		return "", "", nil
	}
	c := f.calls[len(f.calls)-1]
	return c.dir, c.cmd, c.args
}

func (f *fakeRunner) Reset() {
	f.calls = nil
}

func TestDindDriver_Backend(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)
	if d.Backend() != "dind" {
		t.Errorf("expected backend 'dind', got %q", d.Backend())
	}
}

func TestDindDriver_StartCallsDockerComposeUp(t *testing.T) {
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
	if cmd != "docker" {
		t.Errorf("expected cmd 'docker', got %q", cmd)
	}
	if len(args) != 3 || args[0] != "compose" || args[1] != "up" || args[2] != "-d" {
		t.Errorf("expected args [compose up -d], got %v", args)
	}
}

func TestDindDriver_StopCallsDockerComposeDown(t *testing.T) {
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
	if cmd != "docker" {
		t.Errorf("expected cmd 'docker', got %q", cmd)
	}
	if len(args) != 3 || args[0] != "compose" || args[1] != "down" || args[2] != "--remove-orphans" {
		t.Errorf("expected args [compose down --remove-orphans], got %v", args)
	}
}

func TestDindDriver_CreateStoresProjectRoot(t *testing.T) {
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

func TestDindDriver_CreateRequiresProjectRoot(t *testing.T) {
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

func TestDindDriver_RestoreCallsDockerComposeUp(t *testing.T) {
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
	if cmd != "docker" {
		t.Errorf("expected cmd 'docker', got %q", cmd)
	}
	if len(args) != 3 || args[0] != "compose" || args[1] != "up" || args[2] != "-d" {
		t.Errorf("expected args [compose up -d], got %v", args)
	}
}

func TestDindDriver_DestroyCallsDockerComposeDownWithVolume(t *testing.T) {
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
	if cmd != "docker" {
		t.Errorf("expected cmd 'docker', got %q", cmd)
	}
	if len(args) != 4 || args[0] != "compose" || args[1] != "down" || args[2] != "--remove-orphans" || args[3] != "-v" {
		t.Errorf("expected args [compose down --remove-orphans -v], got %v", args)
	}
}

func TestDindDriver_DestroyWithoutCreateIsNoOp(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Destroy(context.Background(), "ws-1")

	if len(runner.calls) != 0 {
		t.Errorf("expected no calls, got %v", runner.calls)
	}
}

func TestDindDriver_StartWithoutCreateIsNoOp(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Start(context.Background(), "ws-1")

	if len(runner.calls) != 0 {
		t.Errorf("expected no calls, got %v", runner.calls)
	}
}

func TestDindDriver_StartErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("docker compose failed"))

	err := d.Start(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Start, got nil")
	}
}

func TestDindDriver_StopErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("docker compose failed"))

	err := d.Stop(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Stop, got nil")
	}
}

func TestDindDriver_RestoreErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("docker compose failed"))

	err := d.Restore(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Restore, got nil")
	}
}

func TestDindDriver_DestroyErrorPropagates(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	_ = d.Create(context.Background(), runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "test-workspace",
		ProjectRoot:   "/projects/ws-1",
	})
	runner.SetError(errors.New("docker compose failed"))

	err := d.Destroy(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error from Destroy, got nil")
	}
}

func TestDindDriver_StopWithoutCreateReturnsError(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	err := d.Stop(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error when stopping non-existent workspace")
	}
}

func TestDindDriver_RestoreWithoutCreateReturnsError(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)

	err := d.Restore(context.Background(), "ws-1")
	if err == nil {
		t.Fatal("expected error when restoring non-existent workspace")
	}
}

func TestDindDriver_DestroyPreventsSubsequentStart(t *testing.T) {
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
