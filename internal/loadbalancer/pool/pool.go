package pool

import (
	"sync"

	"github.com/Lucascluz/reverse/internal/config"
)

type Pool struct {
	backends []*Backend

	mu sync.RWMutex
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

	for _, backend := range p.backends {
		if !backend.IsHealthy() {
			return false
		}
	}
	return true
}

// Backends returns a copy of the backends slice
func (p *Pool) Backends() []*Backend {
	p.mu.RLock()
	defer p.mu.RUnlock()

	backends := make([]*Backend, len(p.backends))
	copy(backends, p.backends)
	return backends
}
