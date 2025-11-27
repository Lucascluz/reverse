package backend

import (
	"math/rand"
	"sync"

	"github.com/Lucascluz/reverse/internal/config"
)

type Pool struct {
	backends      []*Backend
	healthChecker *HealthChecker

	mu sync.RWMutex
}

func NewPool(cfg *config.PoolConfig) *Pool {

	backends := make([]*Backend, len(cfg.Backends))
	for i, backendCfg := range cfg.Backends {
		backends[i] = NewBackend(backendCfg)
	}

	pool := &Pool{
		backends:      backends,
		healthChecker: NewHealthChecker(&cfg.HealthChecker),
		mu:            sync.RWMutex{},
	}

	go pool.healthChecker.Start(backends)

	return pool
}

func (p *Pool) NextUrl() string {
    p.mu.RLock()
    defer p.mu.RUnlock()

    // Filter healthy backends
    var healthy []*Backend
    for _, backend := range p.backends {
        backend.mu.RLock()
        isHealthy := backend.Healthy
        backend.mu.RUnlock()
        
        if isHealthy {
            healthy = append(healthy, backend)
        }
    }

    // TODO: Implement proper loadbalancing
    if len(healthy) > 0 {
        return healthy[rand.Intn(len(healthy))].Url
    }

    // Fallback: return first backend (or handle differently)
    if len(p.backends) > 0 {
        return p.backends[0].Url
    }

    return "" // or panic, or error
}

