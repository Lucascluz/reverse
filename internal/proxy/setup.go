package proxy

import (
	"fmt"
	"net/http"

	"github.com/Lucascluz/reverse/internal/config"
	"github.com/Lucascluz/reverse/internal/observability"
	"github.com/Lucascluz/reverse/internal/proxy/middleware"
	"github.com/Lucascluz/reverse/internal/ratelimiter"
)

// Setup encapsulates the complete proxy initialization
type Setup struct {
	proxy *Proxy
	cfg   *config.Config
}

// NewSetup creates a proxy with its configuration ready for handler building
func NewSetup(cfg *config.Config) (*Setup, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	p := New(cfg)

	return &Setup{
		proxy: p,
		cfg:   cfg,
	}, nil
}

// Proxy returns the underlying Proxy instance
func (s *Setup) Proxy() *Proxy {
	return s.proxy
}

// Builds and returns the complete middleware-wrapped handler
func (s *Setup) Handler() (http.Handler, error) {

	// Create logger
	log := observability.NewLogger("proxy")

	// Create rate limiter
	limiter := ratelimiter.New(s.cfg.RateLimiter)

	// Create IP extractor
	extractor, err := ratelimiter.NewExtractor(s.cfg.RateLimiter.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("failed to create IP extractor: %w", err)
	}

	// Build middleware chain from innermost to outermost
	handler := http.Handler(s.proxy)

	// Apply rate limiting first (rejects early)
	handler = middleware.RateLimiting(limiter, extractor, handler)

	// Apply logging last (wraps everything)
	handler = middleware.Logging(log, handler)

	return handler, nil
}
