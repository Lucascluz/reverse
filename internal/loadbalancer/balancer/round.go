package balancer

import (
	"sync/atomic"

	"github.com/Lucascluz/reverse/internal/loadbalancer/pool"
)

type roundRobin struct {
	backends []*pool.Backend
	index    atomic.Int32
}

func NewRoundRobin(backends []*pool.Backend) *roundRobin {
	return &roundRobin{
		backends: backends,
		index:    atomic.Int32{}}
}

func (rr *roundRobin) Next() *pool.Backend {
	n := len(rr.backends)
	if n == 0 {
		return nil
	}

	val := rr.index.Add(1)
	idx := (val - 1) % int32(n)

	return (rr.backends)[idx]
}
