package preparation

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
	"testing"
)

// MockTranslator2 implements translator.Translator for testing (different name to avoid conflict)
type MockTranslator2 struct {
	name     string
	response string
	err      error
}

func (m *MockTranslator2) Translate(ctx context.Context, text, context string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *MockTranslator2) TranslateWithProgress(ctx context.Context, text, context string, eventBus *events.EventBus, sessionID string) (string, error) {
	return m.Translate(ctx, text, context)
}

func (m *MockTranslator2) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (m *MockTranslator2) GetName() string {
	return m.name
}

// MockLLMDetector implements language.LLMDetector for testing
type MockLLMDetector struct {
	detectedLang string
	err          error
}

func (m *MockLLMDetector) DetectLanguage(ctx context.Context, text string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.detectedLang, nil
}

func TestNewPreparationAwareTranslator(t *testing.T) {
	mockTranslator := &MockTranslator2{name: "test-translator"}
	mockLLMDetector := &MockLLMDetector{detectedLang: "en"}
	mockDetector := language.NewDetector(mockLLMDetector)

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}

	tests := []struct {
		name              string
		preparationConfig *PreparationConfig
		expectPreparation bool
	}{
		{
			name: "with preparation enabled",
			preparationConfig: &PreparationConfig{
				PassCount:      2,
				SourceLanguage: "en",
				TargetLanguage: "es",
			},
			expectPreparation: true,
		},
		{
			name:              "with preparation disabled",
			preparationConfig: nil,
			expectPreparation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat := NewPreparationAwareTranslator(
				mockTranslator,
				mockDetector,
				sourceLang,
				targetLang,
				tt.preparationConfig,
			)

			if pat.baseTranslator != mockTranslator {
				t.Error("Base translator not set correctly")
			}
			if pat.langDetector != mockDetector {
				t.Error("Language detector not set correctly")
			}
			if pat.enablePreparation != tt.expectPreparation {
				t.Errorf("Expected enablePreparation %v, got %v", tt.expectPreparation, pat.enablePreparation)
			}
		})
	}
}

func TestPreparationAwareTranslator_TranslateBook_WithoutPreparation(t *testing.T) {
	mockTranslator := &MockTranslator2{
		name:     "test-translator",
		response: "Translated text",
	}
	mockLLMDetector := &MockLLMDetector{detectedLang: "en"}
	mockDetector := language.NewDetector(mockLLMDetector)

	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}

	pat := NewPreparationAwareTranslator(
		mockTranslator,
		mockDetector,
		sourceLang,
		targetLang,
		nil, // No preparation
	)

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Test Book",
			Description: "Test description",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Chapter content"},
				},
			},
		},
	}

	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "test-session"

	err := pat.TranslateBook(ctx, book, eventBus, sessionID)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify metadata was translated
	if book.Metadata.Title != "Translated text" {
		t.Errorf("Expected title to be translated, got: %s", book.Metadata.Title)
	}
	if book.Metadata.Description != "Translated text" {
		t.Errorf("Expected description to be translated, got: %s", book.Metadata.Description)
	}

	// Verify chapters were translated
	if book.Chapters[0].Title != "Translated text" {
		t.Errorf("Expected chapter title to be translated, got: %s", book.Chapters[0].Title)
	}
	if book.Chapters[0].Sections[0].Content != "Translated text" {
		t.Errorf("Expected chapter content to be translated, got: %s", book.Chapters[0].Sections[0].Content)
	}

	// Verify language was updated
	if book.Metadata.Language != "es" {
		t.Errorf("Expected book language to be 'es', got: %s", book.Metadata.Language)
	}
}

