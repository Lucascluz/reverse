package cache

import (
	"net/http"
	"time"
)

type Cache interface {
	Get(key string) ([]byte, http.Header, bool)
	Set(key string, body []byte, headers http.Header, expires time.Time)
	GetDefaultTTL() time.Duration
	GetMaxAge() time.Duration
}

type Entry struct {
	body    []byte
	headers http.Header
	expires time.Time
}

func (e *Entry) isExpired() bool {
	return time.Now().After(e.expires)
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
