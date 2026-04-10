# SQLite Node Store Cutover Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the sqlite-only persistence cutover for workspace and spotlight state, remove legacy JSON persistence codepaths, and keep RPC behavior stable.

**Architecture:** Introduce explicit persistence interfaces at package boundaries so `workspacemgr` and `spotlight` depend on abstractions rather than concrete sqlite details. Implement those interfaces in `pkg/store` using goose-managed sqlite tables, then wire server lifecycle hydration/mutation through the interfaces. Preserve in-memory behavior while replacing file-based JSON persistence.

**Tech Stack:** Go, modernc sqlite, pressly/goose, existing daemon RPC handlers/tests

---

### Task 1: Introduce Storage Abstractions and Sqlite Implementations

**Files:**
- Create: `packages/nexus/pkg/store/workspace_repo.go`
- Create: `packages/nexus/pkg/store/spotlight_repo.go`
- Modify: `packages/nexus/pkg/store/node_store.go`
- Test: `packages/nexus/pkg/store/node_store_test.go`

- [ ] **Step 1: Write failing abstraction compile test (workspace + spotlight repo contracts)**

```go
// packages/nexus/pkg/store/node_store_test.go
func TestNodeStore_ImplementsRepositories(t *testing.T) {
    path := filepath.Join(t.TempDir(), "node.db")
    st, err := store.Open(path)
    if err != nil {
        t.Fatalf("open store: %v", err)
    }
    defer st.Close()

    var _ store.WorkspaceRepository = st
    var _ store.SpotlightRepository = st
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/store -run TestNodeStore_ImplementsRepositories -count=1`
Expected: FAIL with undefined interface or missing method implementation.

- [ ] **Step 3: Add repository interfaces and map NodeStore methods to contracts**

```go
// packages/nexus/pkg/store/workspace_repo.go
package store

type WorkspaceRepository interface {
    UpsertWorkspaceRow(row WorkspaceRow) error
    DeleteWorkspace(id string) error
    ListWorkspaceRows() ([]WorkspaceRow, error)
}

// packages/nexus/pkg/store/spotlight_repo.go
package store

type SpotlightRepository interface {
    UpsertSpotlightForwardRow(row SpotlightForwardRow) error
    DeleteSpotlightForwardRow(id string) error
    ListSpotlightForwardRows() ([]SpotlightForwardRow, error)
}
```

