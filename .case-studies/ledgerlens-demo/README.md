# LedgerLens Demo

Minimal **case study** project for Nexus: three parallel workstreams that justify isolation (separate branches, git worktrees, or Nexus workspaces) without shipping real product code.

## Scenario

**LedgerLens** is a fake product that ingests bank CSV exports and shows a read-only dashboard.

| Workstream | Focus | Suggested branch prefix |
|------------|--------|-------------------------|
| **API** | Validation rules, CSV ingest HTTP API | `feat/ledgerlens-rules-api` |
| **UI** | Dashboard shell and charts | `feat/ledgerlens-ui` |
| **Infra** | Compose, probes, doctor suites | `feat/ledgerlens-infra` |

**Done** for this folder means: repo metadata under `.nexus/` matches `nexus init` conventions, and a developer can follow the commands below against a running Nexus daemon (remote or local).

## Prerequisites

- Nexus CLI installed and configured to talk to your **workspace daemon** (daemon may run on another machine than the CLI).
- This directory is a normal path on disk; use an **absolute** `--project-root` when the CLI requires it.

## Initialize project metadata

From this directory:

```bash
nexus init --force
```

Or pass a path explicitly (positional or `--project-root`); the CLI resolves it to an absolute path before writing `.nexus/`. `--force` overwrites generated `.nexus` files when you are refreshing the scaffold.

`nexus exec` and `nexus doctor` require a **literal absolute** `--project-root` flag value (they do not accept relative paths).

## Create a remote workspace

```bash
cd /path/to/ledgerlens-demo
nexus create
nexus list
nexus start <workspace-id>
```

Use the printed `<workspace-id>` with `ssh`, `tunnel`, `stop`, and `remove`.

### Host auth bundle (CLI vs SDK)

- **`nexus create`** builds a tarball on **the machine running the CLI** (`authbundle.BuildFromHome`) and sends it as `hostAuthBundle`. Only **registry-allowed** files are packed (see `AGENTS.md` / `docs/reference/cli.md`: fixed roots, `.json`/`.yaml`/`.yml`, per-file size cap, no `.claude/projects/**`).
- **SDK `workspace.create`:** omitting `hostAuthBundle` sends nothing; the daemon does not invent a bundle. Custom bundles are not re-filtered server-side—match the CLI registry if you need the same contents.

## Worktrees (parallel local branches)

Isolate workstreams without switching branches in a single working tree:

```bash
git worktree add -b feat/ledgerlens-rules-api ../ledgerlens-demo-rules-api
git worktree add -b feat/ledgerlens-ui ../ledgerlens-demo-ui
```

Each worktree can run `nexus init --project-root "$(pwd)"` if `.nexus` should be tracked per branch, or share one clone—pick one convention per team.

## Tunnel, exec, doctor

After `nexus start <workspace-id>`:

- **`nexus tunnel <workspace-id>`** — forwards compose ports discovered from `docker-compose.yml` / `docker-compose.yaml` in the project root; blocks until Ctrl-C.
- **`nexus exec --project-root <abs-path> -- <command>`** — runs project-scoped automation with the same constraints as doctor (absolute `--project-root`, timeout optional).
- **`nexus doctor --project-root <abs-path> --suite <name>`** — runs executable probes under `.nexus/probe/` and checks under `.nexus/check/`. Suite name is your label (for example `local`).

Example:

```bash
nexus doctor --project-root "$(pwd -P)" --suite local
```

On a **firecracker** backend, the first `doctor` run can spend a long time in internal runtime-bootstrap probes (for example `docker info` and package installs inside the guest). Expect minutes, not seconds, before suite completion.

(Add a `docker-compose.yml` at the project root when you want tunnel port discovery; this demo ships without one.)

## Layout

- `.nexus/workspace.json` — version `1` only (see `docs/reference/workspace-config.md`).
- `.nexus/lifecycles/*.sh` — setup / start / teardown hooks.
- `.nexus/probe/`, `.nexus/check/` — doctor discovery.

## SDK note

When creating workspaces programmatically, pass **`hostAuthBundle`** explicitly if the guest should receive client-side AI tool configs; otherwise the create request sends no bundle.
