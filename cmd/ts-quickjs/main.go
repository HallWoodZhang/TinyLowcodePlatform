package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"toy-platform/core/config"
	"toy-platform/core/db"
	"toy-platform/core/handler"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	cfg := config.Load("cmd/ts-quickjs/conf/config.json")

	db, err := db.New("snippets.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	h := &handler.Handler{DB: db}

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("failed to setup static files: %v", err)
	}
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("GET /api/snippets", h.ListSnippets)
	mux.HandleFunc("POST /api/snippets", h.CreateSnippet)
	mux.HandleFunc("GET /api/snippets/{id}", h.GetSnippet)
	mux.HandleFunc("PUT /api/snippets/{id}", h.UpdateSnippet)
	mux.HandleFunc("DELETE /api/snippets/{id}", h.DeleteSnippet)
	mux.HandleFunc("POST /api/snippets/{id}/run", h.RunSnippet)

	addr := cfg.Address()
	log.Printf("Server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
