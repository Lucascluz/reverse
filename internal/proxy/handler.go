package proxy

import (
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"time"
)

type Proxy struct {
	targets []string
	client  *http.Client
}

func NewProxy() *Proxy {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &Proxy{
		targets: []string{"http://localhost:8081", "http://localhost:8082"},
		// Initialize the client with the custom transport
		client: &http.Client{
			Transport: transport,
		},
	}
}

// The handler function that sits in the middle
func (p *Proxy) ServeHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// TODO: Implement proper loadbalancing
		nextTarget := p.targets[rand.IntN(len(p.targets))]

		// Create a new HTTP request to send to the backend.
		outReq, err := http.NewRequest(r.Method, nextTarget+r.URL.Path, r.Body)
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
		resp, err := p.client.Do(outReq)
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
}
