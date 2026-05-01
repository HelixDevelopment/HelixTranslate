package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server       ServerConfig       `json:"server"`
	Security     SecurityConfig     `json:"security"`
	Translation  TranslationConfig  `json:"translation"`
	Preparation  PreparationConfig  `json:"preparation"`
	Distributed  DistributedConfig  `json:"distributed"`
	Logging      LoggingConfig      `json:"logging"`
	LLMsVerifier LLMsVerifierConfig `json:"llmsverifier"`
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
	EnableAuth     bool     `json:"enable_auth"`
	JWTSecret      string   `json:"jwt_secret"`
	APIKeyHeader   string   `json:"api_key_header"`
	RateLimitRPS   int      `json:"rate_limit_rps"`
	RateLimitBurst int      `json:"rate_limit_burst"`
	CORSOrigins    []string `json:"cors_origins"`
}

// TranslationConfig represents translation configuration
type TranslationConfig struct {
	DefaultProvider string                    `json:"default_provider"`
	DefaultModel    string                    `json:"default_model"`
	CacheEnabled    bool                      `json:"cache_enabled"`
	CacheTTL        int                       `json:"cache_ttl"`
	MaxConcurrent   int                       `json:"max_concurrent"`
	Providers       map[string]ProviderConfig `json:"providers"`
}

// ProviderConfig represents LLM provider configuration
type ProviderConfig struct {
	APIKey  string                 `json:"api_key,omitempty"`
	BaseURL string                 `json:"base_url,omitempty"`
	Model   string                 `json:"model"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// DistributedConfig represents distributed work configuration
type DistributedConfig struct {
	Enabled             bool                    `json:"enabled"`
	Workers             map[string]WorkerConfig `json:"workers"`
	SSHTimeout          int                     `json:"ssh_timeout"`
	SSHMaxRetries       int                     `json:"ssh_max_retries"`
	HealthCheckInterval int                     `json:"health_check_interval"`
	MaxRemoteInstances  int                     `json:"max_remote_instances"`
}

// WorkerConfig represents a remote worker configuration
type WorkerConfig struct {
	Name        string   `json:"name"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	User        string   `json:"user"`
	KeyFile     string   `json:"key_file,omitempty"`
	Password    string   `json:"password,omitempty"`
	MaxCapacity int      `json:"max_capacity"`
	Tags        []string `json:"tags,omitempty"`
	Enabled     bool     `json:"enabled"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputFile string `json:"output_file"`
}

// PreparationConfig represents preparation phase configuration
type PreparationConfig struct {
	Enabled            bool     `json:"enabled"`
	PassCount          int      `json:"pass_count"`
	Providers          []string `json:"providers"`
	AnalyzeContentType bool     `json:"analyze_content_type"`
	AnalyzeCharacters  bool     `json:"analyze_characters"`
	AnalyzeTerminology bool     `json:"analyze_terminology"`
	AnalyzeCulture     bool     `json:"analyze_culture"`
	AnalyzeChapters    bool     `json:"analyze_chapters"`
	DetailLevel        string   `json:"detail_level"`
}

// LLMsVerifierConfig holds the LLMsVerifier integration settings
type LLMsVerifierConfig struct {
	Enabled             bool          `json:"enabled"`
	APIURL              string        `json:"api_url"`
	APIKey              string        `json:"api_key"`
	DBPath              string        `json:"db_path"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	StrictMode          bool          `json:"strict_mode"`
	VerificationEnabled bool          `json:"verification_enabled"`
	MinScoreThreshold   float64       `json:"min_score_threshold"`
	MaxProviders        int           `json:"max_providers"`
	ScoringWeights      ScoreWeights  `json:"scoring_weights"`
}

