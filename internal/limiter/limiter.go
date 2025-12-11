package limiter

import (
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

var TrustedProxies []string

type Limiter interface {
	// Allow checks if a request from 'key' (IP/User) is permitted.
	// Returns allowed (bool) and retryAfter (duration to wait, 0 if allowed).
	Allow(key string) (bool, time.Duration)
}

// Keep parameter order consistent with callers. This returns the interface type.
func New(cfg config.RateLimiterConfig) Limiter {
	switch cfg.Type {
	case "fixed-window":
		return newFixed(cfg)
	}
	return newFixed(cfg)
}
