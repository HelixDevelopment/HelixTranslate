package preparation

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// MockTranslator implements translator.Translator for testing
type MockTranslator struct {
	name     string
	response string
	err      error
}

func (m *MockTranslator) Translate(ctx context.Context, text, context string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	
	// Check if this is a chapter analysis prompt and extract chapter number
	if strings.Contains(text, "You are analyzing Chapter") && strings.Contains(text, "for translation preparation") {
		// Extract chapter number from the first line which contains "Chapter X"
		lines := strings.Split(text, "\n")
		if len(lines) > 0 {
			// Parse from "You are analyzing Chapter X"
			re := regexp.MustCompile(`Chapter (\d+)`)
			matches := re.FindStringSubmatch(lines[0])
			if len(matches) > 1 {
				if chapterNum, err := strconv.Atoi(matches[1]); err == nil {
					// Return response with the correct chapter number
					return fmt.Sprintf(`{
						"chapter_num": %d,
						"title": "Test Chapter",
						"summary": "Test summary",
						"key_points": ["key_point1"],
						"caveats": ["caveat1"],
						"tone": "neutral",
						"complexity": "moderate",
						"special_notes": "special note"
					}`, chapterNum), nil
				}
			}
		}
		// Fallback to chapter 1 if we can't extract
		return `{
			"chapter_num": 1,
			"title": "Test Chapter",
			"summary": "Test summary",
			"key_points": ["key_point1"],
			"caveats": ["caveat1"],
			"tone": "neutral",
			"complexity": "moderate",
			"special_notes": "special note"
		}`, nil
	}
	
	return m.response, nil
}

func (m *MockTranslator) TranslateWithProgress(ctx context.Context, text, context string, eventBus *events.EventBus, sessionID string) (string, error) {
	return m.Translate(ctx, text, context)
}

func (m *MockTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (m *MockTranslator) GetName() string {
	return m.name
}

func TestNewPreparationCoordinator(t *testing.T) {
	// Set API key for testing
	os.Setenv("OPENAI_API_KEY", "test-key-for-testing")
	defer os.Unsetenv("OPENAI_API_KEY")
	
	tests := []struct {
		name        string
		config      PreparationConfig
		expectError bool
	}{
		{
			name: "valid config with providers",
			config: PreparationConfig{
				SourceLanguage: "en",
				TargetLanguage: "es",
				PassCount:      2,
				Providers:      []string{"openai"},  // Use a valid provider
				APIKey:         "test-key-for-testing", // Set API key in config
			},
			expectError: false,
		},
		{
			name: "default pass count",
			config: PreparationConfig{
				SourceLanguage: "en",
				TargetLanguage: "es",
				Providers:      []string{"openai"},  // Use a valid provider
				APIKey:         "test-key-for-testing", // Set API key in config
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator, err := NewPreparationCoordinator(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if coordinator == nil {
					t.Errorf("Expected coordinator but got nil")
					return // Skip further checks if coordinator is nil
				}
				if coordinator.config.PassCount < 1 {
					t.Errorf("Expected PassCount >= 1, got %d", coordinator.config.PassCount)
				}
			}
		})
	}
}

func TestPreparationCoordinator_PrepareBook(t *testing.T) {
	// Mock translator
	mockTranslator := &MockTranslator{
		name: "test-provider",
		response: `{
			"content_type": "fiction",
			"genre": "science_fiction",
			"untranslatable_terms": [{"term": "term1", "reason": "test"}, {"term": "term2", "reason": "test"}],
			"footnote_guidance": [{"term": "term1", "explanation": "explanation"}],
			"characters": [{"name": "John", "role": "protagonist"}],
			"cultural_references": [{"reference": "cultural_ref", "explanation": "explanation"}]
		}`,
	}

	coordinator := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage:    "en",
			TargetLanguage:    "es",
			PassCount:         1,
			AnalyzeChapters:   false,
			AnalyzeCulture:    true,
			AnalyzeCharacters: true,
		},
		providers: []translator.Translator{mockTranslator},
	}

	// Create test book
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test Book",
			Authors: []string{"Test Author"},
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "This is test content for chapter 1."},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := coordinator.PrepareBook(ctx, book)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result but got nil")
	}

	// Verify result fields
	if result.SourceLanguage != "en" {
		t.Errorf("Expected SourceLanguage 'en', got '%s'", result.SourceLanguage)
	}
	if result.TargetLanguage != "es" {
		t.Errorf("Expected TargetLanguage 'es', got '%s'", result.TargetLanguage)
	}
	if len(result.Passes) != 1 {
		t.Errorf("Expected 1 pass, got %d", len(result.Passes))
	}

	// Verify analysis
	if result.FinalAnalysis.ContentType != "fiction" {
		t.Errorf("Expected ContentType 'fiction', got '%s'", result.FinalAnalysis.ContentType)
	}
	if len(result.FinalAnalysis.UntranslatableTerms) != 2 {
		t.Errorf("Expected 2 untranslatable terms, got %d", len(result.FinalAnalysis.UntranslatableTerms))
	}
}

