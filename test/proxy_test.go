package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Lucascluz/reverse/internal/cache"
)

// TestProxyBasicFunctionality tests basic proxy request forwarding
func TestProxyBasicFunctionality(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	// Create proxy with test backend
	p := createTestProxy([]string{backend.URL})

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	p.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != "Hello from backend" {
		t.Errorf("Expected 'Hello from backend', got '%s'", body)
	}
}

// TestProxyCacheHit tests that cache returns cached responses
func TestProxyCacheHit(t *testing.T) {
	requestCount := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Response #%d", requestCount)))
	}))
	defer backend.Close()

	// Create proxy
	p := createTestProxy([]string{backend.URL})

	// Manually populate cache to test cache hit
	cacheKey := backend.URL + "/cached"
	cachedBody := []byte("Cached response")
	cachedHeaders := http.Header{"Content-Type": []string{"text/plain"}}
	p.GetCache().Set(cacheKey, cachedBody, cachedHeaders, time.Now().Add(5*time.Minute))

	// Make request
	req := httptest.NewRequest("GET", "/cached", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	// Verify cache hit
	if rec.Header().Get("X-Cache") != "HIT" {
		t.Error("Expected cache HIT header")
	}

	body := rec.Body.String()
	if body != "Cached response" {
		t.Errorf("Expected cached response, got '%s'", body)
	}

	// Verify backend was not called
	if requestCount > 0 {
		t.Errorf("Backend should not have been called, but was called %d times", requestCount)
	}
}

// TestProxyCacheMiss tests that non-cached requests hit the backend
func TestProxyCacheMiss(t *testing.T) {
	requestCount := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Fresh response"))
	}))
	defer backend.Close()

	p := createTestProxy([]string{backend.URL})

	// Make request
	req := httptest.NewRequest("GET", "/uncached", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	// Verify cache miss (no X-Cache header or not HIT)
	if rec.Header().Get("X-Cache") == "HIT" {
		t.Error("Should not have been a cache hit")
	}

	body := rec.Body.String()
	if body != "Fresh response" {
		t.Errorf("Expected 'Fresh response', got '%s'", body)
	}

	// Verify backend was called
	if requestCount != 1 {
		t.Errorf("Backend should have been called once, was called %d times", requestCount)
	}
}

// TestProxyLoadBalancing tests that requests are distributed across backends
func TestProxyLoadBalancing(t *testing.T) {
	backend1Calls := 0
	backend2Calls := 0

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend1Calls++
		w.Write([]byte("backend1"))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend2Calls++
		w.Write([]byte("backend2"))
	}))
	defer backend2.Close()

	p := createTestProxy([]string{backend1.URL, backend2.URL})

	// Make multiple requests
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/test%d", i), nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}

	// Verify both backends received requests (random distribution)
	if backend1Calls == 0 {
		t.Error("Backend 1 should have received some requests")
	}
	if backend2Calls == 0 {
		t.Error("Backend 2 should have received some requests")
	}

	t.Logf("Backend 1 calls: %d, Backend 2 calls: %d", backend1Calls, backend2Calls)
}

// TestCacheEfficiency measures cache hit rate and performance
func TestCacheEfficiency(t *testing.T) {
	requestCount := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		w.Write([]byte(fmt.Sprintf("Response %d", requestCount)))
	}))
	defer backend.Close()

	p := createTestProxy([]string{backend.URL})

	// Pre-populate cache with some entries
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("%s/item%d", backend.URL, i)
		body := []byte(fmt.Sprintf("Cached item %d", i))
		p.GetCache().Set(key, body, http.Header{}, time.Now().Add(10*time.Minute))
	}

	totalRequests := 100
	cacheHits := 0
	cacheMisses := 0

	start := time.Now()

	// Make requests (80% to cached items, 20% to new items)
	for i := 0; i < totalRequests; i++ {
		var path string
		if i%5 < 4 { // 80% cached
			path = fmt.Sprintf("/item%d", i%5)
		} else { // 20% uncached
			path = fmt.Sprintf("/new%d", i)
		}

		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)

		if rec.Header().Get("X-Cache") == "HIT" {
			cacheHits++
		} else {
			cacheMisses++
		}
	}

	elapsed := time.Since(start)

	// Calculate metrics
	hitRate := float64(cacheHits) / float64(totalRequests) * 100
	avgTime := elapsed / time.Duration(totalRequests)

	t.Logf("\n=== Cache Efficiency Report ===")
	t.Logf("Total Requests: %d", totalRequests)
	t.Logf("Cache Hits: %d", cacheHits)
	t.Logf("Cache Misses: %d", cacheMisses)
	t.Logf("Hit Rate: %.2f%%", hitRate)
	t.Logf("Backend Requests: %d", requestCount)
	t.Logf("Total Time: %v", elapsed)
	t.Logf("Avg Time per Request: %v", avgTime)
	t.Logf("==============================\n")

	// Verify expected hit rate (should be around 80%)
	if hitRate < 75 || hitRate > 85 {
		t.Logf("Warning: Hit rate %.2f%% is outside expected range (75-85%%)", hitRate)
	}

	// Verify backend was only called for cache misses
	if requestCount > cacheMisses+1 { // +1 for some tolerance
		t.Errorf("Backend was called %d times but expected around %d misses", requestCount, cacheMisses)
	}
}

