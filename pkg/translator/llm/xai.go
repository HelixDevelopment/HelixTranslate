package llm

import (
	"context"
	"fmt"
)

// XAIClient implements LLMClient for xAI (Grok) API.
type XAIClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewXAIClient creates a new xAI client.
func NewXAIClient(config TranslationConfig) (*XAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("xai API key is required")
	}
	return &XAIClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.x.ai/v1"),
	}, nil
}

// Translate performs translation via xAI.
func (c *XAIClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("xai translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *XAIClient) GetProviderName() string {
	return "xai"
}
