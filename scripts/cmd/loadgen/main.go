package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	proxyURL        = flag.String("url", "http://localhost:8080", "Proxy URL to target")
	probeURL        = flag.String("probe", "http://localhost:8085", "Probe/readiness URL")
	numClients      = flag.Int("clients", 10, "Number of concurrent clients")
	rps             = flag.Int("rps", 100, "Requests per second per client")
	duration        = flag.Duration("duration", 30*time.Second, "Test duration")
	verbose         = flag.Bool("verbose", false, "Verbose output")
	includeSlowPath = flag.Bool("slow", false, "Include /slow endpoint in requests")
	cachePath       = flag.Bool("cache", false, "Include /cache endpoint for testing cache behavior")
)

type RequestStats struct {
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	totalLatency     int64
	minLatency      int64
	maxLatency      int64
	mu              sync.RWMutex
}

type ClientStats struct {
	clientID int
	stats    *RequestStats
}

var (
	globalStats = &RequestStats{
		minLatency: int64(time.Hour),
	}
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
)

// Random header values for testing
var (
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		"curl/7.68.0",
		"PostmanRuntime/7.26.8",
	}

	acceptHeaders = []string{
		"application/json",
		"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"application/json, text/plain, */*",
		"*/*",
	}

	acceptLanguages = []string{
		"en-US,en;q=0.9",
		"en;q=0.9",
		"fr-FR,fr;q=0.9,en;q=0.8",
		"de-DE,de;q=0.9",
		"ja-JP,ja;q=0.9",
	}

	acceptEncodings = []string{
		"gzip, deflate",
		"gzip, deflate, br",
		"deflate",
		"",
	}

	cacheControlHeaders = []string{
		"no-cache",
		"max-age=3600",
		"public, max-age=604800",
		"private",
		"",
	}

	paths = []string{
		"/",
		"/echo",
		"/api/test",
		"/api/data",
		"/status",
	}
)

func randomElement(slice []string) string {
	if len(slice) == 0 {
		return ""
	}
	return slice[rand.Intn(len(slice))]
}

func buildHeaders() http.Header {
	headers := http.Header{}

	// Standard headers
	headers.Set("User-Agent", randomElement(userAgents))
	headers.Set("Accept", randomElement(acceptHeaders))
	headers.Set("Accept-Language", randomElement(acceptLanguages))

	if encoding := randomElement(acceptEncodings); encoding != "" {
		headers.Set("Accept-Encoding", encoding)
	}

	// Custom headers that might be useful for testing
	headers.Set("X-Request-ID", fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000)))
	headers.Set("X-Custom-Header", fmt.Sprintf("value-%d", rand.Intn(100)))

	// Cache headers
	if cacheControl := randomElement(cacheControlHeaders); cacheControl != "" {
		headers.Set("Cache-Control", cacheControl)
	}

	// Random additional headers (30% chance)
	if rand.Float64() < 0.3 {
		headers.Set("X-Forwarded-For", fmt.Sprintf("192.168.1.%d", rand.Intn(256)))
	}

	if rand.Float64() < 0.2 {
		headers.Set("X-Real-IP", fmt.Sprintf("10.0.0.%d", rand.Intn(256)))
	}

	if rand.Float64() < 0.1 {
		headers.Set("Authorization", fmt.Sprintf("Bearer token-%d", rand.Intn(1000)))
	}

	return headers
}

func getPath() string {
	paths := []string{
		"/",
		"/echo",
		"/api/test",
		"/api/data",
		"/status",
	}

	if *includeSlowPath {
		paths = append(paths, "/slow")
	}

	if *cachePath {
		paths = append(paths, "/cache")
	}

	return randomElement(paths)
}

