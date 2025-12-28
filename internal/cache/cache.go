package cache

import (
	"time"

	"github.com/Lucascluz/reverxy/internal/config"
)

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
	Exists(key string) bool
	Stop() error
}

func NewCache(cfg *config.CacheConfig) Cache {

	//TODO: Implement various cache options (redis, memcached, etc.)
	return NewInMemoryCache(cfg)
}
