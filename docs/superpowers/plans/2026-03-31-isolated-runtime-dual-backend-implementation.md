# Isolated Runtime Dual-Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement mandatory isolated workspace runtime with `dind` and `lxc` backends, plus a two-phase `doctor` contract (`probes` then `tests`) and case-study acceptance gates.

**Architecture:** Add a backend-agnostic runtime driver interface in daemon core, implement `dind` and `lxc` drivers, and wire workspace lifecycle (`create/start/stop/restore/remove`) to persisted workspace state. Extend project config with runtime/capability requirements and split health checks into `doctor.probes[]` and `doctor.tests[]`, then gate `.case-studies/*` on both backends with in-workspace `opencode` and `git push` verification.

**Tech Stack:** Go 1.21, TypeScript SDK, Docker (dind), LXC CLI, GitHub Actions, JSON schema validation.

---

## File Structure and Responsibilities

- Create: `packages/workspace-daemon/pkg/runtime/driver.go` - runtime driver interface and shared request/response types.
- Create: `packages/workspace-daemon/pkg/runtime/factory.go` - backend selection and capability admission checks.
- Create: `packages/workspace-daemon/pkg/runtime/factory_test.go` - unit tests for backend/capability selection.
- Create: `packages/workspace-daemon/pkg/runtime/dind/driver.go` - dind driver implementation.
- Create: `packages/workspace-daemon/pkg/runtime/dind/driver_test.go` - dind driver command-level tests.
- Create: `packages/workspace-daemon/pkg/runtime/lxc/driver.go` - lxc driver implementation.
- Create: `packages/workspace-daemon/pkg/runtime/lxc/driver_test.go` - lxc driver command-level tests.
- Modify: `packages/workspace-daemon/pkg/workspacemgr/types.go` - workspace lifecycle states + backend/auth metadata.
- Modify: `packages/workspace-daemon/pkg/workspacemgr/manager.go` - persistent workspace records + stop/restore methods.
- Modify: `packages/workspace-daemon/pkg/workspacemgr/manager_test.go` - persistence and lifecycle tests.
- Create: `packages/workspace-daemon/pkg/handlers/capabilities.go` - `capabilities.list` RPC handler.
- Modify: `packages/workspace-daemon/pkg/handlers/workspace_manager.go` - `workspace.stop` and `workspace.restore` handlers.
- Modify: `packages/workspace-daemon/pkg/server/server.go` - wire new RPC methods and runtime factory.
- Modify: `packages/workspace-daemon/pkg/config/types.go` - add `runtime`, `capabilities`, `doctor.tests` config model.
- Modify: `packages/workspace-daemon/pkg/config/types_test.go` - config validation tests.
- Modify: `schemas/workspace.v1.schema.json` - schema for runtime/capability and `doctor.tests`.
- Modify: `packages/workspace-daemon/cmd/nexus/main.go` - two-phase doctor execution (`probes` then `tests`).
- Modify: `packages/workspace-daemon/cmd/nexus/main_test.go` - doctor phase ordering and failure semantics tests.
- Modify: `packages/workspace-sdk/src/types.ts` - runtime/capability and stop/restore/capabilities DTOs.
- Modify: `packages/workspace-sdk/src/workspace-manager.ts` - expose `stop`, `restore`, `capabilities` methods.
- Modify: `packages/workspace-sdk/src/__tests__/workspace-manager.test.ts` - SDK coverage for new RPC methods.
- Modify: `.case-studies/hanlun-lms/.nexus/workspace.json` - add runtime requirements, probes/tests split.
- Create: `.case-studies/hanlun-lms/.nexus/lifecycles/probe.sh` - startup/readiness/liveness checks.
- Create: `.case-studies/hanlun-lms/.nexus/lifecycles/test-auth-flow.sh` - behavior test (auth flow).
- Create: `.case-studies/hanlun-lms/.nexus/lifecycles/test-tooling.sh` - in-workspace `opencode` and git push test.
- Modify: `.case-studies/hanlun-lms/.github/workflows/nexus-doctor.yml` - backend matrix (`dind`, `lxc`) and report artifacts.
- Modify: `docs/reference/workspace-config.md` - new config contract docs.
- Modify: `docs/reference/workspace-daemon.md` - lifecycle + capability RPC docs.

