package ratelimiter

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Limiter struct {
	rdb    *redis.Client
	script *redis.Script
}

//go:embed script.lua
var ratelimitScript string

func NewLimiter(rdb *redis.Client) *Limiter {
	return &Limiter{
		rdb:    rdb,
		script: redis.NewScript(ratelimitScript),
	}
}

func (l *Limiter) Allow(
	ctx context.Context,
	key string,
	limit int,
	window time.Duration,
) (bool, int, error) {
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()

	result, err := l.script.Run(
		ctx,
		l.rdb,
		[]string{key},
		now,
		windowMs,
		limit,
	).Result()

	if err != nil {
		return false, 0, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, fmt.Errorf("invalid rate limiter response: %T", result)
	}

	allowed, ok := values[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected allowed value type: %T", values[0])
	}

	remaining, ok := values[1].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected remaining value type: %T", values[1])
	}

	return allowed == 1, int(remaining), nil
}
