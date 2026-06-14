package markdown

import (
	"strings"
	"testing"

	"encoding/xml"
)

// WAVE3 BUG: convertMarkdownToXHTML embeds c.metadata.Title into the chapter
// <title> element WITHOUT escaping (markdown_to_epub.go:1056), while every other
// title site escapes (OPF dc:title, toc.ncx, header text). A book title with an
// XML metacharacter (& is extremely common: "Pride & Prejudice", "Crime &
// Punishment") produces "<title>Pride & Prejudice</title>" — a bare & is not a
// valid entity, so the chapter XHTML is MALFORMED: strict EPUB validators
// (epubcheck) reject it and strict XML parsers fail to parse it.
func TestWave3_ChapterTitle_Escaped(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	c.metadata.Title = "Pride & Prejudice <Vol 1>"
	xhtml := c.convertMarkdownToXHTML("Some chapter body text.")

	// The produced chapter XHTML MUST be well-formed XML.
	if err := xml.Unmarshal([]byte(xhtml), new(struct {
		XMLName xml.Name
	})); err != nil {
		t.Fatalf("chapter XHTML is malformed XML (unescaped title): %v\n--- output ---\n%s", err, xhtml)
	}
	// The raw unescaped "& Prejudice" must NOT appear in the <title>.
	if strings.Contains(xhtml, "<title>Pride & Prejudice") {
		t.Fatalf("chapter <title> contains a bare unescaped '&':\n%s", xhtml)
	}
	// The escaped form must be present instead.
	if !strings.Contains(xhtml, "Pride &amp; Prejudice &lt;Vol 1&gt;") {
		t.Fatalf("chapter <title> not properly escaped:\n%s", xhtml)
	}
}
