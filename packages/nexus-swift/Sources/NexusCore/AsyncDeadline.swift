import Foundation

/// Thrown when `withSeconds` elapses before `operation` completes.
public enum AsyncDeadlineError: Error, Equatable {
    case exceeded(seconds: UInt64)
}

/// Races `operation` against a sleep; cancels the group when the first branch finishes.
/// Used for startup and daemon-side-effect caps so the UI cannot hang indefinitely.
public enum AsyncDeadline {
    public static func withSeconds<T>(
        _ seconds: UInt64,
        operation: @escaping () async throws -> T
    ) async throws -> T {
        try await withThrowingTaskGroup(of: T.self) { group in
            group.addTask { try await operation() }
            group.addTask {
                try await Task.sleep(nanoseconds: seconds * 1_000_000_000)
                throw AsyncDeadlineError.exceeded(seconds: seconds)
            }
            defer { group.cancelAll() }
            return try await group.next()!
        }
    }

    /// Same race as `withSeconds`, but runs `operation` on the main actor.
    ///
    /// Use this when `operation` is `@MainActor` work (e.g. `AppState` methods). The plain
    /// `withSeconds` schedules `operation` on an arbitrary executor; awaiting `group.next()`
    /// from the main actor while the inner work must hop to the main actor can deadlock.
    @MainActor
    public static func withSecondsOnMainActor<T>(
        _ seconds: UInt64,
        operation: @escaping @MainActor () async throws -> T
    ) async throws -> T {
        StartupTrace.checkpoint("deadline.main.enter", "\(seconds)s")
        return try await withThrowingTaskGroup(of: T.self) { group in
            group.addTask { @MainActor in
                StartupTrace.checkpoint("deadline.main.work_begin")
                let out = try await operation()
                StartupTrace.checkpoint("deadline.main.work_done")
                return out
            }
            group.addTask {
                try await Task.sleep(nanoseconds: seconds * 1_000_000_000)
                throw AsyncDeadlineError.exceeded(seconds: seconds)
            }
            defer { group.cancelAll() }
            return try await group.next()!
        }
    }
}
