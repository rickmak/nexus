# Daemon Isolated Runtime and Health Contract Design

Date: 2026-03-31
Status: Proposed (approved in conversation, pending implementation)

## 1. Problem Statement

Current workspace runtime behavior allows host-level Docker port binding for downstream `docker compose` projects. This causes port collisions and violates the intended local-remote model where workspace services are reachable only through Nexus-managed access paths.

Current health checking also conflates environment bring-up and behavior validation, which makes results harder to interpret and less stable for mixed-stack monorepos.

## 2. Goals

1. Enforce mandatory isolated runtime for workspaces in v1.
2. Support both `dind` and `lxc` backends in v1 to validate abstraction quality.
3. Ensure clients connect only to daemon RPC; service access flows through daemon-managed Spotlight tunnels.
4. Split health checks into two clear phases:
   - `doctor.probes[]`: startup/readiness/liveness (environment health)
   - `doctor.tests[]`: behavior correctness (functional flows)
5. Support mixed-stack monorepos (example: docker compose backend + React Native app).
6. Add acceptance testing that proves in-workspace toolchain usability (`opencode`, git push auth flow).

## 3. Non-Goals

- Compatibility with `scapes/main` API/config shapes. It is conceptual guidance only.
- Host-runtime fallback mode for normal workspace execution.
- Per-CI-event filtering logic in Nexus doctor config.

## 4. Core Decisions

1. **Isolation is mandatory**: no host mode for workspace runtime.
2. **v1 backend support includes both**: `dind` and `lxc`.
3. **Access model**: SDK/client connects only to daemon RPC endpoint; workspace service ingress only via daemon-managed Spotlight.
4. **Config source of truth**: project-owned `.nexus/workspace.json`.
5. **Health pipeline**: run probes first; run tests only if all required probes pass.
6. **Doctor execution model**: always run all configured probes/tests (no event/tag filtering in Nexus contract).

## 5. Runtime Architecture

### 5.1 Driver Abstraction

Introduce a backend-neutral runtime driver contract used by daemon core:

- `Create(workspaceSpec)`
- `Start(workspaceId)`
- `Stop(workspaceId)`
- `Restore(workspaceId)`
- `Exec(workspaceId, command)`
- `ExposePort(workspaceId, service, remotePort)`
- `Destroy(workspaceId)`
- `SnapshotMetadata(workspaceId)`

Implementations:

- `dindDriver` (v1)
- `lxcDriver` (v1)

Daemon lifecycle/state logic remains shared and backend-agnostic.

### 5.2 Workspace Lifecycle and State

Workspace state machine:

- `created -> running -> stopped -> restored -> removed`

Semantics:

- `stop`: tear down runtime compute while preserving workspace state.
- `restore`: rehydrate preserved workspace state and resume service context.

Persisted state requirements:

- filesystem/worktree changes
- git state (HEAD/index/branches)
- daemon workspace metadata
- service/lifecycle metadata
- auth profile bindings

## 6. Capabilities and Admission Control

Daemon publishes capabilities via RPC, derived at startup and refreshable:

- `runtime.dind`
- `runtime.lxc`
- `toolchain.android-sdk`
- `toolchain.xcode`
- `spotlight.tunnel`
- `auth.profile.git`
- `auth.profile.codex`
- `auth.profile.opencode`

Workspace creation/restore performs admission checks against project requirements and fails fast with structured diagnostics if unmet.

## 7. Config Contract Changes

### 7.1 Workspace Runtime Requirements

Add explicit runtime/capability requirement fields to `.nexus/workspace.json`:

```json
{
  "runtime": {
    "required": ["dind", "lxc"],
    "selection": "prefer-first"
  },
  "capabilities": {
    "required": [
      "spotlight.tunnel",
      "auth.profile.git"
    ]
  }
}
```

Semantics:

- `runtime.required` lists acceptable backends; daemon must select one available backend from this set.
- `selection=prefer-first` means daemon chooses the first available backend in list order.
- `capabilities.required` must be fully satisfied for create/restore to proceed.

### 7.2 Doctor Contract

`doctor` contains two explicit lists:

- `doctor.probes[]`: environment health commands
- `doctor.tests[]`: behavior validation commands

Each entry uses command-based execution with fields:

- `name`
- `command`
- `args`
- `timeoutMs`
- `retries`
- `required`

Execution order:

1. run all probes
2. fail if any required probe fails
3. run all tests
4. fail if any required test fails

Report output includes phase, status, attempts, duration, and trimmed error output.

## 8. Monorepo Mixed-Stack Behavior

For monorepos combining multiple stacks (e.g., compose services plus React Native):

- use multiple probe/test commands within one workspace config
- probe commands ensure each environment segment is up
- test commands validate behavior per segment

Example decomposition:

- `probes`: compose readiness, mobile emulator readiness
- `tests`: auth login flow, Maestro UI navigation flow

## 9. Acceptance and Verification Strategy

### 9.1 Case Study Gate

For every repository in `.case-studies/*`, required probes/tests must pass on both backends:

- matrix axis `backend = dind, lxc`
- matrix axis `project = each case study`

### 9.2 Mandatory In-Workspace Tooling Checks

Acceptance tests must execute inside isolated workspace runtime and include:

- `opencode` invocation + authenticated operation
- git auth flow + `git push` to test remote (safe ephemeral target)

This proves real-world workspace usability beyond process-up checks.

## 10. Security and Networking Model

- No direct client access to backend runtime networks.
- No required host-published ports for normal workspace operation.
- Spotlight is the only ingress mechanism for workspace services.
- Auth material is managed via daemon-defined profiles with explicit persistence policy.

## 11. Rollout Plan

1. Implement shared runtime driver interface and lifecycle state persistence.
2. Implement `dindDriver`.
3. Implement `lxcDriver` on the same interface.
4. Add `doctor.tests[]` and two-phase execution (`probes` then `tests`).
5. Add capability discovery + admission checks.
6. Wire case-study backend matrix gates.
7. Dogfood acceptance with in-workspace `opencode` and git push checks.

## 12. Risks and Mitigations

- Backend behavior drift (`dind` vs `lxc`) -> mitigate with one shared driver contract + identical acceptance suite.
- Auth restore complexity -> mitigate with explicit profile lifecycle and restore-time validation.
- Remote daemon variability -> mitigate with capability handshake and strict admission control.

## 13. Definition of Done

Done when all are true:

1. Workspace runtime is isolated (no host-mode execution path).
2. `dind` and `lxc` backends both pass case-study required checks.
3. `doctor.probes[]` and `doctor.tests[]` pipeline is enforced and reported.
4. Stop/restore preserves file/git/auth state as specified.
5. In-workspace `opencode` and git push acceptance checks pass.