// ScoreWeights defines the component weights for LLMsVerifier scoring
type ScoreWeights struct {
	ResponseSpeed     float64 `json:"response_speed"`
	CostEffectiveness float64 `json:"cost_effectiveness"`
	ModelEfficiency   float64 `json:"model_efficiency"`
	Capability        float64 `json:"capability"`
	Recency           float64 `json:"recency"`
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
			DefaultProvider: "openai",
			DefaultModel:    "",
			CacheEnabled:    true,
			CacheTTL:        3600,
			MaxConcurrent:   5,
			Providers:       make(map[string]ProviderConfig),
		},
		Preparation: PreparationConfig{
			Enabled:            true,
			PassCount:          2,
			Providers:          []string{"deepseek", "anthropic"},
			AnalyzeContentType: true,
			AnalyzeCharacters:  true,
			AnalyzeTerminology: true,
			AnalyzeCulture:     true,
			AnalyzeChapters:    true,
			DetailLevel:        "standard",
		},
		Distributed: DistributedConfig{
			Enabled:             false,
			Workers:             make(map[string]WorkerConfig),
			SSHTimeout:          30,
			SSHMaxRetries:       3,
			HealthCheckInterval: 30,
			MaxRemoteInstances:  20,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "",
		},
		LLMsVerifier: LLMsVerifierConfig{
			Enabled:             false,
			APIURL:              "http://localhost:8080",
			DBPath:              "./data/verifier.db",
			CacheTTL:            time.Hour,
			StrictMode:          false,
			VerificationEnabled: true,
			MinScoreThreshold:   0.0,
			MaxProviders:        25,
			ScoringWeights: ScoreWeights{
				ResponseSpeed:     0.20,
				CostEffectiveness: 0.30,
				ModelEfficiency:   0.25,
				Capability:        0.20,
				Recency:           0.05,
			},
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
		"openai":       "OPENAI_API_KEY",
		"anthropic":    "ANTHROPIC_API_KEY",
		"zhipu":        "ZHIPU_API_KEY",
		"deepseek":     "DEEPSEEK_API_KEY",
		"qwen":         "QWEN_API_KEY",
		"gemini":       "GEMINI_API_KEY",
		"groq":         "GROQ_API_KEY",
		"cohere":       "COHERE_API_KEY",
		"mistral":      "MISTRAL_API_KEY",
		"xai":          "XAI_API_KEY",
		"replicate":    "REPLICATE_API_KEY",
		"cerebras":     "CEREBRAS_API_KEY",
		"cloudflare":   "CLOUDFLARE_API_KEY",
		"siliconflow":  "SILICONFLOW_API_KEY",
		"hyperbolic":   "HYPERBOLIC_API_KEY",
		"togetherai":   "TOGETHER_API_KEY",
		"sambanova":    "SAMBANOVA_API_KEY",
		"kimi":         "KIMI_API_KEY",
		"novita":       "NOVITA_API_KEY",
		"nlpcloud":     "NLP_CLOUD_API_KEY",
		"upstage":      "UPSTAGE_API_KEY",
		"sarvam":       "SARVAM_API_KEY",
		"modal":        "MODAL_API_KEY",
		"publicai":     "PUBLICAI_API_KEY",
		"nia":          "NIA_API_KEY",
		"vulavula":     "VULAVULA_API_KEY",
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

	// Load LLMsVerifier configuration from environment
	c.loadLLMsVerifierFromEnv()
}

