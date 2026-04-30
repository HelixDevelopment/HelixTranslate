package llmverifier

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"digital.vasic.llmsverifier/config"
)

// LoadConfig loads the configuration from a YAML file with validation and advanced features
func LoadConfig(filePath string) (*config.Config, error) {
	return LoadConfigWithProfile(filePath, "")
}

// LoadConfigWithProfile loads configuration with profile support
func LoadConfigWithProfile(filePath, profile string) (*config.Config, error) {
	// Set up viper with profile support
	if err := setupViper(filePath, profile); err != nil {
		return nil, fmt.Errorf("error setting up viper: %w", err)
	}

	// Set default values
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Apply profile-specific overrides
	if profile != "" {
		if err := applyProfileOverrides(&cfg, profile); err != nil {
			return nil, fmt.Errorf("error applying profile overrides: %w", err)
		}
	}

	// Expand environment variables in the config
	if err := expandEnvironmentVariables(&cfg); err != nil {
		return nil, fmt.Errorf("error expanding environment variables: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set computed defaults
	setComputedDefaults(&cfg)

	return &cfg, nil
}

// setupViper configures viper with profile support
func setupViper(filePath, profile string) error {
	viper.AutomaticEnv() // Allow environment variables to override config
	viper.SetEnvPrefix("LLM_VERIFIER")

	// Set config file
	if profile != "" {
		// Look for profile-specific config file first
		ext := filepath.Ext(filePath)
		baseName := strings.TrimSuffix(filePath, ext)
		profileFile := fmt.Sprintf("%s.%s%s", baseName, profile, ext)

		if _, err := os.Stat(profileFile); err == nil {
			viper.SetConfigFile(profileFile)
		} else {
			viper.SetConfigFile(filePath)
		}
	} else {
		viper.SetConfigFile(filePath)
	}

	return nil
}

// applyProfileOverrides applies profile-specific configuration overrides
func applyProfileOverrides(cfg *config.Config, profile string) error {
	switch strings.ToLower(profile) {
	case "dev", "development":
		// Development profile: relaxed security, debug logging
		cfg.Logging.Level = "debug"
		cfg.Logging.Output = "stdout"
		cfg.API.EnableCORS = true
		cfg.API.CORSOrigins = "http://localhost:3000,http://localhost:4200"
		cfg.Database.Path = "llm_verifier_dev.db"
		cfg.Security.EnableRateLimiting = false
		cfg.Monitoring.EnableMetrics = true
		cfg.Monitoring.MetricsPort = "9090"

	case "prod", "production":
		// Production profile: strict security, structured logging
		cfg.Logging.Level = "info"
		cfg.Logging.Format = "json"
		cfg.Logging.Output = "file"
		cfg.Logging.FilePath = "/var/log/llm-verifier.log"
		cfg.API.EnableHTTPS = true
		cfg.Security.EnableRateLimiting = true
		cfg.Security.EnableIPWhitelist = true
		cfg.Monitoring.EnableMetrics = true
		cfg.Monitoring.EnableTracing = true

	case "test", "testing":
		// Test profile: minimal logging, in-memory database
		cfg.Logging.Level = "error"
		cfg.Logging.Output = "stdout"
		cfg.Database.Path = ":memory:"
		cfg.Security.EnableRateLimiting = false
		cfg.Monitoring.EnableMetrics = false
		cfg.Concurrency = 1
		cfg.Timeout = 5 * time.Second
	}

	return nil
}

// WatchConfig watches for configuration file changes and calls the callback
func WatchConfig(filePath string, callback func(*config.Config)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Write) {
					// Config file changed, reload it
					cfg, err := LoadConfig(filePath)
					if err != nil {
						// Log error but continue watching
						fmt.Printf("Error reloading config: %v\n", err)
						continue
					}

					// Call the callback with new config
					callback(cfg)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Config watcher error: %v\n", err)
			}
		}
	}()

	// Start watching the config file
	if err := watcher.Add(filePath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Profile defaults
	viper.SetDefault("profile", "dev")

	// Global defaults
	viper.SetDefault("global.max_retries", 3)
	viper.SetDefault("global.request_delay", 1*time.Second)
	viper.SetDefault("global.timeout", 30*time.Second)

	// Database defaults
	viper.SetDefault("database.path", "llm_verifier.db")

	// API defaults
	viper.SetDefault("api.port", "8080")
	viper.SetDefault("api.jwt_secret", "your-secret-key-change-in-production") // Default for development
	viper.SetDefault("api.rate_limit", 100)
	viper.SetDefault("api.burst_limit", 200)
	viper.SetDefault("api.rate_limit_window", 60)
	viper.SetDefault("api.enable_cors", false)
	viper.SetDefault("api.cors_origins", "*")
	viper.SetDefault("api.cors_methods", "GET,POST,PUT,DELETE,OPTIONS")
	viper.SetDefault("api.cors_headers", "Origin,Content-Type,Accept,Authorization,X-Requested-With")
	viper.SetDefault("api.enable_https", false)
	viper.SetDefault("api.read_timeout", 30)
	viper.SetDefault("api.write_timeout", 30)
	viper.SetDefault("api.max_header_bytes", 1048576) // 1MB

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)
	viper.SetDefault("logging.compress", true)

	// Monitoring defaults
	viper.SetDefault("monitoring.enable_metrics", false)
	viper.SetDefault("monitoring.metrics_port", "9090")
	viper.SetDefault("monitoring.enable_health", true)
	viper.SetDefault("monitoring.health_port", "8086")
	viper.SetDefault("monitoring.enable_tracing", false)
	viper.SetDefault("monitoring.enable_profiling", false)
	viper.SetDefault("monitoring.profiling_port", "6060")

	// Security defaults
	viper.SetDefault("security.enable_rate_limiting", true)
	viper.SetDefault("security.enable_ip_whitelist", false)
	viper.SetDefault("security.enable_request_logging", true)
	viper.SetDefault("security.sensitive_headers", []string{"authorization", "x-api-key", "cookie"})
	viper.SetDefault("security.enable_csrf_protection", false)
	viper.SetDefault("security.csrf_token_length", 32)
	viper.SetDefault("security.session_timeout", 60)

	// Application defaults
	viper.SetDefault("concurrency", 1)
	viper.SetDefault("timeout", 60*time.Second)
}

