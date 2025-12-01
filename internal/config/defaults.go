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

	// Cache defaults
	DefaultTTL           = 5 * time.Minute  // Conservative fallback
	DefaultMaxAge        = 24 * time.Hour   // Reasonable upper bound
	DefaultPurgeInterval = 10 * time.Minute // Cleanup frequency

	// Backend defaults
	DefaultName     = "backend"
	DefaultWeight   = 1
	DefaultMaxConns = 100

	// Health check defaults
	DefaultTimeout  = 5 * time.Second
	DefaultInterval = 10 * time.Second

	// Load balancer defaults
	DefaultLoadBalancerType = "round-robin"
)

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

	// Apply defaults for cache config

	// Note: cache.Disabled defaults to false (cache enabled by default)

	if c.Cache.DefaultTTL == 0 {
		c.Cache.DefaultTTL = DefaultTTL
	}

	if c.Cache.MaxAge == 0 {
		c.Cache.MaxAge = DefaultMaxAge
	}

	if c.Cache.PurgeInterval == 0 {
		c.Cache.PurgeInterval = DefaultPurgeInterval
	}

	// Apply defaults for backend pool config
	if c.Pool.Backends == nil {
		return fmt.Errorf("backend pool config is missing")
	}

	if len(c.Pool.Backends) == 0 {
		return fmt.Errorf("no backends configured")
	}

	// Apply defaults for backend config
	for i := range c.Pool.Backends {
		// take pointer to element so we mutate the slice element directly
		b := &c.Pool.Backends[i]

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
	if c.Pool.HealthChecker.Interval == 0 {
		c.Pool.HealthChecker.Interval = DefaultInterval
	}

	if c.Pool.HealthChecker.Timeout == 0 {
		c.Pool.HealthChecker.Timeout = DefaultTimeout
	}

	// Apply defaults for load balancer config
	if c.Pool.LoadBalancer.Type == "" {
		c.Pool.LoadBalancer.Type = DefaultLoadBalancerType
	}

	return nil
}
