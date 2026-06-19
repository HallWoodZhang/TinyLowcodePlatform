package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/db/sqlite"
	"tiny-lowcode-platform/core/handler/adminpkg"
	"tiny-lowcode-platform/core/handler/authpkg"
)

// testEnv holds all test dependencies.
type testEnv struct {
	Store       *sqlite.Store
	Secret      []byte
	AuthHandler *authpkg.AuthHandler
	AdminH      *adminpkg.AdminHandler
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	store, err := sqlite.New("file:" + t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	secret := auth.NewSecret()

	return &testEnv{
		Store:  store,
		Secret: secret,
		AuthHandler: &authpkg.AuthHandler{
			Store:       store,
			TokenSecret: secret,
			TokenExpire: time.Hour,
		},
		AdminH: &adminpkg.AdminHandler{Store: store},
	}
}

func (e *testEnv) login(t *testing.T, tenant, username, password string) (int, string) {
	t.Helper()
	body := fmt.Sprintf(`{"tenant":"%s","username":"%s","password":"%s"}`, tenant, username, password)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.AuthHandler.Login(w, req)

	if w.Code != 200 {
		return w.Code, ""
	}

	var resp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	token := ""
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" {
			token = c.Value
		}
	}
	if token == "" {
		token = resp.Token
	}
	return w.Code, token
}

func TestSeedData(t *testing.T) {
	env := newTestEnv(t)

	tenants, err := env.Store.ListTenants()
	if err != nil {
		t.Fatalf("ListTenants: %v", err)
	}
	if len(tenants) == 0 {
		t.Fatal("no tenants seeded")
	}
	if tenants[0].Name != "admin" {
		t.Errorf("first tenant name = %s, want admin", tenants[0].Name)
	}

	users, err := env.Store.ListUsersByTenant(tenants[0].ID)
	if err != nil {
		t.Fatalf("ListUsersByTenant: %v", err)
	}
	if len(users) == 0 || users[0].Username != "admin" {
		t.Errorf("admin user not found, users=%+v", users)
	}
}

