package pool

import (
	"sync"

	"github.com/Lucascluz/reverse/internal/config"
)

type Pool struct {
	backends      []*Backend
	healthChecker *HealthChecker

	mu sync.RWMutex
}

func NewPool(cfg *config.PoolConfig, updateReady func()) *Pool {

	backends := make([]*Backend, len(cfg.Backends))

	for i, backendCfg := range cfg.Backends {
		backends[i] = NewBackend(backendCfg)
	}

	healthChecker := NewHealthChecker(&cfg.HealthChecker)

	pool := &Pool{
		backends:      backends,
		healthChecker: healthChecker,
		mu:            sync.RWMutex{},
	}

	go pool.Start(updateReady)

	return pool
}

// Start starts the pool and its health checker
func (p *Pool) Start(updateReady func()) {
	p.healthChecker.Start(p.backends, updateReady)
}

// Stop stops the pool and its health checker
func (p *Pool) Stop() {
	p.healthChecker.Stop()
}

// A pool is ready if there is at least one healthy backend
func (p *Pool) IsReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, backend := range p.backends {
		if backend.IsHealthy() {
			return true
		}
	}
	return false
}

// Backends returns a copy of the backends slice
func (p *Pool) Backends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()

	backends := make([]*Backend, len(p.backends))
	copy(backends, p.backends)
	return backends
}
