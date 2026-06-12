package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockSqlStore struct {
	listTablesFn func() ([]string, error)
	queryFn      func(sql string) ([]string, [][]any, error)
}

func (m *mockSqlStore) ListTables() ([]string, error) { return m.listTablesFn() }
func (m *mockSqlStore) Query(sql string) ([]string, [][]any, error) {
	return m.queryFn(sql)
}

func TestListTables(t *testing.T) {
	tests := []struct {
		name       string
		store      SqlStore
		wantStatus int
		wantBody   string
	}{
		{
			name: "returns tables",
			store: &mockSqlStore{listTablesFn: func() ([]string, error) {
				return []string{"scripts", "users"}, nil
			}},
			wantStatus: http.StatusOK,
			wantBody:   `["scripts","users"]`,
		},
		{
			name: "empty list",
			store: &mockSqlStore{listTablesFn: func() ([]string, error) {
				return []string{}, nil
			}},
			wantStatus: http.StatusOK,
			wantBody:   `[]`,
		},
		{
			name: "store error",
			store: &mockSqlStore{listTablesFn: func() ([]string, error) {
				return nil, fmt.Errorf("db error")
			}},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SqlHandler{Store: tt.store}
			w := httptest.NewRecorder()
			h.ListTables(w, httptest.NewRequest("GET", "/", nil))

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantBody != "" {
				got := strings.TrimSpace(w.Body.String())
				if got != tt.wantBody {
					t.Errorf("body = %s, want %s", got, tt.wantBody)
				}
			}
		})
	}
}

func TestRunSQL(t *testing.T) {
	okStore := &mockSqlStore{queryFn: func(sql string) ([]string, [][]any, error) {
		return []string{"id", "name"}, [][]any{{"1", "hello"}}, nil
	}}
	errStore := &mockSqlStore{queryFn: func(sql string) ([]string, [][]any, error) {
		return nil, nil, fmt.Errorf("syntax error")
	}}

	tests := []struct {
		name       string
		body       string
		store      SqlStore
		wantStatus int
		wantError  string
		wantCols   int
		wantRows   int
	}{
		{
			name:       "valid SELECT",
			body:       `{"sql":"SELECT * FROM scripts"}`,
			store:      okStore,
			wantStatus: http.StatusOK,
			wantCols:   2,
			wantRows:   1,
		},
		{
			name:       "EXPLAIN allowed",
			body:       `{"sql":"EXPLAIN QUERY PLAN SELECT 1"}`,
			store:      okStore,
			wantStatus: http.StatusOK,
		},
		{
			name:       "WITH allowed",
			body:       `{"sql":"WITH t AS (SELECT 1) SELECT * FROM t"}`,
			store:      okStore,
			wantStatus: http.StatusOK,
		},
		{
			name:       "SELECT with leading whitespace",
			body:       `{"sql":"  SELECT 1"}`,
			store:      okStore,
			wantStatus: http.StatusOK,
		},
		{
			name: "empty SQL",
			body: `{"sql":""}`,
			store:      okStore,
			wantStatus: http.StatusBadRequest,
			wantError:  "SQL is required",
		},
		{
			name: "whitespace-only SQL",
			body: `{"sql":"   "}`,
			store:      okStore,
			wantStatus: http.StatusBadRequest,
			wantError:  "SQL is required",
		},
		{
			name:       "invalid JSON body",
			body:       `{bad`,
			store:      okStore,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "PRAGMA blocked",
			body:       `{"sql":"PRAGMA journal_mode=WAL"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
			wantError:  "PRAGMA statements are not allowed",
		},
		{
			name:       "INSERT blocked",
			body:       `{"sql":"INSERT INTO scripts VALUES (1)"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
			wantError:  "only SELECT, EXPLAIN, and WITH statements are allowed",
		},
		{
			name:       "UPDATE blocked",
			body:       `{"sql":"UPDATE scripts SET name='x'"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "DELETE blocked",
			body:       `{"sql":"DELETE FROM scripts"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "DROP blocked",
			body:       `{"sql":"DROP TABLE scripts"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "ALTER blocked",
			body:       `{"sql":"ALTER TABLE scripts ADD COLUMN x"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "CREATE blocked",
			body:       `{"sql":"CREATE TABLE t(x)"}`,
			store:      okStore,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "query execution error",
			body:       `{"sql":"SELECT * FROM bad"}`,
			store:      errStore,
			wantStatus: http.StatusOK,
			wantError:  "syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SqlHandler{Store: tt.store}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			h.RunSQL(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantError != "" {
				var resp map[string]string
				json.NewDecoder(w.Body).Decode(&resp)
				if resp["error"] != tt.wantError {
					t.Errorf("error = %q, want %q", resp["error"], tt.wantError)
				}
			}

			if tt.wantCols > 0 || tt.wantRows > 0 {
				var resp sqlRunResp
				json.NewDecoder(w.Body).Decode(&resp)
				if len(resp.Columns) != tt.wantCols {
					t.Errorf("columns = %d, want %d", len(resp.Columns), tt.wantCols)
				}
				if len(resp.Rows) != tt.wantRows {
					t.Errorf("rows = %d, want %d", len(resp.Rows), tt.wantRows)
				}
			}
		})
	}
}
