package balancer

import (
	"github.com/Lucascluz/reverse/internal/backend"
	"github.com/Lucascluz/reverse/internal/config"
)

type Balancer interface {
	Next() *backend.Backend
}

// Keep parameter order consistent with callers. This returns the interface type.
func New(backends []*backend.Backend, cfg config.LoadBalancerConfig) Balancer {
	switch cfg.Type {
	case "round-robin":
		return NewRoundRobin(backends)
	default:
		return NewRoundRobin(backends)
	}
}
