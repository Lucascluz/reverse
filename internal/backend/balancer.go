package backend

import (
	"sync/atomic"
)

type LoadBalancer interface {
	Next() *Backend
}

func NewLoadBalancer(backends []*Backend, lbType string) LoadBalancer {
	switch lbType {
	case "round-robin":
		return NewRoundRobinLoadBalancer(backends)
	case "least-connections":
		return NewLeastConnectionsLoadBalancer(backends)
	default:
		return NewRoundRobinLoadBalancer(backends)
	}
}

// Round Robin
type RoundRobinLoadBalancer struct {
	backends []*Backend
	current  atomic.Uint32
}

func NewRoundRobinLoadBalancer(backends []*Backend) *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		backends: backends,
		current:  atomic.Uint32{},
	}
}

func (lb *RoundRobinLoadBalancer) Next() *Backend {
	if len(lb.backends) == 0 {
		return nil
	}

	lb.current.Add(1 % uint32(len(lb.backends)))
	return lb.backends[lb.current.Load()]
}

// Least Connections
type LeastConnectionsLoadBalancer struct {
	backends []*Backend
	current  atomic.Uint32
}

func NewLeastConnectionsLoadBalancer(backends []*Backend) *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
		backends: backends,
		current:  atomic.Uint32{},
	}
}

func (lb *LeastConnectionsLoadBalancer) Next() *Backend {
	if len(lb.backends) == 0 {
		return nil
	}

	lb.current.Add(1 % uint32(len(lb.backends)))
	return lb.backends[lb.current.Load()]
}
