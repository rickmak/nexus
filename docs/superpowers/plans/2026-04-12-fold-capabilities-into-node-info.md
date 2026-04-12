# Fold capabilities.list into node.info Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the redundant `capabilities.list` RPC endpoint; callers use `node.info` which already returns the same capabilities data.

**Architecture:** `node.info` already includes a `capabilities` field (same `[]runtime.Capability` source). Delete `capabilities.go` handler, remove the RPC registration, update the three E2E test files and the SDK type exports that referenced the standalone endpoint.

**Tech Stack:** Go, TypeScript

---

## Background

`node.info` (handler: `HandleNodeInfo`) already returns:

```json
{
  "node": { … },
  "compatibility": { … },
  "capabilities": [ { "name": "runtime.firecracker", "available": true }, … ]
}
```

`capabilities.list` (handler: `HandleCapabilitiesList`) returns:

```json
{ "capabilities": [ … ] }
```

They call the same `factory.Capabilities()` method. The only consumer of `capabilities.list` is the E2E harness, which uses it to skip tests when a runtime is unavailable. `node.info` satisfies the same need.

### File changes

- **Delete:** `packages/nexus/pkg/handlers/capabilities.go`
- **Modify:** `packages/nexus/pkg/server/rpc_handlers.go` — remove `capabilities.list` registration
- **Modify:** `packages/e2e/flows/src/cases/harness-smoke.e2e.test.ts` — use `node.info`
- **Modify:** `packages/e2e/flows/src/cases/spotlight/spotlight-compose.e2e.test.ts` — use `node.info`
- **Modify:** `packages/e2e/flows/src/cases/runtime/runtime-selection.e2e.test.ts` — use `node.info`
- **Modify:** `packages/sdk/js/src/types/workspace.ts` — add `NodeInfo` export; `Capability` stays
- **Modify:** `packages/sdk/js/src/index.ts` — export `NodeInfo`

---

## Task 1: Remove the Go handler and RPC registration

**Files:**

- Delete: `packages/nexus/pkg/handlers/capabilities.go`
- Modify: `packages/nexus/pkg/server/rpc_handlers.go`
- **Step 1: Delete the handler file**

```bash
rm packages/nexus/pkg/handlers/capabilities.go
```

- **Step 2: Remove the registration in rpc_handlers.go**

In `packages/nexus/pkg/server/rpc_handlers.go`, delete these lines (around line 92):

```go
r.Register("capabilities.list", func(_ context.Context, _ string, params json.RawMessage, _ any) (interface{}, *rpckit.RPCError) {
    return handlers.HandleCapabilitiesList(ctx, params, s.runtimeFactory)
})
```

- **Step 3: Verify build**

```bash
cd packages/nexus && go build ./...
```

Expected: no errors. If `CapabilitiesListResult` is referenced anywhere else, the compiler will tell you — remove those references too.

- **Step 4: Run Go tests**

```bash
cd packages/nexus && go test ./...
```

Expected: all pass.

- **Step 5: Commit**

```bash
git add packages/nexus/pkg/handlers/capabilities.go packages/nexus/pkg/server/rpc_handlers.go
git commit -m "feat(rpc): remove capabilities.list — data available in node.info"
```

---

## Task 2: Add NodeInfo type to the SDK

The E2E tests need a typed result for `node.info`. Add it alongside the existing `Capability` type.

**Files:**

- Modify: `packages/sdk/js/src/types/workspace.ts`
- Modify: `packages/sdk/js/src/index.ts`
- **Step 1: Add NodeInfo to workspace.ts**

In `packages/sdk/js/src/types/workspace.ts`, add after the `Capability` interface:

```ts
export interface NodeInfo {
  node: {
    id: string;
    name?: string;
  };
  compatibility: {
    arch?: string;
    os?: string;
  };
  capabilities: Capability[];
}
```

- **Step 2: Verify SDK compiles**

```bash
cd packages/sdk/js && pnpm exec tsc --noEmit
```

Expected: exit 0.

- **Step 3: Commit**

```bash
git add packages/sdk/js/src/types/workspace.ts
git commit -m "feat(sdk): add NodeInfo type for node.info RPC result"
```

---

## Task 3: Update E2E tests to use node.info

**Files:**

- Modify: `packages/e2e/flows/src/cases/harness-smoke.e2e.test.ts`
- Modify: `packages/e2e/flows/src/cases/spotlight/spotlight-compose.e2e.test.ts`
- Modify: `packages/e2e/flows/src/cases/runtime/runtime-selection.e2e.test.ts`

Each file currently calls `client.request<{ capabilities: Capability[] }>('capabilities.list', {})`. Change each to:

```ts
client.request<NodeInfo>('node.info', {}).then(r => r.capabilities)
```

- **Step 1: Update harness-smoke.e2e.test.ts**

Change the import line:

```ts
import type { Capability, NodeInfo } from '@nexus/sdk';
```

Change the two call sites (lines ~17 and ~37):

```ts
// Before:
const caps = await client.request<{ capabilities: Capability[] }>('capabilities.list', {}).then(r => r.capabilities);
// After:
const caps = await client.request<NodeInfo>('node.info', {}).then(r => r.capabilities);
```

- **Step 2: Update spotlight-compose.e2e.test.ts**

Add `NodeInfo` to the import:

```ts
import type { Capability, NodeInfo } from '@nexus/sdk';
```

Change the call site:

```ts
// Before:
const caps = await session.client.request<{ capabilities: Capability[] }>('capabilities.list', {}).then(r => r.capabilities);
// After:
const caps = await session.client.request<NodeInfo>('node.info', {}).then(r => r.capabilities);
```

- **Step 3: Update runtime-selection.e2e.test.ts**

Add `NodeInfo` to the import:

```ts
import { WorkspaceHandle, Capability, NodeInfo } from '@nexus/sdk';
```

Change both call sites (lines ~25 and ~69):

```ts
// Before:
const { capabilities: caps } = await session.client.request<{ capabilities: Capability[] }>('capabilities.list', {});
// After:
const { capabilities: caps } = await session.client.request<NodeInfo>('node.info', {});
```

- **Step 4: Verify E2E TypeScript**

```bash
cd packages/sdk/js && pnpm build
cd packages/e2e/flows && pnpm exec tsc --noEmit
```

Expected: both exit 0.

- **Step 5: Commit**

```bash
git add packages/e2e/flows/src/cases/
git commit -m "feat(e2e): use node.info instead of removed capabilities.list"
```

---

## Task 4: Verify and push

- **Step 1: Full test run**

```bash
cd packages/nexus && go test ./...
cd packages/sdk/js && pnpm exec jest --runInBand
```

Expected: all pass.

- **Step 2: Check NodeInfo config field names match Go structs**

Look at `packages/nexus/pkg/config/node.go` to confirm the JSON field names for `NodeIdentity` and `NodeCompatibility`, and update the `NodeInfo` TypeScript type in Task 2 if they differ.

```bash
grep -n "json:" packages/nexus/pkg/config/node.go | head -20
```

Adjust `NodeInfo.node` and `NodeInfo.compatibility` field names to match exactly.