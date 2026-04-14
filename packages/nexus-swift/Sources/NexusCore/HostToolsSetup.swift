import Foundation

public struct HostToolCheck: Identifiable, Sendable {
    public let id: String
    public let name: String
    public let isInstalled: Bool
    public let details: String
    public let installFormula: String?

    public init(id: String, name: String, isInstalled: Bool, details: String, installFormula: String?) {
        self.id = id
        self.name = name
        self.isInstalled = isInstalled
        self.details = details
        self.installFormula = installFormula
    }
}

public struct HostToolSnapshot: Sendable {
    public let checks: [HostToolCheck]
    public let nestedVirtualizationSupported: Bool
    public let hasHomebrew: Bool

    public init(checks: [HostToolCheck], nestedVirtualizationSupported: Bool, hasHomebrew: Bool) {
        self.checks = checks
        self.nestedVirtualizationSupported = nestedVirtualizationSupported
        self.hasHomebrew = hasHomebrew
    }
}

public enum HostToolsSetupError: LocalizedError {
    case homebrewMissing
    case installFailed(String)
    case provisioningFailed(String)
    case daemonInstallFailed(String)

    public var errorDescription: String? {
        switch self {
        case .homebrewMissing:
            return "Homebrew is required to install host tools. Install from https://brew.sh and retry."
        case .installFailed(let output):
            return output.isEmpty ? "Tool installation failed." : output
        case .provisioningFailed(let output):
            return output.isEmpty ? "Runtime provisioning failed." : output
        case .daemonInstallFailed(let output):
            return output.isEmpty ? "Daemon install/update failed." : output
        }
    }
}

public enum HostToolsSetup {
    public static func inspect() async -> HostToolSnapshot {
        let hasBrew = await commandExists("brew")
        let hasLima = await commandExists("limactl")
        let hasMutagen = await commandExists("mutagen")
        let hasTmux = await commandExists("tmux")
        let hasNexus = await commandExists("nexus")
        let hasNexusDaemon = await commandExists("nexus-daemon")
        let nestedSupported = await checkNestedVirtualizationSupport()

        return HostToolSnapshot(
            checks: [
                HostToolCheck(
                    id: "lima",
                    name: "Lima",
                    isInstalled: hasLima,
                    details: hasLima ? "limactl found in PATH" : "Required for macOS firecracker runtime",
                    installFormula: "lima"
                ),
                HostToolCheck(
                    id: "mutagen",
                    name: "Mutagen",
                    isInstalled: hasMutagen,
                    details: hasMutagen ? "mutagen found in PATH" : "Recommended for sync performance",
                    installFormula: "mutagen-io/mutagen/mutagen"
                ),
                HostToolCheck(
                    id: "tmux",
                    name: "tmux",
                    isInstalled: hasTmux,
                    details: hasTmux ? "tmux found in PATH" : "Required for terminal session persistence",
                    installFormula: "tmux"
                ),
                HostToolCheck(
                    id: "nexus",
                    name: "Nexus CLI",
                    isInstalled: hasNexus,
                    details: hasNexus ? "nexus found in PATH" : "Required for daemon install/update flow",
                    installFormula: nil
                ),
                HostToolCheck(
                    id: "nexus-daemon",
                    name: "Nexus Daemon",
                    isInstalled: hasNexusDaemon,
                    details: hasNexusDaemon ? "nexus-daemon found in PATH" : "Installed and updated via Nexus installer/updater",
                    installFormula: nil
                ),
            ],
            nestedVirtualizationSupported: nestedSupported,
            hasHomebrew: hasBrew
        )
    }

    public static func installMissingTools(snapshot: HostToolSnapshot) async throws -> String {
        guard snapshot.hasHomebrew else {
            throw HostToolsSetupError.homebrewMissing
        }

        let formulas = snapshot.checks
            .filter { !$0.isInstalled }
            .compactMap(\.installFormula)
        if formulas.isEmpty {
            return "All managed host tools are already installed."
        }

        let command = "brew install " + formulas.joined(separator: " ")
        let result = await runShell(command)
        if result.exitCode != 0 {
            throw HostToolsSetupError.installFailed(result.output)
        }
        return result.output.isEmpty ? "Installed: \(formulas.joined(separator: ", "))" : result.output
    }

