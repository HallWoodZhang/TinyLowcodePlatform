package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/config"
	"tiny-lowcode-platform/core/db/sqlite"
	"tiny-lowcode-platform/core/logger"
)

var (
	Version   = "v2.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	home := os.Getenv("LOWCODE_HOME")
	if home == "" {
		home = "."
	}
	cfgPath := filepath.Join(home, "cmd/bff-server/conf/config.json")
	logDir := filepath.Join(home, "cmd/bff-server/logs")
	dbPath := filepath.Join(home, "scripts.db")

	cfg := config.Load(cfgPath, "BFF_HOST", "BFF_PORT", "127.0.0.1", "9724")

	logLevels := cfg.Log
	if logLevels == nil {
		logLevels = &config.LogConfig{}
	}
	logs, err := logger.New(logDir, logLevels.Debug, logLevels.Access, logLevels.Panic)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logs.DebugL.Info(context.Background(), "bff-server starting v%s, home=%s", Version, home)

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

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("failed to setup static files: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/bff/ui/index.html", http.StatusFound)
	})
	mux.HandleFunc("GET /bff/ui/index.html", func(w http.ResponseWriter, r *http.Request) {
		data, _ := fs.ReadFile(staticFS, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /bff/ui/", http.StripPrefix("/bff/ui", fileServer))

	// public
	mux.HandleFunc("GET /api/bff/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version":   Version,
			"codename":  "Gateway",
			"buildTime": BuildTime,
			"gitCommit": GitCommit,
		})
	})

	// protected
	authMW := auth.AuthMiddleware(secret)
	mux.Handle("GET /api/bff/entries", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid := auth.TenantID(r.Context())
		role := auth.Role(r.Context())
		tenant, err := store.GetTenant(tid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "tenant not found"})
			return
		}

		var betamap map[string]any
		json.Unmarshal([]byte(tenant.Betamap), &betamap)

		isAdmin := role == "admin"
		entries := []map[string]any{
			{"id": "scripts", "label": "查看脚本", "url": "http://127.0.0.1:9720/ts-quickjs/ui/index.html", "enabled": isAdmin || betamap["script_editor"] == true},
			{"id": "sql_runner", "label": "SQL查询", "url": "http://127.0.0.1:9721/sql-runner/ui/index.html", "enabled": isAdmin || betamap["sql_runner"] == true},
			{"id": "admin", "label": "管理面板", "url": "http://127.0.0.1:9723/admin/ui/index.html", "enabled": isAdmin || betamap["admin_panel"] == true},
		}

		writeJSON(w, http.StatusOK, entries)
	})))

	var srv http.Handler = mux
	srv = logger.AccessLog(logs.AccessL)(srv)
	srv = logger.Recovery(logs.PanicL)(srv)
	srv = logger.TraceMiddleware(logs.DebugL)(srv)

	addr := cfg.Address()
	logs.DebugL.Info(context.Background(), "bff-server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
