package ratelimiter

import (
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
	"github.com/Lucascluz/reverxy/internal/ratelimiter/limiter"
)

type Limiter interface {
	// Allow checks if a request from 'key' (IP/User) is permitted.
	// Returns allowed (bool) and retryAfter (duration to wait, 0 if allowed).
	Allow(key string) (bool, time.Duration)
}

// Keep parameter order consistent with callers. This returns the interface type.
func New(cfg config.RateLimiterConfig) Limiter {
	switch cfg.Type {
	case "fixed-window":
		return limiter.NewFixed(cfg)
	}
	return limiter.NewFixed(cfg)
}