func makeRequest(stats *RequestStats) {
	atomic.AddInt64(&stats.totalRequests, 1)

	start := time.Now()
	path := getPath()
	url := *proxyURL + path

	req, _ := http.NewRequest("GET", url, nil)
	req.Header = buildHeaders()

	resp, err := httpClient.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		atomic.AddInt64(&stats.failedRequests, 1)
		if *verbose {
			log.Printf("[ERROR] Request failed: %v", err)
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		atomic.AddInt64(&stats.successRequests, 1)
	} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		atomic.AddInt64(&stats.failedRequests, 1)
		if *verbose {
			log.Printf("[WARN] Status %d for %s", resp.StatusCode, path)
		}
	} else {
		atomic.AddInt64(&stats.failedRequests, 1)
		if *verbose {
			log.Printf("[ERROR] Status %d for %s", resp.StatusCode, path)
		}
	}

	// Update latency stats
	atomic.AddInt64(&stats.totalLatency, latency)

	// Update min/max latency
	for {
		currentMin := atomic.LoadInt64(&stats.minLatency)
		if latency < currentMin {
			if atomic.CompareAndSwapInt64(&stats.minLatency, currentMin, latency) {
				break
			}
		} else {
			break
		}
	}

	for {
		currentMax := atomic.LoadInt64(&stats.maxLatency)
		if latency > currentMax {
			if atomic.CompareAndSwapInt64(&stats.maxLatency, currentMax, latency) {
				break
			}
		} else {
			break
		}
	}

	if *verbose {
		log.Printf("[OK] %s - %dms - %d", path, latency, resp.StatusCode)
	}
}

func runClient(clientID int, ctx <-chan struct{}) {
	ticker := time.NewTicker(time.Duration(time.Second.Nanoseconds() / int64(*rps)))
	defer ticker.Stop()

	for {
		select {
		case <-ctx:
			return
		case <-ticker.C:
			makeRequest(globalStats)
		}
	}
}

func printStats(title string) {
	total := atomic.LoadInt64(&globalStats.totalRequests)
	success := atomic.LoadInt64(&globalStats.successRequests)
	failed := atomic.LoadInt64(&globalStats.failedRequests)
	totalLatency := atomic.LoadInt64(&globalStats.totalLatency)
	minLatency := atomic.LoadInt64(&globalStats.minLatency)
	maxLatency := atomic.LoadInt64(&globalStats.maxLatency)

	fmt.Println("\n" + title)
	fmt.Println(string(make([]byte, len(title))))
	fmt.Printf("Total Requests:     %d\n", total)
	fmt.Printf("Successful:         %d (%.1f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Failed:             %d (%.1f%%)\n", failed, float64(failed)/float64(total)*100)

	if total > 0 {
		avgLatency := totalLatency / total
		fmt.Printf("Average Latency:    %dms\n", avgLatency)
		fmt.Printf("Min Latency:        %dms\n", minLatency)
		fmt.Printf("Max Latency:        %dms\n", maxLatency)
		fmt.Printf("Throughput:         %.2f req/s\n", float64(total)/time.Since(time.Now().Add(-*duration)).Seconds())
	}
}

func main() {
	flag.Parse()

	log.Printf("Starting load test against %s", *proxyURL)
	log.Printf("Clients: %d, RPS per client: %d, Total RPS: %d", *numClients, *rps, *numClients**rps)
	log.Printf("Duration: %v", *duration)

	// Test readiness probe (on probe port, not proxy port)
	resp, err := http.Get(*probeURL + "/readyz")
	if err != nil {
		log.Fatalf("Cannot reach probe at %s: %v", *probeURL, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Proxy not ready: status %d", resp.StatusCode)
	}

	log.Println("Proxy is ready, starting load test...")
	time.Sleep(1 * time.Second)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	stopChan := make(chan struct{})
	testDone := make(chan struct{})

	// Start clients
	var wg sync.WaitGroup
	for i := 0; i < *numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			runClient(clientID, stopChan)
		}(i)
	}

	// Test duration timer
	go func() {
		time.Sleep(*duration)
		close(stopChan)
		testDone <- struct{}{}
	}()

	// Wait for either test completion or interrupt signal
	select {
	case <-testDone:
		log.Println("Test duration completed")
	case sig := <-sigChan:
		log.Printf("Received signal: %v, stopping test", sig)
		close(stopChan)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Allow final stats to be recorded

	printStats("Load Test Results")
}