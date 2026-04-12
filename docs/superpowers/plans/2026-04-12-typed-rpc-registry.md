# Typed RPC Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace loose `json.RawMessage` / string-keyed RPC dispatch with a typed registry in Go and a typed schema map in TypeScript, making method-name typos and param/result type mismatches compile errors on both sides.

**Architecture:** (1) Add a generic `TypedRegister[Req, Res]` function to `pkg/server/rpc/registry.go` that does the JSON unmarshal centrally. (2) Update every `Handle*` function to accept its typed params struct directly instead of `json.RawMessage`. (3) Update all registrations in `rpc_handlers.go` to use `TypedRegister`. (4) Add an `RPCSchema` interface in the TypeScript SDK and overload `client.request()` to be typed for known methods.

**Tech Stack:** Go 1.21+ (generics), TypeScript

---

## Background

Every RPC handler today follows this pattern:

```go
// registration (rpc_handlers.go)
r.Register("workspace.start", func(_ context.Context, _ string, params json.RawMessage, _ any) (interface{}, *rpckit.RPCError) {
    return handlers.HandleWorkspaceStart(ctx, params, s.workspaceMgr)
})

// handler (workspace_manager.go)
func HandleWorkspaceStart(_ context.Context, params json.RawMessage, mgr *workspacemgr.Manager) (*WorkspaceStartResult, *rpckit.RPCError) {
    var p WorkspaceStartParams
    if err := json.Unmarshal(params, &p); err != nil {
        return nil, rpckit.ErrInvalidParams
    }
    // ...
}
```

Problems:
- Every handler duplicates `json.Unmarshal` boilerplate
- A typo in the method string at registration compiles fine
- TypeScript callers write `client.request<WorkspaceStartResult>('workspace.start', {id})` — the method string and the generic type are unrelated to the compiler

After this plan, registration becomes:

```go
rpc.TypedRegister(r, "workspace.start", func(ctx context.Context, req handlers.WorkspaceStartParams) (*handlers.WorkspaceStartResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceStart(ctx, req, s.workspaceMgr)
})
```

And TypeScript becomes:

```ts
// Compile error if method or params don't match schema
const result = await client.request('workspace.start', { id: '...' });
// result is WorkspaceStartResult — no generic needed
```

### File map

- **Modify:** `packages/nexus/pkg/server/rpc/registry.go` — add `TypedRegister` generic function
- **Modify:** `packages/nexus/pkg/handlers/workspace_manager.go` — change all `Handle*` signatures
- **Modify:** `packages/nexus/pkg/handlers/auth_relay.go` — change `HandleAuthRelayRevoke`
- **Modify:** `packages/nexus/pkg/handlers/exec.go` — change `HandleExec`, `HandleExecWithAuthRelay`
- **Modify:** `packages/nexus/pkg/handlers/fs.go` — change all 7 fs handlers
- **Modify:** `packages/nexus/pkg/handlers/git.go` — change `HandleGitCommand`
- **Modify:** `packages/nexus/pkg/handlers/node.go` — change `HandleNodeInfo` (drops unused params arg)
- **Modify:** `packages/nexus/pkg/handlers/os_picker.go` — change `HandlePickDirectory`
- **Modify:** `packages/nexus/pkg/handlers/service.go` — change `HandleServiceCommand`
- **Modify:** `packages/nexus/pkg/handlers/spotlight.go` — change all 4 spotlight handlers
- **Modify:** `packages/nexus/pkg/handlers/workspace_info.go` — change `HandleWorkspaceInfo`
- **Modify:** `packages/nexus/pkg/handlers/workspace_local.go` — change `HandleWorkspaceSetLocalWorktree`
- **Modify:** `packages/nexus/pkg/handlers/workspace_ready.go` — change `HandleWorkspaceReady`
- **Modify:** `packages/nexus/pkg/handlers/workspace_relations.go` — change `HandleWorkspaceRelationsList`
- **Modify:** `packages/nexus/pkg/server/rpc_handlers.go` — replace all `r.Register(…)` blocks with `rpc.TypedRegister`
- **Create:** `packages/sdk/js/src/rpc/schema.ts` — `RPCSchema` interface
- **Modify:** `packages/sdk/js/src/client.ts` — overloaded `request()` signature

---

## Task 1: Add TypedRegister to the Go RPC registry

