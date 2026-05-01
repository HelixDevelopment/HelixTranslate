package llm

import (
	"context"
	"fmt"
)

// UpstageClient implements LLMClient for Upstage AI API.
type UpstageClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewUpstageClient creates a new Upstage client.
func NewUpstageClient(config TranslationConfig) (*UpstageClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("upstage API key is required")
	}
	return &UpstageClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.upstage.ai/v1"),
	}, nil
}

// Translate performs translation via Upstage.
func (c *UpstageClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("upstage translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *UpstageClient) GetProviderName() string {
	return "upstage"
}
