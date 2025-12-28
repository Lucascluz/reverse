package observability

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ReadyAware interface {
	IsReady() bool
}

type Probe struct {
	client     *http.Client
	ReadyAware ReadyAware
}

func NewProbe(readyAware ReadyAware) *Probe {

	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   1 * time.Second,
	}

	return &Probe{
		client:     client,
		ReadyAware: readyAware,
	}
}

func (p *Probe) Handler() http.Handler {

	mux := http.NewServeMux()

	// Liveness check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness check - always return OK unless proxy is explicitly shutting down
	// Readiness check - consult the provided ReadyAware component
	// Return 200 when ReadyAware reports ready, 503 otherwise. This lets
	// the probe reflect whether the load balancer has at least one healthy
	// backend available.
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if p.ReadyAware != nil && p.ReadyAware.IsReady() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT_READY"))
	})

	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
