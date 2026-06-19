package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/config"
	"tiny-lowcode-platform/core/db/sqlite"
	"tiny-lowcode-platform/core/handler/authpkg"
	"tiny-lowcode-platform/core/logger"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	home := os.Getenv("LOWCODE_HOME")
	if home == "" {
		home = "."
	}
	cfgPath := filepath.Join(home, "cmd/auth-server/conf/config.json")
	logDir := filepath.Join(home, "cmd/auth-server/logs")
	dbPath := filepath.Join(home, "scripts.db")

	cfg := config.Load(cfgPath, "AUTH_HOST", "AUTH_PORT", "127.0.0.1", "9722")

	logLevels := cfg.Log
	if logLevels == nil {
		logLevels = &config.LogConfig{}
	}
	logs, err := logger.New(logDir, logLevels.Debug, logLevels.Access, logLevels.Panic)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logs.DebugL.Info(context.Background(), "auth-server starting, home=%s", home)

	store, err := sqlite.New(dbPath)
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to open database: %v", err)
		log.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	secret := cfg.JWTSecret
	if len(secret) == 0 {
		secret = auth.NewSecret()
	}

	expireHours := cfg.TokenExpireHours
	if expireHours <= 0 {
		expireHours = 1
	}

	h := &authpkg.AuthHandler{
		Store:       store,
		TokenSecret: secret,
		TokenExpire: time.Duration(expireHours) * time.Hour,
	}

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to setup static files: %v", err)
		log.Fatalf("failed to setup static files: %v", err)
	}

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/auth/ui/login.html", http.StatusFound)
	})

	mux.HandleFunc("GET /auth/ui/login.html", func(w http.ResponseWriter, r *http.Request) {
		data, _ := fs.ReadFile(staticFS, "login.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /auth/ui/", http.StripPrefix("/auth/ui", fileServer))

	// public routes
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)

	// protected routes
	authMW := auth.AuthMiddleware(secret)
	mux.Handle("GET /api/auth/me", authMW(http.HandlerFunc(h.Me)))
	mux.Handle("GET /api/auth/betamap", authMW(http.HandlerFunc(h.Betamap)))
	mux.Handle("POST /api/auth/refresh", authMW(http.HandlerFunc(h.Refresh)))

	var srv http.Handler = mux
	srv = logger.AccessLog(logs.AccessL)(srv)
	srv = logger.Recovery(logs.PanicL)(srv)
	srv = logger.TraceMiddleware(logs.DebugL)(srv)

	addr := cfg.Address()
	logs.DebugL.Info(context.Background(), "auth-server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
