package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	_ "modernc.org/sqlite"
)

type SqlStore interface {
	ListTables() ([]string, error)
	Query(sql string) (columns []string, rows [][]any, err error)
}

type SqlHandler struct {
	Store SqlStore
}

type sqlRunReq struct {
	SQL string `json:"sql"`
}

type sqlRunResp struct {
	Columns  []string `json:"columns"`
	Rows     [][]any  `json:"rows"`
	RowCount int      `json:"rowCount"`
	Error    string   `json:"error,omitempty"`
}

type sqliteStore struct {
	db *sql.DB
}

func NewSqlHandler(dbPath string) (*SqlHandler, error) {
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA query_only = ON"); err != nil {
		db.Close()
		return nil, err
	}
	return &SqlHandler{Store: &sqliteStore{db: db}}, nil
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

func (h *SqlHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	tables, err := h.Store.ListTables()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tables)
}

func (h *SqlHandler) RunSQL(w http.ResponseWriter, r *http.Request) {
	var req sqlRunReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	sqlText := strings.TrimSpace(req.SQL)
	if sqlText == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "SQL is required"})
		return
	}

	upper := strings.ToUpper(sqlText)

	if strings.HasPrefix(upper, "PRAGMA") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "PRAGMA statements are not allowed"})
		return
	}

	isQuery := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "WITH")

	if isQuery {
		columns, rows, err := h.Store.Query(sqlText)
		if err != nil {
			writeJSON(w, http.StatusOK, sqlRunResp{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, sqlRunResp{
			Columns:  columns,
			Rows:     rows,
			RowCount: len(rows),
		})
	} else {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only SELECT, EXPLAIN, and WITH statements are allowed"})
	}
}
