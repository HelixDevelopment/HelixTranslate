package llm

import (
	"fmt"
	"strings"
)

// ModalClient implements LLMClient for Modal API (OpenAI-compatible).
type ModalClient struct {
	*OpenAIClient
}

// NewModalClient creates a new Modal client.
func NewModalClient(config TranslationConfig) (*ModalClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("modal API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.modal.com/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("modal model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderModal]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Modal. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for Modal. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &ModalClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *ModalClient) GetProviderName() string {
	return "modal"
}