// expandEnvironmentVariables expands environment variables in configuration
func expandEnvironmentVariables(cfg *config.Config) error {
	// Expand global API key
	cfg.Global.APIKey = expandEnvVar(cfg.Global.APIKey)

	// Expand global base URL
	cfg.Global.BaseURL = expandEnvVar(cfg.Global.BaseURL)

	// Expand database encryption key
	cfg.Database.EncryptionKey = expandEnvVar(cfg.Database.EncryptionKey)

	// Expand API JWT secret
	cfg.API.JWTSecret = expandEnvVar(cfg.API.JWTSecret)

	// Expand API TLS files
	cfg.API.TLSCertFile = expandEnvVar(cfg.API.TLSCertFile)
	cfg.API.TLSKeyFile = expandEnvVar(cfg.API.TLSKeyFile)

	// Expand logging file path
	cfg.Logging.FilePath = expandEnvVar(cfg.Logging.FilePath)

	// Expand monitoring endpoints
	cfg.Monitoring.TracingEndpoint = expandEnvVar(cfg.Monitoring.TracingEndpoint)

	// Expand LLM configurations
	for i := range cfg.LLMs {
		cfg.LLMs[i].Name = expandEnvVar(cfg.LLMs[i].Name)
		cfg.LLMs[i].Endpoint = expandEnvVar(cfg.LLMs[i].Endpoint)
		cfg.LLMs[i].APIKey = expandEnvVar(cfg.LLMs[i].APIKey)
		cfg.LLMs[i].Model = expandEnvVar(cfg.LLMs[i].Model)

		// Expand headers
		if cfg.LLMs[i].Headers != nil {
			for k, v := range cfg.LLMs[i].Headers {
				cfg.LLMs[i].Headers[k] = expandEnvVar(v)
			}
		}
	}

	return nil
}

// expandEnvVar expands a single environment variable with fallback
func expandEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
		// If environment variable is not set, return the original value
		return value
	}
	return os.ExpandEnv(value)
}

