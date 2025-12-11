package limiter

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type Fixed struct {
	limit    int
	counter  atomic.Int32
	lastTick time.Time
	ticker   *time.Ticker
	stop     chan struct{}
	mu       sync.Mutex
}

func newFixed(cfg config.RateLimiterConfig) *Fixed {
	l := &Fixed{
		limit:   cfg.Limit,
		counter: atomic.Int32{},
		ticker:  time.NewTicker(time.Second),
		stop:    make(chan struct{}),
	}

	l.Start()

	return l
}

func (f *Fixed) Start() {
	go func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		for range f.ticker.C {
			f.lastTick = time.Now()
			f.counter.Store(0)
		}
	}()
}

func (f *Fixed) Stop() {
	close(f.stop)
	f.ticker.Stop()
}

func (f *Fixed) Allow(key string) (bool, time.Duration) {
	if f.counter.Load() >= int32(f.limit) {
		return false, time.Until(f.lastTick.Add(time.Second))
	}

	f.counter.Add(1)
	return true, 0
}
