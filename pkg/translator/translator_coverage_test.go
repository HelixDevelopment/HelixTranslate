package translator

import (
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"github.com/stretchr/testify/assert"
)

// TestBaseTranslator_Cache tests cache functionality
func TestBaseTranslator_Cache(t *testing.T) {
	config := TranslationConfig{
		SourceLang: "en",
		TargetLang: "ru",
		Provider:   "openai",
	}

	bt := NewBaseTranslator(config)

	t.Run("CheckCache returns empty string for non-existent key", func(t *testing.T) {
		result, found := bt.CheckCache("hello")
		assert.Empty(t, result)
		assert.False(t, found)
	})

	t.Run("CheckCache returns cached translation", func(t *testing.T) {
		// Add to cache directly
		bt.AddToCache("hello", "привет")

		result, found := bt.CheckCache("hello")
		assert.Equal(t, "привет", result)
		assert.True(t, found)

		// Check stats
		stats := bt.GetStats()
		assert.Equal(t, 1, stats.Cached)
	})

	t.Run("AddToCache updates cache", func(t *testing.T) {
		bt.AddToCache("world", "мир")

		result, found := bt.CheckCache("world")
		assert.Equal(t, "мир", result)
		assert.True(t, found)
	})

	t.Run("Cache works with empty strings", func(t *testing.T) {
		bt.AddToCache("", "")

		result, found := bt.CheckCache("")
		assert.Equal(t, "", result)
		assert.True(t, found)
	})

	t.Run("Cache overwrites existing entries", func(t *testing.T) {
		bt.AddToCache("hello", "привет")
		bt.AddToCache("hello", "здравствуй")

		result, found := bt.CheckCache("hello")
		assert.Equal(t, "здравствуй", result)
		assert.True(t, found)
	})
}

// TestBaseTranslator_UpdateStats tests statistics tracking
func TestBaseTranslator_UpdateStats(t *testing.T) {
	config := TranslationConfig{}
	bt := NewBaseTranslator(config)

	t.Run("Initial stats are zero", func(t *testing.T) {
		stats := bt.GetStats()
		assert.Equal(t, 0, stats.Total)
		assert.Equal(t, 0, stats.Translated)
		assert.Equal(t, 0, stats.Cached)
		assert.Equal(t, 0, stats.Errors)
	})

	t.Run("UpdateStats with success increments total and translated", func(t *testing.T) {
		bt.UpdateStats(true)

		stats := bt.GetStats()
		assert.Equal(t, 1, stats.Total)
		assert.Equal(t, 1, stats.Translated)
		assert.Equal(t, 0, stats.Errors)
	})

	t.Run("UpdateStats with failure increments total and errors", func(t *testing.T) {
		bt.UpdateStats(false)

		stats := bt.GetStats()
		assert.Equal(t, 2, stats.Total)
		assert.Equal(t, 1, stats.Translated)
		assert.Equal(t, 1, stats.Errors)
	})

	t.Run("Multiple updates accumulate correctly", func(t *testing.T) {
		bt.UpdateStats(true)
		bt.UpdateStats(true)
		bt.UpdateStats(false)

		stats := bt.GetStats()
		assert.Equal(t, 5, stats.Total)
		assert.Equal(t, 3, stats.Translated)
		assert.Equal(t, 2, stats.Errors)
	})
}

// TestEmitProgress tests progress event emission
func TestEmitProgress(t *testing.T) {
	t.Run("EmitProgress with nil event bus does nothing", func(t *testing.T) {
		// Should not panic
		assert.NotPanics(t, func() {
			EmitProgress(nil, "session1", "test message", nil)
		})
	})

	t.Run("EmitProgress publishes event", func(t *testing.T) {
		eventBus := events.NewEventBus()
		receivedEvent := false

		// Subscribe to events
		eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
			if event.Type == events.EventTranslationProgress {
				receivedEvent = true
				assert.Equal(t, "session1", event.SessionID)
				assert.Equal(t, "test message", event.Message)
			}
		})

		EmitProgress(eventBus, "session1", "test message", map[string]interface{}{"key": "value"})

		// Give time for async processing
		time.Sleep(10 * time.Millisecond)
		assert.True(t, receivedEvent)
	})

	t.Run("EmitProgress with complex data", func(t *testing.T) {
		eventBus := events.NewEventBus()
		var receivedData map[string]interface{}

		eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
			if event.Type == events.EventTranslationProgress {
				receivedData = event.Data
			}
		})

		testData := map[string]interface{}{
			"progress": 50.5,
			"chapter":  3,
			"total":    10,
		}

		EmitProgress(eventBus, "session2", "complex message", testData)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 50.5, receivedData["progress"])
		assert.Equal(t, 3, receivedData["chapter"])
		assert.Equal(t, 10, receivedData["total"])
	})
}

