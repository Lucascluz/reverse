package cache

import (
	"net/http"
	"time"
)

type Entry struct {
	body    []byte
	headers http.Header
	expires time.Time
}

func (e *Entry) isExpired() bool {
	return time.Now().After(e.expires)
}
