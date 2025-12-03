package cache

import (
	"net/http"
)

type Cache interface {
	Get(key string) ([]byte, http.Header, bool)
	Set(key string, body []byte, headers http.Header)
	GenKey(method string, host string, path string, headers http.Header) string
}

func cloneHeaders(headers http.Header) http.Header {
	newHeaders := make(http.Header)
	for key, values := range headers {
		newHeaders[key] = append([]string(nil), values...)
	}
	return newHeaders
}

func stripHopByHop(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	out := make(http.Header, len(h))
	for k, vv := range h {
		switch http.CanonicalHeaderKey(k) {
		case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
			"TE", "Trailer", "Transfer-Encoding", "Upgrade":
			continue
		default:
			out[k] = append([]string(nil), vv...)
		}
	}
	return out
}