// TestEmitError tests error event emission
func TestEmitError(t *testing.T) {
	t.Run("EmitError with nil event bus does nothing", func(t *testing.T) {
		// Should not panic
		assert.NotPanics(t, func() {
			EmitError(nil, "session1", "test error", assert.AnError)
		})
	})

	t.Run("EmitError publishes error event", func(t *testing.T) {
		eventBus := events.NewEventBus()
		receivedEvent := false
		testError := assert.AnError

		eventBus.Subscribe(events.EventTranslationError, func(event events.Event) {
			if event.Type == events.EventTranslationError {
				receivedEvent = true
				assert.Equal(t, "session1", event.SessionID)
				assert.Equal(t, "test error", event.Message)
			}
		})

		EmitError(eventBus, "session1", "test error", testError)
		time.Sleep(10 * time.Millisecond)

		assert.True(t, receivedEvent)
	})

	t.Run("EmitError includes error in data", func(t *testing.T) {
		eventBus := events.NewEventBus()
		var receivedData map[string]interface{}
		testError := assert.AnError

		eventBus.Subscribe(events.EventTranslationError, func(event events.Event) {
			if event.Type == events.EventTranslationError {
				receivedData = event.Data
			}
		})

		EmitError(eventBus, "session2", "error message", testError)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, testError.Error(), receivedData["error"])
	})

	t.Run("EmitError with nil error", func(t *testing.T) {
		eventBus := events.NewEventBus()
		var receivedData map[string]interface{}

		eventBus.Subscribe(events.EventTranslationError, func(event events.Event) {
			if event.Type == events.EventTranslationError {
				receivedData = event.Data
			}
		})

		EmitError(eventBus, "session3", "nil error", nil)
		time.Sleep(10 * time.Millisecond)

		// Should handle nil error gracefully
		assert.NotNil(t, receivedData["error"])
	})
}

// TestTranslationConfig tests translation configuration
func TestTranslationConfig(t *testing.T) {
	t.Run("Valid config creates successfully", func(t *testing.T) {
		config := TranslationConfig{
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
			Options:     map[string]interface{}{"option1": "value1"},
		}

		assert.Equal(t, "en", config.SourceLang)
		assert.Equal(t, "ru", config.TargetLang)
		assert.Equal(t, "openai", config.Provider)
		assert.Equal(t, "gpt-4", config.Model)
		assert.Equal(t, 0.7, config.Temperature)
		assert.Equal(t, 1000, config.MaxTokens)
		assert.Equal(t, 30*time.Second, config.Timeout)
		assert.Equal(t, "test-key", config.APIKey)
		assert.Equal(t, "https://api.openai.com", config.BaseURL)
		assert.Equal(t, "latin", config.Script)
		assert.Equal(t, "value1", config.Options["option1"])
	})

	t.Run("Config with zero values", func(t *testing.T) {
		config := TranslationConfig{}

		assert.Empty(t, config.SourceLang)
		assert.Empty(t, config.TargetLang)
		assert.Empty(t, config.Provider)
		assert.Empty(t, config.Model)
		assert.Equal(t, 0.0, config.Temperature)
		assert.Equal(t, 0, config.MaxTokens)
		assert.Equal(t, time.Duration(0), config.Timeout)
		assert.Empty(t, config.APIKey)
		assert.Empty(t, config.BaseURL)
		assert.Empty(t, config.Script)
		assert.Nil(t, config.Options)
	})

	t.Run("Config with nil options map", func(t *testing.T) {
		config := TranslationConfig{
			SourceLang: "en",
			TargetLang: "ru",
			Options:    nil,
		}

		assert.Equal(t, "en", config.SourceLang)
		assert.Equal(t, "ru", config.TargetLang)
		assert.Nil(t, config.Options)
	})
}

// TestUniversalTranslator tests universal translator functionality
func TestUniversalTranslator(t *testing.T) {
	mockTranslator := &MockTranslator{}
	mockDetector := NewMockDetector()

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	t.Run("NewUniversalTranslator creates correctly", func(t *testing.T) {
		ut := NewUniversalTranslator(mockTranslator, mockDetector, sourceLang, targetLang)

		assert.Equal(t, mockTranslator, ut.translator)
		assert.Equal(t, mockDetector, ut.langDetector)
		assert.Equal(t, sourceLang, ut.sourceLanguage)
		assert.Equal(t, targetLang, ut.targetLanguage)
	})

	t.Run("NewUniversalTranslator with nil detector", func(t *testing.T) {
		ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)

		assert.Equal(t, mockTranslator, ut.translator)
		assert.Nil(t, ut.langDetector)
		assert.Equal(t, sourceLang, ut.sourceLanguage)
		assert.Equal(t, targetLang, ut.targetLanguage)
	})
}

// TestErrors tests error variables
func TestErrors(t *testing.T) {
	t.Run("ErrNoLLMInstances", func(t *testing.T) {
		assert.Equal(t, "no LLM instances available", ErrNoLLMInstances.Error())
	})

	t.Run("ErrInvalidProvider", func(t *testing.T) {
		assert.Equal(t, "invalid translation provider", ErrInvalidProvider.Error())
	})
}

// BenchmarkBaseTranslator_Cache benchmarks cache operations
func BenchmarkBaseTranslator_Cache(b *testing.B) {
	config := TranslationConfig{}
	bt := NewBaseTranslator(config)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := "test-key-" + string(rune(i))
		value := "test-value-" + string(rune(i))

		bt.AddToCache(key, value)
		bt.CheckCache(key)
	}
}

// BenchmarkEmitProgress benchmarks progress emission
func BenchmarkEmitProgress(b *testing.B) {
	eventBus := events.NewEventBus()
	sessionID := "bench-session"
	message := "benchmark message"
	data := map[string]interface{}{"key": "value"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		EmitProgress(eventBus, sessionID, message, data)
	}
}
