package config

import (
	"fmt"
	"time"
)

const (
	DefaultHost         = "localhost"
	DefaultPort         = "8080"

	DefaultTTL           = 5 * time.Minute  // Conservative fallback
	DefaultMaxAge        = 24 * time.Hour   // Reasonable upper bound
	DefaultPurgeInterval = 10 * time.Minute // Cleanup frequency
)

func (c *Config) applyDefaults() error {

	// Apply defaults for proxy config
	if c.Proxy.Host == "" {
		c.Proxy.Host = DefaultHost
	}

	if c.Proxy.Port == "" {
		c.Proxy.Port = DefaultPort
	}

	// Validate targets (required)
	if len(c.Proxy.Targets) == 0 {
		return fmt.Errorf("at least one target must be configured")
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

	return nil
}
