package limiter

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
)

type FixedWindow struct {
	limit    int
	counter  atomic.Int32
	lastTick time.Time
	ticker   *time.Ticker
	stop     chan struct{}
	mu       sync.Mutex
}

func NewFixedWindow(cfg config.RateLimiterConfig) *FixedWindow {
	l := &FixedWindow{
		limit:   cfg.Limit,
		counter: atomic.Int32{},
		ticker:  time.NewTicker(time.Second),
		stop:    make(chan struct{}),
	}

	l.Start()

	return l
}

func (f *FixedWindow) Start() {
	go func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		for range f.ticker.C {
			f.lastTick = time.Now()
			f.counter.Store(0)
		}
	}()
}

func (f *FixedWindow) Stop() {
	close(f.stop)
	f.ticker.Stop()
}

func (f *FixedWindow) Allow(key string) (bool, time.Duration) {
	if f.counter.Load() >= int32(f.limit) {
		return false, time.Until(f.lastTick.Add(time.Second))
	}

	f.counter.Add(1)
	return true, 0
}
