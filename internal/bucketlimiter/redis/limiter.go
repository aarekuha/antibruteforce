// Package redis реализует списание попыток в Redis
package redis

import (
	"antibruteforce/internal/antibruteforce"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

type Increment func(ctx context.Context, key string, limit uint32) (bool, error)

type Limits struct {
	IP       uint32
	Login    uint32
	Password uint32
}

type Limiter struct {
	increment Increment
	limits    Limits
	secret    []byte
}

func NewLimiter(increment Increment, limits Limits, secret string) (*Limiter, error) {
	if increment == nil || secret == "" {
		return nil, errors.New("redis limiter requires an increment function and an HMAC secret")
	}
	return &Limiter{increment: increment, limits: limits, secret: []byte(secret)}, nil
}

func (l *Limiter) TryConsume(ctx context.Context, key antibruteforce.BucketKey) (bool, error) {
	var redisKey string
	var limit uint32
	switch key.Kind {
	case antibruteforce.BucketIP:
		redisKey = "abf:v2:ip:" + key.Value
		limit = l.limits.IP
	case antibruteforce.BucketLogin:
		sum := sha256.Sum256([]byte(key.Value))
		redisKey = "abf:v2:login:" + hex.EncodeToString(sum[:])
		limit = l.limits.Login
	case antibruteforce.BucketPassword:
		mac := hmac.New(sha256.New, l.secret)
		mac.Write([]byte(key.Value))
		redisKey = "abf:v2:password:" + hex.EncodeToString(mac.Sum(nil))
		limit = l.limits.Password
	default:
		return false, errors.New("unknown bucket kind")
	}

	return l.increment(ctx, redisKey, limit)
}