**Files:**
- Modify: `packages/nexus/pkg/server/rpc/registry.go`

- [ ] **Step 1: Add the generic function**

Append to `packages/nexus/pkg/server/rpc/registry.go`:

```go
func TypedRegister[Req any, Res any](r *Registry, method string, h func(ctx context.Context, req Req) (Res, *rpckit.RPCError)) {
    r.Register(method, func(ctx context.Context, msgID string, raw json.RawMessage, conn any) (interface{}, *rpckit.RPCError) {
        var req Req
        if err := json.Unmarshal(raw, &req); err != nil {
            return nil, &rpckit.RPCError{Code: rpckit.ErrInvalidParams.Code, Message: "invalid params: " + err.Error()}
        }
        return h(ctx, req)
    })
}
```

- [ ] **Step 2: Verify build**

```bash
cd packages/nexus && go build ./pkg/server/rpc/...
```

Expected: exit 0. (No callers changed yet — purely additive.)

- [ ] **Step 3: Commit**

```bash
git add packages/nexus/pkg/server/rpc/registry.go
git commit -m "feat(rpc): add TypedRegister generic helper to RPC registry"
```

---

## Task 2: Update workspace_manager.go handlers to accept typed params

All 9 handlers in this file currently unmarshal `json.RawMessage` internally. Change each to accept the already-defined `*Params` struct directly.

**Files:**
- Modify: `packages/nexus/pkg/handlers/workspace_manager.go`

- [ ] **Step 1: Remove the json import if no longer needed after this task**

After all changes, if `"encoding/json"` is no longer used in this file, remove it from the import block.

- [ ] **Step 2: Change HandleWorkspaceCreate**

```go
// Before:
func HandleWorkspaceCreate(ctx context.Context, params json.RawMessage, mgr *workspacemgr.Manager, factory *runtime.Factory) (*WorkspaceCreateResult, *rpckit.RPCError) {
    var p WorkspaceCreateParams
    if err := json.Unmarshal(params, &p); err != nil {
        return nil, rpckit.ErrInvalidParams
    }
    // uses p.Spec

// After:
func HandleWorkspaceCreate(ctx context.Context, req WorkspaceCreateParams, mgr *workspacemgr.Manager, factory *runtime.Factory) (*WorkspaceCreateResult, *rpckit.RPCError) {
    // uses req.Spec directly — delete the Unmarshal block
```

- [ ] **Step 3: Change HandleWorkspaceList**

```go
// Before: func HandleWorkspaceList(_ context.Context, _ json.RawMessage, mgr) ...
// After:  func HandleWorkspaceList(_ context.Context, _ WorkspaceListParams, mgr) ...
```

(No body change needed — the function ignores params.)

- [ ] **Step 4: Change HandleWorkspaceRemove**

```go
// Before:
func HandleWorkspaceRemove(ctx context.Context, params json.RawMessage, mgr *workspacemgr.Manager, factory *runtime.Factory) (*WorkspaceRemoveResult, *rpckit.RPCError) {
    var p WorkspaceRemoveParams
    if err := json.Unmarshal(params, &p); err != nil { return nil, rpckit.ErrInvalidParams }
    // uses p.ID

// After:
func HandleWorkspaceRemove(ctx context.Context, req WorkspaceRemoveParams, mgr *workspacemgr.Manager, factory *runtime.Factory) (*WorkspaceRemoveResult, *rpckit.RPCError) {
    // uses req.ID — delete Unmarshal block
```

- [ ] **Step 5: Change HandleWorkspaceStop, HandleWorkspaceStart, HandleWorkspaceRestore, HandleWorkspacePause, HandleWorkspaceResume, HandleWorkspaceFork**

Apply the same pattern to each:
- Rename `params json.RawMessage` → `req WorkspaceXxxParams`
- Delete the `var p WorkspaceXxxParams; json.Unmarshal(params, &p)` block
- Replace all `p.` references with `req.`

- [ ] **Step 6: Verify build**

```bash
cd packages/nexus && go build ./pkg/handlers/...
```

Expected: build errors citing callers in `rpc_handlers.go` — those are fixed in Task 6. At this stage, only check that `workspace_manager.go` itself compiles (run `go vet ./pkg/handlers/...` if needed).

---

## Task 3: Update remaining handler files

