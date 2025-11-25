package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var requestCounter atomic.Int64

func main() {
	port := flag.Int("port", 8081, "Port to run the backend server on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRequest(*port))
	mux.HandleFunc("/health", handleHealth(*port))
	mux.HandleFunc("/slow", handleSlowRequest(*port))
	mux.HandleFunc("/data", handleDataRequest(*port))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("[Backend:%d] Starting server on %s", *port, addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[Backend:%d] Server failed: %v", *port, err)
	}
}

func handleRequest(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count := requestCounter.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Backend-Port", fmt.Sprintf("%d", port))
		w.Header().Set("X-Request-ID", fmt.Sprintf("%d", count))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Backend:%d | Path:%s | Request#%d", port, r.URL.Path, count)
	}
}

func handleHealth(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","port":%d,"timestamp":"%s"}`, port, time.Now().Format(time.RFC3339))
	}
}

func handleSlowRequest(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		count := requestCounter.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Backend-Port", fmt.Sprintf("%d", port))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Slow response from Backend:%d | Request#%d", port, count)
	}
}

func handleDataRequest(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count := requestCounter.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Port", fmt.Sprintf("%d", port))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"backend":%d,"path":"%s","request":%d,"timestamp":"%s"}`,
			port, r.URL.Path, count, time.Now().Format(time.RFC3339))
	}
}
