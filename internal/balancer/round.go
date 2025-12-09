package balancer

import (
	"sync/atomic"

	"github.com/Lucascluz/reverse/internal/backend"
)

type roundRobin struct {
	backends []*backend.Backend
	index    uint64 // Changed to uint64 for atomic operations
}

func NewRoundRobin(backends []*backend.Backend) *roundRobin {
	return &roundRobin{backends: backends, index: 0}
}

func (rr *roundRobin) Next() *backend.Backend {
	n := len(rr.backends)
	if n == 0 {
		return nil
	}

	val := atomic.AddUint64(&rr.index, 1)
	idx := (val - 1) % uint64(n)

	return (rr.backends)[idx]
}
