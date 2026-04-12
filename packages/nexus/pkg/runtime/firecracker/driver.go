package firecracker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inizio/nexus/packages/nexus/pkg/runtime"
	"github.com/inizio/nexus/packages/nexus/pkg/secrets/discovery"
	"github.com/inizio/nexus/packages/nexus/pkg/secrets/server"
	"github.com/inizio/nexus/packages/nexus/pkg/secrets/vending"
	"github.com/mdlayher/vsock"
)

var _ runtime.Driver = (*Driver)(nil)

type CommandRunner interface {
	Run(ctx context.Context, dir string, cmd string, args ...string) error
}

// ManagerInterface defines the interface for VM lifecycle management
type ManagerInterface interface {
	Spawn(ctx context.Context, spec SpawnSpec) (*Instance, error)
	Stop(ctx context.Context, workspaceID string) error
	Get(workspaceID string) (*Instance, error)
	GrowWorkspace(ctx context.Context, workspaceID string, newSizeBytes int64) error
}

type Driver struct {
	runner         CommandRunner
	manager        ManagerInterface
	projectRoots   map[string]string
	agents         map[string]*AgentClient
	vendingServers map[string]*server.Server
	mu             sync.RWMutex
}


func (d *Driver) AgentConn(ctx context.Context, workspaceID string) (net.Conn, error) {
	if d.manager == nil {
		return nil, errors.New("manager is required for firecracker driver")
	}

	inst, err := d.manager.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace instance lookup failed: %w", err)
	}

	if inst == nil || inst.CID == 0 {
		return nil, errors.New("workspace instance has no guest CID")
	}

	conn, err := vsock.Dial(inst.CID, DefaultAgentVSockPort, nil)
	if err != nil {
		return nil, fmt.Errorf("vsock dial failed: %w", err)
	}

	return conn, nil
}

type Option func(*Driver)

func WithManager(manager ManagerInterface) Option {
	return func(d *Driver) {
		d.manager = manager
	}
}

func NewDriver(runner CommandRunner, opts ...Option) *Driver {
	d := &Driver{
		runner:         runner,
		projectRoots:   make(map[string]string),
		agents:         make(map[string]*AgentClient),
		vendingServers: make(map[string]*server.Server),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Driver) Backend() string {
	return "firecracker"
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

	if d.manager == nil {
		return errors.New("manager is required for firecracker driver")
	}

	memMiB := 1024
	if req.Options != nil {
		if memStr, ok := req.Options["mem_mib"]; ok && memStr != "" {
			if val, err := strconv.Atoi(memStr); err == nil {
				memMiB = val
			}
		}
	}

	spec := SpawnSpec{
		WorkspaceID: req.WorkspaceID,
		ProjectRoot: req.ProjectRoot,
		MemoryMiB:   memMiB,
		VCPUs:       1,
	}

	inst, err := d.manager.Spawn(ctx, spec)
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.projectRoots[req.WorkspaceID] = req.ProjectRoot
	d.mu.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[secrets] Warning: user home dir: %v", err)
	} else {
		configs, discoverErr := discovery.Discover(homeDir)
		if discoverErr != nil {
			log.Printf("[secrets] Warning: credential discovery failed: %v", discoverErr)
		} else if len(configs) > 0 {
			svc := vending.NewService(configs)
			vendServer := server.New(svc, VendingVSockPort)
			if startErr := vendServer.Start(); startErr != nil {
				log.Printf("[secrets] Warning: failed to start vending: %v", startErr)
			} else {
				log.Printf("[secrets] Started vending for %d providers", len(configs))
				d.mu.Lock()
				d.vendingServers[req.WorkspaceID] = vendServer
				d.mu.Unlock()
			}
		}
	}

	// TODO: Connect to agent via vsock when AgentClient dial is implemented
	_ = inst

	return nil
}

func (d *Driver) GrowWorkspace(ctx context.Context, workspaceID string, newSizeBytes int64) error {
	if d.manager == nil {
		return errors.New("manager is required for firecracker driver")
	}

	if err := d.manager.GrowWorkspace(ctx, workspaceID, newSizeBytes); err != nil {
		return fmt.Errorf("host-side grow failed: %w", err)
	}

	conn, err := d.AgentConn(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("agent connect for disk.grow: %w", err)
	}
	defer conn.Close()

	client := NewAgentClient(conn)
	req := ExecRequest{
		ID:      fmt.Sprintf("disk-grow-%d", time.Now().UnixNano()),
		Type:    "disk.grow",
		Command: "",
	}
	result, err := client.Exec(ctx, req)
	if err != nil {
		return fmt.Errorf("disk.grow exec failed: %w", err)
	}
	if result.ExitCode != 0 {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			detail = fmt.Sprintf("exit code %d", result.ExitCode)
		}
		return fmt.Errorf("resize2fs failed: %s", detail)
	}

	return nil
}

func (d *Driver) Start(ctx context.Context, workspaceID string) error {
	// Native firecracker VMs start immediately after Spawn
	// This is a no-op for the native implementation
	return nil
}

func (d *Driver) Stop(ctx context.Context, workspaceID string) error {
	if d.manager == nil {
		return errors.New("manager is required for firecracker driver")
	}

	d.mu.Lock()
	if vendServer, ok := d.vendingServers[workspaceID]; ok {
		_ = vendServer.Stop()
		delete(d.vendingServers, workspaceID)
	}
	delete(d.agents, workspaceID)
	d.mu.Unlock()

	return d.manager.Stop(ctx, workspaceID)
}

func (d *Driver) Restore(ctx context.Context, workspaceID string) error {
	// Native firecracker doesn't support restore in this cutover
	return errors.New("restore not supported in native firecracker driver")
}

func (d *Driver) Pause(ctx context.Context, workspaceID string) error {
	// Native firecracker doesn't support pause in this cutover
	return errors.New("pause not supported in native firecracker driver")
}

func (d *Driver) Resume(ctx context.Context, workspaceID string) error {
	// Native firecracker doesn't support resume in this cutover
	return errors.New("resume not supported in native firecracker driver")
}

func (d *Driver) Fork(ctx context.Context, workspaceID, childWorkspaceID string) error {
	// Native firecracker doesn't support fork in this cutover
	return errors.New("fork not supported in native firecracker driver")
}

func (d *Driver) Destroy(ctx context.Context, workspaceID string) error {
	d.mu.Lock()
	if vendServer, ok := d.vendingServers[workspaceID]; ok {
		_ = vendServer.Stop()
		delete(d.vendingServers, workspaceID)
	}
	delete(d.projectRoots, workspaceID)
	delete(d.agents, workspaceID)
	d.mu.Unlock()

	// Stop the VM if manager is available
	if d.manager != nil {
		// Ignore error - workspace may already be stopped
		_ = d.manager.Stop(ctx, workspaceID)
	}

	return nil
}
