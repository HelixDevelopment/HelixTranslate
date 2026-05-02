// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package ssrf

import (
	"errors"
	"net"
	"strings"
	"testing"
)

type stubResolver struct {
	ips map[string][]net.IP
	err error
}

func (s stubResolver) LookupIP(_, host string) ([]net.IP, error) {
	if s.err != nil {
		return nil, s.err
	}
	if ips, ok := s.ips[host]; ok {
		return ips, nil
	}
	return nil, errors.New("stub: no such host")
}

func TestValidate_RejectsEmpty(t *testing.T) {
	if err := Validate("", Config{}); err == nil {
		t.Error("empty URL must be rejected")
	}
}

func TestValidate_RejectsUnknownScheme(t *testing.T) {
	err := Validate("gopher://example.com/", Config{})
	if err == nil || !strings.Contains(err.Error(), "scheme") {
		t.Errorf("expected scheme rejection, got %v", err)
	}
}

func TestValidate_RejectsLoopback(t *testing.T) {
	for _, target := range []string{
		"http://127.0.0.1/",
		"http://127.0.0.5:8080/",
		"http://[::1]/",
	} {
		err := Validate(target, Config{})
		if err == nil || !errors.Is(err, ErrBlocked) {
			t.Errorf("loopback %q must block, got %v", target, err)
		}
	}
}

func TestValidate_RejectsRFC1918(t *testing.T) {
	for _, target := range []string{
		"http://10.0.0.5/",
		"http://172.16.0.1/",
		"http://172.31.255.254/",
		"http://192.168.1.1/",
	} {
		err := Validate(target, Config{})
		if err == nil || !errors.Is(err, ErrBlocked) {
			t.Errorf("private %q must block, got %v", target, err)
		}
	}
}

func TestValidate_RejectsCloudMetadata(t *testing.T) {
	err := Validate("http://169.254.169.254/latest/meta-data/", Config{})
	if err == nil || !errors.Is(err, ErrBlocked) {
		t.Errorf("cloud metadata must block, got %v", err)
	}
}

func TestValidate_RejectsIPv6Private(t *testing.T) {
	for _, target := range []string{"http://[fe80::1]/", "http://[fc00::1]/"} {
		err := Validate(target, Config{})
		if err == nil || !errors.Is(err, ErrBlocked) {
			t.Errorf("IPv6 private %q must block, got %v", target, err)
		}
	}
}

func TestValidate_RejectsUnspecified(t *testing.T) {
	for _, target := range []string{"http://0.0.0.0/", "http://[::]/"} {
		err := Validate(target, Config{})
		if err == nil || !errors.Is(err, ErrBlocked) {
			t.Errorf("unspecified %q must block, got %v", target, err)
		}
	}
}

func TestValidate_PublicPasses(t *testing.T) {
	if err := Validate("https://1.1.1.1/", Config{}); err != nil {
		t.Errorf("public IP must pass, got %v", err)
	}
}

func TestValidate_HostnamePublicPasses(t *testing.T) {
	err := Validate("https://api.example.com/", Config{
		Resolver: stubResolver{ips: map[string][]net.IP{
			"api.example.com": {net.ParseIP("203.0.113.7")},
		}},
	})
	if err != nil {
		t.Errorf("public hostname must pass, got %v", err)
	}
}

func TestValidate_HostnameToPrivateBlocked(t *testing.T) {
	err := Validate("https://sneaky.example.com/", Config{
		Resolver: stubResolver{ips: map[string][]net.IP{
			"sneaky.example.com": {net.ParseIP("10.0.0.5")},
		}},
	})
	if err == nil || !errors.Is(err, ErrBlocked) {
		t.Errorf("hostname → private must block, got %v", err)
	}
}

