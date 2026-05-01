package llm

import (
	"context"
	"fmt"
)

// CloudflareClient implements LLMClient for Cloudflare Workers AI.
type CloudflareClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewCloudflareClient creates a new Cloudflare client.
func NewCloudflareClient(config TranslationConfig) (*CloudflareClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("cloudflare API key is required")
	}
	return &CloudflareClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.cloudflare.com/client/v4/ai"),
	}, nil
}

// Translate performs translation via Cloudflare.
func (c *CloudflareClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("cloudflare translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *CloudflareClient) GetProviderName() string {
	return "cloudflare"
}
