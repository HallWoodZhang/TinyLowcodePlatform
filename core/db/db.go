package db

import (
	"database/sql"
	"log"
	"time"

	"github.com/bwmarrin/snowflake"
	_ "modernc.org/sqlite"
)

var idgen *snowflake.Node

func init() {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	idgen = node
}

type Script struct {
	ID        int64     `json:"id,string"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	Type      string    `json:"type"`
	TSCode    string    `json:"tsCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ScriptSummary struct {
	ID        int64     `json:"id,string"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DB struct {
	*sql.DB
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	var hasOldTable int
	db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='snippets'").Scan(&hasOldTable)

	if hasOldTable > 0 {
		if err := migrate(db); err != nil {
			db.Close()
			return nil, err
		}
	} else {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS scripts (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			label TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT '',
			ts_code TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
			db.Close()
			return nil, err
		}
	}

	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	log.Println("migrating snippets -> scripts...")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS scripts (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		label TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT '',
		ts_code TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	rows, err := tx.Query(`SELECT id, name, label, ts_code, created_at, updated_at FROM snippets`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var oldID int64
		var name, label, tsCode string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&oldID, &name, &label, &tsCode, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return err
		}
		newID := idgen.Generate().Int64()
		if _, err := tx.Exec(`INSERT OR IGNORE INTO scripts (id, name, label, type, ts_code, created_at, updated_at) VALUES (?, ?, ?, 'typescript', ?, ?, ?)`,
			newID, name, label, tsCode, createdAt, updatedAt); err != nil {
			rows.Close()
			return err
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if _, err := tx.Exec(`DROP TABLE snippets`); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("migration complete")
	return nil
}

func (d *DB) ListScripts() ([]ScriptSummary, error) {
	rows, err := d.Query(`SELECT id, name, label, type, created_at, updated_at FROM scripts ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scripts []ScriptSummary
	for rows.Next() {
		var s ScriptSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Label, &s.Type, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		scripts = append(scripts, s)
	}
	return scripts, rows.Err()
}

func (d *DB) GetScript(id int64) (*Script, error) {
	var s Script
	err := d.QueryRow(`SELECT id, name, label, type, ts_code, created_at, updated_at FROM scripts WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Label, &s.Type, &s.TSCode, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateScript(name, label, scriptType, tsCode string) (*Script, error) {
	id := idgen.Generate().Int64()
	_, err := d.Exec(`INSERT INTO scripts (id, name, label, type, ts_code) VALUES (?, ?, ?, ?, ?)`, id, name, label, scriptType, tsCode)
	if err != nil {
		return nil, err
	}
	return d.GetScript(id)
}

func (d *DB) UpdateScript(id int64, name, label, scriptType, tsCode *string) (*Script, error) {
	if name != nil {
		_, err := d.Exec(`UPDATE scripts SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *name, id)
		if err != nil {
			return nil, err
		}
	}
	if label != nil {
		_, err := d.Exec(`UPDATE scripts SET label = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *label, id)
		if err != nil {
			return nil, err
		}
	}
	if scriptType != nil {
		_, err := d.Exec(`UPDATE scripts SET type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *scriptType, id)
		if err != nil {
			return nil, err
		}
	}
	if tsCode != nil {
		_, err := d.Exec(`UPDATE scripts SET ts_code = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *tsCode, id)
		if err != nil {
			return nil, err
		}
	}
	return d.GetScript(id)
}

func (d *DB) DeleteScript(id int64) error {
	_, err := d.Exec(`DELETE FROM scripts WHERE id = ?`, id)
	return err
}
