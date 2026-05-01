package llm

import (
	"fmt"
	"strings"
)

// MistralClient implements LLMClient for Mistral API (OpenAI-compatible).
type MistralClient struct {
	*OpenAIClient
}

// NewMistralClient creates a new Mistral client.
func NewMistralClient(config TranslationConfig) (*MistralClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("mistral API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.mistral.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("mistral model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderMistral]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Mistral. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Mistral. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &MistralClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *MistralClient) GetProviderName() string {
	return "mistral"
}