Apply the same typed-params pattern to every other handler file.

**Files:** auth_relay.go, exec.go, fs.go, git.go, node.go, os_picker.go, service.go, spotlight.go, workspace_info.go, workspace_local.go, workspace_ready.go, workspace_relations.go

- [ ] **Step 1: auth_relay.go**

`HandleAuthRelayRevoke` currently takes `params json.RawMessage`:
```go
// After:
func HandleAuthRelayRevoke(_ context.Context, req AuthRelayRevokeParams, broker *authrelay.Broker) (*AuthRelayRevokeResult, *rpckit.RPCError) {
    // delete Unmarshal; use req.Token directly
```

`HandleAuthRelayMint` (in same file) — apply same change using `AuthRelayMintParams`.

- [ ] **Step 2: exec.go**

```go
// After:
func HandleExec(ctx context.Context, req ExecParams, ws *workspace.Workspace) (*ExecResult, *rpckit.RPCError) {
func HandleExecWithAuthRelay(ctx context.Context, req ExecParams, ws *workspace.Workspace, broker *authrelay.Broker) (*ExecResult, *rpckit.RPCError) {
```

- [ ] **Step 3: fs.go — all 7 handlers**

Change each handler to accept its typed struct (`ReadFileParams`, `WriteFileParams`, `ExistsParams`, `ReaddirParams`, `MkdirParams`, `RmParams`, `StatParams`) and delete the `json.Unmarshal` block in each.

- [ ] **Step 4: git.go**

```go
// After:
func HandleGitCommand(ctx context.Context, req GitCommandParams, ws *workspace.Workspace) (map[string]interface{}, *rpckit.RPCError) {
```

- [ ] **Step 5: node.go**

`HandleNodeInfo` ignores its params; change the signature to drop it entirely:

```go
// After:
func HandleNodeInfo(_ context.Context, nodeCfg *config.NodeConfig, factory *runtime.Factory) (*NodeInfoResult, *rpckit.RPCError) {
```

When registered with `TypedRegister`, use `struct{}` as the request type:

```go
rpc.TypedRegister(r, "node.info", func(ctx context.Context, _ struct{}) (*handlers.NodeInfoResult, *rpckit.RPCError) {
    return handlers.HandleNodeInfo(ctx, s.nodeCfg, s.runtimeFactory)
})
```

- [ ] **Step 6: os_picker.go**

```go
// After:
func HandlePickDirectory(_ context.Context, req PickDirectoryParams) (*PickDirectoryResult, *rpckit.RPCError) {
```

- [ ] **Step 7: service.go**

```go
// After:
func HandleServiceCommand(ctx context.Context, req ServiceCommandParams, ws *workspace.Workspace, mgr *services.Manager) (map[string]interface{}, *rpckit.RPCError) {
```

- [ ] **Step 8: spotlight.go — all 4 handlers**

```go
func HandleSpotlightExpose(ctx context.Context, req SpotlightExposeParams, mgr *spotlight.Manager) (*SpotlightExposeResult, *rpckit.RPCError)
func HandleSpotlightList(_ context.Context, req SpotlightListParams, mgr *spotlight.Manager) (*SpotlightListResult, *rpckit.RPCError)
func HandleSpotlightClose(_ context.Context, req SpotlightCloseParams, mgr *spotlight.Manager) (*SpotlightCloseResult, *rpckit.RPCError)
func HandleSpotlightApplyComposePorts(ctx context.Context, req SpotlightApplyComposePortsParams, mgr *spotlight.Manager) (*SpotlightApplyComposePortsResult, *rpckit.RPCError)
```

Note: `HandleSpotlightApplyComposePorts` currently takes an explicit `rootPath string` for the workspace root (injected by the registration closure). Keep that — it's not a param but a server-side dependency:

```go
func HandleSpotlightApplyComposePorts(ctx context.Context, req SpotlightApplyComposePortsParams, rootPath string, mgr *spotlight.Manager) (*SpotlightApplyComposePortsResult, *rpckit.RPCError)
```

The registration will capture `rootPath` via closure (same as now):

