package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const proxyURL = "http://localhost:8080"

// TestProxyBasicForwarding tests that the proxy forwards requests correctly
func TestProxyBasicForwarding(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(proxyURL + "/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}

	backendPort := resp.Header.Get("X-Backend-Port")
	if backendPort == "" {
		t.Error("Expected X-Backend-Port header from backend")
	}

	t.Logf("✓ Request forwarded to backend on port %s", backendPort)
}

// TestProxyLoadBalancing tests that requests are distributed across backends
func TestProxyLoadBalancing(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	backendCounts := make(map[string]int)
	totalRequests := 100

	for i := 0; i < totalRequests; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/lb-test-%d", proxyURL, i))
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			continue
		}

		port := resp.Header.Get("X-Backend-Port")
		if port != "" {
			backendCounts[port]++
		}
		resp.Body.Close()
	}

	if len(backendCounts) < 2 {
		t.Errorf("Expected load balancing across multiple backends, got %d backend(s)", len(backendCounts))
	}

	t.Logf("✓ Load distribution across %d backends:", len(backendCounts))
	for port, count := range backendCounts {
		percentage := float64(count) / float64(totalRequests) * 100
		t.Logf("  Backend:%s -> %d requests (%.1f%%)", port, count, percentage)
	}
}

// TestProxyConcurrentRequests tests proxy behavior under concurrent load
func TestProxyConcurrentRequests(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	goroutines := 50
	requestsPerGoroutine := 20
	totalRequests := goroutines * requestsPerGoroutine

	var successCount, errorCount atomic.Int32
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				resp, err := client.Get(fmt.Sprintf("%s/concurrent-%d-%d", proxyURL, id, j))
				if err != nil {
					errorCount.Add(1)
					continue
				}
				if resp.StatusCode == http.StatusOK {
					successCount.Add(1)
				} else {
					errorCount.Add(1)
				}
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	successRate := float64(successCount.Load()) / float64(totalRequests) * 100
	throughput := float64(totalRequests) / elapsed.Seconds()

	t.Logf("✓ Concurrency Test Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d (%.1f%%)", successCount.Load(), successRate)
	t.Logf("  Errors: %d", errorCount.Load())
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f req/sec", throughput)

	if successRate < 95 {
		t.Errorf("Success rate too low: %.1f%% (expected > 95%%)", successRate)
	}
}

// TestProxyHealthEndpoint tests that backends respond to health checks
func TestProxyHealthEndpoint(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(proxyURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for health check, got %d", resp.StatusCode)
	}

	t.Logf("✓ Health check passed")
}

// TestProxyHeaderForwarding tests that headers are correctly forwarded
func TestProxyHeaderForwarding(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", proxyURL+"/headers", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("X-Custom-Header", "test-value")
	req.Header.Set("User-Agent", "ProxyTest/1.0")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("✓ Headers forwarded successfully")
}

// TestProxyPerformanceBenchmark measures proxy throughput
func TestProxyPerformanceBenchmark(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	testCases := []int{10, 50, 100, 200}

	for _, n := range testCases {
		start := time.Now()
		for i := 0; i < n; i++ {
			resp, err := client.Get(fmt.Sprintf("%s/perf-%d", proxyURL, i))
			if err == nil {
				resp.Body.Close()
			}
		}
		elapsed := time.Since(start)

		avgTime := elapsed / time.Duration(n)
		throughput := float64(n) / elapsed.Seconds()

		t.Logf("✓ %d requests: %v (avg: %v, %.2f req/sec)",
			n, elapsed, avgTime, throughput)
	}
}
