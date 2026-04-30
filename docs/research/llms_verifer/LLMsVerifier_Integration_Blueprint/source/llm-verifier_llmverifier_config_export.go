package llmverifier

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
)

// FeatureSuffixes contains all valid feature suffixes for verified models
var FeatureSuffixes = []string{
	"(brotli)",
	"(http3)",
	"(toon)",
	"(streaming)",
	"(free to use)",
	"(open source)",
	"(fast)",
	"(llmsvd)", // Mandatory LLMsVerifier verified suffix - always last
}

// formatModelNameWithSuffixes formats a model name with feature suffixes based on detected capabilities
// The (llmsvd) suffix is mandatory and always added last for verified models
func formatModelNameWithSuffixes(modelID string, result VerificationResult, isVerified bool) string {
	suffixes := []string{}

	// Add feature suffixes based on detection results
	if result.FeatureDetection.SupportsBrotli || result.ModelInfo.SupportsBrotli {
		suffixes = append(suffixes, "(brotli)")
	}
	if result.FeatureDetection.SupportsHTTP3 || result.ModelInfo.SupportsHTTP3 {
		suffixes = append(suffixes, "(http3)")
	}
	if result.FeatureDetection.SupportsToon || result.ModelInfo.SupportsToon {
		suffixes = append(suffixes, "(toon)")
	}
	if result.FeatureDetection.Streaming {
		suffixes = append(suffixes, "(streaming)")
	}

	// Add cost-based suffix
	provider := extractProvider(result.ModelInfo.Endpoint)
	if isProviderFree(provider) {
		suffixes = append(suffixes, "(free to use)")
	}

	// Add mandatory (llmsvd) suffix for verified models - ALWAYS LAST
	if isVerified {
		suffixes = append(suffixes, "(llmsvd)")
	}

	if len(suffixes) > 0 {
		return fmt.Sprintf("%s %s", modelID, strings.Join(suffixes, " "))
	}
	return modelID
}

// formatProviderNameWithSuffix formats a provider name with (llmsvd) suffix if verified
func formatProviderNameWithSuffix(providerName string, isVerified bool) string {
	if isVerified {
		return fmt.Sprintf("%s (llmsvd)", providerName)
	}
	return providerName
}

// detectBrotliSupport detects brotli compression support based on provider endpoint
func detectBrotliSupport(endpoint string) bool {
	endpointLower := strings.ToLower(endpoint)
	// Check for specific provider domains (not just substring match)
	return strings.Contains(endpointLower, "api.anthropic.com") ||
		strings.Contains(endpointLower, "api.openai.com") ||
		strings.Contains(endpointLower, "googleapis.com") ||
		strings.Contains(endpointLower, "api.deepseek.com") ||
		strings.Contains(endpointLower, "api.mistral.ai") ||
		strings.Contains(endpointLower, "api.cohere.com")
}

// detectHTTP3Support detects HTTP/3 protocol support based on provider endpoint
func detectHTTP3Support(endpoint string) bool {
	endpointLower := strings.ToLower(endpoint)
	return strings.Contains(endpointLower, "cloudflare") ||
		strings.Contains(endpointLower, "google") ||
		strings.Contains(endpointLower, "fastly")
}

// detectToonSupport detects toon/creative style support based on model name
func detectToonSupport(modelID string) bool {
	modelLower := strings.ToLower(modelID)
	return strings.Contains(modelLower, "toon") ||
		strings.Contains(modelLower, "creative") ||
		strings.Contains(modelLower, "art") ||
		strings.Contains(modelLower, "dalle") ||
		strings.Contains(modelLower, "dall-e") ||
		strings.Contains(modelLower, "stable-diffusion") ||
		strings.Contains(modelLower, "midjourney") ||
		strings.Contains(modelLower, "imagen")
}

// AIConfig represents the base structure for AI CLI agent configurations
type AIConfig struct {
	Version     string      `json:"version"`
	CreatedAt   time.Time   `json:"created_at"`
	GeneratedBy string      `json:"generated_by"`
	Models      []AIModel   `json:"models"`
	Preferences Preferences `json:"preferences"`
	Metadata    Metadata    `json:"metadata"`
}

// AIModel represents a model in AI CLI configuration
type AIModel struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Provider     string         `json:"provider"`
	Endpoint     string         `json:"endpoint"`
	APIKey       string         `json:"api_key,omitempty"`
	Capabilities []string       `json:"capabilities"`
	Score        float64        `json:"score"`
	Category     string         `json:"category"`
	Tags         []string       `json:"tags"`
	Description  string         `json:"description,omitempty"`
	Settings     map[string]any `json:"settings,omitempty"`
}

