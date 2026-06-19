package adminpkg

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"toy-platform/core/auth"
	"toy-platform/core/db"
)

type adminMockStore struct {
	listTenantsFn     func() ([]db.Tenant, error)
	createTenantFn    func(name, label, betamap string) (*db.Tenant, error)
	getTenantFn       func(id string) (*db.Tenant, error)
	deleteTenantFn    func(id string) error
	listUsersByTenantFn func(tenantID string) ([]db.User, error)
	createUserFn      func(tenantID, username, passwordHash, role string) (*db.User, error)
	deleteUserFn      func(id string) error
	listAllUsersFn    func() ([]db.User, error)
	updatePasswordFn  func(id, passwordHash string) error
	updateTenantFn    func(id string, name, label, betamap *string) (*db.Tenant, error)
	getScriptFn       func(id string) (*db.Script, error)
}

func (m *adminMockStore) CreateTenant(name, label, betamap string) (*db.Tenant, error) {
	if m.createTenantFn != nil { return m.createTenantFn(name, label, betamap) }
	return &db.Tenant{ID: "001a1", Name: name, Label: label, Betamap: betamap}, nil
}
func (m *adminMockStore) GetTenant(id string) (*db.Tenant, error) {
	if m.getTenantFn != nil { return m.getTenantFn(id) }
	return nil, fmt.Errorf("not found")
}
func (m *adminMockStore) GetTenantByName(name string) (*db.Tenant, error) { return nil, nil }
func (m *adminMockStore) ListTenants() ([]db.Tenant, error) {
	if m.listTenantsFn != nil { return m.listTenantsFn() }
	return nil, nil
}
func (m *adminMockStore) UpdateTenant(id string, name, label, betamap *string) (*db.Tenant, error) {
	if m.updateTenantFn != nil { return m.updateTenantFn(id, name, label, betamap) }
	return nil, nil
}
func (m *adminMockStore) DeleteTenant(id string) error {
	if m.deleteTenantFn != nil { return m.deleteTenantFn(id) }
	return nil
}
func (m *adminMockStore) CreateUser(tenantID, username, passwordHash, role string) (*db.User, error) {
	if m.createUserFn != nil { return m.createUserFn(tenantID, username, passwordHash, role) }
	return &db.User{ID: "001b1", TenantID: tenantID, Username: username, Role: role}, nil
}
func (m *adminMockStore) GetUser(id string) (*db.User, error) { return nil, nil }
func (m *adminMockStore) GetUserByTenantAndUsername(tenantName, username string) (*db.User, error) { return nil, nil }
func (m *adminMockStore) ListUsersByTenant(tenantID string) ([]db.User, error) {
	if m.listUsersByTenantFn != nil { return m.listUsersByTenantFn(tenantID) }
	return nil, nil
}
func (m *adminMockStore) ListAllUsers() ([]db.User, error) {
	if m.listAllUsersFn != nil { return m.listAllUsersFn() }
	return nil, nil
}
func (m *adminMockStore) DeleteUser(id string) error {
	if m.deleteUserFn != nil { return m.deleteUserFn(id) }
	return nil
}
func (m *adminMockStore) UpdateUserPassword(id, passwordHash string) error {
	if m.updatePasswordFn != nil { return m.updatePasswordFn(id, passwordHash) }
	return nil
}
func (m *adminMockStore) ListScripts(tenantID string) ([]db.ScriptSummary, error) { return nil, nil }
func (m *adminMockStore) ListAllScripts() ([]db.ScriptSummary, error)             { return nil, nil }
func (m *adminMockStore) GetScript(id string) (*db.Script, error) {
	if m.getScriptFn != nil { return m.getScriptFn(id) }
	return nil, nil
}
func (m *adminMockStore) GetScriptByName(tenantID, name string) (*db.Script, error) { return nil, nil }
func (m *adminMockStore) CreateScript(tenantID, name, label, scriptType, tsCode string) (*db.Script, error) { return nil, nil }
func (m *adminMockStore) UpdateScript(id string, name, label, scriptType, tsCode *string) (*db.Script, error) { return nil, nil }
func (m *adminMockStore) DeleteScript(id string) error { return nil }
func (m *adminMockStore) ListBreakpoints(scriptID string) ([]db.Breakpoint, error) { return nil, nil }
func (m *adminMockStore) SetBreakpoint(scriptID string, line int, enabled bool) error { return nil }
func (m *adminMockStore) DeleteBreakpoint(scriptID string, line int) error { return nil }

