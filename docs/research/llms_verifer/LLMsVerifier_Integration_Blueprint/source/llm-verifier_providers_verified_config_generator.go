package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.llmsverifier/logging"
)

// VerifiedConfigGenerator generates configuration files with only verified models
type VerifiedConfigGenerator struct {
	enhancedService *EnhancedModelProviderService
	logger          *logging.Logger
	outputDir       string
}

// VerifiedConfig represents a configuration with only verified models
type VerifiedConfig struct {
	GeneratedAt       time.Time                          `json:"generated_at"`
	VerificationEnabled bool                            `json:"verification_enabled"`
	StrictMode        bool                              `json:"strict_mode"`
	TotalModels       int                               `json:"total_models"`
	VerifiedModels    int                               `json:"verified_models"`
	Providers         map[string]VerifiedProviderConfig `json:"providers"`
}

// VerifiedProviderConfig represents a provider with verified models
type VerifiedProviderConfig struct {
	ProviderID      string                 `json:"provider_id"`
	ProviderName    string                 `json:"provider_name"`
	BaseURL         string                 `json:"base_url"`
	ModelCount      int                    `json:"model_count"`
	VerifiedModels  []VerifiedModelConfig  `json:"verified_models"`
}

// VerifiedModelConfig represents a verified model configuration
type VerifiedModelConfig struct {
	ModelID               string                 `json:"model_id"`
	ModelName             string                 `json:"model_name"`
	DisplayName           string                 `json:"display_name"`
	Features              map[string]interface{} `json:"features"`
	MaxTokens             int                    `json:"max_tokens"`
	CostPer1MInput        float64                `json:"cost_per_1m_input"`
	CostPer1MOutput       float64                `json:"cost_per_1m_output"`
	VerificationScore     float64                `json:"verification_score"`
	CanSeeCode            bool                   `json:"can_see_code"`
	AffirmativeResponse   bool                   `json:"affirmative_response"`
	LastVerifiedAt        time.Time              `json:"last_verified_at"`
}

// NewVerifiedConfigGenerator creates a new verified configuration generator
func NewVerifiedConfigGenerator(enhancedService *EnhancedModelProviderService, logger *logging.Logger, outputDir string) *VerifiedConfigGenerator {
	return &VerifiedConfigGenerator{
		enhancedService: enhancedService,
		logger:          logger,
		outputDir:       outputDir,
	}
}

// GenerateVerifiedConfig generates a configuration with only verified models
func (vcg *VerifiedConfigGenerator) GenerateVerifiedConfig() (*VerifiedConfig, error) {
	vcg.logger.Info("Generating verified configuration with mandatory model verification", nil)
	
	// Get all models with verification
	allModels, err := vcg.enhancedService.GetAllModelsWithVerification(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get verified models: %w", err)
	}
	
	if len(allModels) == 0 {
		vcg.logger.Warning("No verified models found", nil)
		return &VerifiedConfig{
			GeneratedAt:         time.Now(),
			VerificationEnabled: vcg.enhancedService.verificationConfig.Enabled,
			StrictMode:          vcg.enhancedService.verificationConfig.StrictMode,
			TotalModels:         0,
			VerifiedModels:      0,
			Providers:           make(map[string]VerifiedProviderConfig),
		}, nil
	}
	
	// Build verified configuration
	verifiedConfig := &VerifiedConfig{
		GeneratedAt:         time.Now(),
		VerificationEnabled: vcg.enhancedService.verificationConfig.Enabled,
		StrictMode:          vcg.enhancedService.verificationConfig.StrictMode,
		TotalModels:         0,
		VerifiedModels:      0,
		Providers:           make(map[string]VerifiedProviderConfig),
	}
	
	verificationResults := vcg.enhancedService.GetVerificationResults()
	
	for providerID, models := range allModels {
		if len(models) == 0 {
			continue
		}
		
		providerClient, exists := vcg.enhancedService.ModelProviderService.providerClients[providerID]
		if !exists {
			vcg.logger.Warning(fmt.Sprintf("Provider client not found for %s", providerID), nil)
			continue
		}
		
		verifiedProviderConfig := VerifiedProviderConfig{
			ProviderID:     providerID,
			ProviderName:   models[0].ProviderName,
			BaseURL:        providerClient.BaseURL,
			ModelCount:     len(models),
			VerifiedModels: make([]VerifiedModelConfig, 0, len(models)),
		}
		
		for _, model := range models {
			verificationKey := fmt.Sprintf("%s:%s", providerID, model.ID)
			verificationResult, exists := verificationResults[verificationKey]
			
			if !exists {
				vcg.logger.Warning(fmt.Sprintf("No verification result for model %s", model.ID), nil)
				continue
			}
			
			verifiedModelConfig := VerifiedModelConfig{
				ModelID:             model.ID,
				ModelName:           model.Name,
				DisplayName:         model.DisplayName,
				Features:            model.Features,
				MaxTokens:           model.MaxTokens,
				CostPer1MInput:      model.CostPer1MInput,
				CostPer1MOutput:     model.CostPer1MOutput,
				VerificationScore:   verificationResult.VerificationScore,
				CanSeeCode:          verificationResult.CanSeeCode,
				AffirmativeResponse: verificationResult.AffirmativeResponse,
				LastVerifiedAt:      verificationResult.LastVerifiedAt,
			}
			
			verifiedProviderConfig.VerifiedModels = append(verifiedProviderConfig.VerifiedModels, verifiedModelConfig)
			verifiedConfig.TotalModels++
			verifiedConfig.VerifiedModels++
		}
		
		if len(verifiedProviderConfig.VerifiedModels) > 0 {
			verifiedConfig.Providers[providerID] = verifiedProviderConfig
		}
	}
	
	vcg.logger.Info(fmt.Sprintf("Generated verified configuration with %d verified models across %d providers", 
		verifiedConfig.VerifiedModels, len(verifiedConfig.Providers)), nil)
	
	return verifiedConfig, nil
}

