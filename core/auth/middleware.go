package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"toy-platform/core/db"
)

func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				writeAuthError(w, "missing token")
				return
			}

			claims, err := Verify(token, secret)
			if err != nil {
				writeAuthError(w, err.Error())
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, CtxUserID, claims.Sub)
			ctx = context.WithValue(ctx, CtxTenantID, claims.TID)
			ctx = context.WithValue(ctx, CtxTenantName, claims.TN)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if Role(r.Context()) != "admin" {
				writeError(w, http.StatusForbidden, "admin required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func BetamapMiddleware(store db.TenantStore, feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tid := TenantID(r.Context())
			if tid == "" {
				writeError(w, http.StatusUnauthorized, "missing tenant")
				return
			}

			role := Role(r.Context())
			if role == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			tenant, err := store.GetTenant(tid)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to load tenant")
				return
			}

			var betamap map[string]any
			if err := json.Unmarshal([]byte(tenant.Betamap), &betamap); err != nil {
				writeError(w, http.StatusInternalServerError, "invalid betamap")
				return
			}

			if v, ok := betamap[feature]; !ok || v != true {
				writeError(w, http.StatusForbidden, "feature disabled: "+feature)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	cookie, err := r.Cookie("token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusUnauthorized, msg)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
