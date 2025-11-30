package proxy

import (
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Lucascluz/reverse/internal/backend"
	"github.com/Lucascluz/reverse/internal/cache"
	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/logger"
)

type Proxy struct {
	Host      string
	Port      string
	ProbePort string
	Logger    *logger.Logger

	client      *http.Client
	probeClient *http.Client
	pool        *backend.Pool
	cache       cache.Cache

	ready atomic.Bool
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

	probeTransport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     5 * time.Second,
		TLSHandshakeTimeout: 5 * time.Microsecond,
		DisableKeepAlives:   false,
	}

	// Initialize logger
	baseLogger := logger.New("proxy")

	proxy := &Proxy{

		Host:      cfg.Proxy.Host,
		Port:      cfg.Proxy.Port,
		ProbePort: cfg.Proxy.ProbePort,
		Logger:    baseLogger,

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

		cache: cache.NewMemoryCache(&cfg.Cache),
		ready: atomic.Bool{},
	}

	// Create pool with readiness callback
	proxy.pool = backend.NewPool(&cfg.Pool, func() {
		proxy.SetReady(proxy.pool.HealthyCount() > 0)
	})

	// Initial readiness (will be updated by callback soon)
	proxy.ready.Store(proxy.pool.HealthyCount() > 0)

	return proxy
}

func (p *Proxy) SetReady(v bool) {
	p.ready.Store(v)
}

func (p *Proxy) IsReady() bool {
	return p.ready.Load()
}
