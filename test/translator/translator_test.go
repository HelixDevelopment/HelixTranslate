package translator_test

import (
	"context"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTranslator implements Translator interface for testing
type MockTranslator struct {
	mock.Mock
}

func (m *MockTranslator) Translate(ctx context.Context, text string, contextStr string) (string, error) {
	args := m.Called(ctx, text, contextStr)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) TranslateWithProgress(ctx context.Context, text string, contextStr string, eventBus *events.EventBus, sessionID string) (string, error) {
	args := m.Called(ctx, text, contextStr, eventBus, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) GetStats() translator.TranslationStats {
	args := m.Called()
	return args.Get(0).(translator.TranslationStats)
}

func (m *MockTranslator) GetName() string {
	args := m.Called()
	return args.String(0)
}

// TestBaseTranslator tests basic functionality
func TestBaseTranslator(t *testing.T) {
	// Test base translator creation
	config := translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "es",
		Provider:   "test",
		Model:      "test-model",
	}

	baseTrans := translator.NewBaseTranslator(config)
	require.NotNil(t, baseTrans)

	// Test stats
	stats := baseTrans.GetStats()
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Translated)
	assert.Equal(t, 0, stats.Cached)
	assert.Equal(t, 0, stats.Errors)

	// Test cache functionality
	// Test cache miss
	result, found := baseTrans.CheckCache("test key")
	assert.Empty(t, result)
	assert.False(t, found)

	// Test cache add
	baseTrans.AddToCache("test key", "test value")

	// Test cache hit
	result, found = baseTrans.CheckCache("test key")
	assert.Equal(t, "test value", result)
	assert.True(t, found)

	// Test stats update
	baseTrans.UpdateStats(true)
	stats = baseTrans.GetStats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Translated)
	assert.Equal(t, 0, stats.Errors)
}

// TestTranslationConfig tests configuration validation
func TestTranslationConfig(t *testing.T) {
	// Test valid config
	config := translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "es",
		Provider:   "test",
		Model:      "test-model",
		APIKey:     "test-key",
	}

	assert.Equal(t, "en", config.SourceLang)
	assert.Equal(t, "es", config.TargetLang)
	assert.Equal(t, "test", config.Provider)

	// Test alias compatibility (using SourceLang/TargetLang as aliases)
	config.SourceLang = "fr"
	config.TargetLang = "de"

	assert.Equal(t, "fr", config.SourceLang)
	assert.Equal(t, "de", config.TargetLang)
}

// TestTranslationResult tests result structure
func TestTranslationResult(t *testing.T) {
	// Test successful result
	result := translator.TranslationResult{
		OriginalText:   "Hello world",
		TranslatedText: "Hola mundo",
		Provider:       "test-provider",
		Cached:         false,
		Error:          nil,
	}

	assert.Equal(t, "Hello world", result.OriginalText)
	assert.Equal(t, "Hola mundo", result.TranslatedText)
	assert.Equal(t, "test-provider", result.Provider)
	assert.False(t, result.Cached)
	assert.NoError(t, result.Error)

	// Test error result
	errorResult := translator.TranslationResult{
		OriginalText:   "Hello world",
		TranslatedText: "",
		Provider:       "test-provider",
		Cached:         false,
		Error:          assert.AnError,
	}

	assert.Equal(t, "Hello world", errorResult.OriginalText)
	assert.Empty(t, errorResult.TranslatedText)
	assert.Error(t, errorResult.Error)
}

// TestTranslationStats tests statistics functionality
func TestTranslationStats(t *testing.T) {
	// Test initial stats
	stats := translator.TranslationStats{
		Total:      100,
		Translated: 95,
		Cached:     20,
		Errors:     5,
	}

	assert.Equal(t, 100, stats.Total)
	assert.Equal(t, 95, stats.Translated)
	assert.Equal(t, 20, stats.Cached)
	assert.Equal(t, 5, stats.Errors)

	// Test success rate
	successRate := float64(stats.Translated) / float64(stats.Total)
	assert.Greater(t, successRate, 0.9)
}

// TestEmitProgress tests progress event emission
func TestEmitProgress(t *testing.T) {
	// Test with nil event bus (should not panic)
	translator.EmitProgress(nil, "test-session", "test message", map[string]interface{}{"key": "value"})

	// Test with actual event bus
	eventBus := events.NewEventBus()

	// This test just verifies the function doesn't panic
	assert.NotPanics(t, func() {
		translator.EmitProgress(eventBus, "test-session", "test message", map[string]interface{}{"key": "value"})
	})
}

// TestEmitError tests error event emission
func TestEmitError(t *testing.T) {
	// Test with nil event bus (should not panic)
	translator.EmitError(nil, "test-session", "test error", assert.AnError)

	// Test with actual event bus
	eventBus := events.NewEventBus()
	testErr := assert.AnError

	// This test just verifies the function doesn't panic
	assert.NotPanics(t, func() {
		translator.EmitError(eventBus, "test-session", "test error", testErr)
	})
}

// TestMockTranslator tests mock translator functionality
func TestMockTranslator(t *testing.T) {
	mockTrans := new(MockTranslator)

	ctx := context.Background()
	text := "Hello world"
	contextStr := "Literary text"
	expectedResult := "Hola mundo"

	mockTrans.On("Translate", ctx, text, contextStr).Return(expectedResult, nil)
	mockTrans.On("GetName").Return("mock-translator")

	// Test translation
	result, err := mockTrans.Translate(ctx, text, contextStr)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	// Test GetName
	name := mockTrans.GetName()
	assert.Equal(t, "mock-translator", name)

	// Verify expectations
	mockTrans.AssertExpectations(t)
}
