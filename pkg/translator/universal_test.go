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

// TestUniversalTranslatorBasicFunctionality tests basic universal translator functionality
func TestUniversalTranslatorBasicFunctionality(t *testing.T) {
	// Create mock LLM client
	mockLLM := new(MockLLMClient)
	mockLLM.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Привет мир", nil)
	mockLLM.On("GetProviderName").Return("openai")

	// Create universal translator using LLM translator
	uniTrans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockLLM,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test basic translation
	result, err := uniTrans.Translate(ctx, "Hello world", "en", "ru")
	require.NoError(t, err)
	assert.Equal(t, "Привет мир", result)

	// Test provider detection
	provider := uniTrans.GetProvider()
	assert.Equal(t, "openai", provider)

	// Verify mock was called
	mockLLM.AssertExpectations(t)
}

// TestUniversalTranslatorProviderSwitching tests provider switching
func TestUniversalTranslatorProviderSwitching(t *testing.T) {
	// Create mock LLM clients
	mockOpenAI := new(MockLLMClient)
	mockOpenAI.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Привет мир", nil)
	mockOpenAI.On("GetProviderName").Return("openai")

	mockDeepSeek := new(MockLLMClient)
	mockDeepSeek.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Здраво свете", nil)
	mockDeepSeek.On("GetProviderName").Return("deepseek")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create OpenAI translator
	openaiTrans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderOpenAI,
		Model:    "gpt-4",
		Client:   mockOpenAI,
	})
	require.NoError(t, err)

	// Create DeepSeek translator
	deepseekTrans, err := llm.NewLLMTranslator(llm.Config{
		Provider: llm.ProviderDeepSeek,
		Model:    "deepseek-chat",
		Client:   mockDeepSeek,
	})
	require.NoError(t, err)

	// Test translation with OpenAI
	result1, err := openaiTrans.Translate(ctx, "Hello world", "en", "ru")
	require.NoError(t, err)
	assert.Equal(t, "Привет мир", result1)

	// Test translation with DeepSeek
	result2, err := deepseekTrans.Translate(ctx, "Hello world", "en", "sr")
	require.NoError(t, err)
	assert.Equal(t, "Здраво свете", result2)

	// Verify mocks were called
	mockOpenAI.AssertExpectations(t)
	mockDeepSeek.AssertExpectations(t)
}

// TestUniversalTranslatorMultipleLanguages tests translation between multiple language pairs
func TestUniversalTranslatorMultipleLanguages(t *testing.T) {
	// Create mock LLM client
	mockLLM := new(MockLLMClient)
	
	// Setup expectations for different language pairs
	testCases := []struct {
		text      string
		source    string
		target    string
		expected  string
	}{
		{"Hello world", "en", "ru", "Привет мир"},
		{"How are you", "en", "sr", "Како си"},
		{"Good morning", "en", "de", "Guten Morgen"},
		{"Thank you", "en", "fr", "Merci"},
	}

	for _, tc := range testCases {
		mockLLM.On("Translate", mock.Anything, tc.text, mock.AnythingOfType("string")).Return(tc.expected, nil)
	}
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

	// Test each language pair
	for _, tc := range testCases {
		result, err := trans.Translate(ctx, tc.text, tc.source, tc.target)
		require.NoError(t, err, "Translation failed for %s->%s", tc.source, tc.target)
		assert.Equal(t, tc.expected, result, "Translation mismatch for %s->%s", tc.source, tc.target)
	}

	// Verify all mocks were called
	mockLLM.AssertExpectations(t)
}

// TestUniversalTranslatorErrorHandling tests error handling
func TestUniversalTranslatorErrorHandling(t *testing.T) {
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

	// Test error handling
	result, err := trans.Translate(ctx, "Hello world", "en", "ru")
	assert.Error(t, err)
	assert.Empty(t, result)

	// Verify mock was called
	mockLLM.AssertExpectations(t)
}

// TestUniversalTranslatorContextCancellation tests context cancellation
func TestUniversalTranslatorContextCancellation(t *testing.T) {
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

// BenchmarkUniversalTranslatorTranslation benchmarks universal translator translation
func BenchmarkUniversalTranslatorTranslation(b *testing.B) {
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

// BenchmarkUniversalTranslatorProviderSwitching benchmarks provider switching
func BenchmarkUniversalTranslatorProviderSwitching(b *testing.B) {
	// Create mock LLM clients
	mockOpenAI := new(MockLLMClient)
	mockOpenAI.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Привет мир", nil)
	mockOpenAI.On("GetProviderName").Return("openai")

	mockDeepSeek := new(MockLLMClient)
	mockDeepSeek.On("Translate", mock.Anything, "Hello world", mock.AnythingOfType("string")).Return("Здраво свете", nil)
	mockDeepSeek.On("GetProviderName").Return("deepseek")

	providers := []llm.Provider{
		llm.ProviderOpenAI,
		llm.ProviderDeepSeek,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		provider := providers[i%len(providers)]
		
		var mockLLM *MockLLMClient
		if provider == llm.ProviderOpenAI {
			mockLLM = mockOpenAI
		} else {
			mockLLM = mockDeepSeek
		}
		
		trans, err := llm.NewLLMTranslator(llm.Config{
			Provider: provider,
			Model:    "test-model",
			Client:   mockLLM,
		})
		if err != nil {
			b.Fatal(err)
		}
		
		_, err = trans.Translate(ctx, "Hello world", "en", "ru")
		if err != nil {
			b.Fatal(err)
		}
	}
}