package proxy

import (
	"io"
	"net/http"
	"strings"
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

	// TODO: Implement proper load balancing strategy
	nextTarget := p.pool.NextUrl()

	outReq, err := http.NewRequest(r.Method, nextTarget + r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Copy headers but STRIP hop-by-hop headers
	copyHeader(outReq.Header, r.Header)

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
		if cw, ok := w.(cacheDecisionWriter); ok {
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
