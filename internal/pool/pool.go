package pool

import (
	"fmt"
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/backend"
	"github.com/Lucascluz/reverse/internal/balancer"
	"github.com/Lucascluz/reverse/internal/checker"
	"github.com/Lucascluz/reverse/internal/config"
)

type Pool struct {
	backends      []*backend.Backend
	loadBalancer  balancer.Balancer
	healthChecker *checker.HealthChecker

	mu sync.RWMutex
}

func New(cfg *config.PoolConfig, updateReady func()) *Pool {

	backends := make([]*backend.Backend, len(cfg.Backends))

	for i, backendCfg := range cfg.Backends {
		backends[i] = backend.New(backendCfg)
	}

	loadBalancer := balancer.New(backends, cfg.LoadBalancer)

	healthChecker := checker.New(&cfg.HealthChecker)

	pool := &Pool{
		backends:      backends,
		loadBalancer:  loadBalancer,
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

func (p *Pool) NextUrl() (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// TODO: Define retry policy for backend selection
	if next := p.loadBalancer.Next(); next != nil {
		return next.Url, nil
	}

	time.Sleep(3 * time.Second)
	next := p.loadBalancer.Next()
	if next != nil {
		return next.Url, nil
	}

	return "", fmt.Errorf("no healthy backend available")
}
