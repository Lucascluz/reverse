package proxy

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Date       time.Time
}

// Proxy serializes before storing
func (p *Proxy) storeResponse(method string, uri string, statusCode int, headers map[string][]string, body []byte, expiresAt time.Time) error {

	cached := &CachedResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
		Date:       time.Now(),
	}

	value, err := serialize(cached)
	if err != nil {
		return err
	}

	key := genKey(method, uri, headers)

	ttl := time.Until(expiresAt)

	p.cache.Set(key, value, ttl)

	return nil
}

func (p *Proxy) getResponse(method string, uri string, headers http.Header) (*CachedResponse, bool) {

	key := genKey(method, uri, headers)

	value, found := p.cache.Get(key)
	if !found {
		return nil, false
	}

	cached, err := deserialize(value)
	if err != nil {
		return nil, false
	}

	return cached, true
}

// genKey generates a unique key for a given request.
func genKey(method string, uri string, headers http.Header) string {

	// Define base resource key
	key := fmt.Sprintf("%s|%s", method, uri)

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

		// Build variant key
		variantKey := fmt.Sprintf("|vary:%s", strings.Join(values, ","))

		// Full cache key = base key + variant key.
		key = fmt.Sprintf("%s%s", key, variantKey)
	}

	return key
}

// Proxy serializes before storing
func serialize(v *CachedResponse) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Proxy deserializes when retrieving
func deserialize(data []byte) (*CachedResponse, error) {
	var cached CachedResponse
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&cached); err != nil {
		return nil, err
	}
	return &cached, nil
}
