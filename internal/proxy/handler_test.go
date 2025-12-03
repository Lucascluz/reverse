package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lucascluz/reverse/internal/backend"
	"github.com/Lucascluz/reverse/internal/cache"
	"github.com/Lucascluz/reverse/internal/config"
)

// TestIsCachable tests the isCachable function
func TestIsCachable(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		status  int
		headers http.Header
		want    bool
	}{
		{
			name:    "GET request with 200 status",
			method:  "GET",
			status:  200,
			headers: http.Header{},
			want:    true,
		},
		{
			name:    "HEAD request with 200 status",
			method:  "HEAD",
			status:  200,
			headers: http.Header{},
			want:    true,
		},
		{
			name:    "POST request should not be cached",
			method:  "POST",
			status:  200,
			headers: http.Header{},
			want:    false,
		},
		{
			name:    "PUT request should not be cached",
			method:  "PUT",
			status:  200,
			headers: http.Header{},
			want:    false,
		},
		{
			name:    "404 status should not be cached",
			method:  "GET",
			status:  404,
			headers: http.Header{},
			want:    false,
		},
		{
			name:    "500 status should not be cached",
			method:  "GET",
			status:  500,
			headers: http.Header{},
			want:    false,
		},
		{
			name:   "Cache-Control: no-store should not be cached",
			method: "GET",
			status: 200,
			headers: http.Header{
				"Cache-Control": []string{"no-store"},
			},
			want: false,
		},
		{
			name:   "Cache-Control: private should not be cached",
			method: "GET",
			status: 200,
			headers: http.Header{
				"Cache-Control": []string{"private"},
			},
			want: false,
		},
		{
			name:   "Cache-Control: public should be cached",
			method: "GET",
			status: 200,
			headers: http.Header{
				"Cache-Control": []string{"public, max-age=3600"},
			},
			want: true,
		},
		{
			name:   "Response with Set-Cookie should not be cached",
			method: "GET",
			status: 200,
			headers: http.Header{
				"Set-Cookie": []string{"session=abc123"},
			},
			want: false,
		},
		{
			name:    "206 Partial Content should be cached",
			method:  "GET",
			status:  206,
			headers: http.Header{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCachable(tt.method, tt.status, tt.headers)
			if got != tt.want {
				t.Errorf("isCachable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseMaxAge tests the parseMaxAge function
func TestParseMaxAge(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl string
		want         time.Duration
	}{
		{
			name:         "simple max-age",
			cacheControl: "max-age=3600",
			want:         3600 * time.Second,
		},
		{
			name:         "max-age with other directives",
			cacheControl: "public, max-age=7200, must-revalidate",
			want:         7200 * time.Second,
		},
		{
			name:         "max-age with spaces",
			cacheControl: "max-age = 1800",
			want:         0, // Should not parse with spaces around =
		},
		{
			name:         "no max-age directive",
			cacheControl: "public, must-revalidate",
			want:         0,
		},
		{
			name:         "invalid max-age value",
			cacheControl: "max-age=invalid",
			want:         0,
		},
		{
			name:         "negative max-age",
			cacheControl: "max-age=-100",
			want:         0,
		},
		{
			name:         "zero max-age",
			cacheControl: "max-age=0",
			want:         0,
		},
		{
			name:         "empty cache control",
			cacheControl: "",
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMaxAge(tt.cacheControl)
			if got != tt.want {
				t.Errorf("parseMaxAge() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetermineTTL tests the determineTTL function
func TestDetermineTTL(t *testing.T) {
	// Create a mock cache
	mockCache := cache.NewMemoryCache(&config.CacheConfig{
		Disabled:      false,
		DefaultTTL:    5 * time.Minute,
		MaxAge:        1 * time.Hour,
		PurgeInterval: 10 * time.Minute,
	})

	p := &Proxy{
		cache: mockCache,
	}

	tests := []struct {
		name    string
		headers http.Header
		want    time.Duration
	}{
		{
			name: "Cache-Control max-age within limit",
			headers: http.Header{
				"Cache-Control": []string{"max-age=1800"},
			},
			want: 30 * time.Minute,
		},
		{
			name: "Cache-Control max-age exceeds MaxAge",
			headers: http.Header{
				"Cache-Control": []string{"max-age=7200"},
			},
			want: 1 * time.Hour, // Capped at MaxAge
		},
		{
			name: "Expires header in future",
			headers: http.Header{
				"Expires": []string{time.Now().Add(10 * time.Minute).Format(http.TimeFormat)},
			},
			want: 10 * time.Minute,
		},
		{
			name: "Expires header way in future (exceeds MaxAge)",
			headers: http.Header{
				"Expires": []string{time.Now().Add(5 * time.Hour).Format(http.TimeFormat)},
			},
			want: 1 * time.Hour, // Capped at MaxAge
		},
		{
			name: "Expires header in past",
			headers: http.Header{
				"Expires": []string{time.Now().Add(-10 * time.Minute).Format(http.TimeFormat)},
			},
			want: 5 * time.Minute, // Should use default TTL
		},
		{
			name:    "No cache headers",
			headers: http.Header{},
			want:    5 * time.Minute, // Default TTL
		},
		{
			name: "Cache-Control takes precedence over Expires",
			headers: http.Header{
				"Cache-Control": []string{"max-age=600"},
				"Expires":       []string{time.Now().Add(30 * time.Minute).Format(http.TimeFormat)},
			},
			want: 10 * time.Minute, // Uses max-age, not Expires
		},
		{
			name: "Invalid Expires format",
			headers: http.Header{
				"Expires": []string{"invalid-date"},
			},
			want: 5 * time.Minute, // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.determineTTL(tt.headers)

			// Allow small delta for time-based tests (within 1 second)
			delta := got - tt.want
			if delta < 0 {
				delta = -delta
			}
			if delta > time.Second {
				t.Errorf("determineTTL() = %v, want %v (delta: %v)", got, tt.want, delta)
			}
		})
	}
}

// TestServeHTTP_CacheHit tests cache hit scenario
func TestServeHTTP_CacheHit(t *testing.T) {
	// Create a backend that should NOT be called
	backendCalled := false
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		t.Error("Backend should not be called on cache hit")
	}))
	defer backendServer.Close()

	// Create proxy with cache
	mockCache := cache.NewMemoryCache(&config.CacheConfig{
		Disabled:      false,
		DefaultTTL:    5 * time.Minute,
		MaxAge:        1 * time.Hour,
		PurgeInterval: 10 * time.Minute,
	})

	// Create a backend pool with the test backend
	pool := backend.NewPool(&config.PoolConfig{
		Backends: []config.BackendConfig{
			{Url: backendServer.URL, Weight: 1, MaxConns: 100},
		},
	}, func(){})

	p := &Proxy{
		pool:   pool,
		client: &http.Client{},
		cache:  mockCache,
	}

	// Pre-populate cache
	cacheKey := "GET:/test"
	cachedBody := []byte("cached response")
	cachedHeaders := http.Header{
		"Content-Type": []string{"text/plain"},
		"X-Custom":     []string{"cached"},
	}
	mockCache.Set(cacheKey, cachedBody, cachedHeaders, time.Now().Add(10*time.Minute))

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("X-Cache") != "HIT" {
		t.Error("Expected X-Cache: HIT header")
	}

	if rec.Body.String() != "cached response" {
		t.Errorf("Expected cached response, got %q", rec.Body.String())
	}

	if backendCalled {
		t.Error("Backend should not be called on cache hit")
	}

	// Verify headers were copied
	if rec.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type header to be copied")
	}
}

// TestServeHTTP_CacheMiss tests cache miss scenario
func TestServeHTTP_CacheMiss(t *testing.T) {
	// Create a backend
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=300")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"hello"}`))
	}))
	defer backendServer.Close()

	// Create proxy with cache
	mockCache := cache.NewMemoryCache(&config.CacheConfig{
		Disabled:      false,
		DefaultTTL:    5 * time.Minute,
		MaxAge:        1 * time.Hour,
		PurgeInterval: 10 * time.Minute,
	})

	// Create a backend pool with the test backend
	pool := backend.NewPool(&config.PoolConfig{
		Backends: []config.BackendConfig{
			{Url: backendServer.URL, Weight: 1, MaxConns: 100},
		},
	}, func() {})

	p := &Proxy{
		pool:   pool,
		client: &http.Client{},
		cache:  mockCache,
	}

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != `{"message":"hello"}` {
		t.Errorf("Expected backend response, got %q", rec.Body.String())
	}

	// Verify response was cached
	cacheKey := "GET:/test"
	if cached, _, ok := mockCache.Get(cacheKey); !ok {
		t.Error("Response should have been cached")
	} else if string(cached) != `{"message":"hello"}` {
		t.Errorf("Cached wrong content: %q", string(cached))
	}
}

// TestServeHTTP_NonCachableMethod tests that POST requests are not cached
func TestServeHTTP_NonCachableMethod(t *testing.T) {
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer backendServer.Close()

	mockCache := cache.NewMemoryCache(&config.CacheConfig{
		Disabled:      false,
		DefaultTTL:    5 * time.Minute,
		MaxAge:        1 * time.Hour,
		PurgeInterval: 10 * time.Minute,
	})

	// Create a backend pool with the test backend
	pool := backend.NewPool(&config.PoolConfig{
		Backends: []config.BackendConfig{
			{Url: backendServer.URL, Weight: 1, MaxConns: 100},
		},
	}, func() {})

	p := &Proxy{
		pool:   pool,
		client: &http.Client{},
		cache:  mockCache,
	}

	// Make POST request
	req := httptest.NewRequest("POST", "/test", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	// Verify response was NOT cached
	cacheKey := "POST:/test"
	if _, _, ok := mockCache.Get(cacheKey); ok {
		t.Error("POST request should not be cached")
	}
}

// TestServeHTTP_CacheKeyGeneration tests that the cache key is generated correctly
func TestServeHTTP_CacheKeyGeneration(t *testing.T) {
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer backendServer.Close()

	mockCache := cache.NewMemoryCache(&config.CacheConfig{
		Disabled:      false,
		DefaultTTL:    5 * time.Minute,
		MaxAge:        1 * time.Hour,
		PurgeInterval: 10 * time.Minute,
	})

	pool := backend.NewPool(&config.PoolConfig{
		Backends: []config.BackendConfig{
			{Url: backendServer.URL, Weight: 1, MaxConns: 100},
		},
	}, func() {})

	p := &Proxy{
		pool:   pool,
		client: &http.Client{},
		cache:  mockCache,
	}

	// 1. Request to a URL without a query string
	req1 := httptest.NewRequest("GET", "/test", nil)
	rec1 := httptest.NewRecorder()
	p.ServeHTTP(rec1, req1)

	cacheKey1 := "GET:/test"
	if _, _, ok := mockCache.Get(cacheKey1); !ok {
		t.Errorf("Expected cache entry for key %q, but not found", cacheKey1)
	}

	// 2. Request to a URL with a query string
	req2 := httptest.NewRequest("GET", "/test?param=1", nil)
	rec2 := httptest.NewRecorder()
	p.ServeHTTP(rec2, req2)

	cacheKey2 := "GET:/test?param=1"
	if _, _, ok := mockCache.Get(cacheKey2); !ok {
		t.Errorf("Expected cache entry for key %q, but not found", cacheKey2)
	}

	// 3. Second request to the same URL without a query string (should be a HIT)
	req3 := httptest.NewRequest("GET", "/test", nil)
	rec3 := httptest.NewRecorder()
	p.ServeHTTP(rec3, req3)
	if rec3.Header().Get("X-Cache") != "HIT" {
		t.Errorf("Expected cache HIT for key %q, but got MISS", cacheKey1)
	}

	// 4. Second request to the same URL with a query string (should be a HIT)
	req4 := httptest.NewRequest("GET", "/test?param=1", nil)
	rec4 := httptest.NewRecorder()
	p.ServeHTTP(rec4, req4)
	if rec4.Header().Get("X-Cache") != "HIT" {
		t.Errorf("Expected cache HIT for key %q, but got MISS", cacheKey2)
	}

	// 5. Request to a URL with a different query string (should be a MISS)
	req5 := httptest.NewRequest("GET", "/test?param=2", nil)
	rec5 := httptest.NewRecorder()
	p.ServeHTTP(rec5, req5)
	if rec5.Header().Get("X-Cache") == "HIT" {
		t.Errorf("Expected cache MISS for new query string, but got HIT")
	}
	cacheKey5 := "GET:/test?param=2"
	if _, _, ok := mockCache.Get(cacheKey5); !ok {
		t.Errorf("Expected cache entry for key %q, but not found", cacheKey5)
	}
}