    public static func provisionFirecrackerRuntime(projectRoot: String, useAdministratorPrivileges: Bool) async throws -> String {
        let escapedRoot = shellQuote(projectRoot)
        let command = "nexus init --force \(escapedRoot)"
        let result: CommandResult

        if useAdministratorPrivileges {
            result = await runPrivileged(command)
        } else {
            result = await runShell(command)
        }

        if result.exitCode != 0 {
            throw HostToolsSetupError.provisioningFailed(result.output)
        }
        return result.output.isEmpty
            ? "Firecracker runtime provisioning completed."
            : result.output
    }

    public static func installOrUpdateDaemon() async throws -> String {
        let command = """
if command -v nexus >/dev/null 2>&1; then
  nexus update --force
else
  curl -fsSL https://raw.githubusercontent.com/inizio/nexus/main/install.sh | bash
fi
if ! command -v nexus-daemon >/dev/null 2>&1; then
  echo "nexus-daemon was not found in PATH after install/update." >&2
  exit 1
fi
"""
        let result = await runShell(command)
        if result.exitCode != 0 {
            throw HostToolsSetupError.daemonInstallFailed(result.output)
        }
        return result.output.isEmpty
            ? "Nexus daemon install/update completed."
            : result.output
    }

    private static func checkNestedVirtualizationSupport() async -> Bool {
        let result = await runShell("sysctl -n kern.hv_support")
        if result.exitCode != 0 {
            return false
        }
        return result.output.trimmingCharacters(in: .whitespacesAndNewlines) == "1"
    }

    private static func commandExists(_ command: String) async -> Bool {
        let result = await runShell("command -v \(command)")
        return result.exitCode == 0
    }

    private static func shellQuote(_ value: String) -> String {
        if value.isEmpty { return "''" }
        let escaped = value.replacingOccurrences(of: "'", with: "'\"'\"'")
        return "'\(escaped)'"
    }

    private static func runPrivileged(_ command: String) async -> CommandResult {
        let escaped = command
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
        let script = "do shell script \"\(escaped)\" with administrator privileges"
        return await runProcess(
            executable: "/usr/bin/osascript",
            arguments: ["-e", script]
        )
    }

    private static func runShell(_ command: String) async -> CommandResult {
        await runProcess(
            executable: "/bin/zsh",
            arguments: ["-lc", command],
            includeCommonHomebrewPath: true
        )
    }

    private static func runProcess(
        executable: String,
        arguments: [String],
        includeCommonHomebrewPath: Bool = false
    ) async -> CommandResult {
        await withCheckedContinuation { continuation in
            let process = Process()
            process.executableURL = URL(fileURLWithPath: executable)
            process.arguments = arguments

            if includeCommonHomebrewPath {
                var env = ProcessInfo.processInfo.environment
                let existingPath = env["PATH"] ?? "/usr/bin:/bin:/usr/sbin:/sbin"
                let extras = ["/opt/homebrew/bin", "/usr/local/bin"]
                env["PATH"] = (extras + [existingPath]).joined(separator: ":")
                process.environment = env
            }

            let output = Pipe()
            process.standardOutput = output
            process.standardError = output

            do {
                try process.run()
            } catch {
                continuation.resume(
                    returning: CommandResult(
                        exitCode: 1,
                        output: error.localizedDescription
                    )
                )
                return
            }

            process.terminationHandler = { proc in
                let data = output.fileHandleForReading.readDataToEndOfFile()
                let text = String(data: data, encoding: .utf8)?
                    .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                continuation.resume(
                    returning: CommandResult(
                        exitCode: Int(proc.terminationStatus),
                        output: text
                    )
                )
            }
        }
    }
}

private struct CommandResult {
    let exitCode: Int
    let output: String
}
