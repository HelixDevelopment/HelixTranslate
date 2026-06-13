package ebook

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// reproHTML parses a fragment and returns trimmed extracted text.
func reproHTML(t *testing.T, frag string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(frag))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p := NewHTMLParser()
	return strings.TrimSpace(p.extractText(doc))
}

// BUG #1 (FACT, reproduced): table/definition-list structural cells are
// concatenated with NO separator, gluing distinct cells into one nonsense
// token. For an ebook translator this corrupts both extracted text and any
// downstream translation: "cell A" + "cell B" -> "cell Acell B".
//
// Root cause: isBlockElement omits td/th/tr/dt/dd, so no separator is
// inserted between sibling cells in extractTextWithContext.
func TestBugHunt_HTML_TableCellsGlued(t *testing.T) {
	got := reproHTML(t, `<table><tr><td>cell A</td><td>cell B</td></tr></table>`)
	if strings.Contains(got, "Acell") {
		t.Errorf("table cells glued together: got %q (cells must be separated)", got)
	}
	if !strings.Contains(got, "cell A") || !strings.Contains(got, "cell B") {
		t.Errorf("table cell content lost: got %q", got)
	}
}

func TestBugHunt_HTML_TableHeaderCellsGlued(t *testing.T) {
	got := reproHTML(t, `<table><tr><th>Name</th><th>Age</th></tr></table>`)
	if strings.Contains(got, "NameAge") {
		t.Errorf("table header cells glued: got %q", got)
	}
}

func TestBugHunt_HTML_DefinitionListGlued(t *testing.T) {
	got := reproHTML(t, `<dl><dt>HTTP</dt><dd>HyperText Transfer Protocol</dd></dl>`)
	if strings.Contains(got, "HTTPHyperText") {
		t.Errorf("definition list term/def glued: got %q", got)
	}
}

// BUG #2 (FACT, reproduced): the <br> line-break void element is dropped,
// gluing the end of one line directly onto the start of the next:
// "line one<br>line two" -> "line oneline two".
//
// Root cause: <br> is neither a TextNode nor handled as a separator; it has
// no child text so extractTextWithContext emits nothing for it.
func TestBugHunt_HTML_BrLineBreakDropped(t *testing.T) {
	got := reproHTML(t, `<p>line one<br>line two</p>`)
	if strings.Contains(got, "oneline") {
		t.Errorf("<br> dropped, words glued: got %q (expected a break between lines)", got)
	}
	if !strings.Contains(got, "line one") || !strings.Contains(got, "line two") {
		t.Errorf("line content lost around <br>: got %q", got)
	}
}

// Guard: previously-working behavior must stay intact after the fix.
func TestBugHunt_HTML_ParagraphsStillSeparated(t *testing.T) {
	got := reproHTML(t, `<p>First paragraph</p><p>Second paragraph</p>`)
	want := "First paragraph\n\nSecond paragraph"
	if got != want {
		t.Errorf("paragraph separation regressed: got %q want %q", got, want)
	}
}

func TestBugHunt_HTML_InlineNotOverSplit(t *testing.T) {
	got := reproHTML(t, `<p>An <b>important</b> note</p>`)
	want := "An important note"
	if got != want {
		t.Errorf("inline element over-split: got %q want %q", got, want)
	}
}
