package adminpkg

import (
	"encoding/json"
	"net/http"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/db"
)

type AdminHandler struct {
	Store db.Store
}

func (h *AdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.Store.ListTenants()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tenants == nil {
		tenants = []db.Tenant{}
	}
	writeJSON(w, http.StatusOK, tenants)
}

type createTenantReq struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Betamap string `json:"betamap,omitempty"`
}

func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req createTenantReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" || req.Label == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and label are required"})
		return
	}
	tenant, err := h.Store.CreateTenant(req.Name, req.Label, req.Betamap)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, tenant)
}

func (h *AdminHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenant, err := h.Store.GetTenant(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	writeJSON(w, http.StatusOK, tenant)
}

func (h *AdminHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Store.DeleteTenant(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	users, err := h.Store.ListUsersByTenant(tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	if users == nil {
		users = []db.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

type createUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	user, err := h.Store.CreateUser(tenantID, req.Username, hash, req.Role)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	user.PasswordHash = ""
	writeJSON(w, http.StatusCreated, user)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")
	if err := h.Store.DeleteUser(uid); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Backdoor ---

func (h *AdminHandler) ListAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Store.ListAllUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if users == nil {
		users = []db.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

type updatePasswordReq struct {
	Password string `json:"password"`
}

func (h *AdminHandler) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")
	var req updatePasswordReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password is required"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	if err := h.Store.UpdateUserPassword(uid, hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Betamap management ---

func (h *AdminHandler) GetTenantBetamap(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenant, err := h.Store.GetTenant(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(tenant.Betamap))
}

func (h *AdminHandler) UpdateTenantBetamap(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	betamap := string(raw)
	_, err := h.Store.UpdateTenant(id, nil, nil, &betamap)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "updated"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
