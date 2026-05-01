package llm

import (
	"context"
	"fmt"
)

// TogetherAIClient implements LLMClient for Together AI API.
type TogetherAIClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewTogetherAIClient creates a new Together AI client.
func NewTogetherAIClient(config TranslationConfig) (*TogetherAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("togetherai API key is required")
	}
	return &TogetherAIClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.together.xyz/v1"),
	}, nil
}

// Translate performs translation via Together AI.
func (c *TogetherAIClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("togetherai translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *TogetherAIClient) GetProviderName() string {
	return "togetherai"
}
