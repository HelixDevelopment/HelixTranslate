package llm

import (
	"context"
	"fmt"
)

// HyperbolicClient implements LLMClient for Hyperbolic API.
type HyperbolicClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewHyperbolicClient creates a new Hyperbolic client.
func NewHyperbolicClient(config TranslationConfig) (*HyperbolicClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("hyperbolic API key is required")
	}
	return &HyperbolicClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.hyperbolic.xyz/v1"),
	}, nil
}

// Translate performs translation via Hyperbolic.
func (c *HyperbolicClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("hyperbolic translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *HyperbolicClient) GetProviderName() string {
	return "hyperbolic"
}
