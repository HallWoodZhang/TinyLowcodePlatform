package db

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Snippet struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	TSCode    string    `json:"tsCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SnippetSummary struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DB struct {
	*sql.DB
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS snippets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		label TEXT NOT NULL,
		ts_code TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db}, nil
}

func (d *DB) ListSnippets() ([]SnippetSummary, error) {
	rows, err := d.Query(`SELECT id, name, label, created_at, updated_at FROM snippets ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snippets []SnippetSummary
	for rows.Next() {
		var s SnippetSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Label, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		snippets = append(snippets, s)
	}
	return snippets, rows.Err()
}

func (d *DB) GetSnippet(id int64) (*Snippet, error) {
	var s Snippet
	err := d.QueryRow(`SELECT id, name, label, ts_code, created_at, updated_at FROM snippets WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Label, &s.TSCode, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateSnippet(name, label, tsCode string) (*Snippet, error) {
	result, err := d.Exec(`INSERT INTO snippets (name, label, ts_code) VALUES (?, ?, ?)`, name, label, tsCode)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return d.GetSnippet(id)
}

func (d *DB) UpdateSnippet(id int64, name, label, tsCode *string) (*Snippet, error) {
	if name != nil {
		_, err := d.Exec(`UPDATE snippets SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *name, id)
		if err != nil {
			return nil, err
		}
	}
	if label != nil {
		_, err := d.Exec(`UPDATE snippets SET label = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *label, id)
		if err != nil {
			return nil, err
		}
	}
	if tsCode != nil {
		_, err := d.Exec(`UPDATE snippets SET ts_code = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, *tsCode, id)
		if err != nil {
			return nil, err
		}
	}
	return d.GetSnippet(id)
}

func (d *DB) DeleteSnippet(id int64) error {
	_, err := d.Exec(`DELETE FROM snippets WHERE id = ?`, id)
	return err
}
