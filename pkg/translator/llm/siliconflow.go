package llm

import (
	"fmt"
	"strings"
)

// SiliconFlowClient implements LLMClient for SiliconFlow API (OpenAI-compatible).
type SiliconFlowClient struct {
	*OpenAIClient
}

// NewSiliconFlowClient creates a new SiliconFlow client.
func NewSiliconFlowClient(config TranslationConfig) (*SiliconFlowClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("siliconflow API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.siliconflow.cn/v1"
	}

	if config.Model == "" {
		return nil, fmt.Errorf("siliconflow model is required")
	}

	if strings.TrimSpace(config.Model) == "" {
		return nil, fmt.Errorf("model cannot be empty or whitespace")
	}

	validModels := ValidModels[ProviderSiliconFlow]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for SiliconFlow. Valid models: %v",
			config.Model, validModels)
	}

	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for SiliconFlow. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	openaiClient, err := NewOpenAIClient(config)
	if err != nil {
		return nil, err
	}

	return &SiliconFlowClient{OpenAIClient: openaiClient}, nil
}

// GetProviderName returns the provider name.
func (c *SiliconFlowClient) GetProviderName() string {
	return "siliconflow"
}
