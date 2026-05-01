package llm

import (
	"fmt"
	"strings"
)

// VulavulaClient implements LLMClient for Vulavula API (OpenAI-compatible).
type VulavulaClient struct {
	*OpenAIClient
}

// NewVulavulaClient creates a new Vulavula client.
func NewVulavulaClient(config TranslationConfig) (*VulavulaClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("vulavula API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.vulavula.com/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("vulavula model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderVulavula]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Vulavula. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Vulavula. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &VulavulaClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *VulavulaClient) GetProviderName() string {
	return "vulavula"
}
