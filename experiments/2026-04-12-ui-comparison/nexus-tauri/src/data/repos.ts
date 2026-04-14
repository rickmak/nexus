import type { Repo } from "../types";

export const REPOS: Repo[] = [
  {
    name: "nexus",
    workspaces: [
      {
        id: "ws-1",
        name: "auth-feature",
        branch: "feat/oauth",
        status: "running",
        ports: [3000, 8080],
        snapshots: 4,
      },
      {
        id: "ws-2",
        name: "api-refactor",
        branch: "refactor/v2",
        status: "paused",
        ports: [],
        snapshots: 2,
      },
    ],
  },
  {
    name: "magic",
    workspaces: [
      {
        id: "ws-3",
        name: "main",
        branch: "main",
        status: "running",
        ports: [4000],
        snapshots: 7,
      },
    ],
  },
];
