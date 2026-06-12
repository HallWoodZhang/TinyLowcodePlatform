package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"toy-platform/core/config"
	"toy-platform/core/db"
	"toy-platform/core/handler"
	"toy-platform/core/logger"
	"toy-platform/core/runtime"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	cfg := config.Load("cmd/ts-quickjs/conf/config.json", "TS_HOST", "TS_PORT", "127.0.0.1", "9720")

	logLevels := cfg.Log
	if logLevels == nil {
		logLevels = &config.LogConfig{}
	}
	logs, err := logger.New("cmd/ts-quickjs/logs", logLevels.Debug, logLevels.Access, logLevels.Panic)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	logs.DebugL.Info(context.Background(), "ts-runner starting")

	database, err := db.New("scripts.db")
	if err != nil {
		logs.PanicL.Error(context.Background(), "failed to open database: %v", err)
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	h := &handler.Handler{
		Store:   database,
		Runner:  &runtime.Engine{},
		BpStore: database,
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

	mux.HandleFunc("GET /api/scripts", h.ListScripts)
	mux.HandleFunc("POST /api/scripts", h.CreateScript)
	mux.HandleFunc("GET /api/scripts/{id}", h.GetScript)
	mux.HandleFunc("PUT /api/scripts/{id}", h.UpdateScript)
	mux.HandleFunc("DELETE /api/scripts/{id}", h.DeleteScript)
	mux.HandleFunc("POST /api/scripts/{id}/run", h.RunScript)
	mux.HandleFunc("POST /api/scripts/{id}/debug", h.DebugScript)
	mux.HandleFunc("GET /api/scripts/{id}/breakpoints", h.ListBreakpoints)
	mux.HandleFunc("POST /api/scripts/{id}/breakpoints", h.SetBreakpoint)
	mux.HandleFunc("DELETE /api/scripts/{id}/breakpoints/{line}", h.DeleteBreakpoint)

	var srv http.Handler = mux
	srv = logger.AccessLog(logs.AccessL)(srv)
	srv = logger.Recovery(logs.PanicL)(srv)
	srv = logger.TraceMiddleware(logs.DebugL)(srv)

	addr := cfg.Address()
	logs.DebugL.Info(context.Background(), "server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}
