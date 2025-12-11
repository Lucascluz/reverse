package config

import (
	"time"
)

type Config struct {
	Proxy        ProxyConfig        `yaml:"proxy"`
	Cache        CacheConfig        `yaml:"cache"`
	LoadBalancer LoadBalancerConfig `yaml:"load_balancer"`
	RateLimiter  RateLimiterConfig  `yaml:"rate_limiter"`
}

type ProxyConfig struct {
	Host       string        `yaml:"host"`
	Port       string        `yaml:"port"`
	ProbePort  string        `yaml:"probe_port"`
	DefaultTTL time.Duration `yaml:"default_ttl"`
	MaxAge     time.Duration `yaml:"max_age"`
}

type CacheConfig struct {
	Disabled      bool          `yaml:"disabled"`
	PurgeInterval time.Duration `yaml:"purge_interval"`
}

type LoadBalancerConfig struct {
	Type string     `yaml:"type"`
	Pool PoolConfig `yaml:"pool"`
}

type PoolConfig struct {
	Backends      []BackendConfig     `yaml:"backends"`
	HealthChecker HealthCheckerConfig `yaml:"health_checker"`
}

type BackendConfig struct {
	Name      string `yaml:"name"`
	Url       string `yaml:"url"`
	HealthUrl string `yaml:"health_url"`
	Weight    int    `yaml:"weight"`
	MaxConns  int    `yaml:"max_conns"`
}

type HealthCheckerConfig struct {
	MaxConcurrentChecks int           `yaml:"max_concurrent_checks"`
	Interval            time.Duration `yaml:"interval"`
	Timeout             time.Duration `yaml:"timeout"`
}

type RateLimiterConfig struct {
	Type           string   `yaml:"type"`
	Limit          int      `yaml:"limit"`
	TrustedProxies []string `yaml:"trusted_proxies"`
}
