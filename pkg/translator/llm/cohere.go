package llm

import (
	"context"
	"fmt"
)

// CohereClient implements LLMClient for Cohere API.
type CohereClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewCohereClient creates a new Cohere client.
func NewCohereClient(config TranslationConfig) (*CohereClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("cohere API key is required")
	}
	return &CohereClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.cohere.com"),
	}, nil
}

// Translate performs translation via Cohere.
func (c *CohereClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("cohere translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *CohereClient) GetProviderName() string {
	return "cohere"
}
