package balancer

import (
	"sync/atomic"

	"github.com/Lucascluz/reverse/internal/backend"
)

type roundRobinBalancer struct {
	backends []*backend.Backend
	index    uint64 // Changed to uint64 for atomic operations
}

func NewRoundRobin(backends []*backend.Backend) *roundRobinBalancer {
	// Initialize at 0. The first call to Next() will increment it to 1,
	// and we subtract 1 to get index 0.
	return &roundRobinBalancer{backends: backends, index: 0}
}

func (r *roundRobinBalancer) Next() *backend.Backend {
	n := len(r.backends)
	if n == 0 {
		return nil
	}

	// Atomically increment the counter.
	val := atomic.AddUint64(&r.index, 1)

	// Calculate the actual index.
	idx := (val - 1) % uint64(n)

	return r.backends[idx]
}
