package ebook

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"
)

// buildTestDOCX assembles a minimal valid OOXML .docx (a zip containing
// word/document.xml and, if coreXML != "", docProps/core.xml). zip.NewReader +
// the stdlib DOCX parser only need word/document.xml to be present.
func buildTestDOCX(t *testing.T, documentXML, coreXML string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create document.xml: %v", err)
	}
	if _, err := w.Write([]byte(documentXML)); err != nil {
		t.Fatalf("write document.xml: %v", err)
	}
	if coreXML != "" {
		cw, err := zw.Create("docProps/core.xml")
		if err != nil {
			t.Fatalf("zip create core.xml: %v", err)
		}
		if _, err := cw.Write([]byte(coreXML)); err != nil {
			t.Fatalf("write core.xml: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

const sampleDocumentXML = `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>First paragraph text.</w:t></w:r></w:p>
<w:p><w:r><w:t>Second </w:t></w:r><w:r><w:t>paragraph</w:t></w:r><w:r><w:t> runs joined.</w:t></w:r></w:p>
<w:p><w:r><w:t>Кириллица сохранена.</w:t></w:r></w:p>
</w:body>
</w:document>`

const sampleCoreXML = `<?xml version="1.0" encoding="UTF-8"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/">
<dc:title>My Doc Title</dc:title>
<dc:creator>Jane Doe</dc:creator>
<dc:description>A short description.</dc:description>
<dcterms:created>2024-01-02T03:04:05Z</dcterms:created>
</cp:coreProperties>`

// TestDOCXParser_ParsesRealDOCX_Stdlib proves the stdlib DOCX parser extracts
// real paragraph text + metadata from a genuine .docx. On the PRIOR (unioffice)
// implementation this FAILED for every real .docx with "unioffice license
// required" — DOCX input was non-functional. This is the reproduce-first guard
// for that capability (§11.4.115/§11.4.135).
func TestDOCXParser_ParsesRealDOCX_Stdlib(t *testing.T) {
	data := buildTestDOCX(t, sampleDocumentXML, sampleCoreXML)
	parser := NewDOCXParser(nil)

	book, err := parser.ParseWithContext(context.Background(), data)
	if err != nil {
		t.Fatalf("ParseWithContext failed on a valid .docx: %v", err)
	}

	// Content: all paragraph text present; runs within a paragraph joined; the
	// three paragraphs separated by the blank-line boundary.
	if len(book.Chapters) == 0 || len(book.Chapters[0].Sections) == 0 {
		t.Fatalf("no chapter/section produced")
	}
	content := book.Chapters[0].Sections[0].Content
	for _, want := range []string{
		"First paragraph text.",
		"Second paragraph runs joined.", // proves multi-run concatenation within a <w:p>
		"Кириллица сохранена.",          // proves UTF-8 preserved
	} {
		if !strings.Contains(content, want) {
			t.Errorf("extracted content missing %q\n--- content ---\n%s", want, content)
		}
	}
	if !strings.Contains(content, "First paragraph text.\n\nSecond") {
		t.Errorf("paragraph boundary (\\n\\n) not preserved between para 1 and 2:\n%s", content)
	}

	// Metadata from docProps/core.xml.
	if book.Metadata.Title != "My Doc Title" {
		t.Errorf("title: got %q want %q", book.Metadata.Title, "My Doc Title")
	}
	if len(book.Metadata.Authors) != 1 || book.Metadata.Authors[0] != "Jane Doe" {
		t.Errorf("authors: got %v want [Jane Doe]", book.Metadata.Authors)
	}
	if book.Metadata.Description != "A short description." {
		t.Errorf("description: got %q", book.Metadata.Description)
	}
	if book.Metadata.Date == "" {
		t.Errorf("date not extracted from dcterms:created")
	}

	// GetMetadata + Validate also work on the real artifact.
	if err := parser.Validate(data); err != nil {
		t.Errorf("Validate rejected a valid .docx: %v", err)
	}
	md, err := parser.GetMetadata(data)
	if err != nil || md.Title != "My Doc Title" {
		t.Errorf("GetMetadata: md=%+v err=%v", md, err)
	}
}

// TestDOCXParser_MinTextLengthFiltersShortParagraphs is the §11.4.135 RED test
// for the ATM-069 MinTextLength wiring. A DOCX with both short ("Hi") and long
// paragraphs is parsed with MinTextLength=5. The short paragraph MUST be absent
// from the output; the long paragraph MUST be present.
func TestDOCXParser_MinTextLengthFiltersShortParagraphs(t *testing.T) {
	// Build a DOCX with a short paragraph ("Hi" = 2 chars) and a long one.
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>Hi</w:t></w:r></w:p>
<w:p><w:r><w:t>This is a long enough paragraph for the filter.</w:t></w:r></w:p>
<w:p><w:r><w:t>Ok</w:t></w:r></w:p>
<w:p><w:r><w:t>Another sufficiently long paragraph that should survive the filter.</w:t></w:r></w:p>
</w:body>
</w:document>`

	data := buildTestDOCX(t, xml, "")

	config := &DOCXConfig{
		MinTextLength: 5,
		IgnoreStyles:  []string{},
	}
	parser := NewDOCXParser(config)

	book, err := parser.ParseWithContext(context.Background(), data)
	if err != nil {
		t.Fatalf("ParseWithContext failed: %v", err)
	}

	if len(book.Chapters) == 0 || len(book.Chapters[0].Sections) == 0 {
		t.Fatal("no chapter/section produced")
	}

	content := book.Chapters[0].Sections[0].Content

	// Short paragraphs ("Hi", "Ok") must be filtered out.
	if strings.Contains(content, "Hi") {
		t.Errorf("short paragraph 'Hi' should be filtered by MinTextLength=5, but found in:\n%s", content)
	}
	if strings.Contains(content, "\nOk\n") || strings.Contains(content, "\nOk") || content == "Ok" {
		t.Errorf("short paragraph 'Ok' should be filtered by MinTextLength=5, but found in:\n%s", content)
	}

	// Long paragraphs must survive.
	if !strings.Contains(content, "long enough paragraph") {
		t.Errorf("long paragraph should survive MinTextLength filter, but not found in:\n%s", content)
	}
	if !strings.Contains(content, "Another sufficiently long") {
		t.Errorf("second long paragraph should survive MinTextLength filter, but not found in:\n%s", content)
	}
}
