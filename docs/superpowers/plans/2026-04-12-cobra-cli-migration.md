# Cobra CLI Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace manual `flag.FlagSet` dispatch in the nexus CLI with cobra, eliminating flag-order bugs and inconsistent positional-arg handling.

**Architecture:** Add `github.com/spf13/cobra` to `go.mod`; rewrite the top-level dispatch in `main.go` and `workspace.go` as cobra `Command` trees; keep all internal logic functions (`runInit`, `run`, `runRunCommand`, etc.) unchanged — only the dispatch layer changes.

**Tech Stack:** Go, `github.com/spf13/cobra` v1.8+

---

## Background

`packages/nexus/cmd/nexus/main.go` (~2179 lines) has a manual `switch command` at line ~62 that dispatches to per-command functions. Each function calls `flag.NewFlagSet(…)` and parses args manually. `flag.FlagSet` stops at the first non-flag argument — so `nexus init /path --force` silently misparses, treating `--force` as a second positional arg and exiting with a usage error. The fix we applied is a band-aid. Cobra handles this correctly by default.

`workspace.go` (~816 lines) contains the workspace subcommand functions (`runWorkspaceCreateCommand`, etc.), each using the same fragile `flag.NewFlagSet` pattern.

### Commands to migrate

| Command | File | Positional args | Named flags |
|---------|------|-----------------|-------------|
| `doctor` | `main.go` | none | `--report-json` |
| `init` | `main.go` | `[project-root]` | `--force` |
| `run` | `main.go` | `-- <cmd> [args...]` | `--backend`, `--timeout` |
| `list` | `workspace.go` | none | none |
| `create` | `workspace.go` | none | `--repo`, `--name`, `--agent-profile`, `--backend`, `--wait` |
| `start` | `workspace.go` | `<id>` | none |
| `stop` | `workspace.go` | `<id>` | none |
| `remove` | `workspace.go` | `<id>` | none |
| `fork` | `workspace.go` | `<id> <name>` | `--ref` |
| `shell` | `workspace.go` | `<id>` | `--timeout` |
| `exec` | `workspace.go` | `<id> -- <cmd> [args...]` | `--timeout` |
| `tunnel` | `workspace.go` | `<id>` | none |
| `pause` | `workspace.go` | `<id>` | none |
| `resume` | `workspace.go` | `<id>` | none |
| `restore` | `workspace.go` | `<id>` | none |

### File changes

- **Modify:** `packages/nexus/cmd/nexus/main.go` — replace `main()` dispatch switch + per-command flag parsing with cobra root + subcommand tree
- **Modify:** `packages/nexus/cmd/nexus/workspace.go` — replace each `runWorkspace*Command(args []string)` function with a cobra `*Command` constructor
- **Modify:** `packages/nexus/go.mod` / `go.sum` — add cobra
- **Modify:** `packages/nexus/cmd/nexus/main_test.go` — update test helpers that set `os.Args` to use `rootCmd.SetArgs()`
- **Modify:** `packages/nexus/cmd/nexus/workspace_test.go` — update usage assertion

---

## Task 1: Add cobra to go.mod

**Files:**
- Modify: `packages/nexus/go.mod`

- [ ] **Step 1: Add cobra**

```bash
cd packages/nexus
go get github.com/spf13/cobra@v1.8.0
```

Expected: `go.mod` now includes `github.com/spf13/cobra v1.8.0`.

