package main

import (
    "io"
    "log"
    "net/http"
)

func main() {
    // The target backend we are proxying to
    targetURL := "http://localhost:8081"

    // The handler function that sits in the middle
    proxyHandler := func(w http.ResponseWriter, r *http.Request) {

        // Create a new HTTP request to send to the backend.
        // Copy the method (GET, POST) and the body from the original request.
        outReq, err := http.NewRequest(r.Method, targetURL+r.URL.Path, r.Body)
        if err != nil {
            http.Error(w, "Bad Request", http.StatusBadRequest)
            return
        }

        // Copy headers from the incoming request to the outgoing request
        for key, values := range r.Header {
            for _, value := range values {
                outReq.Header.Add(key, value)
            }
        }

        // Send the Request to the Backend
        resp, err := http.DefaultClient.Do(outReq)
        if err != nil {
            http.Error(w, "Bad Gateway", http.StatusBadGateway)
            return
        }
        defer resp.Body.Close()
        
        // Copy headers from the backend response to the client response
        for key, values := range resp.Header {
            for _, value := range values {
                w.Header().Add(key, value)
            }
        }

        // Write the status code (Must be done before writing the body!)
        w.WriteHeader(resp.StatusCode)

        // Copy the response body
        io.Copy(w, resp.Body)
    }

    // Start the server
    log.Println("Proxy server listening on :8080")
    if err := http.ListenAndServe(":8080", http.HandlerFunc(proxyHandler)); err != nil {
        log.Fatal(err)
    }
}