### Task 1: Add Config Model for Runtime Requirements and Doctor Tests

**Files:**
- Modify: `packages/workspace-daemon/pkg/config/types.go`
- Modify: `packages/workspace-daemon/pkg/config/types_test.go`

- [ ] **Step 1: Write failing config tests for runtime/capability/tests fields**

```go
func TestWorkspaceConfig_RuntimeAndDoctorTestsValidation(t *testing.T) {
	cfg := WorkspaceConfig{
		Version: 1,
		Runtime: RuntimeConfig{Required: []string{"dind", "lxc"}, Selection: "prefer-first"},
		Capabilities: CapabilityRequirements{Required: []string{"spotlight.tunnel"}},
		Doctor: DoctorConfig{Tests: []DoctorCommandCheck{{Name: "auth-flow", Command: "bash", Args: []string{".nexus/lifecycles/test-auth-flow.sh"}, Required: true}}},
	}
	if err := cfg.ValidateBasic(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}
```

- [ ] **Step 2: Run config tests to verify failure**

Run: `go test ./pkg/config -run RuntimeAndDoctorTestsValidation -v`
Expected: FAIL with unknown fields/types until model is implemented.

- [ ] **Step 3: Implement runtime/capability/tests types and validation**

```go
type RuntimeConfig struct {
	Required  []string `json:"required,omitempty"`
	Selection string   `json:"selection,omitempty"`
}

type CapabilityRequirements struct {
	Required []string `json:"required,omitempty"`
}

type DoctorCommandCheck struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	TimeoutMs int      `json:"timeoutMs,omitempty"`
	Retries   int      `json:"retries,omitempty"`
	Required  bool     `json:"required,omitempty"`
}
```

- [ ] **Step 4: Re-run config tests**

Run: `go test ./pkg/config -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/pkg/config/types.go packages/workspace-daemon/pkg/config/types_test.go
git commit -m "feat(config): add runtime requirements and doctor tests model"
```

### Task 2: Extend JSON Schema for New Contract

**Files:**
- Modify: `schemas/workspace.v1.schema.json`
- Test: `packages/workspace-daemon/pkg/config/loader_test.go`

- [ ] **Step 1: Add failing loader test for `doctor.tests` and `runtime.required`**

```go
func TestLoadWorkspaceConfig_DoctorTestsAndRuntime(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".nexus"), 0o755)
	_ = os.WriteFile(filepath.Join(root, ".nexus", "workspace.json"), []byte(`{"version":1,"runtime":{"required":["dind"]},"doctor":{"tests":[{"name":"tooling","command":"bash"}]}}`), 0o644)
	cfg, _, err := LoadWorkspaceConfig(root)
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if len(cfg.Doctor.Tests) != 1 { t.Fatalf("expected 1 test") }
}
```

- [ ] **Step 2: Run loader test to verify failure**

Run: `go test ./pkg/config -run DoctorTestsAndRuntime -v`
Expected: FAIL until schema/types are fully aligned.

- [ ] **Step 3: Update schema with runtime/capabilities/doctor.tests**

```json
"runtime": {
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "required": { "type": "array", "items": { "type": "string", "enum": ["dind", "lxc"] } },
    "selection": { "type": "string", "enum": ["prefer-first"] }
  }
}
```

- [ ] **Step 4: Re-run config tests and schema parse check**

Run: `go test ./pkg/config -v && node -e "JSON.parse(require('fs').readFileSync('schemas/workspace.v1.schema.json','utf8'))"`
Expected: PASS and no schema parse error.

- [ ] **Step 5: Commit**

```bash
git add schemas/workspace.v1.schema.json packages/workspace-daemon/pkg/config/loader_test.go
git commit -m "feat(schema): add runtime capabilities and doctor tests fields"
```

### Task 3: Add Runtime Driver Interface and Capability Admission

