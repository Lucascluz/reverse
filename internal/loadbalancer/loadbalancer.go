package loadbalancer

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Lucascluz/reverxy/internal/config"
	"github.com/Lucascluz/reverxy/internal/loadbalancer/balancer"
	"github.com/Lucascluz/reverxy/internal/loadbalancer/pool"
)

type LoadBalancer struct {
	mu   sync.Mutex
	pool *pool.Pool

	balancer Balancer
	ready    atomic.Bool
}

type Balancer interface {
	Next() *pool.Backend
}

func NewLoadBalancer(cfg *config.LoadBalancerConfig) *LoadBalancer {
	// Create the pool with a callback that updates our readiness
	pool := pool.NewPool(&cfg.Pool)

	// Create the balancing strategy
	balancer := newBalancingStrategy(pool.Backends(), cfg.Type)

	return &LoadBalancer{
		pool:   pool,
		balancer: balancer,
		ready: atomic.Bool{},
	}
}

func (lb *LoadBalancer) Next() (*pool.Backend, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backends := lb.pool.Backends()
	maxTries := len(backends)

	for range maxTries {
		backend := lb.balancer.Next()

		if backend == nil || !backend.IsHealthy() {
			continue
		}

		if backend.IsAtCapacity() {
			continue
		}

		lb.SetReady(true)
		return backend, nil
	}

	lb.SetReady(false)
	return nil, fmt.Errorf("no healthy backends available")
}

// IsReady returns true if the load balancer is ready to serve requests
func (lb *LoadBalancer) IsReady() bool {
	return lb.ready.Load()
}

// SetReady sets the readiness of the load balancer
func (lb *LoadBalancer) SetReady(ready bool) {
	lb.ready.Store(ready)
}

// Pool returns the underlying Pool instance
func (lb *LoadBalancer) Pool() *pool.Pool {
	return lb.pool
}

func newBalancingStrategy(backends []*pool.Backend, balancerType string) Balancer {
	switch balancerType {
	case "round-robin":
		return balancer.NewRoundRobin(backends)
	default:
		return balancer.NewRoundRobin(backends)
	}
}
