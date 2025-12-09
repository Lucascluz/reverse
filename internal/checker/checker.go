package checker

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/backend"
	"github.com/Lucascluz/reverse/internal/config"
)

type HealthChecker struct {
	maxConcurrentChecks int

	client *http.Client
	ticker *time.Ticker
	stop   chan struct{}
}

func New(cfg *config.HealthCheckerConfig) *HealthChecker {

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

func (hc *HealthChecker) Start(backends []*backend.Backend, updateReady func()) {

	// Semaphore concurrent checks
	sem := make(chan struct{}, hc.maxConcurrentChecks)

	for {
		select {
		case <-hc.ticker.C:

			var wg sync.WaitGroup

			for _, b := range backends {
				wg.Add(1)

				go func(backend *backend.Backend) {
					defer wg.Done()

					// Claim a spot
					sem <- struct{}{}

					// Release spot when done
					defer func() { <-sem }()

					healthCheck(hc.client, backend)
				}(b)
			}

			// Update proxy readyness during health check
			if updateReady != nil {
				updateReady()
			}

		case <-hc.stop:
			hc.ticker.Stop()
			return
		}
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stop)
}

func healthCheck(client *http.Client, backend *backend.Backend) {

	// If backend is backed off, abort current health check
	if backend.IsBackedOff() {
		return
	}

	// Health check request
	resp, err := client.Get(backend.HealthUrl)

	// Close body if we got a response
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// Success case
	success := (err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300)

	backend.UpdateHealth(success)
}