func TestPreparationAwareTranslator_TranslateBook_WithPreparation(t *testing.T) {
	mockTranslator := &MockTranslator2{
		name: "test-translator",
		response: `{
			"content_type": "fiction",
			"genre": "science_fiction",
			"untranslatable_terms": [],
			"footnote_guidance": [],
			"characters": [],
			"cultural_references": []
		}`,
	}
	mockLLMDetector := &MockLLMDetector{detectedLang: "en"}
	mockDetector := language.NewDetector(mockLLMDetector)

	sourceLang := language.Language{Code: "", Name: ""} // Will be detected
	targetLang := language.Language{Code: "es", Name: "Spanish"}

	prepConfig := &PreparationConfig{
		PassCount:      1,
		SourceLanguage: "en",
		TargetLanguage: "es",
		Providers:      []string{"openai"},  // Use a valid provider
		APIKey:         "test-key-for-testing",
	}

	pat := NewPreparationAwareTranslator(
		mockTranslator,
		mockDetector,
		sourceLang,
		targetLang,
		prepConfig,
	)

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Test Book",
			Description: "Test description",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Chapter content"},
				},
			},
		},
	}

	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "test-session"

	err := pat.TranslateBook(ctx, book, eventBus, sessionID)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify preparation result was set
	if pat.preparationResult == nil {
		t.Error("Expected preparation result to be set")
	}

	// Verify language detection worked
	if pat.sourceLanguage.Code != "en" {
		t.Errorf("Expected source language to be detected as 'en', got: %s", pat.sourceLanguage.Code)
	}
}

func TestPreparationAwareTranslator_getChapterContext(t *testing.T) {
	tests := []struct {
		name              string
		preparationResult *PreparationResult
		chapterNum        int
		expectedContains  []string
	}{
		{
			name:              "no preparation result",
			preparationResult: nil,
			chapterNum:        1,
			expectedContains:  []string{"Literary text"},
		},
		{
			name: "with preparation result",
			preparationResult: &PreparationResult{
				FinalAnalysis: ContentAnalysis{
					ContentType: "fiction",
					Genre:       "science_fiction",
					Tone:        "formal",
					Characters: []Character{
						{Name: "John", Role: "protagonist", SpeechPattern: "Formal and articulate"},
					},
					ChapterAnalyses: []ChapterAnalysis{
						{ChapterNum: 1, Summary: "Chapter 1 summary"},
					},
				},
			},
			chapterNum: 1,
			expectedContains: []string{
				"fiction",
				"science_fiction",
				"formal",
				"John",
				"Formal and articulate",
				"Chapter 1 summary",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat := &PreparationAwareTranslator{
				preparationResult: tt.preparationResult,
			}

			context := pat.getChapterContext(tt.chapterNum)

			for _, expected := range tt.expectedContains {
				if !containsString(context, expected) {
					t.Errorf("Expected context to contain '%s', got: %s", expected, context)
				}
			}
		})
	}
}

func TestPreparationAwareTranslator_isUntranslatable(t *testing.T) {
	tests := []struct {
		name              string
		preparationResult *PreparationResult
		term              string
		expected          bool
	}{
		{
			name:              "no preparation result",
			preparationResult: nil,
			term:              "any term",
			expected:          false,
		},
		{
			name: "term is untranslatable",
			preparationResult: &PreparationResult{
				FinalAnalysis: ContentAnalysis{
					UntranslatableTerms: []UntranslatableTerm{
						{Term: "Wand"},
						{Term: "Hogwarts"},
					},
				},
			},
			term:     "magic wand",
			expected: true,
		},
		{
			name: "term is translatable",
			preparationResult: &PreparationResult{
				FinalAnalysis: ContentAnalysis{
					UntranslatableTerms: []UntranslatableTerm{
						{Term: "Wand"},
						{Term: "Hogwarts"},
					},
				},
			},
			term:     "regular word",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat := &PreparationAwareTranslator{
				preparationResult: tt.preparationResult,
			}

			result := pat.isUntranslatable(tt.term)

			if result != tt.expected {
				t.Errorf("Expected %v for term '%s', got %v", tt.expected, tt.term, result)
			}
		})
	}
}

