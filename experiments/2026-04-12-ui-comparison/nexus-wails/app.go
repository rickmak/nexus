package main

import (
	"context"
	"time"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type Workspace struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Branch        string `json:"branch"`
	Status        string `json:"status"`
	Ports         []int  `json:"ports"`
	SnapshotCount int    `json:"snapshotCount"`
}

type RepoGroup struct {
	Name        string      `json:"name"`
	Workspaces  []Workspace `json:"workspaces"`
}

func (a *App) GetWorkspaces() []RepoGroup {
	return []RepoGroup{
		{
			Name: "nexus",
			Workspaces: []Workspace{
				{ID: "ws-1", Name: "auth-feature", Branch: "feat/oauth", Status: "running", Ports: []int{3000, 8080}, SnapshotCount: 4},
				{ID: "ws-2", Name: "api-refactor", Branch: "refactor/v2", Status: "paused", Ports: []int{}, SnapshotCount: 2},
			},
		},
		{
			Name: "magic",
			Workspaces: []Workspace{
				{ID: "ws-3", Name: "main", Branch: "main", Status: "running", Ports: []int{4000}, SnapshotCount: 7},
			},
		},
	}
}

func (a *App) WorkspaceAction(id string, action string) error {
	time.Sleep(200 * time.Millisecond)
	return nil
}
