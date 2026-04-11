# Host auth bundle

The **host auth bundle** is an optional **base64-encoded gzip+tar** that the runtime unpacks in the guest so AI-tool configs (opencode, codex, claude, etc.) exist under `$HOME` in the workspace VM. It is how the **CLI** and **SDK** can supply the same kind of content without the daemon reading its own `$HOME`.

## Who builds what

| Source | Behavior |
|--------|----------|
| **`nexus create`** | Runs `authbundle.BuildFromHome()` on the **machine running the CLI** and sends the result as `hostAuthBundle` on `workspace.create`. |
| **`nexus auth-bundle`** | Prints the same base64 string (or writes `--output file`) for CI/scripts—use as `WorkspaceCreateSpec.hostAuthBundle`. |
| **SDK `workspace.create`** | If you omit `hostAuthBundle`, **nothing** is sent. The daemon does **not** re-filter or replace it; build the tarball yourself or use `nexus auth-bundle`. |

**RPC validation:** `ResolveFromOptions` accepts the blob only if base64 decodes and decoded size ≤ **4 MiB**.

## CLI pack rules (`BuildFromHome`)

Roots under `$HOME` (only these trees are walked):

- `.config/opencode/`
- `.config/codex/`
- `.codex/`
- `.config/openai/`
- `.claude/`

**Included files:** regular files only (symlinks skipped). Typical extensions: `.json`, `.yaml`, `.yml`. Under `.claude/`, `CLAUDE.md` / `claude.md` at the tree root is included; **`.claude/projects/**` is excluded**.

**Per-file cap:** **512 KiB**; larger files are skipped.

**Total:** gzip payload before base64 must stay ≤ **4 MiB** or `BuildFromHome` returns empty.

Implementation: `packages/nexus/pkg/runtime/authbundle/bundle.go`.

## CI checklist

1. Set `NEXUS_ENDPOINT` / `NEXUS_TOKEN` (or your auth).
2. Run `nexus auth-bundle > bundle.b64` on the runner that should own the config (or build a matching tarball offline).
3. `workspace.create` with `hostAuthBundle` set to the file contents (trim whitespace).
4. Size CI timeouts for **cold Firecracker** (first start can take several minutes)—see [Operations](operations.md).

## Related

- [CLI](cli.md) — `nexus auth-bundle`, `nexus doctor` and backends
- [SDK](sdk.md) — `WorkspaceCreateSpec`
- [Operations](../tutorials/operations.md) — doctor latency, backends, paths
- Repository [AGENTS.md](https://github.com/inizio/nexus/blob/main/AGENTS.md) — remote-first constraints
