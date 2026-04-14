import Foundation

enum WorkspaceStatus: Equatable {
    case running
    case paused
    case stopped
}

struct Workspace: Identifiable {
    let id: String
    let name: String
    let branch: String
    let status: WorkspaceStatus
    let ports: [Int]
    let snapshotCount: Int
}

struct Repo: Identifiable {
    let id = UUID()
    let name: String
    var workspaces: [Workspace]
}

enum MockData {
    static let repos: [Repo] = [
        Repo(name: "nexus", workspaces: [
            Workspace(
                id: "ws-1",
                name: "auth-feature",
                branch: "feat/oauth",
                status: .running,
                ports: [3000, 8080],
                snapshotCount: 4
            ),
            Workspace(
                id: "ws-2",
                name: "api-refactor",
                branch: "refactor/v2",
                status: .paused,
                ports: [],
                snapshotCount: 2
            ),
        ]),
        Repo(name: "magic", workspaces: [
            Workspace(
                id: "ws-3",
                name: "main",
                branch: "main",
                status: .running,
                ports: [4000],
                snapshotCount: 7
            ),
        ]),
    ]
}
