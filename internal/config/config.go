package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `json:"server"`
	Security   SecurityConfig   `json:"security"`
	Translation TranslationConfig `json:"translation"`
	Logging    LoggingConfig    `json:"logging"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	EnableHTTP3   bool   `json:"enable_http3"`
	TLSCertFile   string `json:"tls_cert_file"`
	TLSKeyFile    string `json:"tls_key_file"`
	ReadTimeout   int    `json:"read_timeout"`
	WriteTimeout  int    `json:"write_timeout"`
	MaxUploadSize int64  `json:"max_upload_size"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	EnableAuth     bool   `json:"enable_auth"`
	JWTSecret      string `json:"jwt_secret"`
	APIKeyHeader   string `json:"api_key_header"`
	RateLimitRPS   int    `json:"rate_limit_rps"`
	RateLimitBurst int    `json:"rate_limit_burst"`
	CORSOrigins    []string `json:"cors_origins"`
}

// TranslationConfig represents translation configuration
type TranslationConfig struct {
	DefaultProvider   string            `json:"default_provider"`
	DefaultModel      string            `json:"default_model"`
	CacheEnabled      bool              `json:"cache_enabled"`
	CacheTTL          int               `json:"cache_ttl"`
	MaxConcurrent     int               `json:"max_concurrent"`
	Providers         map[string]ProviderConfig `json:"providers"`
}

// ProviderConfig represents LLM provider configuration
type ProviderConfig struct {
	APIKey   string            `json:"api_key,omitempty"`
	BaseURL  string            `json:"base_url,omitempty"`
	Model    string            `json:"model"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputFile string `json:"output_file"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:          "0.0.0.0",
			Port:          8443,
			EnableHTTP3:   true,
			TLSCertFile:   "certs/server.crt",
			TLSKeyFile:    "certs/server.key",
			ReadTimeout:   30,
			WriteTimeout:  30,
			MaxUploadSize: 100 * 1024 * 1024, // 100MB
		},
		Security: SecurityConfig{
			EnableAuth:     true,
			JWTSecret:      "",
			APIKeyHeader:   "X-API-Key",
			RateLimitRPS:   10,
			RateLimitBurst: 20,
			CORSOrigins:    []string{"*"},
		},
		Translation: TranslationConfig{
			DefaultProvider:   "dictionary",
			DefaultModel:      "",
			CacheEnabled:      true,
			CacheTTL:          3600,
			MaxConcurrent:     5,
			Providers:         make(map[string]ProviderConfig),
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "",
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Load API keys from environment variables
	config.loadAPIKeysFromEnv()

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(filename string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadAPIKeysFromEnv loads API keys from environment variables
func (c *Config) loadAPIKeysFromEnv() {
	envMappings := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"zhipu":     "ZHIPU_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
	}

	for provider, envVar := range envMappings {
		if key := os.Getenv(envVar); key != "" {
			if providerConfig, ok := c.Translation.Providers[provider]; ok {
				providerConfig.APIKey = key
				c.Translation.Providers[provider] = providerConfig
			} else {
				c.Translation.Providers[provider] = ProviderConfig{
					APIKey: key,
				}
			}
		}
	}

	// Load JWT secret
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		c.Security.JWTSecret = jwtSecret
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Server.EnableHTTP3 {
		if c.Server.TLSCertFile == "" || c.Server.TLSKeyFile == "" {
			return fmt.Errorf("TLS certificate and key files are required for HTTP/3")
		}
	}

	if c.Security.EnableAuth && c.Security.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required when authentication is enabled")
	}

	return nil
}