func TestValidate_MixedResolutionBlocked(t *testing.T) {
	err := Validate("https://mixed.example.com/", Config{
		Resolver: stubResolver{ips: map[string][]net.IP{
			"mixed.example.com": {
				net.ParseIP("203.0.113.7"),
				net.ParseIP("192.168.1.1"),
			},
		}},
	})
	if err == nil || !errors.Is(err, ErrBlocked) {
		t.Errorf("mixed resolution must block, got %v", err)
	}
}

func TestValidate_AllowPrivateOptIn(t *testing.T) {
	cfg := Config{AllowPrivateNetworks: true}
	for _, target := range []string{
		"http://127.0.0.1/",
		"http://192.168.1.1/",
	} {
		if err := Validate(target, cfg); err != nil {
			t.Errorf("opt-in: %q must pass, got %v", target, err)
		}
	}
}

func TestValidate_LookupFailureBlocks(t *testing.T) {
	err := Validate("https://missing.example/", Config{
		Resolver: stubResolver{err: errors.New("nxdomain")},
	})
	if err == nil || !errors.Is(err, ErrBlocked) {
		t.Errorf("lookup failure must block, got %v", err)
	}
}

func TestValidate_RejectsIntegerEncodedLoopback(t *testing.T) {
	for _, target := range []string{
		"http://2130706433/", // 127.0.0.1
		"http://3232235777/", // 192.168.1.1
	} {
		err := Validate(target, Config{})
		if err == nil || !errors.Is(err, ErrBlocked) {
			t.Errorf("integer-encoded %q must block, got %v", target, err)
		}
	}
}

func TestValidate_RejectsShortDotted(t *testing.T) {
	for _, target := range []string{
		"http://127.1/",
		"http://127.0.1/",
		"http://10.1/",
		"http://192.168.1/",
	} {
		err := Validate(target, Config{})
		if err == nil || !errors.Is(err, ErrBlocked) {
			t.Errorf("short-dotted %q must block, got %v", target, err)
		}
	}
}

func TestValidate_IntegerPublicPasses(t *testing.T) {
	// 16843009 = 1.1.1.1
	if err := Validate("http://16843009/", Config{}); err != nil {
		t.Errorf("integer-encoded public IP must pass, got %v", err)
	}
}

func TestValidate_CustomSchemeAllowList(t *testing.T) {
	cfg := Config{
		AllowedSchemes: []string{"wss"},
		Resolver: stubResolver{ips: map[string][]net.IP{
			"example.com": {net.ParseIP("203.0.113.7")},
		}},
	}
	if err := Validate("wss://example.com/", cfg); err != nil {
		t.Errorf("wss opt-in must pass, got %v", err)
	}
}

func TestParseIntegerIP(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"2130706433", "127.0.0.1"},
		{"16843009", "1.1.1.1"},
		{"", ""},
		{"abc", ""},
		{"12345678901", ""}, // too long
	}
	for _, c := range cases {
		got := ParseIntegerIP(c.host)
		if c.want == "" {
			if got != nil {
				t.Errorf("%q: expected nil, got %s", c.host, got)
			}
			continue
		}
		if got == nil || got.String() != c.want {
			t.Errorf("%q: want %s, got %v", c.host, c.want, got)
		}
	}
}

func TestParseShortDottedIP(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"127.1", "127.0.0.1"},
		{"10.1", "10.0.0.1"},
		{"192.168.1", "192.168.0.1"},
		{"", ""},
		{"1.2.3.4", ""}, // full form is not short
		{"1.2.3.4.5", ""},
		{"1.500", "1.0.1.244"}, // 500 → 0x1F4 in low 24 bits
		{"256.1", ""},          // first octet > 255
	}
	for _, c := range cases {
		got := ParseShortDottedIP(c.host)
		if c.want == "" {
			if got != nil {
				t.Errorf("%q: expected nil, got %s", c.host, got)
			}
			continue
		}
		if got == nil || got.String() != c.want {
			t.Errorf("%q: want %s, got %v", c.host, c.want, got)
		}
	}
}
