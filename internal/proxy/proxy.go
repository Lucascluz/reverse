package proxy

import (
	"net"
	"net/http"
	"time"

	"github.com/Lucascluz/reverse/internal/cache"
	"github.com/Lucascluz/reverse/internal/config"
)

type Proxy struct {

	// TODO: Implement proper target management
	targets []string

	client *http.Client

	cache cache.Cache
}

func NewProxy(cfg *config.Config) *Proxy {

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

	// TODO: Implement configuration options for cache and targets
	return &Proxy{
		targets: cfg.Proxy.Targets,
		cache:   cache.NewMemoryCache(&cfg.Cache),
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
