import Foundation
import Combine

/// Manages multiple PTY sessions (tabs) for a workspace
@MainActor
public final class PTYSessionManager: ObservableObject {

    /// A single terminal tab
    public struct Tab: Identifiable, Equatable {
        public let id: String
        public var name: String
        public var isActive: Bool
        public var isLoading: Bool
        public var error: String?

        public init(id: String, name: String, isActive: Bool = false, isLoading: Bool = true, error: String? = nil) {
            self.id = id
            self.name = name
            self.isActive = isActive
            self.isLoading = isLoading
            self.error = error
        }

        public static func == (lhs: Tab, rhs: Tab) -> Bool {
            lhs.id == rhs.id &&
            lhs.name == rhs.name &&
            lhs.isActive == rhs.isActive &&
            lhs.isLoading == rhs.isLoading &&
            lhs.error == rhs.error
        }
    }

    @Published public private(set) var tabs: [Tab] = []
    @Published public var activeTabId: String? {
        didSet {
            updateActiveState()
        }
    }

    public let workspaceId: String
    private let client: WebSocketDaemonClient
    private var refreshTask: Task<Void, Never>?

    public init(workspaceId: String, client: WebSocketDaemonClient) {
        self.workspaceId = workspaceId
        self.client = client
    }

    deinit {
        refreshTask?.cancel()
    }

    // MARK: - Tab Management

    /// Create a new terminal tab
    public func createTab(name: String? = nil) async {
        let tabName = name ?? "Tab \(tabs.count + 1)"
        let tempId = UUID().uuidString

        // Add loading tab
        await MainActor.run {
            let newTab = Tab(id: tempId, name: tabName, isLoading: true)
            tabs.append(newTab)
            activeTabId = tempId
        }

        do {
            // Calculate default terminal size
            let cols = 120
            let rows = 40

            let sessionId = try await client.openPTY(
                workspaceId: workspaceId,
                name: tabName,
                cols: cols,
                rows: rows,
                useTmux: true
            )

            // Update with real session ID
            await MainActor.run {
                if let index = tabs.firstIndex(where: { $0.id == tempId }) {
                    tabs[index] = Tab(
                        id: sessionId,
                        name: tabName,
                        isActive: true,
                        isLoading: false
                    )
                    activeTabId = sessionId
                }
            }

            // Subscribe to PTY events
            setupPTYHandlers(sessionId: sessionId)

        } catch {
            await MainActor.run {
                if let index = tabs.firstIndex(where: { $0.id == tempId }) {
                    tabs[index].isLoading = false
                    tabs[index].error = error.localizedDescription
                }
            }
        }
    }

    /// Close a specific tab
    public func closeTab(id: String) async {
        // Close PTY session
        try? await client.closePTY(sessionId: id)

        // Remove from tabs
        await MainActor.run {
            tabs.removeAll { $0.id == id }

            // Select another tab if needed
            if activeTabId == id {
                activeTabId = tabs.first?.id
            }
        }
    }

    /// Rename a tab
    public func renameTab(id: String, to newName: String) async {
        let success = try? await client.renamePTYSession(sessionId: id, name: newName)

        if success == true {
            await MainActor.run {
                if let index = tabs.firstIndex(where: { $0.id == id }) {
                    tabs[index].name = newName
                }
            }
        }
    }

    /// Switch to a different tab
    public func switchToTab(id: String) {
        activeTabId = id
    }

    /// Refresh tabs from server (sync state)
    public func refreshTabs() async {
        do {
            let sessions = try await client.listPTYSessions(workspaceId: workspaceId)

            await MainActor.run {
                // Update existing tabs and add new ones
                var newTabs: [Tab] = []
                for session in sessions {
                    if let existing = tabs.first(where: { $0.id == session.id }) {
                        // Keep existing tab but update name if changed
                        newTabs.append(Tab(
                            id: session.id,
                            name: session.name,
                            isActive: session.id == activeTabId,
                            isLoading: existing.isLoading,
                            error: existing.error
                        ))
                    } else {
                        // New tab from server
                        newTabs.append(Tab(
                            id: session.id,
                            name: session.name,
                            isActive: false,
                            isLoading: false
                        ))
                    }
                }
                self.tabs = newTabs

                // Set active tab if none selected
                if activeTabId == nil || !tabs.contains(where: { $0.id == activeTabId }) {
                    activeTabId = tabs.first?.id
                }
            }
        } catch {
            // Silent fail - tabs will sync next time
        }
    }

    /// Start auto-refresh (call when view appears)
    public func startRefreshLoop() {
        refreshTask?.cancel()
        refreshTask = Task { [weak self] in
            while !Task.isCancelled {
                // Sleep first: avoids a tight retry loop (fail fast → no delay → allocate → repeat)
                // when the daemon is unreachable at the time the view appears.
                try? await Task.sleep(for: .seconds(5))
                guard !Task.isCancelled, let self else { break }
                await self.refreshTabs()
            }
        }
    }

    /// Stop auto-refresh (call when view disappears)
    public func stopRefreshLoop() {
        refreshTask?.cancel()
    }

    // MARK: - Private Helpers

    private func updateActiveState() {
        for index in tabs.indices {
            tabs[index].isActive = tabs[index].id == activeTabId
        }
    }

    private func setupPTYHandlers(sessionId: String) {
        client.subscribePTY(
            sessionId: sessionId,
            onData: { _ in /* Data handled by terminal view */ },
            onExit: { [weak self] _ in
                Task { @MainActor [weak self] in
                    await self?.handleSessionExit(sessionId: sessionId)
                }
            }
        )
    }

    private func handleSessionExit(sessionId: String) async {
        client.unsubscribePTY(sessionId: sessionId)

        await MainActor.run {
            if let index = tabs.firstIndex(where: { $0.id == sessionId }) {
                tabs[index].isLoading = false
                tabs[index].error = "Process exited"
            }
        }
    }
}
