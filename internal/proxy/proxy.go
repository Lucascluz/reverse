package proxy

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Lucascluz/reverse/internal/cache"
	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/pool"
)

type Proxy struct {
	Host      string
	Port      string
	ProbePort string

	client      *http.Client
	probeClient *http.Client
	pool        *pool.Pool
	cache       cache.Cache

	defaultTTL time.Duration
	maxAge     time.Duration

	ready atomic.Bool
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

	probeTransport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}

	proxy := &Proxy{

		Host:      cfg.Proxy.Host,
		Port:      cfg.Proxy.Port,
		ProbePort: cfg.Proxy.ProbePort,

		client: &http.Client{
			Transport: transport,
			// Do not follow redirects automatically in a proxy
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},

		probeClient: &http.Client{
			Transport: probeTransport,
			Timeout:   1 * time.Second,
		},
		cache: cache.NewInMemoryCache(&cfg.Cache),

		defaultTTL: time.Duration(cfg.Proxy.DefaultTTL) * time.Second,
		maxAge:     time.Duration(cfg.Proxy.MaxAge) * time.Second,

		ready: atomic.Bool{},
	}

	// Create pool with readiness callback
	proxy.pool = pool.New(&cfg.Pool, func() {
		proxy.SetReady(proxy.pool.IsReady())
	})

	// Initial readiness (will be updated by callback soon)
	proxy.ready.Store(proxy.pool.IsReady())

	return proxy
}

func (p *Proxy) SetReady(v bool) {
	p.ready.Store(v)
}

func (p *Proxy) IsReady() bool {
	return p.ready.Load()
}
