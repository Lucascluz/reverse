package observability

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
)

type HealthAware interface {
	Name() string
	HealthUrl() string
	IsBackedOff() bool
	UpdateHealth(success bool)
}

type HealthChecker struct {
	maxConcurrentChecks int
	client              *http.Client
	ticker              *time.Ticker
	stop                chan struct{}
}

func NewHealthChecker(cfg *config.HealthCheckerConfig) *HealthChecker {

	// Defensive defaults: fallback to config package defaults when tests left values zero
	var interval, timeout time.Duration
	var maxConcurrentChecks int
	if cfg == nil {
		interval = config.DefaultInterval
		timeout = config.DefaultTimeout
		maxConcurrentChecks = config.DefaultMaxConcurrentChecks
	} else {
		interval = cfg.Interval
		timeout = cfg.Timeout
		if interval <= 0 {
			interval = config.DefaultInterval
		}
		if timeout <= 0 {
			timeout = config.DefaultTimeout
		}
		if cfg.MaxConcurrentChecks <= 0 {
			maxConcurrentChecks = config.DefaultMaxConcurrentChecks
		}
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   2 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			MaxIdleConns:          20,
			MaxIdleConnsPerHost:   4,
			IdleConnTimeout:       5 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
		},
	}

	return &HealthChecker{
		maxConcurrentChecks: maxConcurrentChecks,
		client:              client,
		ticker:              time.NewTicker(interval),
		stop:                make(chan struct{}),
	}
}

func (hc *HealthChecker) Start(backends []HealthAware, updateReady func()) {

	fmt.Fprintf(os.Stderr, "[HEALTH] Starting checks for %d backends\n", len(backends))

	// Run initial health checks immediately before waiting for ticker
	doHealthChecks := func() {

		os.Stderr.Sync()

		// Run health checks synchronously for now to ensure they complete
		for _, b := range backends {

			os.Stderr.Sync()
			healthCheck(hc.client, b)

			os.Stderr.Sync()
		}

		os.Stderr.Sync()

		// Update proxy readyness during health check
		if updateReady != nil {
			updateReady()
		}
	}

	// Execute immediate health check
	doHealthChecks()

	for {
		select {
		case <-hc.ticker.C:
			doHealthChecks()

		case <-hc.stop:
			hc.ticker.Stop()
			return
		}
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stop)
}

func healthCheck(client *http.Client, backend HealthAware) {

	// If backend is backed off, abort current health check
	if backend.IsBackedOff() {
		return
	}

	// Health check request
	resp, err := client.Get(backend.HealthUrl())

	// Close body if we got a response
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// Success case
	success := (err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300)

	if success {
		fmt.Fprintf(os.Stderr, "[HEALTH] %s is HEALTHY (status %d)\n", backend.Name(), resp.StatusCode)
	} else {
		if err != nil {
			fmt.Fprintf(os.Stderr, "[HEALTH] %s FAILED: %v\n", backend.Name(), err)
		} else {
			fmt.Fprintf(os.Stderr, "[HEALTH] %s FAILED: status %d\n", backend.Name(), resp.StatusCode)
		}
	}

	backend.UpdateHealth(success)
}
