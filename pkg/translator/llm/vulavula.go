package llm

import (
	"context"
	"fmt"
)

// VulavulaClient implements LLMClient for Vulavula API.
type VulavulaClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewVulavulaClient creates a new Vulavula client.
func NewVulavulaClient(config TranslationConfig) (*VulavulaClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("vulavula API key is required")
	}
	return &VulavulaClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.vulavula.io/v1"),
	}, nil
}

// Translate performs translation via Vulavula.
func (c *VulavulaClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("vulavula translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *VulavulaClient) GetProviderName() string {
	return "vulavula"
}
