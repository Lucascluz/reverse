package cache

import (
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type inMemoryCache struct {
	store  map[string]*entry
	ticker *time.Ticker
	stop   chan bool
	mu     sync.RWMutex
}

func NewInMemoryCache(cfg *config.CacheConfig) *inMemoryCache {
	cache := &inMemoryCache{
		store:  make(map[string]*entry),
		ticker: time.NewTicker(cfg.PurgeInterval),
		stop:   make(chan bool),
	}

	go cache.start()

	return cache
}

type entry struct {
	value     []byte
	expiresAt time.Time
	storedAt  time.Time
}

func (c *inMemoryCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.store[key] = &entry{
		value:     value,
		expiresAt: now.Add(ttl),
		storedAt:  now,
	}
}

func (c *inMemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, exists := c.store[key]
	if !exists {
		return nil, false
	}

	// Simple TTL check - no HTTP logic
	if time.Now().After(e.expiresAt) {
		return nil, false
	}

	return e.value, true
}

func (c *inMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
}

func (c *inMemoryCache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.store[key]
	return exists
}

func (c *inMemoryCache) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stop <- true
	return nil
}

func (c *inMemoryCache) start() {
	for {
		select {
		case <-c.ticker.C:
			c.cleanup()
		case <-c.stop:
			c.ticker.Stop()
			return
		}
	}
}

func (c *inMemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, e := range c.store {
		if now.After(e.expiresAt) {
			delete(c.store, key)
		}
	}
}
