package backend

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type Backend struct {
	Name            string
	Url             string
	HealthUrl       string
	Weight          int
	MaxConns        int
	Healthy         bool
	LastCheck       time.Time
	FailureCount    int
	BackoffTime     time.Duration
	ActiveConns     int
	TotalRequests   int
	AvgResponseTime time.Duration

	mu sync.RWMutex
}

func NewBackend(cfg config.BackendConfig) *Backend {
	return &Backend{
		Name:            cfg.Name,
		Url:             cfg.Url,
		HealthUrl:       cfg.HealthUrl,
		Weight:          cfg.Weight,
		MaxConns:        cfg.MaxConns,
		Healthy:         false,
		LastCheck:       time.Now(),
		FailureCount:    0,
		BackoffTime:     1 * time.Second,
		ActiveConns:     0,
		TotalRequests:   0,
		AvgResponseTime: time.Duration(0),
		mu:              sync.RWMutex{},
	}
}
