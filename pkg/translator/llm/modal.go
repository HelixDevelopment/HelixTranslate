package llm

import (
	"context"
	"fmt"
)

// ModalClient implements LLMClient for Modal API.
type ModalClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewModalClient creates a new Modal client.
func NewModalClient(config TranslationConfig) (*ModalClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("modal API key is required")
	}
	return &ModalClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.modal.com/v1"),
	}, nil
}

// Translate performs translation via Modal.
func (c *ModalClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("modal translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *ModalClient) GetProviderName() string {
	return "modal"
}
