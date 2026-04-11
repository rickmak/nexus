package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rpckit "github.com/inizio/nexus/packages/nexus/pkg/rpcerrors"
	"github.com/inizio/nexus/packages/nexus/pkg/runtime"
	"github.com/inizio/nexus/packages/nexus/pkg/runtime/selection"
	"github.com/inizio/nexus/packages/nexus/pkg/workspacemgr"
)

func RuntimeSelectionFromRepo(repo string) ([]string, []string, error) {
	repoPath := strings.TrimSpace(repo)
	if repoPath == "" {
		return nil, nil, fmt.Errorf("repo is required")
	}
	if !filepath.IsAbs(repoPath) {
		abs, err := filepath.Abs(repoPath)
		if err == nil {
			repoPath = abs
		}
	}

	info, err := os.Stat(repoPath)
	if err != nil || !info.IsDir() {
		return nil, nil, fmt.Errorf("repo must be a local directory with .nexus/workspace.json: %s", repo)
	}

	return []string{"darwin", "linux"}, nil, nil
}

func PrepareCreate(ctx context.Context, spec workspacemgr.CreateSpec, factory *runtime.Factory) (workspacemgr.CreateSpec, *rpckit.RPCError, bool) {
	if factory == nil {
		return spec, nil, false
	}
	requiredBackends, requiredCaps, cfgErr := RuntimeSelectionFromRepo(spec.Repo)
	if cfgErr != nil {
		return workspacemgr.CreateSpec{}, &rpckit.RPCError{Code: rpckit.ErrInvalidParams.Code, Message: cfgErr.Error()}, true
	}
	backend, selErr := selection.SelectBackend(ctx, spec.Repo, requiredBackends, requiredCaps, factory)
	if selErr != nil {
		return workspacemgr.CreateSpec{}, selErr, false
	}
	spec.Backend = backend
	return spec, nil, false
}
