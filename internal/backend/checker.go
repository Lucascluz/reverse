package backend

import (
	"net/http"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type HealthChecker struct {
	client *http.Client

	ticker *time.Ticker

	stop chan struct{}
}

func NewHealthChecker(cfg *config.HealthCheckerConfig) *HealthChecker {
	hc := &HealthChecker{
		client: &http.Client{Timeout: cfg.Timeout},
		ticker: time.NewTicker(cfg.Interval),
		stop:   make(chan struct{}),
	}

	return hc
}

func (hc *HealthChecker) Start(backends []*Backend) {
	for {
		select {
		case <-hc.ticker.C:
			for _, backend := range backends {
				go healthCheck(hc.client, backend)
			}
		case <-hc.stop:
			hc.ticker.Stop()
			return
		}
	}
}

func (hc *HealthChecker) Stop() {
	hc.stop <- struct{}{}
}

func healthCheck(client *http.Client, backend *Backend) {
	// Lock to safely check backoff and update LastCheck
	backend.mu.Lock()
	
	// Check if backend is backed off
	if time.Now().Before(backend.LastCheck.Add(backend.BackoffTime)) {
		backend.mu.Unlock()
		return
	}
	
	backend.LastCheck = time.Now()
	
	// Unlock before http request
	backend.mu.Unlock()

	// Health check 
	resp, err := client.Get(backend.HealthUrl)
	
	// Close body if we got a response
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// Lock to update health status
	backend.mu.Lock()
	defer backend.mu.Unlock()

	// Success case
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if !backend.Healthy {
			backend.BackoffTime = 1 * time.Second
		}
		backend.Healthy = true
		return
	}

	// Failure case (either error or bad status code)
	backend.FailureCount += 1
	backend.Healthy = false

	// Exponential backoff with upper limit of 60 seconds
	if backend.BackoffTime < 60*time.Second {
		backend.BackoffTime *= 2
	}
}
