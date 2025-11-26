package config

import (
	"time"
)

type Config struct {
	Proxy ProxyConfig `yaml:"proxy"`
	Cache CacheConfig `yaml:"cache"`
}

type ProxyConfig struct {
	Host    string   `yaml:"host"`
	Port    string   `yaml:"port"`
	Targets []string `yaml:"targets"`
}

type CacheConfig struct {
	Disabled      bool          `yaml:"disabled"`
	DefaultTTL    time.Duration `yaml:"default_ttl"`
	MaxAge        time.Duration `yaml:"max_age"`
	PurgeInterval time.Duration `yaml:"purge_interval"`
}
