import SwiftUI

private let bgApp = Color(red: 0.051, green: 0.051, blue: 0.055)
private let bgSidebar = Color(red: 0.067, green: 0.067, blue: 0.075)
private let bgMain = Color(red: 0.078, green: 0.078, blue: 0.086)
private let bgHover = Color(red: 0.110, green: 0.110, blue: 0.122)
private let bgSelected = Color(red: 0.122, green: 0.122, blue: 0.137)
private let hairline = Color.white.opacity(0.07)
private let text1 = Color(red: 0.910, green: 0.910, blue: 0.929)
private let text2 = Color(red: 0.557, green: 0.557, blue: 0.576)
private let text3 = Color(red: 0.282, green: 0.282, blue: 0.290)
private let accent = Color(red: 0.039, green: 0.518, blue: 1.000)
private let green = Color(red: 0.188, green: 0.820, blue: 0.345)
private let orange = Color(red: 1.000, green: 0.624, blue: 0.039)

struct ContentView: View {
    @State private var selectedWorkspaceID: String?

    private var selectedWorkspace: Workspace? {
        for repo in MockData.repos {
            if let ws = repo.workspaces.first(where: { $0.id == selectedWorkspaceID }) {
                return ws
            }
        }
        return nil
    }

    var body: some View {
        NavigationSplitView {
            SidebarView(selectedWorkspaceID: $selectedWorkspaceID)
                .frame(minWidth: 220, maxWidth: 220)
        } detail: {
            Group {
                if let ws = selectedWorkspace {
                    WorkspaceDetailView(workspace: ws)
                } else {
                    Text("Select a workspace")
                        .foregroundColor(text2)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .background(bgMain)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .background(bgApp)
        .onAppear {
            if selectedWorkspaceID == nil {
                selectedWorkspaceID = MockData.repos.first?.workspaces.first?.id
            }
        }
    }
}

private struct SidebarView: View {
    @Binding var selectedWorkspaceID: String?

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    ForEach(MockData.repos) { repo in
                        Text(repo.name.uppercased())
                            .font(.system(size: 10))
                            .foregroundColor(text3)
                            .padding(EdgeInsets(top: 12, leading: 12, bottom: 4, trailing: 12))
                            .frame(maxWidth: .infinity, alignment: .leading)

                        ForEach(repo.workspaces) { ws in
                            WorkspaceSidebarRow(
                                workspace: ws,
                                isSelected: selectedWorkspaceID == ws.id
                            ) {
                                selectedWorkspaceID = ws.id
                            }
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            Rectangle()
                .fill(hairline)
                .frame(height: 1)

            Button(action: {}) {
                Text("⌘N  New workspace")
                    .font(.system(size: 12))
                    .foregroundColor(text3)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 12)
                    .padding(.vertical, 10)
            }
            .buttonStyle(.plain)
            .background(bgSidebar)
        }
        .background(bgSidebar)
    }
}

private struct WorkspaceSidebarRow: View {
    let workspace: Workspace
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            ZStack(alignment: .leading) {
                if isSelected {
                    Rectangle()
                        .fill(accent)
                        .frame(width: 2)
                }

                HStack(spacing: 8) {
                    statusDot
                    Text(workspace.name)
                        .font(.system(size: 13))
                        .foregroundColor(text1)
                        .lineLimit(1)
                }
                .padding(EdgeInsets(top: 6, leading: 24, bottom: 6, trailing: 12))
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .background(isSelected ? bgSelected : Color.clear)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }

    private var statusDot: some View {
        Circle()
            .fill(statusColor)
            .frame(width: 6, height: 6)
    }

    private var statusColor: Color {
        switch workspace.status {
        case .running: return green
        case .paused: return orange
        case .stopped: return text3
        }
    }
}

private struct WorkspaceDetailView: View {
    let workspace: Workspace

    @State private var bottomTab: BottomTab = .snapshots

    var body: some View {
        VStack(spacing: 0) {
            TopBar(workspace: workspace)
                .frame(height: 40)
            panelDivider
            TerminalView()
                .frame(minHeight: 0, maxHeight: .infinity)
            panelDivider
            BottomPanel(workspace: workspace, selectedTab: $bottomTab)
                .frame(height: 160)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(bgMain)
    }

    private var panelDivider: some View {
        Rectangle()
            .fill(hairline)
            .frame(height: 1)
    }
}

private struct TopBar: View {
    let workspace: Workspace

    var body: some View {
        HStack(alignment: .center, spacing: 0) {
            HStack(spacing: 0) {
                Text(workspace.name)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundColor(text1)
                Text(" · ")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundColor(text2)
                Text(workspace.branch)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(text2)
            }

            Spacer(minLength: 12)

            HStack(spacing: 8) {
                StatusChip(status: workspace.status)

                ghostIconButton(systemName: "square.and.arrow.up")
                ghostIconButton(systemName: "ellipsis")
            }
        }
        .padding(.horizontal, 12)
        .background(bgMain)
    }

    private func ghostIconButton(systemName: String) -> some View {
        Button(action: {}) {
            Image(systemName: systemName)
                .font(.system(size: 13))
                .foregroundColor(text2)
                .frame(width: 28, height: 28)
        }
        .buttonStyle(.plain)
    }
}

private struct StatusChip: View {
    let status: WorkspaceStatus

    var body: some View {
        Text(label)
            .font(.system(size: 11, weight: .medium))
            .foregroundColor(text2)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(bgHover)
            .overlay(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(hairline, lineWidth: 1)
            )
    }

    private var label: String {
        switch status {
        case .running: return "Running"
        case .paused: return "Paused"
        case .stopped: return "Stopped"
        }
    }
}

private struct TerminalView: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 4) {
                terminalLine(prefix: "❯ ", prefixColor: green, rest: "claude --continue")
                lineBody("[claude] Analyzing codebase...")
                lineBody("[claude] Reading src/auth/oauth.ts")
                lineBody("[claude] Editing src/auth/oauth.ts")
                HStack(alignment: .firstTextBaseline, spacing: 0) {
                    Text("❯ ")
                        .font(.system(size: 13, design: .monospaced))
                        .foregroundColor(green)
                    Text("_")
                        .font(.system(size: 13, design: .monospaced))
                        .foregroundColor(text3)
                }
            }
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .background(bgMain)
    }

