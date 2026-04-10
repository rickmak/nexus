package limafirecracker

import (
	"context"
	"testing"

	"github.com/inizio/nexus/packages/nexus/pkg/runtime"
)

type stubDriver struct {
	lastReq runtime.CreateRequest
}

func (d *stubDriver) Backend() string { return "seatbelt" }
func (d *stubDriver) Create(_ context.Context, req runtime.CreateRequest) error {
	d.lastReq = req
	return nil
}
func (d *stubDriver) Start(_ context.Context, _ string) error   { return nil }
func (d *stubDriver) Stop(_ context.Context, _ string) error    { return nil }
func (d *stubDriver) Restore(_ context.Context, _ string) error { return nil }
func (d *stubDriver) Pause(_ context.Context, _ string) error   { return nil }
func (d *stubDriver) Resume(_ context.Context, _ string) error  { return nil }
func (d *stubDriver) Fork(_ context.Context, _, _ string) error { return nil }
func (d *stubDriver) Destroy(_ context.Context, _ string) error { return nil }

func TestDriver_Backend(t *testing.T) {
	d := NewDriver(&stubDriver{})
	if got := d.Backend(); got != "firecracker" {
		t.Fatalf("expected firecracker backend, got %q", got)
	}
}

func TestDriver_CreateInjectsLimaInstance(t *testing.T) {
	inner := &stubDriver{}
	d := NewDriver(inner)

	req := runtime.CreateRequest{
		WorkspaceID:   "ws-1",
		WorkspaceName: "alpha",
		ProjectRoot:   "/tmp/repo",
	}
	if err := d.Create(context.Background(), req); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if got := inner.lastReq.Options["lima.instance"]; got != defaultLimaInstance {
		t.Fatalf("expected lima.instance %q, got %q", defaultLimaInstance, got)
	}
}
