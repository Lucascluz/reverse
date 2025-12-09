package backend

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type Backend struct {
	Name      string
	Url       string
	HealthUrl string
	Weight    int
	MaxConns  int

	healthy         bool
	lastCheck       time.Time
	failureCount    int
	backoffTime     time.Duration
	activeConns     int
	totalRequests   int
	avgResponseTime time.Duration

	mu sync.RWMutex
}

func New(cfg config.BackendConfig) *Backend {
	return &Backend{
		Name:      cfg.Name,
		Url:       cfg.Url,
		HealthUrl: cfg.HealthUrl,
		Weight:    cfg.Weight,
		MaxConns:  cfg.MaxConns,

		healthy:         false,
		lastCheck:       time.Now(),
		failureCount:    0,
		backoffTime:     1 * time.Second,
		activeConns:     0,
		totalRequests:   0,
		avgResponseTime: time.Duration(0),
		mu:              sync.RWMutex{},
	}
}

func (b *Backend) IsHealthy() bool {
	// Lock to safely check health status
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.healthy
}

func (b *Backend) IsBackedOff() bool {
	// Lock to safely check backoff and update LastCheck
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if backed off
	return time.Now().Before(b.lastCheck.Add(b.backoffTime))
}

func (b *Backend) UpdateHealth(success bool) {

	// Lock to update health status
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastCheck = time.Now()

	if success {

		b.backoffTime = 1 * time.Second
		b.healthy = true
		return

	} else {

		// Failure case (either error or bad status code)
		b.failureCount += 1
		b.healthy = false

		// Exponential backoff with upper limit of 60 seconds
		if b.backoffTime < 60*time.Second {
			b.backoffTime *= 2
		}
	}

}
