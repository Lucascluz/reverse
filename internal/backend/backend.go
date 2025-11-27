package backend

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type Backend struct {
	Url      string
	Weight   int
	MaxConns int

	Healthy         bool
	LastCheck       time.Time
	FailureCount    int
	ActiveConns     int
	TotalRequests   int
	AvgResponseTime time.Duration

	mu sync.RWMutex
}

func NewBackend(cfg config.BackendConfig) *Backend {
	return &Backend{
		Url:             cfg.Url,
		Weight:          cfg.Weight,
		MaxConns:        cfg.MaxConns,
		Healthy:         false,
		FailureCount:    0,
		ActiveConns:     0,
		TotalRequests:   0,
		AvgResponseTime: time.Duration(0),
		mu: sync.RWMutex{},
	}
}
