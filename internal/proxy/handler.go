package proxy

import (
	"io"
	"net/http"
	"strings"

	"github.com/Lucascluz/reverse/internal/middleware"
)

// Implement http.Handler interface directly
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Try to serve from cache
	if p.cache != nil {
		hit, cached := p.tryServingCachedResponse(r)

		if hit {
			// Write response body to client
			w.WriteHeader(cached.StatusCode)
			w.Write(cached.Body)
			return
		}
	}

	// Check if load balancer is ready
	if !p.loadBalancer.IsReady() {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Get next backend from load balancer
	backend, err := p.loadBalancer.Next()
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// Create new request with backend URL and original request details
	outReq, err := http.NewRequest(r.Method, backend.Url+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Copy headers but STRIP hop-by-hop headers
	copyHeader(outReq.Header, r.Header)

	// Increment backend connection count
	backend.IncrementConnections()
	defer backend.DecrementConnections()

	resp, err := p.client.Do(outReq)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response body (needed for caching)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading backend response", http.StatusBadGateway)
		return
	}

	// Copy response headers (stripping hop-by-hop again)
	copyHeader(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	if p.cache != nil {
		cached, reason := p.tryCachingResponse(r, resp.StatusCode, resp.Header, body)

		// Notify middleware of cache decision
		if cw, ok := w.(middleware.CacheDecisionWriter); ok {
			if cached {
				cw.SetCacheDecision("CACHED", reason, r.RequestURI)
			} else {
				cw.SetCacheDecision("NOT_CACHED", reason, r.RequestURI)
			}
		}
	}
}

// Helper to copy headers while skipping hop-by-hop ones
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		if isHopHeader(k) {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func isHopHeader(header string) bool {
	for _, h := range hopHeaders {
		if strings.EqualFold(h, header) {
			return true
		}
	}
	return false
}

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}