```go
// packages/nexus/pkg/store/node_store.go
func (s *NodeStore) UpsertSpotlightForwardRow(row SpotlightForwardRow) error {
    if row.ID == "" || row.WorkspaceID == "" || row.LocalPort <= 0 || len(row.Payload) == 0 {
        return fmt.Errorf("invalid spotlight row")
    }
    _, err := s.db.Exec(
        `INSERT INTO spotlight_forwards(id, workspace_id, local_port, payload_json, created_at)
         VALUES(?, ?, ?, ?, ?)
         ON CONFLICT(id) DO UPDATE SET
           workspace_id=excluded.workspace_id,
           local_port=excluded.local_port,
           payload_json=excluded.payload_json,
           created_at=excluded.created_at`,
        row.ID, row.WorkspaceID, row.LocalPort, string(row.Payload), row.CreatedAt.UTC().Format(time.RFC3339Nano),
    )
    if err != nil {
        return fmt.Errorf("upsert spotlight forward: %w", err)
    }
    return nil
}

func (s *NodeStore) DeleteSpotlightForwardRow(id string) error {
    if id == "" {
        return nil
    }
    _, err := s.db.Exec(`DELETE FROM spotlight_forwards WHERE id = ?`, id)
    if err != nil {
        return fmt.Errorf("delete spotlight forward: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Replace bulk spotlight replace test with upsert/delete behavior tests**

```go
// packages/nexus/pkg/store/node_store_test.go
func TestNodeStore_UpsertDeleteSpotlightRow(t *testing.T) {
    st, _ := store.Open(filepath.Join(t.TempDir(), "node.db"))
    defer st.Close()

    row := store.SpotlightForwardRow{ID: "spot-1", WorkspaceID: "ws-1", LocalPort: 18000, Payload: []byte(`{"id":"spot-1"}`), CreatedAt: time.Now().UTC()}
    if err := st.UpsertSpotlightForwardRow(row); err != nil { t.Fatal(err) }
    rows, err := st.ListSpotlightForwardRows()
    if err != nil || len(rows) != 1 { t.Fatalf("rows=%d err=%v", len(rows), err) }

    if err := st.DeleteSpotlightForwardRow("spot-1"); err != nil { t.Fatal(err) }
    rows, err = st.ListSpotlightForwardRows()
    if err != nil || len(rows) != 0 { t.Fatalf("rows=%d err=%v", len(rows), err) }
}
```

- [ ] **Step 5: Run store tests**

Run: `go test ./pkg/store -count=1`
Expected: PASS with interface/CRUD tests green.

- [ ] **Step 6: Commit**

```bash
git add packages/nexus/pkg/store/workspace_repo.go packages/nexus/pkg/store/spotlight_repo.go packages/nexus/pkg/store/node_store.go packages/nexus/pkg/store/node_store_test.go
git commit -m "refactor(store): add repository interfaces"
```

### Task 2: Remove Workspace JSON Persistence and Depend on Workspace Repository

**Files:**
- Modify: `packages/nexus/pkg/workspacemgr/manager.go`
- Modify: `packages/nexus/pkg/workspacemgr/manager_test.go`
- Test: `packages/nexus/pkg/workspacemgr/manager_test.go`

- [ ] **Step 1: Add failing test that manager load does not read legacy JSON files**

```go
func TestManager_LoadAll_IgnoresLegacyJSON(t *testing.T) {
    root := t.TempDir()
    _ = os.MkdirAll(filepath.Join(root, "workspaces"), 0o755)
    _ = os.WriteFile(filepath.Join(root, "workspaces", "ws-legacy.json"), []byte(`{"id":"ws-legacy"}`), 0o644)

    mgr := NewManager(root)
    all := mgr.List()
    if len(all) != 0 {
        t.Fatalf("expected 0 workspaces from legacy json, got %d", len(all))
    }
}
```

- [ ] **Step 2: Run test to verify it fails against current JSON fallback behavior**

Run: `go test ./pkg/workspacemgr -run TestManager_LoadAll_IgnoresLegacyJSON -count=1`
Expected: FAIL with workspace loaded from JSON fallback.

- [ ] **Step 3: Inject repository abstraction and remove JSON path helpers/load fallback/write/delete**

```go
// packages/nexus/pkg/workspacemgr/manager.go
type workspaceRepo interface {
    UpsertWorkspaceRow(row store.WorkspaceRow) error
    DeleteWorkspace(id string) error
    ListWorkspaceRows() ([]store.WorkspaceRow, error)
}

type Manager struct {
    root       string
    repo       workspaceRepo
    mu         sync.RWMutex
    workspaces map[string]*Workspace
}

func NewManager(root string) *Manager {
    m := &Manager{root: root, workspaces: make(map[string]*Workspace)}
    st, err := store.Open(resolveNodeDBPath(root))
    if err == nil {
        m.repo = st
    } else {
        fmt.Fprintf(os.Stderr, "workspacemgr: warning: sqlite store disabled (%v)\n", err)
    }
    _ = m.loadAll()
    return m
}

