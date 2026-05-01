package ssrf

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAllowedURL(t *testing.T) {
	// Test with a URL that resolves to a public IP
	cfg := Config{}
	err := Validate("https://example.com/path", cfg)
	// example.com resolves to public IPs, should pass
	if err != nil {
		// If DNS fails in test environment, skip rather than fail
		assert.Contains(t, err.Error(), "could not resolve")
	}
}

func TestValidateBlockedScheme(t *testing.T) {
	cfg := Config{
		AllowedSchemes: []string{"https"},
	}
	err := Validate("http://example.com", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "scheme")
}

func TestValidateAllowedScheme(t *testing.T) {
	cfg := Config{
		AllowedSchemes: []string{"https"},
	}
	err := Validate("https://example.com", cfg)
	// May fail on resolution but should pass scheme check
	if err != nil {
		assert.NotContains(t, err.Error(), "scheme")
	}
}

func TestValidateBlockedHost(t *testing.T) {
	cfg := Config{
		BlockedHosts: []string{"evil.com"},
	}
	err := Validate("https://evil.com/path", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "evil.com")
}

func TestValidateInvalidURL(t *testing.T) {
	cfg := Config{}
	err := Validate("://not-a-url", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestValidateLocalhostBlocked(t *testing.T) {
	cfg := Config{AllowLocalhost: false}
	err := Validate("http://localhost:8080/api", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "loopback")
}

func TestValidateLocalhostAllowed(t *testing.T) {
	cfg := Config{AllowLocalhost: true}
	err := Validate("http://localhost:8080/api", cfg)
	require.NoError(t, err)
}

func TestValidate127Allowed(t *testing.T) {
	cfg := Config{AllowLocalhost: true}
	err := Validate("http://127.0.0.1:8080/api", cfg)
	require.NoError(t, err)
}

func TestValidatePrivateIPBlocked(t *testing.T) {
	cfg := Config{AllowPrivateIPs: false}
	privateIPs := []string{
		"http://10.0.0.1/api",
		"http://172.16.0.1/api",
		"http://192.168.1.1/api",
	}
	for _, url := range privateIPs {
		t.Run(url, func(t *testing.T) {
			err := Validate(url, cfg)
			require.Error(t, err, "expected %s to be blocked", url)
			assert.ErrorIs(t, err, ErrBlocked)
			assert.Contains(t, err.Error(), "private")
		})
	}
}

func TestValidateLinkLocalBlocked(t *testing.T) {
	cfg := Config{AllowPrivateIPs: false}
	err := Validate("http://169.254.1.1/api", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "link-local")
}

func TestValidateLinkLocalAllowed(t *testing.T) {
	cfg := Config{AllowPrivateIPs: true}
	err := Validate("http://169.254.1.1/api", cfg)
	require.NoError(t, err)
}

func TestValidatePrivateIPAllowed(t *testing.T) {
	cfg := Config{AllowPrivateIPs: true}
	err := Validate("http://192.168.1.1/api", cfg)
	require.NoError(t, err)
}

func TestValidateMetadataBlocked(t *testing.T) {
	cfg := Config{AllowMetadataIPs: false}
	err := Validate("http://169.254.169.254/latest/meta-data/", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "metadata")
}

func TestValidateMetadataAllowed(t *testing.T) {
	cfg := Config{AllowMetadataIPs: true}
	err := Validate("http://169.254.169.254/latest/meta-data/", cfg)
	require.NoError(t, err)
}

func TestValidateUnresolvedHost(t *testing.T) {
	cfg := Config{}
	err := Validate("https://this-domain-definitely-does-not-exist-12345.xyz", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "could not resolve")
}

func TestIsPrivate(t *testing.T) {
	assert.True(t, isPrivate(netParseIP(t, "10.0.0.1")))
	assert.True(t, isPrivate(netParseIP(t, "172.16.0.1")))
	assert.True(t, isPrivate(netParseIP(t, "192.168.1.1")))
	// Link-local and loopback are handled separately, not by isPrivate
	assert.False(t, isPrivate(netParseIP(t, "169.254.1.1")))
	assert.False(t, isPrivate(netParseIP(t, "127.0.0.1")))
	assert.False(t, isPrivate(netParseIP(t, "8.8.8.8")))
}

func TestIsMetadata(t *testing.T) {
	assert.True(t, isMetadata(netParseIP(t, "169.254.169.254")))
	assert.False(t, isMetadata(netParseIP(t, "8.8.8.8")))
}

func TestValidateEmptyConfig(t *testing.T) {
	// Zero value config should be safe (block private ranges)
	cfg := Config{}
	assert.False(t, cfg.AllowLocalhost)
	assert.False(t, cfg.AllowPrivateIPs)
	assert.False(t, cfg.AllowMetadataIPs)
	assert.Empty(t, cfg.AllowedSchemes)
	assert.Empty(t, cfg.BlockedHosts)
}

func TestValidateMultipleChecks(t *testing.T) {
	// Test that blocked hosts are checked before DNS resolution
	cfg := Config{
		BlockedHosts: []string{"blocked.example.com"},
	}
	err := Validate("https://blocked.example.com/api", cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBlocked)
	assert.Contains(t, err.Error(), "blocked")
}

func netParseIP(t *testing.T, ip string) net.IP {
	t.Helper()
	parsed := net.ParseIP(ip)
	require.NotNil(t, parsed, "failed to parse IP: %s", ip)
	return parsed
}
