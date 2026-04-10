# Workspace Config

Nexus uses file-driven defaults with a minimal config file.

For most projects, `.nexus/workspace.json` can be omitted or kept minimal.

## File Location

- `.nexus/workspace.json` in project root

## Supported Fields

Only these fields are supported:

```json
{
  "$schema": "./schemas/workspace.v1.schema.json",
  "version": 1
}
```

- `$schema` is optional.
- `version` is optional; if omitted, Nexus defaults it to `1`.

All other fields are hard-cut and ignored/rejected by strict parsing.

## Convention Defaults

Nexus behavior is driven by files and scripts:

- Lifecycle scripts:
  - `.nexus/lifecycles/setup.sh`
  - `.nexus/lifecycles/start.sh`
  - `.nexus/lifecycles/teardown.sh`
- Doctor probes:
  - executable scripts under `.nexus/probe/`
- Doctor checks:
  - executable scripts under `.nexus/check/`
- Tunneling:
  - compose ports discovered from `docker-compose.yml`/`docker-compose.yaml` when using `nexus tunnel <workspace-id>`

## Runtime Selection

Runtime is selected automatically by Nexus:

- Linux: Firecracker-first
- macOS: Firecracker via Lima when available, otherwise seatbelt fallback

Project-level runtime selection fields in `workspace.json` are not supported.