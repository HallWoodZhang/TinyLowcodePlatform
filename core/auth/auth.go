package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	CtxUserID   contextKey = "userID"
	CtxTenantID contextKey = "tenantID"
	CtxTenantName contextKey = "tenantName"
	CtxRole     contextKey = "role"
)

type Claims struct {
	Sub  string `json:"sub"`
	TID  string `json:"tid"`
	TN   string `json:"tn"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
	JTI  string `json:"jti"`
}

func Sign(claims Claims, secret []byte) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

func Verify(token string, secret []byte) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func NewSecret() []byte {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("auth: crypto/rand failed: " + err.Error())
	}
	return secret
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(CtxUserID).(string)
	return v
}

func TenantID(ctx context.Context) string {
	v, _ := ctx.Value(CtxTenantID).(string)
	return v
}

func Role(ctx context.Context) string {
	v, _ := ctx.Value(CtxRole).(string)
	return v
}

func TenantName(ctx context.Context) string {
	v, _ := ctx.Value(CtxTenantName).(string)
	return v
}

func JTI(ctx context.Context) string {
	v, _ := ctx.Value(contextKey("jti")).(string)
	return v
}

func WithUserContext(ctx context.Context, userID, tenantID, tenantName, role string) context.Context {
	ctx = context.WithValue(ctx, CtxUserID, userID)
	ctx = context.WithValue(ctx, CtxTenantID, tenantID)
	ctx = context.WithValue(ctx, CtxTenantName, tenantName)
	ctx = context.WithValue(ctx, CtxRole, role)
	return ctx
}
