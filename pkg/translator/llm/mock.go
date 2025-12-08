package llm

import (
	"context"
	"fmt"
)

// MockLLMClient implements LLMClient interface for testing
type MockLLMClient struct {
	responses map[string]string
}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),
	}
}

// SetResponse sets a predefined response for given text
func (m *MockLLMClient) SetResponse(text, response string) {
	m.responses[text] = response
}

// Translate implements LLMClient interface
func (m *MockLLMClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	if response, ok := m.responses[text]; ok {
		return response, nil
	}
	return fmt.Sprintf("translated: %s", text), nil
}

// GetProviderName implements LLMClient interface
func (m *MockLLMClient) GetProviderName() string {
	return "mock"
}