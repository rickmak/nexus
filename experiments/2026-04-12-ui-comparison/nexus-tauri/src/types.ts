export type WorkspaceStatus = "running" | "paused";

export interface Workspace {
  id: string;
  name: string;
  branch: string;
  status: WorkspaceStatus;
  ports: number[];
  snapshots: number;
}

export interface Repo {
  name: string;
  workspaces: Workspace[];
}
