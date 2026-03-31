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
func (m *mockDriver) Destroy(ctx context.Context, workspaceID string) error { return nil }

func TestSelectDriver_PreferFirst(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.dind", Available: true}},
		map[string]Driver{"dind": &mockDriver{backend: "dind"}, "lxc": &mockDriver{backend: "lxc"}},
	)
	driver, err := f.SelectDriver([]string{"dind", "lxc"}, "prefer-first", nil)
	if err != nil {
		t.Fatalf("expected dind selection, got %v", err)
	}
	if driver.Backend() != "dind" {
		t.Fatalf("expected backend dind, got %s", driver.Backend())
	}
}

func TestSelectDriver_PreferFirst_FallsToSecond(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.lxc", Available: true}},
		map[string]Driver{"dind": &mockDriver{backend: "dind"}, "lxc": &mockDriver{backend: "lxc"}},
	)
	driver, err := f.SelectDriver([]string{"dind", "lxc"}, "prefer-first", nil)
	if err != nil {
		t.Fatalf("expected lxc selection (dind not registered), got %v", err)
	}
	if driver.Backend() != "lxc" {
		t.Fatalf("expected backend lxc, got %s", driver.Backend())
	}
}

func TestSelectDriver_NoRequiredBackendAvailable(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.dind", Available: true}},
		map[string]Driver{"lxc": &mockDriver{backend: "lxc"}},
	)
	_, err := f.SelectDriver([]string{"dind"}, "prefer-first", nil)
	if err == nil {
		t.Fatal("expected error when no required backend available")
	}
}

func TestSelectDriver_RequiredCapabilityMissing(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.dind", Available: false}},
		map[string]Driver{"dind": &mockDriver{backend: "dind"}},
	)
	_, err := f.SelectDriver([]string{"dind"}, "prefer-first", []string{"runtime.dind"})
	if err == nil {
		t.Fatal("expected error when required capability missing")
	}
	if err.Error() != `required capability "runtime.dind" is not available` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectDriver_CapabilityAvailable(t *testing.T) {
	f := NewFactory(
		[]Capability{{Name: "runtime.dind", Available: true}, {Name: "spotlight.tunnel", Available: true}},
		map[string]Driver{"dind": &mockDriver{backend: "dind"}},
	)
	driver, err := f.SelectDriver([]string{"dind"}, "prefer-first", []string{"runtime.dind", "spotlight.tunnel"})
	if err != nil {
		t.Fatalf("expected selection to succeed, got %v", err)
	}
	if driver.Backend() != "dind" {
		t.Fatalf("expected backend dind, got %s", driver.Backend())
	}
}
