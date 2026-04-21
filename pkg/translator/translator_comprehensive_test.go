package translator

import (
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"github.com/stretchr/testify/assert"
)

// TestNewUniversalTranslator tests the creation of a universal translator
func TestNewUniversalTranslator(t *testing.T) {
	mockTranslator := new(MockTranslator)
	mockDetector := NewMockDetector()

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	ut := NewUniversalTranslator(mockTranslator, mockDetector, sourceLang, targetLang)

	assert.NotNil(t, ut)
	assert.Equal(t, mockTranslator, ut.translator)
	assert.Equal(t, mockDetector, ut.langDetector)
	assert.Equal(t, sourceLang, ut.sourceLanguage)
	assert.Equal(t, targetLang, ut.targetLanguage)
}

// TestBaseTranslatorTests tests the BaseTranslator functionality
func TestBaseTranslatorTests(t *testing.T) {
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
		Options:     make(map[string]interface{}),
	}

	bt := NewBaseTranslator(config)
	assert.NotNil(t, bt)
	assert.Equal(t, config, bt.config)
	assert.Equal(t, TranslationStats{}, bt.stats)
	assert.NotNil(t, bt.cache)

	// Test GetStats
	stats := bt.GetStats()
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Translated)
	assert.Equal(t, 0, stats.Cached)
	assert.Equal(t, 0, stats.Errors)

	// Test CheckCache when cache is empty
	translated, found := bt.CheckCache("test")
	assert.Equal(t, "", translated)
	assert.False(t, found)

	// Test AddToCache and CheckCache
	bt.AddToCache("test", "translation")
	translated, found = bt.CheckCache("test")
	assert.Equal(t, "translation", translated)
	assert.True(t, found)

	// Test UpdateStats with success
	bt.UpdateStats(true)
	stats = bt.GetStats()
	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Translated)
	assert.Equal(t, 0, stats.Errors)

	// Test UpdateStats with error
	bt.UpdateStats(false)
	stats = bt.GetStats()
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 1, stats.Translated)
	assert.Equal(t, 1, stats.Errors)
}

// TestTranslationEvents tests event emission functionality
func TestTranslationEvents(t *testing.T) {
	eventBus := events.NewEventBus()
	sessionID := "test-session"

	// Capture published events using a channel for synchronization
	capturedEvents := make(chan events.Event, 10)
	eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
		capturedEvents <- event
	})

	// Test EmitProgress
	progressData := map[string]interface{}{"progress": 50}
	EmitProgress(eventBus, sessionID, "Test progress", progressData)

	// Wait for the event to be processed (with timeout)
	select {
	case event := <-capturedEvents:
		assert.Equal(t, events.EventTranslationProgress, event.Type)
		assert.Equal(t, "Test progress", event.Message)
		assert.Equal(t, sessionID, event.SessionID)
		assert.Equal(t, progressData, event.Data)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected event was not received")
	}

	// Test EmitError - subscribe to error events
	errorEvents := make(chan events.Event, 10)
	eventBus.Subscribe(events.EventTranslationError, func(event events.Event) {
		errorEvents <- event
	})

	testError := assert.AnError
	EmitError(eventBus, sessionID, "Test error", testError)

	// Wait for the error event
	select {
	case event := <-errorEvents:
		assert.Equal(t, events.EventTranslationError, event.Type)
		assert.Equal(t, "Test error", event.Message)
		assert.Equal(t, sessionID, event.SessionID)
		assert.Equal(t, testError.Error(), event.Data["error"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected error event was not received")
	}
}

// TestEmitProgressWithNilEventBus tests that EmitProgress doesn't panic with nil event bus
func TestEmitProgressWithNilEventBus(t *testing.T) {
	// This should not panic
	EmitProgress(nil, "session", "message", nil)
}

// TestEmitErrorWithNilEventBus tests that EmitError doesn't panic with nil event bus
func TestEmitErrorWithNilEventBus(t *testing.T) {
	// This should not panic
	EmitError(nil, "session", "message", assert.AnError)
}

// TestCreatePromptForLanguages tests the prompt creation function
func TestCreatePromptForLanguages(t *testing.T) {
	text := "Hello world"
	sourceLang := "English"
	targetLang := "Russian"
	context := "Book title"

	prompt := CreatePromptForLanguages(text, sourceLang, targetLang, context)

	assert.Contains(t, prompt, "You are a professional translator")
	assert.Contains(t, prompt, "English to Russian translation")
	assert.Contains(t, prompt, text)
	assert.Contains(t, prompt, context)
	assert.Contains(t, prompt, "Hello world")
	assert.Contains(t, prompt, "Russian translation:")
}

// TestCreatePromptForLanguagesWithEmptyContext tests prompt creation with empty context
func TestCreatePromptForLanguagesWithEmptyContext(t *testing.T) {
	text := "Hello world"
	sourceLang := "English"
	targetLang := "Russian"
	context := ""

	prompt := CreatePromptForLanguages(text, sourceLang, targetLang, context)

	assert.Contains(t, prompt, "Literary text") // Default context
	assert.Contains(t, prompt, text)
}

// TestUniversalTranslatorGetters tests the getter methods
func TestUniversalTranslatorGetters(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	ut := NewUniversalTranslator(nil, nil, sourceLang, targetLang)

	assert.Equal(t, sourceLang, ut.GetSourceLanguage())
	assert.Equal(t, targetLang, ut.GetTargetLanguage())
}

// TestTranslationConfigStructure tests the translation config structure
func TestTranslationConfigStructure(t *testing.T) {
	options := map[string]interface{}{
		"option1": "value1",
		"option2": 42,
	}

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
		Options:     options,
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
	assert.Equal(t, options, config.Options)
}

// TestUniversalTranslatorSimpleWorkflow tests a simplified translation workflow
func TestUniversalTranslatorSimpleWorkflow(t *testing.T) {
	mockTranslator := new(MockTranslator)
	mockDetector := NewMockDetector()

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	ut := NewUniversalTranslator(mockTranslator, mockDetector, sourceLang, targetLang)

	// Test that the universal translator was created correctly
	assert.NotNil(t, ut)
	assert.Equal(t, sourceLang, ut.sourceLanguage)
	assert.Equal(t, targetLang, ut.targetLanguage)
	assert.Equal(t, mockTranslator, ut.translator)
	assert.Equal(t, mockDetector, ut.langDetector)
}

// TestErrorDefinitions tests that error variables are properly defined
func TestErrorDefinitions(t *testing.T) {
	assert.NotNil(t, ErrNoLLMInstances)
	assert.NotNil(t, ErrInvalidProvider)
	assert.Equal(t, "no LLM instances available", ErrNoLLMInstances.Error())
	assert.Equal(t, "invalid translation provider", ErrInvalidProvider.Error())
}
