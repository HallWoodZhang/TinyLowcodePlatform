package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"toy-platform/core/db"
	"toy-platform/core/runtime"
)

type mockStore struct {
	listFn   func() ([]db.ScriptSummary, error)
	getFn    func(id int64) (*db.Script, error)
	createFn func(name, label, scriptType, tsCode string) (*db.Script, error)
	updateFn func(id int64, name, label, scriptType, tsCode *string) (*db.Script, error)
	deleteFn func(id int64) error
}

func (m *mockStore) List() ([]db.ScriptSummary, error)  { return m.listFn() }
func (m *mockStore) Get(id int64) (*db.Script, error)    { return m.getFn(id) }
func (m *mockStore) Create(name, label, scriptType, tsCode string) (*db.Script, error) {
	return m.createFn(name, label, scriptType, tsCode)
}
func (m *mockStore) Update(id int64, name, label, scriptType, tsCode *string) (*db.Script, error) {
	return m.updateFn(id, name, label, scriptType, tsCode)
}
func (m *mockStore) Delete(id int64) error { return m.deleteFn(id) }

type mockRunner struct {
	runFn func(tsCode string, timeoutMs int64) runtime.RunResult
}

func (m *mockRunner) Run(tsCode string, timeoutMs int64) runtime.RunResult {
	return m.runFn(tsCode, timeoutMs)
}

func TestListScripts(t *testing.T) {
	errStore := &mockStore{listFn: func() ([]db.ScriptSummary, error) {
		return nil, fmt.Errorf("db down")
	}}
	emptyStore := &mockStore{listFn: func() ([]db.ScriptSummary, error) {
		return nil, nil
	}}
	filledStore := &mockStore{listFn: func() ([]db.ScriptSummary, error) {
		return []db.ScriptSummary{
			{ID: 1, Name: "a", Label: "A"},
			{ID: 2, Name: "b", Label: "B"},
		}, nil
	}}

	tests := []struct {
		name       string
		store      db.ScriptStore
		wantStatus int
		wantCount  int
	}{
		{"returns list", filledStore, http.StatusOK, 2},
		{"nil becomes empty array", emptyStore, http.StatusOK, 0},
		{"store error", errStore, http.StatusInternalServerError, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Store: tt.store, Runner: &mockRunner{}}
			w := httptest.NewRecorder()
			h.ListScripts(w, httptest.NewRequest("GET", "/", nil))

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				var scripts []db.ScriptSummary
				json.NewDecoder(w.Body).Decode(&scripts)
				if len(scripts) != tt.wantCount {
					t.Errorf("count = %d, want %d", len(scripts), tt.wantCount)
				}
			}
		})
	}
}

