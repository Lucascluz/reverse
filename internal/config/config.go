package config

import (
	"time"
)

type Config struct {
	Proxy ProxyConfig `yaml:"proxy"`
	Cache CacheConfig `yaml:"cache"`
	Pool  PoolConfig  `yaml:"pool"`
}

type ProxyConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type CacheConfig struct {
	Disabled      bool          `yaml:"disabled"`
	DefaultTTL    time.Duration `yaml:"default_ttl"`
	MaxAge        time.Duration `yaml:"max_age"`
	PurgeInterval time.Duration `yaml:"purge_interval"`
}

type PoolConfig struct {
	Backends []BackendConfig `yaml:"backends"`
}

type BackendConfig struct {
	Url      string `yaml:"url"`
	Weight   int    `yaml:"weight"`
	MaxConns int    `yaml:"max_conns"`
}
