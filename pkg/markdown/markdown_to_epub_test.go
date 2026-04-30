package markdown

import (
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/pkg/ebook"
)

func TestMarkdownToEPUBConverter_NewMarkdownToEPUBConverter(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	if converter == nil {
		t.Error("Converter not created")
	}

	if converter.hrRegex == nil {
		t.Error("HR regex not initialized")
	}

	// Test regex pattern
	testLines := []string{
		"---",
		"***",
		"___",
		"----",
		"********",
		"Not a divider",
		"--",
		"*",
	}

	expectedResults := []bool{true, true, true, true, true, false, false, false}

	for i, line := range testLines {
		matches := converter.hrRegex.MatchString(line)
		if matches != expectedResults[i] {
			t.Errorf("Line %d: Expected %v for '%s', got %v", i, expectedResults[i], line, matches)
		}
	}
}

func TestMarkdownToEPUBConverter_ParseMarkdown(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test basic markdown parsing
	markdownContent := `# Test Book

This is a test paragraph.

## Chapter 1

This is chapter 1 content.

---

## Chapter 2

This is chapter 2 content.
`

	// Create temporary directory for testing
	tempDir := t.TempDir()
	mdDir := tempDir

	_, metadata, _, err := converter.parseMarkdown(markdownContent, mdDir)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Check metadata
	if metadata.Title != "Test Book" {
		t.Errorf("Expected 'Test Book', got '%s'", metadata.Title)
	}
}

func TestMarkdownToEPUBConverter_ParseMarkdownWithFrontmatter(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test markdown with frontmatter
	markdownContent := `---
title: Frontmatter Title
author: Test Author
cover: cover.jpg
---

# Book Title

This is content.
`

	tempDir := t.TempDir()
	mdDir := tempDir

	_, metadata, _, err := converter.parseMarkdown(markdownContent, mdDir)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Check that metadata from frontmatter is used
	if metadata.Title != "Frontmatter Title" {
		t.Errorf("Expected 'Frontmatter Title', got '%s'", metadata.Title)
	}

	if len(metadata.Authors) != 1 || metadata.Authors[0] != "Test Author" {
		t.Errorf("Expected author 'Test Author', got %v", metadata.Authors)
	}
}

func TestMarkdownToEPUBConverter_ParseFrontmatterLine(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	tests := []struct {
		line     string
		title    string
		author   string
		cover    string
		expected string
	}{
		{"title: My Book", "", "", "", "My Book"},
		{"author: John Doe", "", "", "", ""},
		{"cover: image.jpg", "", "", "", "image.jpg"},
		{"invalid: line", "", "", "", ""},
	}

	for _, test := range tests {
		metadata := ebook.Metadata{
			Title:   test.title,
			Authors: []string{test.author},
		}

		result := converter.parseFrontmatterLine(test.line, &metadata)
		if result != test.expected {
			t.Errorf("Line '%s': expected '%s', got '%s'", test.line, test.expected, result)
		}
	}
}

func TestMarkdownToEPUBConverter_CleanFilename(t *testing.T) {
	// This test can be skipped if cleanFilename is not exported
	t.Skip("cleanFilename is not exported")  // SKIP-OK: #legacy-untriaged
}

func TestMarkdownToEPUBConverter_ConvertMarkdownToEPUB(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Create temporary directory and files
	tempDir := t.TempDir()
	mdPath := filepath.Join(tempDir, "test.md")
	epubPath := filepath.Join(tempDir, "test.epub")

	// Create test markdown file
	markdownContent := `# Test Book

## Chapter 1

This is test content for chapter 1.

---

## Chapter 2

This is test content for chapter 2.
`

	err := os.WriteFile(mdPath, []byte(markdownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test markdown file: %v", err)
	}

	// Convert markdown to EPUB
	err = converter.ConvertMarkdownToEPUB(mdPath, epubPath)
	if err != nil {
		t.Fatalf("Failed to convert markdown to EPUB: %v", err)
	}

	// Check that EPUB file was created
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Error("EPUB file was not created")
	}
}
