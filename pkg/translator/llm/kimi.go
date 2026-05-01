package llm

import (
	"context"
	"fmt"
)

// KimiClient implements LLMClient for Moonshot AI (Kimi) API.
type KimiClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewKimiClient creates a new Kimi client.
func NewKimiClient(config TranslationConfig) (*KimiClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("kimi API key is required")
	}
	return &KimiClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.moonshot.cn/v1"),
	}, nil
}

// Translate performs translation via Kimi.
func (c *KimiClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("kimi translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *KimiClient) GetProviderName() string {
	return "kimi"
}
