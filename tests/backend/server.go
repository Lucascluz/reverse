package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var requestCounter atomic.Int64

func main() {
	portFlag := flag.Int("port", 8081, "Preferred port to run the backend server on")
	maxAttempts := flag.Int("max-attempts", 100, "How many ports to try before giving up")
	flag.Parse()

	// Try to bind to the requested port, or the next free port up to maxAttempts.
	startPort := *portFlag
	var ln net.Listener
	var chosenPort int
	for i := 0; i < *maxAttempts; i++ {
		tryPort := startPort + i
		addr := fmt.Sprintf(":%d", tryPort)
		l, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("[Backend:%d] Port %d unavailable: %v. Trying next port...", startPort, tryPort, err)
			continue
		}
		ln = l
		chosenPort = tryPort
		break
	}

	if ln == nil {
		log.Fatalf("[Backend:%d] Could not bind any port in range %d..%d", startPort, startPort, startPort+*maxAttempts-1)
	}
	defer ln.Close()

	// Seed math/rand for small random payloads
	rand.Seed(time.Now().UnixNano())

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot(chosenPort))
	mux.HandleFunc("/health", handleHealth(chosenPort))
	mux.HandleFunc("/slow", handleSlowRequest(chosenPort))
	mux.HandleFunc("/data", handleDataRequest(chosenPort))
	mux.HandleFunc("/forward", handleForward(chosenPort))
	mux.HandleFunc("/vary", handleVary(chosenPort))

	log.Printf("[Backend:%d] Listening on %s (requested %d). PID=%d", chosenPort, ln.Addr().String(), startPort, os.Getpid())

	server := &http.Server{
		Addr:         ln.Addr().String(),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Serve (this blocks)
	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[Backend:%d] Server failed: %v", chosenPort, err)
	}
}

func logRequest(port int, r *http.Request, reqID int64, note string) {
	ua := r.Header.Get("User-Agent")
	cl := r.Header.Get("Content-Length")
	remote := r.RemoteAddr
	if remote == "" {
		remote = "unknown"
	}
	// brief headers to show
	var headers []string
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		headers = append(headers, fmt.Sprintf("X-Forwarded-For=%s", v))
	}
	if v := r.Header.Get("X-Request-Nonce"); v != "" {
		headers = append(headers, fmt.Sprintf("nonce=%s", v))
	}
	now := time.Now().Format(time.RFC3339Nano)
	log.Printf("[Backend:%d] %s | req=%d | %s %s | remote=%s | UA=%s | CL=%s | %s | %s",
		port, now, reqID, r.Method, r.URL.RequestURI(), remote, ua, cl, strings.Join(headers, " "), note)
}

func handleRoot(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)
		logRequest(port, r, id, "root")
		http.Redirect(w, r, "/data", http.StatusFound)
	}
}

// /health returns status and logs the health probe
func handleHealth(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)
		logRequest(port, r, id, "health")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","port":%d,"timestamp":"%s"}`, port, time.Now().Format(time.RFC3339))
	}
}

// /slow simulates a slow handler
func handleSlowRequest(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)
		logRequest(port, r, id, "slow:start")
		// optionally accept ?delay=ms
		delay := 200 * time.Millisecond
		if v := r.URL.Query().Get("delay"); v != "" {
			if ms, err := strconv.Atoi(v); err == nil && ms >= 0 && ms < 10000 {
				delay = time.Duration(ms) * time.Millisecond
			}
		}
		time.Sleep(delay)
		logRequest(port, r, id, "slow:end")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Backend-Port", strconv.Itoa(port))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Slow response from Backend:%d | Request#%d | delay=%s", port, id, delay)
	}
}

// /data returns JSON. If ?cache=1 set a Cache-Control header, otherwise no-cache.
// Also includes a small random payload and echoes the query params and headers
func handleDataRequest(port int) http.HandlerFunc {
	type payload struct {
		Backend   int               `json:"backend"`
		Path      string            `json:"path"`
		Request   int64             `json:"request"`
		Timestamp string            `json:"timestamp"`
		Random    int               `json:"random"`
		Query     map[string]string `json:"query"`
		Headers   map[string]string `json:"headers"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)

		// Determine caching behavior
		cacheParam := r.URL.Query().Get("cache")
		cacheable := cacheParam == "1" || cacheParam == "true"

		// If a nonce param exists in query or header, include in logs (helps trace)
		note := "data"
		if r.URL.Query().Get("_nonce") != "" {
			note = "data:nonce"
		}
		logRequest(port, r, id, note)

		// Prepare response
		qmap := map[string]string{}
		for k, vs := range r.URL.Query() {
			if len(vs) > 0 {
				qmap[k] = vs[0]
			}
		}
		hmap := map[string]string{
			"User-Agent": r.Header.Get("User-Agent"),
		}
		if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
			hmap["X-Forwarded-For"] = xf
		}

		p := payload{
			Backend:   port,
			Path:      r.URL.Path,
			Request:   id,
			Timestamp: time.Now().Format(time.RFC3339Nano),
			Random:    rand.Intn(1000000),
			Query:     qmap,
			Headers:   hmap,
		}

		// Set caching headers
		if cacheable {
			// short cacheable TTL so demo still refreshes after a while
			w.Header().Set("Cache-Control", "public, max-age=30")
		} else {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Port", strconv.Itoa(port))
		w.Header().Set("X-Request-ID", strconv.FormatInt(id, 10))

		// Write JSON body
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(p)
	}
}

// /forward echoes forwarded headers so you can verify proxy forwarding
func handleForward(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)
		logRequest(port, r, id, "forward")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend-Port", strconv.Itoa(port))
		w.WriteHeader(http.StatusOK)
		resp := map[string]string{
			"x-forwarded-for":   r.Header.Get("X-Forwarded-For"),
			"x-forwarded-proto": r.Header.Get("X-Forwarded-Proto"),
			"request-id":        strconv.FormatInt(id, 10),
			"timestamp":         time.Now().Format(time.RFC3339Nano),
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// /vary sets a Vary header and returns small content — useful to exercise cache Vary behavior
func handleVary(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestCounter.Add(1)
		logRequest(port, r, id, "vary")
		w.Header().Set("Vary", "User-Agent")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Backend-Port", strconv.Itoa(port))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Vary response from backend %d | req=%d | ua=%s", port, id, r.Header.Get("User-Agent"))
	}
}
