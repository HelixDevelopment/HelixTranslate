package ebook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFB2Parse_MultiParagraphSectionTitleNotDropped is a §11.4.115 reproduce-first
// bug-hunt test.
//
// DEFECT: convertFB2Section reads ONLY fb2Sec.Title.Paragraphs[0] for the chapter
// title. FB2 <title> is a sequence of <p> lines (multi-line headings are common —
// a chapter number line plus a chapter name line, or a title spanning lines). Every
// title line after the first is SILENTLY DROPPED, so a translated book loses the
// second (and further) lines of every multi-line chapter heading.
func TestFB2Parse_MultiParagraphSectionTitleNotDropped(t *testing.T) {
	const fb2XML = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Multiline Title</book-title>
      <lang>en</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title>
        <p>Chapter Seven</p>
        <p>The Long Road Home</p>
      </title>
      <p>Body paragraph here.</p>
    </section>
  </body>
</FictionBook>`

	dir := t.TempDir()
	path := filepath.Join(dir, "title.fb2")
	if err := os.WriteFile(path, []byte(fb2XML), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	book, err := NewFB2Parser().Parse(path)
	if err != nil {
		t.Fatalf("FB2Parser.Parse: %v", err)
	}
	if len(book.Chapters) == 0 {
		t.Fatal("no chapters parsed")
	}
	title := book.Chapters[0].Title

	if !strings.Contains(title, "Chapter Seven") {
		t.Fatalf("sanity: first title line missing; title=%q", title)
	}
	if !strings.Contains(title, "The Long Road Home") {
		t.Errorf("second title line DROPPED: %q not found in chapter title %q",
			"The Long Road Home", title)
	}
}
