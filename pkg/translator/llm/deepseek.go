package llm

import (
	"digital.vasic.translator/pkg/translator"
	"fmt"
)

// DeepSeekClient implements DeepSeek API client (uses OpenAI-compatible API)
type DeepSeekClient struct {
	*OpenAIClient
}

// NewDeepSeekClient creates a new DeepSeek client
func NewDeepSeekClient(config translator.TranslationConfig) (*DeepSeekClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}

	// DeepSeek uses OpenAI-compatible API
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com/v1"
	}

	if config.Model == "" {
		config.Model = "deepseek-chat"
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
