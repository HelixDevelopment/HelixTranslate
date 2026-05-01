package llm

import (
	"context"
	"fmt"
)

// SambaNovaClient implements LLMClient for SambaNova API.
type SambaNovaClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewSambaNovaClient creates a new SambaNova client.
func NewSambaNovaClient(config TranslationConfig) (*SambaNovaClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("sambanova API key is required")
	}
	return &SambaNovaClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.sambanova.ai/v1"),
	}, nil
}

// Translate performs translation via SambaNova.
func (c *SambaNovaClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("sambanova translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *SambaNovaClient) GetProviderName() string {
	return "sambanova"
}
