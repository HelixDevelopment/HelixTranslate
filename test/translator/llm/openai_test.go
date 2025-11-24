package llm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// MockLLMClient implements LLMClient interface for testing
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

// TestOpenAIClientCreation tests OpenAI client creation
func TestOpenAIClientCreation(t *testing.T) {
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:    "openai",
		Model:      "gpt-4",
		SourceLang:  "en",
		TargetLang:  "es",
		BaseURL:     "https://api.openai.com/v1",
		Options: map[string]interface{}{
			"temperature": 0.3,
			"max_tokens":  4000,
		},
	}

	// Test client creation
	client, err := llm.NewOpenAIClient(config)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Test provider name
	assert.Equal(t, "openai", client.GetProviderName())
}

// TestOpenAIClientCreationWithoutAPIKey tests error handling for missing API key
func TestOpenAIClientCreationWithoutAPIKey(t *testing.T) {
	config := translator.TranslationConfig{
		Provider:   "openai",
		Model:      "gpt-4",
		SourceLang: "en",
		TargetLang: "es",
	}

	// Should fail with missing API key
	client, err := llm.NewOpenAIClient(config)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key is required")
}

// TestOpenAIConfiguration tests various OpenAI configurations
func TestOpenAIConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		config   translator.TranslationConfig
		provider string
	}{
		{
			name: "GPT-3.5",
			config: translator.TranslationConfig{
				APIKey:     "test-api-key",
				Provider:   "openai",
				Model:      "gpt-3.5-turbo",
				SourceLang: "en",
				TargetLang: "es",
			},
			provider: "openai",
		},
		{
			name: "GPT-4",
			config: translator.TranslationConfig{
				APIKey:     "test-api-key",
				Provider:   "openai",
				Model:      "gpt-4",
				SourceLang: "en",
				TargetLang: "es",
			},
			provider: "openai",
		},
		{
			name: "GPT-4-Turbo",
			config: translator.TranslationConfig{
				APIKey:     "test-api-key",
				Provider:   "openai",
				Model:      "gpt-4-turbo",
				SourceLang: "en",
				TargetLang: "es",
			},
			provider: "openai",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := llm.NewOpenAIClient(tc.config)
			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tc.provider, client.GetProviderName())
		})
	}
}

// TestLLMTranslatorCreation tests LLM translator creation with OpenAI
func TestLLMTranslatorCreation(t *testing.T) {
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:   "openai",
		Model:      "gpt-4",
		SourceLang: "en",
		TargetLang: "es",
	}

	trans, err := llm.NewLLMTranslator(config)
	assert.NoError(t, err)
	assert.NotNil(t, trans)
	assert.Equal(t, "llm-openai", trans.GetName())
}

// TestLLMTranslatorUnsupportedProvider tests error handling for unsupported providers
func TestLLMTranslatorUnsupportedProvider(t *testing.T) {
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:   "unsupported",
		Model:      "test-model",
		SourceLang: "en",
		TargetLang: "es",
	}

	trans, err := llm.NewLLMTranslator(config)
	assert.Error(t, err)
	assert.Nil(t, trans)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}

// TestLLMTranslatorWithMock tests LLM translator with mock client
func TestLLMTranslatorWithMock(t *testing.T) {
	// This test is for future integration when we can inject mock clients
	// For now, we'll test the basic structure
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:   "openai",
		Model:      "gpt-4",
		SourceLang: "en",
		TargetLang: "es",
	}

	trans, err := llm.NewLLMTranslator(config)
	if err != nil {
		t.Skipf("Skipping mock test due to creation error: %v", err)
		return
	}

	// Test basic methods
	assert.NotNil(t, trans)
	assert.Equal(t, "llm-openai", trans.GetName())

	// Test stats
	stats := trans.GetStats()
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Translated)
	assert.Equal(t, 0, stats.Cached)
	assert.Equal(t, 0, stats.Errors)
}

// TestLLMTranslatorTranslateWithProgress tests progress reporting
func TestLLMTranslatorTranslateWithProgress(t *testing.T) {
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:   "openai",
		Model:      "gpt-4",
		SourceLang: "en",
		TargetLang: "es",
	}

	trans, err := llm.NewLLMTranslator(config)
	if err != nil {
		t.Skipf("Skipping progress test due to creation error: %v", err)
		return
	}

	ctx := context.Background()
	text := "Hello world"
	contextStr := "Simple greeting"
	var eventBus *events.EventBus = nil // We'll use nil to avoid event bus setup
	sessionID := "test-session"

	// This test verifies the function structure works
	// Actual API call would fail with test API key
	result, err := trans.TranslateWithProgress(ctx, text, contextStr, eventBus, sessionID)
	
	// With nil event bus and test API key, we expect an error
	assert.Error(t, err)
	assert.Empty(t, result)
}

// BenchmarkOpenAIClientCreation benchmarks OpenAI client creation
func BenchmarkOpenAIClientCreation(b *testing.B) {
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:   "openai",
		Model:      "gpt-4",
		SourceLang: "en",
		TargetLang: "es",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := llm.NewOpenAIClient(config)
		if err != nil {
			b.Skipf("Skipping benchmark due to creation failure: %v", err)
		}
	}
}

// BenchmarkLLMTranslatorCreation benchmarks LLM translator creation
func BenchmarkLLMTranslatorCreation(b *testing.B) {
	config := translator.TranslationConfig{
		APIKey:     "test-api-key",
		Provider:   "openai",
		Model:      "gpt-4",
		SourceLang: "en",
		TargetLang: "es",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := llm.NewLLMTranslator(config)
		if err != nil {
			b.Skipf("Skipping benchmark due to creation failure: %v", err)
		}
	}
}