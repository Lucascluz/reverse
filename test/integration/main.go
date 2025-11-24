package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Backend server simulator
type BackendServer struct {
	port         int
	requestCount atomic.Int32
	server       *http.Server
}

func NewBackendServer(port int) *BackendServer {
	bs := &BackendServer{
		port: port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", bs.handleRequest)
	mux.HandleFunc("/slow", bs.handleSlowRequest)
	mux.HandleFunc("/data", bs.handleDataRequest)

	bs.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return bs
}

func (bs *BackendServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	count := bs.requestCount.Add(1)
	log.Printf("[Backend:%d] Request #%d: %s %s", bs.port, count, r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Backend-Port", fmt.Sprintf("%d", bs.port))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello from backend:%d (request #%d)", bs.port, count)
}

func (bs *BackendServer) handleSlowRequest(w http.ResponseWriter, r *http.Request) {
	count := bs.requestCount.Add(1)
	log.Printf("[Backend:%d] Slow request #%d: %s %s", bs.port, count, r.Method, r.URL.Path)
	time.Sleep(500 * time.Millisecond) // Simulate slow processing
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Slow response from backend:%d", bs.port)
}

func (bs *BackendServer) handleDataRequest(w http.ResponseWriter, r *http.Request) {
	count := bs.requestCount.Add(1)
	log.Printf("[Backend:%d] Data request #%d: %s %s", bs.port, count, r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"backend": %d, "timestamp": "%s", "request": %d}`, bs.port, time.Now().Format(time.RFC3339), count)
}

func (bs *BackendServer) Start() {
	go func() {
		log.Printf("[Backend:%d] Starting on port %d", bs.port, bs.port)
		if err := bs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Backend:%d] Error: %v", bs.port, err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // Give server time to start
}

func (bs *BackendServer) Stop() {
	bs.server.Close()
}

func (bs *BackendServer) GetRequestCount() int32 {
	return bs.requestCount.Load()
}

// Test client
type TestClient struct {
	proxyURL string
	client   *http.Client
}

func NewTestClient(proxyURL string) *TestClient {
	return &TestClient{
		proxyURL: proxyURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (tc *TestClient) MakeRequest(path string) (*http.Response, error) {
	url := tc.proxyURL + path
	resp, err := tc.client.Get(url)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (tc *TestClient) ReadResponse(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Test suite
func runBasicFunctionalityTest(client *TestClient) {
	fmt.Println("\n=== Test 1: Basic Proxy Functionality ===")
	resp, err := client.MakeRequest("/")
	if err != nil {
		log.Printf("❌ FAILED: %v", err)
		return
	}

	body, _ := client.ReadResponse(resp)
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ PASSED: Got response from backend\n")
		fmt.Printf("   Status: %d\n", resp.StatusCode)
		fmt.Printf("   Body: %s\n", body)
		fmt.Printf("   Backend Port: %s\n", resp.Header.Get("X-Backend-Port"))
	} else {
		fmt.Printf("❌ FAILED: Expected 200, got %d\n", resp.StatusCode)
	}
}

func runLoadBalancingTest(client *TestClient) {
	fmt.Println("\n=== Test 2: Load Balancing ===")
	backendCounts := make(map[string]int)
	totalRequests := 20

	for i := 0; i < totalRequests; i++ {
		resp, err := client.MakeRequest(fmt.Sprintf("/test%d", i))
		if err != nil {
			log.Printf("Request %d failed: %v", i, err)
			continue
		}
		backend := resp.Header.Get("X-Backend-Port")
		backendCounts[backend]++
		resp.Body.Close()
	}

	fmt.Printf("✅ Distribution across backends:\n")
	for backend, count := range backendCounts {
		percentage := float64(count) / float64(totalRequests) * 100
		fmt.Printf("   Backend:%s -> %d requests (%.1f%%)\n", backend, count, percentage)
	}
}

func runCacheEfficiencyTest(client *TestClient, backend1, backend2 *BackendServer) {
	fmt.Println("\n=== Test 3: Cache Efficiency ===")

	// Reset counters
	backend1.requestCount.Store(0)
	backend2.requestCount.Store(0)

	// Note: This test assumes proxy caching is implemented
	// Make multiple requests to the same endpoint
	totalRequests := 100
	cacheHits := 0
	cacheMisses := 0
	paths := []string{"/data", "/slow", "/test"}

	start := time.Now()

	for i := 0; i < totalRequests; i++ {
		path := paths[i%len(paths)]
		resp, err := client.MakeRequest(path)
		if err != nil {
			log.Printf("Request failed: %v", err)
			continue
		}

		// Check for cache hit header
		if resp.Header.Get("X-Cache") == "HIT" {
			cacheHits++
		} else {
			cacheMisses++
		}
		resp.Body.Close()
	}

	elapsed := time.Since(start)
	totalBackendRequests := backend1.GetRequestCount() + backend2.GetRequestCount()

	fmt.Printf("Results:\n")
	fmt.Printf("   Total Requests: %d\n", totalRequests)
	fmt.Printf("   Cache Hits: %d (%.1f%%)\n", cacheHits, float64(cacheHits)/float64(totalRequests)*100)
	fmt.Printf("   Cache Misses: %d (%.1f%%)\n", cacheMisses, float64(cacheMisses)/float64(totalRequests)*100)
	fmt.Printf("   Backend Requests: %d\n", totalBackendRequests)
	fmt.Printf("   Total Time: %v\n", elapsed)
	fmt.Printf("   Avg Time/Request: %v\n", elapsed/time.Duration(totalRequests))

	if totalBackendRequests < int32(totalRequests) {
		fmt.Printf("✅ PASSED: Cache reduced backend load by %d%%\n",
			(totalRequests-int(totalBackendRequests))*100/totalRequests)
	} else {
		fmt.Printf("⚠️  WARNING: No cache efficiency detected\n")
	}
}

func runConcurrencyTest(client *TestClient) {
	fmt.Println("\n=== Test 4: Concurrent Load Test ===")

	var wg sync.WaitGroup
	goroutines := 50
	requestsPerGoroutine := 20
	totalRequests := goroutines * requestsPerGoroutine

	successCount := atomic.Int32{}
	errorCount := atomic.Int32{}

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				resp, err := client.MakeRequest(fmt.Sprintf("/load%d-%d", id, j))
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

	fmt.Printf("Results:\n")
	fmt.Printf("   Goroutines: %d\n", goroutines)
	fmt.Printf("   Total Requests: %d\n", totalRequests)
	fmt.Printf("   Successful: %d\n", successCount.Load())
	fmt.Printf("   Errors: %d\n", errorCount.Load())
	fmt.Printf("   Total Time: %v\n", elapsed)
	fmt.Printf("   Throughput: %.2f req/sec\n", float64(totalRequests)/elapsed.Seconds())

	if errorCount.Load() == 0 {
		fmt.Printf("✅ PASSED: All concurrent requests succeeded\n")
	} else {
		fmt.Printf("⚠️  WARNING: %d requests failed\n", errorCount.Load())
	}
}

func runPerformanceTest(client *TestClient) {
	fmt.Println("\n=== Test 5: Performance Benchmark ===")

	iterations := []int{10, 50, 100, 500}

	for _, n := range iterations {
		start := time.Now()
		for i := 0; i < n; i++ {
			resp, err := client.MakeRequest(fmt.Sprintf("/perf%d", i))
			if err == nil {
				resp.Body.Close()
			}
		}
		elapsed := time.Since(start)

		fmt.Printf("   %d requests: %v (%.2f req/sec)\n",
			n, elapsed, float64(n)/elapsed.Seconds())
	}
}

func main() {
	fmt.Println("==============================================")
	fmt.Println("   Traefik Proxy Integration Test Suite")
	fmt.Println("==============================================")

	// Start backend servers
	backend1 := NewBackendServer(8081)
	backend2 := NewBackendServer(8082)

	backend1.Start()
	backend2.Start()

	defer backend1.Stop()
	defer backend2.Stop()

	fmt.Println("\n✅ Backend servers started on ports 8081 and 8082")
	fmt.Println("⚠️  Make sure your proxy is running on port 8080")
	fmt.Println("   Run: go run cmd/main.go")

	time.Sleep(2 * time.Second)

	// Create test client
	client := NewTestClient("http://localhost:8080")

	// Run tests
	runBasicFunctionalityTest(client)
	runLoadBalancingTest(client)
	runCacheEfficiencyTest(client, backend1, backend2)
	runConcurrencyTest(client)
	runPerformanceTest(client)

	// Summary
	fmt.Println("\n==============================================")
	fmt.Println("   Test Suite Complete")
	fmt.Println("==============================================")
	fmt.Printf("\nBackend Statistics:\n")
	fmt.Printf("   Backend 8081: %d total requests\n", backend1.GetRequestCount())
	fmt.Printf("   Backend 8082: %d total requests\n", backend2.GetRequestCount())
	fmt.Printf("   Combined: %d requests\n", backend1.GetRequestCount()+backend2.GetRequestCount())
}
