package authpkg

import (
	"encoding/json"
	"net/http"
	"time"

	"tiny-lowcode-platform/core/auth"
	"tiny-lowcode-platform/core/db"
	"github.com/google/uuid"
)

type AuthHandler struct {
	Store       db.Store
	TokenSecret []byte
	TokenExpire time.Duration
	Blacklist   auth.TokenBlacklist
}

type loginReq struct {
	Tenant   string `json:"tenant"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResp struct {
	Token string   `json:"token"`
	User  *db.User `json:"user"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Tenant == "" || req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant, username, and password are required"})
		return
	}

	user, err := h.Store.GetUserByTenantAndUsername(req.Tenant, req.Username)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	tenant, err := h.Store.GetTenantByName(req.Tenant)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	now := time.Now()
	claims := auth.Claims{
		Sub:  user.ID,
		TID:  tenant.ID,
		TN:   tenant.Name,
		Role: user.Role,
		Exp:  now.Add(h.TokenExpire).Unix(),
		Iat:  now.Unix(),
		JTI:  uuid.New().String(),
	}

	token, err := auth.Sign(claims, h.TokenSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}

	user.PasswordHash = ""
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  now.Add(h.TokenExpire),
	})

	writeJSON(w, http.StatusOK, loginResp{Token: token, User: user})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Add token to blacklist if Redis is configured
	jti := auth.JTI(r.Context())
	if jti != "" && h.Blacklist != nil {
		h.Blacklist.Add(jti, h.TokenExpire)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"ok": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	user, err := h.Store.GetUser(userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	user.PasswordHash = ""

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"tenantId":   user.TenantID,
		"tenantName": auth.TenantName(r.Context()),
	})
}

func (h *AuthHandler) Betamap(w http.ResponseWriter, r *http.Request) {
	tid := auth.TenantID(r.Context())
	if tid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	tenant, err := h.Store.GetTenant(tid)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(tenant.Betamap))
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	user, err := h.Store.GetUser(userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "user not found"})
		return
	}

	tenant, err := h.Store.GetTenant(user.TenantID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant not found"})
		return
	}

	now := time.Now()
	claims := auth.Claims{
		Sub:  user.ID,
		TID:  tenant.ID,
		TN:   tenant.Name,
		Role: user.Role,
		Exp:  now.Add(h.TokenExpire).Unix(),
		Iat:  now.Unix(),
		JTI:  uuid.New().String(),
	}

	token, err := auth.Sign(claims, h.TokenSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to sign token"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  now.Add(h.TokenExpire),
	})

	writeJSON(w, http.StatusOK, loginResp{Token: token, User: user})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
