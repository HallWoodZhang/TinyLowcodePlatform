package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type Store interface {
	ListTables() ([]string, error)
	Query(sqlText string) (columns []string, rows [][]any, err error)
}

func New(driver, dsn string) (Store, error) {
	switch strings.ToLower(driver) {
	case "sqlite", "sqlite3", "":
		return newSQLite(dsn)
	case "mysql":
		return nil, fmt.Errorf("mysql driver not yet implemented; add github.com/go-sql-driver/mysql dependency")
	case "postgres", "postgresql":
		return nil, fmt.Errorf("postgres driver not yet implemented; add github.com/lib/pq dependency")
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}

type sqliteStore struct {
	db *sql.DB
}

func newSQLite(path string) (*sqliteStore, error) {
	if path == "" {
		path = "scripts.db"
	}
	db, err := sql.Open("sqlite", path+"?_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA query_only = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("readonly: %w", err)
	}
	return &sqliteStore{db: db}, nil
}

func (s *sqliteStore) ListTables() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if tables == nil {
		tables = []string{}
	}
	return tables, rows.Err()
}

func (s *sqliteStore) Query(sqlText string) ([]string, [][]any, error) {
	rows, err := s.db.Query(sqlText)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var resultRows [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		for i, v := range values {
			switch t := v.(type) {
			case []byte:
				values[i] = string(t)
			case int64:
				values[i] = fmt.Sprintf("%d", t)
			}
		}
		resultRows = append(resultRows, values)
	}
	if resultRows == nil {
		resultRows = [][]any{}
	}
	return columns, resultRows, rows.Err()
}
