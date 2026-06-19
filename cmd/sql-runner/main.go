package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/config"
	"tiny-lowcode-platform/core/db/sqlite"
	"tiny-lowcode-platform/core/handler/sqlpkg"
	"tiny-lowcode-platform/core/logger"
	"tiny-lowcode-platform/core/sqlstore"
	"tiny-lowcode-platform/core/validator"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	home := os.Getenv("LOWCODE_HOME")
	if home == "" {
		home = "."
	}
	cfgPath := filepath.Join(home, "cmd/sql-runner/conf/config.json")
	logDir := filepath.Join(home, "cmd/sql-runner/logs")
	dbPath := filepath.Join(home, "scripts.db")

	cfg := config.Load(cfgPath, "SQL_HOST", "SQL_PORT", "127.0.0.1", "9721")

	logLevels := cfg.Log
	if logLevels == nil {
		logLevels = &config.LogConfig{}
	}
	logs, err := logger.New(logDir, logLevels.Debug, logLevels.Access, logLevels.Panic)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logs.DebugL.Info(context.Background(), "sql-runner starting, home=%s", home)

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

	sqlH, err := sqlstore.New("sqlite", dbPath)
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to open SQL store: %v", err)
		log.Fatalf("failed to open SQL store: %v", err)
	}
	sqlHandler := sqlpkg.New(sqlH)

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to setup static files: %v", err)
		log.Fatalf("failed to setup static files: %v", err)
	}

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/sql-runner/ui/index.html", http.StatusFound)
	})

	mux.HandleFunc("GET /sql-runner/ui/index.html", func(w http.ResponseWriter, r *http.Request) {
		data, _ := fs.ReadFile(staticFS, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /sql-runner/ui/", http.StripPrefix("/sql-runner/ui", fileServer))

	blacklist := auth.NewBlacklist(cfg.RedisAddr, "", 0)
	authMW := auth.AuthMiddleware(secret, blacklist)
	adminMW := auth.AdminMiddleware()
	bpMW := auth.BetamapMiddleware(store, "sql_runner")

	mux.Handle("GET /api/sql/tables", authMW(adminMW(bpMW(http.HandlerFunc(sqlHandler.ListTables)))))
	mux.Handle("POST /api/sql/run", authMW(adminMW(bpMW(validator.Middleware(validator.RunSQLSchema)(http.HandlerFunc(sqlHandler.RunSQL))))))

	var srv http.Handler = mux
	srv = logger.AccessLog(logs.AccessL)(srv)
	srv = logger.Recovery(logs.PanicL)(srv)
	srv = logger.TraceMiddleware(logs.DebugL)(srv)

	addr := cfg.Address()
	logs.DebugL.Info(context.Background(), "server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
