package pool

import (
	"sync"

	"github.com/Lucascluz/reverxy/internal/config"
)

type Pool struct {
	mu sync.RWMutex
	backends []*Backend
}

func NewPool(cfg *config.PoolConfig) *Pool {

	backends := make([]*Backend, len(cfg.Backends))

	for i, backendCfg := range cfg.Backends {
		backends[i] = NewBackend(backendCfg)
	}

	pool := &Pool{
		backends: backends,
		mu:       sync.RWMutex{},
	}

	return pool
}

func (p *Pool) IsReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Pool is ready if at least one backend is healthy
	// This prevents the proxy from going not-ready if a single backend fails
	if len(p.backends) == 0 {
		return false
	}

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