func TestPreparationCoordinator_performPass(t *testing.T) {
	mockTranslator := &MockTranslator{
		name: "test-provider",
		response: `{
			"content_type": "non-fiction",
			"genre": "biography",
			"untranslatable_terms": [],
			"footnote_guidance": [],
			"characters": [],
			"cultural_references": []
		}`,
	}

	coordinator := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "en",
			TargetLanguage: "es",
		},
		providers: []translator.Translator{mockTranslator},
	}

	ctx := context.Background()
	content := "Test book content"

	pass, err := coordinator.performPass(ctx, 1, mockTranslator, content, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if pass == nil {
		t.Fatal("Expected pass but got nil")
	}

	if pass.PassNumber != 1 {
		t.Errorf("Expected PassNumber 1, got %d", pass.PassNumber)
	}
	if pass.Provider != "test-provider" {
		t.Errorf("Expected Provider 'test-provider', got '%s'", pass.Provider)
	}
	if pass.Analysis.ContentType != "non-fiction" {
		t.Errorf("Expected ContentType 'non-fiction', got '%s'", pass.Analysis.ContentType)
	}
}

func TestPreparationCoordinator_analyzeChapters(t *testing.T) {
	mockTranslator := &MockTranslator{
		name: "test-provider",
		// Default response (not used for chapter analysis)
		response: `{"status": "ok"}`,
	}

	coordinator := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "en",
			TargetLanguage: "es",
		},
		providers: []translator.Translator{mockTranslator},
	}

	book := &ebook.Book{
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Chapter 1 content"},
				},
			},
			{
				Title: "Chapter 2",
				Sections: []ebook.Section{
					{Content: "Chapter 2 content"},
				},
			},
		},
	}

	ctx := context.Background()
	analyses, err := coordinator.analyzeChapters(ctx, book)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(analyses) != 2 {
		t.Errorf("Expected 2 chapter analyses, got %d", len(analyses))
	}

	// Check that we have both chapters (order might vary due to concurrency)
	chapterNums := make(map[int]bool)
	for _, analysis := range analyses {
		chapterNums[analysis.ChapterNum] = true
	}
	
	if !chapterNums[1] {
		t.Error("Missing Chapter 1")
	}
	if !chapterNums[2] {
		t.Error("Missing Chapter 2")
	}
}

func TestPreparationCoordinator_consolidateAnalyses(t *testing.T) {
	mockTranslator := &MockTranslator{
		name: "test-provider",
		response: `{
			"content_type": "fiction",
			"genre": "fantasy",
			"untranslatable_terms": [{"term": "consolidated_term", "original_script": "Latin", "reason": "Test term", "context": [], "transliteration": ""}],
			"footnote_guidance": [],
			"characters": [],
			"cultural_references": []
		}`,
	}

	coordinator := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "en",
			TargetLanguage: "es",
		},
		providers: []translator.Translator{mockTranslator},
	}

	passes := []PreparationPass{
		{
			PassNumber: 1,
			Provider:   "provider1",
			Analysis: ContentAnalysis{
				ContentType: "fiction",
				Genre:       "science_fiction",
			},
		},
		{
			PassNumber: 2,
			Provider:   "provider2",
			Analysis: ContentAnalysis{
				ContentType: "fiction",
				Genre:       "fantasy",
			},
		},
	}

	ctx := context.Background()
	consolidated, err := coordinator.consolidateAnalyses(ctx, passes)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if consolidated == nil {
		t.Fatal("Expected consolidated analysis but got nil")
	}

	if consolidated.ContentType != "fiction" {
		t.Errorf("Expected ContentType 'fiction', got '%s'", consolidated.ContentType)
	}
	if consolidated.Genre != "fantasy" {
		t.Errorf("Expected Genre 'fantasy', got '%s'", consolidated.Genre)
	}
}

