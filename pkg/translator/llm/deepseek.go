package llm

import (
	"fmt"
	"strings"
)

// DeepSeekClient implements DeepSeek API client (uses OpenAI-compatible API)
type DeepSeekClient struct {
	*OpenAIClient
}

// NewDeepSeekClient creates a new DeepSeek client
func NewDeepSeekClient(config TranslationConfig) (*DeepSeekClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}

	// Validate provider
	if config.Provider != "deepseek" && config.Provider != "" {
		return nil, fmt.Errorf("invalid provider for DeepSeek client: %s", config.Provider)
	}

	// DeepSeek uses OpenAI-compatible API
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("DeepSeek model is required")
	} else {
		// Validate model if provided
		if strings.TrimSpace(config.Model) == "" {
			return nil, fmt.Errorf("model cannot be empty or whitespace")
		}
		validModels := ValidModels[ProviderDeepSeek]
		modelValid := false
		for _, validModel := range validModels {
			if config.Model == validModel {
				modelValid = true
				break
			}
		}
		if !modelValid {
			return nil, fmt.Errorf("model '%s' is not valid for DeepSeek. Valid models: %v",
				config.Model, validModels)
		}
	}

	// Validate temperature if provided
	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for DeepSeek. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &DeepSeekClient{
		OpenAIClient: openaiClient,
	}, nil
}

// GetProviderName returns the provider name
func (c *DeepSeekClient) GetProviderName() string {
	return "deepseek"
}