// validateConfig validates the configuration
func validateConfig(cfg *config.Config) error {
	// Validate JWT secret for production use
	if cfg.API.JWTSecret == "your-secret-key-change-in-production" {
		if profile := os.Getenv("LLM_VERIFIER_PROFILE"); profile == "prod" || profile == "production" {
			return fmt.Errorf("production deployment detected: JWT secret must be changed from default value")
		}
		fmt.Fprintf(os.Stderr, "WARNING: Using default JWT secret. Please set LLM_VERIFIER_API_JWT_SECRET environment variable in production.\n")
	}

	// Validate concurrency
	if cfg.Concurrency < 1 || cfg.Concurrency > 100 {
		return fmt.Errorf("concurrency must be between 1 and 100, got %d", cfg.Concurrency)
	}

	// Validate timeout
	if cfg.Timeout < 1*time.Second || cfg.Timeout > 10*time.Minute {
		return fmt.Errorf("timeout must be between 1s and 10m, got %v", cfg.Timeout)
	}

	// Validate global config
	if err := validateGlobalConfig(&cfg.Global); err != nil {
		return fmt.Errorf("global config validation failed: %w", err)
	}

	// Validate database config
	if err := validateDatabaseConfig(&cfg.Database); err != nil {
		return fmt.Errorf("database config validation failed: %w", err)
	}

	// Validate API config
	if err := validateAPIConfig(&cfg.API); err != nil {
		return fmt.Errorf("API config validation failed: %w", err)
	}

	// Validate LLM configs
	for i, llm := range cfg.LLMs {
		if err := validateLLMConfig(&llm, i); err != nil {
			return fmt.Errorf("LLM config %d validation failed: %w", i, err)
		}
	}

	// Validate logging config
	if err := validateLoggingConfig(&cfg.Logging); err != nil {
		return fmt.Errorf("logging config validation failed: %w", err)
	}

	// Validate monitoring config
	if err := validateMonitoringConfig(&cfg.Monitoring); err != nil {
		return fmt.Errorf("monitoring config validation failed: %w", err)
	}

	// Validate security config
	if err := validateSecurityConfig(&cfg.Security); err != nil {
		return fmt.Errorf("security config validation failed: %w", err)
	}

	return nil
}

// validateGlobalConfig validates global configuration
func validateGlobalConfig(global *config.GlobalConfig) error {
	// Validate base URL if provided
	if global.BaseURL != "" && !strings.HasPrefix(global.BaseURL, "http") {
		return fmt.Errorf("base_url must start with http:// or https://")
	}

	// Validate max retries
	if global.MaxRetries < 0 || global.MaxRetries > 10 {
		return fmt.Errorf("max_retries must be between 0 and 10, got %d", global.MaxRetries)
	}

	// Validate request delay
	if global.RequestDelay < 0 || global.RequestDelay > 1*time.Minute {
		return fmt.Errorf("request_delay must be between 0 and 1m, got %v", global.RequestDelay)
	}

	// Validate timeout
	if global.Timeout < 1*time.Second || global.Timeout > 10*time.Minute {
		return fmt.Errorf("timeout must be between 1s and 10m, got %v", global.Timeout)
	}

	return nil
}

// validateDatabaseConfig validates database configuration
func validateDatabaseConfig(db *config.DatabaseConfig) error {
	// Validate database path
	if db.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Validate encryption key length if provided
	if db.EncryptionKey != "" && len(db.EncryptionKey) < 16 {
		return fmt.Errorf("encryption key must be at least 16 characters long")
	}

	return nil
}

// validateAPIConfig validates API configuration
func validateAPIConfig(api *config.APIConfig) error {
	// Validate port
	if port, err := strconv.Atoi(api.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("API port must be between 1 and 65535, got %s", api.Port)
	}

	// Validate rate limit
	if api.RateLimit < 1 || api.RateLimit > 10000 {
		return fmt.Errorf("rate_limit must be between 1 and 10000, got %d", api.RateLimit)
	}

	// Validate burst limit
	if api.BurstLimit < api.RateLimit {
		return fmt.Errorf("burst_limit (%d) must be greater than or equal to rate_limit (%d)", api.BurstLimit, api.RateLimit)
	}

	// Validate rate limit window
	if api.RateLimitWindow < 1 || api.RateLimitWindow > 3600 {
		return fmt.Errorf("rate_limit_window must be between 1 and 3600 seconds, got %d", api.RateLimitWindow)
	}

	return nil
}

// validateLLMConfig validates LLM configuration
func validateLLMConfig(llm *config.LLMConfig, index int) error {
	// Validate name
	if strings.TrimSpace(llm.Name) == "" {
		return fmt.Errorf("LLM[%d] name cannot be empty", index)
	}

	// Validate endpoint
	if llm.Endpoint == "" {
		return fmt.Errorf("LLM[%d] endpoint cannot be empty", index)
	}
	if !strings.HasPrefix(llm.Endpoint, "http") {
		return fmt.Errorf("LLM[%d] endpoint must start with http:// or https://", index)
	}

	// Validate API key (required for most providers)
	if llm.APIKey == "" && !strings.Contains(llm.Endpoint, "localhost") && !strings.Contains(llm.Endpoint, "127.0.0.1") {
		return fmt.Errorf("LLM API key is required for non-local endpoints")
	}

	return nil
}

