package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	_ "modernc.org/sqlite"
)

type SqlHandler struct {
	roDB *sql.DB
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

func NewSqlHandler(dbPath string) (*SqlHandler, error) {
	roDB, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if _, err := roDB.Exec("PRAGMA query_only = ON"); err != nil {
		roDB.Close()
		return nil, err
	}
	return &SqlHandler{roDB: roDB}, nil
}

func (h *SqlHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	rows, err := h.roDB.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		tables = append(tables, name)
	}
	if tables == nil {
		tables = []string{}
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
		rows, err := h.roDB.Query(sqlText)
		if err != nil {
			writeJSON(w, http.StatusOK, sqlRunResp{Error: err.Error()})
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			writeJSON(w, http.StatusOK, sqlRunResp{Error: err.Error()})
			return
		}

		var resultRows [][]any
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				writeJSON(w, http.StatusOK, sqlRunResp{Error: err.Error()})
				return
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
		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusOK, sqlRunResp{Error: err.Error()})
			return
		}

		if resultRows == nil {
			resultRows = [][]any{}
		}

		writeJSON(w, http.StatusOK, sqlRunResp{
			Columns:  columns,
			Rows:     resultRows,
			RowCount: len(resultRows),
		})
	} else {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only SELECT, EXPLAIN, and WITH statements are allowed"})
	}
}
