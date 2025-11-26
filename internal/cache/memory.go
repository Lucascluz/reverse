package cache

import (
	"net/http"
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type memoryCache struct {
	mu         sync.RWMutex
	disabled   bool
	items      map[string]Entry // Map to store cached entries
	ticker     *time.Ticker     // Ticker used to periodically purge expired entries
	defaultTTL time.Duration    // Default time-to-live for cached entries
	stop       chan struct{}    // Channel to stop the ticker
}

func NewMemoryCache(config config.CacheConfig) *memoryCache {

	ticker := time.NewTicker(config.PurgeInterval)
	stop := make(chan struct{})

	mc := &memoryCache{
		mu:         sync.RWMutex{},
		disabled:   config.Disabled,
		items:      make(map[string]Entry),
		ticker:     ticker,
		defaultTTL: config.DefaultTTL,
		stop:       stop,
	}

	go mc.initPurgeTicker(ticker, stop)

	return mc
}

func (m *memoryCache) Get(key string) ([]byte, http.Header, bool) {
	m.mu.RLock()
	e, ok := m.items[key]
	m.mu.RUnlock()

	// Check if entry exists
	if !ok {
		return nil, nil, false
	}

	// Check if entry expired
	if e.isExpired() {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return nil, nil, false
	}

	// Return copy of entry
	bodyCopy := append([]byte(nil), e.body...)
	headersCopy := cloneHeaders(e.headers)
	return bodyCopy, headersCopy, true
}

func (m *memoryCache) Set(key string, body []byte, headers http.Header, expires time.Time) {

	if time.Now().After(expires) {
		return
	}

	filtered := stripHopByHop(headers)

	entry := Entry{
		body:    body,
		headers: filtered,
		expires: expires,
	}

	m.mu.Lock()
	m.items[key] = entry
	m.mu.Unlock()
}

func (mc *memoryCache) IsEnabled() bool {
	return !mc.disabled
}

func (mc *memoryCache) GetDefaultTTL() time.Duration {
	return mc.defaultTTL
}

func (m *memoryCache) initPurgeTicker(ticker *time.Ticker, stop chan struct{}) {
	for {
		select {
		case <-ticker.C:
			m.purgeExpired()
		case <-stop:
			ticker.Stop()
			return
		}
	}
}

func (m *memoryCache) purgeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.items {
		if entry.isExpired() {
			delete(m.items, key)
		}
	}
}
