package middleware

import (
	"net/http"
	"strconv"

	"github.com/Lucascluz/reverxy/internal/ratelimiter"
)

func RateLimiting(l ratelimiter.Limiter, e *ratelimiter.Extractor, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		key := e.Extract(r)

		allowed, retryAfter := l.Allow(key)

		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
