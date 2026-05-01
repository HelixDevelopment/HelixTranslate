package llm

import (
	"context"
	"fmt"
)

// NovitaClient implements LLMClient for Novita AI API.
type NovitaClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewNovitaClient creates a new Novita client.
func NewNovitaClient(config TranslationConfig) (*NovitaClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("novita API key is required")
	}
	return &NovitaClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.novita.ai/v3"),
	}, nil
}

// Translate performs translation via Novita.
func (c *NovitaClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("novita translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *NovitaClient) GetProviderName() string {
	return "novita"
}
