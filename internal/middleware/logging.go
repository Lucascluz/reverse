package middleware

import (
	"net/http"
	"time"

	"github.com/Lucascluz/reverse/internal/logger"
)

// CacheDecisionWriter allows handlers to communicate cache decisions to middleware
type CacheDecisionWriter interface {
	SetCacheDecision(status, reason, backend string)
}

// SetCacheDecision implements CacheDecisionWriter interface
func (r *ResponseRecorder) SetCacheDecision(status, reason, backend string) {
	r.cacheStatus = status
	r.cacheReason = reason
	r.cacheBackend = backend
}

// Logging wraps an HTTP handler with request/response logging
func Logging(baseLogger *logger.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or propagate request ID
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
			r.Header.Set("X-Request-ID", reqID)
		}

		// Create request-scoped logger and add to context
		requestLogger := baseLogger.WithRequestFields(reqID, r.Method, r.URL.Path)
		ctx := logger.LoggerToContext(r.Context(), requestLogger)
		r = r.WithContext(ctx)

		// Wrap response writer to capture metadata
		recorder := NewResponseRecorder(w)
		start := time.Now()

		// Call next handler in chain
		next.ServeHTTP(recorder, r)

		// Log access line with collected metadata
		latencyMs := time.Since(start).Milliseconds()
		requestLogger.Infof(
			"status=%d bytes=%d backend=%s cache=%s reason=%q latency_ms=%d",
			recorder.StatusCode(),
			recorder.BytesWritten(),
			recorder.CacheBackend(),
			recorder.CacheStatus(),
			recorder.CacheReason(),
			latencyMs,
		)
	})
}

// generateRequestID creates a simple timestamp-based request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}