// SaveVerifiedConfig saves the verified configuration to files
func (vcg *VerifiedConfigGenerator) SaveVerifiedConfig(config *VerifiedConfig, baseFilename string) error {
	if err := os.MkdirAll(vcg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Save full configuration
	fullConfigPath := filepath.Join(vcg.outputDir, fmt.Sprintf("%s_verified_config.json", baseFilename))
	if err := vcg.saveConfigToFile(config, fullConfigPath); err != nil {
		return fmt.Errorf("failed to save full config: %w", err)
	}
	
	// Save redacted configuration (without sensitive data)
	redactedConfig := vcg.createRedactedConfig(config)
	redactedConfigPath := filepath.Join(vcg.outputDir, fmt.Sprintf("%s_verified_config_redacted.json", baseFilename))
	if err := vcg.saveConfigToFile(redactedConfig, redactedConfigPath); err != nil {
		return fmt.Errorf("failed to save redacted config: %w", err)
	}
	
	// Save verification summary
	summaryPath := filepath.Join(vcg.outputDir, fmt.Sprintf("%s_verification_summary.json", baseFilename))
	if err := vcg.saveVerificationSummary(config, summaryPath); err != nil {
		return fmt.Errorf("failed to save verification summary: %w", err)
	}
	
	vcg.logger.Info(fmt.Sprintf("Saved verified configuration files to %s", vcg.outputDir), map[string]interface{}{
		"full_config":    fullConfigPath,
		"redacted_config": redactedConfigPath,
		"summary":        summaryPath,
	})
	
	return nil
}

// GenerateAndSaveVerifiedConfig generates and saves the verified configuration in one step
func (vcg *VerifiedConfigGenerator) GenerateAndSaveVerifiedConfig(baseFilename string) error {
	config, err := vcg.GenerateVerifiedConfig()
	if err != nil {
		return fmt.Errorf("failed to generate verified config: %w", err)
	}
	
	if err := vcg.SaveVerifiedConfig(config, baseFilename); err != nil {
		return fmt.Errorf("failed to save verified config: %w", err)
	}
	
	return nil
}

// Helper methods

func (vcg *VerifiedConfigGenerator) saveConfigToFile(config interface{}, filepath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

func (vcg *VerifiedConfigGenerator) createRedactedConfig(config *VerifiedConfig) *VerifiedConfig {
	// Create a deep copy and redact sensitive information
	redacted := *config
	redacted.Providers = make(map[string]VerifiedProviderConfig)
	
	for providerID, provider := range config.Providers {
		redactedProvider := provider
		// Redact base URL and other potentially sensitive info
		redactedProvider.BaseURL = "REDACTED"
		redacted.Providers[providerID] = redactedProvider
	}
	
	return &redacted
}

func (vcg *VerifiedConfigGenerator) saveVerificationSummary(config *VerifiedConfig, filepath string) error {
	summary := map[string]interface{}{
		"generated_at":         config.GeneratedAt,
		"verification_enabled": config.VerificationEnabled,
		"strict_mode":          config.StrictMode,
		"total_models":         config.TotalModels,
		"verified_models":      config.VerifiedModels,
		"verification_rate":    float64(config.VerifiedModels) / float64(config.TotalModels) * 100,
		"provider_count":       len(config.Providers),
		"providers":            make(map[string]interface{}),
	}
	
	for providerID, provider := range config.Providers {
		summary["providers"].(map[string]interface{})[providerID] = map[string]interface{}{
			"model_count":    provider.ModelCount,
			"verified_count": len(provider.VerifiedModels),
		}
	}
	
	return vcg.saveConfigToFile(summary, filepath)
}

// GetVerificationStatistics returns statistics about the verification process
func (vcg *VerifiedConfigGenerator) GetVerificationStatistics() (map[string]interface{}, error) {
	config, err := vcg.GenerateVerifiedConfig()
	if err != nil {
		return nil, err
	}
	
	statistics := map[string]interface{}{
		"total_models_scanned": config.TotalModels,
		"verified_models":      config.VerifiedModels,
		"verification_rate":    float64(config.VerifiedModels) / float64(config.TotalModels) * 100,
		"providers_with_models": len(config.Providers),
		"verification_enabled": config.VerificationEnabled,
		"strict_mode":          config.StrictMode,
		"generated_at":         config.GeneratedAt,
	}
	
	// Add provider breakdown
	providerStats := make(map[string]interface{})
	for providerID, provider := range config.Providers {
		providerStats[providerID] = map[string]interface{}{
			"total_models":   provider.ModelCount,
			"verified_count": len(provider.VerifiedModels),
			"success_rate":   float64(len(provider.VerifiedModels)) / float64(provider.ModelCount) * 100,
		}
	}
	statistics["provider_breakdown"] = providerStats
	
	return statistics, nil
}