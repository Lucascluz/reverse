package observability

import (
	"fmt"
	"net/http"
	"os"
	"time"
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
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness check
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		ready := p.ReadyAware.IsReady()
		fmt.Fprintf(os.Stderr, "[Probe] /readyz check: ready=%v\n", ready)

		if ready {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT OK"))
		}
	})

	return mux
}