func TestPreparationAwareTranslator_translateMetadata(t *testing.T) {
	mockTranslator := &MockTranslator2{
		name:     "test-translator",
		response: "Translated text",
	}

	pat := &PreparationAwareTranslator{
		baseTranslator: mockTranslator,
		preparationResult: &PreparationResult{
			FinalAnalysis: ContentAnalysis{
				ContentType: "fiction",
				Tone:        "formal",
			},
		},
	}

	tests := []struct {
		name     string
		metadata *ebook.Metadata
	}{
		{
			name: "complete metadata",
			metadata: &ebook.Metadata{
				Title:       "Original Title",
				Description: "Original description",
				Authors:     []string{"Author 1"},
				Publisher:   "Publisher",
				ISBN:        "1234567890",
				Date:        "2023-01-01",
			},
		},
		{
			name: "minimal metadata",
			metadata: &ebook.Metadata{
				Title: "Just Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMetadata := *tt.metadata
			ctx := context.Background()
			eventBus := events.NewEventBus()
			sessionID := "test-session"

			err := pat.translateMetadata(ctx, tt.metadata, eventBus, sessionID)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Title should be translated
			if tt.metadata.Title != "Translated text" {
				t.Errorf("Expected title to be translated, got: %s", tt.metadata.Title)
			}

			// Description should be translated
			if originalMetadata.Description != "" && tt.metadata.Description != "Translated text" {
				t.Errorf("Expected description to be translated, got: %s", tt.metadata.Description)
			}

			// Authors, Publisher, ISBN, Date should remain unchanged
			if len(tt.metadata.Authors) != len(originalMetadata.Authors) {
				t.Error("Authors should not be translated")
			}
			if tt.metadata.Publisher != originalMetadata.Publisher {
				t.Error("Publisher should not be translated")
			}
			if tt.metadata.ISBN != originalMetadata.ISBN {
				t.Error("ISBN should not be translated")
			}
			if tt.metadata.Date != originalMetadata.Date {
				t.Error("Date should not be translated")
			}
		})
	}
}

func TestPreparationAwareTranslator_translateChapter(t *testing.T) {
	mockTranslator := &MockTranslator2{
		name:     "test-translator",
		response: "Translated text",
	}

	pat := &PreparationAwareTranslator{
		baseTranslator: mockTranslator,
	}

	chapter := &ebook.Chapter{
		Title: "Original Chapter Title",
		Sections: []ebook.Section{
			{
				Title:   "Section Title",
				Content: "Section content",
				Subsections: []ebook.Section{
					{
						Title:   "Subsection Title",
						Content: "Subsection content",
					},
				},
			},
		},
	}

	ctx := context.Background()
	eventBus := events.NewEventBus()
	sessionID := "test-session"
	prepContext := "Test context"

	err := pat.translateChapter(ctx, chapter, prepContext, eventBus, sessionID)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify chapter title was translated
	if chapter.Title != "Translated text" {
		t.Errorf("Expected chapter title to be translated, got: %s", chapter.Title)
	}

	// Verify section was translated
	section := chapter.Sections[0]
	if section.Title != "Translated text" {
		t.Errorf("Expected section title to be translated, got: %s", section.Title)
	}
	if section.Content != "Translated text" {
		t.Errorf("Expected section content to be translated, got: %s", section.Content)
	}

	// Verify subsection was translated
	subsection := section.Subsections[0]
	if subsection.Title != "Translated text" {
		t.Errorf("Expected subsection title to be translated, got: %s", subsection.Title)
	}
	if subsection.Content != "Translated text" {
		t.Errorf("Expected subsection content to be translated, got: %s", subsection.Content)
	}
}

func TestPreparationAwareTranslator_GetPreparationResult(t *testing.T) {
	expectedResult := &PreparationResult{
		SourceLanguage: "en",
		TargetLanguage: "es",
	}

	pat := &PreparationAwareTranslator{
		preparationResult: expectedResult,
	}

	result := pat.GetPreparationResult()

	if result != expectedResult {
		t.Error("Expected preparation result to be returned")
	}
}

func TestPreparationAwareTranslator_SavePreparationAnalysis(t *testing.T) {
	tests := []struct {
		name              string
		preparationResult *PreparationResult
		outputPath        string
		expectError       bool
	}{
		{
			name:              "no preparation result",
			preparationResult: nil,
			outputPath:        "/tmp/test.json",
			expectError:       true,
		},
		{
			name: "with preparation result",
			preparationResult: &PreparationResult{
				SourceLanguage: "en",
				TargetLanguage: "es",
			},
			outputPath:  "/tmp/test_preparation.json",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat := &PreparationAwareTranslator{
				preparationResult: tt.preparationResult,
			}

			err := pat.SavePreparationAnalysis(tt.outputPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
