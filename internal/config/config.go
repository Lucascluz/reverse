package config

import "time"

type Config struct {
	Proxy ProxyConfig
	Cache CacheConfig
}

type ProxyConfig struct {
	Host    string   `yaml:"host"`
	Port    string   `yaml:"port"`
	Targets []string `yaml:"targets"`
}

type CacheConfig struct {
	Enabled       bool          `yaml:"enabled"`
	DefaultTTL    time.Duration `yaml:"default_ttl"`
	PurgeInterval time.Duration `yaml:"purge_interval"`
}
