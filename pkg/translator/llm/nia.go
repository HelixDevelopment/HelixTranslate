package llm

import (
	"fmt"
	"strings"
)

// NIAClient implements LLMClient for NIA API (OpenAI-compatible).
type NIAClient struct {
	*OpenAIClient
}

// NewNIAClient creates a new NIA client.
func NewNIAClient(config TranslationConfig) (*NIAClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("nia API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.nia.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("nia model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderNIA]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for NIA. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for NIA. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &NIAClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *NIAClient) GetProviderName() string {
	return "nia"
}
