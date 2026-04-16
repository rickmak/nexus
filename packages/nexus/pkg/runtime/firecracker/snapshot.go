package firecracker

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type baseSnapshot struct {
	vmstatePath string
	memFilePath string
	kernelPath  string
	rootfsPath  string
}

func probeReflink(workDirRoot string) bool {
	tmp1 := filepath.Join(workDirRoot, ".reflink-probe-tmp1")
	tmp2 := filepath.Join(workDirRoot, ".reflink-probe-tmp2")

	if err := os.WriteFile(tmp1, []byte("probe"), 0o644); err != nil {
		log.Printf("[firecracker] reflink probe: failed to write temp file: %v", err)
		return false
	}
	defer os.Remove(tmp1)
	defer os.Remove(tmp2)

	cmd := exec.Command("cp", "--reflink=always", tmp1, tmp2)
	if err := cmd.Run(); err != nil {
		log.Printf("[firecracker] XFS reflink not available on %s; fork will use full copy. Format WorkDirRoot as XFS for better performance.", workDirRoot)
		return false
	}

	log.Printf("[firecracker] XFS reflink available on %s, using CoW fork", workDirRoot)
	return true
}

func (m *Manager) cowCopy(src, dst string) error {
	if m.reflinkAvailable {
		cmd := exec.Command("cp", "--reflink=always", src, dst)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("reflink copy %s → %s: %w: %s", src, dst, err, string(out))
		}
		return nil
	}
	return copyFile(src, dst)
}
