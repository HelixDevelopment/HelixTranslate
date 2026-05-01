package llm

import (
	"fmt"
	"strings"
)

// CerebrasClient implements LLMClient for Cerebras API (OpenAI-compatible).
type CerebrasClient struct {
	*OpenAIClient
}

// NewCerebrasClient creates a new Cerebras client.
func NewCerebrasClient(config TranslationConfig) (*CerebrasClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("cerebras API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.cerebras.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("cerebras model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderCerebras]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Cerebras. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Cerebras. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &CerebrasClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *CerebrasClient) GetProviderName() string {
	return "cerebras"
}
