package sqlite

import (
	"testing"

	"tiny-lowcode-platform/core/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New("file:" + t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMigrateCreatesTables(t *testing.T) {
	s := newTestStore(t)

	var tables []string
	rows, err := s.db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	found := make(map[string]bool)
	for _, name := range tables {
		found[name] = true
	}
	for _, want := range []string{"tenants", "users", "scripts", "breakpoints"} {
		if !found[want] {
			t.Errorf("missing table: %s (got: %v)", want, tables)
		}
	}
}

func TestSeedAdminData(t *testing.T) {
	s := newTestStore(t)

	tenant, err := s.GetTenantByName("admin")
	if err != nil {
		t.Fatalf("admin tenant not found: %v", err)
	}
	if tenant.Name != "admin" {
		t.Errorf("tenant name = %s, want admin", tenant.Name)
	}
	if len(tenant.ID) != 24 || tenant.ID[:4] != "001a" {
		t.Errorf("bad tenant id: %s", tenant.ID)
	}

	users, err := s.ListUsersByTenant(tenant.ID)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) == 0 {
		t.Fatal("no users in admin tenant")
	}
	if users[0].Username != "admin" {
		t.Errorf("username = %s, want admin", users[0].Username)
	}
	if users[0].Role != "admin" {
		t.Errorf("role = %s, want admin", users[0].Role)
	}
}

func TestSeedIdempotent(t *testing.T) {
	s := newTestStore(t)
	// second seed should not fail
	if err := s.seed(); err != nil {
		t.Fatalf("second seed failed: %v", err)
	}
	tenants, _ := s.ListTenants()
	if len(tenants) != 1 {
		t.Errorf("expected 1 tenant after double seed, got %d", len(tenants))
	}
}

func TestCreateTenantCreatesMasterUser(t *testing.T) {
	s := newTestStore(t)

	tenant, err := s.CreateTenant("acme", "ACME Corp", `{"script_debug":true}`)
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if tenant.Name != "acme" {
		t.Errorf("name = %s, want acme", tenant.Name)
	}
	if tenant.Betamap != `{"script_debug":true}` {
		t.Errorf("betamap = %s", tenant.Betamap)
	}

	users, err := s.ListUsersByTenant(tenant.ID)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 master user, got %d", len(users))
	}
	if users[0].Username != "acme" {
		t.Errorf("master username = %s, want acme", users[0].Username)
	}
	if users[0].Role != "tenant_admin" {
		t.Errorf("master role = %s, want tenant_admin", users[0].Role)
	}
}

func TestScriptCRUD(t *testing.T) {
	s := newTestStore(t)
	admin, _ := s.GetTenantByName("admin")

	sc, err := s.CreateScript(admin.ID, "hello", "Hello", "ts", "console.log(1)")
	if err != nil {
		t.Fatalf("CreateScript: %v", err)
	}
	if sc.TenantID != admin.ID {
		t.Errorf("tenant_id = %s, want %s", sc.TenantID, admin.ID)
	}

	got, err := s.GetScript(sc.ID)
	if err != nil {
		t.Fatalf("GetScript: %v", err)
	}
	if got.Name != "hello" {
		t.Errorf("name = %s, want hello", got.Name)
	}

	list, err := s.ListScripts(admin.ID)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 script, got %d", len(list))
	}

	newName := "hello2"
	updated, err := s.UpdateScript(sc.ID, &newName, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateScript: %v", err)
	}
	if updated.Name != "hello2" {
		t.Errorf("updated name = %s", updated.Name)
	}

	if err := s.DeleteScript(sc.ID); err != nil {
		t.Fatalf("DeleteScript: %v", err)
	}
	list, _ = s.ListScripts(admin.ID)
	if len(list) != 0 {
		t.Errorf("expected 0 scripts after delete, got %d", len(list))
	}
}

func TestTenantIsolation(t *testing.T) {
	s := newTestStore(t)
	admin, _ := s.GetTenantByName("admin")
	acme, _ := s.CreateTenant("acme", "ACME", "{}")

	s.CreateScript(admin.ID, "admin_script", "A", "ts", "1")
	s.CreateScript(acme.ID, "acme_script", "A", "ts", "1")

	adminList, _ := s.ListScripts(admin.ID)
	if len(adminList) != 1 || adminList[0].Name != "admin_script" {
		t.Errorf("admin scripts: %+v", adminList)
	}

	acmeList, _ := s.ListScripts(acme.ID)
	if len(acmeList) != 1 || acmeList[0].Name != "acme_script" {
		t.Errorf("acme scripts: %+v", acmeList)
	}

	allList, _ := s.ListAllScripts()
	if len(allList) != 2 {
		t.Errorf("expected 2 scripts total, got %d", len(allList))
	}
}

