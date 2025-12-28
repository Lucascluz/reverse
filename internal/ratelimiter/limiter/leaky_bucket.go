package limiter

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
)

type LeakyBucket struct {
	capacity   int           // max queued requests
	leakRate   int           // requests processed per second
	available  int           // current queued requests
	lastLeak   time.Time
	mu         sync.Mutex
}

func NewLeakyBucket(cfg config.RateLimiterConfig) *LeakyBucket {
	return &LeakyBucket{
		capacity:  cfg.Capacity,
		leakRate:  cfg.RefillRate,          // reused name ==> now leak rate
		available: 0,                       // start empty
		lastLeak:  time.Now(),
	}
}

// leak drains the bucket based on elapsed time
func (lb *LeakyBucket) leak() {
	now := time.Now()
	elapsed := now.Sub(lb.lastLeak).Seconds()
	lb.lastLeak = now

	leaked := int(elapsed * float64(lb.leakRate))
	if leaked > 0 {
		lb.available -= leaked
		if lb.available < 0 {
			lb.available = 0
		}
	}
}

// Allow tries to enqueue request into the bucket
func (lb *LeakyBucket) Allow(key string) (bool, time.Duration) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.leak()

	// If bucket has room, accept request
	if lb.available < lb.capacity {
		lb.available++
		return true, 0
	}

	// Otherwise bucket is full → request should be rejected
	// Optionally estimate wait time for next slot
	waitTime := time.Second / time.Duration(lb.leakRate)
	return false, waitTime
}
