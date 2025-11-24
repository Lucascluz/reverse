package config

import "time"

type Config struct {
	Proxy ProxyConfig
	Cache CacheConfig
}

type ProxyConfig struct {
	Host    string
	Port    string
	Targets []string
}

type CacheConfig struct {
	DefaultTTL    time.Duration
	PurgeInterval time.Duration
}
