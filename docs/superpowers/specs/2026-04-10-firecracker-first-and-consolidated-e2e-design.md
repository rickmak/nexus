# Firecracker-First Runtime and Consolidated E2E Design

Date: 2026-04-10
Status: Approved for planning
Owner: workspace-daemon + sdk

## Context

Current behavior and testing are misaligned with product intent:

1. Runtime selection on darwin currently resolves to seatbelt-first in normal flows.
2. Firecracker startup readiness depends on Lima/firecracker host prerequisites that are not surfaced as a clear preflight contract.
3. Existing end-to-end coverage is fragmented across ad-hoc shell and JS scripts, making user-journey confidence hard to prove.

User direction for this design:

1. Firecracker-first policy is required.
2. If firecracker cannot run, fail hard, except when mac nested virtualization is detected unsupported.
3. Use preflight-only detection for the mac exception.
4. On workspace create, attempt one automatic setup pass for installable prerequisites.
5. Build a dedicated TS e2e package (`packages/e2e/sdk-runtime`) that imports SDK.
6. Remove scattered ad-hoc scripts and consolidate to structured test cases.

## Decisions

1. Runtime policy becomes firecracker-first.
2. Workspace create uses structured preflight classification and explicit branching.
3. Setup automation is one-attempt, in-band with create flow, reusing existing setup hooks.
4. Add internal test override controls for deterministic branch coverage.
5. Replace ad-hoc e2e scripts with consolidated TS package.

## Goals

1. Guarantee deterministic backend selection behavior with explicit diagnostics.
2. Ensure firecracker/Lima readiness is validated before runtime create.
3. Make create/start/pty/spotlight/auth/tool bootstrap behavior testable from SDK user perspective.
4. Establish CI confidence such that passing e2e implies UI/CLI action-path confidence.
5. Validate local-repo fixture, host/workspace sync, lifecycle hooks, and compose-driven spotlight behavior per run.

## Non-Goals

1. Supporting unlimited fallback chains across runtimes.
2. Introducing broad production-only feature flags for runtime policy behavior.
3. Keeping legacy ad-hoc script suite as primary validation path.

## Runtime Selection and Create Flow

### Policy

1. Firecracker is the primary target backend.
2. Seatbelt is only allowed when preflight returns `unsupported_nested_virt` on mac.
3. Any other firecracker readiness failure is a hard create failure.

### Create decision flow

1. Run firecracker/Lima preflight.
2. If `status=pass`:
   - select firecracker, continue create.
3. If `status=installable_missing`:
   - run one auto-setup attempt (sudo/session path).
   - rerun preflight once.
   - continue only if now `pass`.
   - otherwise fail with structured diagnostics.
4. If `status=unsupported_nested_virt`:
   - allow seatbelt backend selection.
5. If `status=hard_fail`:
   - fail create with diagnostics.

## Preflight Contract

Standardize preflight result payload:

```json
{
  "status": "pass | installable_missing | unsupported_nested_virt | hard_fail",
  "checks": [
    {
      "name": "string",
      "ok": true,
      "message": "string",
      "remediation": "string"
    }
  ],
  "setupAttempted": false,
  "setupOutcome": "success | failed | skipped"
}
```

Rules:

1. `workspace.create` failure responses include this structured object.
2. setup attempt status is included when auto-setup path runs.
3. classification for nested virtualization support is preflight-only.

## Internal Test Overrides

Add internal override mechanism (test/dev mode only) to force branch coverage:

1. force `pass`
2. force `installable_missing`
3. force `unsupported_nested_virt`
4. force `hard_fail`

Requirements:

1. override must be env-gated and disabled by default.
2. override activation must be logged in diagnostics.
3. override usage must be assertable in e2e outputs.

## Consolidated E2E Package

### New package

Create `packages/e2e/sdk-runtime`:

1. TypeScript test package.
2. Imports `@nexus/sdk`.
3. Executes user-journey tests against live daemon.

### Coverage matrix

The suite must cover:

1. workspace lifecycle:
   - create, open, list, start, stop, pause, resume, restore, remove, fork.
2. PTY path:
   - open, write, resize, close on persistent connection.
3. FS/exec/git/service operations.
4. Spotlight:
   - expose, list, close, applyDefaults, applyComposePorts.
5. Tool bootstrap checks in guest:
   - codex, opencode, claude presence and basic invocation.
6. Auth flows:
   - capabilities, auth relay mint/revoke, forwarded credential behavior.
7. Backend branch coverage:
   - firecracker success path,
   - installable-missing with successful auto-setup retry,
   - unsupported nested virt exception path,
   - hard-fail path.
8. Local fixture and sync fidelity:
   - create fresh local git fixture per test run,
   - create workspace from fixture,
   - verify host -> workspace and workspace -> host propagation for tracked file mutations,
   - verify resulting git status/contents on both sides.
9. Lifecycle hooks:
   - verify pre-start/post-start/pre-stop/post-stop hooks execute in expected order,
   - verify hook failures propagate according to policy and produce diagnostics.
10. Compose + spotlight auto-detection:
   - include fixture with docker-compose exposed ports,
   - verify auto-detected ports become spotlight forwards,
   - verify forwards are reachable on host and represented in list output.
11. Spotlight control and visibility UX parity:
   - verify close/toggle behavior via RPC and CLI paths,
   - verify active host-mapped ports overview is consistent across UI data payloads and CLI outputs.

### UI/CLI confidence contract

Maintain explicit mapping artifact:

1. UI action -> RPC method -> e2e test case ID.
2. CLI command -> RPC method -> e2e test case ID.
3. Spotlight visibility actions (including toggle/close and active-port overview) must have explicit UI+CLI mappings.

This mapping is required for confidence claims that passing e2e implies high confidence in UI/CLI action paths.

## Decommission Legacy Ad-hoc Tests

In this effort:

1. Migrate scenarios from `scripts/ci/*.sh` and ad-hoc JS smoke files into `packages/e2e/sdk-runtime`.
2. Update CI to execute the new package as the primary end-to-end gate.
3. Remove migrated legacy scripts.
4. Keep one migration manifest documenting old script -> new test case mapping for review parity.

## Observability and Diagnostics

1. Emit structured logs for:
   - backend chosen,
   - preflight classification,
   - setup attempt and outcome,
   - override activation.
2. Include machine-readable artifacts from e2e runs for branch and action coverage evidence.

## Risks and Mitigations

1. Risk: firecracker capability appears available but host runtime still fails.
   - Mitigation: strengthen preflight checks and classify accurately before create.
2. Risk: migration from ad-hoc scripts loses edge-case coverage.
   - Mitigation: parity checklist per legacy script before deletion.
3. Risk: internal override leaks into non-test behavior.
   - Mitigation: strict env gating and test-only docs.

## Acceptance Criteria

1. Firecracker-first selection is enforced in runtime decision logic.
2. Create flow executes one setup attempt for installable failures and reruns preflight exactly once.
3. Seatbelt fallback only occurs on `unsupported_nested_virt` classification.
4. `packages/e2e/sdk-runtime` exists and covers required action matrix.
5. Legacy ad-hoc e2e scripts are removed after parity migration.
6. CI gates on new e2e package and publishes coverage evidence artifacts.
7. Structured diagnostics are present for preflight/setup/override/backend decisions.
8. E2E run creates isolated local git fixtures and verifies bidirectional host/workspace sync.
9. E2E run verifies lifecycle hooks and docker-compose spotlight auto-detection behavior.
10. E2E run verifies spotlight close/toggle and active host-port overview parity across UI and CLI pathways.
