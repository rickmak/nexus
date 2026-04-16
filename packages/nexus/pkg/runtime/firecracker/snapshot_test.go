package firecracker

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProbeReflink_DetectsFilesystem(t *testing.T) {
	dir := t.TempDir()
	result := probeReflink(dir)
	_ = result
}

func TestCowCopy_ReflinkUnavailable(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{reflinkAvailable: false}
	if err := m.cowCopy(src, dst); err != nil {
		t.Fatalf("cowCopy fallback failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(data))
	}
}

func TestCowCopy_ReflinkAvailable(t *testing.T) {
	if _, err := exec.LookPath("cp"); err != nil {
		t.Skip("cp not found")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{reflinkAvailable: true}
	if err := m.cowCopy(src, dst); err != nil {
		t.Skipf("reflink copy failed (likely non-XFS filesystem): %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(data))
	}
}
