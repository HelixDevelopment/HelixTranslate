package llm

import (
	"context"
	"fmt"
)

// SarvamClient implements LLMClient for Sarvam AI API.
type SarvamClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewSarvamClient creates a new Sarvam client.
func NewSarvamClient(config TranslationConfig) (*SarvamClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("sarvam API key is required")
	}
	return &SarvamClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.sarvam.ai/v1"),
	}, nil
}

// Translate performs translation via Sarvam.
func (c *SarvamClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("sarvam translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *SarvamClient) GetProviderName() string {
	return "sarvam"
}
