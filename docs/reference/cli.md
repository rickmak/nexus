# CLI Reference

Direct control for creating, starting, and accessing isolated workspaces.

## Install and first run

```bash
curl -fsSL https://raw.githubusercontent.com/inizio/nexus/main/install.sh | bash
cd /path/to/project
nexus init && nexus create && nexus list && nexus start <workspace-id>
```

`nexus create` prints the workspace id used by `start`, `ssh`, `tunnel`, `stop`, `remove`.

**Create and host auth bundle:** The `nexus workspace create` command (invoked by `nexus create`) builds a tarball of local AI-tool config directories from **the machine running the CLI** (`authbundle.BuildFromHome`), base64-encodes it, and sends it as `hostAuthBundle` on `workspace.create`. The daemon never substitutes its own `$HOME` for that step. Omitting the CLI (e.g. pure SDK create without `hostAuthBundle`) means no config tarball is sent.

## Common commands

```bash
nexus init [project-root] [--force]
nexus create [--backend firecracker]
nexus list
nexus start|stop|remove|ssh|tunnel <workspace-id>
nexus fork --id <workspace-id> --name <child-name> [--ref <child-ref>]
nexus exec --project-root <abs-path> [--timeout 10m] -- <command> [args...]
nexus doctor --project-root <abs-path> --suite <name> \
  [--compose-file docker-compose.yml] [--required-host-ports 5173,5174] [--report-json path]
```

- `**nexus ssh`:** optional `--shell`, `--command` (non-interactive one shot).
- `**nexus tunnel`:** applies compose port forwards; blocks until Ctrl-C.
- `**nexus init`:** default path is cwd; `--force` overwrites `.nexus` scaffold. Host setup may escalate privileges (`sudo`); use `sudo -E nexus init --force` only where non-interactive sudo is unavailable.

## Related

- SDK: `[sdk.md](sdk.md)`
- Workspace config: `[workspace-config.md](workspace-config.md)`

