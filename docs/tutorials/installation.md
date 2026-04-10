# Installation

## Fast Path

Install the CLI in one line:

```bash
curl -fsSL https://raw.githubusercontent.com/inizio/nexus/main/install.sh | bash
```

Verify:

```bash
nexus --help
```

## Binary Install

If you do not want to use `go install`, download a release archive from [GitHub releases](https://github.com/IniZio/nexus/releases), extract it, and place the `nexus` binary on your `PATH`.

## First Run

```bash
cd /path/to/project
nexus init
nexus create
nexus list
nexus start <workspace-id>
```

Use the workspace id printed by `nexus create` when running `start`, `ssh`, `tunnel`, `stop`, or `remove`.

## Install From Source (contributors)

```bash
git clone https://github.com/inizio/nexus
cd nexus
pnpm install
task build
```

## Next

- [CLI Reference](../reference/cli.md)
- [SDK Reference](../reference/sdk.md)
- [Workspace Config Reference](../reference/workspace-config.md)
