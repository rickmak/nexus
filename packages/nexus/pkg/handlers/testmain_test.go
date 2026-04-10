package handlers

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"
)

const (
	defaultTestFirecrackerVersion = "1.15.1"
	defaultTestLimaVersion        = "2.1.1"
)

func TestMain(m *testing.M) {
	cliPath, toolDir, err := prepareHandlerTestCLIAndPreflightPATH()
	if err != nil {
		fmt.Fprintf(os.Stderr, "handlers TestMain: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	if toolDir != "" {
		_ = os.RemoveAll(toolDir)
	}
	if cliPath != "" {
		_ = os.Remove(cliPath)
	}
	os.Exit(code)
}

func prepareHandlerTestCLIAndPreflightPATH() (cliPath string, toolDir string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	if p := strings.TrimSpace(os.Getenv("NEXUS_CLI_PATH")); p != "" {
		st, err := os.Stat(p)
		if err != nil {
			return "", "", fmt.Errorf("NEXUS_CLI_PATH: %w", err)
		}
		if st.IsDir() {
			return "", "", fmt.Errorf("NEXUS_CLI_PATH is a directory")
		}
	} else {
		modRoot, err := goModDir(wd)
		if err != nil {
			return "", "", err
		}
		cliPath = filepath.Join(os.TempDir(), fmt.Sprintf("nexus-handlers-cli-%d", os.Getpid()))
		build := exec.Command("go", "build", "-o", cliPath, "./cmd/nexus")
		build.Dir = modRoot
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			return "", "", fmt.Errorf("build nexus CLI: %w", err)
		}
		os.Setenv("NEXUS_CLI_PATH", cliPath)
	}

	needTools := false
	switch goruntime.GOOS {
	case "linux":
		if _, err := exec.LookPath("firecracker"); err != nil {
			needTools = true
		}
	case "darwin":
		if _, err := exec.LookPath("limactl"); err != nil {
			needTools = true
		}
	}
	if !needTools {
		return cliPath, "", nil
	}

	toolDir, err = os.MkdirTemp("", "nexus-handlers-tools-*")
	if err != nil {
		return cliPath, "", err
	}

	switch goruntime.GOOS {
	case "linux":
		if _, err := exec.LookPath("firecracker"); err != nil {
			fmt.Fprintf(os.Stderr, "handlers test: downloading Firecracker for preflight (not on PATH)...\n")
			if err := installFirecrackerRelease(toolDir); err != nil {
				return cliPath, toolDir, err
			}
			prependPATH(filepath.Join(toolDir, "bin"))
		}
	case "darwin":
		if _, err := exec.LookPath("limactl"); err != nil {
			fmt.Fprintf(os.Stderr, "handlers test: downloading Lima for preflight (limactl not on PATH)...\n")
			if err := installLimaDarwinRelease(toolDir); err != nil {
				return cliPath, toolDir, err
			}
			prependPATH(filepath.Join(toolDir, "lima", "bin"))
		}
	}

	return cliPath, toolDir, nil
}

func prependPATH(dir string) {
	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func testVersion(envKey, fallback string) string {
	s := strings.TrimSpace(os.Getenv(envKey))
	if s != "" {
		return strings.TrimPrefix(s, "v")
	}
	return fallback
}

func installFirecrackerRelease(toolDir string) error {
	ver := testVersion("NEXUS_TEST_FIRECRACKER_VERSION", defaultTestFirecrackerVersion)
	var fcSuffix string
	switch goruntime.GOARCH {
	case "amd64":
		fcSuffix = "x86_64"
	case "arm64":
		fcSuffix = "aarch64"
	default:
		return fmt.Errorf("unsupported GOARCH for Firecracker download: %s", goruntime.GOARCH)
	}

	asset := fmt.Sprintf("firecracker-v%s-%s.tgz", ver, fcSuffix)
	url := fmt.Sprintf(
		"https://github.com/firecracker-microvm/firecracker/releases/download/v%s/%s",
		ver, asset,
	)

	staging := filepath.Join(toolDir, "firecracker-staging")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	if err := downloadExtractTarGz(url, staging); err != nil {
		return fmt.Errorf("firecracker: %w", err)
	}

	binPath, err := findFirecrackerBinary(staging)
	if err != nil {
		return err
	}

	binDir := filepath.Join(toolDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(binDir, "firecracker")
	if err := copyFile(dst, binPath, 0o755); err != nil {
		return err
	}
	return nil
}

func findFirecrackerBinary(root string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := d.Name()
		if strings.HasPrefix(base, "firecracker-v") && !strings.Contains(base, ".debug") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("firecracker binary not found under %s", root)
	}
	return found, nil
}

func installLimaDarwinRelease(toolDir string) error {
	if goruntime.GOOS != "darwin" {
		return nil
	}
	ver := testVersion("NEXUS_TEST_LIMA_VERSION", defaultTestLimaVersion)
	var darwinArch string
	switch goruntime.GOARCH {
	case "amd64":
		darwinArch = "x86_64"
	case "arm64":
		darwinArch = "arm64"
	default:
		return fmt.Errorf("unsupported GOARCH for Lima download: %s", goruntime.GOARCH)
	}

	asset := fmt.Sprintf("lima-%s-Darwin-%s.tar.gz", ver, darwinArch)
	url := fmt.Sprintf(
		"https://github.com/lima-vm/lima/releases/download/v%s/%s",
		ver, asset,
	)

	dest := filepath.Join(toolDir, "lima")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := downloadExtractTarGz(url, dest); err != nil {
		return fmt.Errorf("lima: %w", err)
	}
	limactl := filepath.Join(dest, "bin", "limactl")
	if _, err := os.Stat(limactl); err != nil {
		return fmt.Errorf("limactl not found after extract: %w", err)
	}
	return nil
}

func copyFile(dst, src string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func downloadExtractTarGz(url string, destDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(hdr.Name, "./")
		if name == "" || name == "." {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(name))
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), cleanDest) && filepath.Clean(target) != filepath.Clean(destDir) {
			return fmt.Errorf("unsafe tar path %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, fileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

func fileMode(m int64) os.FileMode {
	return os.FileMode(m) & 0o777
}

func goModDir(start string) (string, error) {
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Dir = start
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	mod := strings.TrimSpace(string(out))
	if mod == "" || mod == "/dev/null" {
		return "", fmt.Errorf("not inside a Go module (cwd=%s)", start)
	}
	return filepath.Dir(mod), nil
}
