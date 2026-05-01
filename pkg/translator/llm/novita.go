package llm

import (
	"fmt"
	"strings"
)

// NovitaClient implements LLMClient for Novita API (OpenAI-compatible).
type NovitaClient struct {
	*OpenAIClient
}

// NewNovitaClient creates a new Novita client.
func NewNovitaClient(config TranslationConfig) (*NovitaClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("novita API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.novita.ai/v3/openai"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("novita model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderNovita]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Novita. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Novita. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &NovitaClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *NovitaClient) GetProviderName() string {
	return "novita"
}
