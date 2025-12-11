package ip

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

type Extractor struct {
	trustedCIDRs []*net.IPNet
}

func NewExtractor(trustedProxies []string) (*Extractor, error) {
	var cidrs []*net.IPNet
	for _, proxy := range trustedProxies {
		_, cidr, err := net.ParseCIDR(proxy)
		if err != nil {

			// Handle single IPs
			ip := net.ParseIP(proxy)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address: %s", proxy)
			}

			// Handle IPv4 vs IPv6 masks
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}

			cidrs = append(cidrs, &net.IPNet{IP: ip, Mask: mask})
			continue
		}
		cidrs = append(cidrs, cidr)
	}
	return &Extractor{trustedCIDRs: cidrs}, nil
}

func (e *Extractor) Extract(r *http.Request) string {
	// Get direct peer IP
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// If no trusted CIDRs are configured, return the direct peer IP
	if len(e.trustedCIDRs) == 0 {
		return remoteIP
	}

	// Check if direct peer is trusted
	if !e.IsTrusted(remoteIP) {
		return remoteIP
	}

	// Parse X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")

		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip == "" {
				continue
			}

			// The first IP that is not trusted is the real client
			if !e.IsTrusted(ip) {
				return ip
			}
		}

		// If every ip was trusted return the first one (likely the client)
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fallback
	return remoteIP
}

func (e *Extractor) IsTrusted(ipStr string) bool {
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	
	for _, cidr := range e.trustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	
	return false
}
