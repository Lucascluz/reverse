package backend

import (
	"math/rand"
	"sync"

	"github.com/Lucascluz/reverse/internal/config"
)

type Pool struct {
	backends []*Backend
	mu       sync.RWMutex
}

func NewPool(cfg *config.PoolConfig) *Pool {

	backends := make([]*Backend, len(cfg.Backends))
	for i, backendCfg := range cfg.Backends {
		backends[i] = NewBackend(backendCfg)
	}

	return &Pool{
		backends: backends,
		mu:       sync.RWMutex{},
	}
}

func (p *Pool) NextUrl() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.backends[rand.Intn(len(p.backends))].Url
}
