package cache

import (
	"net/http"
	"sync"
	"time"
)

type memoryCache struct {
	mu    sync.RWMutex
	items map[string]Entry

	// TODO: Implement ticker to trigger async cache operations
}

func NewMemoryCache() *memoryCache {

	// TODO: Start ticker

	return &memoryCache{
		mu:    sync.RWMutex{},
		items: make(map[string]Entry),
	}
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
	if e.expired() {
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

func (m *memoryCache) Set(key string, body []byte, headers http.Header, ttl time.Duration) {

	exp := time.Now().Add(ttl)

	if time.Now().After(exp) {
		return
	}

	filtered := stripHopByHop(headers)

	entry := Entry{
		body:    body,
		headers: filtered,
		expires: exp,
	}

	m.mu.Lock()
	m.items[key] = entry
	m.mu.Unlock()

	// Lazy purge expired entries
	go m.purgeExpired()
}

func (m *memoryCache) purgeExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.items {
		if entry.expired() {
			delete(m.items, key)
		}
	}
}
