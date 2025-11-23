package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig tests default configuration creation
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	require.NotNil(t, config)

	// Server defaults
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, 8443, config.Server.Port)
	assert.True(t, config.Server.EnableHTTP3)
	assert.Equal(t, "certs/server.crt", config.Server.TLSCertFile)
	assert.Equal(t, "certs/server.key", config.Server.TLSKeyFile)
	assert.Equal(t, 30, config.Server.ReadTimeout)
	assert.Equal(t, 30, config.Server.WriteTimeout)
	assert.Equal(t, int64(100*1024*1024), config.Server.MaxUploadSize)

	// Security defaults
	assert.True(t, config.Security.EnableAuth)
	assert.Equal(t, "X-API-Key", config.Security.APIKeyHeader)
	assert.Equal(t, 10, config.Security.RateLimitRPS)
	assert.Equal(t, 20, config.Security.RateLimitBurst)
	assert.Contains(t, config.Security.CORSOrigins, "*")

	// Translation defaults
	assert.Equal(t, "openai", config.Translation.DefaultProvider)
	assert.True(t, config.Translation.CacheEnabled)
	assert.Equal(t, 3600, config.Translation.CacheTTL)
	assert.Equal(t, 5, config.Translation.MaxConcurrent)
	assert.NotNil(t, config.Translation.Providers)

	// Logging defaults
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
}

// TestLoadConfig_Success tests successful config loading
func TestLoadConfig_Success(t *testing.T) {
	// Create temporary config file
	configJSON := `{
  "server": {
    "host": "localhost",
    "port": 9000,
    "enable_http3": false
  },
  "security": {
    "enable_auth": false,
    "jwt_secret": "test-secret",
    "rate_limit_rps": 100
  },
  "translation": {
    "default_provider": "openai",
    "cache_enabled": true,
    "cache_ttl": 7200,
    "providers": {}
  },
  "logging": {
    "level": "debug",
    "format": "text"
  }
}`

	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configJSON)
	require.NoError(t, err)
	tmpFile.Close()

	// Load config
	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify values
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, 9000, config.Server.Port)
	assert.False(t, config.Server.EnableHTTP3)
	assert.False(t, config.Security.EnableAuth)
	assert.Equal(t, "test-secret", config.Security.JWTSecret)
	assert.Equal(t, 100, config.Security.RateLimitRPS)
	assert.Equal(t, "openai", config.Translation.DefaultProvider)
	assert.Equal(t, 7200, config.Translation.CacheTTL)
	assert.Equal(t, "debug", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
}

// TestLoadConfig_FileNotFound tests missing config file
func TestLoadConfig_FileNotFound(t *testing.T) {
	config, err := LoadConfig("/non/existent/config.json")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read config file")
}

// TestLoadConfig_InvalidJSON tests invalid JSON parsing
func TestLoadConfig_InvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid json {")
	require.NoError(t, err)
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse config")
}

// TestSaveConfig tests saving configuration
func TestSaveConfig(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := DefaultConfig()
	config.Server.Port = 8080
	config.Translation.DefaultProvider = "deepseek"

	err = SaveConfig(tmpFile.Name(), config)
	require.NoError(t, err)

	// Load back and verify
	loaded, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, 8080, loaded.Server.Port)
	assert.Equal(t, "deepseek", loaded.Translation.DefaultProvider)
}

// TestSaveConfig_InvalidPath tests saving to invalid path
func TestSaveConfig_InvalidPath(t *testing.T) {
	config := DefaultConfig()
	err := SaveConfig("/invalid/path/config.json", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write config file")
}

// TestConfig_LoadAPIKeysFromEnv tests loading API keys from environment
func TestConfig_LoadAPIKeysFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	os.Setenv("ANTHROPIC_API_KEY", "test-anthropic-key")
	os.Setenv("ZHIPU_API_KEY", "test-zhipu-key")
	os.Setenv("DEEPSEEK_API_KEY", "test-deepseek-key")
	os.Setenv("JWT_SECRET", "test-jwt-secret")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ZHIPU_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("JWT_SECRET")
	}()

	config := DefaultConfig()
	config.loadAPIKeysFromEnv()

	// Verify API keys loaded
	assert.Equal(t, "test-openai-key", config.Translation.Providers["openai"].APIKey)
	assert.Equal(t, "test-anthropic-key", config.Translation.Providers["anthropic"].APIKey)
	assert.Equal(t, "test-zhipu-key", config.Translation.Providers["zhipu"].APIKey)
	assert.Equal(t, "test-deepseek-key", config.Translation.Providers["deepseek"].APIKey)
	assert.Equal(t, "test-jwt-secret", config.Security.JWTSecret)
}

// TestConfig_LoadAPIKeysFromEnv_PartialKeys tests loading with some keys missing
func TestConfig_LoadAPIKeysFromEnv_PartialKeys(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	config := DefaultConfig()
	config.loadAPIKeysFromEnv()

	// Only OpenAI should be set
	assert.Equal(t, "test-key", config.Translation.Providers["openai"].APIKey)
	assert.NotContains(t, config.Translation.Providers, "anthropic")
}

