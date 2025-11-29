package proxy

import "net/http"

func (p *Proxy) ProbeMux() http.Handler {

	mux := http.NewServeMux()

	// Liveness check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}