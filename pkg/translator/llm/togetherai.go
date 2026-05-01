package llm

import (
	"fmt"
	"strings"
)

// TogetherAIClient implements LLMClient for TogetherAI API (OpenAI-compatible).
type TogetherAIClient struct {
	*OpenAIClient
}

// NewTogetherAIClient creates a new TogetherAI client.
func NewTogetherAIClient(config TranslationConfig) (*TogetherAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("togetherai API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.together.xyz/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("togetherai model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderTogetherAI]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for TogetherAI. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for TogetherAI. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &TogetherAIClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *TogetherAIClient) GetProviderName() string {
	return "togetherai"
}
