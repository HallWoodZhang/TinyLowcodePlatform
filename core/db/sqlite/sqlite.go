package sqlite

import (
	"database/sql"
	"fmt"

	"toy-platform/core/db"
	"toy-platform/core/idgen"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	database, err := sql.Open("sqlite", path+"?_busy_timeout=5000&_fk=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &Store{db: database}
	if err := s.migrate(); err != nil {
		database.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if err := s.seed(); err != nil {
		database.Close()
		return nil, fmt.Errorf("seed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS tenants (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL UNIQUE,
			label      TEXT NOT NULL DEFAULT '',
			betamap    TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			tenant_id     TEXT NOT NULL,
			username      TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'user',
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, username),
			FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,
		`CREATE TABLE IF NOT EXISTS scripts (
			id         TEXT PRIMARY KEY,
			tenant_id  TEXT NOT NULL DEFAULT '',
			name       TEXT NOT NULL,
			label      TEXT NOT NULL,
			type       TEXT NOT NULL DEFAULT '',
			ts_code    TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name),
			FOREIGN KEY (tenant_id) REFERENCES tenants(id)
		)`,
		`CREATE TABLE IF NOT EXISTS breakpoints (
			script_id TEXT NOT NULL,
			line      INTEGER NOT NULL,
			enabled   INTEGER NOT NULL DEFAULT 1,
			PRIMARY KEY (script_id, line),
			FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE
		)`,
	}

	for _, ddl := range tables {
		if _, err := s.db.Exec(ddl); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	return nil
}

func (s *Store) seed() error {
	adminID := idgen.NewTenantID()
	_, err := s.db.Exec(`INSERT OR IGNORE INTO tenants (id, name, label, betamap) VALUES (?, 'admin', '默认管理租户', '{}')`, adminID)
	if err != nil {
		return err
	}

	row := s.db.QueryRow(`SELECT id FROM tenants WHERE name = 'admin'`)
	if err := row.Scan(&adminID); err != nil {
		return err
	}

	adminUserID := idgen.NewUserID()
	_, err = s.db.Exec(
		`INSERT OR IGNORE INTO users (id, tenant_id, username, password_hash, role) VALUES (?, ?, 'admin', ?, 'admin')`,
		adminUserID, adminID, "$2a$10$placeholder_admin123_hash",
	)
	return err
}

// --- TenantStore ---

func (s *Store) CreateTenant(name, label, betamap string) (*db.Tenant, error) {
	if betamap == "" {
		betamap = "{}"
	}
	id := idgen.NewTenantID()
	if _, err := s.db.Exec(`INSERT INTO tenants (id, name, label, betamap) VALUES (?, ?, ?, ?)`,
		id, name, label, betamap); err != nil {
		return nil, err
	}
	userID := idgen.NewUserID()
	if _, err := s.db.Exec(`INSERT OR IGNORE INTO users (id, tenant_id, username, password_hash, role) VALUES (?, ?, ?, ?, 'tenant_admin')`,
		userID, id, name, "$2a$10$placeholder_master_hash"); err != nil {
		return nil, fmt.Errorf("create master user: %w", err)
	}
	return s.GetTenant(id)
}

func (s *Store) GetTenant(id string) (*db.Tenant, error) {
	var t db.Tenant
	err := s.db.QueryRow(`SELECT id, name, label, betamap, created_at, updated_at FROM tenants WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Label, &t.Betamap, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) GetTenantByName(name string) (*db.Tenant, error) {
	var t db.Tenant
	err := s.db.QueryRow(`SELECT id, name, label, betamap, created_at, updated_at FROM tenants WHERE name = ?`, name).
		Scan(&t.ID, &t.Name, &t.Label, &t.Betamap, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTenants() ([]db.Tenant, error) {
	rows, err := s.db.Query(`SELECT id, name, label, betamap, created_at, updated_at FROM tenants ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []db.Tenant
	for rows.Next() {
		var t db.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Label, &t.Betamap, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

func (s *Store) UpdateTenant(id string, name, label, betamap *string) (*db.Tenant, error) {
	if name != nil {
		if _, err := s.db.Exec(`UPDATE tenants SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *name, id); err != nil {
			return nil, err
		}
	}
	if label != nil {
		if _, err := s.db.Exec(`UPDATE tenants SET label = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *label, id); err != nil {
			return nil, err
		}
	}
	if betamap != nil {
		if _, err := s.db.Exec(`UPDATE tenants SET betamap = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *betamap, id); err != nil {
			return nil, err
		}
	}
	return s.GetTenant(id)
}

func (s *Store) DeleteTenant(id string) error {
	_, err := s.db.Exec(`DELETE FROM tenants WHERE id = ?`, id)
	return err
}

// --- UserStore ---

func (s *Store) CreateUser(tenantID, username, passwordHash, role string) (*db.User, error) {
	id := idgen.NewUserID()
	if _, err := s.db.Exec(`INSERT INTO users (id, tenant_id, username, password_hash, role) VALUES (?, ?, ?, ?, ?)`,
		id, tenantID, username, passwordHash, role); err != nil {
		return nil, err
	}
	return s.GetUser(id)
}

func (s *Store) GetUser(id string) (*db.User, error) {
	var u db.User
	err := s.db.QueryRow(`SELECT id, tenant_id, username, password_hash, role, created_at, updated_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByTenantAndUsername(tenantName, username string) (*db.User, error) {
	var u db.User
	err := s.db.QueryRow(`
		SELECT u.id, u.tenant_id, u.username, u.password_hash, u.role, u.created_at, u.updated_at
		FROM users u JOIN tenants t ON u.tenant_id = t.id
		WHERE t.name = ? AND u.username = ?`, tenantName, username).
		Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListUsersByTenant(tenantID string) ([]db.User, error) {
	rows, err := s.db.Query(`SELECT id, tenant_id, username, password_hash, role, created_at, updated_at FROM users WHERE tenant_id = ? ORDER BY created_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []db.User
	for rows.Next() {
		var u db.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) ListAllUsers() ([]db.User, error) {
	rows, err := s.db.Query(`SELECT id, tenant_id, username, password_hash, role, created_at, updated_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []db.User
	for rows.Next() {
		var u db.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) DeleteUser(id string) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) UpdateUserPassword(id, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, passwordHash, id)
	return err
}

// --- ScriptStore ---

func (s *Store) ListScripts(tenantID string) ([]db.ScriptSummary, error) {
	rows, err := s.db.Query(`SELECT id, tenant_id, name, label, type, created_at, updated_at FROM scripts WHERE tenant_id = ? ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scripts []db.ScriptSummary
	for rows.Next() {
		var sc db.ScriptSummary
		if err := rows.Scan(&sc.ID, &sc.TenantID, &sc.Name, &sc.Label, &sc.Type, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		scripts = append(scripts, sc)
	}
	return scripts, rows.Err()
}

func (s *Store) ListAllScripts() ([]db.ScriptSummary, error) {
	rows, err := s.db.Query(`SELECT id, tenant_id, name, label, type, created_at, updated_at FROM scripts ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scripts []db.ScriptSummary
	for rows.Next() {
		var sc db.ScriptSummary
		if err := rows.Scan(&sc.ID, &sc.TenantID, &sc.Name, &sc.Label, &sc.Type, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		scripts = append(scripts, sc)
	}
	return scripts, rows.Err()
}

func (s *Store) GetScript(id string) (*db.Script, error) {
	var sc db.Script
	err := s.db.QueryRow(`SELECT id, tenant_id, name, label, type, ts_code, created_at, updated_at FROM scripts WHERE id = ?`, id).
		Scan(&sc.ID, &sc.TenantID, &sc.Name, &sc.Label, &sc.Type, &sc.TSCode, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sc, nil
}

func (s *Store) GetScriptByName(tenantID, name string) (*db.Script, error) {
	var sc db.Script
	err := s.db.QueryRow(`SELECT id, tenant_id, name, label, type, ts_code, created_at, updated_at FROM scripts WHERE tenant_id = ? AND name = ?`, tenantID, name).
		Scan(&sc.ID, &sc.TenantID, &sc.Name, &sc.Label, &sc.Type, &sc.TSCode, &sc.CreatedAt, &sc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sc, nil
}

func (s *Store) CreateScript(tenantID, name, label, scriptType, tsCode string) (*db.Script, error) {
	id := idgen.NewScriptID()
	if _, err := s.db.Exec(`INSERT INTO scripts (id, tenant_id, name, label, type, ts_code) VALUES (?, ?, ?, ?, ?, ?)`,
		id, tenantID, name, label, scriptType, tsCode); err != nil {
		return nil, err
	}
	return s.GetScript(id)
}

func (s *Store) UpdateScript(id string, name, label, scriptType, tsCode *string) (*db.Script, error) {
	if name != nil {
		if _, err := s.db.Exec(`UPDATE scripts SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *name, id); err != nil {
			return nil, err
		}
	}
	if label != nil {
		if _, err := s.db.Exec(`UPDATE scripts SET label = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *label, id); err != nil {
			return nil, err
		}
	}
	if scriptType != nil {
		if _, err := s.db.Exec(`UPDATE scripts SET type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *scriptType, id); err != nil {
			return nil, err
		}
	}
	if tsCode != nil {
		if _, err := s.db.Exec(`UPDATE scripts SET ts_code = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *tsCode, id); err != nil {
			return nil, err
		}
	}
	return s.GetScript(id)
}

func (s *Store) DeleteScript(id string) error {
	_, err := s.db.Exec(`DELETE FROM scripts WHERE id = ?`, id)
	return err
}

// --- BreakpointStore ---

func (s *Store) ListBreakpoints(scriptID string) ([]db.Breakpoint, error) {
	rows, err := s.db.Query(`SELECT script_id, line, enabled FROM breakpoints WHERE script_id = ? ORDER BY line`, scriptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bps []db.Breakpoint
	for rows.Next() {
		var bp db.Breakpoint
		var sid string
		if err := rows.Scan(&sid, &bp.Line, &bp.Enabled); err != nil {
			return nil, err
		}
		bp.ScriptID = sid
		bps = append(bps, bp)
	}
	return bps, rows.Err()
}

func (s *Store) SetBreakpoint(scriptID string, line int, enabled bool) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO breakpoints (script_id, line, enabled) VALUES (?, ?, ?)`, scriptID, line, enabled)
	return err
}

func (s *Store) DeleteBreakpoint(scriptID string, line int) error {
	_, err := s.db.Exec(`DELETE FROM breakpoints WHERE script_id = ? AND line = ?`, scriptID, line)
	return err
}

var _ db.TenantStore = (*Store)(nil)
var _ db.UserStore = (*Store)(nil)
var _ db.ScriptStore = (*Store)(nil)
var _ db.BreakpointStore = (*Store)(nil)
