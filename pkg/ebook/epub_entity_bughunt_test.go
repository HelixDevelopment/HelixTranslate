package ebook

import (
	"archive/zip"
	"strings"
	"testing"
)

// epubChapterText builds a minimal EPUB with a single chapter whose body is the
// supplied xhtml fragment, parses it, and returns the extracted chapter text.
func epubChapterText(t *testing.T, bodyXHTML string) string {
	t.Helper()

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

	chapter := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>C</title></head><body>` + bodyXHTML + `</body></html>`

	files := []zipFile{
		{Name: "META-INF/container.xml", Content: containerXML},
		{Name: "OEBPS/content.opf", Content: contentOPF},
		{Name: "OEBPS/ch1.xhtml", Content: chapter},
	}

	tmpFile, err := createTempZipFile(t, "test_entity.epub", files)
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	parser := NewEPUBParser()
	book, err := parser.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(book.Chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(book.Chapters))
	}
	return book.Chapters[0].Sections[0].Content
}

// BUG (FACT, reproduced): EPUB chapter text extraction (parseContentFile ->
// removeHTMLTags in epub_parser.go) strips TAGS but never DECODES HTML/XML
// character references. Every entity in the chapter body — named (&amp; &lt;
// &gt; &quot; &apos;) and numeric (&#233; &#x2014;) — survives verbatim as
// literal markup in the extracted text that is then sent to the translator and
// written back out. A reader sees "Tom &amp; Jerry" instead of "Tom & Jerry"
// and "caf&#233;" instead of "café".
//
// Contrast: the standalone HTML parser (html_parser.go) goes through
// golang.org/x/net/html which decodes entities automatically. The EPUB path is
// the regression because it uses its own regex tag-stripper with no decode step.
//
// Root cause: epub_parser.go parseContentFile builds content from the raw bytes
// via removeHTMLTags only — there is no html.UnescapeString (or equivalent)
// pass over the result.
func TestBugHunt_EPUB_NamedEntitiesNotDecoded(t *testing.T) {
	got := epubChapterText(t, `<p>Tom &amp; Jerry said &quot;hi&quot; &lt;here&gt;</p>`)

	want := `Tom & Jerry said "hi" <here>`
	if !strings.Contains(got, want) {
		t.Errorf("named entities not decoded in EPUB chapter text:\n got  %q\n want it to contain %q", got, want)
	}
	// Explicit anti-bluff guards: the literal entity markup MUST NOT survive.
	for _, leak := range []string{"&amp;", "&quot;", "&lt;", "&gt;"} {
		if strings.Contains(got, leak) {
			t.Errorf("literal entity %q leaked into extracted chapter text: %q", leak, got)
		}
	}
}

func TestBugHunt_EPUB_NumericEntitiesNotDecoded(t *testing.T) {
	// &#233; = é (decimal), &#x2014; = — (em dash, hex).
	got := epubChapterText(t, `<p>caf&#233; &#x2014; ok</p>`)

	if !strings.Contains(got, "café") {
		t.Errorf("decimal numeric entity not decoded: got %q, want it to contain %q", got, "café")
	}
	if !strings.Contains(got, "—") {
		t.Errorf("hex numeric entity not decoded: got %q, want it to contain %q", got, "—")
	}
	for _, leak := range []string{"&#233;", "&#x2014;"} {
		if strings.Contains(got, leak) {
			t.Errorf("literal numeric entity %q leaked into extracted chapter text: %q", leak, got)
		}
	}
}

// Guard: ordinary text and already-decoded UTF-8 must pass through unchanged
// (the decode step must be idempotent on entity-free content).
func TestBugHunt_EPUB_PlainTextUnchanged(t *testing.T) {
	got := epubChapterText(t, `<p>Plain café — résumé text</p>`)
	if !strings.Contains(got, "Plain café — résumé text") {
		t.Errorf("plain UTF-8 text corrupted: got %q", got)
	}
}

// silence unused import if zip ever becomes unreferenced after edits.
var _ = zip.OpenReader
