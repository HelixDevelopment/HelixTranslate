package markdown

import (
	"os"
	"path/filepath"
	"strings"
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

Content for chapter 1.

### Section 1.1

Subsection content.

## Chapter 2

Content for chapter 2.

More paragraph content.
`

	chapters, metadata, coverPath, err := converter.parseMarkdown(markdownContent, "")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify chapters
	if len(chapters) == 0 {
		t.Error("No chapters parsed")
	}

	// Should have 2 main chapters (level 1 and 2 headers)
	mainChapters := 0
	for _, chapter := range chapters {
		if chapter.Level <= 2 {
			mainChapters++
		}
	}

	if mainChapters < 2 {
		t.Errorf("Expected at least 2 main chapters, got %d", mainChapters)
	}

	// Verify metadata
	if metadata.Title != "Test Book" {
		t.Errorf("Expected title 'Test Book', got '%s'", metadata.Title)
	}

	if coverPath != "" {
		t.Error("Expected no cover path, got path")
	}

	// Verify chapter content
	chapter1 := chapters[0]
	if !strings.Contains(chapter1.Content, "Content for chapter 1") {
		t.Error("Chapter 1 content not found")
	}
}

func TestMarkdownToEPUBConverter_ParseMarkdownWithMetadata(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test markdown with metadata
	markdownContent := `# Advanced Translation Guide

Author: Test Author
Language: sr
Publisher: Test Publisher
Description: This is a test book for translation.

## Introduction

This is the introduction chapter.

## Chapter 1

Main chapter content.
`

	chapters, metadata, coverPath, err := converter.parseMarkdown(markdownContent, "")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify metadata extraction
	if metadata.Title != "Advanced Translation Guide" {
		t.Errorf("Expected title 'Advanced Translation Guide', got '%s'", metadata.Title)
	}

	if metadata.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", metadata.Author)
	}

	if metadata.Language != "sr" {
		t.Errorf("Expected language 'sr', got '%s'", metadata.Language)
	}

	if metadata.Publisher != "Test Publisher" {
		t.Errorf("Expected publisher 'Test Publisher', got '%s'", metadata.Publisher)
	}

	if metadata.Description != "This is a test book for translation." {
		t.Errorf("Expected description 'This is a test book for translation.', got '%s'", metadata.Description)
	}

	if coverPath != "" {
		t.Error("Expected no cover path, got path")
	}
}

func TestMarkdownToEPUBConverter_ParseMarkdownWithCover(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cover_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test cover image
	coverPath := filepath.Join(tmpDir, "cover.jpg")
	coverData := []byte("fake image data")
	err = os.WriteFile(coverPath, coverData, 0644)
	if err != nil {
		t.Fatalf("Failed to create cover image: %v", err)
	}

	// Test markdown with cover reference
	markdownContent := `# Book with Cover

Cover: cover.jpg

## Chapter 1

Content with cover.
`

	chapters, metadata, foundCoverPath, err := converter.parseMarkdown(markdownContent, tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify cover path
	if foundCoverPath == "" {
		t.Error("Expected cover path to be found")
	}

	if foundCoverPath != coverPath {
		t.Errorf("Expected cover path '%s', got '%s'", coverPath, foundCoverPath)
	}

	// Verify metadata includes cover data
	if len(metadata.Cover) == 0 {
		t.Error("Expected cover data in metadata")
	}
}

func TestMarkdownToEPUBConverter_ParseMarkdownFormatting(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test markdown with various formatting
	markdownContent := `# Formatting Test

## Chapter 1

This paragraph contains **bold text** and *italic text*.

- List item 1
- List item 2
- List item 3

### Code Section

Some inline ` + "`" + `code` + "`" + ` and a code block:

` + "``" + `
code line 1
code line 2
` + "``" + `

### Links and Images

[Link text](https://example.com)

---

Horizontal separator.

## Chapter 2

More content.
`

	chapters, metadata, coverPath, err := converter.parseMarkdown(markdownContent, "")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify chapters were parsed
	if len(chapters) == 0 {
		t.Error("No chapters parsed")
	}

	// Find content chapters
	var chapter1, chapter2 *ChapterData
	for _, chapter := range chapters {
		if chapter.Title == "Chapter 1" {
			chapter1 = chapter
		} else if chapter.Title == "Chapter 2" {
			chapter2 = chapter
		}
	}

	if chapter1 == nil {
		t.Error("Chapter 1 not found")
	}

	if chapter2 == nil {
		t.Error("Chapter 2 not found")
	}

	// Verify chapter 1 content contains formatting
	if !strings.Contains(chapter1.Content, "bold text") {
		t.Error("Bold text not preserved in chapter 1")
	}

	if !strings.Contains(chapter1.Content, "italic text") {
		t.Error("Italic text not preserved in chapter 1")
	}

	if !strings.Contains(chapter1.Content, "List item") {
		t.Error("List items not preserved in chapter 1")
	}

	if !strings.Contains(chapter1.Content, "inline code") {
		t.Error("Inline code not preserved in chapter 1")
	}

	if !strings.Contains(chapter1.Content, "code line") {
		t.Error("Code block not preserved in chapter 1")
	}

	if !strings.Contains(chapter1.Content, "Link text") {
		t.Error("Link not preserved in chapter 1")
	}
}

func TestMarkdownToEPUBConverter_ConvertMarkdownToEPUB(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "convert_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test markdown file
	mdPath := filepath.Join(tmpDir, "test.md")
	epubPath := filepath.Join(tmpDir, "test.epub")

	markdownContent := `# Test EPUB Conversion

Author: Test Author
Language: sr

## Chapter 1

This is chapter 1 content.

## Chapter 2

This is chapter 2 content with **bold text**.
`

	err = os.WriteFile(mdPath, []byte(markdownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test markdown: %v", err)
	}

	// Test conversion
	converter := NewMarkdownToEPUBConverter()
	err = converter.ConvertMarkdownToEPUB(mdPath, epubPath)
	if err != nil {
		t.Fatalf("Failed to convert markdown to EPUB: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Error("Output EPUB file was not created")
	}

	// Verify file size is reasonable
	fileInfo, err := os.Stat(epubPath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("Output EPUB file is empty")
	}

	// Basic EPUB validation - should contain ZIP signature
	file, err := os.Open(epubPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	// Read first 4 bytes to check for ZIP signature
	signature := make([]byte, 4)
	_, err = file.Read(signature)
	if err != nil {
		t.Fatalf("Failed to read file signature: %v", err)
	}

	// ZIP files start with "PK" (0x50 0x4B)
	if signature[0] != 0x50 || signature[1] != 0x4B {
		t.Error("Output file is not a valid ZIP/EPUB file")
	}
}

func TestMarkdownToEPUBConverter_CreateEPUBStructure(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test creating EPUB structure
	metadata := ebook.Metadata{
		Title:      "Test Book",
		Author:     "Test Author",
		Language:   "sr",
		Publisher:  "Test Publisher",
		Identifier: "test-book-123",
	}

	chapters := []*ChapterData{
		{
			Title:   "Chapter 1",
			Content: "<h1>Chapter 1</h1><p>Chapter 1 content.</p>",
			Level:   2,
		},
		{
			Title:   "Chapter 2",
			Content: "<h1>Chapter 2</h1><p>Chapter 2 content.</p>",
			Level:   2,
		},
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "epub_structure_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	epubPath := filepath.Join(tmpDir, "test.epub")

	// Test EPUB creation
	err = converter.createEPUBStructure(epubPath, metadata, chapters)
	if err != nil {
		t.Fatalf("Failed to create EPUB structure: %v", err)
	}

	// Verify EPUB file was created
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Error("EPUB file was not created")
	}

	// Verify it's a valid ZIP file
	file, err := os.Open(epubPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB file: %v", err)
	}
	defer file.Close()

	// Check ZIP signature
	signature := make([]byte, 4)
	_, err = file.Read(signature)
	if err != nil {
		t.Fatalf("Failed to read ZIP signature: %v", err)
	}

	if signature[0] != 0x50 || signature[1] != 0x4B {
		t.Error("Created file is not a valid ZIP/EPUB file")
	}
}

func TestMarkdownToEPUBConverter_CreateChapterXHTML(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test creating chapter XHTML
	chapter := &ChapterData{
		Title:   "Test Chapter",
		Content: "<p>Chapter content with some text.</p>",
		Level:   2,
	}

	// Generate XHTML
	xhtml, err := converter.createChapterXHTML(chapter, 1)
	if err != nil {
		t.Fatalf("Failed to create chapter XHTML: %v", err)
	}

	// Verify XHTML structure
	if !strings.Contains(xhtml, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE declaration")
	}

	if !strings.Contains(xhtml, "<html") {
		t.Error("Missing HTML tag")
	}

	if !strings.Contains(xhtml, "<head>") {
		t.Error("Missing HEAD tag")
	}

	if !strings.Contains(xhtml, "<body>") {
		tError("Missing BODY tag")
	}

	if !strings.Contains(xhtml, chapter.Content) {
		t.Error("Chapter content not included in XHTML")
	}

	if !strings.Contains(xhtml, chapter.Title) {
		t.Error("Chapter title not included in XHTML")
	}
}

func TestMarkdownToEPUBConverter_ErrorHandling(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test conversion with non-existent file
	err := converter.ConvertMarkdownToEPUB("/nonexistent/file.md", "/tmp/output.epub")
	if err == nil {
		t.Error("Expected error when converting non-existent file")
	}

	if !strings.Contains(err.Error(), "failed to read markdown") {
		t.Errorf("Expected markdown read error, got: %v", err)
	}

	// Test parsing empty content
	_, _, _, err = converter.parseMarkdown("", "")
	if err != nil {
		t.Errorf("Unexpected error parsing empty content: %v", err)
	}

	// Test parsing nil content
	_, _, _, err = converter.parseMarkdown("", "")
	if err != nil {
		t.Errorf("Unexpected error parsing nil content: %v", err)
	}
}

func TestMarkdownToEPUBConverter_CreateContentOPF(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	metadata := ebook.Metadata{
		Title:      "Test Book",
		Author:     "Test Author",
		Language:   "sr",
		Publisher:  "Test Publisher",
		Identifier: "test-book-123",
	}

	chapters := []*ChapterData{
		{Title: "Chapter 1", Level: 2},
		{Title: "Chapter 2", Level: 2},
	}

	// Test creating content OPF
	opf, err := converter.createContentOPF(metadata, chapters)
	if err != nil {
		t.Fatalf("Failed to create content OPF: %v", err)
	}

	// Verify OPF structure
	if !strings.Contains(opf, "<?xml") {
		t.Error("Missing XML declaration")
	}

	if !strings.Contains(opf, "<package") {
		t.Error("Missing package tag")
	}

	if !strings.Contains(opf, "<metadata>") {
		t.Error("Missing metadata section")
	}

	if !strings.Contains(opf, "<manifest>") {
		t.Error("Missing manifest section")
	}

	if !strings.Contains(opf, "<spine>") {
		t.Error("Missing spine section")
	}

	if !strings.Contains(opf, metadata.Title) {
		t.Error("Title not included in OPF")
	}

	if !strings.Contains(opf, metadata.Author) {
		t.Error("Author not included in OPF")
	}

	// Verify chapters are referenced
	for i, chapter := range chapters {
		chapterID := fmt.Sprintf("chapter%d.xhtml", i+1)
		if !strings.Contains(opf, chapterID) {
			t.Errorf("Chapter %d not referenced in OPF", i+1)
		}
	}
}

func TestMarkdownToEPUBConverter_CreateNCX(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	chapters := []*ChapterData{
		{Title: "Chapter 1", Level: 2},
		{Title: "Chapter 2", Level: 2},
		{Title: "Section 1.1", Level: 3},
	}

	// Test creating NCX
	ncx, err := converter.createNCX(chapters)
	if err != nil {
		t.Fatalf("Failed to create NCX: %v", err)
	}

	// Verify NCX structure
	if !strings.Contains(ncx, "<?xml") {
		t.Error("Missing XML declaration")
	}

	if !strings.Contains(ncx, "<ncx") {
		t.Error("Missing NCX tag")
	}

	if !strings.Contains(ncx, "<navMap>") {
		t.Error("Missing navMap section")
	}

	// Verify chapters are included
	for _, chapter := range chapters {
		if !strings.Contains(ncx, chapter.Title) {
			t.Errorf("Chapter '%s' not included in NCX", chapter.Title)
		}
	}

	// Verify nesting structure
	if !strings.Contains(ncx, "navPoint") {
		t.Error("Missing navPoint elements")
	}
}

// Helper function for test validation
func TestMarkdownToEPUBConverter_validateChapterStructure(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	// Test markdown with complex structure
	markdownContent := `# Main Title

## Chapter 1

Content here.

### Subsection 1.1

Subsection content.

#### Sub-subsection

Deep content.

## Chapter 2

More content.
`

	chapters, _, _, err := converter.parseMarkdown(markdownContent, "")
	if err != nil {
		t.Fatalf("Failed to parse markdown: %v", err)
	}

	// Verify hierarchy
	var level1Count, level2Count, level3Count int
	for _, chapter := range chapters {
		switch chapter.Level {
		case 1:
			level1Count++
		case 2:
			level2Count++
		case 3:
			level3Count++
		}
	}

	if level2Count < 2 {
		t.Errorf("Expected at least 2 level 2 chapters, got %d", level2Count)
	}

	if level3Count < 1 {
		t.Errorf("Expected at least 1 level 3 chapter, got %d", level3Count)
	}

	// Verify content preservation
	for _, chapter := range chapters {
		if len(chapter.Content) == 0 {
			t.Errorf("Chapter '%s' has no content", chapter.Title)
		}
	}
}

// Benchmark tests
func BenchmarkMarkdownToEPUBConverter_ParseMarkdown(b *testing.B) {
	converter := NewMarkdownToEPUBConverter()

	markdownContent := `# Test Book

## Chapter 1

This is test content for benchmarking.

### Section

More content.

## Chapter 2

More chapter content for benchmarking.

Additional content for parsing performance testing.
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.parseMarkdown(markdownContent, "")
	}
}

func BenchmarkMarkdownToEPUBConverter_ConvertMarkdownToEPUB(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_convert_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test markdown file
	mdPath := filepath.Join(tmpDir, "test.md")
	markdownContent := `# Benchmark Book

## Chapter 1

Test content for benchmarking.

## Chapter 2

More test content.
`

	err = os.WriteFile(mdPath, []byte(markdownContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test markdown: %v", err)
	}

	converter := NewMarkdownToEPUBConverter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		epubPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.epub", i))
		converter.ConvertMarkdownToEPUB(mdPath, epubPath)
	}
}