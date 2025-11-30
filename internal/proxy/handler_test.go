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

// TestIsHopHeader tests the isHopHeader function
func TestIsHopHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "Connection", header: "Connection", want: true},
		{name: "connection lowercase", header: "connection", want: true},
		{name: "Keep-Alive", header: "Keep-Alive", want: true},
		{name: "Proxy-Authenticate", header: "Proxy-Authenticate", want: true},
		{name: "Proxy-Authorization", header: "Proxy-Authorization", want: true},
		{name: "Te", header: "Te", want: true},
		{name: "Trailers", header: "Trailers", want: true},
		{name: "Transfer-Encoding", header: "Transfer-Encoding", want: true},
		{name: "Upgrade", header: "Upgrade", want: true},
		{name: "Content-Type", header: "Content-Type", want: false},
		{name: "Content-Length", header: "Content-Length", want: false},
		{name: "User-Agent", header: "User-Agent", want: false},
		{name: "Cache-Control", header: "Cache-Control", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHopHeader(tt.header)
			if got != tt.want {
				t.Errorf("isHopHeader(%q) = %v, want %v", tt.header, got, tt.want)
			}
		})
	}
}

// TestCopyHeader tests the copyHeader function
func TestCopyHeader(t *testing.T) {
	tests := []struct {
		name     string
		src      http.Header
		wantCopy map[string][]string
		wantSkip []string
	}{
		{
			name: "copy regular headers",
			src: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"test-agent"},
				"X-Custom":     []string{"custom-value"},
			},
			wantCopy: map[string][]string{
				"Content-Type": {"application/json"},
				"User-Agent":   {"test-agent"},
				"X-Custom":     {"custom-value"},
			},
		},
		{
			name: "skip hop-by-hop headers",
			src: http.Header{
				"Content-Type": []string{"text/plain"},
				"Connection":   []string{"keep-alive"},
				"Keep-Alive":   []string{"timeout=5"},
				"Upgrade":      []string{"websocket"},
			},
			wantCopy: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			wantSkip: []string{"Connection", "Keep-Alive", "Upgrade"},
		},
		{
			name: "copy multi-value headers",
			src: http.Header{
				"Accept":          []string{"text/html", "application/json"},
				"X-Forwarded-For": []string{"192.168.1.1", "10.0.0.1"},
			},
			wantCopy: map[string][]string{
				"Accept":          {"text/html", "application/json"},
				"X-Forwarded-For": {"192.168.1.1", "10.0.0.1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := make(http.Header)
			copyHeader(dst, tt.src)

			// Check that expected headers were copied
			for key, wantValues := range tt.wantCopy {
				gotValues := dst[key]
				if len(gotValues) != len(wantValues) {
					t.Errorf("Header %q: got %d values, want %d", key, len(gotValues), len(wantValues))
					continue
				}
				for i, want := range wantValues {
					if gotValues[i] != want {
						t.Errorf("Header %q[%d]: got %q, want %q", key, i, gotValues[i], want)
					}
				}
			}

			// Check that hop-by-hop headers were skipped
			for _, skip := range tt.wantSkip {
				if _, exists := dst[skip]; exists {
					t.Errorf("Hop-by-hop header %q should not be copied", skip)
				}
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
	cacheKey := "GET:" + backendServer.URL + "/test"
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
	cacheKey := "GET:" + backendServer.URL + "/test"
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
	cacheKey := "POST:" + backendServer.URL + "/test"
	if _, _, ok := mockCache.Get(cacheKey); ok {
		t.Error("POST request should not be cached")
	}
}