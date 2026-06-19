package auth

import (
	"testing"
	"time"
)

func TestSignAndVerify(t *testing.T) {
	secret := NewSecret()
	claims := Claims{
		Sub:  "001b00000000000000000001",
		TID:  "001a00000000000000000001",
		TN:   "admin",
		Role: "admin",
		Exp:  time.Now().Add(time.Hour).Unix(),
		Iat:  time.Now().Unix(),
		JTI:  "jti-001",
	}

	token, err := Sign(claims, secret)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	parsed, err := Verify(token, secret)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if parsed.Sub != claims.Sub {
		t.Errorf("sub = %s, want %s", parsed.Sub, claims.Sub)
	}
	if parsed.Role != "admin" {
		t.Errorf("role = %s", parsed.Role)
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	claims := Claims{Sub: "x", Exp: time.Now().Add(time.Hour).Unix()}
	token, _ := Sign(claims, NewSecret())
	_, err := Verify(token, NewSecret())
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestVerifyExpiredToken(t *testing.T) {
	secret := NewSecret()
	claims := Claims{
		Sub: "x",
		Exp: time.Now().Add(-time.Hour).Unix(),
	}
	token, _ := Sign(claims, secret)
	_, err := Verify(token, secret)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestVerifyTamperedPayload(t *testing.T) {
	secret := NewSecret()
	claims := Claims{Sub: "x", Exp: time.Now().Add(time.Hour).Unix()}
	token, _ := Sign(claims, secret)
	tampered := token[:len(token)-3] + "xxx"
	_, err := Verify(tampered, secret)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestVerifyBadFormat(t *testing.T) {
	_, err := Verify("not.a.jwt.token.extra", NewSecret())
	if err == nil {
		t.Error("expected error for bad format")
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("test123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword("test123", hash) {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword("wrong", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashPasswordUniqueSalt(t *testing.T) {
	h1, _ := HashPassword("test123")
	h2, _ := HashPassword("test123")
	if h1 == h2 {
		t.Error("same password should produce different hashes (different salt)")
	}
}

func TestNewSecret(t *testing.T) {
	s1 := NewSecret()
	s2 := NewSecret()
	if len(s1) != 32 {
		t.Errorf("secret length = %d, want 32", len(s1))
	}
	if string(s1) == string(s2) {
		t.Error("two secrets should be different")
	}
}
