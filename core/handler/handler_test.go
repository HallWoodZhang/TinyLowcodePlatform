package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/db"
	"tiny-lowcode-platform/core/runtime"
)

type mockStore struct {
	listScriptsFn    func(tenantID string) ([]db.ScriptSummary, error)
	listAllScriptsFn func() ([]db.ScriptSummary, error)
	getScriptFn      func(id string) (*db.Script, error)
	getScriptByNameFn func(tenantID, name string) (*db.Script, error)
	createScriptFn   func(tenantID, name, label, scriptType, tsCode string) (*db.Script, error)
	updateScriptFn   func(id string, name, label, scriptType, tsCode *string) (*db.Script, error)
	deleteScriptFn   func(id string) error
	listBpFn         func(scriptID string) ([]db.Breakpoint, error)
	setBpFn          func(scriptID string, line int, enabled bool) error
	deleteBpFn       func(scriptID string, line int) error
}

func (m *mockStore) CreateTenant(name, label, betamap string) (*db.Tenant, error) { return nil, nil }
func (m *mockStore) GetTenant(id string) (*db.Tenant, error)                       { return nil, nil }
func (m *mockStore) GetTenantByName(name string) (*db.Tenant, error)               { return nil, nil }
func (m *mockStore) ListTenants() ([]db.Tenant, error)                             { return nil, nil }
func (m *mockStore) UpdateTenant(id string, name, label, betamap *string) (*db.Tenant, error) { return nil, nil }
func (m *mockStore) DeleteTenant(id string) error                                  { return nil }
func (m *mockStore) CreateUser(tenantID, username, passwordHash, role string) (*db.User, error) { return nil, nil }
func (m *mockStore) GetUser(id string) (*db.User, error)                           { return nil, nil }
func (m *mockStore) GetUserByTenantAndUsername(tenantName, username string) (*db.User, error) { return nil, nil }
func (m *mockStore) ListUsersByTenant(tenantID string) ([]db.User, error)          { return nil, nil }
func (m *mockStore) ListAllUsers() ([]db.User, error)                              { return nil, nil }
func (m *mockStore) DeleteUser(id string) error                                    { return nil }
func (m *mockStore) UpdateUserPassword(id, passwordHash string) error              { return nil }

func (m *mockStore) ListScripts(tenantID string) ([]db.ScriptSummary, error) {
	if m.listScriptsFn != nil { return m.listScriptsFn(tenantID) }
	return nil, nil
}
func (m *mockStore) ListAllScripts() ([]db.ScriptSummary, error) {
	if m.listAllScriptsFn != nil { return m.listAllScriptsFn() }
	return nil, nil
}
func (m *mockStore) GetScript(id string) (*db.Script, error) {
	if m.getScriptFn != nil { return m.getScriptFn(id) }
	return nil, fmt.Errorf("not found")
}
func (m *mockStore) GetScriptByName(tenantID, name string) (*db.Script, error) {
	if m.getScriptByNameFn != nil { return m.getScriptByNameFn(tenantID, name) }
	return nil, fmt.Errorf("not found")
}
func (m *mockStore) CreateScript(tenantID, name, label, scriptType, tsCode string) (*db.Script, error) {
	if m.createScriptFn != nil { return m.createScriptFn(tenantID, name, label, scriptType, tsCode) }
	return nil, nil
}
func (m *mockStore) UpdateScript(id string, name, label, scriptType, tsCode *string) (*db.Script, error) {
	if m.updateScriptFn != nil { return m.updateScriptFn(id, name, label, scriptType, tsCode) }
	return nil, nil
}
func (m *mockStore) DeleteScript(id string) error {
	if m.deleteScriptFn != nil { return m.deleteScriptFn(id) }
	return nil
}
func (m *mockStore) ListBreakpoints(scriptID string) ([]db.Breakpoint, error) {
	if m.listBpFn != nil { return m.listBpFn(scriptID) }
	return nil, nil
}
func (m *mockStore) SetBreakpoint(scriptID string, line int, enabled bool) error {
	if m.setBpFn != nil { return m.setBpFn(scriptID, line, enabled) }
	return nil
}
func (m *mockStore) DeleteBreakpoint(scriptID string, line int) error {
	if m.deleteBpFn != nil { return m.deleteBpFn(scriptID, line) }
	return nil
}