- [ ] **Step 2: Verify build still passes**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore(deps): add cobra for CLI flag parsing"
```

---

## Task 2: Build the root cobra command in main.go

This task replaces the manual `main()` switch with a cobra root command. The internal logic functions (`run`, `runInit`, `runRunCommand`, etc.) are **not changed** — only the top-level dispatch.

**Files:**
- Modify: `packages/nexus/cmd/nexus/main.go`

- [ ] **Step 1: Add cobra import and root command**

At the top of `main.go`, add the import and replace the global `main()` body. Remove the `printUsage()` function and the `flag.NewFlagSet("doctor", …)` block. Replace the whole `main()` function and the dispatch switch with:

```go
import (
    // existing imports ...
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:          "nexus",
    SilenceUsage: true,
    SilenceErrors: true,
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Remove the old `main()` body (the `if len(os.Args) < 2` block and the `switch command` block) entirely.

- [ ] **Step 2: Add doctor subcommand**

Replace the inline `flag.NewFlagSet("doctor", …)` block with a cobra command. Add this `init()` block and `doctorCmd` variable to `main.go`:

```go
var doctorReportJSON string

var doctorCmd = &cobra.Command{
    Use:   "doctor",
    Short: "Run workspace health checks",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        projectRoot, err := filepath.Abs(".")
        if err != nil {
            return fmt.Errorf("resolve project root: %w", err)
        }
        return run(options{
            projectRoot: projectRoot,
            reportJSON:  strings.TrimSpace(doctorReportJSON),
        })
    },
}

func init() {
    doctorCmd.Flags().StringVar(&doctorReportJSON, "report-json", "", "write probe results as JSON to this path")
    rootCmd.AddCommand(doctorCmd)
}
```

- [ ] **Step 3: Add init subcommand**

Replace `runInitCommand(args []string)` (the manual loop parsing `--force`) with:

```go
var initForce bool

var initCmd = &cobra.Command{
    Use:   "init [project-root]",
    Short: "Scaffold .nexus in a git repository",
    Args:  cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        projectRoot := "."
        if len(args) == 1 {
            projectRoot = args[0]
        }
        abs, err := filepath.Abs(strings.TrimSpace(projectRoot))
        if err != nil {
            return fmt.Errorf("resolve project root: %w", err)
        }
        return runInit(initOptions{projectRoot: abs, force: initForce})
    },
}

func init() {
    initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing .nexus files")
    rootCmd.AddCommand(initCmd)
}
```

Delete the old `runInitCommand(args []string)` function.

- [ ] **Step 4: Add run subcommand**

Replace `runRunCommand(args []string)` dispatch with a cobra command. The `--` separator is preserved: cobra passes everything after `--` as positional args automatically.

```go
var runBackend string
var runTimeout time.Duration

var runCmd = &cobra.Command{
    Use:   "run [--backend name] [--timeout dur] -- <command> [args...]",
    Short: "Run a command in a new ephemeral workspace",
    Args:  cobra.ArbitraryArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) == 0 {
            return fmt.Errorf("command required after --")
        }
        return runRun(runBackend, runTimeout, args)
    },
}

func init() {
    runCmd.Flags().StringVar(&runBackend, "backend", "", "runtime backend override")
    runCmd.Flags().DurationVar(&runTimeout, "timeout", 10*time.Minute, "max time for the workspace run")
    rootCmd.AddCommand(runCmd)
}
```

Then rename the existing `runRunCommand(args []string)` internals to `runRun(backend string, timeout time.Duration, args []string) error` (or refactor in place to accept these parameters directly rather than re-parsing from `args`).

- [ ] **Step 5: Verify build**

```bash
cd packages/nexus && go build ./cmd/nexus/...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add packages/nexus/cmd/nexus/main.go
git commit -m "feat(cli): migrate doctor/init/run to cobra commands"
```

---

## Task 3: Migrate workspace commands in workspace.go

**Files:**
- Modify: `packages/nexus/cmd/nexus/workspace.go`

Replace every `runWorkspace*Command(args []string)` function with a cobra `*Command` constructor registered in an `init()` block.

- [ ] **Step 1: list command**

```go
var listCmd = &cobra.Command{
    Use:   "list",
    Short: "List workspaces",
    Args:  cobra.NoArgs,
    Run: func(cmd *cobra.Command, args []string) {
        listWorkspaces()
    },
}

func init() { rootCmd.AddCommand(listCmd) }
```

Move the body of the old `runWorkspaceListCommand` into a `listWorkspaces()` helper (or inline it in `Run`).

- [ ] **Step 2: create command**

```go
var createFlags struct {
    repo         string
    name         string
    agentProfile string
    backend      string
    wait         bool
}

var createCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new workspace",
    Args:  cobra.NoArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        return createWorkspace(createFlags.repo, createFlags.name, createFlags.agentProfile, createFlags.backend, createFlags.wait)
    },
}

func init() {
    createCmd.Flags().StringVar(&createFlags.repo, "repo", "", "path or URL of the git repository")
    createCmd.Flags().StringVar(&createFlags.name, "name", "", "workspace name")
    createCmd.Flags().StringVar(&createFlags.agentProfile, "agent-profile", "default", "agent profile")
    createCmd.Flags().StringVar(&createFlags.backend, "backend", "", "runtime backend override")
    createCmd.Flags().BoolVar(&createFlags.wait, "wait", true, "wait for workspace to become ready")
    rootCmd.AddCommand(createCmd)
}
```

Move the body of `runWorkspaceCreateCommand` into `createWorkspace(…)`.

- [ ] **Step 3: single-ID commands (start, stop, remove, pause, resume, restore, tunnel)**

These all have the same shape: one positional `<id>` argument. For each one:

```go
var startCmd = &cobra.Command{
    Use:   "start <id>",
    Short: "Start a stopped workspace",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        return startWorkspace(args[0])
    },
}
func init() { rootCmd.AddCommand(startCmd) }
```

Repeat for `stop`, `remove`, `pause`, `resume`, `restore`, `tunnel` — each with its own `Use` string, `Short` description, and a dedicated helper function extracted from the old `runWorkspace*Command` body.

- [ ] **Step 4: fork command (two positional args)**

```go
var forkRef string

var forkCmd = &cobra.Command{
    Use:   "fork <id> <name>",
    Short: "Fork a workspace into a new named worktree",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        return forkWorkspace(args[0], args[1], forkRef)
    },
}

func init() {
    forkCmd.Flags().StringVar(&forkRef, "ref", "", "git ref to base the fork on")
    rootCmd.AddCommand(forkCmd)
}
```

- [ ] **Step 5: shell command**

```go
var shellTimeout time.Duration

var shellCmd = &cobra.Command{
    Use:   "shell <id>",
    Short: "Open an interactive shell in a workspace",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        return shellWorkspace(args[0], shellTimeout)
    },
}

func init() {
    shellCmd.Flags().DurationVar(&shellTimeout, "timeout", 0, "session timeout (0 = no limit)")
    rootCmd.AddCommand(shellCmd)
}
```

- [ ] **Step 6: exec command**

```go
var execTimeout time.Duration

var execCmd = &cobra.Command{
    Use:   "exec <id> -- <command> [args...]",
    Short: "Run a non-interactive command in a workspace",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        if len(args) < 1 {
            return fmt.Errorf("workspace id required")
        }
        id := args[0]
        rest := args[1:]
        if len(rest) == 0 {
            return fmt.Errorf("command required after --")
        }
        return execWorkspace(id, execTimeout, rest)
    },
}

func init() {
    execCmd.Flags().DurationVar(&execTimeout, "timeout", 0, "command timeout (0 = no limit)")
    rootCmd.AddCommand(execCmd)
}
```

- [ ] **Step 7: Verify build and tests**

```bash
cd packages/nexus && go build ./cmd/nexus/... && go test ./cmd/nexus/...
```

Expected: build passes. Tests may fail — fix in Task 4.

- [ ] **Step 8: Commit**

```bash
git add packages/nexus/cmd/nexus/workspace.go
git commit -m "feat(cli): migrate workspace subcommands to cobra"
```

---

## Task 4: Update tests

**Files:**
- Modify: `packages/nexus/cmd/nexus/main_test.go`
- Modify: `packages/nexus/cmd/nexus/workspace_test.go`

Current tests set `os.Args` and call `main()`. With cobra, they should call `rootCmd.SetArgs(tc.args); err := rootCmd.Execute()`.

- [ ] **Step 1: Update main_test.go helpers**

Replace all test patterns like:
```go
os.Args = append([]string{"nexus"}, tc.args...)
// invoke main() or check stderr
```

With:
```go
rootCmd.SetArgs(tc.args)
err := rootCmd.Execute()
```

For tests that assert exit codes, capture error instead: a non-nil error from `Execute()` corresponds to exit code 1; cobra's `SilenceErrors: true` means errors are returned not printed.

- [ ] **Step 2: Update workspace_test.go usage assertion**

`TestPrintUsageIncludesFlatWorkspaceCommands` currently checks a hardcoded `printUsage()` string. Replace it with a test that calls `rootCmd.UsageString()` and asserts the command names are present, e.g.:

```go
func TestRootCommandIncludesWorkspaceSubcommands(t *testing.T) {
    usage := rootCmd.UsageString()
    for _, name := range []string{"create", "list", "start", "stop", "shell", "exec", "run", "fork", "doctor", "init"} {
        if !strings.Contains(usage, name) {
            t.Errorf("usage missing subcommand %q", name)
        }
    }
}
```

- [ ] **Step 3: Run all tests**

```bash
cd packages/nexus && go test ./cmd/nexus/... -v 2>&1 | tail -20
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add packages/nexus/cmd/nexus/main_test.go packages/nexus/cmd/nexus/workspace_test.go
git commit -m "test(cli): update CLI tests for cobra dispatch"
```

---

## Task 5: CI check

- [ ] **Step 1: Verify full Go test suite**

```bash
cd packages/nexus && go test ./...
```

Expected: all pass, no regressions.

- [ ] **Step 2: Smoke-test the binary**

```bash
cd packages/nexus && go run ./cmd/nexus -- --help
go run ./cmd/nexus -- doctor --help
go run ./cmd/nexus -- init --help
go run ./cmd/nexus -- create --help
```

Expected: each prints structured help with flags listed.

- [ ] **Step 3: Verify `nexus init /tmp/test --force` works**

```bash
mkdir -p /tmp/nexus-cobra-test && cd /tmp/nexus-cobra-test && git init && cd - 
cd packages/nexus && go run ./cmd/nexus -- init /tmp/nexus-cobra-test --force
```

Expected: exits 0, `.nexus/` created in `/tmp/nexus-cobra-test`.

```bash
rm -rf /tmp/nexus-cobra-test
```
