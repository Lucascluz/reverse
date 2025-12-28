package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// Proxy defaults
	DefaultHost      = "localhost"
	DefaultPort      = "8080"
	DefaultProbePort = "8085"
	DefaultTTL       = 5 * time.Minute // Conservative fallback
	DefaultMaxAge    = 24 * time.Hour  // Reasonable upper bound

	// Cache defaults
	DefaultPurgeInterval = 10 * time.Minute // Cleanup frequency

	// Backend defaults
	DefaultName     = "backend"
	DefaultWeight   = 1
	DefaultMaxConns = 100

	// Health check defaults
	DefaultTimeout             = 5 * time.Second
	DefaultInterval            = 10 * time.Second
	DefaultMaxConcurrentChecks = 10

	// Load balancer defaults
	DefaultLoadBalancerType = "round-robin"

	// Rate limiter defaults
	DefaultRateLimiterType = "fixed-window"
	DefaultRateLimit       = 5  // Requests per second
	DefaultCapacity        = 50 // Token bucket capacity
	DefaultRefillRate      = 5  // Tokens per second
)

var DefaultTrustedProxies = []string{"", ""}

func (c *Config) applyDefaults() error {

	// Apply defaults for proxy config
	if c.Proxy.Host == "" {
		c.Proxy.Host = DefaultHost
	}

	if c.Proxy.Port == "" {
		c.Proxy.Port = DefaultPort
	}

	if c.Proxy.ProbePort == "" {
		c.Proxy.ProbePort = DefaultProbePort
	}

	if c.Proxy.DefaultTTL == 0 {
		c.Proxy.DefaultTTL = DefaultTTL
	}

	if c.Proxy.MaxAge == 0 {
		c.Proxy.MaxAge = DefaultMaxAge
	}

	// Apply defaults for cache config

	// Note: cache.Disabled defaults to false (cache enabled by default)

	if c.Cache.PurgeInterval == 0 {
		c.Cache.PurgeInterval = DefaultPurgeInterval
	}

	// Apply defaults for backend pool config
	if c.LoadBalancer.Pool.Backends == nil {
		return fmt.Errorf("backend pool config is missing")
	}

	if len(c.LoadBalancer.Pool.Backends) == 0 {
		return fmt.Errorf("no backends configured")
	}

	// Apply defaults for backend config
	for i := range c.LoadBalancer.Pool.Backends {
		// take pointer to element so we mutate the slice element directly
		b := &c.LoadBalancer.Pool.Backends[i]

		if b.Name == "" {
			b.Name = DefaultName + strconv.Itoa(i)
		}

		if b.Url == "" {
			return fmt.Errorf("%s missing URL", b.Name)
		}

		// If health_url is empty, build it from Url; if it's a relative path like "/health",
		// prepend the backend URL.
		if b.HealthUrl == "" {
			b.HealthUrl = strings.TrimRight(b.Url, "/") + "/health"
		} else if strings.HasPrefix(b.HealthUrl, "/") {
			// relative path -> join with base URL
			b.HealthUrl = strings.TrimRight(b.Url, "/") + b.HealthUrl
		}

		if b.Weight == 0 {
			b.Weight = DefaultWeight
		}

		if b.MaxConns == 0 {
			b.MaxConns = DefaultMaxConns
		}
	}

	// Apply defaults for health checker config
	if c.LoadBalancer.Pool.HealthChecker.Interval == 0 {
		c.LoadBalancer.Pool.HealthChecker.Interval = DefaultInterval
	}

	if c.LoadBalancer.Pool.HealthChecker.Timeout == 0 {
		c.LoadBalancer.Pool.HealthChecker.Timeout = DefaultTimeout
	}

	if c.LoadBalancer.Pool.HealthChecker.MaxConcurrentChecks == 0 {
		c.LoadBalancer.Pool.HealthChecker.MaxConcurrentChecks = DefaultMaxConcurrentChecks
	}

	// Apply defaults for load balancer config
	if c.LoadBalancer.Type == "" {
		c.LoadBalancer.Type = DefaultLoadBalancerType
	}

	// Apply defaults for rate limiter config
	if c.RateLimiter.Type == "" {
		c.RateLimiter.Type = DefaultRateLimiterType
	}

	if c.RateLimiter.TrustedProxies == nil {
		c.RateLimiter.TrustedProxies = DefaultTrustedProxies
	}

	if c.RateLimiter.Limit == 0 {
		c.RateLimiter.Limit = DefaultRateLimit
	}

	if c.RateLimiter.Capacity == 0 {
		c.RateLimiter.Capacity = DefaultCapacity
	}

	if c.RateLimiter.RefillRate == 0 {
		c.RateLimiter.RefillRate = DefaultRefillRate
	}

	return nil
}
