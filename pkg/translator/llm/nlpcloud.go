package llm

import (
	"context"
	"fmt"
)

// NLPCloudClient implements LLMClient for NLP Cloud API.
type NLPCloudClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewNLPCloudClient creates a new NLP Cloud client.
func NewNLPCloudClient(config TranslationConfig) (*NLPCloudClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("nlpcloud API key is required")
	}
	return &NLPCloudClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.nlpcloud.io/v1"),
	}, nil
}

// Translate performs translation via NLP Cloud.
func (c *NLPCloudClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("nlpcloud translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *NLPCloudClient) GetProviderName() string {
	return "nlpcloud"
}
