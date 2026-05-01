package llm

import (
	"fmt"
	"strings"
)

// SarvamClient implements LLMClient for Sarvam API (OpenAI-compatible).
type SarvamClient struct {
	*OpenAIClient
}

// NewSarvamClient creates a new Sarvam client.
func NewSarvamClient(config TranslationConfig) (*SarvamClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("sarvam API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.sarvam.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("sarvam model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderSarvam]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Sarvam. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Sarvam. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &SarvamClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *SarvamClient) GetProviderName() string {
	return "sarvam"
}
