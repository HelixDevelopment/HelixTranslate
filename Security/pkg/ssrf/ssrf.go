// Package ssrf provides SSRF (Server-Side Request Forgery) protection.
package ssrf

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrBlocked is returned when a URL is rejected by the SSRF guard.
var ErrBlocked = errors.New("SSRF guard blocked the request")

// Config tunes the guard. Zero value is safe: all private ranges rejected.
type Config struct {
	AllowLocalhost   bool     // If true, localhost/127.0.0.1 is permitted
	AllowPrivateIPs  bool     // If true, RFC1918 addresses are permitted
	AllowMetadataIPs bool     // If true, cloud metadata endpoints are permitted
	AllowedSchemes   []string // e.g. ["https"]; empty = allow all
	BlockedHosts     []string // Additional hostnames to block
}

// Resolver is the narrow DNS contract the guard needs.
type Resolver interface {
	LookupHost(host string) ([]string, error)
}

// defaultResolver implements Resolver using net.LookupHost.
type defaultResolver struct{}

func (defaultResolver) LookupHost(host string) ([]string, error) {
	return net.LookupHost(host)
}

// Validate parses target and runs every guard check.
// Returns ErrBlocked (wrapped with a reason) on rejection, nil on pass.
func Validate(target string, cfg Config) error {
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("%w: invalid URL: %v", ErrBlocked, err)
	}

	if len(cfg.AllowedSchemes) > 0 {
		schemeOK := false
		for _, s := range cfg.AllowedSchemes {
			if strings.EqualFold(u.Scheme, s) {
				schemeOK = true
				break
			}
		}
		if !schemeOK {
			return fmt.Errorf("%w: scheme %q is not allowed", ErrBlocked, u.Scheme)
		}
	}

	host := u.Hostname()
	for _, blocked := range cfg.BlockedHosts {
		if strings.EqualFold(host, blocked) {
			return fmt.Errorf("%w: host %q is blocked", ErrBlocked, host)
		}
	}

	ips, err := defaultResolver{}.LookupHost(host)
	if err != nil {
		// If we can't resolve, we can't verify — but we also can't safely allow.
		// Be conservative: block unresolved hosts unless explicitly allowed.
		if cfg.AllowLocalhost && (host == "localhost" || host == "127.0.0.1") {
			return nil
		}
		return fmt.Errorf("%w: could not resolve host %q: %v", ErrBlocked, host, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		if ip.IsLoopback() && !cfg.AllowLocalhost {
			return fmt.Errorf("%w: loopback address %q is blocked", ErrBlocked, ipStr)
		}
		if isMetadata(ip) && !cfg.AllowMetadataIPs {
			return fmt.Errorf("%w: metadata address %q is blocked", ErrBlocked, ipStr)
		}
		if isLinkLocal(ip) && !cfg.AllowPrivateIPs {
			// Metadata endpoints are link-local; allow them if explicitly permitted
			if isMetadata(ip) && cfg.AllowMetadataIPs {
				continue
			}
			return fmt.Errorf("%w: link-local address %q is blocked", ErrBlocked, ipStr)
		}
		if isPrivate(ip) && !cfg.AllowPrivateIPs {
			return fmt.Errorf("%w: private address %q is blocked", ErrBlocked, ipStr)
		}
	}

	return nil
}

func isPrivate(ip net.IP) bool {
	privates := []*net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
	}
	for _, n := range privates {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func isLinkLocal(ip net.IP) bool {
	linkLocal := &net.IPNet{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)}
	return linkLocal.Contains(ip)
}

func isMetadata(ip net.IP) bool {
	// AWS, GCP, Azure metadata endpoints
	metadata := []string{
		"169.254.169.254",
	}
	for _, m := range metadata {
		if ip.Equal(net.ParseIP(m)) {
			return true
		}
	}
	return false
}
