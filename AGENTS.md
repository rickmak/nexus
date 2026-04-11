# Agent Guidelines

## Project Overview

Nexus remote workspace core: **Workspace Daemon** (Go, `packages/nexus`) and **Workspace SDK** (TypeScript, `packages/sdk/js`). Keep changes centered on those packages; do not reintroduce removed non-core surfaces.

## Remote-First Architecture

**The daemon may run on a different machine than the user.** Design and verify under that assumption.

- Daemon host paths are not user paths; do not read user credentials from the daemon’s `$HOME` and assume they belong to the user.
- Symlink-based credential tricks break when the daemon is remote; user-owned secrets should travel via RPC (`workspace.create` fields, `AuthBinding`, auth relay at exec time, or explicit client-supplied payloads).

**Host auth bundle (AI tool configs in the guest):** `packages/nexus/pkg/runtime/authbundle` → `ResolveFromOptions` accepts **only** a client-supplied `host_auth_bundle` (base64 gzip+tar), validated and capped at **4MiB decoded**; it never reads the daemon disk for that payload.

`BuildFromHome()` (used by `nexus workspace create` on the **CLI host**) walks fixed roots under `$HOME` (`.config/opencode`, `.config/codex`, `.codex`, `.config/openai`, `.claude`) but **includes only registry-allowed files**: regular files only (no symlinks), **≤512KiB each**, extensions `.json`/`.yaml`/`.yml` (plus `CLAUDE.md` at `.claude/` root), and **excludes** `.claude/projects/**`. Anything else (e.g. `.pem`, caches, binaries) is skipped so the archive stays small and predictable.

SDK-supplied `hostAuthBundle` is **not** re-filtered by the daemon—only size/base64 checks apply; match the CLI rules when building your own tarball for parity.

Programmatic clients that omit `hostAuthBundle` send no bundle (guest may still install CLIs based on daemon `PATH` during bootstrap).

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

