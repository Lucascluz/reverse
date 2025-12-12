package main

import (
	"encoding/json"
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
	port         = flag.Int("port", 8081, "Port to listen on")
	name         = flag.String("name", "backend", "Server name for identification")
	latencyMs    = flag.Int("latency", 10, "Response latency in milliseconds")
	errorRate    = flag.Float64("error-rate", 0.0, "Percentage of requests that fail (0.0-1.0)")
	healthyDelay = flag.Duration("healthy-delay", 0, "Delay before marking server as healthy")
)

type Server struct {
	name           string
	startTime      time.Time
	requestCount   int64
	errorCount     int64
	healthy        atomic.Bool
	lastHealthTime time.Time
	mu             sync.RWMutex
}

type HealthResponse struct {
	Status   string    `json:"status"`
	Name     string    `json:"name"`
	Uptime   string    `json:"uptime"`
	Requests int64     `json:"requests"`
	Errors   int64     `json:"errors"`
	Time     time.Time `json:"time"`
}

type EchoResponse struct {
	Message    string            `json:"message"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	RemoteAddr string            `json:"remote_addr"`
	Latency    int               `json:"latency_ms"`
	Timestamp  time.Time         `json:"timestamp"`
}

func NewServer(name string) *Server {
	s := &Server{
		name:      name,
		startTime: time.Now(),
	}

	// Mark as healthy after delay
	if *healthyDelay > 0 {
		go func() {
			time.Sleep(*healthyDelay)
			s.healthy.Store(true)
			log.Printf("Server marked as healthy after %v", *healthyDelay)
		}()
	} else {
		s.healthy.Store(true)
	}

	return s
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !s.healthy.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:   "unhealthy",
			Name:     s.name,
			Uptime:   time.Since(s.startTime).String(),
			Requests: atomic.LoadInt64(&s.requestCount),
			Errors:   atomic.LoadInt64(&s.errorCount),
			Time:     time.Now(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:   "healthy",
		Name:     s.name,
		Uptime:   time.Since(s.startTime).String(),
		Requests: atomic.LoadInt64(&s.requestCount),
		Errors:   atomic.LoadInt64(&s.errorCount),
		Time:     time.Now(),
	})
}

func (s *Server) EchoHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&s.requestCount, 1)

	// Simulate latency
	if *latencyMs > 0 {
		time.Sleep(time.Duration(*latencyMs) * time.Millisecond)
	}

	// Simulate errors
	if rand.Float64() < *errorRate {
		atomic.AddInt64(&s.errorCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "simulated error",
		})
		return
	}

	// Echo response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Copy important headers
	headers := make(map[string]string)
	for k := range r.Header {
		if len(r.Header[k]) > 0 {
			headers[k] = r.Header[k][0]
		}
	}

	json.NewEncoder(w).Encode(EchoResponse{
		Message:    fmt.Sprintf("Echo from %s", s.name),
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    headers,
		RemoteAddr: r.RemoteAddr,
		Latency:    *latencyMs,
		Timestamp:  time.Now(),
	})
}

func main() {
	flag.Parse()

	server := NewServer(*name)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.HealthHandler)
	mux.HandleFunc("/", server.EchoHandler)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("[%s] Starting dummy server on %s", *name, addr)
	log.Printf("[%s] Latency: %dms, Error Rate: %.1f%%", *name, *latencyMs, *errorRate*100)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		log.Printf("[%s] Received signal: %v", *name, sig)
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[%s] Server error: %v", *name, err)
	}

	log.Printf("[%s] Server stopped", *name)
}