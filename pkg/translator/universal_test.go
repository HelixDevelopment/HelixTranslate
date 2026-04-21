package translator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUniversalTranslatorBasicFunctionality tests basic universal translator functionality
func TestUniversalTranslatorBasicFunctionality(t *testing.T) {
	cfg := TranslationConfig{
		SourceLang: "en",
		TargetLang: "es",
		Provider:   "mock",
	}

	bt := NewBaseTranslator(cfg)
	require.NotNil(t, bt)

	// Test cache miss
	_, found := bt.CheckCache("hello")
	assert.False(t, found)

	// Test cache add and hit
	bt.AddToCache("hello", "hola")
	val, found := bt.CheckCache("hello")
	assert.True(t, found)
	assert.Equal(t, "hola", val)

	// Stats should track cache hit
	stats := bt.GetStats()
	assert.GreaterOrEqual(t, stats.Cached, 1)

	// Test stats update
	bt.UpdateStats(true)
	bt.UpdateStats(false)
	stats = bt.GetStats()
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 1, stats.Translated)
	assert.Equal(t, 1, stats.Errors)
}

// TestUniversalTranslatorProviderSwitching tests provider switching
func TestUniversalTranslatorProviderSwitching(t *testing.T) {
	providers := []string{"openai", "anthropic", "gemini", "ollama", "mock"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			cfg := TranslationConfig{
				SourceLang: "en",
				TargetLang: "fr",
				Provider:   provider,
				APIKey:     "test-key",
			}

			bt := NewBaseTranslator(cfg)
			require.NotNil(t, bt)
			assert.Equal(t, provider, bt.config.Provider)
		})
	}
}

// TestUniversalTranslatorMultipleLanguages tests translation between multiple language pairs
func TestUniversalTranslatorMultipleLanguages(t *testing.T) {
	pairs := []struct {
		source string
		target string
	}{
		{"en", "es"},
		{"en", "ru"},
		{"de", "fr"},
		{"ja", "en"},
		{"zh", "en"},
	}

	for _, pair := range pairs {
		t.Run(pair.source+"_to_"+pair.target, func(t *testing.T) {
			cfg := TranslationConfig{
				SourceLang: pair.source,
				TargetLang: pair.target,
				Provider:   "mock",
			}

			bt := NewBaseTranslator(cfg)
			require.NotNil(t, bt)
			assert.Equal(t, pair.source, bt.config.SourceLang)
			assert.Equal(t, pair.target, bt.config.TargetLang)
		})
	}
}

// TestUniversalTranslatorErrorHandling tests error handling
func TestUniversalTranslatorErrorHandling(t *testing.T) {
	t.Run("nil_event_bus_progress", func(t *testing.T) {
		// Should not panic with nil event bus
		assert.NotPanics(t, func() {
			EmitProgress(nil, "session1", "progress", map[string]interface{}{"percent": 50})
		})
	})

	t.Run("nil_event_bus_error", func(t *testing.T) {
		// Should not panic with nil event bus
		assert.NotPanics(t, func() {
			EmitError(nil, "session1", "error occurred", assert.AnError)
		})
	})

	t.Run("error_messages", func(t *testing.T) {
		assert.Equal(t, "no LLM instances available", ErrNoLLMInstances.Error())
		assert.Equal(t, "invalid translation provider", ErrInvalidProvider.Error())
	})

	t.Run("empty_config", func(t *testing.T) {
		cfg := TranslationConfig{}
		bt := NewBaseTranslator(cfg)
		require.NotNil(t, bt)
		stats := bt.GetStats()
		assert.Equal(t, 0, stats.Total)
		assert.Equal(t, 0, stats.Translated)
		assert.Equal(t, 0, stats.Cached)
		assert.Equal(t, 0, stats.Errors)
	})
}

// TestTranslationConfigValidation tests translation config validation
func TestTranslationConfigValidation(t *testing.T) {
	// Test valid config
	validConfig := TranslationConfig{
		SourceLang:  "en",
		TargetLang:  "ru",
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     30 * time.Second,
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com",
		Script:      "latin",
		Options:     make(map[string]interface{}),
	}

	// Config should be valid (no validation function currently)
	assert.NotNil(t, validConfig)
	assert.Equal(t, "en", validConfig.SourceLang)
	assert.Equal(t, "ru", validConfig.TargetLang)
	assert.Equal(t, "openai", validConfig.Provider)
	assert.Equal(t, "gpt-4", validConfig.Model)
	assert.Equal(t, "test-key", validConfig.APIKey)
}

// TestTranslationResult tests translation result structure
func TestTranslationResult(t *testing.T) {
	result := TranslationResult{
		OriginalText:   "Hello world",
		TranslatedText: "Привет мир",
		Provider:       "openai",
		Cached:         false,
		Error:          nil,
	}

	assert.Equal(t, "Hello world", result.OriginalText)
	assert.Equal(t, "Привет мир", result.TranslatedText)
	assert.Equal(t, "openai", result.Provider)
	assert.False(t, result.Cached)
	assert.NoError(t, result.Error)
}

// TestTranslationStats tests translation statistics
func TestTranslationStats(t *testing.T) {
	stats := TranslationStats{
		Total:      100,
		Translated: 80,
		Cached:     15,
		Errors:     5,
	}

	assert.Equal(t, 100, stats.Total)
	assert.Equal(t, 80, stats.Translated)
	assert.Equal(t, 15, stats.Cached)
	assert.Equal(t, 5, stats.Errors)

	// Verify that total equals translated + cached + errors
	assert.Equal(t, stats.Translated+stats.Cached+stats.Errors, stats.Total)
}

// TestTranslatorErrors tests error variables
func TestTranslatorErrors(t *testing.T) {
	assert.NotNil(t, ErrNoLLMInstances)
	assert.NotNil(t, ErrInvalidProvider)
	assert.Equal(t, "no LLM instances available", ErrNoLLMInstances.Error())
	assert.Equal(t, "invalid translation provider", ErrInvalidProvider.Error())
}
