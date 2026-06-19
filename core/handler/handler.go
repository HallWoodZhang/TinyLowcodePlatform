package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/db"
	"tiny-lowcode-platform/core/runtime"
)

type Handler struct {
	Store  db.Store
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
	role := auth.Role(r.Context())
	tenantID := auth.TenantID(r.Context())

	var scripts []db.ScriptSummary
	var err error
	if role == "admin" {
		scripts, err = h.Store.ListAllScripts()
	} else {
		scripts, err = h.Store.ListScripts(tenantID)
	}
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

	tenantID := auth.TenantID(r.Context())
	script, err := h.Store.CreateScript(tenantID, req.Name, req.Label, req.Type, req.TSCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, script)
}

func (h *Handler) GetScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	script, err := h.Store.GetScript(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "script not found"})
		return
	}

	role := auth.Role(r.Context())
	if role != "admin" {
		tenantID := auth.TenantID(r.Context())
		if script.TenantID != tenantID {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "script not found"})
			return
		}
	}

	writeJSON(w, http.StatusOK, script)
}

func (h *Handler) UpdateScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req updateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	script, err := h.Store.UpdateScript(id, req.Name, req.Label, req.Type, req.TSCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, script)
}

func (h *Handler) DeleteScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.Store.DeleteScript(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RunScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	script, err := h.Store.GetScript(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "script not found"})
		return
	}

	tenantID := auth.TenantID(r.Context())
	resolver := func(name string) (string, error) {
		name = strings.TrimPrefix(name, "./")
		s, err := h.Store.GetScriptByName(tenantID, name)
		if err != nil {
			return "", err
		}
		return s.TSCode, nil
	}

	result := h.Runner.Run(script.TSCode, resolver, 10000)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) ListBreakpoints(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	bps, err := h.Store.ListBreakpoints(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if bps == nil {
		bps = []db.Breakpoint{}
	}
	writeJSON(w, http.StatusOK, bps)
}

func (h *Handler) SetBreakpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Line    int  `json:"line"`
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := h.Store.SetBreakpoint(id, req.Line, req.Enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DeleteBreakpoint(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	line := r.PathValue("line")
	if line == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid line"})
		return
	}
	var lineNum int
	if err := json.Unmarshal([]byte(line), &lineNum); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid line"})
		return
	}
	if err := h.Store.DeleteBreakpoint(id, lineNum); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DebugScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Skip int `json:"skip"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	script, err := h.Store.GetScript(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "script not found"})
		return
	}
	bps, err := h.Store.ListBreakpoints(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	tenantID := auth.TenantID(r.Context())
	resolver := func(name string) (string, error) {
		name = strings.TrimPrefix(name, "./")
		s, err := h.Store.GetScriptByName(tenantID, name)
		if err != nil {
			return "", err
		}
		return s.TSCode, nil
	}

	bpLines := make([]runtime.BpLine, len(bps))
	for i, bp := range bps {
		bpLines[i] = runtime.BpLine{Line: bp.Line, Enabled: bp.Enabled}
	}

	result := h.Runner.Debug(script.TSCode, resolver, bpLines, req.Skip, 30000)
	writeJSON(w, http.StatusOK, result)
}