func TestFullAuthFlow(t *testing.T) {
	env := newTestEnv(t)

	// login with default admin (seed uses placeholder hash; override with known)
	hash, _ := auth.HashPassword("admin123")
	adminUser, _ := env.Store.GetUserByTenantAndUsername("admin", "admin")
	env.Store.UpdateUserPassword(adminUser.ID, hash)

	code, token := env.login(t, "admin", "admin", "admin123")
	if code != 200 {
		t.Fatalf("login status = %d", code)
	}
	if token == "" {
		t.Fatal("no token returned")
	}

	// verify claims
	claims, err := auth.Verify(token, env.Secret)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Role != "admin" {
		t.Errorf("role = %s, want admin", claims.Role)
	}
	if claims.TN != "admin" {
		t.Errorf("tenant = %s, want admin", claims.TN)
	}

	// access /me (through auth middleware)
	authMW := auth.AuthMiddleware(env.Secret, nil)
	meHandler := authMW(http.HandlerFunc(env.AuthHandler.Me))
	meReq := httptest.NewRequest("GET", "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meW := httptest.NewRecorder()
	meHandler.ServeHTTP(meW, meReq)
	if meW.Code != 200 {
		t.Fatalf("me status = %d: %s", meW.Code, meW.Body.String())
	}

	// access betamap (through auth middleware)
	bpHandler := authMW(http.HandlerFunc(env.AuthHandler.Betamap))
	bpReq := httptest.NewRequest("GET", "/api/auth/betamap", nil)
	bpReq.Header.Set("Authorization", "Bearer "+token)
	bpW := httptest.NewRecorder()
	bpHandler.ServeHTTP(bpW, bpReq)
	if bpW.Code != 200 {
		t.Fatalf("betamap status = %d", bpW.Code)
	}

	// refresh token
	refreshHandler := authMW(http.HandlerFunc(env.AuthHandler.Refresh))
	refReq := httptest.NewRequest("POST", "/api/auth/refresh", nil)
	refReq.Header.Set("Authorization", "Bearer "+token)
	refW := httptest.NewRecorder()
	refreshHandler.ServeHTTP(refW, refReq)
	if refW.Code != 200 {
		t.Fatalf("refresh status = %d: %s", refW.Code, refW.Body.String())
	}
}

func TestTenantIsolation(t *testing.T) {
	env := newTestEnv(t)

	// create two tenants
	acme, _ := env.Store.CreateTenant("acme", "ACME", `{"script_editor":true}`)
	xyz, _ := env.Store.CreateTenant("xyz", "XYZ", `{"script_editor":true}`)

	// create scripts for each
	env.Store.CreateScript(acme.ID, "acme_script", "A", "ts", "1")
	env.Store.CreateScript(xyz.ID, "xyz_script", "X", "ts", "1")

	// verify isolation
	acmeScripts, _ := env.Store.ListScripts(acme.ID)
	if len(acmeScripts) != 1 || acmeScripts[0].Name != "acme_script" {
		t.Errorf("acme scripts: %+v", acmeScripts)
	}

	xyzScripts, _ := env.Store.ListScripts(xyz.ID)
	if len(xyzScripts) != 1 || xyzScripts[0].Name != "xyz_script" {
		t.Errorf("xyz scripts: %+v", xyzScripts)
	}

	// admin sees all
	all, _ := env.Store.ListAllScripts()
	if len(all) != 2 {
		t.Errorf("all scripts count = %d, want 2", len(all))
	}
}

func TestBetamapControl(t *testing.T) {
	env := newTestEnv(t)

	// create tenant with script_debug disabled
	acme, _ := env.Store.CreateTenant("debugtest", "Debug Test", `{"script_debug":false,"script_editor":true}`)

	// create master user (tenant_admin)
	users, _ := env.Store.ListUsersByTenant(acme.ID)
	masterUser := users[0]

	// set real password for login
	hash, _ := auth.HashPassword("debugtest")
	env.Store.UpdateUserPassword(masterUser.ID, hash)

	// verify betamap
	tenant, _ := env.Store.GetTenant(acme.ID)
	if !strings.Contains(tenant.Betamap, `"script_debug":false`) {
		t.Errorf("betamap = %s", tenant.Betamap)
	}

	// update betamap to enable debug
	betamap := `{"script_debug":true,"script_editor":true}`
	_, err := env.Store.UpdateTenant(acme.ID, nil, nil, &betamap)
	if err != nil {
		t.Fatalf("UpdateTenant: %v", err)
	}

	tenant, _ = env.Store.GetTenant(acme.ID)
	if !strings.Contains(tenant.Betamap, `"script_debug":true`) {
		t.Errorf("betamap after update = %s", tenant.Betamap)
	}
}

func TestAdminBackdoor(t *testing.T) {
	env := newTestEnv(t)

	// create tenant + user
	acme, _ := env.Store.CreateTenant("backtest", "Backdoor Test", "{}")

	// set password
	users, _ := env.Store.ListUsersByTenant(acme.ID)
	hash, _ := auth.HashPassword("newpass")
	env.Store.UpdateUserPassword(users[0].ID, hash)

	// list all users - check backdoor returns all
	allUsers, _ := env.Store.ListAllUsers()
	if len(allUsers) < 2 {
		t.Errorf("expected at least 2 users (admin + acme master), got %d", len(allUsers))
	}

	// verify password_hash is included for backdoor audit
	found := false
	for _, u := range allUsers {
		if u.Username == "admin" && u.PasswordHash != "" {
			found = true
		}
	}
	if !found {
		t.Error("backdoor should return password_hash")
	}
}

func TestLoginWrongTenant(t *testing.T) {
	env := newTestEnv(t)

	code, _ := env.login(t, "nonexistent", "admin", "admin123")
	if code != 401 {
		t.Errorf("status = %d, want 401", code)
	}
}

func TestLogoutFlow(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	env.AuthHandler.Logout(w, req)

	if w.Code != 200 {
		t.Errorf("logout status = %d", w.Code)
	}

	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "token" && c.MaxAge < 0 {
			found = true
		}
	}
	if !found {
		t.Error("logout should clear token cookie")
	}
}
