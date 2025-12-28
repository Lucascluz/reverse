package limiter

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
)

type TokenBucket struct {
	tokens     int
	capacity   int
	refillRate int // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(cfg config.RateLimiterConfig) *TokenBucket {
	return &TokenBucket{
		tokens:     cfg.Capacity,
		capacity:   cfg.Capacity,
		refillRate: cfg.RefillRate,
		lastRefill: time.Now(),
		mu:         sync.Mutex{},
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	newTokens := int(elapsed * float64(tb.refillRate))
	if newTokens > 0 {
		tb.tokens += newTokens
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
	}
}

func (tb *TokenBucket) Allow(key string) (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens > 0 {
		tb.tokens--
		return true, 0
	}

	waitTime := time.Duration(float64(time.Second) / float64(tb.refillRate))
	return false, waitTime
}
