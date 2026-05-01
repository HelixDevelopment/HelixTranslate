package llm

import (
	"context"
	"fmt"
)

// NIAClient implements LLMClient for NIA API.
type NIAClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewNIAClient creates a new NIA client.
func NewNIAClient(config TranslationConfig) (*NIAClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("nia API key is required")
	}
	return &NIAClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.nia.ai/v1"),
	}, nil
}

// Translate performs translation via NIA.
func (c *NIAClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("nia translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *NIAClient) GetProviderName() string {
	return "nia"
}
