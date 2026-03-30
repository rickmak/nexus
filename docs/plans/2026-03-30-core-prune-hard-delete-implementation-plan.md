# Core-Only Hard Prune Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Hard-delete all non-core Enforcer/Boulder/plugin surfaces and leave a repository focused on `workspace-daemon` + `workspace-sdk` only.

**Architecture:** Apply a single-shot deletion pass with a strict keep-list, then repair root wiring (Taskfile/workspace manifests/CI/docs) in the same PR so daemon/sdk build and test pipelines remain intact.

**Tech Stack:** Go (`workspace-daemon`), TypeScript (`workspace-sdk`), pnpm workspace, Taskfile, GitHub Actions.

---

### Task 1: Snapshot and enforce keep/delete scope

**Files:**
- Modify: `docs/plans/2026-03-30-core-prune-design.md` (if adjustments are needed)

**Step 1: Confirm keep-list directories exist**

Run:

```bash
ls packages/workspace-daemon packages/workspace-sdk
```

Expected: both paths exist.

**Step 2: Enumerate delete-set paths and verify they exist before delete**

Run:

```bash
ls -d packages/enforcer packages/opencode packages/opencode-plugin packages/claude packages/cursor .opencode 2>/dev/null
```

Expected: existing non-core paths listed (missing paths are acceptable).

**Step 3: Commit planning doc updates (if any)**

```bash
git add docs/plans/2026-03-30-core-prune-design.md
git commit -m "docs(plan): finalize core-only hard prune scope"
```

### Task 2: Hard-delete non-core package/module directories

**Files:**
- Delete: `packages/enforcer/**`
- Delete: `packages/opencode/**`
- Delete: `packages/opencode-plugin/**`
- Delete: `packages/claude/**`
- Delete: `packages/cursor/**`
- Delete: `.opencode/**`
- Delete: `boulder/**` (if present)

**Step 1: Delete directories from working tree**

Run with exact existing paths only.

**Step 2: Verify deletion from git status**

Run:

```bash
git status --short
```

Expected: deleted entries for non-core directories.

**Step 3: Commit deletion chunk**

```bash
git add -A
git commit -m "refactor(repo): remove non-core enforcer and plugin packages"
```

### Task 3: Rewire root build/task/workspace configuration

**Files:**
- Modify: `Taskfile.yml`
- Modify: `package.json`
- Modify: `pnpm-workspace.yaml`
- Modify: root ts/go config files only if they reference deleted packages

**Step 1: Remove task targets for deleted modules**

Update Taskfile so `build`, `test`, `lint`, `ci` call only daemon/sdk related tasks.

**Step 2: Remove workspace manifest references that expect deleted packages**

If explicit package references exist, remove them. If glob-based, ensure root scripts do not assume deleted modules.

**Step 3: Run root install/build wiring check**

Run:

```bash
pnpm install
```

Expected: succeeds without missing deleted-package references.

**Step 4: Commit wiring changes**

```bash
git add Taskfile.yml package.json pnpm-workspace.yaml
git commit -m "build(core): trim root tasks and workspace wiring to daemon and sdk"
```

### Task 4: Remove non-core runtime/type/schema couplings

**Files:**
- Modify: `packages/workspace-daemon/pkg/handlers/workspace_ready.go`
- Modify: `packages/workspace-daemon/pkg/handlers/workspace_ready_test.go`
- Modify: `packages/workspace-sdk/src/types.ts`
- Modify: `schemas/workspace.v1.schema.json` (only if needed by sdk/daemon scope)

**Step 1: Remove plugin-auth identities no longer valid for core-only scope**

Update daemon/sdk/schema types to remove stale identities tied solely to deleted modules.

**Step 2: Keep ACP behavior optional and capability-based**

Do not reintroduce hard dependencies on removed plugin packages.

**Step 3: Run focused tests/typecheck**

Run:

```bash
cd packages/workspace-daemon && go test ./pkg/handlers -run WorkspaceReady -v
cd packages/workspace-sdk && pnpm exec tsc --noEmit
```

Expected: pass.

**Step 4: Commit runtime/type cleanup**

```bash
git add packages/workspace-daemon/pkg/handlers/workspace_ready.go packages/workspace-daemon/pkg/handlers/workspace_ready_test.go packages/workspace-sdk/src/types.ts schemas/workspace.v1.schema.json
git commit -m "refactor(core): remove non-core auth and readiness couplings"
```

### Task 5: Hard-prune non-core docs and add migration note

**Files:**
- Modify/Delete: docs pages referencing Enforcer/Boulder/plugins
- Keep/Update: `docs/reference/workspace-daemon.md`, `docs/reference/workspace-sdk.md`, `docs/reference/workspace-config.md`
- Create: `docs/dev/migration-core-prune.md`

**Step 1: Delete non-core docs in docs tree**

Remove docs that exclusively describe removed modules.

**Step 2: Update retained docs to daemon+sdk-only framing**

Ensure no claims about deleted modules remain.

**Step 3: Add migration note**

Include:
- what was removed
- why
- where core functionality now lives.

**Step 4: Run docs reference sanity checks**

Run:

```bash
grep -RIn "enforcer\|boulder\|opencode-plugin\|packages/claude\|packages/cursor" docs || true
```

Expected: only intentional migration-note references remain.

**Step 5: Commit docs prune**

```bash
git add docs
git commit -m "docs(core): prune non-core docs and add migration note"
```

### Task 6: Update CI workflows to core-only

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: other workflow files if they target deleted modules

**Step 1: Remove deleted-module build/test jobs**

Keep jobs needed for daemon/sdk verification only.

**Step 2: Run local workflow command parity check via Taskfile**

Run the root core-only CI command.

**Step 3: Commit CI updates**

```bash
git add .github/workflows/ci.yml
git commit -m "ci(core): limit ci scope to workspace daemon and sdk"
```

### Task 7: Final verification gate

**Files:**
- Modify: any touched files if fixes are required

**Step 1: Run daemon full tests**

```bash
cd packages/workspace-daemon && go test ./...
```

Expected: PASS.

**Step 2: Run sdk full checks**

```bash
cd packages/workspace-sdk && pnpm exec tsc --noEmit
cd packages/workspace-sdk && pnpm exec jest --runInBand
```

Expected: PASS.

**Step 3: Run core-only root CI task**

```bash
task ci
```

Expected: PASS with no references to deleted modules.

**Step 4: Validate no dangling references in root configs/docs**

```bash
grep -RIn "packages/enforcer\|packages/opencode\|packages/claude\|packages/cursor" .github Taskfile.yml package.json pnpm-workspace.yaml docs || true
```

Expected: no stale references outside migration note.

**Step 5: Final commit**

```bash
git add -A
git commit -m "refactor(core): hard-prune repository to workspace daemon and sdk"
```
