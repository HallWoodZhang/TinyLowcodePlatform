package authpkg

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"toy-platform/core/auth"
	"toy-platform/core/db"
)

type authMockStore struct {
	getUserByTenantAndUsernameFn func(tenantName, username string) (*db.User, error)
	getUserFn                    func(id string) (*db.User, error)
	getTenantFn                  func(id string) (*db.Tenant, error)
	getTenantByNameFn            func(name string) (*db.Tenant, error)
}

func (m *authMockStore) CreateTenant(name, label, betamap string) (*db.Tenant, error) { return nil, nil }
func (m *authMockStore) GetTenant(id string) (*db.Tenant, error) {
	if m.getTenantFn != nil { return m.getTenantFn(id) }
	return nil, nil
}
func (m *authMockStore) GetTenantByName(name string) (*db.Tenant, error) {
	if m.getTenantByNameFn != nil { return m.getTenantByNameFn(name) }
	return nil, nil
}
func (m *authMockStore) ListTenants() ([]db.Tenant, error)                                         { return nil, nil }
func (m *authMockStore) UpdateTenant(id string, name, label, betamap *string) (*db.Tenant, error)  { return nil, nil }
func (m *authMockStore) DeleteTenant(id string) error                                              { return nil }
func (m *authMockStore) CreateUser(tenantID, username, passwordHash, role string) (*db.User, error) { return nil, nil }
func (m *authMockStore) GetUser(id string) (*db.User, error) {
	if m.getUserFn != nil { return m.getUserFn(id) }
	return nil, nil
}
func (m *authMockStore) GetUserByTenantAndUsername(tenantName, username string) (*db.User, error) {
	if m.getUserByTenantAndUsernameFn != nil { return m.getUserByTenantAndUsernameFn(tenantName, username) }
	return nil, fmt.Errorf("not found")
}
func (m *authMockStore) ListUsersByTenant(tenantID string) ([]db.User, error)                { return nil, nil }
func (m *authMockStore) ListAllUsers() ([]db.User, error)                                      { return nil, nil }
func (m *authMockStore) DeleteUser(id string) error                                           { return nil }
func (m *authMockStore) UpdateUserPassword(id, passwordHash string) error                     { return nil }
func (m *authMockStore) ListScripts(tenantID string) ([]db.ScriptSummary, error)              { return nil, nil }
func (m *authMockStore) ListAllScripts() ([]db.ScriptSummary, error)                          { return nil, nil }
func (m *authMockStore) GetScript(id string) (*db.Script, error)                              { return nil, nil }
func (m *authMockStore) GetScriptByName(tenantID, name string) (*db.Script, error)            { return nil, nil }
func (m *authMockStore) CreateScript(tenantID, name, label, scriptType, tsCode string) (*db.Script, error) { return nil, nil }
func (m *authMockStore) UpdateScript(id string, name, label, scriptType, tsCode *string) (*db.Script, error) { return nil, nil }
func (m *authMockStore) DeleteScript(id string) error                                         { return nil }
func (m *authMockStore) ListBreakpoints(scriptID string) ([]db.Breakpoint, error)             { return nil, nil }
func (m *authMockStore) SetBreakpoint(scriptID string, line int, enabled bool) error          { return nil }
func (m *authMockStore) DeleteBreakpoint(scriptID string, line int) error                     { return nil }

func newAuthHandler(store *authMockStore) *AuthHandler {
	hash, _ := auth.HashPassword("admin123")
	_ = hash
	return &AuthHandler{
		Store:       store,
		TokenSecret: auth.NewSecret(),
		TokenExpire: time.Hour,
	}
}

func TestLoginSuccess(t *testing.T) {
	hash, _ := auth.HashPassword("admin123")
	store := &authMockStore{
		getUserByTenantAndUsernameFn: func(tenantName, username string) (*db.User, error) {
			if tenantName == "admin" && username == "admin" {
				return &db.User{ID: "001b1", TenantID: "001a1", Username: "admin", PasswordHash: hash, Role: "admin"}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		getTenantByNameFn: func(name string) (*db.Tenant, error) {
			return &db.Tenant{ID: "001a1", Name: "admin"}, nil
		},
	}

	h := newAuthHandler(store)
	body := `{"tenant":"admin","username":"admin","password":"admin123"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var resp loginResp
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Token == "" {
		t.Error("token is empty")
	}
	if resp.User.Username != "admin" {
		t.Errorf("username = %s", resp.User.Username)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("admin123")
	store := &authMockStore{
		getUserByTenantAndUsernameFn: func(tenantName, username string) (*db.User, error) {
			return &db.User{ID: "001b1", TenantID: "001a1", Username: "admin", PasswordHash: hash, Role: "admin"}, nil
		},
	}

	h := newAuthHandler(store)
	body := `{"tenant":"admin","username":"admin","password":"wrong"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401: %s", w.Code, w.Body.String())
	}
}

func TestLoginMissingTenant(t *testing.T) {
	h := newAuthHandler(&authMockStore{})
	body := `{"tenant":"noexist","username":"admin","password":"x"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestLoginMissingField(t *testing.T) {
	h := newAuthHandler(&authMockStore{})
	body := `{"tenant":"admin"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestLogout(t *testing.T) {
	h := newAuthHandler(&authMockStore{})
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "token" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("token cookie not cleared")
	}
}

func TestMe(t *testing.T) {
	store := &authMockStore{
		getUserFn: func(id string) (*db.User, error) {
			return &db.User{ID: id, TenantID: "001a1", Username: "admin", Role: "admin"}, nil
		},
	}
	h := newAuthHandler(store)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	ctx := auth.WithUserContext(req.Context(), "001b1", "001a1", "admin", "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["username"] != "admin" {
		t.Errorf("username = %v", body["username"])
	}
}

func TestMeNoAuth(t *testing.T) {
	h := newAuthHandler(&authMockStore{})
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestBetamap(t *testing.T) {
	store := &authMockStore{
		getTenantFn: func(id string) (*db.Tenant, error) {
			return &db.Tenant{ID: id, Betamap: `{"script_editor":true}`}, nil
		},
	}
	h := newAuthHandler(store)
	req := httptest.NewRequest("GET", "/api/auth/betamap", nil)
	ctx := auth.WithUserContext(req.Context(), "u1", "001a1", "admin", "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Betamap(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "script_editor") {
		t.Errorf("body = %s", w.Body.String())
	}
}