type mockRunner struct {
	runFn   func(tsCode string, resolver runtime.ScriptResolver, timeoutMs int64) runtime.RunResult
	debugFn func(tsCode string, resolver runtime.ScriptResolver, bps []runtime.BpLine, skip int, timeoutMs int64) runtime.RunResult
}

func (m *mockRunner) Run(tsCode string, resolver runtime.ScriptResolver, timeoutMs int64) runtime.RunResult {
	if m.runFn != nil { return m.runFn(tsCode, resolver, timeoutMs) }
	return runtime.RunResult{Output: "ok"}
}
func (m *mockRunner) Debug(tsCode string, resolver runtime.ScriptResolver, bps []runtime.BpLine, skip int, timeoutMs int64) runtime.RunResult {
	if m.debugFn != nil { return m.debugFn(tsCode, resolver, bps, skip, timeoutMs) }
	return runtime.RunResult{Output: "debug ok"}
}

func adminCtx() context.Context {
	return auth.WithUserContext(context.Background(), "001b1", "001a1", "admin", "admin")
}

func userCtx() context.Context {
	return auth.WithUserContext(context.Background(), "001b2", "001a2", "acme", "user")
}

func TestListScriptsUser(t *testing.T) {
	s := &mockStore{
		listScriptsFn: func(tenantID string) ([]db.ScriptSummary, error) {
			if tenantID == "001a2" {
				return []db.ScriptSummary{{ID: "001c1", Name: "s1"}}, nil
			}
			return nil, nil
		},
	}
	h := &Handler{Store: s}
	req := httptest.NewRequest("GET", "/api/scripts", nil).WithContext(userCtx())
	w := httptest.NewRecorder()
	h.ListScripts(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp []db.ScriptSummary
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 || resp[0].Name != "s1" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestListScriptsAdmin(t *testing.T) {
	s := &mockStore{
		listAllScriptsFn: func() ([]db.ScriptSummary, error) {
			return []db.ScriptSummary{{ID: "001c1", Name: "a1"}, {ID: "001c2", Name: "a2"}}, nil
		},
	}
	h := &Handler{Store: s}
	req := httptest.NewRequest("GET", "/api/scripts", nil).WithContext(adminCtx())
	w := httptest.NewRecorder()
	h.ListScripts(w, req)
	var resp []db.ScriptSummary
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 scripts, got %d", len(resp))
	}
}

func TestCreateScript(t *testing.T) {
	s := &mockStore{
		createScriptFn: func(tenantID, name, label, scriptType, tsCode string) (*db.Script, error) {
			return &db.Script{ID: "001c1", TenantID: tenantID, Name: name, Label: label}, nil
		},
	}
	h := &Handler{Store: s}
	body := `{"name":"test","label":"Test","type":"ts","tsCode":"1"}`
	req := httptest.NewRequest("POST", "/api/scripts", strings.NewReader(body)).WithContext(userCtx())
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateScript(w, req)
	if w.Code != 201 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestTenantIsolationGet(t *testing.T) {
	s := &mockStore{
		getScriptFn: func(id string) (*db.Script, error) {
			return &db.Script{ID: id, TenantID: "001a9", Name: "other"}, nil
		},
	}
	h := &Handler{Store: s}
	req := httptest.NewRequest("GET", "/api/scripts/001c9", nil).WithContext(userCtx())
	req.SetPathValue("id", "001c9")
	w := httptest.NewRecorder()
	h.GetScript(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404 for cross-tenant access, got %d", w.Code)
	}
}
