package llm

import (
	"fmt"
	"strings"
)

// XAIClient implements LLMClient for XAI API (OpenAI-compatible).
type XAIClient struct {
	*OpenAIClient
}

// NewXAIClient creates a new XAI client.
func NewXAIClient(config TranslationConfig) (*XAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("xai API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.x.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("xai model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderXAI]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for XAI. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for XAI. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &XAIClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *XAIClient) GetProviderName() string {
	return "xai"
}
