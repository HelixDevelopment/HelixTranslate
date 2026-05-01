package translator

import (
	"context"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestUniversalTranslator_Creation tests universal translator creation
func TestUniversalTranslator_Creation(t *testing.T) {
	mockTranslator := &MockTranslator{}

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	t.Run("NewUniversalTranslator with detector", func(t *testing.T) {
		mockLLMDetector := &MockLLMDetector{}
		mockDetector := language.NewDetector(mockLLMDetector)

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

	t.Run("NewUniversalTranslator with same languages", func(t *testing.T) {
		ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, sourceLang)

		assert.Equal(t, sourceLang, ut.sourceLanguage)
		assert.Equal(t, sourceLang, ut.targetLanguage)
	})
}

// TestUniversalTranslator_TranslateBook_Basic tests basic book translation scenarios
func TestUniversalTranslator_TranslateBook_Basic(t *testing.T) {
	mockTranslator := &MockTranslator{}
	mockLLMDetector := &MockLLMDetector{}
	mockDetector := language.NewDetector(mockLLMDetector)

	// Set up mock expectations
	mockLLMDetector.On("DetectLanguage", mock.Anything, mock.Anything).Return("en", nil)
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)
	mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil).Maybe()

	sourceLang := language.Language{Code: "", Name: ""} // Empty source language
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	ut := NewUniversalTranslator(mockTranslator, mockDetector, sourceLang, targetLang)

	t.Run("TranslateBook with nil book", func(t *testing.T) {
		ctx := context.Background()
		eventBus := events.NewEventBus()
		sessionID := "test-session"

		err := ut.TranslateBook(ctx, nil, eventBus, sessionID)

		assert.Error(t, err)
	})

	t.Run("TranslateBook with empty book", func(t *testing.T) {
		ctx := context.Background()
		eventBus := events.NewEventBus()
		sessionID := "test-session"

		book := &ebook.Book{}

		err := ut.TranslateBook(ctx, book, eventBus, sessionID)

		assert.NoError(t, err)
		assert.Equal(t, "ru", book.Metadata.Language)
	})

	t.Run("TranslateBook with basic chapters", func(t *testing.T) {
		ctx := context.Background()
		eventBus := events.NewEventBus()
		sessionID := "test-session"

		book := &ebook.Book{
			Metadata: ebook.Metadata{Title: "Test Book", Authors: []string{"Author"}},
			Chapters: []ebook.Chapter{
				{Title: "Ch1", Sections: []ebook.Section{{Title: "Sec1", Content: "Hello"}}},
			},
		}

		mockLLMDetector.On("DetectLanguage", ctx, mock.AnythingOfType("string")).Return("en", nil)
		mockTranslator.On("TranslateWithProgress", ctx, "Test Book", "Book title", eventBus, sessionID).Return("Книга", nil)
		mockTranslator.On("TranslateWithProgress", ctx, "Ch1", "Chapter title", eventBus, sessionID).Return("Глава", nil)
		mockTranslator.On("TranslateWithProgress", ctx, "Hello", "Section content", eventBus, sessionID).Return("Привет", nil)

		err := ut.TranslateBook(ctx, book, eventBus, sessionID)
		assert.NoError(t, err)
		assert.Equal(t, "ru", book.Metadata.Language)
	})
}

// TestUniversalTranslator_MultipleBooks tests translating multiple books
func TestUniversalTranslator_MultipleBooks(t *testing.T) {
	mockTranslator := &MockTranslator{}
	mockLLMDetector := &MockLLMDetector{}
	mockDetector := language.NewDetector(mockLLMDetector)

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}
	ut := NewUniversalTranslator(mockTranslator, mockDetector, sourceLang, targetLang)

	books := []*ebook.Book{
		{Metadata: ebook.Metadata{Title: "Book 1"}, Chapters: []ebook.Chapter{{Title: "Ch1", Sections: []ebook.Section{{Content: "Hello"}}}}},
		{Metadata: ebook.Metadata{Title: "Book 2"}, Chapters: []ebook.Chapter{{Title: "Ch1", Sections: []ebook.Section{{Content: "World"}}}}},
	}

	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "test-session"

	mockLLMDetector.On("DetectLanguage", ctx, mock.AnythingOfType("string")).Return("en", nil)
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)

	for _, book := range books {
		err := ut.TranslateBook(ctx, book, eventBus, sessionID)
		assert.NoError(t, err)
		assert.Equal(t, "ru", book.Metadata.Language)
	}
}

// TestUniversalTranslator_LanguageDetection tests language detection scenarios
func TestUniversalTranslator_LanguageDetection(t *testing.T) {
	mockTranslator := &MockTranslator{}
	mockLLMDetector := &MockLLMDetector{}
	mockDetector := language.NewDetector(mockLLMDetector)

	targetLang := language.Language{Code: "ru", Name: "Russian"}
	ut := NewUniversalTranslator(mockTranslator, mockDetector, language.Language{}, targetLang)

	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "test-session"

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "Hello World", Language: ""},
		Chapters: []ebook.Chapter{{Title: "Ch1", Sections: []ebook.Section{{Content: "English text here"}}}},
	}

	mockLLMDetector.On("DetectLanguage", ctx, mock.AnythingOfType("string")).Return("en", nil)
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)

	err := ut.TranslateBook(ctx, book, eventBus, sessionID)
	assert.NoError(t, err)
	mockLLMDetector.AssertExpectations(t)
}

// BenchmarkUniversalTranslator_New benchmarks UniversalTranslator creation
func BenchmarkUniversalTranslator_New(b *testing.B) {
	mockTranslator := &MockTranslator{}
	mockLLMDetector := &MockLLMDetector{}
	mockDetector := language.NewDetector(mockLLMDetector)
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		NewUniversalTranslator(mockTranslator, mockDetector, sourceLang, targetLang)
	}
}

// BenchmarkUniversalTranslator_TranslateBook benchmarks book translation
func BenchmarkUniversalTranslator_TranslateBook(b *testing.B) {
	mockTranslator := &MockTranslator{}
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "ru", Name: "Russian"}

	ut := NewUniversalTranslator(mockTranslator, nil, sourceLang, targetLang)

	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "bench-session"

	// Mock translation
	mockTranslator.On("TranslateWithProgress", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("Translated", nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Use a fresh book for each iteration
		freshBook := &ebook.Book{
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

		// Reset mock expectations periodically
		if i%100 == 0 {
			mockTranslator.ExpectedCalls = nil
		}

		ut.TranslateBook(ctx, freshBook, eventBus, sessionID)
	}
}
