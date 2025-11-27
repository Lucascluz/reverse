package config

import (
	"fmt"
	"time"
)

const (
	DefaultHost = "localhost"
	DefaultPort = "8080"

	DefaultTTL           = 5 * time.Minute  // Conservative fallback
	DefaultMaxAge        = 24 * time.Hour   // Reasonable upper bound
	DefaultPurgeInterval = 10 * time.Minute // Cleanup frequency

	DefaultWeight   = 1
	DefaultMaxConns = 100
)

func (c *Config) applyDefaults() error {

	// Apply defaults for proxy config
	if c.Proxy.Host == "" {
		c.Proxy.Host = DefaultHost
	}

	if c.Proxy.Port == "" {
		c.Proxy.Port = DefaultPort
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
	for i, backend := range c.Pool.Backends {
		if backend.Weight == 0 {
			c.Pool.Backends[i].Weight = DefaultWeight
		}

		if backend.MaxConns == 0 {
			c.Pool.Backends[i].MaxConns = DefaultMaxConns
		}
	}

	return nil
}
