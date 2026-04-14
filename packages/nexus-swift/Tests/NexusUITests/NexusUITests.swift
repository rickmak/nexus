import XCTest
import Foundation

// MARK: - Nexus UI Test Suite
//
// HOW TO RECORD A NEW TEST
// ─────────────────────────
// 1. Open NexusApp.xcodeproj in Xcode.
// 2. Select the NexusUITests target.
// 3. Place the cursor inside an empty test method body (any `func testXxx` below).
// 4. Click the red ● Record button in the Debug bar (or Editor ▸ Start Recording UI Test).
// 5. Interact with the app — Xcode types the XCUIElement calls for you.
// 6. Stop recording, then run the test to verify.
//
// ACCESSIBILITY IDs ALREADY WIRED UP
// ────────────────────────────────────
//   startup_view          — spinner shown while daemon launches
//   empty_state_view      — "Select a workspace" placeholder
//   error_message         — error text when daemon unreachable
//   connection_status     — footer pill ("Connected" / "Starting…" / "Offline")
//   new_workspace_button  — (+) button in the sidebar header
//   workspace_row_<id>    — each row in the sidebar list

final class NexusUITests: XCTestCase {

    var app: XCUIApplication!

    override func setUpWithError() throws {
        continueAfterFailure = false
        app = XCUIApplication()
        let env = ProcessInfo.processInfo.environment
        // Point at the already-running daemon so tests don't need to cold-start one.
        // Remove or adjust these env vars if you want to test the full auto-start path.
        app.launchEnvironment["NEXUS_DAEMON_URL"] = env["NEXUS_UI_TEST_DAEMON_URL"] ?? "ws://localhost:8080"
        app.launchEnvironment["NEXUS_DAEMON_TOKEN"] = env["NEXUS_UI_TEST_DAEMON_TOKEN"] ?? (resolveDaemonToken() ?? "")
        app.launchEnvironment["NEXUS_UI_TEST_OPEN_DAEMON_PANEL"] = "1"
    }

    // ── Smoke: app launches and reaches either connected or startup state ──

    func testAppLaunches() throws {
        app.launch()

        // The window should appear within a few seconds.
        let window = app.windows.firstMatch
        XCTAssertTrue(window.waitForExistence(timeout: 10), "Main window should appear")
    }

    func testConnectsOrShowsStartup() throws {
        app.launch()

        // Either the startup spinner or the sidebar "Workspaces" header should appear,
        // indicating the app initialised successfully.
        let startupView  = app.otherElements["startup_view"]
        let sidebarLabel = app.staticTexts["Workspaces"]

        let appeared = startupView.waitForExistence(timeout: 5)
                    || sidebarLabel.waitForExistence(timeout: 5)
        XCTAssertTrue(appeared, "App should show startup view or sidebar within 5 s")
    }

    func testConnectionStatusIndicatorAppears() throws {
        app.launch()

        // Wait up to 15 s for the daemon to connect (auto-start may run).
        let status = app.otherElements["connection_status"]
        XCTAssertTrue(status.waitForExistence(timeout: 15),
                      "Connection status pill should be visible in the sidebar footer")
    }

    func testConnectionEventuallyConnects() throws {
        app.launch()

        // Poll until the status label shows "Connected" (auto-start may take a moment).
        let connected = app.staticTexts["Connected"]
        XCTAssertTrue(connected.waitForExistence(timeout: 30),
                      "Daemon should connect within 30 s")
    }

    // ── Sidebar interaction ───────────────────────────────────────────────

    func testNewWorkspaceButtonExists() throws {
        app.launch()

        // Wait for sidebar header to load.
        _ = app.staticTexts["Workspaces"].waitForExistence(timeout: 15)

        // The + button is a SidebarHeaderBtn (plain Button), so query as a button.
        let plusBtn = app.buttons["new_workspace_button"]
        XCTAssertTrue(plusBtn.waitForExistence(timeout: 5),
                      "New-workspace (+) button should be in the sidebar header")
    }