func TestListTenants(t *testing.T) {
	s := &adminMockStore{
		listTenantsFn: func() ([]db.Tenant, error) {
			return []db.Tenant{{ID: "001a1", Name: "admin"}, {ID: "001a2", Name: "acme"}}, nil
		},
	}
	h := &AdminHandler{Store: s}
	req := httptest.NewRequest("GET", "/api/admin/tenants", nil)
	w := httptest.NewRecorder()
	h.ListTenants(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var tenants []db.Tenant
	json.NewDecoder(w.Body).Decode(&tenants)
	if len(tenants) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(tenants))
	}
}

func TestCreateTenant(t *testing.T) {
	s := &adminMockStore{}
	h := &AdminHandler{Store: s}
	body := `{"name":"acme","label":"ACME Corp"}`
	req := httptest.NewRequest("POST", "/api/admin/tenants", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateTenant(w, req)
	if w.Code != 201 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateUser(t *testing.T) {
	s := &adminMockStore{}
	h := &AdminHandler{Store: s}
	body := `{"username":"dev1","password":"pass1","role":"user"}`
	req := httptest.NewRequest("POST", "/api/admin/tenants/001a1/users", strings.NewReader(body))
	req.SetPathValue("id", "001a1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateUser(w, req)
	if w.Code != 201 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestListAllUsers(t *testing.T) {
	hash, _ := auth.HashPassword("pass1")
	s := &adminMockStore{
		listAllUsersFn: func() ([]db.User, error) {
			return []db.User{
				{ID: "001b1", TenantID: "001a1", Username: "admin", PasswordHash: hash, Role: "admin"},
				{ID: "001b2", TenantID: "001a2", Username: "dev1", PasswordHash: hash, Role: "user"},
			}, nil
		},
	}
	h := &AdminHandler{Store: s}
	req := httptest.NewRequest("GET", "/api/admin/users", nil)
	w := httptest.NewRecorder()
	h.ListAllUsers(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var users []db.User
	json.NewDecoder(w.Body).Decode(&users)
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUpdatePassword(t *testing.T) {
	s := &adminMockStore{}
	h := &AdminHandler{Store: s}
	body := `{"password":"newpass"}`
	req := httptest.NewRequest("PUT", "/api/admin/users/001b1/password", strings.NewReader(body))
	req.SetPathValue("uid", "001b1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateUserPassword(w, req)
	if w.Code != 204 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTenantBetamap(t *testing.T) {
	s := &adminMockStore{
		getTenantFn: func(id string) (*db.Tenant, error) {
			return &db.Tenant{ID: id, Betamap: `{"script_debug":true}`}, nil
		},
	}
	h := &AdminHandler{Store: s}
	req := httptest.NewRequest("GET", "/api/admin/tenants/001a1/betamap", nil)
	req.SetPathValue("id", "001a1")
	w := httptest.NewRecorder()
	h.GetTenantBetamap(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "script_debug") {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestUpdateTenantBetamap(t *testing.T) {
	s := &adminMockStore{
		updateTenantFn: func(id string, name, label, betamap *string) (*db.Tenant, error) {
			return &db.Tenant{ID: id, Betamap: *betamap}, nil
		},
	}
	h := &AdminHandler{Store: s}
	body := `{"script_debug":false}`
	req := httptest.NewRequest("PUT", "/api/admin/tenants/001a1/betamap", strings.NewReader(body))
	req.SetPathValue("id", "001a1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.UpdateTenantBetamap(w, req)
	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
}
