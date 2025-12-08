package llm

import (
	"context"
	"fmt"
	"errors"
)

// MockLLMClient implements LLMClient interface for testing
type MockLLMClient struct {
	responses      map[string]string
	shouldFail     bool
	sizeError      bool
	callCount      int
	maxCallsToFail int
}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses:      make(map[string]string),
		shouldFail:     false,
		sizeError:      false,
		callCount:      0,
		maxCallsToFail: 0,
	}
}

// SetResponse sets a predefined response for given text
func (m *MockLLMClient) SetResponse(text, response string) {
	m.responses[text] = response
}

// SetFailure sets the mock client to fail on specific number of calls
func (m *MockLLMClient) SetFailure(shouldFail bool, maxCallsToFail int) {
	m.shouldFail = shouldFail
	m.maxCallsToFail = maxCallsToFail
}

// SetSizeError sets whether to return size limit errors
func (m *MockLLMClient) SetSizeError(sizeError bool) {
	m.sizeError = sizeError
}

// Translate implements LLMClient interface
func (m *MockLLMClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	m.callCount++
	
	if m.shouldFail && m.callCount <= m.maxCallsToFail {
		if m.sizeError {
			return "", errors.New("max_tokens limit exceeded")
		}
		return "", errors.New("mock API error")
	}
	
	if response, ok := m.responses[text]; ok {
		return response, nil
	}
	return fmt.Sprintf("translated: %s", text), nil
}

// GetProviderName implements LLMClient interface
func (m *MockLLMClient) GetProviderName() string {
	return "mock"
}