```go
rpc.TypedRegister(r, "spotlight.applyComposePorts", func(ctx context.Context, req handlers.SpotlightApplyComposePortsParams) (*handlers.SpotlightApplyComposePortsResult, *rpckit.RPCError) {
    ws, rpcErr := resolveWorkspace(ctx, req.WorkspaceID, s)
    if rpcErr != nil { return nil, rpcErr }
    return handlers.HandleSpotlightApplyComposePorts(ctx, req, ws.RootPath, s.spotlightMgr)
})
```

- [ ] **Step 9: workspace_info.go, workspace_local.go, workspace_ready.go, workspace_relations.go**

Apply the same typed-params pattern to `HandleWorkspaceInfo`, `HandleWorkspaceSetLocalWorktree`, `HandleWorkspaceReady`, `HandleWorkspaceRelationsList`.

- [ ] **Step 10: Commit handler changes**

```bash
git add packages/nexus/pkg/handlers/
git commit -m "refactor(handlers): accept typed param structs instead of json.RawMessage"
```

---

## Task 4: Update rpc_handlers.go to use TypedRegister

This task replaces all `r.Register(…)` blocks in `packages/nexus/pkg/server/rpc_handlers.go` with `rpc.TypedRegister`. This is the task that actually wires up the typed dispatch.

**Files:**
- Modify: `packages/nexus/pkg/server/rpc_handlers.go`

- [ ] **Step 1: Add rpc package import**

```go
import (
    // existing imports
    "github.com/inizio/nexus/packages/nexus/pkg/server/rpc"
)
```

- [ ] **Step 2: Replace all workspace registrations**

```go
rpc.TypedRegister(r, "workspace.create", func(ctx context.Context, req handlers.WorkspaceCreateParams) (*handlers.WorkspaceCreateResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceCreate(ctx, req, s.workspaceMgr, s.runtimeFactory)
})
rpc.TypedRegister(r, "workspace.list", func(ctx context.Context, req handlers.WorkspaceListParams) (*handlers.WorkspaceListResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceList(ctx, req, s.workspaceMgr)
})
rpc.TypedRegister(r, "workspace.remove", func(ctx context.Context, req handlers.WorkspaceRemoveParams) (*handlers.WorkspaceRemoveResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceRemove(ctx, req, s.workspaceMgr, s.runtimeFactory)
})
rpc.TypedRegister(r, "workspace.stop", func(ctx context.Context, req handlers.WorkspaceStopParams) (*handlers.WorkspaceStopResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceStop(ctx, req, s.workspaceMgr)
})
rpc.TypedRegister(r, "workspace.start", func(ctx context.Context, req handlers.WorkspaceStartParams) (*handlers.WorkspaceStartResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceStart(ctx, req, s.workspaceMgr)
})
rpc.TypedRegister(r, "workspace.restore", func(ctx context.Context, req handlers.WorkspaceRestoreParams) (*handlers.WorkspaceRestoreResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceRestore(ctx, req, s.workspaceMgr, s.runtimeFactory)
})
rpc.TypedRegister(r, "workspace.pause", func(ctx context.Context, req handlers.WorkspacePauseParams) (*handlers.WorkspacePauseResult, *rpckit.RPCError) {
    return handlers.HandleWorkspacePause(ctx, req, s.workspaceMgr, s.runtimeFactory)
})
rpc.TypedRegister(r, "workspace.resume", func(ctx context.Context, req handlers.WorkspaceResumeParams) (*handlers.WorkspaceResumeResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceResume(ctx, req, s.workspaceMgr, s.runtimeFactory)
})
rpc.TypedRegister(r, "workspace.fork", func(ctx context.Context, req handlers.WorkspaceForkParams) (*handlers.WorkspaceForkResult, *rpckit.RPCError) {
    return handlers.HandleWorkspaceFork(ctx, req, s.workspaceMgr, s.runtimeFactory)
})
```

- [ ] **Step 3: Replace auth relay, exec, node.info, os.pickDirectory**

