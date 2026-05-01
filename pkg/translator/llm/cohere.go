package llm

import (
	"fmt"
	"strings"
)

// CohereClient implements LLMClient for Cohere API (OpenAI-compatible).
type CohereClient struct {
	*OpenAIClient
}

// NewCohereClient creates a new Cohere client.
func NewCohereClient(config TranslationConfig) (*CohereClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("cohere API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.cohere.com/compatibility/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("cohere model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderCohere]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Cohere. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Cohere. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &CohereClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *CohereClient) GetProviderName() string {
	return "cohere"
}
