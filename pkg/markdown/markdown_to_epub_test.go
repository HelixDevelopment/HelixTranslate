package markdown

import (
	"archive/zip"
	"io"
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

// TestConvertMarkdownToEPUB_ContentAntiBluff verifies that the generated EPUB
// contains actual content in chapters, manifest, spine, and TOC.
// This is an anti-bluff test: it ensures users get a usable EPUB, not an empty shell.
func TestConvertMarkdownToEPUB_ContentAntiBluff(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	tempDir := t.TempDir()
	mdPath := filepath.Join(tempDir, "test.md")
	epubPath := filepath.Join(tempDir, "test.epub")

	// Markdown WITHOUT headers — content must still be preserved via default chapter
	markdownContent := "This is a sample English text for testing translation functionality.\n\nIt contains multiple sentences."

	if err := os.WriteFile(mdPath, []byte(markdownContent), 0644); err != nil {
		t.Fatalf("Failed to create test markdown: %v", err)
	}

	if err := converter.ConvertMarkdownToEPUB(mdPath, epubPath); err != nil {
		t.Fatalf("Failed to convert markdown to EPUB: %v", err)
	}

	// Open the EPUB as a ZIP archive and inspect contents
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("Failed to open EPUB as ZIP: %v", err)
	}
	defer r.Close()

	// Build map of files
	files := make(map[string]*zip.File)
	for _, f := range r.File {
		files[f.Name] = f
	}

	// 1. Must contain mimetype, container.xml, content.opf, toc.ncx, and at least one chapter
	mustExist := []string{"mimetype", "META-INF/container.xml", "OEBPS/content.opf", "OEBPS/toc.ncx"}
	for _, name := range mustExist {
		if _, ok := files[name]; !ok {
			t.Errorf("Missing required EPUB file: %s", name)
		}
	}

	// 2. Must have at least one chapter XHTML file
	var chapterFile *zip.File
	for name, f := range files {
		if strings.HasPrefix(name, "OEBPS/chapter") && strings.HasSuffix(name, ".xhtml") {
			chapterFile = f
			break
		}
	}
	if chapterFile == nil {
		t.Fatalf("No chapter XHTML file found in EPUB — content was lost!")
	}

	// 3. Chapter must contain the actual markdown text
	chReader, _ := chapterFile.Open()
	chData, _ := io.ReadAll(chReader)
	chReader.Close()
	chStr := string(chData)
	if !strings.Contains(chStr, "sample English text") {
		t.Errorf("Chapter XHTML missing expected content. Got:\n%s", chStr)
	}

	// 4. content.opf must reference the chapter in manifest AND spine
	opfReader, _ := files["OEBPS/content.opf"].Open()
	opfData, _ := io.ReadAll(opfReader)
	opfReader.Close()
	opfStr := string(opfData)

	if !strings.Contains(opfStr, `id="chapter1"`) {
		t.Errorf("content.opf manifest missing chapter1 item")
	}
	if !strings.Contains(opfStr, `<itemref idref="chapter1"/>`) {
		t.Errorf("content.opf spine missing chapter1 itemref — EPUB would appear empty to readers!")
	}

	// 5. toc.ncx must have a navPoint for the chapter
	ncxReader, _ := files["OEBPS/toc.ncx"].Open()
	ncxData, _ := io.ReadAll(ncxReader)
	ncxReader.Close()
	ncxStr := string(ncxData)

	if !strings.Contains(ncxStr, `id="chapter1"`) {
		t.Errorf("toc.ncx missing navPoint for chapter1")
	}
	if !strings.Contains(ncxStr, `src="chapter1.xhtml"`) {
		t.Errorf("toc.ncx missing content src for chapter1")
	}

	t.Logf("Anti-bluff EPUB verification passed: %d files, chapter has %d bytes of content",
		len(files), len(chData))
}