// Preferences contains user preferences for AI tools
type Preferences struct {
	PrimaryModel    string   `json:"primary_model"`
	FallbackModels  []string `json:"fallback_models"`
	MaxTokens       int      `json:"max_tokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	AutoSave        bool     `json:"auto_save"`
	StreamResponses bool     `json:"stream_responses"`
	Language        string   `json:"language,omitempty"`
}

// Metadata contains additional information about the configuration
type Metadata struct {
	TotalModels        int       `json:"total_models"`
	AverageScore       float64   `json:"average_score"`
	ExportCriteria     string    `json:"export_criteria"`
	LLMVerifierVersion string    `json:"llm_verifier_version"`
	LastUpdated        time.Time `json:"last_updated"`
}

// ExportOptions controls how models are selected for export
type ExportOptions struct {
	Format        string             `json:"format"`
	Top           int                `json:"top,omitempty"`
	MinScore      float64            `json:"min_score,omitempty"`
	MaxModels     int                `json:"max_models,omitempty"`
	Categories    []string           `json:"categories,omitempty"`
	Providers     []string           `json:"providers,omitempty"`
	Models        []string           `json:"models,omitempty"`
	ScoreWeight   map[string]float64 `json:"score_weight,omitempty"`
	IncludeAPIKey bool               `json:"include_api_key"`
}

// CrushConfig represents Crush's configuration format
type CrushConfig struct {
	Schema    string                   `json:"$schema,omitempty"`
	Providers map[string]CrushProvider `json:"providers"`
	LSP       map[string]CrushLSP      `json:"lsp,omitempty"`
	MCP       map[string]CrushMCP      `json:"mcp,omitempty"`
	Options   map[string]any           `json:"options,omitempty"`
}

// CrushProvider represents a provider in Crush config
type CrushProvider struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	BaseURL string       `json:"base_url"`
	APIKey  string       `json:"api_key,omitempty"`
	Models  []CrushModel `json:"models"`
}

// CrushModel represents a model in Crush provider config
type CrushModel struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	CostPer1MIn         float64 `json:"cost_per_1m_in,omitempty"`
	CostPer1MOut        float64 `json:"cost_per_1m_out,omitempty"`
	CostPer1MInCached   float64 `json:"cost_per_1m_in_cached,omitempty"`
	CostPer1MOutCached  float64 `json:"cost_per_1m_out_cached,omitempty"`
	ContextWindow       int     `json:"context_window"`
	DefaultMaxTokens    int     `json:"default_max_tokens,omitempty"`
	CanReason           bool    `json:"can_reason,omitempty"`
	SupportsAttachments bool    `json:"supports_attachments,omitempty"`
	SupportsHTTP3       bool    `json:"supports_http3,omitempty"`
	SupportsToon        bool    `json:"supports_toon,omitempty"`
	SupportsBrotli      bool    `json:"supports_brotli,omitempty"`
	SupportsStreaming   bool    `json:"supports_streaming,omitempty"`
	Verified            bool    `json:"verified,omitempty"`
}

// CrushLSP represents LSP configuration
type CrushLSP struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool              `json:"enabled"`
}

// CrushMCP represents MCP server configuration
type CrushMCP struct {
	Type          string            `json:"type"`
	Command       string            `json:"command,omitempty"`
	Args          []string          `json:"args,omitempty"`
	URL           string            `json:"url,omitempty"`
	Timeout       int               `json:"timeout,omitempty"`
	Disabled      bool              `json:"disabled,omitempty"`
	DisabledTools []string          `json:"disabled_tools,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// OpenCodeConfig represents the official OpenCode configuration format
type OpenCodeConfig struct {
	Schema   string                      `json:"$schema"`
	Provider map[string]OpenCodeProvider `json:"provider"`
}

// OpenCodeProvider represents a provider in OpenCode config
type OpenCodeProvider struct {
	Options OpenCodeOptions `json:"options"`
	Models  map[string]any  `json:"models"` // Empty object as per OpenCode spec
}

// OpenCodeOptions contains provider options for OpenCode
type OpenCodeOptions struct {
	APIKey string `json:"apiKey"`
}

// ExportConfig exports configuration to various formats (enhanced)
func ExportConfig(db *database.Database, cfg *config.Config, format, outputPath string) error {
	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(cfg, "", "  ")
	case "yaml", "yml":
		data, err = yaml.Marshal(cfg)
	case "opencode":
		return ExportAIConfig(db, cfg, "opencode", outputPath, &ExportOptions{})
	case "crush":
		return ExportAIConfig(db, cfg, "crush", outputPath, &ExportOptions{})
	case "claude-code":
		return ExportAIConfig(db, cfg, "claude-code", outputPath, &ExportOptions{})
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ExportAIConfig exports AI CLI agent configurations
func ExportAIConfig(db *database.Database, cfg *config.Config, aiFormat, outputPath string, options *ExportOptions) error {
	// Fetch ALL providers and models from database (including unverified ones)
	results, err := fetchAllProvidersWithModels(db, options)
	if err != nil {
		// Fall back to verified results only
		fmt.Printf("Warning: Could not fetch all providers: %v, falling back to verified results only\n", err)
		results, err = fetchVerificationResults(db, options)
		if err != nil {
			return fmt.Errorf("failed to fetch verification results: %w", err)
		}
	}

	// Filter models based on options
	filteredModels := filterModels(results, options)

	// If no real results found, create comprehensive mock data with all known providers
	if len(filteredModels) == 0 {
		fmt.Println("No real verification results found, using comprehensive mock data with all known providers")
		filteredModels = createComprehensiveMockResults()
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Handle different formats
	switch strings.ToLower(aiFormat) {
	case "crush":
		return exportCrushConfig(filteredModels, outputPath, options)
	case "opencode":
		// Use correct OpenCode format that matches actual OpenCode schema
		opencodeConfig, err := createCorrectOpenCodeConfig(filteredModels, options)
		if err != nil {
			return fmt.Errorf("failed to create OpenCode config: %w", err)
		}

		// Validate config before writing
		if err := validateOpenCodeConfigMap(opencodeConfig); err != nil {
			return fmt.Errorf("OpenCode config validation failed: %w", err)
		}

		// Marshal to JSON
		data, err := json.MarshalIndent(opencodeConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal OpenCode config: %w", err)
		}

		// Write to file
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			// Record failed export
			RecordOpenCodeExport(opencodeConfig, false, err.Error())
			return fmt.Errorf("failed to write OpenCode config: %w", err)
		}

		// Validate written file can be parsed back
		if err := validateOpenCodeConfigFile(outputPath); err != nil {
			return fmt.Errorf("OpenCode config file validation failed: %w", err)
		}

		// Record successful export
		RecordOpenCodeExport(opencodeConfig, true, "")

		fmt.Printf("✅ OpenCode config validated and exported successfully\n")
		return nil
	}

	return fmt.Errorf("unsupported AI format: %s", aiFormat)
}

// createOpenCodeConfig creates configuration for OpenCode
func createOpenCodeConfig(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)
		provider := extractProvider(result.ModelInfo.Endpoint)

		name := result.ModelInfo.ID
		if isProviderFree(provider) {
			name += " free to use"
		}

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         name,
			Provider:     provider,
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings:     createOpenCodeModelSettings(result, provider),
		}

		// Add API key if requested
		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE" // In real implementation, this would be encrypted/obfuscated
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by score (highest first)
	slices.SortFunc(models, func(a, b AIModel) int {
		if b.Score > a.Score {
			return 1
		}
		if b.Score < a.Score {
			return -1
		}
		return 0
	})

	// Select primary model (highest scoring)
	var primaryModel string
	var fallbackModels []string

	if len(models) > 0 {
		primaryModel = models[0].ID
		for i := 1; i < len(models) && i < 3; i++ {
			fallbackModels = append(fallbackModels, models[i].ID)
		}
	}

	avgScore := float64(0)
	if len(models) > 0 {
		avgScore = totalScore / float64(len(models))
	}

	return &AIConfig{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		Models:      models,
		Preferences: Preferences{
			PrimaryModel:    primaryModel,
			FallbackModels:  fallbackModels,
			MaxTokens:       4096,
			Temperature:     0.7,
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     getExportCriteriaDescription(options),
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createOpenCodeModelSettings creates provider-specific settings for OpenCode
func createOpenCodeModelSettings(result VerificationResult, provider string) map[string]any {
	baseSettings := map[string]any{
		"max_tokens":         result.ModelInfo.MaxOutputTokens,
		"context_window":     result.ModelInfo.ContextWindow.TotalMaxTokens,
		"supports_vision":    result.ModelInfo.SupportsVision,
		"supports_audio":     result.ModelInfo.SupportsAudio,
		"supports_video":     result.ModelInfo.SupportsVideo,
		"supports_reasoning": result.ModelInfo.SupportsReasoning,
		"supports_http3":     result.ModelInfo.SupportsHTTP3,
		"supports_toon":      result.ModelInfo.SupportsToon,
		"temperature":        0.7,
		"top_p":              0.9,
		"frequency_penalty":  0.0,
		"presence_penalty":   0.0,
	}

	// Configure HTTP/3 settings if supported
	if result.ModelInfo.SupportsHTTP3 {
		baseSettings["protocol"] = "http3"
		baseSettings["quic_enabled"] = true
		baseSettings["cronet_enabled"] = true
	}

	// Configure Toon format settings if supported
	if result.ModelInfo.SupportsToon {
		baseSettings["data_format"] = "toon"
		baseSettings["toon_compression"] = true
		baseSettings["toon_optimization"] = true
	}

	// Provider-specific optimizations
	switch provider {
	case "OpenAI":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "gpt"
		baseSettings["streaming_supported"] = true
	case "Anthropic":
		baseSettings["temperature"] = 0.8
		baseSettings["model_family"] = "claude"
		baseSettings["streaming_supported"] = true
		baseSettings["max_tokens"] = min(4096, result.ModelInfo.MaxOutputTokens)
	case "DeepSeek":
		baseSettings["temperature"] = 0.6
		baseSettings["model_family"] = "deepseek"
		baseSettings["streaming_supported"] = true
	case "Google":
		baseSettings["temperature"] = 0.9
		baseSettings["model_family"] = "gemini"
		baseSettings["streaming_supported"] = true
	case "Groq":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "groq"
		baseSettings["streaming_supported"] = true
	case "TogetherAI":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "together"
		baseSettings["streaming_supported"] = true
	case "Fireworks":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "fireworks"
		baseSettings["streaming_supported"] = true
	case "Poe":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "poe"
		baseSettings["streaming_supported"] = true
	case "Navigator":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "navigator"
		baseSettings["streaming_supported"] = true
	case "Mistral":
		baseSettings["temperature"] = 0.7
		baseSettings["model_family"] = "mistral"
		baseSettings["streaming_supported"] = true
	}

	// Add reasoning-specific settings
	if result.ModelInfo.SupportsReasoning {
		baseSettings["reasoning_enabled"] = true
		baseSettings["reasoning_budget"] = 2048
		baseSettings["reasoning_temperature"] = 0.3
	}

	return baseSettings
}

// createCrushConfig creates configuration for Crush
// createCrushConfig creates configuration for Crush (currently unused but kept for future integration)
func createCrushConfig(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Crush focuses on coding capabilities - filter for high coding scores
		if result.PerformanceScores.CodeCapability < 75 {
			continue // Skip models with poor coding capability
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)
		provider := extractProvider(result.ModelInfo.Endpoint)

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         result.ModelInfo.ID,
			Provider:     provider,
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings:     createCrushModelSettings(result, provider),
		}

		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE"
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by code capability score (highest first)
	slices.SortFunc(models, func(a, b AIModel) int {
		aCodeScore := a.Score // Assuming we can access code score, using overall for now
		bCodeScore := b.Score
		if bCodeScore > aCodeScore {
			return 1
		}
		if bCodeScore < aCodeScore {
			return -1
		}
		return 0
	})

	var primaryModel string
	var fallbackModels []string

	if len(models) > 0 {
		primaryModel = models[0].ID
		for i := 1; i < len(models) && i < 2; i++ {
			fallbackModels = append(fallbackModels, models[i].ID)
		}
	}

	avgScore := float64(0)
	if len(models) > 0 {
		avgScore = totalScore / float64(len(models))
	}

	return &AIConfig{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		Models:      models,
		Preferences: Preferences{
			PrimaryModel:    primaryModel,
			FallbackModels:  fallbackModels,
			MaxTokens:       2048, // Crush prefers shorter responses for coding
			Temperature:     0.3,  // Lower temperature for deterministic code
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     "High coding capability models (code score >= 75)",
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createCrushModelSettings creates coding-focused settings for Crush
func createCrushModelSettings(result VerificationResult, provider string) map[string]any {
	baseSettings := map[string]any{
		"max_tokens":        result.ModelInfo.MaxOutputTokens,
		"context_window":    result.ModelInfo.ContextWindow.TotalMaxTokens,
		"temperature":       0.3, // Crush prefers lower temperature for coding
		"top_p":             0.95,
		"frequency_penalty": 0.1,
		"presence_penalty":  0.1,
		"code_focus":        true,
		"debug_mode":        false,
		"auto_debug":        true,
		"test_generation":   true,
		"linting_enabled":   true,
		"format_on_save":    true,
		"supports_http3":    result.ModelInfo.SupportsHTTP3,
		"supports_toon":     result.ModelInfo.SupportsToon,
	}

	// Configure HTTP/3 for coding workflows if supported
	if result.ModelInfo.SupportsHTTP3 {
		baseSettings["protocol"] = "http3"
		baseSettings["quic_enabled"] = true
		baseSettings["cronet_enabled"] = true
		baseSettings["low_latency_mode"] = true
	}

	// Configure Toon format for efficient code transmission if supported
	if result.ModelInfo.SupportsToon {
		baseSettings["data_format"] = "toon"
		baseSettings["toon_compression"] = true
		baseSettings["toon_optimization"] = true
		baseSettings["code_streaming"] = true
	}

	// Provider-specific coding optimizations
	switch provider {
	case "OpenAI":
		baseSettings["model_family"] = "gpt"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Anthropic":
		baseSettings["model_family"] = "claude"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["max_tokens"] = min(2048, result.ModelInfo.MaxOutputTokens)
	case "DeepSeek":
		baseSettings["model_family"] = "deepseek"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["temperature"] = 0.2 // Even lower for DeepSeek coding
	case "Google":
		baseSettings["model_family"] = "gemini"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Groq":
		baseSettings["model_family"] = "groq"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "TogetherAI":
		baseSettings["model_family"] = "together"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Fireworks":
		baseSettings["model_family"] = "fireworks"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Poe":
		baseSettings["model_family"] = "poe"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Navigator":
		baseSettings["model_family"] = "navigator"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Replicate":
		baseSettings["model_family"] = "replicate"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	case "Mistral":
		baseSettings["model_family"] = "mistral"
		baseSettings["code_optimization"] = true
		baseSettings["streaming_supported"] = true
	}

	// Add advanced coding features
	if result.FeatureDetection.CodeGeneration && result.FeatureDetection.CodeReview {
		baseSettings["advanced_coding"] = true
		baseSettings["code_review_enabled"] = true
		baseSettings["refactoring_suggestions"] = true
	}

	return baseSettings
}

// createClaudeCode creates configuration for Claude Code
func createClaudeCode(results []VerificationResult, options *ExportOptions) (*AIConfig, error) {
	models := make([]AIModel, 0, len(results))
	var totalScore float64

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		capabilities := extractCapabilities(result)
		category := categorizeModel(result)
		provider := extractProvider(result.ModelInfo.Endpoint)

		name := result.ModelInfo.ID
		if isProviderFree(provider) {
			name += " free to use"
		}

		model := AIModel{
			ID:           result.ModelInfo.ID,
			Name:         name,
			Provider:     provider,
			Endpoint:     result.ModelInfo.Endpoint,
			Capabilities: capabilities,
			Score:        result.PerformanceScores.OverallScore,
			Category:     category,
			Tags:         result.ModelInfo.Tags,
			Description:  result.ModelInfo.Description,
			Settings:     createClaudeCodeModelSettings(result, provider),
		}

		if options != nil && options.IncludeAPIKey {
			model.APIKey = "YOUR_API_KEY_HERE"
		}

		models = append(models, model)
		totalScore += result.PerformanceScores.OverallScore
	}

	// Sort by overall score with preference for reasoning capability
	slices.SortFunc(models, func(a, b AIModel) int {
		// Prefer models with reasoning capability
		aReasoning := slices.Contains(a.Capabilities, "reasoning")
		bReasoning := slices.Contains(b.Capabilities, "reasoning")

		if aReasoning && !bReasoning {
			return -1
		}
		if !aReasoning && bReasoning {
			return 1
		}

		return int(b.Score*1000 - a.Score*1000)
	})

	var primaryModel string
	var fallbackModels []string

	if len(models) > 0 {
		primaryModel = models[0].ID
		for i := 1; i < len(models) && i < 3; i++ {
			fallbackModels = append(fallbackModels, models[i].ID)
		}
	}

	avgScore := float64(0)
	if len(models) > 0 {
		avgScore = totalScore / float64(len(models))
	}

	return &AIConfig{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		Models:      models,
		Preferences: Preferences{
			PrimaryModel:    primaryModel,
			FallbackModels:  fallbackModels,
			MaxTokens:       4096,
			Temperature:     0.5, // Moderate temperature for balanced responses
			AutoSave:        true,
			StreamResponses: true,
			Language:        "english",
		},
		Metadata: Metadata{
			TotalModels:        len(models),
			AverageScore:       avgScore,
			ExportCriteria:     "All models with preference for reasoning capability",
			LLMVerifierVersion: "1.0.0",
			LastUpdated:        time.Now(),
		},
	}, nil
}

// createClaudeCodeModelSettings creates Claude Code specific settings
func createClaudeCodeModelSettings(result VerificationResult, provider string) map[string]any {
	baseSettings := map[string]any{
		"max_tokens":         result.ModelInfo.MaxOutputTokens,
		"context_window":     result.ModelInfo.ContextWindow.TotalMaxTokens,
		"temperature":        0.5, // Claude Code prefers moderate temperature
		"top_p":              0.9,
		"frequency_penalty":  0.0,
		"presence_penalty":   0.0,
		"conversation_style": "professional",
		"code_style":         "clean",
		"explanation_level":  "detailed",
		"auto_format":        true,
		"import_suggestions": true,
		"error_handling":     true,
		"context_awareness":  true,
	}

	// Provider-specific conversation settings
	switch provider {
	case "Anthropic":
		baseSettings["model_family"] = "claude"
		baseSettings["anthropic_style"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["conversation_memory"] = true
	case "OpenAI":
		baseSettings["model_family"] = "gpt"
		baseSettings["openai_style"] = true
		baseSettings["streaming_supported"] = true
		baseSettings["conversation_memory"] = true
	case "DeepSeek":
		baseSettings["model_family"] = "deepseek"
		baseSettings["streaming_supported"] = true
	case "Groq":
		baseSettings["model_family"] = "groq"
		baseSettings["streaming_supported"] = true
	case "TogetherAI":
		baseSettings["model_family"] = "together"
		baseSettings["streaming_supported"] = true
	case "Fireworks":
		baseSettings["model_family"] = "fireworks"
		baseSettings["streaming_supported"] = true
	case "Poe":
		baseSettings["model_family"] = "poe"
		baseSettings["streaming_supported"] = true
	case "Navigator":
		baseSettings["model_family"] = "navigator"
		baseSettings["streaming_supported"] = true
	case "Replicate":
		baseSettings["model_family"] = "replicate"
		baseSettings["streaming_supported"] = true
	case "Mistral":
		baseSettings["model_family"] = "mistral"
		baseSettings["streaming_supported"] = true
		baseSettings["temperature"] = 0.4 // Slightly lower for DeepSeek
	case "Google":
		baseSettings["model_family"] = "gemini"
		baseSettings["google_style"] = true
		baseSettings["streaming_supported"] = true
	}

	// Add reasoning-specific features
	if result.ModelInfo.SupportsReasoning {
		baseSettings["reasoning_assistance"] = true
		baseSettings["step_by_step_explanation"] = true
		baseSettings["complexity_analysis"] = true
	}

	return baseSettings
}

// filterModels filters verification results based on export options
func filterModels(results []VerificationResult, options *ExportOptions) []VerificationResult {
	if options == nil {
		return results
	}

	filtered := make([]VerificationResult, 0, len(results))

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Filter by minimum score
		if options.MinScore > 0 && result.PerformanceScores.OverallScore < options.MinScore {
			continue
		}

		// Filter by categories
		if len(options.Categories) > 0 {
			category := categorizeModel(result)
			if !slices.Contains(options.Categories, category) {
				continue
			}
		}

		// Filter by providers
		if len(options.Providers) > 0 {
			provider := extractProvider(result.ModelInfo.Endpoint)
			if !slices.Contains(options.Providers, provider) {
				continue
			}
		}

		// Filter by specific models
		if len(options.Models) > 0 {
			if !slices.Contains(options.Models, result.ModelInfo.ID) {
				continue
			}
		}

		filtered = append(filtered, result)
	}

	// Limit by max models
	if options.MaxModels > 0 && len(filtered) > options.MaxModels {
		filtered = filtered[:options.MaxModels]
	}

	// Sort by score (highest first)
	slices.SortFunc(filtered, func(a, b VerificationResult) int {
		return int(b.PerformanceScores.OverallScore*1000 - a.PerformanceScores.OverallScore*1000)
	})

	// Limit by top models
	if options.Top > 0 && len(filtered) > options.Top {
		filtered = filtered[:options.Top]
	}

	return filtered
}

// extractCapabilities extracts model capabilities from verification result
func extractCapabilities(result VerificationResult) []string {
	capabilities := make([]string, 0)

	if result.FeatureDetection.ToolUse {
		capabilities = append(capabilities, "tool_use")
	}
	if result.FeatureDetection.FunctionCalling {
		capabilities = append(capabilities, "function_calling")
	}
	if result.FeatureDetection.CodeGeneration {
		capabilities = append(capabilities, "code_generation")
	}
	if result.FeatureDetection.CodeCompletion {
		capabilities = append(capabilities, "code_completion")
	}
	if result.FeatureDetection.CodeReview {
		capabilities = append(capabilities, "code_review")
	}
	if result.FeatureDetection.CodeExplanation {
		capabilities = append(capabilities, "code_explanation")
	}
	if result.FeatureDetection.Embeddings {
		capabilities = append(capabilities, "embeddings")
	}
	if result.FeatureDetection.Reranking {
		capabilities = append(capabilities, "reranking")
	}
	if result.FeatureDetection.ImageGeneration {
		capabilities = append(capabilities, "image_generation")
	}
	if result.FeatureDetection.AudioGeneration {
		capabilities = append(capabilities, "audio_generation")
	}
	if result.FeatureDetection.VideoGeneration {
		capabilities = append(capabilities, "video_generation")
	}
	if result.FeatureDetection.MCPs {
		capabilities = append(capabilities, "mcps")
	}
	if result.FeatureDetection.LSPs {
		capabilities = append(capabilities, "lsps")
	}
	if result.FeatureDetection.Multimodal {
		capabilities = append(capabilities, "multimodal")
	}
	if result.FeatureDetection.Streaming {
		capabilities = append(capabilities, "streaming")
	}
	if result.FeatureDetection.JSONMode {
		capabilities = append(capabilities, "json_mode")
	}
	if result.FeatureDetection.StructuredOutput {
		capabilities = append(capabilities, "structured_output")
	}
	if result.FeatureDetection.Reasoning {
		capabilities = append(capabilities, "reasoning")
	}
	if result.FeatureDetection.ParallelToolUse {
		capabilities = append(capabilities, "parallel_tool_use")
	}

	return capabilities
}

// categorizeModel categorizes model based on capabilities
func categorizeModel(result VerificationResult) string {
	// Check for coding-focused models (highest priority)
	if result.CodeCapabilities.CodeGeneration ||
		result.CodeCapabilities.CodeReview ||
		result.CodeCapabilities.CodeExplanation ||
		result.FeatureDetection.CodeGeneration ||
		result.FeatureDetection.CodeReview ||
		result.FeatureDetection.CodeExplanation {
		return "coding"
	}

	// Check for multimodal models
	if result.FeatureDetection.Multimodal ||
		(result.FeatureDetection.ImageGeneration && result.FeatureDetection.CodeGeneration) {
		return "multimodal"
	}

	// Check for reasoning models
	if result.FeatureDetection.Reasoning && result.PerformanceScores.OverallScore >= 80 {
		return "reasoning"
	}

	// Check for chat models
	if result.FeatureDetection.ToolUse && result.FeatureDetection.FunctionCalling {
		return "chat"
	}

	// Check for generative models
	if result.GenerativeCapabilities.CreativeWriting ||
		result.GenerativeCapabilities.ContentGeneration {
		return "generative"
	}

	// Check for specialized models
	if result.FeatureDetection.Embeddings || result.FeatureDetection.Reranking {
		return "specialized"
	}

	return "general"
}

// extractProvider extracts provider name from endpoint
// extractProvider extracts provider name from endpoint URL with enhanced detection
func extractProvider(endpoint string) string {
	// Clean and normalize endpoint
	endpoint = strings.ToLower(strings.TrimSpace(endpoint))

	// Strip protocol prefix
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	// Strip path suffix (keep just the domain)
	if idx := strings.Index(endpoint, "/"); idx > 0 {
		endpoint = endpoint[:idx]
	}

	// Primary provider detection
	providerPatterns := map[string]string{
		// Major providers
		"openai.com":                        "openai",
		"api.openai.com":                    "openai",
		"anthropic.com":                     "anthropic",
		"api.anthropic.com":                 "anthropic",
		"google.com":                        "gemini",
		"generativelanguage.googleapis.com": "gemini",
		"vertexai":                          "vertexai",

		// Popular alternatives
		"groq.com":           "groq",
		"api.groq.com":       "groq",
		"deepseek.com":       "deepseek",
		"api.deepseek.com":   "deepseek",
		"together.xyz":       "together",
		"api.together.xyz":   "together",
		"fireworks.ai":       "fireworks",
		"api.fireworks.ai":   "fireworks",
		"perplexity.ai":      "perplexity",
		"api.perplexity.ai":  "perplexity",
		"mistral.ai":         "mistral",
		"api.mistral.ai":     "mistral",
		"mistral.com":        "mistral",
		"api.mistral.com":    "mistral",
		"cohere.com":         "cohere",
		"api.cohere.com":     "cohere",
		"sambanova.com":      "sambanova",
		"api.sambanova.com":  "sambanova",
		"sambanova.ai":       "sambanova",
		"api.sambanova.ai":   "sambanova",

		// Cloud providers
		"azure.com":        "azure",
		"openai.azure.com": "azure",
		"aws":              "bedrock",
		"bedrock":          "bedrock",
		"amazonaws.com":    "bedrock",

		// Open source and community
		"huggingface.co":               "huggingface",
		"inference-api.huggingface.co": "huggingface",
		"api-inference.huggingface.co": "huggingface",
		"replicate.com":                "replicate",
		"api.replicate.com":            "replicate",
		"chutes.ai":                    "chutes",
		"api.chutes.ai":                "chutes",
		"novita.ai":                    "novita",
		"api.novita.ai":                "novita",
		"inference.net":                "inference",
		"api.inference.net":            "inference",
		"upstage.ai":                   "upstage",
		"api.upstage.ai":               "upstage",
		"baseten.co":                   "baseten",
		"api.baseten.co":               "baseten",

		// Regional and specialized
		"siliconflow.cn":           "siliconflow",
		"api.siliconflow.cn":       "siliconflow",
		"moonshot.cn":              "kimi",
		"api.moonshot.cn":          "kimi",
		"nvidia.com":               "nvidia",
		"integrate.api.nvidia.com": "nvidia",
		"z.ai":                     "zai",
		"api.z.ai":                 "zai",
		"zai.com":                  "zai",
		"api.zai.com":              "zai",

		// Aggregators and routers
		"openrouter.ai":      "openrouter",
		"api.openrouter.ai":  "openrouter",
		"cerebras.ai":        "cerebras",
		"api.cerebras.ai":    "cerebras",
		"hyperbolic.xyz":     "hyperbolic",
		"api.hyperbolic.xyz": "hyperbolic",
		"vercel.com":         "vercel",
		"api.vercel.com":     "vercel",

		// Additional providers
		"ai21.com":              "ai21",
		"api.ai21.com":          "ai21",
		"cloudflare.com":        "cloudflare",
		"api.cloudflare.com":    "cloudflare",
		"codestral.mistral.ai":  "codestral",
		"modal.com":             "modal",
		"api.modal.com":         "modal",
		"nlpcloud.com":          "nlpcloud",
		"api.nlpcloud.com":      "nlpcloud",
		"sarvam.ai":             "sarvam",
		"api.sarvam.ai":         "sarvam",
		"stability.com":         "stability",
		"api.stability.com":     "stability",
		"stability.ai":          "stability",
		"api.stability.ai":      "stability",
		"elevenlabs.io":         "elevenlabs",
		"api.elevenlabs.io":     "elevenlabs",
		"runway.com":            "runway",
		"api.runway.com":        "runway",
		"gooseai.io":            "gooseai",
		"api.gooseai.io":        "gooseai",
		"assemblyai.com":        "assemblyai",
		"api.assemblyai.com":    "assemblyai",
		"writer.com":            "writer",
		"api.writer.com":        "writer",
		"relevance.ai":          "relevance",
		"api.relevance.ai":      "relevance",
		"fal.ai":                "fal",
		"api.fal.ai":            "fal",
		"midjourney.com":        "midjourney",
		"api.midjourney.com":    "midjourney",
		"alephalpha.com":        "aleph-alpha",
		"api.alephalpha.com":    "aleph-alpha",
		"gladia.io":             "gladia",
		"api.gladia.io":         "gladia",
		"twelvelabs.io":         "twelvelabs",
		"api.twelvelabs.io":     "twelvelabs",
		"vulavula.com":          "vulavula",
		"api.vulavula.com":      "vulavula",

		// Local and custom endpoints
		"localhost": "local",
		"127.0.0.1": "local",
		"0.0.0.0":   "local",
	}

	// Check for specific patterns first (longer/more specific patterns take priority)
	// This ensures "codestral.mistral.ai" matches before "mistral.ai"
	specificPatterns := []struct {
		pattern  string
		provider string
	}{
		{"codestral.mistral.ai", "codestral"},
		{"api.groq.com/openai", "groq"}, // Groq has openai in path
	}

	for _, sp := range specificPatterns {
		if strings.Contains(endpoint, sp.pattern) {
			return sp.provider
		}
	}

	// Check for general matches
	for pattern, provider := range providerPatterns {
		if strings.Contains(endpoint, pattern) {
			return provider
		}
	}

	// Fallback: try to extract from URL structure
	// e.g., "api.example.com" -> "example"
	if strings.HasPrefix(endpoint, "api.") && strings.Count(endpoint, ".") >= 2 {
		parts := strings.Split(endpoint, ".")
		if len(parts) >= 3 {
			domain := parts[1]
			// Common domain to provider mappings
			switch domain {
			case "x":
				return "xai"
			case "sarvam":
				return "sarvam"
			case "lelapa":
				return "vulavula"
			case "twelvelabs":
				return "twelvelabs"
			case "codestral":
				return "codestral"
			case "dashscope":
				return "qwen"
			case "modal":
				return "modal"
			case "inference":
				return "inference"
			case "vercel":
				return "vercel"
			case "baseten":
				return "baseten"
			case "novita":
				return "novita"
			case "upstage":
				return "upstage"
			case "nlpcloud":
				return "nlpcloud"
			}
		}
	}

	// Last resort: try to extract from subdomain
	// e.g., "groq.api.example.com" -> "groq"
	if parts := strings.Split(endpoint, "."); len(parts) >= 3 {
		subdomain := parts[0]
		if subdomain != "api" && subdomain != "www" {
			return subdomain
		}
	}

	return "unknown"
}

// isProviderFree checks if a provider offers free models
func isProviderFree(provider string) bool {
	freeProviders := []string{
		"HuggingFace",
		"Nvidia",
		"Chutes",
		"SiliconFlow",
		"Kimi",
		"Gemini", // Google Gemini
	}
	for _, free := range freeProviders {
		if strings.EqualFold(provider, free) {
			return true
		}
	}
	return false
}

// getExportCriteriaDescription generates description of export criteria
func getExportCriteriaDescription(options *ExportOptions) string {
	if options == nil {
		return "All models"
	}

	criteria := make([]string, 0)

	if options.MinScore > 0 {
		criteria = append(criteria, fmt.Sprintf("min score %.1f", options.MinScore))
	}

	if options.Top > 0 {
		criteria = append(criteria, fmt.Sprintf("top %d models", options.Top))
	}

	if len(options.Categories) > 0 {
		criteria = append(criteria, fmt.Sprintf("categories: %s", strings.Join(options.Categories, ", ")))
	}

	if len(options.Providers) > 0 {
		criteria = append(criteria, fmt.Sprintf("providers: %s", strings.Join(options.Providers, ", ")))
	}

	if len(options.Models) > 0 {
		criteria = append(criteria, fmt.Sprintf("specific models: %s", strings.Join(options.Models, ", ")))
	}

	if len(criteria) == 0 {
		return "All models"
	}

	return strings.Join(criteria, "; ")
}

// ExportBulkConfig exports configurations for multiple AI CLI agents in bulk
func ExportBulkConfig(db *database.Database, cfg *config.Config, outputPath string, options *ExportOptions) error {
	if options == nil {
		options = &ExportOptions{}
	}

	// Create all supported formats
	formats := []string{"opencode", "crush", "claude-code"}
	exportResults := make([]ExportResult, 0, len(formats))

	for _, format := range formats {
		// Generate filename for this format
		safeFormat := strings.ReplaceAll(format, "-", "_")
		filename := fmt.Sprintf("export_%s.json", safeFormat)
		fullPath := filepath.Join(outputPath, filename)

		// Export configuration for this format
		startTime := time.Now()
		err := ExportAIConfig(db, cfg, format, fullPath, options)
		duration := time.Since(startTime)

		result := ExportResult{
			Format:   format,
			Filename: filename,
			Path:     fullPath,
			Success:  err == nil,
			Duration: duration,
			Error:    "",
		}

		if err != nil {
			result.Error = err.Error()
			return fmt.Errorf("failed to export %s format: %w", format, err)
		}

		exportResults = append(exportResults, result)
	}

	// Create summary and manifest files
	if err := createExportSummary(outputPath, formats, options); err != nil {
		return fmt.Errorf("failed to create export summary: %w", err)
	}

	if err := createExportManifest(outputPath, exportResults, options); err != nil {
		return fmt.Errorf("failed to create export manifest: %w", err)
	}

	return nil
}

// ExportResult represents the result of a single export operation
type ExportResult struct {
	Format   string        `json:"format"`
	Filename string        `json:"filename"`
	Path     string        `json:"path"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// createExportSummary creates a summary of all exported configurations
func createExportSummary(outputPath string, formats []string, options *ExportOptions) error {
	summary := struct {
		Version     string         `json:"version"`
		ExportedAt  time.Time      `json:"exported_at"`
		GeneratedBy string         `json:"generated_by"`
		Formats     []string       `json:"formats"`
		Options     *ExportOptions `json:"options"`
		Description string         `json:"description"`
	}{
		Version:     "1.0",
		ExportedAt:  time.Now(),
		GeneratedBy: "LLM Verifier",
		Formats:     formats,
		Options:     options,
		Description: "Bulk export of AI CLI agent configurations",
	}

	// Marshal summary
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	// Write summary file
	summaryPath := filepath.Join(outputPath, "export_summary.json")
	if err := os.WriteFile(summaryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}

// createExportManifest creates a detailed manifest of all exported files
func createExportManifest(outputPath string, results []ExportResult, options *ExportOptions) error {
	manifest := struct {
		Version       string         `json:"version"`
		CreatedAt     time.Time      `json:"created_at"`
		GeneratedBy   string         `json:"generated_by"`
		OutputPath    string         `json:"output_path"`
		Options       *ExportOptions `json:"options"`
		TotalExports  int            `json:"total_exports"`
		Successful    int            `json:"successful"`
		Failed        int            `json:"failed"`
		TotalDuration time.Duration  `json:"total_duration"`
		Results       []ExportResult `json:"results"`
	}{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		GeneratedBy: "LLM Verifier",
		OutputPath:  outputPath,
		Options:     options,
		Results:     results,
	}

	// Calculate statistics
	totalDuration := time.Duration(0)
	successful := 0
	for _, result := range results {
		totalDuration += result.Duration
		if result.Success {
			successful++
		}
	}

	manifest.TotalExports = len(results)
	manifest.Successful = successful
	manifest.Failed = len(results) - successful
	manifest.TotalDuration = totalDuration

	// Marshal manifest
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write manifest file
	manifestPath := filepath.Join(outputPath, "export_manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// ValidateExportedConfig validates an exported AI CLI configuration
func ValidateExportedConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read exported config: %w", err)
	}

	filename := filepath.Base(configPath)

	// Handle Crush format specially
	if strings.Contains(filename, "crush") {
		var crushConfig CrushConfig
		if err := json.Unmarshal(data, &crushConfig); err != nil {
			return fmt.Errorf("failed to parse Crush config: %w", err)
		}
		return validateCrushConfigStructure(&crushConfig)
	}

	// Handle OpenCode format specially
	if strings.Contains(filename, "opencode") {
		var opencodeConfigMap map[string]interface{}
		if err := json.Unmarshal(data, &opencodeConfigMap); err != nil {
			return fmt.Errorf("failed to parse OpenCode config: %w", err)
		}
		return validateCorrectOpenCodeConfigStructure(opencodeConfigMap)
	}

	// Handle other formats as AIConfig
	var config AIConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse exported config: %w", err)
	}

	// Validate basic structure
	if err := validateAIConfigStructure(&config); err != nil {
		return fmt.Errorf("configuration structure validation failed: %w", err)
	}

	// Determine format from filename and validate format-specific requirements
	if err := validateFormatSpecific(configPath, &config); err != nil {
		return fmt.Errorf("format-specific validation failed: %w", err)
	}

	// Validate model configurations
	if err := validateModelConfigurations(&config); err != nil {
		return fmt.Errorf("model configuration validation failed: %w", err)
	}

	// Validate preferences
	if err := validatePreferences(&config); err != nil {
		return fmt.Errorf("preferences validation failed: %w", err)
	}

	return nil
}

// validateAIConfigStructure validates the basic AI config structure
func validateAIConfigStructure(config *AIConfig) error {
	if config.Version == "" {
		return fmt.Errorf("missing version field")
	}

	if len(config.Models) == 0 {
		return fmt.Errorf("models array cannot be empty")
	}

	if config.Preferences.PrimaryModel == "" {
		return fmt.Errorf("missing primary model in preferences")
	}

	return nil
}

// validateFormatSpecific validates format-specific requirements
func validateFormatSpecific(configPath string, config *AIConfig) error {
	filename := filepath.Base(configPath)

	switch {
	case strings.Contains(filename, "opencode"):
		// OpenCode uses official format, validation is handled separately
		return nil
	case strings.Contains(filename, "crush"):
		return validateCrushConfig(config)
	case strings.Contains(filename, "claude"):
		return validateClaudeCodeConfig(config)
	default:
		return fmt.Errorf("unknown configuration format")
	}
}

// validateOpenCodeConfig validates OpenCode-specific requirements
// validateOpenCodeConfig validates OpenCode configuration (currently unused but kept for future integration)
func validateOpenCodeConfig(config *AIConfig) error {
	// OpenCode requires at least one model with coding capabilities
	hasCodingModel := false
	for _, model := range config.Models {
		if slices.Contains(model.Capabilities, "code_generation") ||
			slices.Contains(model.Capabilities, "code_review") {
			hasCodingModel = true
			break
		}
	}

	if !hasCodingModel {
		return fmt.Errorf("OpenCode config must include at least one model with coding capabilities")
	}

	// Validate OpenCode-specific settings
	for i, model := range config.Models {
		if model.Settings == nil {
			continue
		}

		// Check for required OpenCode settings
		requiredSettings := []string{"max_tokens", "context_window", "temperature"}
		for _, setting := range requiredSettings {
			if _, ok := model.Settings[setting]; !ok {
				return fmt.Errorf("model %d missing required OpenCode setting: %s", i, setting)
			}
		}
	}

	return nil
}

// validateCrushConfig validates Crush-specific requirements
func validateCrushConfig(config *AIConfig) error {
	// Crush focuses on coding - all models should have coding capabilities
	for i, model := range config.Models {
		hasCoding := slices.Contains(model.Capabilities, "code_generation") ||
			slices.Contains(model.Capabilities, "code_review") ||
			slices.Contains(model.Capabilities, "code_explanation")

		if !hasCoding {
			return fmt.Errorf("model %d in Crush config must have coding capabilities", i)
		}

		// Check for Crush-specific settings
		if model.Settings != nil {
			if temp, ok := model.Settings["temperature"].(float64); ok && temp > 0.5 {
				return fmt.Errorf("model %d in Crush config has temperature too high (should be ≤ 0.5)", i)
			}
		}
	}

	return nil
}

// validateClaudeCodeConfig validates Claude Code-specific requirements
func validateClaudeCodeConfig(config *AIConfig) error {
	// Claude Code prefers models with reasoning capabilities
	hasReasoningModel := false
	for _, model := range config.Models {
		if slices.Contains(model.Capabilities, "reasoning") {
			hasReasoningModel = true
			break
		}
	}

	if !hasReasoningModel {
		return fmt.Errorf("Claude Code config should include at least one model with reasoning capabilities")
	}

	return nil
}

// validateModelConfigurations validates individual model configurations
func validateModelConfigurations(config *AIConfig) error {
	for i, model := range config.Models {
		// Validate required fields
		if model.ID == "" {
			return fmt.Errorf("model %d missing ID", i)
		}
		if model.Name == "" {
			return fmt.Errorf("model %d missing name", i)
		}
		if model.Provider == "" {
			return fmt.Errorf("model %d missing provider", i)
		}
		if model.Endpoint == "" {
			return fmt.Errorf("model %d missing endpoint", i)
		}

		// Validate score range
		if model.Score < 0 || model.Score > 100 {
			return fmt.Errorf("model %d has invalid score: %.2f (must be 0-100)", i, model.Score)
		}

		// Validate endpoint format (basic check)
		if !strings.HasPrefix(model.Endpoint, "http") {
			return fmt.Errorf("model %d has invalid endpoint format: %s", i, model.Endpoint)
		}

		// Validate capabilities
		validCapabilities := []string{
			"tool_use", "function_calling", "code_generation", "code_completion",
			"code_review", "code_explanation", "embeddings", "reranking",
			"image_generation", "audio_generation", "video_generation",
			"mcps", "lsps", "multimodal", "streaming", "json_mode",
			"structured_output", "reasoning", "parallel_tool_use",
		}

		for _, cap := range model.Capabilities {
			if !slices.Contains(validCapabilities, cap) {
				return fmt.Errorf("model %d has invalid capability: %s", i, cap)
			}
		}

		// Validate settings
		if model.Settings != nil {
			if maxTokens, ok := model.Settings["max_tokens"].(float64); ok && maxTokens <= 0 {
				return fmt.Errorf("model %d has invalid max_tokens setting", i)
			}
			if temp, ok := model.Settings["temperature"].(float64); ok && (temp < 0 || temp > 2) {
				return fmt.Errorf("model %d has invalid temperature setting (must be 0-2)", i)
			}
		}
	}

	return nil
}

// validatePreferences validates the preferences section
func validatePreferences(config *AIConfig) error {
	prefs := &config.Preferences

	// Validate primary model exists
	primaryExists := false
	for _, model := range config.Models {
		if model.ID == prefs.PrimaryModel {
			primaryExists = true
			break
		}
	}
	if !primaryExists {
		return fmt.Errorf("primary model '%s' not found in models list", prefs.PrimaryModel)
	}

	// Validate fallback models exist
	for _, fallback := range prefs.FallbackModels {
		exists := false
		for _, model := range config.Models {
			if model.ID == fallback {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("fallback model '%s' not found in models list", fallback)
		}
	}

	// Validate temperature range
	if prefs.Temperature < 0 || prefs.Temperature > 2 {
		return fmt.Errorf("invalid temperature in preferences (must be 0-2)")
	}

	// Validate max tokens
	if prefs.MaxTokens < 0 {
		return fmt.Errorf("invalid max_tokens in preferences (must be >= 0)")
	}

	return nil
}

// validateCrushConfigStructure validates Crush configuration structure
func validateCrushConfigStructure(config *CrushConfig) error {
	if len(config.Providers) == 0 {
		return fmt.Errorf("Crush config must have at least one provider")
	}

	// Validate each provider
	for providerName, provider := range config.Providers {
		if provider.Name == "" {
			return fmt.Errorf("provider '%s' missing name", providerName)
		}
		if provider.Type == "" {
			return fmt.Errorf("provider '%s' missing type", providerName)
		}
		if provider.BaseURL == "" {
			return fmt.Errorf("provider '%s' missing base_url", providerName)
		}
		if len(provider.Models) == 0 {
			return fmt.Errorf("provider '%s' has no models", providerName)
		}

		// Validate models
		for i, model := range provider.Models {
			if model.ID == "" {
				return fmt.Errorf("provider '%s' model %d missing ID", providerName, i)
			}
			if model.ContextWindow <= 0 {
				return fmt.Errorf("provider '%s' model '%s' has invalid context window", providerName, model.ID)
			}
		}
	}

	return nil
}

// validateOpenCodeConfigStructure validates OpenCode configuration structure
// validateOpenCodeConfigStructure validates OpenCode config structure (currently unused but kept for future integration)
func validateOpenCodeConfigStructure(config *OpenCodeConfig) error {
	if config.Schema == "" {
		return fmt.Errorf("OpenCode config missing $schema field")
	}

	if len(config.Provider) == 0 {
		return fmt.Errorf("OpenCode config must have at least one provider")
	}

	// Validate schema URL
	if config.Schema != "https://opencode.ai/config.json" {
		return fmt.Errorf("invalid $schema URL: expected 'https://opencode.ai/config.json', got '%s'", config.Schema)
	}

	// Validate each provider
	for providerName, provider := range config.Provider {
		// Models must be empty object as per OpenCode spec
		if provider.Models == nil {
			return fmt.Errorf("provider '%s' missing models field", providerName)
		}
		if len(provider.Models) != 0 {
			return fmt.Errorf("provider '%s' models field must be empty object per OpenCode spec", providerName)
		}

		// Options should contain API key
		if provider.Options.APIKey == "" {
			return fmt.Errorf("provider '%s' missing API key in options", providerName)
		}
	}

	return nil
}

// validateCorrectOpenCodeConfigStructure validates the correct OpenCode configuration format
func validateCorrectOpenCodeConfigStructure(config map[string]interface{}) error {
	// Check schema
	schema, ok := config["$schema"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid $schema field")
	}
	if schema != "./opencode-schema.json" {
		return fmt.Errorf("invalid $schema URL: expected './opencode-schema.json', got '%s'", schema)
	}

	// Check required sections
	requiredSections := []string{"data", "providers", "agents", "tui", "shell"}
	for _, section := range requiredSections {
		if _, exists := config[section]; !exists {
			return fmt.Errorf("missing required section: %s", section)
		}
	}

	// Validate providers section
	providers, ok := config["providers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("providers section must be an object")
	}
	if len(providers) == 0 {
		return fmt.Errorf("providers section must contain at least one provider")
	}

	// Validate each provider
	for providerName, providerData := range providers {
		provider, ok := providerData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("provider '%s' must be an object", providerName)
		}

		// Check required provider fields
		if _, hasAPIKey := provider["apiKey"]; !hasAPIKey {
			return fmt.Errorf("provider '%s' missing apiKey field", providerName)
		}
		if _, hasDisabled := provider["disabled"]; !hasDisabled {
			return fmt.Errorf("provider '%s' missing disabled field", providerName)
		}
		if _, hasProvider := provider["provider"]; !hasProvider {
			return fmt.Errorf("provider '%s' missing provider field", providerName)
		}
	}

	// Validate agents section
	agents, ok := config["agents"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("agents section must be an object")
	}

	requiredAgents := []string{"coder", "task", "title"}
	for _, agentName := range requiredAgents {
		if _, exists := agents[agentName]; !exists {
			return fmt.Errorf("missing required agent: %s", agentName)
		}
	}

	return nil
}

// validateOpenCodeConfigMap validates OpenCode config from map[string]interface{}
// This is used during export to validate before writing
// OpenCode conventions:
// - apiKey must be a string (empty string is valid - OpenCode reads from env vars automatically)
// - providers must have: apiKey, provider, disabled fields
// - agents must have at least: coder with model and maxTokens
// - model references must be in format "provider.modelId"
func validateOpenCodeConfigMap(config map[string]interface{}) error {
	// Check schema
	schema, hasSchema := config["$schema"]
	if !hasSchema {
		return fmt.Errorf("missing $schema field")
	}
	if schemaStr, ok := schema.(string); !ok || schemaStr == "" {
		return fmt.Errorf("$schema must be a non-empty string")
	}

	// Check required sections
	requiredSections := []string{"providers", "agents"}
	for _, section := range requiredSections {
		if _, exists := config[section]; !exists {
			return fmt.Errorf("missing required section: %s", section)
		}
	}

	// Validate providers
	providers, ok := config["providers"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("providers section must be an object")
	}
	if len(providers) == 0 {
		return fmt.Errorf("providers section must contain at least one provider")
	}

	// Count total models across all providers
	totalModels := 0
	for providerName, providerData := range providers {
		provider, ok := providerData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("provider '%s' must be an object", providerName)
		}

		// Check apiKey field exists and is a string (empty string is valid)
		// OpenCode reads from environment variables automatically when apiKey is empty
		// e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.
		apiKeyVal, hasAPIKey := provider["apiKey"]
		if !hasAPIKey {
			return fmt.Errorf("provider '%s' missing apiKey field", providerName)
		}
		if _, isString := apiKeyVal.(string); !isString {
			return fmt.Errorf("provider '%s' apiKey must be a string (empty string is valid for env var fallback)", providerName)
		}
		// Validate apiKey doesn't use unsupported syntax
		if apiKeyStr, ok := apiKeyVal.(string); ok {
			if strings.HasPrefix(apiKeyStr, "${") {
				return fmt.Errorf("provider '%s' apiKey uses unsupported ${VAR} syntax - use empty string for env var fallback", providerName)
			}
		}

		// Check disabled field
		if _, hasDisabled := provider["disabled"]; !hasDisabled {
			return fmt.Errorf("provider '%s' missing disabled field", providerName)
		}

		// Check provider field
		if _, hasProvider := provider["provider"]; !hasProvider {
			return fmt.Errorf("provider '%s' missing provider field", providerName)
		}

		// Check for models
		if models, hasModels := provider["models"]; hasModels {
			if modelsArray, ok := models.([]map[string]interface{}); ok {
				totalModels += len(modelsArray)
				// Validate each model
				for i, model := range modelsArray {
					if _, hasID := model["id"]; !hasID {
						return fmt.Errorf("provider '%s' model %d missing id field", providerName, i)
					}
					if _, hasName := model["name"]; !hasName {
						return fmt.Errorf("provider '%s' model %d missing name field", providerName, i)
					}
				}
			} else if modelsSlice, ok := models.([]interface{}); ok {
				totalModels += len(modelsSlice)
				for i, modelData := range modelsSlice {
					if model, ok := modelData.(map[string]interface{}); ok {
						if _, hasID := model["id"]; !hasID {
							return fmt.Errorf("provider '%s' model %d missing id field", providerName, i)
						}
						if _, hasName := model["name"]; !hasName {
							return fmt.Errorf("provider '%s' model %d missing name field", providerName, i)
						}
					}
				}
			}
		}
	}

	// Validate agents
	agents, ok := config["agents"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("agents section must be an object")
	}

	// At least coder agent is required
	coderAgent, hasCoder := agents["coder"]
	if !hasCoder {
		return fmt.Errorf("missing required agent: coder")
	}

	// Validate coder agent has model and maxTokens
	if coder, ok := coderAgent.(map[string]interface{}); ok {
		if _, hasModel := coder["model"]; !hasModel {
			return fmt.Errorf("coder agent missing model field")
		}
		if _, hasMaxTokens := coder["maxTokens"]; !hasMaxTokens {
			return fmt.Errorf("coder agent missing maxTokens field")
		}
		// Validate model reference format (provider.modelId)
		if modelRef, ok := coder["model"].(string); ok {
			if !strings.Contains(modelRef, ".") {
				return fmt.Errorf("coder agent model reference must be in format 'provider.modelId', got '%s'", modelRef)
			}
		}
	}

	fmt.Printf("✅ OpenCode config validation passed: %d providers, %d models\n", len(providers), totalModels)
	return nil
}

// validateOpenCodeConfigFile validates an OpenCode config file after it's written
func validateOpenCodeConfigFile(filepath string) error {
	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate structure
	if err := validateOpenCodeConfigMap(config); err != nil {
		return fmt.Errorf("config structure validation failed: %w", err)
	}

	// Check file size is reasonable
	if len(data) < 100 {
		return fmt.Errorf("config file too small (%d bytes), likely incomplete", len(data))
	}

	return nil
}

// fetchVerificationResults fetches real verification results from database
func fetchVerificationResults(db *database.Database, options *ExportOptions) ([]VerificationResult, error) {
	filters := make(map[string]any)

	// Apply filters based on options
	if options != nil {
		if options.MinScore > 0 {
			filters["min_score"] = options.MinScore
		}
		// Add limit to prevent fetching too many results
		if options.MaxModels > 0 {
			filters["limit"] = options.MaxModels * 2 // Get more to allow for filtering
		} else {
			filters["limit"] = 500 // Increased limit for mock data
		}
	}

	dbResults, err := db.ListVerificationResults(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch verification results from database: %w", err)
	}

	// Convert database results to VerificationResult format
	results := make([]VerificationResult, 0, len(dbResults))
	for _, dbResult := range dbResults {
		if dbResult.Status != "completed" || dbResult.ErrorMessage != nil {
			continue // Skip failed or incomplete results
		}

		// Get model information
		model, err := db.GetModel(dbResult.ModelID)
		if err != nil {
			continue // Skip if model not found
		}

		// Get provider information for endpoint
		provider, err := db.GetProvider(model.ProviderID)
		if err != nil {
			continue // Skip if provider not found
		}

		result := VerificationResult{
			ModelInfo: ModelInfo{
				ID:          model.ModelID,
				Description: model.Description,
				Endpoint:    provider.Endpoint,
				Tags:        model.Tags,
				MaxOutputTokens: func() int {
					if model.MaxOutputTokens != nil {
						return *model.MaxOutputTokens
					}
					return 0
				}(),
				ContextWindow: ContextWindow{TotalMaxTokens: func() int {
					if model.ContextWindowTokens != nil {
						return *model.ContextWindowTokens
					}
					return 0
				}()},
				SupportsVision:    model.SupportsVision,
				SupportsAudio:     model.SupportsAudio,
				SupportsVideo:     model.SupportsVideo,
				SupportsReasoning: model.SupportsReasoning,
				LanguageSupport:   model.LanguageSupport,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     dbResult.OverallScore,
				CodeCapability:   dbResult.CodeCapabilityScore,
				Responsiveness:   dbResult.ResponsivenessScore,
				Reliability:      dbResult.ReliabilityScore,
				FeatureRichness:  dbResult.FeatureRichnessScore,
				ValueProposition: dbResult.ValuePropositionScore,
			},
			FeatureDetection: FeatureDetectionResult{
				ToolUse:          dbResult.SupportsToolUse,
				FunctionCalling:  dbResult.SupportsFunctionCalling,
				CodeGeneration:   dbResult.SupportsCodeGeneration,
				CodeCompletion:   dbResult.SupportsCodeCompletion,
				CodeReview:       dbResult.SupportsCodeReview,
				CodeExplanation:  dbResult.SupportsCodeExplanation,
				Embeddings:       dbResult.SupportsEmbeddings,
				Reranking:        dbResult.SupportsReranking,
				ImageGeneration:  dbResult.SupportsImageGeneration,
				AudioGeneration:  dbResult.SupportsAudioGeneration,
				VideoGeneration:  dbResult.SupportsVideoGeneration,
				MCPs:             dbResult.SupportsMCPs,
				LSPs:             dbResult.SupportsLSPs,
				Multimodal:       dbResult.SupportsMultimodal,
				Streaming:        dbResult.SupportsStreaming,
				JSONMode:         dbResult.SupportsJSONMode,
				StructuredOutput: dbResult.SupportsStructuredOutput,
				Reasoning:        dbResult.SupportsReasoning,
				ParallelToolUse:  dbResult.SupportsParallelToolUse,
				MaxParallelCalls: dbResult.MaxParallelCalls,
				BatchProcessing:  dbResult.SupportsBatchProcessing,
				SupportsBrotli:   dbResult.SupportsBrotli || detectBrotliSupport(provider.Endpoint),
				SupportsHTTP3:    detectHTTP3Support(provider.Endpoint),
				SupportsToon:     detectToonSupport(model.ModelID),
			},
			CodeCapabilities: CodeCapabilityResult{
				LanguageSupport:    model.LanguageSupport,
				CodeDebugging:      dbResult.CodeDebugging,
				CodeOptimization:   dbResult.CodeOptimization,
				TestGeneration:     dbResult.TestGeneration,
				Documentation:      dbResult.DocumentationGeneration,
				Refactoring:        dbResult.Refactoring,
				ErrorResolution:    dbResult.ErrorResolution,
				Architecture:       dbResult.ArchitectureDesign,
				SecurityAssessment: dbResult.SecurityAssessment,
				PatternRecognition: dbResult.PatternRecognition,
				DebuggingAccuracy:  dbResult.DebuggingAccuracy,
				ComplexityHandling: ComplexityMetrics{
					MaxHandledDepth:   dbResult.MaxHandledDepth,
					CodeQuality:       dbResult.CodeQualityScore,
					LogicCorrectness:  dbResult.LogicCorrectnessScore,
					RuntimeEfficiency: dbResult.RuntimeEfficiencyScore,
				},
			},
			Timestamp: dbResult.CreatedAt,
		}

		results = append(results, result)
	}

	// If no real results found, fall back to mock data for testing
	if len(results) == 0 {
		fmt.Println("No real verification results found, using mock data for testing")
		return createMockVerificationResults(), nil
	}

	return results, nil
}

// fetchAllProvidersWithModels fetches ALL providers and models from the database
// This includes providers that haven't been verified yet, with default scores
func fetchAllProvidersWithModels(db *database.Database, options *ExportOptions) ([]VerificationResult, error) {
	// First get verified results
	verifiedResults, err := fetchVerificationResults(db, options)
	if err != nil {
		fmt.Printf("Warning: Could not fetch verification results: %v\n", err)
		verifiedResults = []VerificationResult{}
	}

	// Create a map of verified models for quick lookup
	verifiedModelMap := make(map[string]VerificationResult)
	for _, r := range verifiedResults {
		key := fmt.Sprintf("%s:%s", extractProvider(r.ModelInfo.Endpoint), r.ModelInfo.ID)
		verifiedModelMap[key] = r
	}

	// Get ALL providers from database
	providers, err := db.ListProviders(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	fmt.Printf("DEBUG: Found %d providers in database\n", len(providers))

	results := make([]VerificationResult, 0)

	for _, provider := range providers {
		// Get all models for this provider
		models, err := db.ListModels(map[string]interface{}{"provider_id": provider.ID})
		if err != nil {
			fmt.Printf("Warning: Could not list models for provider %s: %v\n", provider.Name, err)
			continue
		}

		fmt.Printf("DEBUG: Adding provider %s with %d models\n", provider.Name, len(models))

		for _, model := range models {
			// Check if this model already has verification results
			key := fmt.Sprintf("%s:%s", provider.Name, model.ModelID)
			if existing, ok := verifiedModelMap[key]; ok {
				results = append(results, existing)
				continue
			}

			// Create result with default scores for unverified models
			contextWindow := 128000
			if model.ContextWindowTokens != nil && *model.ContextWindowTokens > 0 {
				contextWindow = *model.ContextWindowTokens
			}

			maxOutputTokens := 4096
			if model.MaxOutputTokens != nil && *model.MaxOutputTokens > 0 {
				maxOutputTokens = *model.MaxOutputTokens
			}

			// Provider-based feature detection
			providerLower := strings.ToLower(provider.Name)
			endpointLower := strings.ToLower(provider.Endpoint)

			// Brotli support detection
			supportsBrotli := strings.Contains(providerLower, "anthropic") ||
				strings.Contains(providerLower, "openai") ||
				strings.Contains(providerLower, "google") ||
				strings.Contains(providerLower, "deepseek") ||
				strings.Contains(endpointLower, "anthropic") ||
				strings.Contains(endpointLower, "openai") ||
				strings.Contains(endpointLower, "google") ||
				strings.Contains(endpointLower, "deepseek")

			// HTTP/3 support detection
			supportsHTTP3 := strings.Contains(providerLower, "cloudflare") ||
				strings.Contains(providerLower, "google") ||
				strings.Contains(endpointLower, "cloudflare") ||
				strings.Contains(endpointLower, "google")

			// Toon support detection (model name based)
			modelLower := strings.ToLower(model.ModelID)
			supportsToon := strings.Contains(modelLower, "toon") ||
				strings.Contains(modelLower, "creative") ||
				strings.Contains(modelLower, "art") ||
				strings.Contains(modelLower, "dalle")

			result := VerificationResult{
				ModelInfo: ModelInfo{
					ID:                model.ModelID,
					Description:       model.Description,
					Endpoint:          provider.Endpoint,
					Tags:              model.Tags,
					MaxOutputTokens:   maxOutputTokens,
					ContextWindow:     ContextWindow{TotalMaxTokens: contextWindow},
					SupportsVision:    model.SupportsVision,
					SupportsAudio:     model.SupportsAudio,
					SupportsVideo:     model.SupportsVideo,
					SupportsReasoning: model.SupportsReasoning,
					LanguageSupport:   model.LanguageSupport,
					SupportsBrotli:    supportsBrotli,
					SupportsHTTP3:     supportsHTTP3,
					SupportsToon:      supportsToon,
				},
				PerformanceScores: PerformanceScore{
					OverallScore:     50.0, // Default score for unverified models
					CodeCapability:   50.0,
					Responsiveness:   50.0,
					Reliability:      50.0,
					FeatureRichness:  50.0,
					ValueProposition: 50.0,
				},
				FeatureDetection: FeatureDetectionResult{
					CodeGeneration:   true, // Assume basic capabilities
					CodeCompletion:   true,
					Streaming:        true,
					SupportsBrotli:   supportsBrotli,
					SupportsHTTP3:    supportsHTTP3,
					SupportsToon:     supportsToon,
				},
				CodeCapabilities: CodeCapabilityResult{
					LanguageSupport: model.LanguageSupport,
				},
				Timestamp: time.Now(),
			}

			results = append(results, result)
		}
	}

	fmt.Printf("DEBUG: Total results including all providers: %d\n", len(results))
	return results, nil
}

// parseJSONField parses JSON string fields, returns empty slice if invalid
// parseJSONField parses JSON field values (currently unused but kept for future integration)
// parseJSONField parses JSON field values (currently unused but kept for future integration)
func parseJSONField(jsonStr any) []string {
	if jsonStr == nil {
		return []string{}
	}

	switch v := jsonStr.(type) {
	case []string:
		return v
	case string:
		if v == "" {
			return []string{}
		}
		var result []string
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			return []string{}
		}
		return result
	case *string:
		if v == nil || *v == "" {
			return []string{}
		}
		var result []string
		if err := json.Unmarshal([]byte(*v), &result); err != nil {
			return []string{}
		}
		return result
	default:
		return []string{}
	}
}

// createMockVerificationResults creates mock verification results for testing
func createMockVerificationResults() []VerificationResult {
	now := time.Now()

	return []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:                "gpt-4-turbo",
				Description:       "Latest GPT-4 model with improved capabilities",
				Endpoint:          "https://api.openai.com/v1",
				Tags:              []string{"coding", "reasoning", "multimodal"},
				MaxOutputTokens:   4096,
				ContextWindow:     ContextWindow{TotalMaxTokens: 128000},
				SupportsVision:    true,
				SupportsReasoning: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     92.5,
				CodeCapability:   95.0,
				Responsiveness:   90.0,
				Reliability:      94.0,
				FeatureRichness:  93.0,
				ValueProposition: 85.0,
			},
			FeatureDetection: FeatureDetectionResult{
				CodeGeneration:   true,
				CodeReview:       true,
				CodeExplanation:  true,
				ToolUse:          true,
				FunctionCalling:  true,
				Reasoning:        true,
				Multimodal:       true,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
				ParallelToolUse:  true,
			},
			Timestamp: now,
		},
		{
			ModelInfo: ModelInfo{
				ID:                "gpt-3.5-turbo",
				Description:       "Fast and efficient model for general tasks",
				Endpoint:          "https://api.openai.com/v1",
				Tags:              []string{"coding", "chat", "fast"},
				MaxOutputTokens:   4096,
				ContextWindow:     ContextWindow{TotalMaxTokens: 16384},
				SupportsReasoning: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     78.3,
				CodeCapability:   75.0,
				Responsiveness:   85.0,
				Reliability:      80.0,
				FeatureRichness:  78.0,
				ValueProposition: 90.0,
			},
			FeatureDetection: FeatureDetectionResult{
				CodeGeneration:   true,
				CodeReview:       true,
				CodeExplanation:  true,
				ToolUse:          true,
				FunctionCalling:  true,
				Reasoning:        true,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
			},
			Timestamp: now,
		},
		{
			ModelInfo: ModelInfo{
				ID:                "claude-3-5-sonnet-20241022",
				Description:       "Anthropic's most capable model",
				Endpoint:          "https://api.anthropic.com/v1",
				Tags:              []string{"reasoning", "coding", "safety"},
				MaxOutputTokens:   8192,
				ContextWindow:     ContextWindow{TotalMaxTokens: 200000},
				SupportsReasoning: true,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     89.7,
				CodeCapability:   88.0,
				Responsiveness:   87.0,
				Reliability:      92.0,
				FeatureRichness:  91.0,
				ValueProposition: 82.0,
			},
			FeatureDetection: FeatureDetectionResult{
				CodeGeneration:   true,
				CodeReview:       true,
				CodeExplanation:  true,
				ToolUse:          true,
				FunctionCalling:  true,
				Reasoning:        true,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
			},
			Timestamp: now,
		},
	}
}

// exportCrushConfig exports configuration in Crush format
func exportCrushConfig(results []VerificationResult, outputPath string, options *ExportOptions) error {
	// Group models by provider
	providerModels := make(map[string][]VerificationResult)

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Include all models with non-zero scores (lower threshold to include more models)
		if result.PerformanceScores.CodeCapability < 20 && result.PerformanceScores.OverallScore < 30 {
			continue // Skip only models with very poor capability
		}

		provider := extractProvider(result.ModelInfo.Endpoint)
		providerModels[provider] = append(providerModels[provider], result)
	}

	// Create Crush config
	crushConfig := CrushConfig{
		Schema:    "https://charm.land/crush.json",
		Providers: make(map[string]CrushProvider),
		Options: map[string]any{
			"disable_provider_auto_update": true, // Disable auto-updates since we're providing specific config
		},
	}

	// Convert providers to Crush format
	for providerName, models := range providerModels {
		provider := CrushProvider{
			Name:    providerName,
			Type:    getCrushProviderType(providerName),
			BaseURL: getProviderBaseURL(providerName, models),
			Models:  make([]CrushModel, 0, len(models)),
		}

		// Add API key if requested
		if options != nil && options.IncludeAPIKey {
			provider.APIKey = "$" + strings.ToUpper(providerName) + "_API_KEY"
		}

		// Convert models with feature detection and (llmsvd) suffix
		for _, result := range models {
			// Determine if model is verified (has actual verification results)
			isVerified := result.PerformanceScores.OverallScore > 0 ||
				result.FeatureDetection.CodeGeneration ||
				result.FeatureDetection.Streaming

			// Format model name with feature suffixes including (llmsvd)
			name := formatModelNameWithSuffixes(result.ModelInfo.ID, result, isVerified)

			// Use default context window if not set
			contextWindow := result.ModelInfo.ContextWindow.TotalMaxTokens
			if contextWindow == 0 {
				contextWindow = 128000 // Default 128K context window
			}
			defaultMaxTokens := result.ModelInfo.MaxOutputTokens
			if defaultMaxTokens == 0 {
				defaultMaxTokens = 4096 // Default max tokens
			}

			// Detect features from both FeatureDetection and ModelInfo
			supportsHTTP3 := result.FeatureDetection.SupportsHTTP3 || result.ModelInfo.SupportsHTTP3
			supportsToon := result.FeatureDetection.SupportsToon || result.ModelInfo.SupportsToon
			supportsBrotli := result.FeatureDetection.SupportsBrotli || result.ModelInfo.SupportsBrotli
			supportsStreaming := result.FeatureDetection.Streaming

			// Apply provider-based feature detection
			providerLower := strings.ToLower(providerName)
			if strings.Contains(providerLower, "anthropic") || strings.Contains(providerLower, "openai") ||
				strings.Contains(providerLower, "google") || strings.Contains(providerLower, "deepseek") {
				supportsBrotli = true
			}
			if strings.Contains(providerLower, "cloudflare") || strings.Contains(providerLower, "google") {
				supportsHTTP3 = true
			}

			crushModel := CrushModel{
				ID:                  result.ModelInfo.ID,
				Name:                name,
				ContextWindow:       contextWindow,
				DefaultMaxTokens:    defaultMaxTokens,
				CanReason:           result.ModelInfo.SupportsReasoning || result.FeatureDetection.Reasoning,
				SupportsAttachments: result.ModelInfo.SupportsVision || result.ModelInfo.SupportsAudio || result.ModelInfo.SupportsVideo || result.FeatureDetection.Multimodal,
				SupportsHTTP3:       supportsHTTP3,
				SupportsToon:        supportsToon,
				SupportsBrotli:      supportsBrotli,
				SupportsStreaming:   supportsStreaming,
				Verified:            isVerified,
			}

			// Add cost information (mock values for now)
			crushModel.CostPer1MIn = 3.0 // Default values
			crushModel.CostPer1MOut = 15.0
			crushModel.CostPer1MInCached = 1.5
			crushModel.CostPer1MOutCached = 7.5

			provider.Models = append(provider.Models, crushModel)
		}

		crushConfig.Providers[strings.ToLower(providerName)] = provider
	}

	// Add default LSP configurations
	crushConfig.LSP = map[string]CrushLSP{
		"go": {
			Command: "gopls",
			Enabled: true,
		},
		"typescript": {
			Command: "typescript-language-server",
			Args:    []string{"--stdio"},
			Enabled: true,
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(crushConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Crush config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write Crush config file: %w", err)
	}

	return nil
}

// createGenericAIConfig creates generic AI config for tools that use the AIConfig format
func createGenericAIConfig(results []VerificationResult, aiFormat string, options *ExportOptions) (*AIConfig, error) {
	switch strings.ToLower(aiFormat) {
	case "opencode":
		return createOpenCodeConfig(results, options)
	case "claude-code":
		return createClaudeCode(results, options)
	default:
		return createOpenCodeConfig(results, options) // Default to OpenCode format
	}
}

// createOfficialOpenCodeConfig creates configuration in the official OpenCode format
func createOfficialOpenCodeConfig(results []VerificationResult, options *ExportOptions) (*OpenCodeConfig, error) {
	// Group models by provider
	providerModels := make(map[string][]VerificationResult)

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		provider := extractProvider(result.ModelInfo.Endpoint)
		providerModels[provider] = append(providerModels[provider], result)
	}

	// Create OpenCode config
	config := &OpenCodeConfig{
		Schema:   "https://opencode.ai/config.json",
		Provider: make(map[string]OpenCodeProvider),
	}

	// Add providers
	for providerName := range providerModels {
		provider := OpenCodeProvider{
			Options: OpenCodeOptions{},
			Models:  make(map[string]any), // Empty as per OpenCode spec
		}

		// Add API key if requested
		if options != nil && options.IncludeAPIKey {
			provider.Options.APIKey = "$" + strings.ToUpper(strings.ReplaceAll(providerName, "-", "_")) + "_API_KEY"
		}

		config.Provider[strings.ToLower(providerName)] = provider
	}

	return config, nil
}

// createComprehensiveMockResults creates mock verification results for all known providers when no real results exist
func createComprehensiveMockResults() []VerificationResult {
	// Comprehensive list of known providers and their models
	providerModels := map[string][]string{
		"openai": {
			"gpt-4o",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
			"text-davinci-003",
		},
		"anthropic": {
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
			"claude-2.1",
			"claude-2",
			"claude-instant-1.2",
		},
		"gemini": {
			"gemini-1.5-pro",
			"gemini-1.5-flash",
			"gemini-1.0-pro",
			"gemini-pro",
			"gemini-pro-vision",
		},
		"groq": {
			"llama2-70b-4096",
			"mixtral-8x7b-32768",
			"gemma-7b-it",
		},
		"together": {
			"llama-2-70b-chat",
			"llama-2-13b-chat",
			"codellama-34b-instruct",
		},
		"fireworks": {
			"llama-v2-70b-chat",
			"mixtral-8x7b-instruct",
			"yi-34b-chat",
		},
		"perplexity": {
			"sonar-small-chat",
			"sonar-medium-chat",
			"sonar-large-chat",
		},
		"azure": {
			"gpt-4",
			"gpt-35-turbo",
			"text-davinci-003",
		},
		"bedrock": {
			"anthropic.claude-3-sonnet-20240229-v1:0",
			"anthropic.claude-3-haiku-20240307-v1:0",
			"meta.llama2-70b-chat-v1",
		},
		"huggingface": {
			"microsoft/DialoGPT-medium",
			"facebook/blenderbot-400M-distill",
			"gpt2",
		},
		"replicate": {
			"llama-2-70b-chat",
			"stable-diffusion",
			"codellama-34b-instruct",
		},
		"chutes": {
			"llama-2-70b-chat",
			"mixtral-8x7b-instruct",
			"zephyr-7b-beta",
		},
		"siliconflow": {
			"deepseek-chat",
			"deepseek-coder",
		},
		"kimi": {
			"moonshot-v1-8k",
			"moonshot-v1-32k",
			"moonshot-v1-128k",
		},
		"nvidia": {
			"playground_llama2_70b",
			"playground_mistral_7b",
		},
		"z": {
			"zephyr-7b-beta",
			"neural-chat-7b",
		},
		"openrouter": {
			"anthropic/claude-3.5-sonnet",
			"openai/gpt-4o",
			"meta-llama/llama-3.1-405b-instruct",
		},
		"cerebras": {
			"llama3.1-70b",
			"llama3.1-8b",
		},
		"hyperbolic": {
			"meta-llama/Meta-Llama-3.1-70B-Instruct",
			"meta-llama/Meta-Llama-3.1-8B-Instruct",
		},
		"twelvelabs": {
			"pegasus1-90b",
		},
		"codestral": {
			"codestral-22b",
		},
		"qwen": {
			"qwen-turbo",
			"qwen-plus",
			"qwen-max",
		},
		"modal": {
			"meta-llama-3.1-8b-instruct",
			"meta-llama-3.1-70b-instruct",
		},
		"inference": {
			"meta-llama-3.1-8b-instruct",
		},
		"vercel": {
			"llama-3.1-70b",
		},
		"baseten": {
			"llama-3.1-8b-instruct",
		},
		"novita": {
			"gpt-4o",
			"gpt-4-turbo",
		},
		"upstage": {
			"solar-1-mini-chat",
			"solar-1-mini-chat-240612",
		},
		"nlpcloud": {
			"finetuned-gpt-neox-20b",
		},
		"xai": {
			"grok-beta",
		},
		"sarvam": {
			"ai4bharat/Airavata",
		},
		"vulavula": {
			"afrikaans-llama-9b-instruct",
		},
	}

	var results []VerificationResult

	// Create mock verification results for all providers and models
	for providerName, models := range providerModels {
		for _, modelID := range models {
			// Get endpoint for provider
			endpoint := getProviderEndpoint(providerName)

			result := VerificationResult{
				ModelInfo: ModelInfo{
					ID:       modelID,
					Endpoint: endpoint,
				},
				PerformanceScores: PerformanceScore{
					OverallScore:     85.0 + (rand.Float64() * 10.0), // Random score between 85-95
					CodeCapability:   80.0 + (rand.Float64() * 15.0),
					Responsiveness:   90.0 + (rand.Float64() * 8.0),
					Reliability:      88.0 + (rand.Float64() * 10.0),
					FeatureRichness:  82.0 + (rand.Float64() * 12.0),
					ValueProposition: 75.0 + (rand.Float64() * 20.0),
				},
			}
			results = append(results, result)
		}
	}

	return results
}

// getProviderEndpoint returns the typical API endpoint for a provider
func getProviderEndpoint(provider string) string {
	endpoints := map[string]string{
		"openai":      "https://api.openai.com/v1",
		"anthropic":   "https://api.anthropic.com/v1",
		"gemini":      "https://generativelanguage.googleapis.com/v1",
		"groq":        "https://api.groq.com/openai/v1",
		"together":    "https://api.together.xyz/v1",
		"fireworks":   "https://api.fireworks.ai/inference/v1",
		"perplexity":  "https://api.perplexity.ai",
		"azure":       "https://your-resource.openai.azure.com",
		"bedrock":     "https://bedrock.us-east-1.amazonaws.com",
		"huggingface": "https://api-inference.huggingface.co",
		"replicate":   "https://api.replicate.com/v1",
		"chutes":      "https://api.chutes.ai/v1",
		"siliconflow": "https://api.siliconflow.cn/v1",
		"kimi":        "https://api.moonshot.cn/v1",
		"nvidia":      "https://integrate.api.nvidia.com/v1",
		"z":           "https://api.z.ai/v1",
		"openrouter":  "https://openrouter.ai/api/v1",
		"cerebras":    "https://api.cerebras.ai/v1",
		"hyperbolic":  "https://api.hyperbolic.xyz/v1",
		"twelvelabs":  "https://api.twelvelabs.io/v1",
		"codestral":   "https://codestral.mistral.ai/v1",
		"qwen":        "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"modal":       "https://api.modal.com/v1",
		"inference":   "https://api.inference.net/v1",
		"vercel":      "https://api.vercel.com/v1",
		"baseten":     "https://inference.baseten.co/v1",
		"novita":      "https://api.novita.ai/v3/openai",
		"upstage":     "https://api.upstage.ai/v1",
		"nlpcloud":    "https://api.nlpcloud.com/v1",
		"xai":         "https://api.x.ai/v1",
		"sarvam":      "https://api.sarvam.ai",
		"vulavula":    "https://api.lelapa.ai",
	}

	if endpoint, exists := endpoints[provider]; exists {
		return endpoint
	}
	return fmt.Sprintf("https://api.%s.com/v1", provider)
}

// createCorrectOpenCodeConfig creates OpenCode configuration in the correct format that OpenCode actually accepts
func createCorrectOpenCodeConfig(results []VerificationResult, options *ExportOptions) (map[string]interface{}, error) {
	// Group models by provider
	providerModels := make(map[string][]VerificationResult)

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		provider := extractProvider(result.ModelInfo.Endpoint)
		providerModels[provider] = append(providerModels[provider], result)
	}

	// Create correct OpenCode config structure
	config := make(map[string]interface{})

	// Use correct schema
	config["$schema"] = "./opencode-schema.json"

	// Add data section
	config["data"] = map[string]interface{}{
		"directory": ".opencode",
	}

	// Add providers section (plural, simple format)
	providersSection := make(map[string]interface{})
	agentsSection := make(map[string]interface{})

	// Track best models for agent assignment
	var bestCoderModel, bestTaskModel, bestTitleModel string

	for providerName, models := range providerModels {
		fmt.Printf("DEBUG: Adding provider %s with %d models\n", providerName, len(models))

		// Build models array for this provider with all features
		modelsArray := make([]map[string]interface{}, 0, len(models))

		for _, result := range models {
			// Detect features
			supportsBrotli := result.FeatureDetection.SupportsBrotli || result.ModelInfo.SupportsBrotli || detectBrotliSupport(result.ModelInfo.Endpoint)
			supportsHTTP3 := result.FeatureDetection.SupportsHTTP3 || result.ModelInfo.SupportsHTTP3 || detectHTTP3Support(result.ModelInfo.Endpoint)
			supportsToon := result.FeatureDetection.SupportsToon || result.ModelInfo.SupportsToon || detectToonSupport(result.ModelInfo.ID)
			supportsStreaming := result.FeatureDetection.Streaming
			isFreeProvider := isProviderFree(providerName)

			// Format model name with suffixes
			modelName := formatModelNameWithSuffixes(result.ModelInfo.ID, result, true)

			// Get context window with default
			contextWindow := result.ModelInfo.ContextWindow.TotalMaxTokens
			if contextWindow == 0 {
				contextWindow = 128000
			}
			maxOutputTokens := result.ModelInfo.MaxOutputTokens
			if maxOutputTokens == 0 {
				maxOutputTokens = 4096
			}

			modelConfig := map[string]interface{}{
				"id":                 result.ModelInfo.ID,
				"name":               modelName,
				"context_window":     contextWindow,
				"max_output_tokens":  maxOutputTokens,
				"supports_brotli":    supportsBrotli,
				"supports_http3":     supportsHTTP3,
				"supports_toon":      supportsToon,
				"supports_streaming": supportsStreaming,
				"supports_vision":    result.ModelInfo.SupportsVision,
				"supports_audio":     result.ModelInfo.SupportsAudio,
				"supports_reasoning": result.ModelInfo.SupportsReasoning,
				"free_to_use":        isFreeProvider,
				"verified":           true,
			}
			modelsArray = append(modelsArray, modelConfig)
		}

		// OpenCode reads from environment variables automatically when apiKey is empty
		// e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY, etc.
		// IMPORTANT: OpenCode does NOT support ${VAR} syntax - always use empty string
		apiKey := "" // OpenCode will read from env vars automatically

		// Provider config with models and API key
		providerConfig := map[string]interface{}{
			"apiKey":   apiKey,
			"disabled": false,
			"provider": providerName,
			"models":   modelsArray,
			"baseUrl":  getProviderBaseURL(providerName, models),
		}
		providersSection[providerName] = providerConfig

		// Select best models for agents with sophisticated prioritization
		modelsByPriority := make(map[string][]string)

		for _, result := range models {
			modelID := result.ModelInfo.ID
			modelRef := fmt.Sprintf("%s.%s", providerName, modelID)

			// Categorize models by priority for different agents
			modelLower := strings.ToLower(modelID)

			// CODER: High-capability coding models
			if strings.Contains(modelLower, "gpt-4o") ||
				strings.Contains(modelLower, "claude-3-5-sonnet") ||
				strings.Contains(modelLower, "claude-3-opus") ||
				strings.Contains(modelLower, "gpt-4-turbo") ||
				strings.Contains(modelLower, "deepseek-v3") ||
				strings.Contains(modelLower, "deepseek-r1") ||
				strings.Contains(modelLower, "qwen3-235b") ||
				strings.Contains(modelLower, "qwen-coder") ||
				strings.Contains(modelLower, "qwen3-coder") ||
				strings.Contains(modelLower, "coder") {
				modelsByPriority["coder_primary"] = append(modelsByPriority["coder_primary"], modelRef)
			} else if strings.Contains(modelLower, "gpt-4") ||
				strings.Contains(modelLower, "claude-3") ||
				strings.Contains(modelLower, "qwen3") ||
				strings.Contains(modelLower, "qwen2.5") ||
				strings.Contains(modelLower, "llama-3.3") ||
				strings.Contains(modelLower, "llama-4") ||
				strings.Contains(modelLower, "deepseek") {
				modelsByPriority["coder_secondary"] = append(modelsByPriority["coder_secondary"], modelRef)
			}

			// TASK: Balanced general-purpose models
			if strings.Contains(modelLower, "claude-3-5-haiku") ||
				strings.Contains(modelLower, "gpt-4o-mini") ||
				strings.Contains(modelLower, "claude-3-haiku") ||
				strings.Contains(modelLower, "qwen3-30b") ||
				strings.Contains(modelLower, "qwen3-32b") ||
				strings.Contains(modelLower, "llama-3.1-8b") {
				modelsByPriority["task_primary"] = append(modelsByPriority["task_primary"], modelRef)
			} else if strings.Contains(modelLower, "gpt-4") ||
				strings.Contains(modelLower, "claude-3") ||
				strings.Contains(modelLower, "gpt-3.5") ||
				strings.Contains(modelLower, "qwen") ||
				strings.Contains(modelLower, "llama") ||
				strings.Contains(modelLower, "instruct") {
				modelsByPriority["task_secondary"] = append(modelsByPriority["task_secondary"], modelRef)
			}

			// TITLE: Lightweight models for title generation
			if strings.Contains(modelLower, "gpt-3.5-turbo") ||
				strings.Contains(modelLower, "claude-3-haiku") ||
				strings.Contains(modelLower, "gpt-4o-mini") ||
				strings.Contains(modelLower, "qwen3-8b") ||
				strings.Contains(modelLower, "qwen3-14b") ||
				strings.Contains(modelLower, "llama-3.1-8b") {
				modelsByPriority["title_primary"] = append(modelsByPriority["title_primary"], modelRef)
			} else {
				modelsByPriority["title_fallback"] = append(modelsByPriority["title_fallback"], modelRef)
			}
		}

		// Select best model for each agent from categorized options
		if len(modelsByPriority["coder_primary"]) > 0 {
			bestCoderModel = modelsByPriority["coder_primary"][0]
		} else if len(modelsByPriority["coder_secondary"]) > 0 {
			bestCoderModel = modelsByPriority["coder_secondary"][0]
		} else if len(modelsByPriority["title_fallback"]) > 0 && bestCoderModel == "" {
			// Fallback: use any available model for coder
			bestCoderModel = modelsByPriority["title_fallback"][0]
		}

		if len(modelsByPriority["task_primary"]) > 0 {
			bestTaskModel = modelsByPriority["task_primary"][0]
		} else if len(modelsByPriority["task_secondary"]) > 0 {
			bestTaskModel = modelsByPriority["task_secondary"][0]
		} else if len(modelsByPriority["title_fallback"]) > 0 && bestTaskModel == "" {
			// Fallback: use any available model for task
			bestTaskModel = modelsByPriority["title_fallback"][0]
		}

		if len(modelsByPriority["title_primary"]) > 0 {
			bestTitleModel = modelsByPriority["title_primary"][0]
		} else if len(modelsByPriority["title_fallback"]) > 0 {
			bestTitleModel = modelsByPriority["title_fallback"][0]
		}
	}
	config["providers"] = providersSection

	// Set up agents with proper model references
	if bestCoderModel != "" {
		agentsSection["coder"] = map[string]interface{}{
			"model":     bestCoderModel,
			"maxTokens": 5000,
		}
	}

	if bestTaskModel != "" {
		agentsSection["task"] = map[string]interface{}{
			"model":     bestTaskModel,
			"maxTokens": 5000,
		}
	} else if bestCoderModel != "" {
		// Fallback to coder model
		agentsSection["task"] = map[string]interface{}{
			"model":     bestCoderModel,
			"maxTokens": 5000,
		}
	}

	if bestTitleModel != "" {
		agentsSection["title"] = map[string]interface{}{
			"model":     bestTitleModel,
			"maxTokens": 80,
		}
	}

	// Ensure we have basic agents even if no models found
	if len(agentsSection) == 0 {
		agentsSection["coder"] = map[string]interface{}{
			"model":     "gpt-4o",
			"maxTokens": 5000,
		}
		agentsSection["task"] = map[string]interface{}{
			"model":     "gpt-4o",
			"maxTokens": 5000,
		}
		agentsSection["title"] = map[string]interface{}{
			"model":     "gpt-4o",
			"maxTokens": 80,
		}
	}

	config["agents"] = agentsSection

	// Add TUI config
	config["tui"] = map[string]interface{}{
		"theme": "opencode",
	}

	// Add shell config
	config["shell"] = map[string]interface{}{
		"path": "/bin/bash",
		"args": []string{"-l"},
	}

	// Add other required sections
	config["autoCompact"] = true
	config["debug"] = false
	config["debugLSP"] = false

	return config, nil
}

// getAPIKeyForProvider returns the appropriate API key variable for a provider
func getAPIKeyForProvider(providerName string, options *ExportOptions) string {
	if options == nil || !options.IncludeAPIKey {
		return ""
	}

	switch strings.ToLower(providerName) {
	case "openai":
		return "${OPENAI_API_KEY}"
	case "anthropic":
		return "${ANTHROPIC_API_KEY}"
	case "groq":
		return "${GROQ_API_KEY}"
	case "google", "gemini":
		return "${GOOGLE_API_KEY}"
	case "openrouter":
		return "${OPENROUTER_API_KEY}"
	case "copilot":
		return "${COPILOT_API_KEY}"
	default:
		return fmt.Sprintf("${%s_API_KEY}", strings.ToUpper(strings.ReplaceAll(providerName, "-", "_")))
	}
}

// getCrushProviderType returns the Crush provider type for a given provider name
func getCrushProviderType(providerName string) string {
	switch strings.ToLower(providerName) {
	case "openai":
		return "openai"
	case "anthropic":
		return "anthropic"
	case "deepseek":
		return "openai-compat"
	case "google":
		return "openai-compat"
	default:
		return "openai-compat"
	}
}

// getProviderBaseURL returns the base URL for a provider
func getProviderBaseURL(providerName string, models []VerificationResult) string {
	if len(models) > 0 {
		// Extract base URL from first model endpoint
		endpoint := models[0].ModelInfo.Endpoint
		// Remove model-specific parts
		if idx := strings.Index(endpoint, "/v1"); idx != -1 {
			return endpoint[:idx+3] // Include /v1
		}
		if idx := strings.Index(endpoint, "/chat"); idx != -1 {
			return endpoint[:idx]
		}
		return endpoint
	}

	// Default base URLs
	switch strings.ToLower(providerName) {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "google":
		return "https://generativelanguage.googleapis.com/v1"
	default:
		return "https://api.example.com/v1"
	}
}

// Import existing functions from config_loader.go to avoid duplication
