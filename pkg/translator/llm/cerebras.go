package llm

import (
	"context"
	"fmt"
)

// CerebrasClient implements LLMClient for Cerebras API.
type CerebrasClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewCerebrasClient creates a new Cerebras client.
func NewCerebrasClient(config TranslationConfig) (*CerebrasClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("cerebras API key is required")
	}
	return &CerebrasClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.cerebras.ai/v1"),
	}, nil
}

// Translate performs translation via Cerebras.
func (c *CerebrasClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("cerebras translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *CerebrasClient) GetProviderName() string {
	return "cerebras"
}