func (m *Manager) loadAll() error {
    if m.repo == nil {
        return nil
    }
    rows, err := m.repo.ListWorkspaceRows()
    if err != nil {
        return fmt.Errorf("list workspace rows: %w", err)
    }
    // decode rows only; no JSON file fallback
    return nil
}
```

- [ ] **Step 4: Keep runtime behavior by preserving row payload marshaling/unmarshaling and temp-dir DB isolation helper**

```go
func resolveNodeDBPath(root string) string {
    storePath := config.NodeDBPath()
    cleanRoot := filepath.Clean(root)
    tmpPrefix := filepath.Clean(os.TempDir()) + string(filepath.Separator)
    if strings.HasPrefix(cleanRoot+string(filepath.Separator), tmpPrefix) {
        return filepath.Join(cleanRoot, ".nexus", "state", "node.db")
    }
    return storePath
}
```

- [ ] **Step 5: Run workspace manager tests**

Run: `go test ./pkg/workspacemgr -count=1`
Expected: PASS with no JSON filesystem persistence assumptions.

- [ ] **Step 6: Commit**

```bash
git add packages/nexus/pkg/workspacemgr/manager.go packages/nexus/pkg/workspacemgr/manager_test.go
git commit -m "refactor(workspacemgr): remove json persistence fallback"
```

### Task 3: Make Spotlight Manager Storage-Backed Through Abstraction

**Files:**
- Modify: `packages/nexus/pkg/spotlight/manager.go`
- Modify: `packages/nexus/pkg/spotlight/manager_test.go`
- Test: `packages/nexus/pkg/spotlight/manager_test.go`

- [ ] **Step 1: Write failing test for storage-backed startup hydration and mutation persistence**

```go
func TestManager_HydratesAndPersistsViaRepository(t *testing.T) {
    st, _ := store.Open(filepath.Join(t.TempDir(), "node.db"))
    defer st.Close()

    seed := spotlight.Forward{ID: "spot-seed", WorkspaceID: "ws-1", RemotePort: 8000, LocalPort: 18000, Host: "127.0.0.1", CreatedAt: time.Now().UTC()}
    payload, _ := json.Marshal(seed)
    _ = st.UpsertSpotlightForwardRow(store.SpotlightForwardRow{ID: seed.ID, WorkspaceID: seed.WorkspaceID, LocalPort: seed.LocalPort, Payload: payload, CreatedAt: seed.CreatedAt})

    mgr := spotlight.NewManagerWithRepository(st)
    if len(mgr.List("ws-1")) != 1 { t.Fatal("expected hydrated row") }
}
```

- [ ] **Step 2: Run test to verify it fails before abstraction wiring**

Run: `go test ./pkg/spotlight -run TestManager_HydratesAndPersistsViaRepository -count=1`
Expected: FAIL with missing constructor/repository methods.

- [ ] **Step 3: Add spotlight repository interface dependency + constructor injection + hydration method**

```go
// packages/nexus/pkg/spotlight/manager.go
type forwardRepo interface {
    UpsertSpotlightForwardRow(row store.SpotlightForwardRow) error
    DeleteSpotlightForwardRow(id string) error
    ListSpotlightForwardRows() ([]store.SpotlightForwardRow, error)
}

func NewManagerWithRepository(repo forwardRepo) *Manager {
    m := &Manager{forwards: map[string]*Forward{}, localToID: map[int]string{}, repo: repo}
    _ = m.hydrateFromRepo()
    return m
}
```

- [ ] **Step 4: Remove file-based Save/Load API and persist inside Expose/Close paths**

```go
func (m *Manager) Expose(ctx context.Context, spec ExposeSpec) (*Forward, error) {
    // existing in-memory checks
    // after creating fwd:
    if m.repo != nil {
        payload, _ := json.Marshal(fwd)
        if err := m.repo.UpsertSpotlightForwardRow(store.SpotlightForwardRow{
            ID: fwd.ID, WorkspaceID: fwd.WorkspaceID, LocalPort: fwd.LocalPort, Payload: payload, CreatedAt: fwd.CreatedAt,
        }); err != nil {
            delete(m.forwards, fwd.ID)
            delete(m.localToID, fwd.LocalPort)
            return nil, fmt.Errorf("persist spotlight expose: %w", err)
        }
    }
    return &copy, nil
}

