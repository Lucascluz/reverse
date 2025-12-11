package loadbalancer

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/loadbalancer/pool"
	"github.com/Lucascluz/reverse/internal/loadbalancer/strategy"
)

type LoadBalancer struct {
	balancer Balancer
	pool     *pool.Pool

	ready atomic.Bool
	mu    sync.Mutex
}

type Balancer interface {
	Next() *pool.Backend
}

func New(cfg *config.LoadBalancerConfig, onReadyChanged func(ready bool)) *LoadBalancer {
	lb := &LoadBalancer{
		ready: atomic.Bool{},
	}

	// Create the pool with a callback that updates our readiness
	lb.pool = pool.NewPool(&cfg.Pool, func() {
		newReady := lb.pool.IsReady()
		oldReady := lb.ready.Load()

		// Only call callback if state changed
		if newReady != oldReady {
			lb.ready.Store(newReady)
			if onReadyChanged != nil {
				onReadyChanged(newReady)
			}
		}
	})

	// Create the balancing strategy
	lb.balancer = newBalancingStrategy(lb.pool.Backends(), cfg.Type)

	// Set initial readiness
	lb.ready.Store(lb.pool.IsReady())

	return lb
}

func (lb *LoadBalancer) Next() (*pool.Backend, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backends := lb.pool.Backends()
	maxTries := len(backends)

	for _ = range maxTries {
		backend := lb.balancer.Next()

		if backend == nil || !backend.IsHealthy() {
			continue
		}

		if backend.IsAtCapacity() {
			continue
		}

		return backend, nil
	}

	return nil, fmt.Errorf("no healthy backends available")
}

// IsReady returns true if the load balancer is ready to serve requests
func (lb *LoadBalancer) IsReady() bool {
	return lb.ready.Load()
}

func newBalancingStrategy(backends []*pool.Backend, strategyType string) Balancer {
	switch strategyType {
	case "round-robin":
		return strategy.NewRoundRobin(backends)
	default:
		return strategy.NewRoundRobin(backends)
	}
}
