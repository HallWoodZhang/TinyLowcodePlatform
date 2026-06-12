package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"toy-platform/core/db"
	"toy-platform/core/runtime"
)

type Handler struct {
	Store  db.ScriptStore
	Runner runtime.Runner
}

type createReq struct {
	Name   string `json:"name"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	TSCode string `json:"tsCode"`
}

type updateReq struct {
	Name   *string `json:"name"`
	Label  *string `json:"label"`
	Type   *string `json:"type"`
	TSCode *string `json:"tsCode"`
}

func (h *Handler) ListScripts(w http.ResponseWriter, r *http.Request) {
	scripts, err := h.Store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if scripts == nil {
		scripts = []db.ScriptSummary{}
	}
	writeJSON(w, http.StatusOK, scripts)
}

func (h *Handler) CreateScript(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" || req.Label == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and label are required"})
		return
	}
	script, err := h.Store.Create(req.Name, req.Label, req.Type, req.TSCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, script)
}

func (h *Handler) GetScript(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	script, err := h.Store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "script not found"})
		return
	}
	writeJSON(w, http.StatusOK, script)
}

func (h *Handler) UpdateScript(w http.ResponseWriter, r *http.Request) {
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
	script, err := h.Store.Update(id, req.Name, req.Label, req.Type, req.TSCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, script)
}

func (h *Handler) DeleteScript(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.Store.Delete(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RunScript(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	script, err := h.Store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "script not found"})
		return
	}
	result := h.Runner.Run(script.TSCode, 10000)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