    func testDaemonPanelShowsHostToolsActions() throws {
        app.launch()
        _ = app.staticTexts["Connected"].waitForExistence(timeout: 30)

        XCTAssertTrue(app.buttons["daemon_action_refresh_tools"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.buttons["daemon_action_install_tools"].waitForExistence(timeout: 10))
        XCTAssertTrue(app.buttons["daemon_action_provision_runtime"].waitForExistence(timeout: 10))
    }

    func testFixtureComposePortAppears() throws {
        guard isHealthzUp("http://localhost:64001/healthz") else {
            throw XCTSkip("Fixture daemon on :64001 is not available")
        }
        let fixtureWorkspaceID = ProcessInfo.processInfo.environment["NEXUS_UI_TEST_FIXTURE_WORKSPACE_ID"] ?? ""
        let fixtureWorkspaceName = ProcessInfo.processInfo.environment["NEXUS_UI_TEST_FIXTURE_WORKSPACE_NAME"] ?? "xcui-compose"
        app.launchEnvironment["NEXUS_DAEMON_URL"] = "ws://localhost:64001"
        app.launchEnvironment["NEXUS_DAEMON_TOKEN"] = "xcui-token"

        app.launch()
        _ = app.staticTexts["Connected"].waitForExistence(timeout: 30)

        let workspaceRow: XCUIElement
        if !fixtureWorkspaceID.isEmpty {
            workspaceRow = app.buttons["workspace_row_\(fixtureWorkspaceID)"]
        } else {
            workspaceRow = app.buttons.matching(NSPredicate(format: "label CONTAINS %@", fixtureWorkspaceName)).firstMatch
        }
        guard workspaceRow.waitForExistence(timeout: 15) else {
            throw XCTSkip("Fixture workspace row not found")
        }
        workspaceRow.click()
        let portsTab = app.buttons["Ports"]
        if portsTab.waitForExistence(timeout: 3) {
            portsTab.click()
        }

        let composePortRow = app.descendants(matching: .any)["port_row_18080"]
        XCTAssertTrue(composePortRow.waitForExistence(timeout: 20), "Expected compose forwarded port row to appear")
    }

    // ── Recording playground ─────────────────────────────────────────────
    // Paste cursor here and press ● Record to capture any interaction.

    func testRecordInteraction() throws {
        app.launch()
        _ = app.staticTexts["Connected"].waitForExistence(timeout: 30)

        // Select first workspace row if any exist (otherElements, not cells)
        let rows = app.otherElements.matching(
            NSPredicate(format: "identifier BEGINSWITH 'workspace_row_'")
        )
        if rows.firstMatch.waitForExistence(timeout: 10) {
            rows.firstMatch.click()
        }

        // ← Xcode will insert recorded steps below this line.
    }
}

// MARK: - Launch performance

final class NexusUILaunchTests: XCTestCase {

    func testLaunchPerformance() throws {
        measure(metrics: [XCTApplicationLaunchMetric()]) {
            XCUIApplication().launch()
        }
    }
}

// MARK: - Helpers

private func resolveDaemonToken() -> String? {
    if let env = ProcessInfo.processInfo.environment["NEXUS_DAEMON_TOKEN"]?
        .trimmingCharacters(in: .whitespacesAndNewlines),
       !env.isEmpty {
        return env
    }

    let configuredService = ProcessInfo.processInfo.environment["NEXUS_DAEMON_TOKEN_KEYCHAIN_SERVICE"]?
        .trimmingCharacters(in: .whitespacesAndNewlines)
    let services = [
        configuredService,
        "nexus-daemon-token",
        "nexus/token",
        "nexus-daemon",
        "nexus",
    ]
    for maybeService in services {
        guard let service = maybeService, !service.isEmpty else { continue }
        if let token = readMacKeychainPassword(service: service), !token.isEmpty {
            return token
        }
    }
    return nil
}

private func isHealthzUp(_ rawURL: String) -> Bool {
    guard let url = URL(string: rawURL) else { return false }
    var req = URLRequest(url: url)
    req.timeoutInterval = 1.5
    let sem = DispatchSemaphore(value: 0)
    var ok = false
    URLSession.shared.dataTask(with: req) { _, response, _ in
        if let http = response as? HTTPURLResponse {
            ok = http.statusCode == 200
        }
        sem.signal()
    }.resume()
    _ = sem.wait(timeout: .now() + 2)
    return ok
}

private func readMacKeychainPassword(service: String) -> String? {
    let task = Process()
    task.executableURL = URL(fileURLWithPath: "/usr/bin/security")
    task.arguments = ["find-generic-password", "-s", service, "-w"]

    let out = Pipe()
    task.standardOutput = out
    task.standardError = Pipe()

    do {
        try task.run()
    } catch {
        return nil
    }
    task.waitUntilExit()
    guard task.terminationStatus == 0 else { return nil }

    let data = out.fileHandleForReading.readDataToEndOfFile()
    guard let raw = String(data: data, encoding: .utf8) else { return nil }
    let token = raw.trimmingCharacters(in: .whitespacesAndNewlines)
    return token.isEmpty ? nil : token
}
