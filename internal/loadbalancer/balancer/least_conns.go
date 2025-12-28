package balancer

import (
	"github.com/Lucascluz/reverxy/internal/loadbalancer/pool"
)

type leastConns struct {
	backends []*pool.Backend
}

func NewLeastConns(backends []*pool.Backend) *leastConns {
	return &leastConns{
		backends: backends,
	}
}

func (lc *leastConns) Next() *pool.Backend {
	n := len(lc.backends)
	if n == 0 {
		return nil
	}

	var least *pool.Backend
	for _, backend := range lc.backends {
		if least == nil || backend.ActiveConns() < least.ActiveConns() {
			least = backend
		}
	}

	return least
}
