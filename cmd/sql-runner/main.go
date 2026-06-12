package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"toy-platform/core/config"
	"toy-platform/core/handler"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	cfg := config.Load("cmd/sql-runner/conf/config.json", "SQL_HOST", "SQL_PORT", "127.0.0.1", "9721")

	sqlH, err := handler.NewSqlHandler("scripts.db")
	if err != nil {
		log.Fatalf("failed to open SQL handler: %v", err)
	}

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
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

	mux.HandleFunc("GET /api/sql/tables", sqlH.ListTables)
	mux.HandleFunc("POST /api/sql/run", sqlH.RunSQL)

	addr := cfg.Address()
	log.Printf("sql-runner starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
