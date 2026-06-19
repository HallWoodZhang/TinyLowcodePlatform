package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/config"
	"tiny-lowcode-platform/core/db/sqlite"
	"tiny-lowcode-platform/core/handler/adminpkg"
	"tiny-lowcode-platform/core/logger"
)

func main() {
	home := os.Getenv("LOWCODE_HOME")
	if home == "" {
		home = "."
	}
	cfgPath := filepath.Join(home, "cmd/admin-server/conf/config.json")
	logDir := filepath.Join(home, "cmd/admin-server/logs")
	dbPath := filepath.Join(home, "scripts.db")

	cfg := config.Load(cfgPath, "ADMIN_HOST", "ADMIN_PORT", "127.0.0.1", "9723")

	logLevels := cfg.Log
	if logLevels == nil {
		logLevels = &config.LogConfig{}
	}
	logs, err := logger.New(logDir, logLevels.Debug, logLevels.Access, logLevels.Panic)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logs.DebugL.Info(context.Background(), "admin-server starting, home=%s", home)

	store, err := sqlite.New(dbPath)
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to open database: %v", err)
		log.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	secret := cfg.JWTSecret
	if len(secret) == 0 {
		log.Fatal("jwt_secret must be configured (use same secret as auth-server)")
	}

	h := &adminpkg.AdminHandler{Store: store}

	mux := http.NewServeMux()

	authMW := auth.AuthMiddleware(secret)
	adminMW := auth.AdminMiddleware()

	// tenant CRUD
	mux.Handle("GET /api/admin/tenants", authMW(adminMW(http.HandlerFunc(h.ListTenants))))
	mux.Handle("POST /api/admin/tenants", authMW(adminMW(http.HandlerFunc(h.CreateTenant))))
	mux.Handle("GET /api/admin/tenants/{id}", authMW(adminMW(http.HandlerFunc(h.GetTenant))))
	mux.Handle("DELETE /api/admin/tenants/{id}", authMW(adminMW(http.HandlerFunc(h.DeleteTenant))))

	// user CRUD under tenant
	mux.Handle("GET /api/admin/tenants/{id}/users", authMW(adminMW(http.HandlerFunc(h.ListUsers))))
	mux.Handle("POST /api/admin/tenants/{id}/users", authMW(adminMW(http.HandlerFunc(h.CreateUser))))
	mux.Handle("DELETE /api/admin/tenants/{id}/users/{uid}", authMW(adminMW(http.HandlerFunc(h.DeleteUser))))

	// backdoor
	mux.Handle("GET /api/admin/users", authMW(adminMW(http.HandlerFunc(h.ListAllUsers))))
	mux.Handle("PUT /api/admin/users/{uid}/password", authMW(adminMW(http.HandlerFunc(h.UpdateUserPassword))))

	// betamap
	mux.Handle("GET /api/admin/tenants/{id}/betamap", authMW(adminMW(http.HandlerFunc(h.GetTenantBetamap))))
	mux.Handle("PUT /api/admin/tenants/{id}/betamap", authMW(adminMW(http.HandlerFunc(h.UpdateTenantBetamap))))

	var srv http.Handler = mux
	srv = logger.AccessLog(logs.AccessL)(srv)
	srv = logger.Recovery(logs.PanicL)(srv)
	srv = logger.TraceMiddleware(logs.DebugL)(srv)

	addr := cfg.Address()
	logs.DebugL.Info(context.Background(), "admin-server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
