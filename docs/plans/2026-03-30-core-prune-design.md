# Core-Only Hard Prune Design

## Problem

The repository still carries multiple product lines (Enforcer/Boulder + IDE plugins + legacy integration surfaces) that are outside the current core objective: remote workspace execution with a minimal API surface. This increases maintenance cost, CI noise, and architecture drift.

## Goal

Perform a single-shot hard-delete so the repository centers on:

- `packages/workspace-daemon`
- `packages/workspace-sdk`

and only the minimal shared/root wiring needed to build, test, and document those two packages.

## Non-Goals

- Preserving backward compatibility for removed Enforcer/Boulder/plugin packages.
- Keeping historical docs in-place when they document removed components.
- Shipping migration shims for deleted package APIs.

## Keep Set

Primary:

- `packages/workspace-daemon/**`
- `packages/workspace-sdk/**`

Minimal supporting set:

- Root tool wiring required by daemon/sdk (`Taskfile.yml`, `package.json`, `pnpm-workspace.yaml`, ts/go config as needed)
- CI workflows, trimmed to daemon/sdk build/test checks
- Core docs for workspace daemon + sdk + config
- One short migration note listing deleted scopes

## Delete Set

Hard delete non-core implementation and docs, including:

- `packages/enforcer/**`
- `packages/opencode/**`
- `packages/opencode-plugin/**`
- `packages/claude/**`
- `packages/cursor/**`
- `boulder/**` (if present)
- `.opencode/**`
- Enforcer/Boulder/plugin-focused docs/tutorials/reference pages
- non-core e2e/examples tied to removed modules

## Required Wiring Changes (Same PR)

1. Root task/build wiring
   - Remove build/test/lint/ci steps referencing deleted packages.
   - Keep daemon/sdk tasks only.

2. Workspace/package manager wiring
   - Ensure workspace/package manifests do not resolve deleted packages.
   - Keep install/build flows valid for daemon/sdk only.

3. Runtime/readiness/schema cleanup
   - Remove stale references that require deleted plugin surfaces.
   - Keep ACP behavior capability-aware and optional; no hard plugin coupling.

4. Docs cleanup
   - Remove non-core docs.
   - Keep core workspace docs concise and accurate.
   - Add migration/deletion note.

## Risk Assessment

High risk:

- Root CI/task breakage due to stale package references.
- Docs/index drift causing invalid links.

Medium risk:

- Type/schema references still pointing to removed auth/plugin identities.

Low risk:

- Internal helper scripts/examples removed without affecting core runtime.

## Mitigations

- Execute deletes and wiring fixes in the same change set.
- Keep verification gate focused and strict.
- Prefer explicit keep-list in docs and root tasks to avoid accidental reintroduction.

## Verification Strategy

Required verification before completion:

1. `cd packages/workspace-daemon && go test ./...`
2. `cd packages/workspace-sdk && pnpm exec tsc --noEmit`
3. `cd packages/workspace-sdk && pnpm exec jest --runInBand`
4. Root core-only CI/task command runs without references to deleted modules.

## Rollback Strategy

If hard prune causes unacceptable breakage:

- Revert the prune commit/PR as a single unit.
- Re-apply using the same keep/delete matrix with narrower scope only where evidence shows required dependencies were removed incorrectly.

## Acceptance Criteria

- Non-core package directories are removed from repository.
- Root build/task/CI no longer references removed modules.
- Core docs reflect daemon+sdk-only scope.
- Workspace daemon and sdk tests/typechecks pass.
- No dangling references to deleted components in retained docs and root workflows.
