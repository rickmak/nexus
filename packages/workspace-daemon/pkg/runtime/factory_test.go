package runtime

import (
	"context"
	"testing"
)

type mockDriver struct {
	backend string
}

func (m *mockDriver) Backend() string { return m.backend }
func (m *mockDriver) Create(ctx context.Context, req CreateRequest) error {
	return nil
}
func (m *mockDriver) Start(ctx context.Context, workspaceID string) error   { return nil }
func (m *mockDriver) Stop(ctx context.Context, workspaceID string) error    { return nil }
func (m *mockDriver) Restore(ctx context.Context, workspaceID string) error { return nil }
func (m *mockDriver) Pause(ctx context.Context, workspaceID string) error   { return nil }
func (m *mockDriver) Resume(ctx context.Context, workspaceID string) error  { return nil }
func (m *mockDriver) Fork(ctx context.Context, workspaceID, childWorkspaceID string) error {
	return nil
}
func (m *mockDriver) Destroy(ctx context.Context, workspaceID string) error { return nil }

func TestSelectDriver_PreferFirst(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.firecracker", Available: true}},
		map[string]Driver{"firecracker": &mockDriver{backend: "firecracker"}},
	)
	driver, err := f.SelectDriver([]string{"firecracker"}, "prefer-first", nil)
	if err != nil {
		t.Fatalf("expected firecracker selection, got %v", err)
	}
	if driver.Backend() != "firecracker" {
		t.Fatalf("expected backend firecracker, got %s", driver.Backend())
	}
}

func TestSelectDriver_PreferFirst_FallsToSecond(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.firecracker", Available: true}},
		map[string]Driver{"firecracker": &mockDriver{backend: "firecracker"}},
	)
	driver, err := f.SelectDriver([]string{"dind", "firecracker"}, "prefer-first", nil)
	if err != nil {
		t.Fatalf("expected firecracker selection (dind not registered), got %v", err)
	}
	if driver.Backend() != "firecracker" {
		t.Fatalf("expected backend firecracker, got %s", driver.Backend())
	}
}

func TestSelectDriver_NoRequiredBackendAvailable(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.firecracker", Available: true}},
		map[string]Driver{"firecracker": &mockDriver{backend: "firecracker"}},
	)
	_, err := f.SelectDriver([]string{"dind"}, "prefer-first", nil)
	if err == nil {
		t.Fatal("expected error when no required backend available")
	}
}

func TestSelectDriver_RequiredCapabilityMissing(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.firecracker", Available: false}},
		map[string]Driver{"firecracker": &mockDriver{backend: "firecracker"}},
	)
	_, err := f.SelectDriver([]string{"firecracker"}, "prefer-first", []string{"runtime.firecracker"})
	if err == nil {
		t.Fatal("expected error when required capability missing")
	}
	if err.Error() != `required capability "runtime.firecracker" is not available` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectDriver_CapabilityAvailable(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.firecracker", Available: true}, {Name: "spotlight.tunnel", Available: true}},
		map[string]Driver{"firecracker": &mockDriver{backend: "firecracker"}},
	)
	driver, err := f.SelectDriver([]string{"firecracker"}, "prefer-first", []string{"runtime.firecracker", "spotlight.tunnel"})
	if err != nil {
		t.Fatalf("expected selection to succeed, got %v", err)
	}
	if driver.Backend() != "firecracker" {
		t.Fatalf("expected backend firecracker, got %s", driver.Backend())
	}
}
