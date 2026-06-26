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
	"tiny-lowcode-platform/core/handler"
	"tiny-lowcode-platform/core/logger"
	"tiny-lowcode-platform/core/runtime"
	"tiny-lowcode-platform/core/validator"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	home := os.Getenv("LOWCODE_HOME")
	if home == "" {
		home = "."
	}
	cfgPath := filepath.Join(home, "cmd/ts-quickjs/conf/config.json")
	logDir := filepath.Join(home, "cmd/ts-quickjs/logs")
	dbPath := filepath.Join(home, "scripts.db")

	cfg := config.Load(cfgPath, "TS_HOST", "TS_PORT", "127.0.0.1", "9720")

	logLevels := cfg.Log
	if logLevels == nil {
		logLevels = &config.LogConfig{}
	}
	logs, err := logger.New(logDir, logLevels.Debug, logLevels.Access, logLevels.Panic)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logs.DebugL.Info(context.Background(), "ts-runner starting, home=%s", home)

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

	var runner runtime.Runner
	runner = newEngine(cfg.Engine)

	h := &handler.Handler{
		Store:  store,
		Runner: runner,
	}

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to setup static files: %v", err)
		log.Fatalf("failed to setup static files: %v", err)
	}

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ts-quickjs/ui/index.html", http.StatusFound)
	})

	mux.HandleFunc("GET /ts-quickjs/ui/index.html", func(w http.ResponseWriter, r *http.Request) {
		data, _ := fs.ReadFile(staticFS, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /ts-quickjs/ui/", http.StripPrefix("/ts-quickjs/ui", fileServer))

	blacklist := auth.NewBlacklist(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	authMW := auth.AuthMiddleware(secret, blacklist)
	bpMW := auth.BetamapMiddleware(store, "script_debug")

	mux.Handle("GET /api/scripts", authMW(http.HandlerFunc(h.ListScripts)))
	mux.Handle("POST /api/scripts", authMW(validator.Middleware(validator.CreateScriptSchema)(http.HandlerFunc(h.CreateScript))))
	mux.Handle("GET /api/scripts/{id}", authMW(http.HandlerFunc(h.GetScript)))
	mux.Handle("PUT /api/scripts/{id}", authMW(validator.Middleware(validator.UpdateScriptSchema)(http.HandlerFunc(h.UpdateScript))))
	mux.Handle("DELETE /api/scripts/{id}", authMW(http.HandlerFunc(h.DeleteScript)))
	mux.Handle("POST /api/scripts/{id}/run", authMW(http.HandlerFunc(h.RunScript)))
	mux.Handle("POST /api/scripts/{id}/debug", authMW(bpMW(validator.Middleware(validator.DebugScriptSchema)(http.HandlerFunc(h.DebugScript)))))
	mux.Handle("GET /api/scripts/{id}/breakpoints", authMW(http.HandlerFunc(h.ListBreakpoints)))
	mux.Handle("POST /api/scripts/{id}/breakpoints", authMW(validator.Middleware(validator.SetBreakpointSchema)(http.HandlerFunc(h.SetBreakpoint))))
	mux.Handle("DELETE /api/scripts/{id}/breakpoints/{line}", authMW(http.HandlerFunc(h.DeleteBreakpoint)))

	var srv http.Handler = mux
	srv = logger.CORSMiddleware()(srv)
	srv = logger.AccessLog(logs.AccessL)(srv)
	srv = logger.Recovery(logs.PanicL)(srv)
	srv = logger.TraceMiddleware(logs.DebugL)(srv)

	addr := cfg.Address()
	logs.DebugL.Info(context.Background(), "server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
