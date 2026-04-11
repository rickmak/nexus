# Agent Guidelines

## Project Overview

Nexus remote workspace core: **Workspace Daemon** (Go, `packages/nexus`) and **Workspace SDK** (TypeScript, `packages/sdk/js`). Keep changes centered on those packages; do not reintroduce removed non-core surfaces.

## Remote-First Architecture

**The daemon may run on a different machine than the user.** Design and verify under that assumption.

- Daemon host paths are not user paths; do not read user credentials from the daemon’s `$HOME` and assume they belong to the user.
- Symlink-based credential tricks break when the daemon is remote; user-owned secrets should travel via RPC (`workspace.create` fields, `AuthBinding`, auth relay at exec time, or explicit client-supplied payloads).

**Host auth bundle (AI tool configs in the guest):** Runtime resolution is `packages/nexus/pkg/runtime/authbundle` → `ResolveFromOptions`. It accepts **only** a client-supplied `host_auth_bundle` (base64 of gzip+tar), validated and capped at **4MiB decoded**. It does **not** read the daemon filesystem for that bundle. The `nexus workspace create` CLI calls `BuildFromHome()` **on the machine running the CLI** and sends the result as `hostAuthBundle` in the create spec—so the bundle always reflects the user’s machine when using the CLI, not the daemon’s disk. Programmatic clients that omit `hostAuthBundle` get no bundle (guest may still install CLIs based on daemon `PATH` during bootstrap, but no copied `~/.config` trees from any host).

**Lifecycle handlers** (`pause`, `resume`, `fork`, …) call `EnsureLocalRuntimeWorkspace` with an **empty** auth struct so they never re-inject a daemon-side bundle.

Flag any new feature that reads user-owned data from the daemon filesystem without an explicit client-supplied or relayed payload.

## Enforcement

Complete work fully; verify builds, tests, types, and lint; provide evidence; use isolated worktrees for features (not the main worktree). If stopping early, list what is undone, why, and what the user should do next.

## Documentation

User-facing docs live under `docs/`: `tutorials/`, `reference/`, `dev/` (contributing, roadmap). Only document implemented behavior. Do not document removed module surfaces as current capabilities.

```text
docs/
├── index.md
├── tutorials/
├── reference/   (cli, sdk, workspace-config)
└── dev/         (contributing, roadmap)
```

