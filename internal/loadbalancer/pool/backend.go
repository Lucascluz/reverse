package pool

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
)

type Backend struct {
	name      string
	url       string
	healthUrl string
	weight    int
	maxConns  int

	mu              sync.RWMutex
	healthy         bool
	failureCount    int
	activeConns     int
	totalRequests   int
	lastCheck       time.Time
	backoffTime     time.Duration
	avgResponseTime time.Duration
}

func NewBackend(cfg config.BackendConfig) *Backend {
	return &Backend{
		name:      cfg.Name,
		url:       cfg.Url,
		healthUrl: cfg.HealthUrl,
		weight:    cfg.Weight,
		maxConns:  cfg.MaxConns,

		healthy:         false,
		lastCheck:       time.Now().Add(-2 * time.Second), // Initialize to allow immediate health check
		failureCount:    0,
		backoffTime:     1 * time.Second,
		activeConns:     0,
		totalRequests:   0,
		avgResponseTime: time.Duration(0),
		mu:              sync.RWMutex{},
	}
}

func (b *Backend) Name() string {
	return b.name
}

func (b *Backend) Url() string {
	return b.url
}

func (b *Backend) HealthUrl() string {
	return b.healthUrl
}

func (b *Backend) Weight() int {
	return b.weight
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

// IsAtCapacity returns true if backend reached max connections
func (b *Backend) IsAtCapacity() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.maxConns <= 0 {
		// No limit
		return false
	}

	return b.activeConns >= b.maxConns
}

// ActiveConns returns current active connection count
func (b *Backend) ActiveConns() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.activeConns
}

// IncrementConnections increments the active connection count
func (b *Backend) IncrementConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeConns++
	b.totalRequests++
}

// DecrementConnections decrements the active connection count
func (b *Backend) DecrementConnections() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.activeConns > 0 {
		b.activeConns--
	}
}
