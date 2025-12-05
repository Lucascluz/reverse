package middleware

import (
	"net/http"
)

// ResponseRecorder wraps http.ResponseWriter to capture response metadata
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	cacheStatus  string
	cacheReason  string
	cacheBackend string
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default status
	}
}

// WriteHeader captures the status code before writing
func (r *ResponseRecorder) WriteHeader(status int) {
	r.statusCode = status
	r.ResponseWriter.WriteHeader(status)
}

// Write captures bytes written and writes to underlying writer
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

// StatusCode returns the captured HTTP status code
func (r *ResponseRecorder) StatusCode() int {
	return r.statusCode
}

// BytesWritten returns the total bytes written to the response
func (r *ResponseRecorder) BytesWritten() int {
	return r.bytesWritten
}

// CacheStatus returns the cache decision status
func (r *ResponseRecorder) CacheStatus() string {
	if r.cacheStatus == "" {
		return "UNKNOWN"
	}
	return r.cacheStatus
}

// CacheReason returns the reason for the cache decision
func (r *ResponseRecorder) CacheReason() string {
	return r.cacheReason
}

// CacheBackend returns the backend that served the request
func (r *ResponseRecorder) CacheBackend() string {
	if r.cacheBackend == "" {
		return "-"
	}
	return r.cacheBackend
}