func (m *Manager) Close(id string) bool {
    // remove in-memory
    if ok && m.repo != nil {
        if err := m.repo.DeleteSpotlightForwardRow(id); err != nil {
            // keep existing bool behavior; log/ignore at callsite policy
        }
    }
    return ok
}
```

- [ ] **Step 5: Replace save/load tests with repository hydration/persist tests**

```go
func TestExposeAndClose_PersistToRepository(t *testing.T) {
    st, _ := store.Open(filepath.Join(t.TempDir(), "node.db"))
    defer st.Close()
    mgr := NewManagerWithRepository(st)

    fwd, err := mgr.Expose(context.Background(), ExposeSpec{WorkspaceID: "ws-1", Service: "api", RemotePort: 8000, LocalPort: 18000})
    if err != nil { t.Fatal(err) }

    rows, _ := st.ListSpotlightForwardRows()
    if len(rows) != 1 { t.Fatalf("expected 1 row, got %d", len(rows)) }

    _ = mgr.Close(fwd.ID)
    rows, _ = st.ListSpotlightForwardRows()
    if len(rows) != 0 { t.Fatalf("expected 0 rows, got %d", len(rows)) }
}
```

- [ ] **Step 6: Run spotlight tests**

Run: `go test ./pkg/spotlight -count=1`
Expected: PASS with no JSON file-based tests remaining.

- [ ] **Step 7: Commit**

```bash
git add packages/nexus/pkg/spotlight/manager.go packages/nexus/pkg/spotlight/manager_test.go
git commit -m "refactor(spotlight): persist forwards via repository"
```

### Task 4: Wire Server to Sqlite-Backed Spotlight Manager and Remove Snapshot Hooks

**Files:**
- Modify: `packages/nexus/pkg/server/server.go`
- Modify: `packages/nexus/pkg/server/server_test.go`
- Test: `packages/nexus/pkg/server/server_test.go`

- [ ] **Step 1: Add failing server test proving startup does not hydrate from legacy JSON file**

```go
func TestServer_IgnoresLegacySpotlightJSON(t *testing.T) {
    workspaceDir := t.TempDir()
    statePath := filepath.Join(workspaceDir, ".nexus", "state", "spotlight-forwards.json")
    _ = os.MkdirAll(filepath.Dir(statePath), 0o755)
    _ = os.WriteFile(statePath, []byte(`[{"id":"spot-legacy","workspaceId":"ws-1","remotePort":8000,"localPort":18000}]`), 0o644)

    srv, err := NewServer(0, workspaceDir, "secret-token")
    if err != nil { t.Fatal(err) }

    if got := len(srv.spotlightMgr.List("")); got != 0 {
        t.Fatalf("expected 0 forwards, got %d", got)
    }
}
```

- [ ] **Step 2: Run test to verify it fails under JSON Load behavior**

Run: `go test ./pkg/server -run TestServer_IgnoresLegacySpotlightJSON -count=1`
Expected: FAIL with forward loaded from JSON state file.

- [ ] **Step 3: Construct server spotlight manager with sqlite repository from node store**

```go
// packages/nexus/pkg/server/server.go
func NewServer(port int, workspaceDir string, tokenSecret string) (*Server, error) {
    ws, err := workspace.NewWorkspace(workspaceDir)
    if err != nil { return nil, fmt.Errorf("failed to create workspace: %w", err) }

    mgr := workspacemgr.NewManager(workspaceDir)
    spotlightMgr := spotlight.NewManagerWithRepository(mgr.Store()) // expose repo via manager accessor

    return &Server{workspaceMgr: mgr, spotlightMgr: spotlightMgr, ...}, nil
}
```

- [ ] **Step 4: Remove `spotlightStatePath`, `Load`, and `Save` calls from startup/shutdown**

```go
// remove from NewServer:
// if err := spotlightMgr.Load(...)

