package llm

import (
	"context"
	"fmt"
)

// MistralClient implements LLMClient for Mistral AI API.
type MistralClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewMistralClient creates a new Mistral client.
func NewMistralClient(config TranslationConfig) (*MistralClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("mistral API key is required")
	}
	return &MistralClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.mistral.ai/v1"),
	}, nil
}

// Translate performs translation via Mistral.
func (c *MistralClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("mistral translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *MistralClient) GetProviderName() string {
	return "mistral"
}