```go
rpc.TypedRegister(r, "authrelay.mint", func(ctx context.Context, req handlers.AuthRelayMintParams) (*handlers.AuthRelayMintResult, *rpckit.RPCError) {
    return handlers.HandleAuthRelayMint(ctx, req, s.workspaceMgr, s.authRelayBroker)
})
rpc.TypedRegister(r, "authrelay.revoke", func(ctx context.Context, req handlers.AuthRelayRevokeParams) (*handlers.AuthRelayRevokeResult, *rpckit.RPCError) {
    return handlers.HandleAuthRelayRevoke(ctx, req, s.authRelayBroker)
})
rpc.TypedRegister(r, "node.info", func(ctx context.Context, _ struct{}) (*handlers.NodeInfoResult, *rpckit.RPCError) {
    return handlers.HandleNodeInfo(ctx, s.nodeCfg, s.runtimeFactory)
})
rpc.TypedRegister(r, "os.pickDirectory", func(ctx context.Context, req handlers.PickDirectoryParams) (*handlers.PickDirectoryResult, *rpckit.RPCError) {
    return handlers.HandlePickDirectory(ctx, req)
})
```

- [ ] **Step 4: Replace fs and spotlight registrations**

Apply the same pattern to the 7 `fs.*` registrations and 3 spotlight registrations, using the same handler types from the `handlers` package.

For `fs.readFile`:
```go
rpc.TypedRegister(r, "fs.readFile", func(ctx context.Context, req handlers.ReadFileParams) (*handlers.ReadFileResult, *rpckit.RPCError) {
    ws, rpcErr := resolveWorkspace(ctx, req.WorkspaceID, s)
    if rpcErr != nil { return nil, rpcErr }
    return handlers.HandleReadFile(ctx, req, ws)
})
```

Repeat for `fs.writeFile`, `fs.exists`, `fs.readdir`, `fs.mkdir`, `fs.rm`, `fs.stat`.

- [ ] **Step 5: Replace remaining registrations (workspace.info, workspace.relations.list, etc.)**

Apply the same pattern to `workspace.info`, `workspace.relations.list`, `workspace.setLocalWorktree`, `workspace.ready`, `git.command`, `service.command`.

- [ ] **Step 6: Verify build and tests**

```bash
cd packages/nexus && go build ./... && go test ./...
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add packages/nexus/pkg/server/rpc_handlers.go
git commit -m "refactor(rpc): migrate all registrations to TypedRegister"
```

---

## Task 5: Add TypeScript RPCSchema

**Files:**
- Create: `packages/sdk/js/src/rpc/schema.ts`
- Modify: `packages/sdk/js/src/client.ts`

- [ ] **Step 1: Create schema.ts**

Create `packages/sdk/js/src/rpc/schema.ts`:

```ts
import type {
  WorkspaceRecord,
  WorkspaceCreateResult,
  WorkspaceListResult,
  WorkspaceStartResult,
  WorkspaceStopResult,
  WorkspaceRemoveResult,
  WorkspacePauseResult,
  WorkspaceResumeResult,
  WorkspaceRestoreResult,
  WorkspaceForkResult,
  WorkspaceRelationsListResult,
  NodeInfo,
  Capability,
} from '../types';
import type { ExecResultData } from '../types/exec';
import type {
  SpotlightListResult,
  SpotlightExposeResult,
  SpotlightApplyComposePortsResult,
} from '../spotlight';

export interface RPCSchema {
  'workspace.create':        [{ spec: { repo: string; workspaceName: string; agentProfile: string; backend?: string; authBinding?: Record<string, string>; configBundle?: string } }, WorkspaceCreateResult];
  'workspace.list':          [{}, WorkspaceListResult];
  'workspace.info':          [{ workspaceId: string }, WorkspaceRecord];
  'workspace.start':         [{ id: string }, WorkspaceStartResult];
  'workspace.stop':          [{ id: string }, WorkspaceStopResult];
  'workspace.remove':        [{ id: string }, WorkspaceRemoveResult];
  'workspace.pause':         [{ id: string }, WorkspacePauseResult];
  'workspace.resume':        [{ id: string }, WorkspaceResumeResult];
  'workspace.restore':       [{ id: string }, WorkspaceRestoreResult];
  'workspace.fork':          [{ id: string; name: string; ref?: string }, WorkspaceForkResult];
  'workspace.relations.list':[{ repoId?: string }, WorkspaceRelationsListResult];
  'workspace.setLocalWorktree': [{ id: string; localPath: string }, Record<string, never>];
  'workspace.ready':         [{ workspaceId: string }, { ready: boolean }];
  'authrelay.mint':          [{ workspaceId: string; binding: string; ttlSeconds: number }, { token: string }];
  'authrelay.revoke':        [{ token: string }, { revoked: boolean }];
  'exec.exec':               [{ workspaceId: string; command: string; args: string[]; options?: { cwd?: string; env?: Record<string, string>; timeout?: number; authRelayToken?: string } }, ExecResultData];
  'fs.readFile':             [{ workspaceId: string; path: string; encoding?: string }, { content: string }];
  'fs.writeFile':            [{ workspaceId: string; path: string; content: string; encoding?: string }, { written: boolean }];
  'fs.exists':               [{ workspaceId: string; path: string }, { exists: boolean }];
  'fs.readdir':              [{ workspaceId: string; path: string }, { entries: string[] }];
  'fs.mkdir':                [{ workspaceId: string; path: string; recursive?: boolean }, { written: boolean }];
  'fs.rm':                   [{ workspaceId: string; path: string; recursive?: boolean }, { written: boolean }];
  'fs.stat':                 [{ workspaceId: string; path: string }, { name: string; size: number; isDir: boolean; modTime: string }];
  'node.info':               [{}, NodeInfo];
  'capabilities.list':       [{}, { capabilities: Capability[] }];
  'spotlight.expose':        [{ workspaceId: string; remotePort: number; localPort?: number }, SpotlightExposeResult];
  'spotlight.list':          [{ workspaceId: string }, SpotlightListResult];
  'spotlight.close':         [{ id: string; workspaceId: string }, { closed: boolean }];
  'spotlight.applyComposePorts': [{ workspaceId: string }, SpotlightApplyComposePortsResult];
  'git.command':             [{ workspaceId: string; args: string[] }, unknown];
  'service.command':         [{ workspaceId: string; command: string; args?: string[] }, unknown];
  'os.pickDirectory':        [{ title?: string }, { path: string }];
}
```