func TestBreakpointCRUD(t *testing.T) {
	s := newTestStore(t)
	admin, _ := s.GetTenantByName("admin")
	sc, _ := s.CreateScript(admin.ID, "bp_test", "BP", "ts", "1")

	if err := s.SetBreakpoint(sc.ID, 10, true); err != nil {
		t.Fatalf("SetBreakpoint: %v", err)
	}
	bps, err := s.ListBreakpoints(sc.ID)
	if err != nil {
		t.Fatalf("ListBreakpoints: %v", err)
	}
	if len(bps) != 1 || bps[0].Line != 10 || !bps[0].Enabled {
		t.Errorf("bps = %+v", bps)
	}

	if err := s.DeleteBreakpoint(sc.ID, 10); err != nil {
		t.Fatalf("DeleteBreakpoint: %v", err)
	}
	bps, _ = s.ListBreakpoints(sc.ID)
	if len(bps) != 0 {
		t.Errorf("expected 0 breakpoints, got %d", len(bps))
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var s any = &Store{}
	if _, ok := s.(db.TenantStore); !ok {
		t.Error("Store does not implement TenantStore")
	}
	if _, ok := s.(db.UserStore); !ok {
		t.Error("Store does not implement UserStore")
	}
	if _, ok := s.(db.ScriptStore); !ok {
		t.Error("Store does not implement ScriptStore")
	}
	if _, ok := s.(db.BreakpointStore); !ok {
		t.Error("Store does not implement BreakpointStore")
	}
}

func TestCascadeDeleteScriptRemovesBreakpoints(t *testing.T) {
	s := newTestStore(t)
	admin, _ := s.GetTenantByName("admin")
	sc, _ := s.CreateScript(admin.ID, "cascade_test", "CT", "ts", "1")
	s.SetBreakpoint(sc.ID, 5, true)
	s.SetBreakpoint(sc.ID, 10, true)

	bps, _ := s.ListBreakpoints(sc.ID)
	if len(bps) != 2 {
		t.Fatalf("expected 2 breakpoints, got %d", len(bps))
	}

	s.DeleteScript(sc.ID)

	bps, _ = s.ListBreakpoints(sc.ID)
	if len(bps) != 0 {
		t.Errorf("expected 0 breakpoints after cascade, got %d", len(bps))
	}
}

func TestGetScriptNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetScript("001c0000000000000000bad")
	if err == nil {
		t.Error("expected error for nonexistent script")
	}
}

func TestGetTenantNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetTenant("001a0000000000000000bad")
	if err == nil {
		t.Error("expected error for nonexistent tenant")
	}
}

func TestCreateDuplicateScriptName(t *testing.T) {
	s := newTestStore(t)
	admin, _ := s.GetTenantByName("admin")
	_, err := s.CreateScript(admin.ID, "dup_test", "D", "ts", "1")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err = s.CreateScript(admin.ID, "dup_test", "D2", "ts", "2")
	if err == nil {
		t.Error("expected error for duplicate script name in same tenant")
	}
}

func TestListScriptsReturnsEmptySlice(t *testing.T) {
	s := newTestStore(t)
	// create a tenant with no scripts
	acme, _ := s.CreateTenant("empty_tenant", "Empty", "{}")
	scripts, err := s.ListScripts(acme.ID)
	if err != nil {
		t.Fatalf("ListScripts: %v", err)
	}
	if scripts == nil {
		t.Error("expected empty slice, got nil")
	}
}

func TestListBreakpointsReturnsEmptySlice(t *testing.T) {
	s := newTestStore(t)
	admin, _ := s.GetTenantByName("admin")
	sc, _ := s.CreateScript(admin.ID, "no_bp", "NB", "ts", "1")
	bps, err := s.ListBreakpoints(sc.ID)
	if err != nil {
		t.Fatalf("ListBreakpoints: %v", err)
	}
	if bps == nil {
		t.Error("expected empty slice, got nil")
	}
}