**Files:**
- Create: `packages/workspace-daemon/pkg/runtime/driver.go`
- Create: `packages/workspace-daemon/pkg/runtime/factory.go`
- Create: `packages/workspace-daemon/pkg/runtime/factory_test.go`

- [ ] **Step 1: Write failing tests for backend selection and capability checks**

```go
func TestSelectDriver_PreferFirst(t *testing.T) {
	f := NewFactory([]Capability{{Name: "runtime.dind", Available: true}})
	_, err := f.SelectDriver([]string{"dind", "lxc"}, "prefer-first")
	if err != nil { t.Fatalf("expected dind selection, got %v", err) }
}
```

- [ ] **Step 2: Run runtime factory tests to verify failure**

Run: `go test ./pkg/runtime -run SelectDriver -v`
Expected: FAIL due missing package/files.

- [ ] **Step 3: Implement interface and selection logic**

```go
type Driver interface {
	Backend() string
	Create(ctx context.Context, req CreateRequest) error
	Start(ctx context.Context, workspaceID string) error
	Stop(ctx context.Context, workspaceID string) error
	Restore(ctx context.Context, workspaceID string) error
	Destroy(ctx context.Context, workspaceID string) error
}
```

- [ ] **Step 4: Run runtime package tests**

Run: `go test ./pkg/runtime -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/pkg/runtime/driver.go packages/workspace-daemon/pkg/runtime/factory.go packages/workspace-daemon/pkg/runtime/factory_test.go
git commit -m "feat(runtime): add backend driver interface and admission checks"
```

### Task 4: Implement dind Driver

**Files:**
- Create: `packages/workspace-daemon/pkg/runtime/dind/driver.go`
- Create: `packages/workspace-daemon/pkg/runtime/dind/driver_test.go`

- [ ] **Step 1: Write failing dind command execution tests**

```go
func TestDindDriver_StartCallsDockerComposeUp(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)
	_ = d.Start(context.Background(), "ws-1")
	if !runner.Called("docker", "compose", "up", "-d") {
		t.Fatal("expected docker compose up -d")
	}
}
```

- [ ] **Step 2: Run dind tests to verify failure**

Run: `go test ./pkg/runtime/dind -v`
Expected: FAIL until implementation exists.

- [ ] **Step 3: Implement dind driver commands with injected runner**

```go
func (d *Driver) Start(ctx context.Context, workspaceID string) error {
	return d.runner.Run(ctx, d.workspaceDir(workspaceID), "docker", "compose", "up", "-d")
}
```

- [ ] **Step 4: Re-run dind tests**

Run: `go test ./pkg/runtime/dind -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/pkg/runtime/dind/driver.go packages/workspace-daemon/pkg/runtime/dind/driver_test.go
git commit -m "feat(runtime): implement dind backend driver"
```

### Task 5: Implement lxc Driver

**Files:**
- Create: `packages/workspace-daemon/pkg/runtime/lxc/driver.go`
- Create: `packages/workspace-daemon/pkg/runtime/lxc/driver_test.go`

- [ ] **Step 1: Write failing lxc command tests**

```go
func TestLXCDriver_StartCallsLxcStart(t *testing.T) {
	runner := &fakeRunner{}
	d := NewDriver(runner)
	_ = d.Start(context.Background(), "ws-1")
	if !runner.Called("lxc", "start", "ws-1") {
		t.Fatal("expected lxc start")
	}
}
```

- [ ] **Step 2: Run lxc tests to verify failure**

Run: `go test ./pkg/runtime/lxc -v`
Expected: FAIL until implementation exists.

- [ ] **Step 3: Implement lxc lifecycle methods**

```go
func (d *Driver) Stop(ctx context.Context, workspaceID string) error {
	return d.runner.Run(ctx, "", "lxc", "stop", workspaceID)
}
```

- [ ] **Step 4: Re-run lxc tests**

Run: `go test ./pkg/runtime/lxc -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/pkg/runtime/lxc/driver.go packages/workspace-daemon/pkg/runtime/lxc/driver_test.go
git commit -m "feat(runtime): implement lxc backend driver"
```

