package loadbalancer

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/loadbalancer/balancer"
	"github.com/Lucascluz/reverse/internal/loadbalancer/pool"
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

func NewLoadBalancer(cfg *config.LoadBalancerConfig) *LoadBalancer {
	lb := &LoadBalancer{
		ready: atomic.Bool{},
	}

	// Create the pool with a callback that updates our readiness
	lb.pool = pool.NewPool(&cfg.Pool)

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
