package llm

import (
	"fmt"
	"strings"
)

// NLPCloudClient implements LLMClient for NLPCloud API (OpenAI-compatible).
type NLPCloudClient struct {
	*OpenAIClient
}

// NewNLPCloudClient creates a new NLPCloud client.
func NewNLPCloudClient(config TranslationConfig) (*NLPCloudClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("nlpcloud API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.nlpcloud.io/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("nlpcloud model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderNLPCloud]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for NLPCloud. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for NLPCloud. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &NLPCloudClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *NLPCloudClient) GetProviderName() string {
	return "nlpcloud"
}
