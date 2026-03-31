package runtime

import (
	"context"
)

type Driver interface {
	Backend() string
	Create(ctx context.Context, req CreateRequest) error
	Start(ctx context.Context, workspaceID string) error
	Stop(ctx context.Context, workspaceID string) error
	Restore(ctx context.Context, workspaceID string) error
	Destroy(ctx context.Context, workspaceID string) error
}

type CreateRequest struct {
	WorkspaceID   string
	WorkspaceName string
	ProjectRoot   string
	Options       map[string]string
}

type WorkspaceMetadata struct {
	Backend     string
	WorkspaceID string
	State       string
}
