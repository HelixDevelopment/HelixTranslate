package translator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
)

// TestUniversalTranslator_BasicFunctionality tests basic translator functionality
func TestUniversalTranslator_BasicFunctionalitySimple(t *testing.T) {
	mockTranslator := &MockTranslator{}
	
	// Set up mock expectations
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	t.Run("Basic book translation", func(t *testing.T) {
		ctx := context.Background()
		eventBus := events.NewEventBus()
		sessionID := "test-session"
		
		book := &ebook.Book{
			Metadata: ebook.Metadata{Title: "Test Book"},
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{Title: "Section 1", Content: "Content 1"},
					},
				},
			},
		}
		
		// Test that translation doesn't return errors
		err := ut.TranslateBook(ctx, book, eventBus, sessionID)
		assert.NoError(t, err)
		assert.Equal(t, "ru", book.Metadata.Language)
	})
}

// TestUniversalTranslator_ErrorHandling tests error scenarios
func TestUniversalTranslator_ErrorHandlingSimple(t *testing.T) {
	mockTranslator := &MockTranslator{}
	
	// Set up mock to return error
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", assert.AnError)
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	t.Run("Translation error handling", func(t *testing.T) {
		ctx := context.Background()
		eventBus := events.NewEventBus()
		sessionID := "test-session"
		
		book := &ebook.Book{
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{Title: "Section 1", Content: "Content 1"},
					},
				},
			},
		}
		
		// Test that translation errors are properly handled
		err := ut.TranslateBook(ctx, book, eventBus, sessionID)
		assert.Error(t, err)
	})
}

// TestUniversalTranslator_ContextCancellation tests context cancellation
func TestUniversalTranslator_ContextCancellationSimple(t *testing.T) {
	mockTranslator := &MockTranslator{}
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	t.Run("Cancelled context", func(t *testing.T) {
		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		// Set up mock to return error for cancelled context
		mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", context.Canceled)
		
		eventBus := events.NewEventBus()
		sessionID := "test-session"
		
		book := &ebook.Book{
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{Title: "Section 1", Content: "Content 1"},
					},
				},
			},
		}
		
		// Test that cancelled context returns error
		err := ut.TranslateBook(ctx, book, eventBus, sessionID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

// TestUniversalTranslator_ProgressTracking tests progress event emission
func TestUniversalTranslator_ProgressTrackingSimple(t *testing.T) {
	mockTranslator := &MockTranslator{}
	
	// Set up mock expectations
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	t.Run("Progress events emitted", func(t *testing.T) {
		ctx := context.Background()
		eventBus := events.NewEventBus()
		sessionID := "test-session"
		
		// Subscribe to progress events
		progressReceived := false
		eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
			progressReceived = true
		})
		
		book := &ebook.Book{
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{Title: "Section 1", Content: "Content 1"},
					},
				},
			},
		}
		
		err := ut.TranslateBook(ctx, book, eventBus, sessionID)
		assert.NoError(t, err)
		
		// Wait a bit for async events
		time.Sleep(time.Millisecond * 10)
		
		// Verify progress events were emitted
		assert.True(t, progressReceived, "Progress events should be emitted")
	})
}

// TestNewUniversalTranslatorSimple tests constructor
func TestNewUniversalTranslatorSimple(t *testing.T) {
	mockTranslator := &MockTranslator{}
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	require.NotNil(t, ut)
	assert.Equal(t, mockTranslator, ut.translator)
	assert.Nil(t, ut.langDetector)
	assert.Equal(t, sourceLang, ut.sourceLanguage)
	assert.Equal(t, targetLang, ut.targetLanguage)
}

// BenchmarkUniversalTranslator_NewSimple benchmarks constructor
func BenchmarkUniversalTranslator_NewSimple(b *testing.B) {
	mockTranslator := &MockTranslator{}
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	}
}

// BenchmarkUniversalTranslator_TranslateBookSimple benchmarks book translation
func BenchmarkUniversalTranslator_TranslateBookSimple(b *testing.B) {
	mockTranslator := &MockTranslator{}
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "bench-session"
	
	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "Test"},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Title: "Section 1", Content: "Content 1"},
				},
			},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ut.TranslateBook(ctx, book, eventBus, sessionID)
	}
}

