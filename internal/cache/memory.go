package cache

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Lucascluz/reverse/internal/config"
)

type memoryCache struct {
	mu         sync.RWMutex
	disabled   bool
	items      map[string]Entry // Map to store cached entries // rename to store
	defaultTTL time.Duration    // Default time-to-live for cached entries
	maxAge     time.Duration    // Maximum age for cached entries

	ticker *time.Ticker  // Ticker used to periodically purge expired entries
	stop   chan struct{} // Channel to stop the ticker
}

func NewMemoryCache(config *config.CacheConfig) *memoryCache {

	ticker := time.NewTicker(config.PurgeInterval)
	stop := make(chan struct{})

	mc := &memoryCache{
		mu:         sync.RWMutex{},
		disabled:   config.Disabled,
		items:      make(map[string]Entry),
		ticker:     ticker,
		defaultTTL: config.DefaultTTL,
		maxAge:     config.MaxAge,
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

func (m *memoryCache) Set(key string, body []byte, headers http.Header) {

	// Initialize variables
	var (
		ttl      time.Duration
		expires  time.Time
		filtered http.Header
		entry    Entry
	)

	// Determine TTL and expiration time
	ttl = m.determineTTL(headers)
	expires = time.Now().Add(ttl)

	if time.Now().After(expires) {
		return
	}

	// Filter hop-by-hop headers
	filtered = stripHopByHop(headers)

	// Create cache entry
	entry = Entry{
		body:    body,
		headers: filtered,
		expires: expires,
	}

	// Lock mutex and set entry
	m.mu.Lock()
	m.items[key] = entry
	m.mu.Unlock()
}

func (m *memoryCache) GenKey(method string, host string, path string, headers http.Header) string {

	// Define base resource key
	key := fmt.Sprintf("%s|%s|%s", method, host, path)

	// Read `Vary` from response headers.
	vary := headers.Get("Vary")

	// If absent, treat as empty (no variants).
	if vary != "" {
		names := strings.Split(vary, ",")
		values := make([]string, len(names))

		for i, name := range names {
			// Parse header names in `Vary` -> normalize (lowercase, trim).
			trimmed := strings.TrimSpace(strings.ToLower(name))
			// For each header name in `Vary`, obtain the request’s header value(s). Normalize and join them.
			values[i] = strings.Join(headers.Values(trimmed), ",")
		}
		// Build a `variantKey` = hash(serialized list of headerName:headerValue pairs) or a deterministic string suffix (e.g., `|vary:accept-encoding=gzip,accept-language=en-US`).
		variantKey := fmt.Sprintf("|vary:%s", strings.Join(values, ","))

		// Full cache key = base resource key + variantKey.
		key = fmt.Sprintf("%s%s", key, variantKey)
	}

	return key
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

func (m *memoryCache) determineTTL(headers http.Header) time.Duration {
	var ttl time.Duration

	// Check Cache-Control: max-age
	if cc := headers.Get("Cache-Control"); cc != "" {
		if parsed := parseMaxAge(cc); parsed > 0 {
			ttl = parsed
		}
	}

	// Check for Expires header if no max-age found
	if ttl == 0 {
		if expires := headers.Get("Expires"); expires != "" {
			if expireTime, err := http.ParseTime(expires); err == nil {
				ttl = time.Until(expireTime)
			}
		}
	}

	// Use default TTL if no cache headers or negative/zero TTL
	if ttl <= 0 {
		return m.defaultTTL
	}

	// Cap at MaxAge to prevent excessive caching
	if ttl > m.maxAge {
		return m.maxAge
	}

	return ttl
}

func parseMaxAge(cacheControl string) time.Duration {
	for directive := range strings.SplitSeq(cacheControl, ",") {
		directive = strings.TrimSpace(directive)
		if after, found := strings.CutPrefix(directive, "max-age="); found {
			if seconds, err := strconv.Atoi(after); err == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}
	return 0
}
