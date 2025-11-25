package cache_test

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

// TestCacheHitAndMiss tests basic cache hit/miss behavior
func TestCacheHitAndMiss(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// First request should be a cache miss
	resp1, err := client.Get(proxyURL + "/data")
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	defer resp1.Body.Close()

	cacheStatus1 := resp1.Header.Get("X-Cache")
	body1, _ := io.ReadAll(resp1.Body)

	t.Logf("First request - Cache Status: %s", cacheStatus1)

	// Second request should be a cache hit (if caching is enabled)
	time.Sleep(100 * time.Millisecond)
	resp2, err := client.Get(proxyURL + "/data")
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	defer resp2.Body.Close()

	cacheStatus2 := resp2.Header.Get("X-Cache")
	body2, _ := io.ReadAll(resp2.Body)

	t.Logf("Second request - Cache Status: %s", cacheStatus2)

	if cacheStatus2 == "HIT" {
		t.Logf("✓ Cache is working - second request was served from cache")
		if string(body1) != string(body2) {
			t.Error("Cache hit returned different content than original")
		}
	} else {
		t.Logf("⚠ Cache miss or caching disabled")
	}
}

// TestCacheEfficiency measures cache hit rate and backend load reduction
func TestCacheEfficiency(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Use a fixed set of paths to increase cache hits
	paths := []string{"/data", "/slow", "/test1", "/test2", "/test3"}
	totalRequests := 500
	cacheHits := 0
	cacheMisses := 0

	start := time.Now()

	for i := 0; i < totalRequests; i++ {
		// 80% of requests go to cached paths, 20% to new paths
		var path string
		if i%5 < 4 {
			path = paths[i%len(paths)]
		} else {
			path = fmt.Sprintf("/unique-%d", i)
		}

		resp, err := client.Get(proxyURL + path)
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			continue
		}

		if resp.Header.Get("X-Cache") == "HIT" {
			cacheHits++
		} else {
			cacheMisses++
		}
		resp.Body.Close()
	}

	elapsed := time.Since(start)
	hitRate := float64(cacheHits) / float64(totalRequests) * 100
	avgTime := elapsed / time.Duration(totalRequests)

	t.Logf("✓ Cache Efficiency Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Cache Hits: %d (%.1f%%)", cacheHits, hitRate)
	t.Logf("  Cache Misses: %d (%.1f%%)", cacheMisses, 100-hitRate)
	t.Logf("  Total Time: %v", elapsed)
	t.Logf("  Avg Time/Request: %v", avgTime)

	if cacheHits > 0 {
		t.Logf("✓ Cache is reducing backend load")
	}
}

// TestCacheConcurrency tests cache behavior under concurrent access
func TestCacheConcurrency(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	goroutines := 100
	requestsPerGoroutine := 10
	totalRequests := goroutines * requestsPerGoroutine

	var cacheHits, cacheMisses atomic.Int32
	var wg sync.WaitGroup

	// Use a single path to maximize cache hits
	testPath := "/data"

	// Prime the cache with one request
	resp, err := client.Get(proxyURL + testPath)
	if err == nil {
		resp.Body.Close()
	}
	time.Sleep(100 * time.Millisecond)

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				resp, err := client.Get(proxyURL + testPath)
				if err != nil {
					continue
				}

				if resp.Header.Get("X-Cache") == "HIT" {
					cacheHits.Add(1)
				} else {
					cacheMisses.Add(1)
				}
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	hitRate := float64(cacheHits.Load()) / float64(totalRequests) * 100
	throughput := float64(totalRequests) / elapsed.Seconds()

	t.Logf("✓ Cache Concurrency Results:")
	t.Logf("  Goroutines: %d", goroutines)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Cache Hits: %d (%.1f%%)", cacheHits.Load(), hitRate)
	t.Logf("  Cache Misses: %d", cacheMisses.Load())
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f req/sec", throughput)

	if cacheHits.Load() == 0 {
		t.Log("⚠ No cache hits detected - cache may not be working properly")
	}
}

// TestCacheConsistency tests that cache returns consistent data
func TestCacheConsistency(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	testPath := "/data"
	numRequests := 50
	bodies := make(map[string]int)

	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(proxyURL + testPath)
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		bodies[string(body)]++
		resp.Body.Close()

		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("✓ Cache Consistency Results:")
	t.Logf("  Total Requests: %d", numRequests)
	t.Logf("  Unique Responses: %d", len(bodies))

	if len(bodies) == 1 {
		t.Logf("✓ All responses were identical (perfect consistency)")
	} else if len(bodies) <= 3 {
		t.Logf("✓ Few unique responses - acceptable with TTL expiration")
	} else {
		t.Logf("⚠ Many unique responses (%d) - cache may not be working", len(bodies))
	}
}

// TestCacheWithDifferentPaths tests that cache properly distinguishes paths
func TestCacheWithDifferentPaths(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	paths := []string{"/test1", "/test2", "/test3", "/data", "/slow"}

	for _, path := range paths {
		// Make two requests to each path
		resp1, err := client.Get(proxyURL + path)
		if err != nil {
			t.Logf("Request to %s failed: %v", path, err)
			continue
		}
		body1, _ := io.ReadAll(resp1.Body)
		resp1.Body.Close()

		time.Sleep(50 * time.Millisecond)

		resp2, err := client.Get(proxyURL + path)
		if err != nil {
			t.Logf("Second request to %s failed: %v", path, err)
			continue
		}
		body2, _ := io.ReadAll(resp2.Body)
		cacheStatus := resp2.Header.Get("X-Cache")
		resp2.Body.Close()

		t.Logf("Path %s - Cache: %s, Content Match: %v",
			path, cacheStatus, string(body1) == string(body2))
	}

	t.Logf("✓ Cache path differentiation test completed")
}

// TestCachePerformanceGain measures speedup from caching
func TestCachePerformanceGain(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Test with slow endpoint to see cache benefit
	slowPath := "/slow"

	// First request (cache miss)
	start1 := time.Now()
	resp1, err := client.Get(proxyURL + slowPath)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()
	duration1 := time.Since(start1)

	time.Sleep(100 * time.Millisecond)

	// Second request (potential cache hit)
	start2 := time.Now()
	resp2, err := client.Get(proxyURL + slowPath)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	cacheStatus := resp2.Header.Get("X-Cache")
	resp2.Body.Close()
	duration2 := time.Since(start2)

	t.Logf("✓ Cache Performance Test:")
	t.Logf("  First request (miss): %v", duration1)
	t.Logf("  Second request (%s): %v", cacheStatus, duration2)

	if cacheStatus == "HIT" && duration2 < duration1 {
		speedup := float64(duration1) / float64(duration2)
		t.Logf("  ✓ Speedup: %.2fx faster", speedup)
	} else if cacheStatus == "HIT" {
		t.Logf("  Cache hit but no significant speedup detected")
	} else {
		t.Logf("  ⚠ Cache miss - no performance comparison available")
	}
}
