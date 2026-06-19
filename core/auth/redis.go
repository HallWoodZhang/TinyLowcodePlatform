package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const blacklistKeyPrefix = "token:blacklist:"

type TokenBlacklist interface {
	Add(jti string, ttl time.Duration) error
	IsBlocked(jti string) (bool, error)
}

type nopBlacklist struct{}

func (n *nopBlacklist) Add(jti string, ttl time.Duration) error { return nil }
func (n *nopBlacklist) IsBlocked(jti string) (bool, error)       { return false, nil }

type redisBlacklist struct {
	client *redis.Client
}

func NewBlacklist(addr, password string, db int) TokenBlacklist {
	if addr == "" {
		return &nopBlacklist{}
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &redisBlacklist{client: client}
}

func (b *redisBlacklist) Add(jti string, ttl time.Duration) error {
	ctx := context.Background()
	key := blacklistKeyPrefix + jti
	return b.client.Set(ctx, key, "1", ttl).Err()
}

func (b *redisBlacklist) IsBlocked(jti string) (bool, error) {
	ctx := context.Background()
	key := blacklistKeyPrefix + jti
	_, err := b.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis check: %w", err)
	}
	return true, nil
}
