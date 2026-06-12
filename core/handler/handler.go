package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"toy-platform/core/db"
	"toy-platform/core/runtime"
)

type Handler struct {
	DB *db.DB
}

type createReq struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	TSCode string `json:"tsCode"`
}

type updateReq struct {
	Name   *string `json:"name"`
	Label  *string `json:"label"`
	TSCode *string `json:"tsCode"`
}

func (h *Handler) ListSnippets(w http.ResponseWriter, r *http.Request) {
	snippets, err := h.DB.ListSnippets()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if snippets == nil {
		snippets = []db.SnippetSummary{}
	}
	writeJSON(w, http.StatusOK, snippets)
}

func (h *Handler) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" || req.Label == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and label are required"})
		return
	}
	snippet, err := h.DB.CreateSnippet(req.Name, req.Label, req.TSCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, snippet)
}

func (h *Handler) GetSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	snippet, err := h.DB.GetSnippet(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "snippet not found"})
		return
	}
	writeJSON(w, http.StatusOK, snippet)
}

func (h *Handler) UpdateSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req updateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	snippet, err := h.DB.UpdateSnippet(id, req.Name, req.Label, req.TSCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, snippet)
}

func (h *Handler) DeleteSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.DB.DeleteSnippet(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RunSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	snippet, err := h.DB.GetSnippet(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "snippet not found"})
		return
	}
	result := runtime.RunTSCode(snippet.TSCode, 10000)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
