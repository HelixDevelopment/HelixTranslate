package ebook

import (
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/fb2"
	"digital.vasic.translator/pkg/format"
)

func TestNewFB2Parser(t *testing.T) {
	parser := NewFB2Parser()
	if parser == nil {
		t.Fatal("NewFB2Parser returned nil")
	}
}

func TestFB2Parser_GetFormat(t *testing.T) {
	parser := NewFB2Parser()
	if parser.GetFormat() != format.FormatFB2 {
		t.Errorf("GetFormat() = %s, want %s", parser.GetFormat(), format.FormatFB2)
	}
}

func TestFB2Parser_Parse_NonExistentFile(t *testing.T) {
	parser := NewFB2Parser()
	_, err := parser.Parse("/non/existent/file.fb2")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFB2Parser_Parse_InvalidFile(t *testing.T) {
	parser := NewFB2Parser()

	// Create a temporary file with invalid FB2 content
	tmpFile, err := os.CreateTemp("", "test_invalid.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	invalidContent := "This is not valid FB2 XML content"
	if _, err := tmpFile.WriteString(invalidContent); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = parser.Parse(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid FB2 file")
	}
}

func TestFB2Parser_Parse_ValidFile(t *testing.T) {
	parser := NewFB2Parser()

	// Create a minimal valid FB2 file
	tmpFile, err := os.CreateTemp("", "test_valid.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	validFB2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
	<description>
		<title-info>
			<author>
				<first-name>Test</first-name>
				<last-name>Author</last-name>
			</author>
			<book-title>Test Book</book-title>
			<lang>en</lang>
		</title-info>
	</description>
	<body>
		<section>
			<title>
				<p>Chapter 1</p>
			</title>
			<p>This is the first paragraph.</p>
			<p>This is the second paragraph.</p>
		</section>
		<section>
			<title>
				<p>Chapter 2</p>
			</title>
			<p>Content of chapter 2.</p>
		</section>
	</body>
</FictionBook>`

	if _, err := tmpFile.WriteString(validFB2Content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	book, err := parser.Parse(tmpFile.Name())
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

	// Check chapters
	if len(book.Chapters) != 2 {
		t.Errorf("Chapters count = %d, want 2", len(book.Chapters))
	}

	if book.Chapters[0].Title != "Chapter 1" {
		t.Errorf("First chapter title = %s, want Chapter 1", book.Chapters[0].Title)
	}

	if book.Chapters[1].Title != "Chapter 2" {
		t.Errorf("Second chapter title = %s, want Chapter 2", book.Chapters[1].Title)
	}

	// Check content
	if len(book.Chapters[0].Sections) == 0 {
		t.Error("First chapter has no sections")
	}

	content := book.Chapters[0].Sections[0].Content
	if !strings.Contains(content, "This is the first paragraph.") {
		t.Error("First paragraph not found in content")
	}

	if !strings.Contains(content, "This is the second paragraph.") {
		t.Error("Second paragraph not found in content")
	}
}

func TestFB2Parser_Parse_WithMiddleName(t *testing.T) {
	parser := NewFB2Parser()

	tmpFile, err := os.CreateTemp("", "test_middle.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
	<description>
		<title-info>
			<author>
				<first-name>John</first-name>
				<middle-name>Robert</middle-name>
				<last-name>Smith</last-name>
			</author>
			<book-title>Test Book</book-title>
			<lang>en</lang>
		</title-info>
	</description>
	<body>
		<section>
			<p>Content</p>
		</section>
	</body>
</FictionBook>`

	if _, err := tmpFile.WriteString(fb2Content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	book, err := parser.Parse(tmpFile.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(book.Metadata.Authors) != 1 {
		t.Fatalf("Authors count = %d, want 1", len(book.Metadata.Authors))
	}

	expectedAuthor := "John Robert Smith"
	if book.Metadata.Authors[0] != expectedAuthor {
		t.Errorf("Author = %s, want %s", book.Metadata.Authors[0], expectedAuthor)
	}
}

func TestFB2Parser_Parse_EmptyAuthorFields(t *testing.T) {
	parser := NewFB2Parser()

	tmpFile, err := os.CreateTemp("", "test_empty_author.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
	<description>
		<title-info>
			<author>
				<first-name></first-name>
				<last-name></last-name>
			</author>
			<book-title>Test Book</book-title>
			<lang>en</lang>
		</title-info>
	</description>
	<body>
		<section>
			<p>Content</p>
		</section>
	</body>
</FictionBook>`

	if _, err := tmpFile.WriteString(fb2Content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	book, err := parser.Parse(tmpFile.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have no authors since all fields are empty
	if len(book.Metadata.Authors) != 0 {
		t.Errorf("Authors count = %d, want 0", len(book.Metadata.Authors))
	}
}

func TestFB2Parser_Parse_NestedSections(t *testing.T) {
	parser := NewFB2Parser()

	tmpFile, err := os.CreateTemp("", "test_nested.fb2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
	<description>
		<title-info>
			<author>
				<first-name>Test</first-name>
				<last-name>Author</last-name>
			</author>
			<book-title>Test Book</book-title>
			<lang>en</lang>
		</title-info>
	</description>
	<body>
		<section>
			<title>
				<p>Main Chapter</p>
			</title>
			<p>Main content.</p>
			<section>
				<title>
					<p>Subsection</p>
				</title>
				<p>Subsection content.</p>
			</section>
		</section>
	</body>
</FictionBook>`

	if _, err := tmpFile.WriteString(fb2Content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	book, err := parser.Parse(tmpFile.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(book.Chapters) != 1 {
		t.Fatalf("Chapters count = %d, want 1", len(book.Chapters))
	}

	mainChapter := book.Chapters[0]
	if mainChapter.Title != "Main Chapter" {
		t.Errorf("Main chapter title = %s, want Main Chapter", mainChapter.Title)
	}

	if len(mainChapter.Sections) == 0 {
		t.Fatal("Main chapter has no sections")
	}

	mainSection := mainChapter.Sections[0]
	if !strings.Contains(mainSection.Content, "Main content.") {
		t.Error("Main content not found")
	}

	if len(mainSection.Subsections) == 0 {
		t.Error("No subsections found")
		return
	}

	subsection := mainSection.Subsections[0]
	if subsection.Title != "Subsection" {
		t.Errorf("Subsection title = %s, want Subsection", subsection.Title)
	}

	if !strings.Contains(subsection.Content, "Subsection content.") {
		t.Error("Subsection content not found")
	}
}

func TestConvertFB2Section(t *testing.T) {
	// Create a mock FB2 section
	fb2Section := &fb2.Section{
		Title: fb2.Title{
			Paragraphs: []fb2.Paragraph{
				{Text: "Test Chapter"},
			},
		},
		Paragraph: []fb2.Paragraph{
			{Text: "First paragraph."},
			{Text: "Second paragraph."},
		},
		Section: []fb2.Section{
			{
				Title: fb2.Title{
					Paragraphs: []fb2.Paragraph{
						{Text: "Subsection"},
					},
				},
				Paragraph: []fb2.Paragraph{
					{Text: "Subsection content."},
				},
			},
		},
	}

	chapter := convertFB2Section(fb2Section)

	if chapter.Title != "Test Chapter" {
		t.Errorf("Chapter title = %s, want Test Chapter", chapter.Title)
	}

	if len(chapter.Sections) == 0 {
		t.Fatal("Chapter has no sections")
	}

	mainSection := chapter.Sections[0]
	if !strings.Contains(mainSection.Content, "First paragraph.") {
		t.Error("First paragraph not found in section content")
	}

	if !strings.Contains(mainSection.Content, "Second paragraph.") {
		t.Error("Second paragraph not found in section content")
	}

	if len(mainSection.Subsections) == 0 {
		t.Error("No subsections found")
		return
	}

	subsection := mainSection.Subsections[0]
	if subsection.Title != "Subsection" {
		t.Errorf("Subsection title = %s, want Subsection", subsection.Title)
	}

	if !strings.Contains(subsection.Content, "Subsection content.") {
		t.Error("Subsection content not found")
	}
}

func TestConvertFB2Section_EmptyTitle(t *testing.T) {
	fb2Section := &fb2.Section{
		Title: fb2.Title{
			Paragraphs: []fb2.Paragraph{}, // Empty title
		},
		Paragraph: []fb2.Paragraph{
			{Text: "Content without title."},
		},
	}

	chapter := convertFB2Section(fb2Section)

	if chapter.Title != "" {
		t.Errorf("Chapter title = %s, want empty string", chapter.Title)
	}

	if len(chapter.Sections) == 0 {
		t.Fatal("Chapter has no sections")
	}

	section := chapter.Sections[0]
	if !strings.Contains(section.Content, "Content without title.") {
		t.Error("Content not found in section")
	}
}
