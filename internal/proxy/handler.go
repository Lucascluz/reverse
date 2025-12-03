package proxy

import (
	"io"
	"net/http"
	"strings"
)

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

// Implement http.Handler interface directly
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// TODO: Implement proper load balancing strategy
	nextTarget := p.pool.NextUrl()

	// Generate cache key
	key := p.cache.GenKey(r.Method, r.URL.Host, r.URL.Path, r.Header)

	// If cache is enabled, check if the requested resource is cached
	if p.cache != nil {

		// Serve cached response
		if cached, headers, ok := p.cache.Get(key); ok {

			copyHeader(w.Header(), headers)
			w.Header().Set("X-Cache", "HIT")

			// inform middleware about cache decision (if wrapped)
			if cdw, ok := w.(interface{ SetCacheDecision(string, string, string) }); ok {
				cdw.SetCacheDecision("HIT", "cached entry", "")
			}

			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}
	}

	outReq, err := http.NewRequest(r.Method, nextTarget+r.URL.Path, r.Body)
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

	// Determine if should cache the response body
	cachable := p.cache != nil && isCachable(r.Method, resp.StatusCode, resp.Header)

	if cachable {
		// Store cache entry
		p.cache.Set(key, body, resp.Header)

		// inform middleware that this response was stored in cache
		if cdw, ok := w.(interface{ SetCacheDecision(string, string, string) }); ok {
			cdw.SetCacheDecision("MISS", "stored", nextTarget)
		}
	}
}

func isCachable(method string, status int, headers http.Header) bool {

	// TODO: Reason about caching other methods
	// Only cache GET and HEAD requests
	if method != "GET" && method != "HEAD" {
		return false
	}

	// TODO: Handle 404 and permanently redirection
	// Only cache sucess responses
	if status < 200 || status >= 300 {
		return false
	}

	// Check for cache control diretives
	if cc := headers.Get("Cache-Control"); cc != "" {
		cc = strings.ToLower(cc)

		// Only cache allowed responses
		if strings.Contains(cc, "no-store") || strings.Contains(cc, "private") {
			return false
		}
	}

	// Don't cache responses with Vary: *
	if vary := headers.Get("Vary"); vary != "" && strings.Contains(vary, "*") {
		return false
	}

	// Don't cache responses with Set-Cookie
	if headers.Get("Set-Cookie") != "" {
		return false
	}

	return true
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