### Task 6: Persist Workspace Records and Add Stop/Restore Lifecycle

**Files:**
- Modify: `packages/workspace-daemon/pkg/workspacemgr/types.go`
- Modify: `packages/workspace-daemon/pkg/workspacemgr/manager.go`
- Modify: `packages/workspace-daemon/pkg/workspacemgr/manager_test.go`

- [ ] **Step 1: Write failing manager tests for persisted stop/restore**

```go
func TestManager_StopRestorePersistsState(t *testing.T) {
	m := NewManager(t.TempDir())
	ws, _ := m.Create(context.Background(), CreateSpec{Repo: "x", WorkspaceName: "w", AgentProfile: "default"})
	_ = m.Stop(ws.ID)
	m2 := NewManager(m.Root())
	r, ok := m2.Restore(ws.ID)
	if !ok || r.State != StateRestored { t.Fatal("expected restored state") }
}
```

- [ ] **Step 2: Run manager tests to verify failure**

Run: `go test ./pkg/workspacemgr -run StopRestorePersistsState -v`
Expected: FAIL until persistence and new states exist.

- [ ] **Step 3: Implement states, metadata, and record store**

```go
const (
	StateCreated  WorkspaceState = "created"
	StateRunning  WorkspaceState = "running"
	StateStopped  WorkspaceState = "stopped"
	StateRestored WorkspaceState = "restored"
	StateRemoved  WorkspaceState = "removed"
)
```

- [ ] **Step 4: Re-run workspace manager tests**

Run: `go test ./pkg/workspacemgr -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/pkg/workspacemgr/types.go packages/workspace-daemon/pkg/workspacemgr/manager.go packages/workspace-daemon/pkg/workspacemgr/manager_test.go
git commit -m "feat(workspace): persist records and add stop/restore lifecycle"
```

### Task 7: Add RPC Methods for Capabilities and Workspace Stop/Restore

**Files:**
- Create: `packages/workspace-daemon/pkg/handlers/capabilities.go`
- Modify: `packages/workspace-daemon/pkg/handlers/workspace_manager.go`
- Modify: `packages/workspace-daemon/pkg/server/server.go`
- Test: `packages/workspace-daemon/pkg/handlers/workspace_manager_test.go`

- [ ] **Step 1: Write failing handler tests for `workspace.stop`/`workspace.restore`**

```go
func TestHandleWorkspaceStop(t *testing.T) {
	// call handler with {id:"ws-1"}; expect {stopped:true}
}
```

- [ ] **Step 2: Run handler tests to verify failure**

Run: `go test ./pkg/handlers -run WorkspaceStop -v`
Expected: FAIL until methods are wired.

- [ ] **Step 3: Implement handlers and server method routing**

```go
case "workspace.stop":
	result, err = handlers.HandleWorkspaceStop(ctx, msg.Params, s.workspaceMgr)
case "workspace.restore":
	result, err = handlers.HandleWorkspaceRestore(ctx, msg.Params, s.workspaceMgr)
case "capabilities.list":
	result, err = handlers.HandleCapabilitiesList(ctx, msg.Params, s.runtimeFactory)
```

- [ ] **Step 4: Re-run handler/server tests**

Run: `go test ./pkg/handlers ./pkg/server -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/pkg/handlers/capabilities.go packages/workspace-daemon/pkg/handlers/workspace_manager.go packages/workspace-daemon/pkg/server/server.go packages/workspace-daemon/pkg/handlers/workspace_manager_test.go
git commit -m "feat(rpc): add capabilities and workspace stop/restore methods"
```

### Task 8: Implement Doctor Two-Phase Pipeline (`probes` then `tests`)

**Files:**
- Modify: `packages/workspace-daemon/cmd/nexus/main.go`
- Modify: `packages/workspace-daemon/cmd/nexus/main_test.go`

- [ ] **Step 1: Write failing test that tests do not run after required probe failure**

