package llm

import (
	"context"
	"fmt"
)

// PublicAIClient implements LLMClient for PublicAI API.
type PublicAIClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewPublicAIClient creates a new PublicAI client.
func NewPublicAIClient(config TranslationConfig) (*PublicAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("publicai API key is required")
	}
	return &PublicAIClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.publicai.io/v1"),
	}, nil
}

// Translate performs translation via PublicAI.
func (c *PublicAIClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("publicai translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *PublicAIClient) GetProviderName() string {
	return "publicai"
}
