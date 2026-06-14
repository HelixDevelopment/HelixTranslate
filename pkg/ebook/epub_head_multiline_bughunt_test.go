package ebook

import (
	"strings"
	"testing"
)

// TestBugHunt_EPUB_MultilineHeadTitleLeaks is a reproduce-first RED test for the
// head-removal regex in epub_parser.parseContentFile. The pattern
// `(?i)<head[^>]*>.*?</head>` has NO `s` flag, so `.` does not match newlines —
// a multi-line <head> (the OVERWHELMINGLY common real-world XHTML shape: <title>,
// charset <meta>, stylesheet <link> on separate lines) is NOT removed. The head
// tags are later stripped by removeHTMLTags, but the <title> TEXT survives and
// leaks into the extracted chapter content, duplicating the chapter heading and
// polluting the translated output.
//
// Cyrillic title so the leak is also exercised on multi-byte content.
func TestBugHunt_EPUB_MultilineHeadTitleLeaks(t *testing.T) {
	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

	contentOPF := `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>T</dc:title></metadata>
	<manifest><item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/></manifest>
	<spine><itemref idref="c1"/></spine>
</package>`

	// Realistic multi-line head: title on its own line, then meta/link lines.
	chapter := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>ЗаголовокИзГоловы</title>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
  <link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
  <p>Настоящий текст главы.</p>
</body>
</html>`

	files := []zipFile{
		{Name: "META-INF/container.xml", Content: containerXML},
		{Name: "OEBPS/content.opf", Content: contentOPF},
		{Name: "OEBPS/ch1.xhtml", Content: chapter},
	}

	tmpFile, err := createTempZipFile(t, "test_mlhead.epub", files)
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	book, err := NewEPUBParser().Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(book.Chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(book.Chapters))
	}
	got := book.Chapters[0].Sections[0].Content

	// Real body text must survive.
	if !strings.Contains(got, "Настоящий текст главы.") {
		t.Fatalf("body text missing: %q", got)
	}
	// The defect: <title> text from a multi-line head leaked into chapter content.
	if strings.Contains(got, "ЗаголовокИзГоловы") {
		t.Errorf("multi-line <head> not removed — <title> text leaked into chapter content: %q", got)
	}
}
