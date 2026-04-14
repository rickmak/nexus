import Foundation

// MARK: - Protocol

/// Abstraction over the Nexus daemon. Set NEXUS_DAEMON_URL to point at the daemon;
/// defaults to ws://localhost:63987.
public protocol DaemonClient: Sendable {

    // ── Discovery ────────────────────────────────────────────────────
    func listWorkspaces() async throws -> [Workspace]
    func listRelations() async throws -> [RelationsGroup]

    // ── Lifecycle ────────────────────────────────────────────────────
    func createWorkspace(spec: WorkspaceCreateSpec) async throws -> Workspace
    func startWorkspace(id: String) async throws
    func stopWorkspace(id: String) async throws
    func removeWorkspace(id: String) async throws

    // ── Ports ────────────────────────────────────────────────────────
    func markWorkspaceReady(id: String) async throws
    func listPorts(workspaceId: String) async throws -> [ForwardedPort]
    func addPort(workspaceId: String, port: Int) async throws
    func removePort(workspaceId: String, port: Int) async throws
    func activateTunnels(workspaceId: String) async throws -> TunnelStatus
    func deactivateTunnels(workspaceId: String) async throws -> TunnelStatus
    func tunnelStatus(workspaceId: String) async throws -> TunnelStatus

    // ── Exec ─────────────────────────────────────────────────────────
    /// Runs a command in the workspace's root directory and returns buffered output.
    func exec(workspaceId: String, command: String, args: [String]) async throws -> ExecOutput

    // ── Workspace info ────────────────────────────────────────────────
    /// Returns rich workspace metadata including spotlight ports.
    func workspaceInfo(id: String) async throws -> WorkspaceInfo
}

public struct TunnelStatus: Sendable {
    public let active: Bool
    public let activeWorkspaceId: String
}
