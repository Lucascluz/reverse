package proxy

import (
	"net"
	"net/http"
	"time"

	"github.com/Lucascluz/reverse/internal/cache"
)

type Proxy struct {

	// TODO: Implement proper target management
	targets []string

	client *http.Client

	cache cache.Cache
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
		cache:   cache.NewMemoryCache(),
		// Initialize the client with the custom transport
		client: &http.Client{
			Transport: transport,
			// Do not follow redirects automatically in a proxy
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}