func TestCreateScript(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		store      db.ScriptStore
		wantStatus int
	}{
		{
			name: "creates successfully",
			body: `{"name":"x","label":"X","type":"ts","tsCode":"1"}`,
			store: &mockStore{createFn: func(name, label, scriptType, tsCode string) (*db.Script, error) {
				return &db.Script{ID: 10, Name: name, Label: label, Type: scriptType}, nil
			}},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid JSON",
			body:       `{bad`,
			store:      &mockStore{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing name",
			body:       `{"name":"","label":"X","type":"ts","tsCode":"1"}`,
			store:      &mockStore{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing label",
			body:       `{"name":"x","label":"","type":"ts","tsCode":"1"}`,
			store:      &mockStore{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "store error",
			body: `{"name":"x","label":"X","type":"ts","tsCode":"1"}`,
			store: &mockStore{createFn: func(name, label, scriptType, tsCode string) (*db.Script, error) {
				return nil, fmt.Errorf("duplicate")
			}},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Store: tt.store, Runner: &mockRunner{}}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			h.CreateScript(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestGetScript(t *testing.T) {
	okStore := &mockStore{getFn: func(id int64) (*db.Script, error) {
		return &db.Script{ID: id, Name: "test"}, nil
	}}
	missStore := &mockStore{getFn: func(id int64) (*db.Script, error) {
		return nil, fmt.Errorf("not found")
	}}

	tests := []struct {
		name       string
		pathID     string
		hasPath    bool
		store      db.ScriptStore
		wantStatus int
	}{
		{"found", "123", true, okStore, http.StatusOK},
		{"not found", "999", true, missStore, http.StatusNotFound},
		{"invalid id", "abc", true, okStore, http.StatusBadRequest},
		{"missing id", "", false, okStore, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Store: tt.store, Runner: &mockRunner{}}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/"+tt.pathID, nil)
			if tt.hasPath {
				req.SetPathValue("id", tt.pathID)
			}
			h.GetScript(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestUpdateScript(t *testing.T) {
	okStore := &mockStore{updateFn: func(id int64, n, l, t, c *string) (*db.Script, error) {
		return &db.Script{ID: id}, nil
	}}
	errStore := &mockStore{updateFn: func(id int64, n, l, t, c *string) (*db.Script, error) {
		return nil, fmt.Errorf("conflict")
	}}

	tests := []struct {
		name       string
		pathID     string
		body       string
		store      db.ScriptStore
		wantStatus int
	}{
		{
			name:       "update all fields",
			pathID:     "1",
			body:       `{"name":"newname","label":"NewLabel","type":"other","tsCode":"console.log(2);"}`,
			store:      okStore,
			wantStatus: http.StatusOK,
		},
		{
			name:       "partial update",
			pathID:     "1",
			body:       `{"name":"newname"}`,
			store:      okStore,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid JSON",
			pathID:     "1",
			body:       `x`,
			store:      okStore,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid id",
			pathID:     "x",
			body:       `{"name":"n"}`,
			store:      okStore,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "store error",
			pathID:     "1",
			body:       `{"name":"n"}`,
			store:      errStore,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Store: tt.store, Runner: &mockRunner{}}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("PUT", "/"+tt.pathID, strings.NewReader(tt.body))
			req.SetPathValue("id", tt.pathID)
			req.Header.Set("Content-Type", "application/json")
			h.UpdateScript(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestDeleteScript(t *testing.T) {
	okStore := &mockStore{deleteFn: func(id int64) error { return nil }}
	errStore := &mockStore{deleteFn: func(id int64) error { return fmt.Errorf("locked") }}

	tests := []struct {
		name       string
		pathID     string
		store      db.ScriptStore
		wantStatus int
	}{
		{"delete ok", "1", okStore, http.StatusNoContent},
		{"delete error", "1", errStore, http.StatusInternalServerError},
		{"invalid id", "abc", okStore, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Store: tt.store, Runner: &mockRunner{}}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("DELETE", "/"+tt.pathID, nil)
			req.SetPathValue("id", tt.pathID)
			h.DeleteScript(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRunScript(t *testing.T) {
	okStore := &mockStore{getFn: func(id int64) (*db.Script, error) {
		return &db.Script{ID: id, TSCode: "console.log(1);"}, nil
	}}
	errStore := &mockStore{getFn: func(id int64) (*db.Script, error) {
		return nil, fmt.Errorf("not found")
	}}
	okRunner := &mockRunner{runFn: func(tsCode string, timeoutMs int64) runtime.RunResult {
		return runtime.RunResult{Output: "1\n"}
	}}
	errRunner := &mockRunner{runFn: func(tsCode string, timeoutMs int64) runtime.RunResult {
		return runtime.RunResult{Error: "compile error"}
	}}

	tests := []struct {
		name       string
		pathID     string
		store      db.ScriptStore
		runner     runtime.Runner
		wantStatus int
	}{
		{"executes ok", "1", okStore, okRunner, http.StatusOK},
		{"compile error still 200", "1", okStore, errRunner, http.StatusOK},
		{"script not found", "999", errStore, okRunner, http.StatusNotFound},
		{"invalid id", "x", okStore, okRunner, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Store: tt.store, Runner: tt.runner}
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/"+tt.pathID+"/run", nil)
			req.SetPathValue("id", tt.pathID)
			h.RunScript(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       any
		wantStatus int
		wantType   string
	}{
		{"object", http.StatusOK, map[string]string{"key": "val"}, http.StatusOK, "application/json"},
		{"nil", http.StatusNoContent, nil, http.StatusNoContent, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSON(w, tt.status, tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if ct := w.Header().Get("Content-Type"); ct != tt.wantType {
				t.Errorf("Content-Type = %q, want %q", ct, tt.wantType)
			}
		})
	}
}
