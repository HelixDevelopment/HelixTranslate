package fb2

import (
	"strings"
	"testing"
)

// TestRegression_FB2VerseSubtitleTextAuthorInlinePreserved is the permanent
// regression guard for the FB2 mixed-content text-loss defect in the <v>
// (verse), <subtitle>, and <text-author> elements.
//
// These FB2 elements are mixed content (they may contain inline <emphasis>,
// <strong>, <a>, <style> formatting), but they were modeled as a plain Go
// string / []string with `,chardata`, which captures ONLY the bare character
// data and silently DROPS every inline element's text. A verse line
// "Verse <strong>STRONG</strong> end" was parsed as "Verse  end" — the
// emphasized/strong/linked words vanished and left a double-space gap, exactly
// the same class of data-loss the Paragraph type already fixes.
//
// Mutation guard: revert <v>/<subtitle>/<text-author> back to a `,chardata`
// string (drop the inline-element text) and this test FAILs — the inline words
// disappear.
func TestRegression_FB2VerseSubtitleTextAuthorInlinePreserved(t *testing.T) {
	doc := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description><title-info><book-title>T</book-title><lang>ru</lang></title-info></description>
  <body><section>
    <subtitle>Sub <emphasis>EMPH</emphasis> tail</subtitle>
    <poem><stanza>
      <v>Verse <strong>STRONG</strong> end</v>
      <v>Текст с <emphasis>выделением</emphasis> внутри</v>
    </stanza></poem>
    <epigraph><p>e</p><text-author>By <emphasis>NAME</emphasis> here</text-author></epigraph>
    <cite><p>c</p><subtitle>Cite <a l:href="#n1">link</a> end</subtitle><text-author>Said <strong>X</strong> now</text-author></cite>
  </section></body>
</FictionBook>`

	p := NewParser()
	fb, err := p.ParseReader(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sec := fb.Body[0].Section[0]

	if got := sec.Subtitle[0]; got != "Sub EMPH tail" {
		t.Errorf("section subtitle = %q, want %q (inline text must be preserved)", got, "Sub EMPH tail")
	}
	if got := sec.Poem[0].Stanza[0].V[0].Text; got != "Verse STRONG end" {
		t.Errorf("verse[0] = %q, want %q (inline text must be preserved)", got, "Verse STRONG end")
	}
	if got := sec.Poem[0].Stanza[0].V[1].Text; got != "Текст с выделением внутри" {
		t.Errorf("verse[1] = %q, want %q (cyrillic inline text must be preserved)", got, "Текст с выделением внутри")
	}
	if got := sec.Epigraph[0].TextAuthor[0]; got != "By NAME here" {
		t.Errorf("epigraph text-author = %q, want %q (inline text must be preserved)", got, "By NAME here")
	}
	if got := sec.Cite[0].Subtitle[0]; got != "Cite link end" {
		t.Errorf("cite subtitle = %q, want %q (inline link text must be preserved)", got, "Cite link end")
	}
	if got := sec.Cite[0].TextAuthor[0]; got != "Said X now" {
		t.Errorf("cite text-author = %q, want %q (inline text must be preserved)", got, "Said X now")
	}
}
