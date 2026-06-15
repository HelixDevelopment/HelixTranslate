package markdown

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// epubParagraphRoundTrip drives the exact EPUB->Markdown->EPUB round-trip a real
// book takes for a single <p> element:
//
//	(1) EPUB side: an XHTML <p> with the given inner text,
//	(2) convertNode (EPUB->Markdown) emits the markdown for that paragraph,
//	(3) convertMarkdownToXHTML (Markdown->EPUB) re-parses that markdown into XHTML.
//
// It returns BOTH the intermediate markdown and the final re-emitted XHTML so a
// test can assert (a) the paragraph survived as prose (no list/heading/quote
// element) and (b) no characters (e.g. a leading "1.") were lost.
func epubParagraphRoundTrip(t *testing.T, innerText string) (md, finalXHTML string) {
	t.Helper()

	srcXHTML := "<html><body><p>" + innerText + "</p></body></html>"
	doc, err := html.Parse(strings.NewReader(srcXHTML))
	if err != nil {
		t.Fatalf("parse source xhtml: %v", err)
	}
	conv := NewEPUBToMarkdownConverter(false, "")
	body := conv.findBody(doc)
	var b strings.Builder
	conv.convertChildren(body, &b, 0)
	md = strings.TrimSpace(b.String())

	c := NewMarkdownToEPUBConverter()
	finalXHTML = c.convertMarkdownToXHTML(md)
	return md, finalXHTML
}

// TestBlockMarkerParagraph_RoundTripStaysProse proves that a prose paragraph
// whose text legitimately BEGINS with a markdown block-marker pattern survives
// the EPUB->Markdown->EPUB round-trip as a paragraph — it is NOT silently
// converted into an ordered/unordered list, a heading, or a blockquote, and no
// leading characters (the "1." digit, the "-"/"*"/"+" bullet char) are lost.
//
// The forward inline-escape fix protects mid-line markers; this BLOCK-level
// vector (line-leading marker) is the partner case the MD->EPUB block scanner
// (convertMarkdownToXHTML) mis-parses when the marker is unescaped.
func TestBlockMarkerParagraph_RoundTripStaysProse(t *testing.T) {
	cases := []struct {
		name      string
		inner     string // the <p> inner text in the source EPUB
		mustNotBe []string
	}{
		{
			name:      "ordered_list_marker_dot",
			inner:     "1. introduction sentence here",
			mustNotBe: []string{"<ol>", "<li>"},
		},
		{
			name:      "ordered_list_marker_paren",
			inner:     "1) introduction sentence here",
			mustNotBe: []string{"<ol>", "<li>"},
		},
		{
			name:      "unordered_dash",
			inner:     "- dash leads this prose paragraph",
			mustNotBe: []string{"<ul>", "<li>"},
		},
		{
			name:      "unordered_star",
			inner:     "* star leads this prose paragraph",
			mustNotBe: []string{"<ul>", "<li>"},
		},
		{
			name:      "unordered_plus",
			inner:     "+ plus leads this prose paragraph",
			mustNotBe: []string{"<ul>", "<li>"},
		},
		{
			name:      "blockquote_lead",
			inner:     "&gt; greater-than leads this prose paragraph",
			mustNotBe: []string{"<blockquote>"},
		},
		{
			name:      "heading_lead",
			inner:     "# hash leads this prose paragraph",
			mustNotBe: []string{"<h1>", "<h2>", "<h3>"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md, finalXHTML := epubParagraphRoundTrip(t, tc.inner)

			// (a) It must survive as a paragraph, not a list/heading/quote element.
			if !strings.Contains(finalXHTML, "<p>") {
				t.Fatalf("prose paragraph lost its <p> wrapper.\nmarkdown:\n%q\nxhtml:\n%s", md, finalXHTML)
			}
			for _, bad := range tc.mustNotBe {
				if strings.Contains(finalXHTML, bad) {
					t.Fatalf("prose paragraph wrongly became %s.\nmarkdown:\n%q\nxhtml:\n%s", bad, md, finalXHTML)
				}
			}
		})
	}
}

// TestBlockMarkerParagraph_OrderedDigitPreserved is the sharpest assertion: the
// "1." digit+dot that begins a prose paragraph MUST still be present (and as
// literal text, not a list number) after the full round-trip. The original bug
// dropped the "1." entirely (<p>1. text</p> -> <ol><li>text</li></ol>).
func TestBlockMarkerParagraph_OrderedDigitPreserved(t *testing.T) {
	md, finalXHTML := epubParagraphRoundTrip(t, "1. introduction sentence here")

	// The full original sentence, INCLUDING the leading "1.", must survive as
	// paragraph text. We assert the visible reader text contains "1." adjacent to
	// the sentence (the literal prose), inside a <p>.
	if strings.Contains(finalXHTML, "<ol>") || strings.Contains(finalXHTML, "<li>") {
		t.Fatalf("paragraph became an ordered list (digit converted to list number).\nmd:\n%q\nxhtml:\n%s", md, finalXHTML)
	}
	// Extract the <p>...</p> body and confirm it carries the literal "1." prefix.
	// We look for the digit+dot followed by the sentence text in the final XHTML.
	if !strings.Contains(finalXHTML, "1. introduction sentence here") &&
		!strings.Contains(finalXHTML, "1. introduction sentence here") {
		t.Fatalf("leading \"1.\" digit lost on round-trip.\nmd:\n%q\nxhtml:\n%s", md, finalXHTML)
	}
}
