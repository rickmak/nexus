# Doctor Auto-Discovery and Hanlun Scope Design

Date: 2026-04-01
Status: Proposed (approved in conversation, pending implementation)

## 1. Problem Statement

The current doctor contract requires explicit script paths in `workspace.json` and carries CI environment wiring that is too heavy for PR runners in `oursky`.

Current friction points:

- Explicit per-script configuration in `doctor.probes[]` and `doctor.tests[]` is repetitive.
- Hanlun PR CI cannot rely on organization secrets.
- Some checks are not clearly Hanlun-project-specific and should be reviewed.
- Auth and opencode checks need a split between secretless CI validation and full local/integration validation.

## 2. Goals

1. Add Nexus core doctor auto-discovery for `.nexus/probe` and `.nexus/check`.
2. Keep `workspace.json` explicit probes/tests as internal compatibility mode.
3. Migrate Hanlun to discovery-first contract (remove explicit per-script list in Hanlun).
4. Keep Hanlun workflow env minimal (backend matrix only).
5. Support secretless PR validation for opencode install and free-model session attempt.
6. Keep full auth/password-grant checks available when credentials are present.

## 3. Non-Goals

- Deprecating explicit `workspace.json` probes/tests globally.
- Removing internal compatibility mode in this phase.
- Requiring secrets in PR CI runners.

## 4. Design Decisions

### 4.1 Discovery Contract in Nexus Core

Nexus doctor adds discovery of executable shell scripts from:

- `.nexus/probe/*.sh`
- `.nexus/check/*.sh`

Behavior:

- Discovery includes all matching `.sh` scripts.
- Missing folders or empty folders are warnings only.
- Any discovered script that runs and exits non-zero fails doctor.

### 4.2 Ordering Rules

Execution ordering is deterministic:

1. Numeric-prefixed scripts first (for example `01-foo.sh`, `02-bar.sh`) sorted by numeric prefix then lexical name.
2. Non-prefixed scripts next, sorted lexically.

Doctor emits warnings for non-prefixed files to encourage ordering clarity without blocking execution.

### 4.3 Compatibility Mode

If discovery does not produce runnable entries, doctor continues to honor `workspace.json` `doctor.probes[]` and `doctor.tests[]` as an internal compatibility path.

Compatibility mode remains supported and is not marked deprecated in this phase.

### 4.4 Hanlun Policy and Scope

Hanlun migrates to folder discovery only:

- Use `.nexus/probe/` and `.nexus/check/` with numeric naming.
- Remove explicit script-path lists from Hanlun `workspace.json` doctor section.
- Remove non-project-specific checks from Hanlun set (review each check for direct project relevance).

### 4.5 Auth Check Policy

`check-auth-flow` behavior:

- Always validate OIDC discovery and token endpoint reachability.
- If auth credentials are present, run password-grant token exchange.
- If credentials are absent, warn and skip password-grant step while keeping probe-level network/discovery validations.

This preserves PR secretless behavior while retaining full auth validation in local/integration contexts.

### 4.6 Opencode Check Policy

`check-tooling-runtime` behavior:

- Always validate `opencode --version`.
- If provider key is present, run normal `opencode run` session.
- If provider key is absent, attempt free-model execution:
  - query model list (`opencode models`)
  - select first match containing `free` (case-insensitive)
  - run `opencode run --model <selected-free-model> ...`
- If no free model is available, warn and skip session run (installation check still passes).

## 5. Workflow Contract for Hanlun

Hanlun workflow passes only:

- `NEXUS_RUNTIME_BACKEND=${{ matrix.backend }}`

No auth/model/provider secret env vars are required in PR workflow configuration.

## 6. Acceptance Criteria

Done when all are true:

1. Nexus doctor supports discovery of `.nexus/probe/*.sh` and `.nexus/check/*.sh` with ordering/warnings described above.
2. Nexus doctor still supports explicit `workspace.json` probe/test entries as internal fallback.
3. Hanlun uses discovery folders and no longer enumerates per-check script paths in doctor config.
4. Hanlun workflow env for doctor action is backend-only.
5. Hanlun auth check passes without secrets by skipping password-grant while retaining discovery/reachability checks.
6. Hanlun opencode check attempts free-model run when no provider key exists, then warn+skip only if no free model is available.

## 7. Risks and Mitigations

- Ambiguous script ordering in mixed naming
  - Mitigation: deterministic sort + warnings for non-prefixed files.
- Repositories depending on explicit config behavior
  - Mitigation: keep internal compatibility fallback path.
- CI flakiness from external model availability
  - Mitigation: free-model attempt first; warn+skip session when unavailable; keep install verification strict.
