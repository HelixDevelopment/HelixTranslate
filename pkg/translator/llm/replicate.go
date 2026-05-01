package llm

import (
	"context"
	"fmt"
)

// ReplicateClient implements LLMClient for Replicate API.
type ReplicateClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewReplicateClient creates a new Replicate client.
func NewReplicateClient(config TranslationConfig) (*ReplicateClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("replicate API key is required")
	}
	return &ReplicateClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.replicate.com/v1"),
	}, nil
}

// Translate performs translation via Replicate.
func (c *ReplicateClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("replicate translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *ReplicateClient) GetProviderName() string {
	return "replicate"
}
