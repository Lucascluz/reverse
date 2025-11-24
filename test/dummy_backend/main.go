package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
)

func main() {
	// Try to find a free port starting from 8081
	var listener net.Listener
	var err error
	startPort := 8081

	for i := range 100 {
		addr := fmt.Sprintf(":%d", startPort+i)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			break
		}
	}

	if listener == nil {
		log.Fatalf("Could not find a free port: %v", err)
	}

	port := listener.Addr().String()

	handler := func(w http.ResponseWriter, r *http.Request) {
		// 1. Log the incoming request details to the console
		// This helps you verify that headers (like User-Agent or Custom-Headers)
		// are being correctly forwarded by your proxy.
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Printf("Error dumping request: %v", err)
		} else {
			log.Printf(
				"--- [Backend] Received Request ---\n%s\n----------------------------------", dump)
		}

		// 2. Set a custom header to prove the response came from this backend
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Backend-Server", "Go-Dummy-Server")

		// 3. Write a response body
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello! I am the backend server.\nYou requested: %s %s\n", r.Method, r.URL.Path)
	}

	log.Printf("Dummy backend server started. Listening on %s", port)
	if err := http.Serve(listener, http.HandlerFunc(handler)); err != nil {
		log.Fatal(err)
	}
}
