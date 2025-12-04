package proxy

import (
	"net/http"
	"time"

	"github.com/Lucascluz/reverse/internal/logger"
)

// cacheDecisionWriter is an interface the handler can use to set cache info.
type cacheDecisionWriter interface {
	SetCacheDecision(status string, reason string, uri string)
}

// responseRecorder captures status/bytes and cache decision for the middleware.
type responseRecorder struct {
	http.ResponseWriter
	status       int
	bytes        int
	cacheStatus  string
	cacheReason  string
	cacheBackend string
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *responseRecorder) SetCacheDecision(status string, reason string, backend string) {
	r.cacheStatus = status
	r.cacheReason = reason
	r.cacheBackend = backend
}

// LoggingMiddleware attaches a request logger and produces an access log after the handler.
func LoggingMiddleware(baseLogger *logger.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// request id propagation/generation
		reqID := req.Header.Get("X-Request-ID")
		if reqID == "" {
			// very small generator - swap for uuid if you prefer
			reqID = time.Now().Format("20060102150405.000")
			req.Header.Set("X-Request-ID", reqID)
		}

		// create per-request logger and add to context
		rl := baseLogger.WithRequestFields(reqID, req.Method, req.URL.Path)
		ctx := logger.NewContext(req.Context(), rl)
		req = req.WithContext(ctx)

		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w}

		// call next handler
		next.ServeHTTP(rr, req)

		// after handler completes, log an access line
		backend := rr.cacheBackend
		if backend == "" {
			backend = "-"
		}
		cacheStatus := rr.cacheStatus
		if cacheStatus == "" {
			cacheStatus = "UNKNOWN"
		}
		latencyMs := time.Since(start).Milliseconds()

		logger := logger.FromContext(ctx)
		logger.Infof("status=%d bytes=%d backend=%s cache=%s reason=%s latency_ms=%d",
			rr.status, rr.bytes, backend, cacheStatus, rr.cacheReason, latencyMs)
	})
}
