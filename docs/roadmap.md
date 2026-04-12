# Roadmap

Focus: remote workspace core quality and usability.

**Verify:** `packages/nexus` — `go test ./...`; `packages/sdk/js` — `pnpm exec tsc --noEmit` and `pnpm exec jest --runInBand`.

---

## Implemented

- Workspace lifecycle: create, start, stop, remove, pause, resume, restore, fork
- `nexus run` — ephemeral workspace: create → exec → remove
- `nexus shell` / `nexus exec` — interactive and non-interactive PTY sessions
- Host credential forwarding (git config, SSH keys) auto-bundled on `create` / `run`
- Tunnel: manual port forward (`add` / `stop` / `list`), compose port detection (`nexus tunnel`)
- SDK: single `WorkspaceClient` with auto-transport (Node/browser); `start()` returns handle
- SDK: `WorkspaceHandle` — `exec`, filesystem ops, `tunnel`, `ready`
- SDK: `client.shell` — full PTY session API

---

## Planned improvements

Priority order: **P0 reliability > P1 UX consistency > P2 surface simplification**.

### P0 — Typed RPC registry (`docs/superpowers/plans/2026-04-12-typed-rpc-registry.md`)

Add a generic `TypedRegister[Req, Res]` function to the Go RPC registry so the dispatch layer handles JSON unmarshalling centrally. Change all handler functions to accept their typed params struct directly. Add an `RPCSchema` interface to the TypeScript SDK and overload `client.request()` so method-name typos and param/result type mismatches are caught at compile time.

### P1 — Cobra CLI (`docs/superpowers/plans/2026-04-12-cobra-cli-migration.md`)

Replace manual `flag.FlagSet` dispatch with cobra. Eliminates flag-order bugs (e.g. `nexus init /path --force` silently failing), generates consistent `--help` output, and handles positional vs. named args correctly across all subcommands.

### P2 — Fold `capabilities.list` into `node.info` (`docs/superpowers/plans/2026-04-12-fold-capabilities-into-node-info.md`)

`node.info` already returns the capabilities array. Remove the redundant `capabilities.list` RPC endpoint and update E2E callers to use `node.info`.

---

## Vision / not yet implemented

### Live port detection

Auto-forward newly opened ports during workspace runtime, similar to VS Code's port panel. The daemon would watch listening sockets inside the workspace and emit events as processes start or stop binding ports. Currently port forwarding is one-shot: `nexus tunnel` reads compose-declared ports at call time; arbitrary process ports require `tunnel.add()` manually.

### Workspace-configured default port rules

Define port forwarding rules in `.nexus/workspace.json` (e.g. always forward port 3000). The `spotlight.applyDefaults` RPC stub was removed because it had no implementation. Resurrect once the schema and daemon behaviour are defined.

### Resumable `nexus run`

Long-running ephemeral jobs that survive client disconnects — the workspace is removed only after the command exits, even across reconnections.