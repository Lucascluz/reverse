package proxy

import (
	"bytes"
	"io"
	"math/rand/v2"
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
	nextTarget := p.targets[rand.IntN(len(p.targets))]

	// If cache is enabled, check if the requested resource is cached
	if p.cache != nil {

		if cached, _, ok := p.cache.Get(nextTarget + r.URL.Path); ok {
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			io.Copy(w, bytes.NewReader(cached))
			return
		}
	}

	outReq, err := http.NewRequest(r.Method, nextTarget+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 1. Copy headers but STRIP hop-by-hop headers
	copyHeader(outReq.Header, r.Header)

	// 2. Use YOUR client, not DefaultClient
	resp, err := p.client.Do(outReq)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 3. Copy response headers (stripping hop-by-hop again)
	copyHeader(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
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
