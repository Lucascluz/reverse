package backend

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck_Success(t *testing.T) {
	// Create a test server that returns 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a backend
	backend := &Backend{
		Url:         server.URL,
		HealthUrl:   server.URL + "/health",
		Healthy:     false,
		BackoffTime: 1 * time.Second,
	}

	// Create a client and perform health check
	client := &http.Client{Timeout: 5 * time.Second}
	healthCheck(client, backend)

	// Verify backend is marked healthy
	backend.mu.RLock()
	defer backend.mu.RUnlock()

	if !backend.Healthy {
		t.Error("Expected backend to be healthy after successful check")
	}

	if backend.BackoffTime != 1*time.Second {
		t.Errorf("Expected BackoffTime to be 1s, got %v", backend.BackoffTime)
	}
}

func TestHealthCheck_Failure(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a backend
	backend := &Backend{
		Url:          server.URL,
		HealthUrl:    server.URL + "/health",
		Healthy:      true,
		BackoffTime:  1 * time.Second,
		FailureCount: 0,
	}

	// Create a client and perform health check
	client := &http.Client{Timeout: 5 * time.Second}
	healthCheck(client, backend)

	// Verify backend is marked unhealthy
	backend.mu.RLock()
	defer backend.mu.RUnlock()

	if backend.Healthy {
		t.Error("Expected backend to be unhealthy after failed check")
	}

	if backend.FailureCount != 1 {
		t.Errorf("Expected FailureCount to be 1, got %d", backend.FailureCount)
	}

	if backend.BackoffTime != 2*time.Second {
		t.Errorf("Expected BackoffTime to be 2s after failure, got %v", backend.BackoffTime)
	}
}
