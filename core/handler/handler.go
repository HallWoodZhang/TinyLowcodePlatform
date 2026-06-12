package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"toy-platform/core/db"
	"toy-platform/core/runtime"
)

type Handler struct {
	Store  db.ScriptStore
	Runner runtime.Runner
	BpStore BreakpointStore
}

type BreakpointStore interface {
	ListBreakpoints(scriptID int64) ([]db.Breakpoint, error)
	SetBreakpoint(scriptID int64, line int, enabled bool) error
	DeleteBreakpoint(scriptID int64, line int) error
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

	var resolver runtime.ScriptResolver
	if store, ok := h.Store.(interface{ GetByName(string) (*db.Script, error) }); ok {
		resolver = func(name string) (string, error) {
			name = strings.TrimPrefix(name, "./")
			s, err := store.GetByName(name)
			if err != nil {
				return "", err
			}
			return s.TSCode, nil
		}
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	bps, err := h.BpStore.ListBreakpoints(id)
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
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
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
	if err := h.BpStore.SetBreakpoint(id, req.Line, req.Enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DeleteBreakpoint(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	line, err := strconv.Atoi(r.PathValue("line"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid line"})
		return
	}
	if err := h.BpStore.DeleteBreakpoint(id, line); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DebugScript(w http.ResponseWriter, r *http.Request) {
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
	bps, err := h.BpStore.ListBreakpoints(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var resolver runtime.ScriptResolver
	if store, ok := h.Store.(interface{ GetByName(string) (*db.Script, error) }); ok {
		resolver = func(name string) (string, error) {
			name = strings.TrimPrefix(name, "./")
			s, err := store.GetByName(name)
			if err != nil {
				return "", err
			}
			return s.TSCode, nil
		}
	}

	bpLines := make([]runtime.BpLine, len(bps))
	for i, bp := range bps {
		bpLines[i] = runtime.BpLine{Line: bp.Line, Enabled: bp.Enabled}
	}

	result := h.Runner.Debug(script.TSCode, resolver, bpLines, 30000)
	writeJSON(w, http.StatusOK, result)
}
