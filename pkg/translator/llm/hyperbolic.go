package llm

import (
	"fmt"
	"strings"
)

// HyperbolicClient implements LLMClient for Hyperbolic API (OpenAI-compatible).
type HyperbolicClient struct {
	*OpenAIClient
}

// NewHyperbolicClient creates a new Hyperbolic client.
func NewHyperbolicClient(config TranslationConfig) (*HyperbolicClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("hyperbolic API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.hyperbolic.xyz/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("hyperbolic model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderHyperbolic]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Hyperbolic. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Hyperbolic. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &HyperbolicClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *HyperbolicClient) GetProviderName() string {
	return "hyperbolic"
}