```go
func TestDoctor_SkipsTestsWhenRequiredProbeFails(t *testing.T) {
	// configure one required failing probe and one test; assert test status is "not_run"
}
```

- [ ] **Step 2: Run doctor tests to verify failure**

Run: `go test ./cmd/nexus -run SkipsTestsWhenRequiredProbeFails -v`
Expected: FAIL until phase pipeline is implemented.

- [ ] **Step 3: Implement phased execution and report structure**

```go
probeResults, probeErr := runChecks(opts, cfg.Doctor.Probes, "probe")
if probeErr != nil {
	testResults := markChecksNotRun(cfg.Doctor.Tests, "probe_failed")
	_ = writeReport(opts.reportJSON, append(probeResults, testResults...))
	return probeErr
}
testResults, testErr := runChecks(opts, cfg.Doctor.Tests, "test")
```

- [ ] **Step 4: Re-run doctor tests**

Run: `go test ./cmd/nexus -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-daemon/cmd/nexus/main.go packages/workspace-daemon/cmd/nexus/main_test.go
git commit -m "feat(doctor): run probes before tests with hard gate"
```

### Task 9: Update SDK for Stop/Restore/Capabilities APIs

**Files:**
- Modify: `packages/workspace-sdk/src/types.ts`
- Modify: `packages/workspace-sdk/src/workspace-manager.ts`
- Modify: `packages/workspace-sdk/src/__tests__/workspace-manager.test.ts`

- [ ] **Step 1: Add failing SDK tests for new manager methods**

```ts
it('calls workspace.stop', async () => {
  await manager.stop('ws-1')
  expect(mockRequest).toHaveBeenCalledWith('workspace.stop', { id: 'ws-1' })
})
```

- [ ] **Step 2: Run SDK test to verify failure**

Run: `pnpm --filter @nexus/workspace-sdk test workspace-manager.test.ts`
Expected: FAIL with missing method.

- [ ] **Step 3: Implement SDK types and methods**

```ts
async stop(id: string): Promise<boolean> {
  const result = await this.client.request<{ stopped: boolean }>('workspace.stop', { id })
  return result.stopped
}
```

- [ ] **Step 4: Re-run SDK tests**

Run: `pnpm --filter @nexus/workspace-sdk test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add packages/workspace-sdk/src/types.ts packages/workspace-sdk/src/workspace-manager.ts packages/workspace-sdk/src/__tests__/workspace-manager.test.ts
git commit -m "feat(sdk): add capabilities and workspace stop/restore APIs"
```

### Task 10: Wire Hanlun Case Study to Probes + Tests Contract

**Files:**
- Modify: `.case-studies/hanlun-lms/.nexus/workspace.json`
- Modify: `.case-studies/hanlun-lms/.nexus/lifecycles/probe.sh`
- Create: `.case-studies/hanlun-lms/.nexus/lifecycles/test-auth-flow.sh`
- Create: `.case-studies/hanlun-lms/.nexus/lifecycles/test-tooling.sh`

- [ ] **Step 1: Write failing doctor run expecting missing test scripts**

Run: `go run ./cmd/nexus doctor --project-root "$(pwd)/.case-studies/hanlun-lms" --suite hanlun-root`
Expected: FAIL because `doctor.tests` scripts are not present yet.

- [ ] **Step 2: Add `doctor.probes[]` and `doctor.tests[]` entries**

```json
"doctor": {
  "probes": [{ "name": "startup-readiness-liveness", "command": "bash", "args": [".nexus/lifecycles/probe.sh"], "required": true }],
  "tests": [
    { "name": "auth-flow", "command": "bash", "args": [".nexus/lifecycles/test-auth-flow.sh"], "required": true },
    { "name": "tooling-opencode-git", "command": "bash", "args": [".nexus/lifecycles/test-tooling.sh"], "required": true }
  ]
}
```

- [ ] **Step 3: Implement behavior test scripts**

```bash
# .nexus/lifecycles/test-tooling.sh
opencode --version
tmp_remote="$(mktemp -d)/remote.git"
git init --bare "$tmp_remote"
git remote add doctor-test "$tmp_remote" || git remote set-url doctor-test "$tmp_remote"
git push doctor-test HEAD:refs/heads/doctor-check
```

