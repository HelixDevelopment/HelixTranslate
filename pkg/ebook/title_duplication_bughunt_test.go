package ebook

import (
	"os"
	"strings"
	"testing"
)

// TestBugHunt_HTMLTitleDuplication is the §11.4.115 reproduce-first / §11.4.135
// standing guard for MINOR-W6-1 in the HTML parser: <title>X</title> in <head>
// AND <h1>X</h1> in <body> both leaked into Section.Content, so the chapter title
// X appeared TWICE inside Content (and a third time when bookToString prepends
// chapter.Title). Root cause:
// docs/qa/minor_w6_1_rootcause_20260616-151123/FINDING.md.
//
// Pre-fix (RED_MODE=1): Content contains the title >= 1 time (leaked).
// Post-fix (RED_MODE=0, default standing guard): the title is carried EXACTLY
// ONCE in chapter.Title and ZERO times in Content; the real body text survives.
func TestBugHunt_HTMLTitleDuplication(t *testing.T) {
	const title = "La Farola"
	const body = "The lighthouse guided the ships."
	red := os.Getenv("RED_MODE") == "1"

	htmlDoc := `<!DOCTYPE html><html><head><title>` + title + `</title></head>` +
		`<body><h1>` + title + `</h1><p>` + body + `</p></body></html>`

	tmp, err := os.CreateTemp(t.TempDir(), "title_dup_*.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmp.WriteString(htmlDoc); err != nil {
		t.Fatal(err)
	}
	_ = tmp.Close()

	book, err := NewHTMLParser().Parse(tmp.Name())
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(book.Chapters) != 1 || len(book.Chapters[0].Sections) != 1 {
		t.Fatalf("unexpected structure: %+v", book.Chapters)
	}
	content := book.Chapters[0].Sections[0].Content
	titleField := book.Chapters[0].Title
	n := strings.Count(content, title)

	// The real body text must always survive.
	if !strings.Contains(content, body) {
		t.Fatalf("body text lost from Content: %q", content)
	}
	// The chapter title must be carried in the Title field exactly once.
	if titleField != title {
		t.Fatalf("chapter Title = %q, want %q", titleField, title)
	}

	if red {
		// Pre-fix reproduction: the title leaked into Content at least once.
		if n < 1 {
			t.Fatalf("PRE-FIX expected the title to leak into Content >=1 time, got %d: %q", n, content)
		}
		return
	}
	// Standing GREEN guard: the title must NOT appear in Content (carried once in Title).
	if n != 0 {
		t.Fatalf("title leaked into Content %d time(s) (want 0): %q", n, content)
	}
}

// TestBugHunt_EPUBTitleDuplication is the EPUB sibling guard. The <head> is
// already stripped, so the residual leak is the <h1> body copy. Same polarity.
func TestBugHunt_EPUBTitleDuplication(t *testing.T) {
	const title = "La Farola"
	const body = "The lighthouse guided the ships."
	red := os.Getenv("RED_MODE") == "1"

	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`
	contentOPF := `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>` + title + `</dc:title></metadata>
	<manifest><item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/></manifest>
	<spine><itemref idref="c1"/></spine>
</package>`
	chapter := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>` + title + `</title>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
</head>
<body>
  <h1>` + title + `</h1>
  <p>` + body + `</p>
</body>
</html>`

	files := []zipFile{
		{Name: "META-INF/container.xml", Content: containerXML},
		{Name: "OEBPS/content.opf", Content: contentOPF},
		{Name: "OEBPS/ch1.xhtml", Content: chapter},
	}
	tmpFile, err := createTempZipFile(t, "title_dup.epub", files)
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
	content := book.Chapters[0].Sections[0].Content
	titleField := book.Chapters[0].Title
	n := strings.Count(content, title)

	if !strings.Contains(content, body) {
		t.Fatalf("body text lost from Content: %q", content)
	}
	if titleField != title {
		t.Fatalf("chapter Title = %q, want %q", titleField, title)
	}

	if red {
		if n < 1 {
			t.Fatalf("PRE-FIX expected the title to leak into Content >=1 time, got %d: %q", n, content)
		}
		return
	}
	if n != 0 {
		t.Fatalf("title leaked into Content %d time(s) (want 0): %q", n, content)
	}
}

// TestStripLeadingTitle_NonDupNoOp guards that the title-strip is a strict no-op
// when Content does not begin with the title (plain content, different first
// line, or a longer word with the title as a prefix). DOCX/PDF/FB2 paths that
// never duplicate must stay byte-identical.
func TestStripLeadingTitle_NonDupNoOp(t *testing.T) {
	cases := []struct {
		name, content, title, want string
	}{
		{"empty title", "Body text here.", "", "Body text here."},
		{"different first line", "Other heading\n\nBody.", "Chapter One", "Other heading\n\nBody."},
		{"title is word-prefix only", "Caterpillar crawled.", "Cat", "Caterpillar crawled."},
		{"leading title newline-sep", "La Farola\n\nThe lighthouse.", "La Farola", "The lighthouse."},
		{"leading title space-sep", "La Farola The lighthouse.", "La Farola", "The lighthouse."},
		{"only the title", "La Farola", "La Farola", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripLeadingTitle(tc.content, tc.title)
			if got != tc.want {
				t.Fatalf("stripLeadingTitle(%q,%q) = %q, want %q", tc.content, tc.title, got, tc.want)
			}
		})
	}
}
