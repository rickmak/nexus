package seatbelt

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/inizio/nexus/packages/nexus/pkg/runtime"
)

type Driver struct {
	mu         sync.RWMutex
	workspaces map[string]*workspaceState
}

type workspaceState struct {
	projectRoot string
	state       string
}

func NewDriver() *Driver {
	return &Driver{workspaces: make(map[string]*workspaceState)}
}

func (d *Driver) Backend() string { return "seatbelt" }

func (d *Driver) Create(ctx context.Context, req runtime.CreateRequest) error {
	_ = ctx
	if strings.TrimSpace(req.WorkspaceID) == "" {
		return fmt.Errorf("workspace id is required")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.workspaces[req.WorkspaceID]; exists {
		return fmt.Errorf("workspace %s already exists", req.WorkspaceID)
	}

	d.workspaces[req.WorkspaceID] = &workspaceState{projectRoot: req.ProjectRoot, state: "created"}
	return nil
}

func (d *Driver) Start(ctx context.Context, workspaceID string) error {
	_ = ctx
	return d.setState(workspaceID, "running")
}

func (d *Driver) Stop(ctx context.Context, workspaceID string) error {
	_ = ctx
	return d.setState(workspaceID, "stopped")
}

func (d *Driver) Restore(ctx context.Context, workspaceID string) error {
	_ = ctx
	return d.setState(workspaceID, "running")
}

func (d *Driver) Pause(ctx context.Context, workspaceID string) error {
	_ = ctx
	return d.setState(workspaceID, "paused")
}

func (d *Driver) Resume(ctx context.Context, workspaceID string) error {
	_ = ctx
	return d.setState(workspaceID, "running")
}

func (d *Driver) Fork(ctx context.Context, workspaceID, childWorkspaceID string) error {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()

	parent, ok := d.workspaces[workspaceID]
	if !ok {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}
	if _, exists := d.workspaces[childWorkspaceID]; exists {
		return fmt.Errorf("workspace %s already exists", childWorkspaceID)
	}

	d.workspaces[childWorkspaceID] = &workspaceState{projectRoot: parent.projectRoot, state: "created"}
	return nil
}

func (d *Driver) Destroy(ctx context.Context, workspaceID string) error {
	_ = ctx
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.workspaces[workspaceID]; !ok {
		return fmt.Errorf("workspace %s not found", workspaceID)
	}
	delete(d.workspaces, workspaceID)
	return nil
}

func (d *Driver) setState(workspaceID, state string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ws, ok := d.workspaces[workspaceID]
	if !ok {
		ws = &workspaceState{state: state}
		d.workspaces[workspaceID] = ws
		return nil
	}
	ws.state = state
	return nil
}

func (d *Driver) AgentConn(ctx context.Context, workspaceID string) (net.Conn, error) {
	_ = ctx
	left, right := net.Pipe()
	go d.serveShellProtocol(context.Background(), workspaceID, right)
	return left, nil
}

func (d *Driver) serveShellProtocol(ctx context.Context, connWorkspaceID string, conn net.Conn) {
	defer conn.Close()

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	type shellSession struct {
		id   string
		cmd  *exec.Cmd
		ptmx *os.File
	}

	var session *shellSession
	closeSession := func() {
		if session == nil {
			return
		}
		_ = session.ptmx.Close()
		if session.cmd.Process != nil {
			_ = session.cmd.Process.Kill()
			_, _ = session.cmd.Process.Wait()
		}
		session = nil
	}

	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			closeSession()
			return
		}

		typ, _ := req["type"].(string)
		id, _ := req["id"].(string)

		switch typ {
		case "shell.open":
			closeSession()

			shell := strings.TrimSpace(asString(req["command"]))
			if shell == "" {
				shell = "bash"
			}

			workdir := strings.TrimSpace(asString(req["workdir"]))
			if workdir == "" || workdir == "/workspace" {
				workdir = d.workspaceProjectRoot(connWorkspaceID)
			}
			if workdir == "" {
				workdir = "."
			}

			cmd := exec.CommandContext(ctx, shell)
			cmd.Dir = workdir
			cmd.Env = append(os.Environ(), "TERM=xterm-256color")

			ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 120})
			if err != nil {
				_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 1, "stderr": err.Error()})
				continue
			}

			session = &shellSession{id: id, cmd: cmd, ptmx: ptmx}
			_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 0})

			go func(s *shellSession) {
				buf := make([]byte, 4096)
				for {
					n, err := s.ptmx.Read(buf)
					if n > 0 {
						_ = enc.Encode(map[string]any{"id": s.id, "type": "chunk", "stream": "stdout", "data": string(buf[:n])})
					}
					if err != nil {
						break
					}
				}

				exitCode := 0
				if s.cmd.Process != nil {
					_, _ = s.cmd.Process.Wait()
				}
				if s.cmd.ProcessState != nil {
					exitCode = s.cmd.ProcessState.ExitCode()
				}
				_ = enc.Encode(map[string]any{"id": s.id, "type": "result", "exit_code": exitCode})
			}(session)

		case "shell.write":
			if session == nil {
				_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 1, "stderr": "no active shell session"})
				continue
			}
			data := asString(req["data"])
			if _, err := session.ptmx.Write([]byte(data)); err != nil {
				_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 1, "stderr": err.Error()})
				continue
			}
			_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 0})

		case "shell.resize":
			if session == nil {
				_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 1, "stderr": "no active shell session"})
				continue
			}
			cols := toInt(req["cols"], 120)
			rows := toInt(req["rows"], 30)
			if err := pty.Setsize(session.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)}); err != nil {
				_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 1, "stderr": err.Error()})
				continue
			}
			_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 0})

		case "shell.close":
			closeSession()
			_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 0})
			return

		default:
			_ = enc.Encode(map[string]any{"id": id, "type": "result", "exit_code": 1, "stderr": fmt.Sprintf("unknown request type %q", typ)})
		}
	}
}

func (d *Driver) workspaceProjectRoot(workspaceID string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if ws, ok := d.workspaces[workspaceID]; ok {
		return ws.projectRoot
	}
	return ""
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func toInt(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		if int(v) > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	}
	return fallback
}