// TestBaseTranslator_New tests NewBaseTranslator constructor
func TestBaseTranslator_New(t *testing.T) {
	config := TranslationConfig{
		Provider: "test",
		Model:    "test-model",
		APIKey:   "test-key",
	}
	
	bt := NewBaseTranslator(config)
	
	require.NotNil(t, bt)
	assert.Equal(t, config, bt.config)
	assert.NotNil(t, bt.stats)
	assert.NotNil(t, bt.cache)
}

// TestBaseTranslator_GetStats tests GetStats function
func TestBaseTranslator_GetStats(t *testing.T) {
	config := TranslationConfig{
		Provider: "test",
		Model:    "test-model",
		APIKey:   "test-key",
	}
	
	bt := NewBaseTranslator(config)
	
	// Initial stats should be empty
	stats := bt.GetStats()
	assert.Equal(t, 0, stats.Translated)
	assert.Equal(t, 0, stats.Cached)
}

// TestBaseTranslator_CheckCache tests CheckCache function
func TestBaseTranslator_CheckCache(t *testing.T) {
	config := TranslationConfig{
		Provider: "test",
		Model:    "test-model",
		APIKey:   "test-key",
	}
	
	bt := NewBaseTranslator(config)
	
	// Test cache miss
	translated, found := bt.CheckCache("test text")
	assert.Equal(t, "", translated)
	assert.False(t, found)
	
	// Add to cache manually
	bt.cache["test text"] = "cached translation"
	
	// Test cache hit
	translated, found = bt.CheckCache("test text")
	assert.Equal(t, "cached translation", translated)
	assert.True(t, found)
	
	// Check stats update
	stats := bt.GetStats()
	assert.Equal(t, 1, stats.Cached)
}

// TestBaseTranslator_AddToCache tests AddToCache function
func TestBaseTranslator_AddToCache(t *testing.T) {
	config := TranslationConfig{
		Provider: "test",
		Model:    "test-model",
		APIKey:   "test-key",
	}
	
	bt := NewBaseTranslator(config)
	
	// Add to cache
	bt.AddToCache("test text", "cached translation")
	
	// Verify it was added
	assert.Equal(t, "cached translation", bt.cache["test text"])
}

// TestUniversalTranslator_GetSourceLanguage tests GetSourceLanguage function
func TestUniversalTranslator_GetSourceLanguage(t *testing.T) {
	mockTranslator := &MockTranslator{}
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	// Test GetSourceLanguage
	result := ut.GetSourceLanguage()
	assert.Equal(t, sourceLang, result)
}

// TestUniversalTranslator_GetTargetLanguage tests GetTargetLanguage function
func TestUniversalTranslator_GetTargetLanguage(t *testing.T) {
	mockTranslator := &MockTranslator{}
	
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	
	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)
	
	// Test GetTargetLanguage
	result := ut.GetTargetLanguage()
	assert.Equal(t, targetLang, result)
}

// TestUniversalTranslator_CreatePromptForLanguages tests CreatePromptForLanguages function
func TestUniversalTranslator_CreatePromptForLanguages(t *testing.T) {
	sourceLang := "en"
	targetLang := "ru"
	context := "Literary text"
	
	// Test prompt creation
	prompt := CreatePromptForLanguages("Hello world", sourceLang, targetLang, context)
	
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Hello world")
	assert.Contains(t, prompt, "en")
	assert.Contains(t, prompt, "ru")
	assert.Contains(t, prompt, context)
}

// TestBaseTranslator_EmitError tests EmitError function
func TestBaseTranslator_EmitError(t *testing.T) {
	// Set up event bus
	eventBus := events.NewEventBus()
	
	// Track error events
	errorReceived := false
	eventBus.Subscribe(events.EventTranslationError, func(event events.Event) {
		errorReceived = true
		assert.Contains(t, event.Message, "Test error")
	})
	
	// Emit error
	EmitError(eventBus, "test-session", "Test error", nil)
	
	// Wait for async event
	time.Sleep(time.Millisecond * 10)
	
	assert.True(t, errorReceived, "Error event should be emitted")
}