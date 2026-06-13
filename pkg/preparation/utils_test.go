package preparation

import (
	"digital.vasic.translator/pkg/ebook"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractJSON tests JSON extraction from LLM responses
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with prefix text",
			input:    `Here is the analysis:\n{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with suffix text",
			input:    `{"key": "value"}\nThat was the analysis.`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with prefix and suffix",
			input:    `Analysis:\n{"key": "value"}\nEnd of analysis`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "nested JSON objects",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:  "JSON with array inside",
			input: `Text before [{"a": 1}, {"b": 2}] text after`,
			// Reconciled per the robust extractor (§11.4.120): the old
			// first-`{`..last-`}` slice returned the INVALID span
			// `{"a": 1}, {"b": 2}`. The balanced scanner now returns the
			// whole valid top-level array.
			expected: `[{"a": 1}, {"b": 2}]`,
		},
		{
			name:     "complex nested structure",
			input:    `{"a": {"b": {"c": [1,2,3]}}}`,
			expected: `{"a": {"b": {"c": [1,2,3]}}}`,
		},
		{
			name:     "no JSON",
			input:    `This is just plain text`,
			expected: `This is just plain text`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractJSON_ParseableOutput tests that extracted JSON is valid
func TestExtractJSON_ParseableOutput(t *testing.T) {
	input := `Here is the result:
{"genre": "Fiction", "tone": "Dark", "themes": ["mystery", "suspense"]}
End of response`

	result := extractJSON(input)

	// Should be valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	assert.NoError(t, err, "Extracted JSON should be valid")
	assert.Equal(t, "Fiction", parsed["genre"])
	assert.Equal(t, "Dark", parsed["tone"])
}

// TestEstimateTokens tests token estimation
func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "4 characters",
			input:    "test",
			expected: 1, // 4 chars / 4 = 1 token
		},
		{
			name:     "8 characters",
			input:    "testtest",
			expected: 2, // 8 chars / 4 = 2 tokens
		},
		{
			name:     "100 characters",
			input:    strings.Repeat("a", 100),
			expected: 25, // 100 chars / 4 = 25 tokens
		},
		{
			name:     "short sentence",
			input:    "Hello, world!",
			expected: 3, // 13 chars / 4 = 3 tokens (rounded down)
		},
		{
			name:     "long paragraph",
			input:    "This is a longer paragraph that contains multiple words and punctuation. It should be estimated at roughly one token per four characters, which is a reasonable approximation for English text.",
			expected: 47, // ~190 chars / 4 = 47 tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokens(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEstimateTokens_Consistency tests that estimation is consistent
func TestEstimateTokens_Consistency(t *testing.T) {
	text := "This is a test sentence."

	// Multiple calls should return same result
	result1 := estimateTokens(text)
	result2 := estimateTokens(text)
	result3 := estimateTokens(text)

	assert.Equal(t, result1, result2)
	assert.Equal(t, result2, result3)
}

// TestEstimateTokens_LargeText tests estimation on large text
func TestEstimateTokens_LargeText(t *testing.T) {
	// Create a large text (10,000 characters)
	largeText := strings.Repeat("Lorem ipsum dolor sit amet, ", 333) // ~9,324 chars

	tokens := estimateTokens(largeText)

	// Should be approximately 2,331 tokens (9324 / 4)
	assert.Greater(t, tokens, 2000)
	assert.Less(t, tokens, 3000)
}

// TestExtractBookContent tests book content extraction
func TestExtractBookContent(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test Book",
			Authors: []string{"Author One", "Author Two"},
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "First section content."},
					{Content: "Second section content."},
				},
			},
			{
				Title: "Chapter 2",
				Sections: []ebook.Section{
					{Content: "Chapter two content."},
				},
			},
		},
	}

	content := pc.extractBookContent(book)

	// Should contain metadata
	assert.Contains(t, content, "Test Book")
	assert.Contains(t, content, "Author One, Author Two")

	// Should contain chapter titles
	assert.Contains(t, content, "Chapter 1")
	assert.Contains(t, content, "Chapter 2")

	// Should contain section content
	assert.Contains(t, content, "First section content.")
	assert.Contains(t, content, "Second section content.")
	assert.Contains(t, content, "Chapter two content.")
}

// TestExtractBookContent_EmptyBook tests empty book handling
func TestExtractBookContent_EmptyBook(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title: "Empty Book",
		},
		Chapters: []ebook.Chapter{},
	}

	content := pc.extractBookContent(book)

	assert.Contains(t, content, "Empty Book")
	assert.NotEmpty(t, content)
}

// TestExtractBookContent_NoAuthors tests book without authors
func TestExtractBookContent_NoAuthors(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title: "Authorless Book",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter",
				Sections: []ebook.Section{
					{Content: "Content"},
				},
			},
		},
	}

	content := pc.extractBookContent(book)

	assert.Contains(t, content, "Authorless Book")
	assert.Contains(t, content, "Chapter")
	assert.Contains(t, content, "Content")
}

// TestExtractChapterContent tests chapter content extraction
func TestExtractChapterContent(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	chapter := &ebook.Chapter{
		Title: "Test Chapter",
		Sections: []ebook.Section{
			{Content: "Section one."},
			{Content: "Section two."},
			{Content: "Section three."},
		},
	}

	content := pc.extractChapterContent(chapter)

	assert.Contains(t, content, "Section one.")
	assert.Contains(t, content, "Section two.")
	assert.Contains(t, content, "Section three.")
}

// TestExtractChapterContent_EmptyChapter tests empty chapter
func TestExtractChapterContent_EmptyChapter(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	chapter := &ebook.Chapter{
		Title:    "Empty Chapter",
		Sections: []ebook.Section{},
	}

	content := pc.extractChapterContent(chapter)

	// Should return empty or minimal content
	assert.NotNil(t, content)
}

// TestExtractChapterContent_SingleSection tests single section chapter
func TestExtractChapterContent_SingleSection(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	chapter := &ebook.Chapter{
		Title: "Single Section",
		Sections: []ebook.Section{
			{Content: "Only content."},
		},
	}

	content := pc.extractChapterContent(chapter)

	assert.Contains(t, content, "Only content.")
}

// TestExtractChapterContent_Whitespace tests whitespace handling
func TestExtractChapterContent_Whitespace(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	chapter := &ebook.Chapter{
		Sections: []ebook.Section{
			{Content: "  Text with whitespace  "},
			{Content: "\n\nNewlines\n\n"},
		},
	}

	content := pc.extractChapterContent(chapter)

	// Should preserve whitespace
	assert.Contains(t, content, "Text with whitespace")
	assert.Contains(t, content, "Newlines")
}

// TestExtractBookContent_ChapterNumbering tests chapter numbering
func TestExtractBookContent_ChapterNumbering(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "Test"},
		Chapters: []ebook.Chapter{
			{Title: "First", Sections: []ebook.Section{{Content: "A"}}},
			{Title: "Second", Sections: []ebook.Section{{Content: "B"}}},
			{Title: "Third", Sections: []ebook.Section{{Content: "C"}}},
		},
	}

	content := pc.extractBookContent(book)

	// Should have chapter numbers
	assert.Contains(t, content, "Chapter 1")
	assert.Contains(t, content, "Chapter 2")
	assert.Contains(t, content, "Chapter 3")
}

// TestExtractBookContent_UntitledChapters tests chapters without titles
func TestExtractBookContent_UntitledChapters(t *testing.T) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "Test"},
		Chapters: []ebook.Chapter{
			{Title: "", Sections: []ebook.Section{{Content: "Untitled content"}}},
		},
	}

	content := pc.extractBookContent(book)

	assert.Contains(t, content, "Chapter 1")
	assert.Contains(t, content, "Untitled content")
}

// BenchmarkExtractJSON benchmarks JSON extraction
func BenchmarkExtractJSON(b *testing.B) {
	input := `Some text before {"key": "value", "nested": {"data": [1,2,3]}} some text after`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractJSON(input)
	}
}

// BenchmarkEstimateTokens benchmarks token estimation
func BenchmarkEstimateTokens(b *testing.B) {
	text := strings.Repeat("This is a test sentence. ", 100) // ~2500 chars

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimateTokens(text)
	}
}

// BenchmarkExtractBookContent benchmarks book content extraction
func BenchmarkExtractBookContent(b *testing.B) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Benchmark Book",
			Authors: []string{"Author"},
		},
		Chapters: make([]ebook.Chapter, 20),
	}

	for i := range book.Chapters {
		book.Chapters[i] = ebook.Chapter{
			Title: "Chapter",
			Sections: []ebook.Section{
				{Content: strings.Repeat("Content ", 100)},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pc.extractBookContent(book)
	}
}

// BenchmarkExtractChapterContent benchmarks chapter extraction
func BenchmarkExtractChapterContent(b *testing.B) {
	pc := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		},
	}

	chapter := &ebook.Chapter{
		Title: "Benchmark Chapter",
		Sections: []ebook.Section{
			{Content: strings.Repeat("Section content. ", 100)},
			{Content: strings.Repeat("More content. ", 100)},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pc.extractChapterContent(chapter)
	}
}