// remove from Shutdown:
// if err := s.spotlightMgr.Save(...)
```

- [ ] **Step 5: Update server tests from file-based persistence assertions to sqlite-backed assertions**

```go
func TestServer_ShutdownDoesNotWriteSpotlightJSON(t *testing.T) {
    workspaceDir := t.TempDir()
    srv, _ := NewServer(0, workspaceDir, "secret-token")
    _, _ = srv.spotlightMgr.Expose(context.Background(), spotlight.ExposeSpec{WorkspaceID: "ws-1", Service: "api", RemotePort: 8000, LocalPort: 18000})
    srv.Shutdown()

    statePath := filepath.Join(workspaceDir, ".nexus", "state", "spotlight-forwards.json")
    if _, err := os.Stat(statePath); !os.IsNotExist(err) {
        t.Fatalf("expected no legacy json file, got err=%v", err)
    }
}
```

- [ ] **Step 6: Run server tests**

Run: `go test ./pkg/server -count=1`
Expected: PASS with sqlite-backed spotlight lifecycle behavior.

- [ ] **Step 7: Commit**

```bash
git add packages/nexus/pkg/server/server.go packages/nexus/pkg/server/server_test.go
git commit -m "refactor(server): remove spotlight json snapshots"
```

### Task 5: Full Regression + Runtime Persistence Proof

**Files:**
- Modify: `docs/dev/internal/testing/2026-04-09-workspace-robustness-persistence-and-versioning.md`
- Test: `packages/nexus/pkg/config/node_test.go`
- Test: `packages/nexus/pkg/handlers/*.go`
- Test: `packages/nexus/pkg/workspacemgr/manager_test.go`
- Test: `packages/nexus/pkg/spotlight/manager_test.go`
- Test: `packages/nexus/pkg/server/server_test.go`

- [ ] **Step 1: Run full touched-suite regression**

Run: `go test ./pkg/store ./pkg/workspacemgr ./pkg/spotlight ./pkg/server ./pkg/handlers ./pkg/config ./cmd/nexus -count=1`
Expected: PASS for all listed packages.

- [ ] **Step 2: Build proof binaries and run isolated daemon on non-8080 port**

Run:

```bash
go build -o /tmp/nexus-proof-daemon ./cmd/nexus
go build -o /tmp/nexus-proof-cli ./packages/sdk/js/cmd/nexus
/tmp/nexus-proof-daemon --port 8101 --token dev-token --workspace-dir /tmp/nexus-proof-clean-8101
```

Expected: health endpoint responds `{"ok":true,"service":"workspace-daemon"}`.

- [ ] **Step 3: Run live restart proof and verify spotlight restore from sqlite**

Run sequence:

```bash
node /tmp/nexus-proof-rpc/rpc_once.cjs ws://127.0.0.1:8101 dev-token spotlight.list '{"workspaceId":"<id>"}'
node /tmp/nexus-proof-rpc/rpc_once.cjs ws://127.0.0.1:8101 dev-token spotlight.applyComposePorts '{"workspaceId":"<id>","rootPath":"<repo>"}'
kill <daemon-pid>
nohup /tmp/nexus-proof-daemon --port 8101 --token dev-token --workspace-dir /tmp/nexus-proof-clean-8101 >/tmp/nexus-proof-clean-8101.log 2>&1 &
node /tmp/nexus-proof-rpc/rpc_once.cjs ws://127.0.0.1:8101 dev-token spotlight.list '{"workspaceId":"<id>"}'
```

Expected: before apply empty, after apply non-empty, after restart equivalent forward count.

- [ ] **Step 4: Update internal testing document with sqlite-only behavior decisions and proof output**

```md
## Sqlite-Only Cutover Decisions
- Legacy JSON migration: none.
- Empty sqlite on upgrade: silent empty state.
- Legacy JSON persistence paths removed.
```

- [ ] **Step 5: Commit verification doc updates**

```bash
git add docs/dev/internal/testing/2026-04-09-workspace-robustness-persistence-and-versioning.md
git commit -m "docs(testing): record sqlite-only persistence verification"
```

### Task 6: Final Hygiene Checks and Push

**Files:**
- Modify: none (verification only)

- [ ] **Step 1: Ensure no legacy JSON persistence callsites remain**

Run: `rg "spotlight-forwards.json|\.Save\(|\.Load\(|workspacesDir\(|recordPath\(" packages/nexus/pkg`
Expected: no runtime persistence callsites remaining for deleted JSON path strategy.

- [ ] **Step 2: Confirm branch status and commit list**

Run: `git status --short --branch && git log --oneline -10`
Expected: clean working tree, expected commit sequence.

- [ ] **Step 3: Push branch**

Run: `git push origin feat/workspace-robustness-persistence-versioning`
Expected: remote updated with cutover commits.
