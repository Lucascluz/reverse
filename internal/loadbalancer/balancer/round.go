package balancer

import (
	"sync/atomic"

	"github.com/Lucascluz/reverse/internal/loadbalancer/pool"
)

type roundRobin struct {
	backends []*pool.Backend
	index    uint64 // Changed to uint64 for atomic operations
}

func NewRoundRobin(backends []*pool.Backend) *roundRobin {
	return &roundRobin{backends: backends, index: 0}
}

func (rr *roundRobin) Next() *pool.Backend {
	n := len(rr.backends)
	if n == 0 {
		return nil
	}

	val := atomic.AddUint64(&rr.index, 1)
	idx := (val - 1) % uint64(n)

	return (rr.backends)[idx]
}