- [ ] **Step 4: Re-run doctor against case study**

Run: `go run ./cmd/nexus doctor --project-root "$(pwd)/.case-studies/hanlun-lms" --suite hanlun-root --report-json /tmp/hanlun-report.json`
Expected: PASS in isolated runtime environment.

- [ ] **Step 5: Commit**

```bash
git add .case-studies/hanlun-lms/.nexus/workspace.json .case-studies/hanlun-lms/.nexus/lifecycles/probe.sh .case-studies/hanlun-lms/.nexus/lifecycles/test-auth-flow.sh .case-studies/hanlun-lms/.nexus/lifecycles/test-tooling.sh
git commit -m "test(case-study): add probe/test contract and tooling checks"
```

### Task 11: Add Backend Matrix CI Gate for Case Studies

**Files:**
- Modify: `.case-studies/hanlun-lms/.github/workflows/nexus-doctor.yml`

- [ ] **Step 1: Add failing matrix check locally (yaml lint or workflow validation)**

Run: `gh workflow view .github/workflows/nexus-doctor.yml --yaml >/dev/null`
Expected: FAIL until matrix keys are correctly added.

- [ ] **Step 2: Add backend matrix and doctor invocation per backend**

```yaml
strategy:
  matrix:
    backend: [dind, lxc]
env:
  NEXUS_RUNTIME_BACKEND: ${{ matrix.backend }}
```

- [ ] **Step 3: Upload per-backend doctor report artifacts**

```yaml
with:
  name: nexus-doctor-report-${{ matrix.backend }}
```

- [ ] **Step 4: Verify workflow syntax**

Run: `gh workflow view .github/workflows/nexus-doctor.yml --yaml >/dev/null`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .case-studies/hanlun-lms/.github/workflows/nexus-doctor.yml
git commit -m "ci(case-study): gate doctor on dind and lxc backends"
```

### Task 12: Update User Docs for New Runtime and Health Contract

**Files:**
- Modify: `docs/reference/workspace-config.md`
- Modify: `docs/reference/workspace-daemon.md`

- [ ] **Step 1: Add docs test checklist (manual) and expected examples**

```md
## Runtime Requirements
"runtime": { "required": ["dind", "lxc"], "selection": "prefer-first" }

## Doctor Phases
1) probes
2) tests
```

- [ ] **Step 2: Update workspace config reference with runtime/capability and probes/tests examples**

```json
"doctor": {
  "probes": [{"name":"startup","command":"bash","args":[".nexus/lifecycles/probe.sh"],"required":true}],
  "tests": [{"name":"auth-flow","command":"bash","args":[".nexus/lifecycles/test-auth-flow.sh"],"required":true}]
}
```

- [ ] **Step 3: Update daemon RPC reference (`capabilities.list`, `workspace.stop`, `workspace.restore`)**

```md
| `capabilities.list` | List runtime and toolchain capabilities |
| `workspace.stop` | Stop compute, persist state |
| `workspace.restore` | Restore persisted workspace state |
```

- [ ] **Step 4: Validate docs for consistency with implemented field names**

Run: `grep -n "runtimeProbe\|authProbe\|probe-tags\|runOnEvents" docs/reference/workspace-*.md`
Expected: no matches.

- [ ] **Step 5: Commit**

```bash
git add docs/reference/workspace-config.md docs/reference/workspace-daemon.md
git commit -m "docs: describe isolated runtime and probe/test health contract"
```

## Final Verification Checklist

- [ ] Run daemon unit tests: `go test ./...` in `packages/workspace-daemon`
- [ ] Run SDK tests: `pnpm --filter @nexus/workspace-sdk test`
- [ ] Run case-study doctor locally with backend override (`dind` and `lxc`)
- [ ] Confirm reports contain both `phase: "probe"` and `phase: "test"`
- [ ] Verify no host port collision requirement in normal workflow (Spotlight-only ingress)
