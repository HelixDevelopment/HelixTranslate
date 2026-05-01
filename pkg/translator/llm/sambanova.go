package llm

import (
	"fmt"
	"strings"
)

// SambaNovaClient implements LLMClient for SambaNova API (OpenAI-compatible).
type SambaNovaClient struct {
	*OpenAIClient
}

// NewSambaNovaClient creates a new SambaNova client.
func NewSambaNovaClient(config TranslationConfig) (*SambaNovaClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("sambanova API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.sambanova.ai/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("sambanova model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderSambaNova]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for SambaNova. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for SambaNova. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &SambaNovaClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *SambaNovaClient) GetProviderName() string {
	return "sambanova"
}
