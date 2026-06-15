package ebook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFB2Parse_SectionSubtitlePoemCiteEpigraphNotDropped is a §11.4.115
// reproduce-first bug-hunt test.
//
// DEFECT: convertFB2Section (pkg/ebook/fb2_parser.go) only converts a section's
// <p> paragraphs and its nested <section>s into Book content. The FB2 schema —
// and the fb2 package's Section struct — also model <subtitle>, <poem>, <cite>,
// and <epigraph> as DIRECT children of a <section>. Those are parsed by the fb2
// package but NEVER converted into the universal Book, so every word of a
// subtitle, poem verse, citation, or epigraph that appears directly under a
// section is SILENTLY DROPPED before translation/output — real user-visible
// content loss on the main FB2 translation pipeline.
//
// This drives the REAL pipeline: a full FB2 XML document parsed via
// FB2Parser.Parse, and asserts the user-visible text survives into Book content.
func TestFB2Parse_SectionSubtitlePoemCiteEpigraphNotDropped(t *testing.T) {
	const fb2XML = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <book-title>Section Content Loss</book-title>
      <lang>en</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Chapter One</p></title>
      <p>An ordinary paragraph.</p>
      <subtitle>A SECTION SUBTITLE LINE</subtitle>
      <epigraph>
        <p>An epigraph paragraph that matters.</p>
        <text-author>EPIGRAPH AUTHOR NAME</text-author>
      </epigraph>
      <poem>
        <stanza>
          <v>First verse line of the poem.</v>
          <v>Second verse line of the poem.</v>
        </stanza>
      </poem>
      <cite>
        <p>A cited sentence worth keeping.</p>
        <text-author>CITE SOURCE AUTHOR</text-author>
      </cite>
    </section>
  </body>
</FictionBook>`

	dir := t.TempDir()
	path := filepath.Join(dir, "section_content.fb2")
	if err := os.WriteFile(path, []byte(fb2XML), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	parser := NewFB2Parser()
	book, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("FB2Parser.Parse: %v", err)
	}

	// Flatten ALL content the pipeline produced for this book.
	var all strings.Builder
	for _, ch := range book.Chapters {
		all.WriteString(ch.Title)
		all.WriteString("\n")
		for _, sec := range ch.Sections {
			collectAllContent(&all, sec)
		}
	}
	got := all.String()

	// Sanity: the ordinary paragraph MUST be present (proves the pipeline ran).
	if !strings.Contains(got, "An ordinary paragraph.") {
		t.Fatalf("sanity: ordinary <p> missing from parsed content; got:\n%s", got)
	}

	// The user-visible content of every section-level construct MUST survive.
	wantFragments := []string{
		"A SECTION SUBTITLE LINE",            // <subtitle>
		"An epigraph paragraph that matters", // <epigraph><p>
		"First verse line of the poem.",      // <poem><stanza><v>
		"Second verse line of the poem.",     // <poem><stanza><v>
		"A cited sentence worth keeping",     // <cite><p>
	}
	for _, frag := range wantFragments {
		if !strings.Contains(got, frag) {
			t.Errorf("section-level content DROPPED: %q not found in parsed Book content", frag)
		}
	}
	if t.Failed() {
		t.Logf("parsed Book content was:\n%s", got)
	}
}

// collectAllContent appends a section's Content plus all nested subsection
// content recursively, so the assertion sees everything the pipeline retained.
func collectAllContent(b *strings.Builder, sec Section) {
	b.WriteString(sec.Title)
	b.WriteString("\n")
	b.WriteString(sec.Content)
	b.WriteString("\n")
	for _, sub := range sec.Subsections {
		collectAllContent(b, sub)
	}
}
