package llm

import (
	"fmt"
	"strings"
)

// UpstageClient implements LLMClient for Upstage API (OpenAI-compatible).
type UpstageClient struct {
	*OpenAIClient
}

// NewUpstageClient creates a new Upstage client.
func NewUpstageClient(config TranslationConfig) (*UpstageClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("upstage API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.upstage.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("upstage model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderUpstage]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Upstage. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Upstage. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &UpstageClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *UpstageClient) GetProviderName() string {
	return "upstage"
}
