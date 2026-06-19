package auth

import (
	"time"
)

type TokenBlacklist interface {
	Add(jti string, ttl time.Duration) error
	IsBlocked(jti string) (bool, error)
}

// nopBlacklist is used when Redis is not configured.
type nopBlacklist struct{}

func (n *nopBlacklist) Add(jti string, ttl time.Duration) error {
	return nil
}

func (n *nopBlacklist) IsBlocked(jti string) (bool, error) {
	return false, nil
}

func NewBlacklist(redisAddr string) TokenBlacklist {
	if redisAddr == "" {
		return &nopBlacklist{}
	}
	// Placeholder: return a real Redis implementation when go-redis is available.
	return &nopBlacklist{}
}
