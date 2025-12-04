package proxy

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var methods = map[string]bool{

	// default
	"GET":  true,
	"HEAD": true,

	// conditional
	"POST":  true,
	"PATCH": true,
}

var codes = map[int]bool{

	// default
	200: true,
	203: true,
	204: true,
	206: true,
	300: true,
	301: true,
	308: true,
	404: true,
	405: true,
	410: true,
	414: true,
	501: true,

	// conditional
	302: true,
	307: true,
	416: true,
	421: true,
	426: true,
	428: true,
	429: true,
	431: true,
	451: true,
	502: true,
	503: true,
	504: true,
}

// START: Response received from origin server
func (p *Proxy) tryCachingResponse(r *http.Request, statusCode int, headers map[string][]string, body []byte) (cached bool, reason string) {

	// [1] Is the request method understood and defined as cacheable?
	if !methods[r.Method] {
		return false, "Method not cacheable"
	}

	// [2] Is the response status code understood by the cache?
	if !codes[statusCode] {
		return false, "Status code not understood"
	}

	for i := range headers["Cache-Control"] {
		// [3] Does request OR response contain Cache-Control: no-store?
		if strings.Contains(headers["Cache-Control"][i], "no-store") {
			return false, "Cache-Control: no-store"
		}

		// [4] Does response contains Cache-Control: private?
		if strings.Contains(headers["Cache-Control"][i], "private") {
			return false, "Cache-Control: private"
		}
	}

	// [5] Does request contains Authorization header?
	if strings.Contains(headers["Authorization"][0], "Bearer") {

		// YES → Does response contain public, s-maxage, or must-revalidate?
		contains := false
		if strings.Contains(headers["Cache-Control"][0], "public") ||
			strings.Contains(headers["Cache-Control"][0], "s-maxage") ||
			strings.Contains(headers["Cache-Control"][0], "ust-revalidate") {
			contains = true
		}

		// NO → DO NOT STORE (authenticated, not explicitly cacheable)
		if !contains {
			return false, "Not explicitly cacheable"
		}
	}

	// [6] Does response meet ANY freshness/cacheability requirements?
	var contains bool
	var explicitFreshness bool

	var expiresAt time.Time

	//     a) Response contains Expires header
	if strings.Contains(headers["Expires"][0], "Expires") {
		contains = true
		explicitFreshness = true

		expiresAt, _ = time.Parse(time.RFC1123, headers["Expires"][0])
	}

	//     b) Response contains Cache-Control: max-age
	if strings.Contains(headers["Cache-Control"][0], "max-age") {
		contains = true
		explicitFreshness = true

		maxAge, _ := strconv.Atoi(strings.Split(headers["Cache-Control"][0], "max-age=")[1])
		expiresAt = time.Now().Add(time.Duration(maxAge) * time.Second)
	}

	//     c) Response contains Cache-Control: s-maxage (for shared cache)
	if strings.Contains(headers["Cache-Control"][0], "s-maxage") {
		contains = true
		explicitFreshness = true

		sMaxAge, _ := strconv.Atoi(strings.Split(headers["Cache-Control"][0], "s-maxage=")[1])
		expiresAt = time.Now().Add(time.Duration(sMaxAge) * time.Second)
	}

	//     d) Response contains Cache-Control: public
	if strings.Contains(headers["Cache-Control"][0], "public") {
		contains = true
	}

	//     e) Response has a status code cacheable by default (see section 2)
	if codes[statusCode] {
		contains = true
	}

	// NONE TRUE → DO NOT STORE (no freshness info, not cacheable by default)
	if !contains {
		return false, "No freshness info, nor cacheable by default"
	}

	// [7] Special method checks:
	// Method is POST?
	if r.Method == "POST" || r.Method == "PATCH" {
		// Does response have explicit freshness (Expires, max-age, s-maxage)?
		if !explicitFreshness {
			return false, "No explicit freshness"
		}
	}

	// Method is PUT, DELETE, CONNECT, TRACE, or OPTIONS?
	if !methods[r.Method] {
		// DO NOT STORE (not cacheable)
		return false, "Method not cacheable"
	}

	// [8] STORE RESPONSE
	err := p.storeResponse(r.Method, r.URL.RequestURI(), statusCode, headers, body, expiresAt)
	if err != nil {
		return false, fmt.Sprintf("Cache error: %s", err.Error())
	}

	return true, "Stored"
}

func (p *Proxy) tryServingCachedResponse(r *http.Request) (result bool, resp *CachedResponse) {

	cachedResp, found := p.getResponse(r.Method, r.URL.String(), r.Header)
	if !found {
		return false, nil
	}

	return true, cachedResp
}
