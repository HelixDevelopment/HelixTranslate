package unit

import (
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEbookStructure(t *testing.T) {
	t.Run("CreateBook", func(t *testing.T) {
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Test Book",
				Authors:  []string{"Test Author"},
				Language: "en",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{
							Content: "This is test content.",
						},
					},
				},
			},
		}

		if book.Metadata.Title != "Test Book" {
			t.Errorf("Expected title 'Test Book', got '%s'", book.Metadata.Title)
		}

		if book.GetChapterCount() != 1 {
			t.Errorf("Expected 1 chapter, got %d", book.GetChapterCount())
		}
	})

	t.Run("ExtractText", func(t *testing.T) {
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title: "Test Book",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{
							Content: "First section content.",
						},
						{
							Content: "Second section content.",
						},
					},
				},
			},
		}

		text := book.ExtractText()

		if text == "" {
			t.Error("ExtractText returned empty string")
		}

		if !containsString(text, "Test Book") {
			t.Error("Extracted text should contain title")
		}

		if !containsString(text, "Chapter 1") {
			t.Error("Extracted text should contain chapter title")
		}

		if !containsString(text, "First section content") {
			t.Error("Extracted text should contain section content")
		}
	})

	t.Run("GetWordCount", func(t *testing.T) {
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title: "Test",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter",
					Sections: []ebook.Section{
						{
							Content: "One two three four five.",
						},
					},
				},
			},
		}

		wordCount := book.GetWordCount()

		if wordCount < 5 {
			t.Errorf("Expected word count >= 5, got %d", wordCount)
		}
	})
}

func TestEPUBParser(t *testing.T) {
	t.Run("ParseValidEPUB", func(t *testing.T) {
		parser := ebook.NewUniversalParser()
		book, err := parser.Parse("../../test_output.epub")

		if err != nil {
			t.Fatalf("Failed to parse valid EPUB: %v", err)
		}

		if book.Format != format.FormatEPUB {
			t.Errorf("Expected format EPUB, got %s", book.Format)
		}

		if book.Metadata.Title == "" {
			t.Error("Expected non-empty title")
		}

		if len(book.Chapters) == 0 {
			t.Error("Expected at least one chapter")
		}
	})

	t.Run("ParseNonexistentFile", func(t *testing.T) {
		parser := ebook.NewUniversalParser()
		_, err := parser.Parse("nonexistent.epub")

		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})

	t.Run("CleanXMLData", func(t *testing.T) {
		parser := ebook.NewEPUBParser()

		// Test with malformed XML containing unescaped ampersand
		malformed := []byte(`<test attr="value & more">content</test>`)
		cleaned := parser.CleanXMLData(malformed)

		cleanedStr := string(cleaned)
		if !strings.Contains(cleanedStr, "&amp;") {
			t.Error("Expected ampersand to be escaped")
		}

		// Test with invalid characters
		invalid := []byte("<?xml version=\"1.0\"?><test>\x00\x01\x02</test>")
		cleaned = parser.CleanXMLData(invalid)

		if strings.Contains(string(cleaned), "\x00") {
			t.Error("Expected null bytes to be removed")
		}

		// Test with missing XML declaration
		noDecl := []byte(`<root><child>content</child></root>`)
		cleaned = parser.CleanXMLData(noDecl)

		if !strings.HasPrefix(string(cleaned), "<?xml") {
			t.Error("Expected XML declaration to be added")
		}
	})

	t.Run("ParseMalformedEPUB", func(t *testing.T) {
		// Create a temporary malformed EPUB file for testing
		tempDir := t.TempDir()
		malformedEpub := filepath.Join(tempDir, "malformed.epub")

		// Create a ZIP file with malformed XML
		// This is a simplified test - in practice, we'd need to create a proper EPUB structure
		err := os.WriteFile(malformedEpub, []byte("not a zip file"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		parser := ebook.NewUniversalParser()
		_, err = parser.Parse(malformedEpub)

		// Should fail gracefully
		if err == nil {
			t.Error("Expected parsing to fail for malformed file")
		}
	})

	t.Run("ParseUnsupportedFormat", func(t *testing.T) {
		tempDir := t.TempDir()
		unsupportedFile := filepath.Join(tempDir, "test.pdf")

		err := os.WriteFile(unsupportedFile, []byte("%PDF-1.4 test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		parser := ebook.NewUniversalParser()
		_, err = parser.Parse(unsupportedFile)

		// Should fail for unsupported format
		if err == nil {
			t.Error("Expected parsing to fail for unsupported format")
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
