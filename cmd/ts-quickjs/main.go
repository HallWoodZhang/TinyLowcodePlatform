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

	db, err := db.New("scripts.db")
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

	mux.HandleFunc("GET /api/scripts", h.ListScripts)
	mux.HandleFunc("POST /api/scripts", h.CreateScript)
	mux.HandleFunc("GET /api/scripts/{id}", h.GetScript)
	mux.HandleFunc("PUT /api/scripts/{id}", h.UpdateScript)
	mux.HandleFunc("DELETE /api/scripts/{id}", h.DeleteScript)
	mux.HandleFunc("POST /api/scripts/{id}/run", h.RunScript)

	addr := cfg.Address()
	log.Printf("Server starting on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
