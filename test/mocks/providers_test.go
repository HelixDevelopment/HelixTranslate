package mocks

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestMockLLMProvider(t *testing.T) {
	provider := &MockLLMProvider{}

	// Set up mock expectations
	provider.On("Translate",
		mock.Anything,
		"Hello", "en", "es").
		Return("Hola", nil)

	provider.On("GetProvider").Return("mock")

	// Test mock functionality
	result, err := provider.Translate(nil, "Hello", "en", "es")
	assert.NoError(t, err)
	assert.Equal(t, "Hola", result)

	provName := provider.GetProvider()
	assert.Equal(t, "mock", provName)

	provider.AssertExpectations(t)
}
