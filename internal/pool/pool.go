package pool

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/backend"
	"github.com/Lucascluz/reverse/internal/balancer"
	"github.com/Lucascluz/reverse/internal/config"
)

type Pool struct {
	backends []*backend.Backend

	loadBalancer  balancer.Balancer
	healthChecker *HealthChecker
	mu            sync.RWMutex
}

func New(cfg *config.PoolConfig, updateReady func()) *Pool {

	backends := make([]*backend.Backend, len(cfg.Backends))

	for i, backendCfg := range cfg.Backends {
		backends[i] = backend.New(backendCfg)
	}

	loadBalancer := balancer.New(backends, cfg.LoadBalancer)

	pool := &Pool{
		backends:      backends,
		healthChecker: NewHealthChecker(&cfg.HealthChecker),
		loadBalancer:  loadBalancer,
		mu:            sync.RWMutex{},
	}

	go pool.healthChecker.Start(backends, updateReady)

	return pool
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

func (p *Pool) NextUrl() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// TODO: Define retry policy for backend selection
	if next := p.loadBalancer.Next(); next != nil {
		return next.Url
	}

	time.Sleep(3 * time.Second)

	return p.loadBalancer.Next().Url
}
