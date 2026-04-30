package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// LoggingConfig holds logging configuration options
type LoggingConfig struct {
	Level      string `mapstructure:"level"`       // Log level (debug, info, warn, error)
	Format     string `mapstructure:"format"`      // Log format (json, text)
	Output     string `mapstructure:"output"`      // Log output (stdout, stderr, file)
	FilePath   string `mapstructure:"file_path"`   // Log file path (if output is file)
	MaxSize    int    `mapstructure:"max_size"`    // Max log file size in MB
	MaxBackups int    `mapstructure:"max_backups"` // Max number of log file backups
	MaxAge     int    `mapstructure:"max_age"`     // Max age of log files in days
	Compress   bool   `mapstructure:"compress"`    // Compress old log files
}

// MonitoringConfig holds monitoring and metrics configuration
type MonitoringConfig struct {
	EnableMetrics   bool   `mapstructure:"enable_metrics"`   // Enable Prometheus metrics
	MetricsPort     string `mapstructure:"metrics_port"`     // Port for metrics endpoint
	EnableHealth    bool   `mapstructure:"enable_health"`    // Enable health check endpoint
	HealthPort      string `mapstructure:"health_port"`      // Port for health check endpoint
	EnableTracing   bool   `mapstructure:"enable_tracing"`   // Enable distributed tracing
	TracingEndpoint string `mapstructure:"tracing_endpoint"` // Tracing collector endpoint
	EnableProfiling bool   `mapstructure:"enable_profiling"` // Enable Go profiling
	ProfilingPort   string `mapstructure:"profiling_port"`   // Port for profiling endpoint
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	EnableRateLimiting   bool     `mapstructure:"enable_rate_limiting"`   // Enable rate limiting
	EnableIPWhitelist    bool     `mapstructure:"enable_ip_whitelist"`    // Enable IP whitelisting
	IPWhitelist          []string `mapstructure:"ip_whitelist"`           // List of allowed IP addresses/CIDRs
	EnableRequestLogging bool     `mapstructure:"enable_request_logging"` // Log all requests
	SensitiveHeaders     []string `mapstructure:"sensitive_headers"`      // Headers to redact in logs
	EnableCSRFProtection bool     `mapstructure:"enable_csrf_protection"` // Enable CSRF protection
	CSRFTokenLength      int      `mapstructure:"csrf_token_length"`      // CSRF token length
	SessionTimeout       int      `mapstructure:"session_timeout"`        // Session timeout in minutes
}