// loadLLMsVerifierFromEnv loads LLMsVerifier-specific configuration from environment
func (c *Config) loadLLMsVerifierFromEnv() {
	if enabled := os.Getenv("LLMSVERIFIER_ENABLED"); enabled != "" {
		c.LLMsVerifier.Enabled = enabled == "true"
	}
	if apiURL := os.Getenv("LLMSVERIFIER_API_URL"); apiURL != "" {
		c.LLMsVerifier.APIURL = apiURL
	}
	if apiKey := os.Getenv("LLMSVERIFIER_API_KEY"); apiKey != "" {
		c.LLMsVerifier.APIKey = apiKey
	}
	if dbPath := os.Getenv("LLMSVERIFIER_DB_PATH"); dbPath != "" {
		c.LLMsVerifier.DBPath = dbPath
	}
	if cacheTTL := os.Getenv("LLMSVERIFIER_CACHE_TTL"); cacheTTL != "" {
		if d, err := time.ParseDuration(cacheTTL); err == nil {
			c.LLMsVerifier.CacheTTL = d
		}
	}
	if strictMode := os.Getenv("LLMSVERIFIER_STRICT_MODE"); strictMode != "" {
		c.LLMsVerifier.StrictMode = strictMode == "true"
	}
	if verificationEnabled := os.Getenv("LLMSVERIFIER_VERIFICATION_ENABLED"); verificationEnabled != "" {
		c.LLMsVerifier.VerificationEnabled = verificationEnabled == "true"
	}
	if minScore := os.Getenv("LLMSVERIFIER_MIN_SCORE"); minScore != "" {
		var score float64
		if _, err := fmt.Sscanf(minScore, "%f", &score); err == nil {
			c.LLMsVerifier.MinScoreThreshold = score
		}
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

	// Validate LLMsVerifier configuration
	if err := c.validateLLMsVerifierConfig(); err != nil {
		return err
	}

	// Validate distributed configuration
	if err := c.validateDistributedConfig(); err != nil {
		return err
	}

	return nil
}

// validateLLMsVerifierConfig validates LLMsVerifier-specific configuration
func (c *Config) validateLLMsVerifierConfig() error {
	if !c.LLMsVerifier.Enabled {
		return nil
	}

	if c.LLMsVerifier.APIURL == "" {
		return fmt.Errorf("LLMsVerifier API URL is required when verifier is enabled")
	}

	if c.LLMsVerifier.APIKey == "" {
		return fmt.Errorf("LLMsVerifier API key is required when verifier is enabled")
	}

	if c.LLMsVerifier.MinScoreThreshold < 0 {
		return fmt.Errorf("LLMsVerifier minimum score threshold cannot be negative")
	}

	// Validate scoring weights sum to approximately 1.0
	weights := c.LLMsVerifier.ScoringWeights
	total := weights.ResponseSpeed + weights.CostEffectiveness + weights.ModelEfficiency + weights.Capability + weights.Recency
	if total > 0 && (total < 0.99 || total > 1.01) {
		return fmt.Errorf("LLMsVerifier scoring weights must sum to 1.0, got %.2f", total)
	}

	return nil
}

// validateDistributedConfig validates distributed work configuration
func (c *Config) validateDistributedConfig() error {
	if !c.Distributed.Enabled {
		return nil // Skip validation if distributed work is disabled
	}

	// Validate SSH timeout
	if c.Distributed.SSHTimeout <= 0 {
		return fmt.Errorf("SSH timeout must be positive")
	}

	// Validate SSH max retries
	if c.Distributed.SSHMaxRetries < 0 {
		return fmt.Errorf("SSH max retries cannot be negative")
	}

	// Validate workers configuration
	if len(c.Distributed.Workers) == 0 {
		return fmt.Errorf("at least one worker must be configured when distributed work is enabled")
	}

	// Validate each worker
	for workerID, worker := range c.Distributed.Workers {
		if err := c.validateWorkerConfig(workerID, worker); err != nil {
			return err
		}
	}

	return nil
}

// validateWorkerConfig validates a single worker configuration
func (c *Config) validateWorkerConfig(workerID string, worker WorkerConfig) error {
	if worker.Name == "" {
		return fmt.Errorf("worker %s: name cannot be empty", workerID)
	}

	if worker.Host == "" {
		return fmt.Errorf("worker %s: host cannot be empty", workerID)
	}

	if worker.Port <= 0 || worker.Port > 65535 {
		return fmt.Errorf("worker %s: invalid port %d", workerID, worker.Port)
	}

	if worker.User == "" {
		return fmt.Errorf("worker %s: user cannot be empty", workerID)
	}

	// Validate authentication - at least one method must be provided
	if worker.KeyFile == "" && worker.Password == "" {
		return fmt.Errorf("worker %s: either key file or password must be provided", workerID)
	}

	if worker.MaxCapacity <= 0 {
		return fmt.Errorf("worker %s: max capacity must be positive", workerID)
	}

	// Validate tags (optional but if provided should be reasonable)
	for _, tag := range worker.Tags {
		if tag == "" {
			return fmt.Errorf("worker %s: empty tag not allowed", workerID)
		}
	}

	return nil
}
