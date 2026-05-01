package llm

import (
	"fmt"
	"strings"
)

// PublicAIClient implements LLMClient for PublicAI API (OpenAI-compatible).
type PublicAIClient struct {
	*OpenAIClient
}

// NewPublicAIClient creates a new PublicAI client.
func NewPublicAIClient(config TranslationConfig) (*PublicAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("publicai API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.publicai.com/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("publicai model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderPublicAI]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for PublicAI. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for PublicAI. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &PublicAIClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *PublicAIClient) GetProviderName() string {
	return "publicai"
}