// NotificationsConfig holds notification system configuration
type NotificationsConfig struct {
	Slack    SlackConfig    `mapstructure:"slack"`
	Email    EmailConfig    `mapstructure:"email"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	Matrix   MatrixConfig   `mapstructure:"matrix"`
	WhatsApp WhatsAppConfig `mapstructure:"whatsapp"`
}

// SlackConfig holds Slack notification configuration
type SlackConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	SMTPHost         string `mapstructure:"smtp_host"`
	SMTPPort         int    `mapstructure:"smtp_port"`
	Username         string `mapstructure:"username"`
	Password         string `mapstructure:"password"`
	DefaultRecipient string `mapstructure:"default_recipient"`
}

// TelegramConfig holds Telegram notification configuration
type TelegramConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
	ChatID   string `mapstructure:"chat_id"`
}

// MatrixConfig holds Matrix notification configuration
type MatrixConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	HomeserverURL string `mapstructure:"homeserver_url"`
	AccessToken   string `mapstructure:"access_token"`
	RoomID        string `mapstructure:"room_id"`
}

// WhatsAppConfig holds WhatsApp notification configuration
type WhatsAppConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	APIKey           string `mapstructure:"api_key"`
	PhoneNumberID    string `mapstructure:"phone_number_id"`
	DefaultRecipient string `mapstructure:"default_recipient"`
}

// Config represents the main configuration for the LLM verifier
type Config struct {
	Profile       string              `mapstructure:"profile"` // Configuration profile (dev, prod, test)
	LLMs          []LLMConfig         `mapstructure:"llms"`
	Global        GlobalConfig        `mapstructure:"global"`
	Database      DatabaseConfig      `mapstructure:"database"`
	API           APIConfig           `mapstructure:"api"`
	Concurrency   int                 `mapstructure:"concurrency"`
	Timeout       time.Duration       `mapstructure:"timeout"`
	Logging       LoggingConfig       `mapstructure:"logging"`
	Monitoring    MonitoringConfig    `mapstructure:"monitoring"`
	Security      SecurityConfig      `mapstructure:"security"`
	Notifications NotificationsConfig `mapstructure:"notifications"`
}

// LLMConfig represents configuration for a single LLM endpoint
type LLMConfig struct {
	Name     string            `mapstructure:"name"`               // Name of the LLM service
	Endpoint string            `mapstructure:"endpoint"`           // API endpoint URL
	APIKey   string            `mapstructure:"api_key"`            // API key for authentication
	Model    string            `mapstructure:"model,omitempty"`    // Specific model to test (optional if auto-discovery)
	Headers  map[string]string `mapstructure:"headers,omitempty"`  // Additional headers to send with requests
	Features map[string]bool   `mapstructure:"features,omitempty"` // Expected features of the model
}

// GlobalConfig holds global configuration options
type GlobalConfig struct {
	BaseURL      string         `mapstructure:"base_url"`      // Base URL for the API
	APIKey       string         `mapstructure:"api_key"`       // Global API key
	DefaultModel string         `mapstructure:"default_model"` // Default model name
	MaxRetries   int            `mapstructure:"max_retries"`   // Maximum number of retries for failed requests
	RequestDelay time.Duration  `mapstructure:"request_delay"` // Delay between requests
	Timeout      time.Duration  `mapstructure:"timeout"`       // Request timeout
	CustomParams map[string]any `mapstructure:"custom_params"` // Custom parameters for API calls
}

// DatabaseConfig holds database configuration options
type DatabaseConfig struct {
	Path          string `mapstructure:"path"`           // Path to the database file
	EncryptionKey string `mapstructure:"encryption_key"` // Encryption key for SQL Cipher
}

// APIConfig holds REST API configuration options
type APIConfig struct {
	Port              string `mapstructure:"port"`                  // Port to run the API server on
	JWTSecret         string `mapstructure:"jwt_secret"`            // Secret key for JWT token signing
	RateLimit         int    `mapstructure:"rate_limit"`            // Global rate limit (requests per minute)
	BurstLimit        int    `mapstructure:"burst_limit"`           // Burst limit for short periods
	RateLimitWindow   int    `mapstructure:"rate_limit_window"`     // Rate limit window in seconds
	EnableCORS        bool   `mapstructure:"enable_cors"`           // Enable CORS headers
	TrustedProxies    string `mapstructure:"trusted_proxies"`       // Comma-separated list of trusted proxy IPs
	RateLimitByAPIKey bool   `mapstructure:"rate_limit_by_api_key"` // Rate limit by API key instead of IP
	CORSOrigins       string `mapstructure:"cors_origins"`          // Comma-separated list of allowed CORS origins
	CORSMethods       string `mapstructure:"cors_methods"`          // Comma-separated list of allowed CORS methods
	CORSHeaders       string `mapstructure:"cors_headers"`          // Comma-separated list of allowed CORS headers
	EnableHTTPS       bool   `mapstructure:"enable_https"`          // Enable HTTPS with TLS
	TLSCertFile       string `mapstructure:"tls_cert_file"`         // Path to TLS certificate file
	TLSKeyFile        string `mapstructure:"tls_key_file"`          // Path to TLS key file
	ReadTimeout       int    `mapstructure:"read_timeout"`          // HTTP read timeout in seconds
	WriteTimeout      int    `mapstructure:"write_timeout"`         // HTTP write timeout in seconds
	MaxHeaderBytes    int    `mapstructure:"max_header_bytes"`      // Maximum header size in bytes
}

// LoadFromFile loads configuration from a file path
// Supports YAML, JSON, and TOML formats based on file extension
func LoadFromFile(path string) (*Config, error) {
	ext := strings.ToLower(filepath.Ext(path))

	// For JSON files, use direct JSON parsing
	if ext == ".json" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	// For YAML/TOML files, use Viper
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveToFile saves configuration to a file path
func SaveToFile(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Export exports configuration to a file path (alias for SaveToFile)
func Export(cfg *Config, path string) error {
	return SaveToFile(cfg, path)
}