// TestConfig_LoadAPIKeysFromEnv_ExistingProviders tests updating existing providers
func TestConfig_LoadAPIKeysFromEnv_ExistingProviders(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "env-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	config := DefaultConfig()
	config.Translation.Providers["openai"] = ProviderConfig{
		Model:   "gpt-4",
		BaseURL: "https://api.openai.com",
	}

	config.loadAPIKeysFromEnv()

	// Should update API key but keep other fields
	assert.Equal(t, "env-key", config.Translation.Providers["openai"].APIKey)
	assert.Equal(t, "gpt-4", config.Translation.Providers["openai"].Model)
	assert.Equal(t, "https://api.openai.com", config.Translation.Providers["openai"].BaseURL)
}

// TestConfig_Validate_Success tests successful validation
func TestConfig_Validate_Success(t *testing.T) {
	config := DefaultConfig()
	config.Security.JWTSecret = "valid-secret"

	err := config.Validate()
	assert.NoError(t, err)
}

// TestConfig_Validate_InvalidPort tests port validation
func TestConfig_Validate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"port too low", 0},
		{"port negative", -1},
		{"port too high", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Server.Port = tt.port

			err := config.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid server port")
		})
	}
}

// TestConfig_Validate_HTTP3WithoutTLS tests HTTP/3 TLS requirement
func TestConfig_Validate_HTTP3WithoutTLS(t *testing.T) {
	tests := []struct {
		name        string
		certFile    string
		keyFile     string
		shouldError bool
	}{
		{"missing both", "", "", true},
		{"missing cert", "", "key.pem", true},
		{"missing key", "cert.pem", "", true},
		{"both present", "cert.pem", "key.pem", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Server.EnableHTTP3 = true
			config.Server.TLSCertFile = tt.certFile
			config.Server.TLSKeyFile = tt.keyFile
			config.Security.JWTSecret = "secret"

			err := config.Validate()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "TLS certificate and key files are required")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfig_Validate_AuthWithoutJWT tests auth without JWT secret
func TestConfig_Validate_AuthWithoutJWT(t *testing.T) {
	config := DefaultConfig()
	config.Security.EnableAuth = true
	config.Security.JWTSecret = ""

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWT secret is required")
}

// TestConfig_Validate_AuthDisabled tests validation with auth disabled
func TestConfig_Validate_AuthDisabled(t *testing.T) {
	config := DefaultConfig()
	config.Security.EnableAuth = false
	config.Security.JWTSecret = ""
	config.Server.EnableHTTP3 = false

	err := config.Validate()
	assert.NoError(t, err, "Should not require JWT secret when auth is disabled")
}

// TestConfig_RoundTrip tests saving and loading
func TestConfig_RoundTrip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create config with custom values
	original := DefaultConfig()
	original.Server.Host = "192.168.1.1"
	original.Server.Port = 9999
	original.Security.RateLimitRPS = 500
	original.Translation.DefaultProvider = "anthropic"
	original.Translation.CacheTTL = 1800
	original.Logging.Level = "warn"

	// Save
	err = SaveConfig(tmpFile.Name(), original)
	require.NoError(t, err)

	// Load
	loaded, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Server.Host, loaded.Server.Host)
	assert.Equal(t, original.Server.Port, loaded.Server.Port)
	assert.Equal(t, original.Security.RateLimitRPS, loaded.Security.RateLimitRPS)
	assert.Equal(t, original.Translation.DefaultProvider, loaded.Translation.DefaultProvider)
	assert.Equal(t, original.Translation.CacheTTL, loaded.Translation.CacheTTL)
	assert.Equal(t, original.Logging.Level, loaded.Logging.Level)
}

// TestProviderConfig tests provider configuration
func TestProviderConfig(t *testing.T) {
	config := DefaultConfig()

	// Add provider with options
	config.Translation.Providers["test-provider"] = ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "https://test.api",
		Model:   "test-model",
		Options: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2000,
		},
	}

	// Verify provider config
	provider := config.Translation.Providers["test-provider"]
	assert.Equal(t, "test-key", provider.APIKey)
	assert.Equal(t, "https://test.api", provider.BaseURL)
	assert.Equal(t, "test-model", provider.Model)
	assert.Equal(t, 0.7, provider.Options["temperature"])
	assert.Equal(t, 2000, provider.Options["max_tokens"])
}

// TestConfig_FilePermissions tests file permission on save
func TestConfig_FilePermissions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := DefaultConfig()
	err = SaveConfig(tmpFile.Name(), config)
	require.NoError(t, err)

	// Check file permissions (should be 0600)
	info, err := os.Stat(tmpFile.Name())
	require.NoError(t, err)

	perm := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0600), perm, "Config file should have 0600 permissions")
}

// BenchmarkLoadConfig benchmarks config loading
func BenchmarkLoadConfig(b *testing.B) {
	// Create temporary config
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	config := DefaultConfig()
	SaveConfig(tmpFile.Name(), config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadConfig(tmpFile.Name())
	}
}

// BenchmarkSaveConfig benchmarks config saving
func BenchmarkSaveConfig(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SaveConfig(tmpFile.Name(), config)
	}
}

// BenchmarkValidate benchmarks config validation
func BenchmarkValidate(b *testing.B) {
	config := DefaultConfig()
	config.Security.JWTSecret = "test-secret"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}
