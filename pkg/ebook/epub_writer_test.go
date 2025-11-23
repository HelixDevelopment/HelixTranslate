package ebook

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/format"
)

func TestNewEPUBWriter(t *testing.T) {
	writer := NewEPUBWriter()
	if writer == nil {
		t.Fatal("NewEPUBWriter returned nil")
	}
}

func TestEPUBWriter_Write_SimpleBook(t *testing.T) {
	writer := NewEPUBWriter()

	book := &Book{
		Metadata: Metadata{
			Title:    "Test Book",
			Authors:  []string{"Test Author"},
			Language: "en",
		},
		Chapters: []Chapter{
			{
				Title: "Chapter 1",
				Sections: []Section{
					{
						Title:   "Section 1",
						Content: "This is the first section content.",
					},
				},
			},
		},
		Format: format.FormatEPUB,
	}

	tmpFile := createTempEPUBWriterFile(t, "test_simple.epub")
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify the EPUB file structure
	verifyEPUBStructure(t, tmpFile, book)
}

func TestEPUBWriter_Write_BookWithCover(t *testing.T) {
	writer := NewEPUBWriter()

	book := &Book{
		Metadata: Metadata{
			Title:    "Book with Cover",
			Authors:  []string{"Cover Author"},
			Language: "en",
			Cover:    []byte("fake cover image data"),
		},
		Chapters: []Chapter{
			{
				Title: "Chapter 1",
				Sections: []Section{
					{
						Content: "Content with cover image.",
					},
				},
			},
		},
		Format: format.FormatEPUB,
	}

	tmpFile := createTempEPUBWriterFile(t, "test_cover.epub")
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify cover is included
	verifyCoverIncluded(t, tmpFile)
}

func TestEPUBWriter_Write_ComplexBook(t *testing.T) {
	writer := NewEPUBWriter()

	book := &Book{
		Metadata: Metadata{
			Title:       "Complex Book",
			Authors:     []string{"Author 1", "Author 2"},
			Description: "A complex book with multiple chapters and sections",
			Publisher:   "Test Publisher",
			Language:    "en",
			ISBN:        "1234567890",
			Date:        "2023-01-01",
		},
		Chapters: []Chapter{
			{
				Title: "Chapter 1",
				Sections: []Section{
					{
						Title:   "Section 1.1",
						Content: "First section content.\n\nSecond paragraph.",
						Subsections: []Section{
							{
								Title:   "Subsection 1.1.1",
								Content: "Nested content here.",
							},
						},
					},
					{
						Title:   "Section 1.2",
						Content: "Second section content.",
					},
				},
			},
			{
				Title: "Chapter 2",
				Sections: []Section{
					{
						Content: "Chapter 2 content without section title.",
					},
				},
			},
		},
		Format: format.FormatEPUB,
	}

	tmpFile := createTempEPUBWriterFile(t, "test_complex.epub")
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify complex structure
	verifyComplexEPUBStructure(t, tmpFile, book)
}

func TestEPUBWriter_Write_EmptyBook(t *testing.T) {
	writer := NewEPUBWriter()

	book := &Book{
		Metadata: Metadata{
			Title: "Empty Book",
		},
		Chapters: []Chapter{},
		Format:   format.FormatEPUB,
	}

	tmpFile := createTempEPUBWriterFile(t, "test_empty.epub")
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify empty book structure
	verifyEmptyEPUBStructure(t, tmpFile, book)
}

func TestEPUBWriter_Write_InvalidPath(t *testing.T) {
	writer := NewEPUBWriter()

	book := &Book{
		Metadata: Metadata{Title: "Test"},
		Chapters: []Chapter{},
		Format:   format.FormatEPUB,
	}

	// Try to write to an invalid path
	err := writer.Write(book, "/invalid/path/test.epub")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Normal text",
			expected: "Normal text",
		},
		{
			input:    "Text with & ampersand",
			expected: "Text with &amp; ampersand",
		},
		{
			input:    "Text with < less than",
			expected: "Text with &lt; less than",
		},
		{
			input:    "Text with > greater than",
			expected: "Text with &gt; greater than",
		},
		{
			input:    "Text with \" quotes",
			expected: "Text with &quot; quotes",
		},
		{
			input:    "Text with ' apostrophe",
			expected: "Text with &apos; apostrophe",
		},
		{
			input:    "Mixed & < > \" ' characters",
			expected: "Mixed &amp; &lt; &gt; &quot; &apos; characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeXML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeXML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()

	if uuid1 == "" {
		t.Error("generateUUID() returned empty string")
	}

	if uuid2 == "" {
		t.Error("generateUUID() returned empty string on second call")
	}

	if uuid1 == uuid2 {
		t.Error("generateUUID() returned same UUID twice")
	}

	// Check format (should contain dash)
	if !strings.Contains(uuid1, "-") {
		t.Errorf("UUID format incorrect: %s", uuid1)
	}
}