    private func terminalLine(prefix: String, prefixColor: Color, rest: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 0) {
            Text(prefix)
                .font(.system(size: 13, design: .monospaced))
                .foregroundColor(prefixColor)
            Text(rest)
                .font(.system(size: 13, design: .monospaced))
                .foregroundColor(text1)
        }
    }

    private func lineBody(_ s: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 0) {
            Text("[claude]")
                .font(.system(size: 13, design: .monospaced))
                .foregroundColor(text2)
            Text(String(s.dropFirst("[claude]".count)))
                .font(.system(size: 13, design: .monospaced))
                .foregroundColor(text1)
        }
    }
}

private enum BottomTab: String, CaseIterable, Identifiable {
    case snapshots = "Snapshots"
    case ports = "Ports"
    case log = "Log"

    var id: String { rawValue }
}

private struct BottomPanel: View {
    let workspace: Workspace
    @Binding var selectedTab: BottomTab

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 0) {
                ForEach(BottomTab.allCases) { tab in
                    Button(action: { selectedTab = tab }) {
                        Text(tab.rawValue)
                            .font(.system(size: 12, weight: selectedTab == tab ? .semibold : .regular))
                            .foregroundColor(selectedTab == tab ? text1 : text2)
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .frame(maxHeight: .infinity)
                            .background(selectedTab == tab ? bgHover : Color.clear)
                            .overlay(alignment: .bottom) {
                                if selectedTab == tab {
                                    Rectangle()
                                        .fill(accent)
                                        .frame(height: 2)
                                }
                            }
                    }
                    .buttonStyle(.plain)
                }
                Spacer(minLength: 0)
            }
            .frame(height: 36)
            .background(bgSidebar)
            Rectangle()
                .fill(hairline)
                .frame(height: 1)

            Group {
                switch selectedTab {
                case .snapshots: SnapshotsStrip(activeIndex: min(3, max(0, workspace.snapshotCount - 1)))
                case .ports: PortsList(ports: workspace.ports)
                case .log: LogPanel()
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(bgMain)
        }
    }
}

private struct SnapshotsStrip: View {
    let activeIndex: Int

    var body: some View {
        ScrollView {
            HStack(spacing: 0) {
                ForEach(0 ..< 4, id: \.self) { i in
                    snapshotDot(isActive: i == activeIndex)
                    if i < 3 {
                        Rectangle()
                            .fill(hairline)
                            .frame(width: 28, height: 1)
                    }
                }
            }
            .padding(16)
            .frame(maxWidth: .infinity)
        }
    }

    private func snapshotDot(isActive: Bool) -> some View {
        ZStack {
            if isActive {
                Circle()
                    .fill(accent)
                    .frame(width: 10, height: 10)
            } else {
                Circle()
                    .strokeBorder(hairline, lineWidth: 1)
                    .background(Circle().fill(Color.clear))
                    .frame(width: 10, height: 10)
            }
        }
        .frame(width: 10, height: 10)
    }
}

private struct PortsList: View {
    let ports: [Int]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 8) {
                if ports.isEmpty {
                    Text("No forwarded ports")
                        .font(.system(size: 12))
                        .foregroundColor(text3)
                        .padding(12)
                } else {
                    ForEach(ports, id: \.self) { port in
                        HStack {
                            Text("\(port) → localhost:\(port)")
                                .font(.system(size: 12, design: .monospaced))
                                .foregroundColor(text1)
                            Spacer()
                            Button("Open") {}
                                .font(.system(size: 12))
                                .foregroundColor(accent)
                                .buttonStyle(.plain)
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.vertical, 8)
        }
    }
}

private struct LogPanel: View {
    private let lines: [(String, String)] = [
        ("14:02:01.204", "workspace ready"),
        ("14:02:02.891", "synced snapshot snap-7a3"),
        ("14:02:05.112", "port 3000 listening"),
        ("14:02:08.330", "agent: plan accepted"),
    ]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 4) {
                ForEach(Array(lines.enumerated()), id: \.offset) { _, row in
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Text(row.0)
                            .font(.system(size: 11, design: .monospaced))
                            .foregroundColor(text3)
                        Text(row.1)
                            .font(.system(size: 12, design: .monospaced))
                            .foregroundColor(text1)
                    }
                }
            }
            .padding(12)
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}

#Preview {
    ContentView()
}