- [ ] **Step 2: Add overloaded request() signature to client.ts**

In `packages/sdk/js/src/client.ts`, change the `request` method from:

```ts
async request<T = unknown>(method: string, params?: Record<string, unknown>): Promise<T>
```

To an overloaded signature that is typed when the method is in `RPCSchema` and falls back to generic for unknown methods:

```ts
import type { RPCSchema } from './rpc/schema';

async request<M extends keyof RPCSchema>(method: M, params: RPCSchema[M][0]): Promise<RPCSchema[M][1]>;
async request<T = unknown>(method: string, params?: Record<string, unknown>): Promise<T>;
async request(method: string, params?: Record<string, unknown>): Promise<unknown> {
  return this.core.request(
    method,
    params,
    (data) => this.transport!.send(data),
    () => this.transport !== null && this.transport.isOpen(),
  );
}
```

- [ ] **Step 3: Verify TypeScript**

```bash
cd packages/sdk/js && pnpm exec tsc --noEmit
```

Expected: exit 0.

- [ ] **Step 4: Run unit tests**

```bash
cd packages/sdk/js && pnpm exec jest --runInBand
```

Expected: all 42 tests pass.

- [ ] **Step 5: Verify E2E types**

```bash
cd packages/e2e/flows && pnpm exec tsc --noEmit
```

Expected: exit 0. If any `client.request(…)` call in the E2E tests now fails type-checking because its params don't match the schema, fix the call to use the correct params shape.

- [ ] **Step 6: Commit**

```bash
git add packages/sdk/js/src/rpc/schema.ts packages/sdk/js/src/client.ts
git commit -m "feat(sdk): add RPCSchema for typed client.request() calls"
```

---

## Task 6: Final verification

- [ ] **Step 1: Full Go build and tests**

```bash
cd packages/nexus && go build ./... && go test ./...
```

Expected: exit 0, all pass.

- [ ] **Step 2: Full TypeScript build and tests**

```bash
cd packages/sdk/js && pnpm build && pnpm exec jest --runInBand
cd packages/e2e/flows && pnpm exec tsc --noEmit
cd packages/nexus-ui && pnpm check
```

Expected: all exit 0.

- [ ] **Step 3: Check that a method-name typo is now a compile error**

Add this to any test file temporarily:

```ts
// @ts-expect-error — should fail: 'workspace.creat' not in schema
await client.request('workspace.creat', { spec: {} });
```

Run `pnpm exec tsc --noEmit` and confirm the `@ts-expect-error` line is satisfied (i.e., it IS an error, meaning the schema is working).

Remove the test line after confirming.
