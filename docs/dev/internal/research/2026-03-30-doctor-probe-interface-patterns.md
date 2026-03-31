# Nexus Doctor Probe Interface Research

Date: 2026-03-30

## Goal

Define a stable, project-agnostic health-check interface for `nexus doctor` that works for:

- web stacks (HTTP/runtime readiness)
- infra stacks (ports/services)
- mobile stacks (Maestro/UI flows)
- custom project checks (scripts/tools)

## External patterns reviewed

### Kubernetes probes

Reference: <https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes>

Key patterns:

- Separate intent by phase:
  - startup (boot complete)
  - readiness (can receive traffic)
  - liveness (still healthy)
- Probe failures have policy-driven behavior (restart/remove from traffic).
- Timeout/retry thresholds are first-class.

Implication for Nexus:

- Keep probe intent explicit (bootstrap vs runtime), not a single boolean.
- Include retry/timeout policy in probe contract, not hard-coded in the binary.

### Testcontainers startup/wait strategies

Reference: <https://java.testcontainers.org/features/startup_and_waits/>

Key patterns:

- Distinguish startup checks from readiness checks.
- Provide multiple built-in wait strategies (TCP, HTTP, healthcheck, log pattern).
- Permit custom strategy implementation when built-ins are insufficient.

Implication for Nexus:

- Support a small set of built-in probe kinds for common use.
- Always keep an escape hatch (`command` probe) for project-specific checks.

### Playwright webServer readiness

Reference: <https://playwright.dev/docs/test-webserver>

Key patterns:

- Per-server config includes command, URL, timeout, and shutdown behavior.
- Readiness status allows broader acceptable response range (`2xx-4xx` in many cases).
- Multiple servers can be declared and checked independently.

Implication for Nexus:

- Per-probe settings should include timeout and expected success criteria.
- Multiple probes should run with explicit ordering/dependencies.

### Maestro flow tags/workspace config

References:

- <https://maestro.mobile.dev/cli/tags>
- <https://maestro.mobile.dev/api-reference/configuration/workspace-configuration>

Key patterns:

- Tags segment checks by CI context (pull request vs nightly/release).
- Workspace-level config controls default selection and execution policy.
- CI can switch behavior with config file selection.

Implication for Nexus:

- Probe selection should be context-aware (`on.pull_request`, `on.push`, `on.schedule`).
- Tag/profile filtering is better than hard-coded suite-specific flags.

## Proposed stable interface shape

### 1) Keep `command` probes as the universal base

This is the most portable primitive and already covers Android/Maestro:

- `command: "maestro"`
- `args: ["test", "--include-tags=pull-request", "flows/"]`

### 2) Add explicit probe metadata

Recommended fields:

- `name`
- `required` (true/false)
- `timeoutMs`
- `retries`

### 3) Treat built-ins as optional convenience adapters

Built-ins like `runtimeProbe`/`authProbe` can remain as compatibility helpers, but they should map to generic probe execution internally.

### 4) Separate contract checks from runtime checks

`nexus doctor` should conceptually execute in phases:

1. static contract (`workspace.json`, lifecycle scripts, compose shape)
2. runtime probes (project-defined commands)
3. optional deep/e2e probes (tag/profile selected)

## Minimal near-term recommendation

For immediate stability and cross-project support:

1. Standardize on `doctor.probes[]` command probes.
2. Keep downstream workflow YAML minimal (single `nexus doctor` invocation).
3. Let each project own probe scripts/commands (web curl checks, Maestro flows, API smoke tests).
4. Run all configured probes by default; keep CI orchestration minimal.

## Open design decisions

- Should event filters be simple booleans or expression-based?
- Should probes run sequentially by default with optional parallel groups?
- Should probe outputs be collected as structured artifacts (JSON report) for CI annotations?
- Should `runtimeProbe`/`authProbe` be deprecated once command probes are fully mature?