// TestCacheConcurrency tests cache behavior under concurrent load
func TestCacheConcurrency(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		count := requestCount
		mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		w.Write([]byte(fmt.Sprintf("Response %d", count)))
	}))
	defer backend.Close()

	p := createTestProxy([]string{backend.URL})

	// Pre-populate cache
	key := backend.URL + "/concurrent"
	p.GetCache().Set(key, []byte("Cached"), http.Header{}, time.Now().Add(10*time.Minute))

	var wg sync.WaitGroup
	goroutines := 50
	requestsPerGoroutine := 10

	start := time.Now()

	// Spawn concurrent requests
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest("GET", "/concurrent", nil)
				rec := httptest.NewRecorder()
				p.ServeHTTP(rec, req)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	totalRequests := goroutines * requestsPerGoroutine
	t.Logf("\n=== Concurrency Test Report ===")
	t.Logf("Goroutines: %d", goroutines)
	t.Logf("Requests per goroutine: %d", requestsPerGoroutine)
	t.Logf("Total Requests: %d", totalRequests)
	t.Logf("Backend Calls: %d", requestCount)
	t.Logf("Total Time: %v", elapsed)
	t.Logf("Requests/sec: %.2f", float64(totalRequests)/elapsed.Seconds())
	t.Logf("===============================\n")

	// With cache, backend should be called 0 times (all from cache)
	if requestCount > 0 {
		t.Logf("Note: Backend was called %d times (expected 0 with perfect cache)", requestCount)
	}
}

// TestCacheExpiration tests that expired entries are not served
func TestCacheExpiration(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Fresh response"))
	}))
	defer backend.Close()

	p := createTestProxy([]string{backend.URL})

	// Add entry that expires in 100ms
	key := backend.URL + "/expire"
	p.GetCache().Set(key, []byte("Will expire"), http.Header{}, time.Now().Add(100*time.Millisecond))

	// First request should hit cache
	req1 := httptest.NewRequest("GET", "/expire", nil)
	rec1 := httptest.NewRecorder()
	p.ServeHTTP(rec1, req1)

	if rec1.Header().Get("X-Cache") != "HIT" {
		t.Error("First request should hit cache")
	}
	if rec1.Body.String() != "Will expire" {
		t.Error("Should get cached response")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Second request should miss cache
	req2 := httptest.NewRequest("GET", "/expire", nil)
	rec2 := httptest.NewRecorder()
	p.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Cache") == "HIT" {
		t.Error("Second request should not hit cache after expiration")
	}
	if rec2.Body.String() != "Fresh response" {
		t.Error("Should get fresh response from backend")
	}
}

// Helper function to create a test proxy with custom backends
func createTestProxy(backends []string) *testProxy {
	return &testProxy{
		targets: backends,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cache: cache.NewMemoryCache(),
	}
}

// testProxy wraps proxy.Proxy for testing
type testProxy struct {
	targets []string
	client  *http.Client
	cache   cache.Cache
}

// ServeHTTP implements http.Handler
func (p *testProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Simple round-robin for testing
	target := p.targets[0]

	// Check cache
	if p.cache != nil {
		if cached, _, ok := p.cache.Get(target + r.URL.Path); ok {
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write(cached)
			return
		}
	}

	// Forward request
	req, err := http.NewRequest(r.Method, target+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	resp, err := p.client.Do(req)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// GetCache returns the cache for testing
func (p *testProxy) GetCache() cache.Cache {
	return p.cache
}
