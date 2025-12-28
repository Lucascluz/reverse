package balancer

import (
	"math/rand"
	"sync/atomic"

	"github.com/Lucascluz/reverxy/internal/loadbalancer/pool"
)

type randomWeight struct {
	backends []*pool.Backend
	index    atomic.Int32
}

func NewRandomWeight(backends []*pool.Backend) *randomWeight {
	return &randomWeight{
		backends: backends,
		index:    atomic.Int32{}}
}

func (rw *randomWeight) Next() *pool.Backend {
	n := len(rw.backends)
	if n == 0 {
		return nil
	}

	// select a N random number of between 1 and half of total backends
	randomN := rand.Intn(n/2) + 1

	// select N random backends and return the biggest weight one
	var selected *pool.Backend
	for range randomN {
		idx := rand.Intn(n)
		backend := rw.backends[idx]

		if selected == nil || backend.Weight() > selected.Weight() {
			selected = backend
		}
	}

	return selected
}
