# nexus-swift

Native macOS client for Nexus — SwiftUI, macOS 14+.

See `[ROADMAP.md](../../ROADMAP.md#native-macos-client-swiftui)` for milestones and architecture.

## Quick start

```bash
# Demo mode — no daemon needed
swift run

# Against a live daemon
NEXUS_DAEMON_URL=ws://localhost:8080 swift run
```

## Structure

```
Sources/NexusApp/
├── NexusApp.swift           # @main entry, window + commands
├── AppState.swift           # Root @MainActor state, drives all views
├── Theme.swift              # Design tokens (colors, fonts, spacing)
├── Models/
│   └── Workspace.swift      # Workspace, Repo, Snapshot, Port models
├── Client/
│   ├── DaemonClient.swift   # Protocol — swap mock ↔ real via env var
│   ├── MockClient.swift     # Offline mock (demo-ready)
│   └── WebSocketClient.swift # JSON-RPC 2.0 over WebSocket (real daemon)
└── Views/
    ├── ContentView.swift        # NavigationSplitView root
    ├── SidebarView.swift        # Repo › Workspace list, ⌘N button
    ├── WorkspaceDetailView.swift # Top bar + terminal + bottom panel
    ├── TerminalView.swift       # Mock terminal (→ real PTY in M3)
    └── BottomPanelView.swift    # Snapshots / Ports / Log tabs
```

## Design principles

- **Mock-first**: `MockDaemonClient` works offline; set `NEXUS_DAEMON_URL` to go live
- **Theme parity**: `Theme.swift` tokens match the Tauri experiment's CSS variables
- **No Xcode required**: SPM-only; Xcode is optional for debugging
- **MVVM**: Views read from `AppState`; mutations go through `AppState.action(_:on:)`

