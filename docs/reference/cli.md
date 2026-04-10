# CLI Reference

Nexus CLI is a direct control surface for creating, starting, accessing, and testing isolated workspaces.

## Common Operations

```bash
nexus init
nexus create
nexus start <workspace-id>
nexus tunnel <workspace-id>
nexus ssh <workspace-id>
nexus stop <workspace-id>
nexus remove <workspace-id>
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/inizio/nexus/main/install.sh | bash
```

## Quick Start

```bash
cd /path/to/project
nexus init
nexus create
nexus list
nexus start <workspace-id>
```

`nexus create` prints the workspace id used by `start`, `ssh`, `tunnel`, `stop`, and `remove`.

## Usage

```bash
nexus <list|create|start|stop|remove|fork|ssh|tunnel>
nexus init [project-root] [--force]
nexus exec --project-root <abs-path> [--timeout 10m] -- <command> [args...]
nexus doctor --project-root <abs-path> --suite <name> [--compose-file docker-compose.yml] [--required-host-ports 5173,5174,8000] [--report-json path]
```

## Workspace Commands

### `nexus create`

Creates a workspace for the current repository path.

```bash
nexus create
```

Flags:

- `--backend` (optional): runtime backend override (currently `firecracker`).

### `nexus list`

Lists all workspaces.

```bash
nexus list
```

### `nexus start`

Starts a workspace by id.

```bash
nexus start <workspace-id>
```

### `nexus stop`

Stops a workspace by id.

```bash
nexus stop <workspace-id>
```

### `nexus remove`

Removes a workspace by id.

```bash
nexus remove <workspace-id>
```

### `nexus fork`

Forks an existing workspace.

```bash
nexus fork --id <workspace-id> --name <child-name> [--ref <child-ref>]
```

### `nexus ssh`

Opens an interactive shell for a workspace.

```bash
nexus ssh <workspace-id>
```

Flags:

- `--shell` (optional): shell executable (default `bash`).
- `--command` (optional): run one command and exit.

### `nexus tunnel`

Applies compose port forwards and blocks until interrupted.

```bash
nexus tunnel <workspace-id>
```

`tunnel` is blocking; press Ctrl-C to close created tunnels.

## Project Commands

### `nexus init`

Initializes `.nexus/` in the project root.

```bash
nexus init
nexus init /absolute/path/to/project
nexus init --force
```

Behavior:

- `nexus init` with no path uses the current directory.
- `--force` overwrites generated `.nexus` scaffold files.
- Host setup preflight attempts privilege escalation automatically (root, `sudo -n`, or interactive sudo).
- Manual `sudo -E nexus init --force` is only needed in non-interactive environments where automatic escalation is unavailable.

### `nexus exec`

Runs one command in the Nexus workspace execution context.

```bash
nexus exec --project-root "$(pwd)" -- pnpm test
```

### `nexus doctor`

Runs configured probes and tests for workspace readiness and behavior.

```bash
nexus doctor --project-root "$(pwd)" --suite local
```

## Related Docs

- SDK client usage: `docs/reference/sdk.md`
- Project config schema: `docs/reference/workspace-config.md`
- Architecture overview: `docs/explanation/architecture.md`