func TestPreparationCoordinator_parseAnalysisResponse(t *testing.T) {
	coordinator := &PreparationCoordinator{}

	tests := []struct {
		name        string
		response    string
		expectError bool
		expected    *ContentAnalysis
	}{
		{
			name:     "valid JSON response",
			response: `{"content_type": "fiction", "genre": "mystery"}`,
			expected: &ContentAnalysis{
				ContentType: "fiction",
				Genre:       "mystery",
			},
			expectError: false,
		},
		{
			name:        "invalid JSON response",
			response:    `{"content_type": "fiction", "genre":}`,
			expectError: true,
		},
		{
			name:     "JSON with extra text",
			response: `Here is the analysis: {"content_type": "fiction", "genre": "mystery"} End of analysis`,
			expected: &ContentAnalysis{
				ContentType: "fiction",
				Genre:       "mystery",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := coordinator.parseAnalysisResponse(tt.response)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result.ContentType != tt.expected.ContentType {
					t.Errorf("Expected ContentType '%s', got '%s'", tt.expected.ContentType, result.ContentType)
				}
				if result.Genre != tt.expected.Genre {
					t.Errorf("Expected Genre '%s', got '%s'", tt.expected.Genre, result.Genre)
				}
			}
		})
	}
}

func TestPreparationCoordinator_ExtractBookContent(t *testing.T) {
	coordinator := &PreparationCoordinator{}

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test Book",
			Authors: []string{"Author 1", "Author 2"},
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Section 1 content"},
					{Content: "Section 2 content"},
				},
			},
			{
				Title: "Chapter 2",
				Sections: []ebook.Section{
					{Content: "Section 3 content"},
				},
			},
		},
	}

	content := coordinator.extractBookContent(book)

	// Verify metadata is included
	if !containsString(content, "Title: Test Book") {
		t.Error("Expected book title in content")
	}
	if !containsString(content, "Authors: Author 1, Author 2") {
		t.Error("Expected authors in content")
	}

	// Verify chapters are included
	if !containsString(content, "## Chapter 1: Chapter 1") {
		t.Error("Expected chapter 1 header in content")
	}
	if !containsString(content, "## Chapter 2: Chapter 2") {
		t.Error("Expected chapter 2 header in content")
	}

	// Verify sections are included
	if !containsString(content, "Section 1 content") {
		t.Error("Expected section 1 content in content")
	}
	if !containsString(content, "Section 3 content") {
		t.Error("Expected section 3 content in content")
	}
}

func TestPreparationCoordinator_ExtractChapterContent(t *testing.T) {
	coordinator := &PreparationCoordinator{}

	chapter := &ebook.Chapter{
		Title: "Test Chapter",
		Sections: []ebook.Section{
			{Content: "First section content"},
			{Content: "Second section content"},
		},
	}

	content := coordinator.extractChapterContent(chapter)

	if !containsString(content, "First section content") {
		t.Error("Expected first section content")
	}
	if !containsString(content, "Second section content") {
		t.Error("Expected second section content")
	}
}

func TestPreparationCoordinator_PrepareBook_WithChapters(t *testing.T) {
	mockTranslator := &MockTranslator{
		name: "test-provider",
		response: `{
			"content_type": "fiction",
			"genre": "science_fiction",
			"untranslatable_terms": [],
			"footnote_guidance": [],
			"characters": [],
			"cultural_references": []
		}`,
	}

	coordinator := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage:  "en",
			TargetLanguage:  "es",
			PassCount:       1,
			AnalyzeChapters: true,
		},
		providers: []translator.Translator{mockTranslator},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title: "Test Book",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Chapter 1 content"},
				},
			},
		},
	}

	ctx := context.Background()
	result, err := coordinator.PrepareBook(ctx, book)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result.Passes) == 0 {
		t.Error("Expected at least one pass")
		return
	}

	// Check if chapter analyses were added
	lastPass := result.Passes[len(result.Passes)-1]
	if len(lastPass.Analysis.ChapterAnalyses) == 0 {
		t.Error("Expected chapter analyses to be added")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsStringRecursive(s, substr)))
}

func containsStringRecursive(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsStringRecursive(s[1:], substr)
}
