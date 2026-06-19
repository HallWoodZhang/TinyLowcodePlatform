package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tiny-lowcode-platform/core/db"
)

func TestAuthMiddlewareHeader(t *testing.T) {
	secret := NewSecret()
	token, _ := Sign(Claims{Sub: "u1", TID: "t1", TN: "admin", Role: "user", Exp: time.Now().Add(time.Hour).Unix(), JTI: "j1"}, secret)

	handler := AuthMiddleware(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		json.NewEncoder(w).Encode(map[string]string{
			"userID":   UserID(ctx),
			"tenantID": TenantID(ctx),
			"role":     Role(ctx),
		})
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["userID"] != "u1" || body["tenantID"] != "t1" || body["role"] != "user" {
		t.Errorf("body = %+v", body)
	}
}

func TestAuthMiddlewareCookie(t *testing.T) {
	secret := NewSecret()
	token, _ := Sign(Claims{Sub: "u2", TID: "t2", TN: "acme", Role: "tenant_admin", Exp: time.Now().Add(time.Hour).Unix(), JTI: "j2"}, secret)

	handler := AuthMiddleware(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

func TestAuthMiddlewareNoToken(t *testing.T) {
	secret := NewSecret()
	handler := AuthMiddleware(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	secret := NewSecret()
	token, _ := Sign(Claims{Sub: "u", TID: "t", TN: "x", Role: "user", Exp: time.Now().Add(-time.Hour).Unix(), JTI: "j"}, secret)

	handler := AuthMiddleware(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddlewareBadToken(t *testing.T) {
	secret := NewSecret()
	handler := AuthMiddleware(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer garbage.token.here")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401, got %d: %s", w.Code, w.Code, w.Body.String())
	}
}

func TestAdminMiddlewareAllow(t *testing.T) {
	handler := AdminMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAdminMiddlewareDeny(t *testing.T) {
	handler := AdminMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), CtxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

type mockTenantStore struct {
	getFn func(id string) (*db.Tenant, error)
}

func (m *mockTenantStore) CreateTenant(name, label, betamap string) (*db.Tenant, error) { return nil, nil }
func (m *mockTenantStore) GetTenant(id string) (*db.Tenant, error) {
	if m.getFn != nil {
		return m.getFn(id)
	}
	return nil, nil
}
func (m *mockTenantStore) GetTenantByName(name string) (*db.Tenant, error)                     { return nil, nil }
func (m *mockTenantStore) ListTenants() ([]db.Tenant, error)                                     { return nil, nil }
func (m *mockTenantStore) UpdateTenant(id string, name, label, betamap *string) (*db.Tenant, error) { return nil, nil }
func (m *mockTenantStore) DeleteTenant(id string) error                                          { return nil }

func TestBetamapMiddlewareEnabled(t *testing.T) {
	store := &mockTenantStore{
		getFn: func(id string) (*db.Tenant, error) {
			return &db.Tenant{Betamap: `{"script_debug":true}`}, nil
		},
	}

	handler := BetamapMiddleware(store, "script_debug")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/debug", nil)
	ctx := context.WithValue(req.Context(), CtxTenantID, "t1")
	ctx = context.WithValue(ctx, CtxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
}

func TestBetamapMiddlewareDisabled(t *testing.T) {
	store := &mockTenantStore{
		getFn: func(id string) (*db.Tenant, error) {
			return &db.Tenant{Betamap: `{"script_debug":false}`}, nil
		},
	}

	handler := BetamapMiddleware(store, "script_debug")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/debug", nil)
	ctx := context.WithValue(req.Context(), CtxTenantID, "t1")
	ctx = context.WithValue(ctx, CtxRole, "user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("status = %d, want 403: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "feature disabled") {
		t.Errorf("body = %s", w.Body.String())
	}
}

func TestBetamapMiddlewareAdminBypass(t *testing.T) {
	store := &mockTenantStore{
		getFn: func(id string) (*db.Tenant, error) {
			return &db.Tenant{Betamap: `{"script_debug":false}`}, nil
		},
	}

	handler := BetamapMiddleware(store, "script_debug")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/debug", nil)
	ctx := context.WithValue(req.Context(), CtxTenantID, "t1")
	ctx = context.WithValue(ctx, CtxRole, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("admin should bypass betamap, status = %d: %s", w.Code, w.Body.String())
	}
}
