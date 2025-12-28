package observability

import (
	"fmt"

	"github.com/Lucascluz/reverxy/internal/config"
)

// Observability is a setup hub that manages all observability components
type Observability struct {
	logger        *Logger
	probe         *Probe
	healthChecker *HealthChecker
}

// NewObservability creates and initializes all observability components
// It requires:
// - config: The application configuration
// - readyAware: An object that implements ReadyAware interface (typically the LoadBalancer)
// - backends: A slice of HealthAware backends to monitor
func NewObservability(
	cfg *config.Config,
	readyAware ReadyAware,
	backends []HealthAware,
) (*Observability, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if readyAware == nil {
		return nil, fmt.Errorf("readyAware cannot be nil")
	}

	if backends == nil {
		return nil, fmt.Errorf("backends cannot be nil")
	}

	// Create logger
	logger := NewLogger("observability")

	// Create probe with the ready-aware component (LoadBalancer or Proxy)
	probe := NewProbe(readyAware)

	// Create health checker from config
	healthChecker := NewHealthChecker(&cfg.LoadBalancer.Pool.HealthChecker)

	obs := &Observability{
		logger:        logger,
		probe:         probe,
		healthChecker: healthChecker,
	}

	return obs, nil
}

// Logger returns the observability logger instance
func (o *Observability) Logger() *Logger {
	return o.logger
}

// Probe returns the observability probe instance (health check handler)
func (o *Observability) Probe() *Probe {
	return o.probe
}

// HealthChecker returns the observability health checker instance
func (o *Observability) HealthChecker() *HealthChecker {
	return o.healthChecker
}

// StartHealthChecks starts the health checking routine with the provided backends
// and a callback function that is invoked when health status changes
func (o *Observability) StartHealthChecks(backends []HealthAware, onReadyChanged func()) error {
	if backends == nil {
		return fmt.Errorf("backends cannot be nil")
	}

	if len(backends) == 0 {
		return fmt.Errorf("at least one backend must be provided")
	}

	// Start health checker in background goroutine
	go o.healthChecker.Start(backends, onReadyChanged)

	return nil
}

// Stop gracefully stops all observability components
func (o *Observability) Stop() error {
	if o.healthChecker != nil {
		o.healthChecker.Stop()
	}
	return nil
}
