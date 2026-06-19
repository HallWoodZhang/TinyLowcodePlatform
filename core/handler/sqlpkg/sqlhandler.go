package sqlpkg

import (
	"encoding/json"
	"net/http"
	"strings"

	"tiny-lowcode-platform/core/sqlstore"
)

type SqlHandler struct {
	Store sqlstore.Store
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

func New(store sqlstore.Store) *SqlHandler {
	return &SqlHandler{Store: store}
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
