package lxc

import (
	"context"
	"errors"
	"sync"

	"github.com/nexus/nexus/packages/workspace-daemon/pkg/runtime"
)

type CommandRunner interface {
	Run(ctx context.Context, dir string, cmd string, args ...string) error
}

type Driver struct {
	runner       CommandRunner
	projectRoots map[string]string
	mu           sync.RWMutex
}

func NewDriver(runner CommandRunner) *Driver {
	return &Driver{
		runner:       runner,
		projectRoots: make(map[string]string),
	}
}

func (d *Driver) Backend() string {
	return "lxc"
}

func (d *Driver) workspaceDir(workspaceID string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if dir, ok := d.projectRoots[workspaceID]; ok {
		return dir
	}
	return ""
}

func (d *Driver) Create(ctx context.Context, req runtime.CreateRequest) error {
	if req.ProjectRoot == "" {
		return errors.New("project root is required")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.projectRoots[req.WorkspaceID] = req.ProjectRoot
	return nil
}

func (d *Driver) Start(ctx context.Context, workspaceID string) error {
	dir := d.workspaceDir(workspaceID)
	if dir == "" {
		return errors.New("workspace not created or project root unknown")
	}
	return d.runner.Run(ctx, dir, "lxc", "start", workspaceID)
}

func (d *Driver) Stop(ctx context.Context, workspaceID string) error {
	dir := d.workspaceDir(workspaceID)
	if dir == "" {
		return errors.New("workspace not created or project root unknown")
	}
	return d.runner.Run(ctx, dir, "lxc", "stop", workspaceID)
}

func (d *Driver) Restore(ctx context.Context, workspaceID string) error {
	dir := d.workspaceDir(workspaceID)
	if dir == "" {
		return errors.New("workspace not created or project root unknown")
	}
	return d.runner.Run(ctx, dir, "lxc", "start", workspaceID)
}

func (d *Driver) Destroy(ctx context.Context, workspaceID string) error {
	d.mu.Lock()
	dir, ok := d.projectRoots[workspaceID]
	if !ok {
		d.mu.Unlock()
		return nil
	}
	delete(d.projectRoots, workspaceID)
	d.mu.Unlock()
	return d.runner.Run(ctx, dir, "lxc", "stop", workspaceID)
}
