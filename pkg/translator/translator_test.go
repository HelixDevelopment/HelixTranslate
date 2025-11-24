package translator

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/translator/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLLMClient implements a mock LLM client for testing
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	args := m.Called(ctx, text, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockLLMClient) GetProviderName() string {
	args := m.Called()
	return args.String(0)
}

// TestTranslatorBasicFunctionality tests basic translation functionality
func TestTranslatorBasicFunctionality(t *testing.T) {
	// Create mock LLM client
	mockLLM := new(MockLLMClient)
	mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Привет мир", nil)
	mockLLM.On("GetProviderName").Return("mock_openai")

	// Create LLM translator with mock
	trans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockLLM,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test basic translation
	result, err := trans.Translate(ctx, "Hello world", "en", "ru")
	require.NoError(t, err)
	assert.Equal(t, "Привет мир", result)
	assert.Equal(t, "openai", trans.GetProvider())

	// Verify mock was called
	mockLLM.AssertExpectations(t)
}

// TestTranslatorErrorHandling tests error handling in translation
func TestTranslatorErrorHandling(t *testing.T) {
	// Create mock LLM client that returns error
	mockLLM := new(MockLLMClient)
	mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("", assert.AnError)
	mockLLM.On("GetProviderName").Return("openai")

	// Create LLM translator
	trans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockLLM,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test translation error
	result, err := trans.Translate(ctx, "Hello world", "en", "ru")
	assert.Error(t, err)
	assert.Empty(t, result)

	// Verify mock was called
	mockLLM.AssertExpectations(t)
}

// TestTranslatorContextCancellation tests context cancellation
func TestTranslatorContextCancellation(t *testing.T) {
	// Create mock LLM client
	mockLLM := new(MockLLMClient)
	mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("", context.Canceled)
	mockLLM.On("GetProviderName").Return("openai")

	// Create LLM translator
	trans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockLLM,
	})
	require.NoError(t, err)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()

	// Test context cancellation
	result, err := trans.Translate(ctx, "Hello world", "en", "ru")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	assert.Empty(t, result)
}

// TestTranslatorMultipleProviders tests translation with multiple providers
func TestTranslatorMultipleProviders(t *testing.T) {
	providers := []llm.Provider{
		llm.ProviderOpenAI,
		llm.ProviderAnthropic,
		llm.ProviderZhipu,
		llm.ProviderDeepSeek,
	}

	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			// Create mock LLM client
			mockLLM := new(MockLLMClient)
			mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Переведено", nil)
			mockLLM.On("GetProviderName").Return(string(provider))

			// Create LLM translator with specific provider
			trans, err := llm.NewLLMTranslator(llm.Config{
				Provider: provider,
				Model:    "test-model",
				Client:   mockLLM,
			})
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Test translation
			result, err := trans.Translate(ctx, "Hello world", "en", "ru")
			require.NoError(t, err)
			assert.Equal(t, "Переведено", result)
			assert.Equal(t, string(provider), trans.GetProvider())

			// Verify mock was called
			mockLLM.AssertExpectations(t)
		})
	}
}

// BenchmarkTranslatorSingleTranslation benchmarks single translation
func BenchmarkTranslatorSingleTranslation(b *testing.B) {
	// Create mock LLM client
	mockLLM := new(MockLLMClient)
	mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Привет мир", nil)
	mockLLM.On("GetProviderName").Return("openai")

	// Create LLM translator
	trans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockLLM,
	})
	require.NoError(b, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := trans.Translate(ctx, "Hello world", "en", "ru")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTranslatorConcurrentTranslation benchmarks concurrent translation
func BenchmarkTranslatorConcurrentTranslation(b *testing.B) {
	// Create mock LLM client
	mockLLM := new(MockLLMClient)
	mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Привет мир", nil)
	mockLLM.On("GetProviderName").Return("openai")

	// Create LLM translator
	trans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockLLM,
	})
	require.NoError(b, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := trans.Translate(ctx, "Hello world", "en", "ru")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}