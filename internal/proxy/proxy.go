package proxy

import (
	"net"
	"net/http"
	"time"

	"github.com/Lucascluz/reverxy/internal/cache"
	"github.com/Lucascluz/reverxy/internal/config"
	"github.com/Lucascluz/reverxy/internal/loadbalancer"
)

type Proxy struct {
	Host      string
	Port      string
	ProbePort string

	defaultTTL time.Duration
	maxAge     time.Duration
	client     *http.Client

	loadBalancer *loadbalancer.LoadBalancer
	cache        cache.Cache
}

func New(cfg *config.Config) *Proxy {

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

		Host:      cfg.Proxy.Host,
		Port:      cfg.Proxy.Port,
		ProbePort: cfg.Proxy.ProbePort,

		defaultTTL: time.Duration(cfg.Proxy.DefaultTTL) * time.Second,
		maxAge:     time.Duration(cfg.Proxy.MaxAge) * time.Second,

		client: &http.Client{
			Transport: transport,
			// Do not follow redirects automatically in a proxy
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},

		loadBalancer: loadbalancer.NewLoadBalancer(&cfg.LoadBalancer),
		cache:        cache.NewCache(&cfg.Cache),
	}
}

func (p *Proxy) IsReady() bool {
	return p.loadBalancer.IsReady()
}

func (p *Proxy) SetReady(ready bool) {
	p.loadBalancer.SetReady(ready)
}

// LoadBalancer returns the underlying LoadBalancer instance
func (p *Proxy) LoadBalancer() *loadbalancer.LoadBalancer {
	return p.loadBalancer
}
