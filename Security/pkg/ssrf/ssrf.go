// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package ssrf is the canonical Server-Side Request Forgery guard
// for the digital.vasic.* ecosystem. It mirrors the
// tldrsec/awesome-secure-defaults "SSRF Defense" row: block private,
// loopback, link-local, metadata, ULA, multicast, and unspecified
// addresses by default; allow opt-in for trusted on-prem endpoints;
// canonicalise alternative IP encodings that libc/cgo can still dial.
//
// Consumers:
//   - catalog-api internal/services/ssrf_guard.go (Catalogizer)
//   - HelixQA pkg/nexus/ai/ssrf_guard.go
//
// Keep those copies in sync with this one. When either changes, port
// the change here and run both downstream test suites. A future
// refactor can migrate both consumers to import this package
// directly — until then, this file is the source of truth for the
// algorithm.
package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrBlocked is returned when the guard rejects a target URL. Wrap
// with fmt.Errorf("%w: <reason>", ErrBlocked) for a typed reason.
var ErrBlocked = errors.New("ssrf blocked")

// Config tunes guard behaviour. Zero value = safe defaults: public
// hosts only, http+https only.
type Config struct {
	// AllowPrivateNetworks lets requests reach RFC1918 / loopback /
	// link-local destinations. Only flip on for a trusted endpoint.
	AllowPrivateNetworks bool

	// AllowedSchemes lists accepted URI schemes. Empty = http+https.
	AllowedSchemes []string

	// Resolver overrides the default DNS resolver. Tests inject a
	// deterministic stub so they run offline.
	Resolver Resolver
}

// Resolver is the narrow DNS contract the guard needs. *net.Resolver
// and net.DefaultResolver both satisfy it via LookupIP.
type Resolver interface {
	LookupIP(network, host string) ([]net.IP, error)
}

type stdlibResolver struct{}

func (stdlibResolver) LookupIP(network, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(context.Background(), network, host)
}

// Validate parses target and runs every guard check. Returns
// ErrBlocked (wrapped with a reason) on rejection, nil on pass.
func Validate(target string, cfg Config) error {
	if target == "" {
		return fmt.Errorf("%w: empty url", ErrBlocked)
	}
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("%w: parse: %v", ErrBlocked, err)
	}
	if err := validateScheme(u.Scheme, cfg); err != nil {
		return err
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%w: empty host", ErrBlocked)
	}
	if host == "0.0.0.0" || host == "::" {
		return fmt.Errorf("%w: unspecified address %q", ErrBlocked, host)
	}

	// Direct IP literal — no DNS needed.
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip, cfg)
	}

	// Alternative IP encodings libc/cgo can still dial even though
	// net.ParseIP rejects them. Catch before the DNS path.
	if ip := ParseIntegerIP(host); ip != nil {
		return checkIP(ip, cfg)
	}
	if ip := ParseShortDottedIP(host); ip != nil {
		return checkIP(ip, cfg)
	}

	// Hostname path: resolve + reject if any returned IP is private.
	resolver := cfg.Resolver
	if resolver == nil {
		resolver = stdlibResolver{}
	}
	ips, lookupErr := resolver.LookupIP("ip", host)
	if lookupErr != nil {
		return fmt.Errorf("%w: lookup %s: %v", ErrBlocked, host, lookupErr)
	}
	if len(ips) == 0 {
		return fmt.Errorf("%w: host %q resolves to zero IPs", ErrBlocked, host)
	}
	for _, ip := range ips {
		if err := checkIP(ip, cfg); err != nil {
			return err
		}
	}
	return nil
}

func validateScheme(scheme string, cfg Config) error {
	scheme = strings.ToLower(scheme)
	allowed := cfg.AllowedSchemes
	if len(allowed) == 0 {
		allowed = []string{"http", "https"}
	}
	for _, s := range allowed {
		if scheme == strings.ToLower(s) {
			return nil
		}
	}
	return fmt.Errorf("%w: scheme %q not in allow list", ErrBlocked, scheme)
}

func checkIP(ip net.IP, cfg Config) error {
	if ip.IsUnspecified() {
		return fmt.Errorf("%w: unspecified address %s", ErrBlocked, ip)
	}
	if cfg.AllowPrivateNetworks {
		return nil
	}
	if ip.IsLoopback() {
		return fmt.Errorf("%w: loopback %s", ErrBlocked, ip)
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("%w: link-local %s", ErrBlocked, ip)
	}
	if ip.IsPrivate() {
		return fmt.Errorf("%w: private address %s", ErrBlocked, ip)
	}
	if isIPv6UniqueLocal(ip) {
		return fmt.Errorf("%w: ULA fc00::/7 %s", ErrBlocked, ip)
	}
	if ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return fmt.Errorf("%w: multicast %s", ErrBlocked, ip)
	}
	return nil
}

func isIPv6UniqueLocal(ip net.IP) bool {
	v6 := ip.To16()
	if v6 == nil || ip.To4() != nil {
		return false
	}
	return v6[0]&0xfe == 0xfc
}

// ParseIntegerIP treats an all-digit host as a 32-bit IPv4 value
// (e.g. "2130706433" → 127.0.0.1). Returns nil on non-digit or
// uint32 overflow — safe against DNS names that end in digits.
func ParseIntegerIP(host string) net.IP {
	if host == "" || len(host) > 10 {
		return nil
	}
	var v uint64
	for _, r := range host {
		if r < '0' || r > '9' {
			return nil
		}
		v = v*10 + uint64(r-'0')
		if v > 0xFFFFFFFF {
			return nil
		}
	}
	return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// ParseShortDottedIP expands a two- or three-octet dotted form to
// the canonical four-octet IPv4 address. "127.1" → 127.0.0.1,
// "10.1" → 10.0.0.1, "192.168.1" → 192.168.0.1. Returns nil if the
// form isn't a short dotted IPv4 or any component is out of range.
func ParseShortDottedIP(host string) net.IP {
	parts := strings.Split(host, ".")
	if len(parts) != 2 && len(parts) != 3 {
		return nil
	}
	nums := make([]uint64, len(parts))
	for i, p := range parts {
		if p == "" || len(p) > 10 {
			return nil
		}
		var v uint64
		for _, r := range p {
			if r < '0' || r > '9' {
				return nil
			}
			v = v*10 + uint64(r-'0')
			if v > 0xFFFFFFFF {
				return nil
			}
		}
		nums[i] = v
	}
	for i := 0; i < len(nums)-1; i++ {
		if nums[i] > 0xFF {
			return nil
		}
	}
	var full uint64
	switch len(nums) {
	case 2:
		if nums[1] > 0xFFFFFF {
			return nil
		}
		full = nums[0]<<24 | nums[1]
	case 3:
		if nums[2] > 0xFFFF {
			return nil
		}
		full = nums[0]<<24 | nums[1]<<16 | nums[2]
	}
	return net.IPv4(byte(full>>24), byte(full>>16), byte(full>>8), byte(full))
}
