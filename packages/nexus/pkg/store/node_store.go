package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type WorkspaceRow struct {
	ID        string
	Payload   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SpotlightForwardRow struct {
	ID          string
	WorkspaceID string
	LocalPort   int
	Payload     []byte
	CreatedAt   time.Time
}

type NodeStore struct {
	db *sql.DB
}

func Open(path string) (*NodeStore, error) {
	if path == "" {
		return nil, fmt.Errorf("store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	st := &NodeStore{db: db}
	if err := st.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return st, nil
}

func (s *NodeStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *NodeStore) migrate() error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(s.db, "migrations"); err != nil {
		return fmt.Errorf("run goose migrations: %w", err)
	}
	return nil
}

func (s *NodeStore) HasTable(name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var count int
	err := s.db.QueryRow(`SELECT count(1) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check table existence: %w", err)
	}
	return count > 0, nil
}

func (s *NodeStore) UpsertWorkspaceRow(row WorkspaceRow) error {
	if row.ID == "" {
		return fmt.Errorf("workspace id is required")
	}
	if len(row.Payload) == 0 {
		return fmt.Errorf("workspace payload is required")
	}
	created := row.CreatedAt.UTC().Format(time.RFC3339Nano)
	updated := row.UpdatedAt.UTC().Format(time.RFC3339Nano)

	_, err := s.db.Exec(
		`INSERT INTO workspaces(id, payload_json, created_at, updated_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			payload_json=excluded.payload_json,
			updated_at=excluded.updated_at`,
		row.ID,
		string(row.Payload),
		created,
		updated,
	)
	if err != nil {
		return fmt.Errorf("upsert workspace: %w", err)
	}

	return nil
}

func (s *NodeStore) DeleteWorkspace(id string) error {
	if id == "" {
		return nil
	}
	if _, err := s.db.Exec(`DELETE FROM workspaces WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	return nil
}

func (s *NodeStore) ListWorkspaceRows() ([]WorkspaceRow, error) {
	rows, err := s.db.Query(`SELECT id, payload_json, created_at, updated_at FROM workspaces ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list workspaces query: %w", err)
	}
	defer rows.Close()

	all := make([]WorkspaceRow, 0)
	for rows.Next() {
		var (
			id      string
			payload string
			created string
			updated string
		)
		if err := rows.Scan(&id, &payload, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan workspace row: %w", err)
		}
		createdAt, _ := time.Parse(time.RFC3339Nano, created)
		updatedAt, _ := time.Parse(time.RFC3339Nano, updated)
		all = append(all, WorkspaceRow{
			ID:        id,
			Payload:   []byte(payload),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace rows: %w", err)
	}

	return all, nil
}

func (s *NodeStore) ReplaceSpotlightForwardRows(forwards []SpotlightForwardRow) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin replace spotlight forwards: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM spotlight_forwards`); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("truncate spotlight forwards: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO spotlight_forwards(id, workspace_id, local_port, payload_json, created_at) VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare spotlight insert: %w", err)
	}
	defer stmt.Close()

	for _, fwd := range forwards {
		if fwd.ID == "" || fwd.WorkspaceID == "" || fwd.LocalPort <= 0 || len(fwd.Payload) == 0 {
			continue
		}
		if _, err := stmt.Exec(
			fwd.ID,
			fwd.WorkspaceID,
			fwd.LocalPort,
			string(fwd.Payload),
			fwd.CreatedAt.UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert spotlight forward: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit spotlight replace: %w", err)
	}

	return nil
}

func (s *NodeStore) ListSpotlightForwardRows() ([]SpotlightForwardRow, error) {
	rows, err := s.db.Query(`SELECT id, workspace_id, local_port, payload_json, created_at FROM spotlight_forwards ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list spotlight forwards query: %w", err)
	}
	defer rows.Close()

	all := make([]SpotlightForwardRow, 0)
	for rows.Next() {
		var (
			id          string
			workspaceID string
			localPort   int
			payload     string
			created     string
		)
		if err := rows.Scan(&id, &workspaceID, &localPort, &payload, &created); err != nil {
			return nil, fmt.Errorf("scan spotlight row: %w", err)
		}
		createdAt, _ := time.Parse(time.RFC3339Nano, created)
		all = append(all, SpotlightForwardRow{
			ID:          id,
			WorkspaceID: workspaceID,
			LocalPort:   localPort,
			Payload:     []byte(payload),
			CreatedAt:   createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spotlight rows: %w", err)
	}

	return all, nil
}