func TestFormatSection(t *testing.T) {
	writer := NewEPUBWriter()

	tests := []struct {
		name     string
		section  *Section
		expected string
	}{
		{
			name: "Simple section",
			section: &Section{
				Title:   "Test Section",
				Content: "This is test content.",
			},
			expected: "<h2>Test Section</h2>\n  <p>This is test content.</p>\n",
		},
		{
			name: "Section with multiple paragraphs",
			section: &Section{
				Title:   "Multi Paragraph",
				Content: "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			},
			expected: "<h2>Multi Paragraph</h2>\n  <p>First paragraph.</p>\n  <p>Second paragraph.</p>\n  <p>Third paragraph.</p>\n",
		},
		{
			name: "Section without title",
			section: &Section{
				Content: "Content without title.",
			},
			expected: "  <p>Content without title.</p>\n",
		},
		{
			name: "Section with subsections",
			section: &Section{
				Title:   "Main Section",
				Content: "Main content.",
				Subsections: []Section{
					{
						Title:   "Subsection",
						Content: "Subsection content.",
					},
				},
			},
			expected: "<h2>Main Section</h2>\n  <p>Main content.</p>\n<h2>Subsection</h2>\n  <p>Subsection content.</p>\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := writer.formatSection(tt.section)
			if result != tt.expected {
				t.Errorf("formatSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Helper functions

func createTempEPUBWriterFile(t *testing.T, filename string) string {
	tmpFile, err := os.CreateTemp("", filename)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func verifyEPUBStructure(t *testing.T, filename string, book *Book) {
	// Open the EPUB file
	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	// Check required files
	requiredFiles := []string{
		"mimetype",
		"META-INF/container.xml",
		"OEBPS/content.opf",
		"OEBPS/toc.ncx",
	}

	for _, required := range requiredFiles {
		found := false
		for _, f := range r.File {
			if f.Name == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required file %s not found in EPUB", required)
		}
	}

	// Check chapter files
	expectedChapterFiles := len(book.Chapters)
	if expectedChapterFiles == 0 {
		expectedChapterFiles = 1 // At least one chapter file should exist
	}

	chapterCount := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "OEBPS/chapter") && strings.HasSuffix(f.Name, ".xhtml") {
			chapterCount++
		}
	}

	if chapterCount < expectedChapterFiles {
		t.Errorf("Expected at least %d chapter files, found %d", expectedChapterFiles, chapterCount)
	}
}

func verifyCoverIncluded(t *testing.T, filename string) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	found := false
	for _, f := range r.File {
		if f.Name == "OEBPS/cover.jpg" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Cover file not found in EPUB")
	}
}

func verifyComplexEPUBStructure(t *testing.T, filename string, book *Book) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	// Check content.opf contains all metadata
	var contentOPF *zip.File
	for _, f := range r.File {
		if f.Name == "OEBPS/content.opf" {
			contentOPF = f
			break
		}
	}

	if contentOPF == nil {
		t.Fatal("content.opf not found")
	}

	rc, err := contentOPF.Open()
	if err != nil {
		t.Fatalf("Failed to open content.opf: %v", err)
	}
	defer rc.Close()

	// Read the content.opf content
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("Failed to read content.opf: %v", err)
	}

	content := string(data)

	// Check metadata is included
	if !strings.Contains(content, book.Metadata.Title) {
		t.Error("Book title not found in EPUB")
	}

	for _, author := range book.Metadata.Authors {
		if !strings.Contains(content, author) {
			t.Errorf("Author %s not found in EPUB", author)
		}
	}

	if book.Metadata.Description != "" && !strings.Contains(content, book.Metadata.Description) {
		t.Error("Description not found in EPUB")
	}

	if book.Metadata.Publisher != "" && !strings.Contains(content, book.Metadata.Publisher) {
		t.Error("Publisher not found in EPUB")
	}
}

func verifyEmptyEPUBStructure(t *testing.T, filename string, book *Book) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()

	// Even empty books should have basic structure
	requiredFiles := []string{
		"mimetype",
		"META-INF/container.xml",
		"OEBPS/content.opf",
		"OEBPS/toc.ncx",
	}

	for _, required := range requiredFiles {
		found := false
		for _, f := range r.File {
			if f.Name == required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Required file %s not found in empty EPUB", required)
		}
	}
}
