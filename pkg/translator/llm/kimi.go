package llm

import (
	"fmt"
	"strings"
)

// KimiClient implements LLMClient for Kimi API (OpenAI-compatible).
type KimiClient struct {
	*OpenAIClient
}

// NewKimiClient creates a new Kimi client.
func NewKimiClient(config TranslationConfig) (*KimiClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("kimi API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.moonshot.cn/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("kimi model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderKimi]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Kimi. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Kimi. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &KimiClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *KimiClient) GetProviderName() string {
	return "kimi"
}
