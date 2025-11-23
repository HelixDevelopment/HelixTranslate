package ebook

import (
	"archive/zip"
	"bytes"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/format"
)

func TestNewEPUBParser(t *testing.T) {
	parser := NewEPUBParser()
	if parser == nil {
		t.Fatal("NewEPUBParser returned nil")
	}
}

func TestEPUBParser_GetFormat(t *testing.T) {
	parser := NewEPUBParser()
	if parser.GetFormat() != format.FormatEPUB {
		t.Errorf("GetFormat() = %s, want %s", parser.GetFormat(), format.FormatEPUB)
	}
}

func TestEPUBParser_Parse_NonExistentFile(t *testing.T) {
	parser := NewEPUBParser()
	_, err := parser.Parse("/non/existent/file.epub")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestEPUBParser_Parse_InvalidFile(t *testing.T) {
	parser := NewEPUBParser()

	// Create a temporary file with invalid EPUB content
	tmpFile, err := createTempZipFile(t, "test_invalid.epub", []zipFile{
		{Name: "not_container.xml", Content: "invalid content"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	_, err = parser.Parse(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid EPUB file")
	}
}

func TestEPUBParser_Parse_ValidFile(t *testing.T) {
	parser := NewEPUBParser()

	// Create a minimal valid EPUB file
	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles>
		<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
	</rootfiles>
</container>`

	contentOPF := `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="BookId">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
		<dc:title>Test Book</dc:title>
		<dc:creator>Test Author</dc:creator>
		<dc:language>en</dc:language>
		<dc:description>A test book description</dc:description>
		<dc:publisher>Test Publisher</dc:publisher>
		<dc:date>2023-01-01</dc:date>
		<dc:identifier id="BookId">urn:isbn:1234567890</dc:identifier>
	</metadata>
	<manifest>
		<item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
		<item id="chapter2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
		<item id="cover" href="cover.jpg" media-type="image/jpeg"/>
	</manifest>
	<spine>
		<itemref idref="chapter1"/>
		<itemref idref="chapter2"/>
	</spine>
</package>`

	chapter1XHTML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
<h1>Chapter 1</h1>
<p>This is the first chapter content.</p>
<p>More content here.</p>
</body>
</html>`

	chapter2XHTML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body>
<h1>Chapter 2</h1>
<p>This is the second chapter content.</p>
</body>
</html>`

	files := []zipFile{
		{Name: "META-INF/container.xml", Content: containerXML},
		{Name: "OEBPS/content.opf", Content: contentOPF},
		{Name: "OEBPS/chapter1.xhtml", Content: chapter1XHTML},
		{Name: "OEBPS/chapter2.xhtml", Content: chapter2XHTML},
		{Name: "OEBPS/cover.jpg", Content: "fake image content"},
	}

	tmpFile, err := createTempZipFile(t, "test_valid.epub", files)
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	book, err := parser.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if book == nil {
		t.Fatal("Book is nil")
	}

	// Check metadata
	if book.Metadata.Title != "Test Book" {
		t.Errorf("Title = %s, want Test Book", book.Metadata.Title)
	}

	if book.Metadata.Language != "en" {
		t.Errorf("Language = %s, want en", book.Metadata.Language)
	}

	if len(book.Metadata.Authors) != 1 {
		t.Errorf("Authors count = %d, want 1", len(book.Metadata.Authors))
	}

	if book.Metadata.Authors[0] != "Test Author" {
		t.Errorf("Author = %s, want Test Author", book.Metadata.Authors[0])
	}

	if book.Metadata.Description != "A test book description" {
		t.Errorf("Description = %s, want 'A test book description'", book.Metadata.Description)
	}

	if book.Metadata.Publisher != "Test Publisher" {
		t.Errorf("Publisher = %s, want Test Publisher", book.Metadata.Publisher)
	}

	if book.Metadata.Date != "2023-01-01" {
		t.Errorf("Date = %s, want 2023-01-01", book.Metadata.Date)
	}

	if book.Metadata.ISBN != "urn:isbn:1234567890" {
		t.Errorf("ISBN = %s, want urn:isbn:1234567890", book.Metadata.ISBN)
	}

	// Check chapters
	if len(book.Chapters) != 2 {
		t.Errorf("Chapters count = %d, want 2", len(book.Chapters))
	}

	// Check chapter content
	chapter1Content := book.Chapters[0].Sections[0].Content
	if !strings.Contains(chapter1Content, "This is the first chapter content.") {
		t.Error("Chapter 1 content not found")
	}

	chapter2Content := book.Chapters[1].Sections[0].Content
	if !strings.Contains(chapter2Content, "This is the second chapter content.") {
		t.Error("Chapter 2 content not found")
	}

	// Check cover image
	if book.Metadata.Cover == nil {
		t.Error("Cover image not extracted")
	} else {
		if !bytes.Equal(book.Metadata.Cover, []byte("fake image content")) {
			t.Error("Cover image content mismatch")
		}
	}
}

func TestEPUBParser_Parse_MissingContainer(t *testing.T) {
	parser := NewEPUBParser()

	// Create EPUB without container.xml
	files := []zipFile{
		{Name: "OEBPS/content.opf", Content: "<package></package>"},
	}

	tmpFile, err := createTempZipFile(t, "test_no_container.epub", files)
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	_, err = parser.Parse(tmpFile)
	if err == nil {
		t.Error("Expected error for missing container.xml")
	}

	if !strings.Contains(err.Error(), "container.xml not found") {
		t.Errorf("Expected 'container.xml not found' error, got: %v", err)
	}
}

func TestEPUBParser_CleanXMLData(t *testing.T) {
	parser := NewEPUBParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid XML",
			input:    "<root>valid content</root>",
			expected: "<root>valid content</root>",
		},
		{
			name:     "Invalid characters",
			input:    "<root>content\x00\x01\x02with invalid</root>",
			expected: "<root>contentwith invalid</root>",
		},
		{
			name:     "Invalid entities",
			input:    "<root>& &< &> &q &a &l &g</root>",
			expected: "<root>&amp; &lt; &gt; &quot; &amp; &lt; &gt;</root>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.CleanXMLData([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("CleanXMLData() = %s, want %s", string(result), tt.expected)
			}
		})
	}
}

func TestEPUBParser_parseContainer(t *testing.T) {
	parser := NewEPUBParser()

	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name: "Valid container",
			content: `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles>
		<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
	</rootfiles>
</container>`,
			expected: "OEBPS/content.opf",
			wantErr:  false,
		},
		{
			name:     "Invalid XML",
			content:  "invalid xml content",
			expected: "",
			wantErr:  true,
		},
		{
			name: "No rootfile",
			content: `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles>
	</rootfiles>
</container>`,
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a zip file with the container.xml
			files := []zipFile{
				{Name: "META-INF/container.xml", Content: tt.content},
			}

			tmpFile, err := createTempZipFile(t, "test_container.epub", files)
			if err != nil {
				t.Fatal(err)
			}
			defer removeTempFile(t, tmpFile)

			// Open the zip and get the container.xml file
			r, err := zip.OpenReader(tmpFile)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()

			var containerFile *zip.File
			for _, f := range r.File {
				if f.Name == "META-INF/container.xml" {
					containerFile = f
					break
				}
			}

			if containerFile == nil {
				t.Fatal("container.xml not found in zip")
			}

			result, err := parser.parseContainer(containerFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.expected {
				t.Errorf("parseContainer() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestRemoveHTMLTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple tags",
			input:    "<p>Hello <b>world</b></p>",
			expected: " Hello  world  ",  // 4 tags = 4 spaces
		},
		{
			name:     "Nested tags",
			input:    "<div><p>Nested <span>content</span></p></div>",
			expected: "  Nested  content   ",  // 6 tags = 6 spaces, but there were consecutive tags at start
		},
		{
			name:     "No tags",
			input:    "Plain text content",
			expected: "Plain text content",
		},
		{
			name:     "Empty tags",
			input:    "<p></p>",
			expected: "  ",  // 2 tags = 2 spaces
		},
		{
			name:     "Mixed content",
			input:    "Text before <tag>inside</tag> text after",
			expected: "Text before  inside  text after",  // 2 tags = 2 spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeHTMLTags(tt.input)
			if result != tt.expected {
				t.Errorf("removeHTMLTags() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestEPUBParser_parseContentFile(t *testing.T) {
	parser := NewEPUBParser()

	tests := []struct {
		name     string
		content  string
		wantErr  bool
		wantNil  bool
		expected string
	}{
		{
			name:     "Valid XHTML",
			content:  "<html><body><p>Test content</p></body></html>",
			wantErr:  false,
			wantNil:  false,
			expected: "Test content",
		},
		{
			name:     "Empty content",
			content:  "",
			wantErr:  false,
			wantNil:  true,
			expected: "",
		},
		{
			name:     "Only tags",
			content:  "<html><body></body></html>",
			wantErr:  false,
			wantNil:  true,
			expected: "",
		},
		{
			name:     "Complex HTML",
			content:  "<html><head><title>Title</title></head><body><h1>Heading</h1><p>Paragraph content.</p></body></html>",
			wantErr:  false,
			wantNil:  false,
			expected: "Heading Paragraph content.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a zip file with the content
			files := []zipFile{
				{Name: "test.xhtml", Content: tt.content},
			}

			tmpFile, err := createTempZipFile(t, "test_content.epub", files)
			if err != nil {
				t.Fatal(err)
			}
			defer removeTempFile(t, tmpFile)

			// Open the zip and get the content file
			r, err := zip.OpenReader(tmpFile)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()

			var contentFile *zip.File
			for _, f := range r.File {
				if f.Name == "test.xhtml" {
					contentFile = f
					break
				}
			}

			if contentFile == nil {
				t.Fatal("test.xhtml not found in zip")
			}

			chapter, err := parser.parseContentFile(contentFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseContentFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantNil && chapter != nil {
				t.Error("parseContentFile() returned non-nil chapter when nil was expected")
				return
			}

			if !tt.wantNil && chapter == nil {
				t.Error("parseContentFile() returned nil chapter when non-nil was expected")
				return
			}

			if !tt.wantNil && chapter != nil {
				content := chapter.Sections[0].Content
				if !strings.Contains(content, tt.expected) {
					t.Errorf("parseContentFile() content = %s, expected to contain %s", content, tt.expected)
				}
			}
		})
	}
}

// Helper functions for creating temporary zip files

type zipFile struct {
	Name    string
	Content string
}

func createTempZipFile(t *testing.T, filename string, files []zipFile) (string, error) {
	tmpFile, err := createTempEPUBFile(t, filename, "")
	if err != nil {
		return "", err
	}

	zipWriter := zip.NewWriter(tmpFile)
	defer zipWriter.Close()

	for _, file := range files {
		writer, err := zipWriter.Create(file.Name)
		if err != nil {
			return "", err
		}

		_, err = writer.Write([]byte(file.Content))
		if err != nil {
			return "", err
		}
	}

	return tmpFile.Name(), nil
}

func createTempEPUBFile(t *testing.T, filename, content string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", filename)
	if err != nil {
		return nil, err
	}

	if content != "" {
		_, err = tmpFile.WriteString(content)
		if err != nil {
			tmpFile.Close()
			return nil, err
		}
	}

	return tmpFile, nil
}

func removeTempFile(t *testing.T, filename string) {
	if err := os.Remove(filename); err != nil {
		t.Logf("Warning: failed to remove temp file %s: %v", filename, err)
	}
}