// setComputedDefaults sets computed default values
func setComputedDefaults(cfg *config.Config) {
	// Set default model if not specified
	if cfg.Global.DefaultModel == "" && len(cfg.LLMs) > 0 {
		cfg.Global.DefaultModel = cfg.LLMs[0].Model
		if cfg.Global.DefaultModel == "" {
			cfg.Global.DefaultModel = cfg.LLMs[0].Name
		}
	}

	// Set default JWT secret if not provided
	if cfg.API.JWTSecret == "" {
		cfg.API.JWTSecret = generateSecureRandomString(32)
	}

	// Set default database encryption key if not provided
	if cfg.Database.EncryptionKey == "" {
		cfg.Database.EncryptionKey = generateSecureRandomString(32)
	}
}

// validateLoggingConfig validates logging configuration
func validateLoggingConfig(logging *config.LoggingConfig) error {
	validLevels := []string{"debug", "info", "warn", "error"}
	if !stringSliceContains(validLevels, logging.Level) {
		return fmt.Errorf("invalid log level '%s', must be one of: %v", logging.Level, validLevels)
	}

	validFormats := []string{"json", "text"}
	if !stringSliceContains(validFormats, logging.Format) {
		return fmt.Errorf("invalid log format '%s', must be one of: %v", logging.Format, validFormats)
	}

	validOutputs := []string{"stdout", "stderr", "file"}
	if !stringSliceContains(validOutputs, logging.Output) {
		return fmt.Errorf("invalid log output '%s', must be one of: %v", logging.Output, validOutputs)
	}

	if logging.Output == "file" && logging.FilePath == "" {
		return fmt.Errorf("file_path is required when output is 'file'")
	}

	if logging.MaxSize < 1 || logging.MaxSize > 1000 {
		return fmt.Errorf("max_size must be between 1 and 1000 MB, got %d", logging.MaxSize)
	}

	if logging.MaxBackups < 1 || logging.MaxBackups > 10 {
		return fmt.Errorf("max_backups must be between 1 and 10, got %d", logging.MaxBackups)
	}

	if logging.MaxAge < 1 || logging.MaxAge > 365 {
		return fmt.Errorf("max_age must be between 1 and 365 days, got %d", logging.MaxAge)
	}

	return nil
}

// validateMonitoringConfig validates monitoring configuration
func validateMonitoringConfig(monitoring *config.MonitoringConfig) error {
	if monitoring.EnableMetrics {
		if port, err := strconv.Atoi(monitoring.MetricsPort); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid metrics port '%s', must be between 1 and 65535", monitoring.MetricsPort)
		}
	}

	if monitoring.EnableHealth {
		if port, err := strconv.Atoi(monitoring.HealthPort); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid health port '%s', must be between 1 and 65535", monitoring.HealthPort)
		}
	}

	if monitoring.EnableTracing && monitoring.TracingEndpoint == "" {
		return fmt.Errorf("tracing_endpoint is required when enable_tracing is true")
	}

	if monitoring.EnableProfiling {
		if port, err := strconv.Atoi(monitoring.ProfilingPort); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid profiling port '%s', must be between 1 and 65535", monitoring.ProfilingPort)
		}
	}

	return nil
}

// validateSecurityConfig validates security configuration
func validateSecurityConfig(security *config.SecurityConfig) error {
	if security.CSRFTokenLength < 16 || security.CSRFTokenLength > 128 {
		return fmt.Errorf("csrf_token_length must be between 16 and 128, got %d", security.CSRFTokenLength)
	}

	if security.SessionTimeout < 5 || security.SessionTimeout > 1440 {
		return fmt.Errorf("session_timeout must be between 5 and 1440 minutes, got %d", security.SessionTimeout)
	}

	return nil
}

// stringSliceContains checks if a slice contains a string
func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// generateSecureRandomString generates a secure random string for secrets
func generateSecureRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	randomBytes := make([]byte, length)

	// Generate secure random bytes
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to simple generation if crypto/rand fails
		for i := range result {
			result[i] = charset[i%len(charset)]
		}
		return string(result)
	}

	// Map random bytes to charset
	for i := range result {
		result[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(result)
}
