package handlers

import (
	"context"
	"errors"

	"github.com/inizio/nexus/packages/nexus/pkg/compose"

	rpckit "github.com/inizio/nexus/packages/nexus/pkg/rpcerrors"
	"github.com/inizio/nexus/packages/nexus/pkg/spotlight"
)

var discoverPublishedPorts = compose.DiscoverPublishedPorts

type SpotlightExposeParams struct {
	Spec spotlight.ExposeSpec `json:"spec"`
}

type SpotlightListParams struct {
	WorkspaceID string `json:"workspaceId,omitempty"`
}

type SpotlightCloseParams struct {
	ID string `json:"id"`
}

type SpotlightExposeResult struct {
	Forward *spotlight.Forward `json:"forward"`
}

type SpotlightListResult struct {
	Forwards []*spotlight.Forward `json:"forwards"`
}

type SpotlightCloseResult struct {
	Closed bool `json:"closed"`
}

type SpotlightApplyComposePortsParams struct {
	WorkspaceID string `json:"workspaceId"`
}

type SpotlightApplyComposePortsError struct {
	Service    string `json:"service"`
	HostPort   int    `json:"hostPort"`
	TargetPort int    `json:"targetPort"`
	Message    string `json:"message"`
}

type SpotlightApplyComposePortsResult struct {
	Forwards []*spotlight.Forward              `json:"forwards"`
	Errors   []SpotlightApplyComposePortsError `json:"errors"`
}

func HandleSpotlightExpose(ctx context.Context, p SpotlightExposeParams, mgr *spotlight.Manager) (*SpotlightExposeResult, *rpckit.RPCError) {
	fwd, err := mgr.Expose(ctx, p.Spec)
	if err != nil {
		return nil, rpckit.ErrInvalidParams
	}

	return &SpotlightExposeResult{Forward: fwd}, nil
}

func HandleSpotlightList(_ context.Context, p SpotlightListParams, mgr *spotlight.Manager) (*SpotlightListResult, *rpckit.RPCError) {
	all := mgr.List(p.WorkspaceID)
	return &SpotlightListResult{Forwards: all}, nil
}

func HandleSpotlightClose(_ context.Context, p SpotlightCloseParams, mgr *spotlight.Manager) (*SpotlightCloseResult, *rpckit.RPCError) {
	closed := mgr.Close(p.ID)
	if !closed {
		return nil, rpckit.ErrInvalidParams
	}

	return &SpotlightCloseResult{Closed: true}, nil
}

func HandleSpotlightApplyComposePorts(ctx context.Context, p SpotlightApplyComposePortsParams, rootPath string, mgr *spotlight.Manager) (*SpotlightApplyComposePortsResult, *rpckit.RPCError) {
	if p.WorkspaceID == "" || rootPath == "" {
		return nil, rpckit.ErrInvalidParams
	}

	published, err := discoverPublishedPorts(ctx, rootPath)
	if err != nil {
		if errors.Is(err, compose.ErrComposeFileNotFound) {
			return &SpotlightApplyComposePortsResult{
				Forwards: []*spotlight.Forward{},
				Errors:   []SpotlightApplyComposePortsError{},
			}, nil
		}
		return nil, rpckit.ErrInvalidParams
	}

	result := &SpotlightApplyComposePortsResult{
		Forwards: make([]*spotlight.Forward, 0, len(published)),
		Errors:   make([]SpotlightApplyComposePortsError, 0),
	}

	for _, entry := range published {
		host := entry.HostIP
		if host == "" {
			host = "127.0.0.1"
		}

		var (
			fwd       *spotlight.Forward
			exposeErr error
		)
		for _, localPort := range composeLocalPortCandidates(entry.HostPort, entry.TargetPort) {
			fwd, exposeErr = mgr.Expose(ctx, spotlight.ExposeSpec{
				WorkspaceID: p.WorkspaceID,
				Service:     entry.Service,
				RemotePort:  entry.TargetPort,
				LocalPort:   localPort,
				Host:        host,
			})
			if exposeErr == nil {
				break
			}
		}
		if exposeErr != nil {
			result.Errors = append(result.Errors, SpotlightApplyComposePortsError{
				Service:    entry.Service,
				HostPort:   entry.HostPort,
				TargetPort: entry.TargetPort,
				Message:    exposeErr.Error(),
			})
			continue
		}
		result.Forwards = append(result.Forwards, fwd)
	}

	return result, nil
}

func composeLocalPortCandidates(hostPort, targetPort int) []int {
	candidates := make([]int, 0, 5)
	seen := map[int]struct{}{}
	add := func(p int) {
		if p <= 0 || p > 65535 {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		candidates = append(candidates, p)
	}
	add(hostPort)
	if hostPort > 0 && hostPort < 1024 {
		add(10000 + hostPort)
		add(20000 + hostPort)
		add(30000 + hostPort)
	}
	if targetPort >= 1024 {
		add(targetPort)
	}
	if len(candidates) == 0 {
		add(10000 + targetPort)
	}
	return candidates
}
