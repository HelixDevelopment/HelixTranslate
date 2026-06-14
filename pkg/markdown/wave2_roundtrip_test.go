package markdown

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// lineContaining returns the first line of s that contains sub, or "".
func lineContaining(s, sub string) string {
	for _, ln := range strings.Split(s, "\n") {
		if strings.Contains(ln, sub) {
			return ln
		}
	}
	return ""
}

// roundTripChapter runs the production EPUB chapter path
// (markdown -> convertMarkdownToXHTML) and then the EPUB->markdown path
// (convertNode) — the exact bytes a book loses when round-tripped.
func roundTripChapter(t *testing.T, md string) string {
	t.Helper()
	c := NewMarkdownToEPUBConverter()
	xhtml := c.convertMarkdownToXHTML(md)
	doc, err := html.Parse(strings.NewReader(xhtml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	conv := NewEPUBToMarkdownConverter(false, "")
	body := conv.findBody(doc)
	var b strings.Builder
	conv.convertChildren(body, &b, 0)
	return strings.TrimSpace(b.String())
}

// WAVE2 BUG A: a multi-line fenced code block is DESTROYED on the EPUB->markdown
// side. The XHTML <pre><code>...</code></pre> holds the code with newlines and
// indentation intact, but convertNode's "pre" case recurses through the inline
// path: the inner <code> wraps the body in stray backticks and the text node is
// run through collapseInlineWhitespace, flattening every newline to a space and
// stripping leading indentation. A reader's code block becomes one mangled line.
func TestWave2_CodeBlock_NewlinesAndIndentSurvive(t *testing.T) {
	md := "```\nfn main() {\n    return 0\n}\n```"
	out := roundTripChapter(t, md)
	// The internal newlines MUST survive (code is multi-line).
	if !strings.Contains(out, "fn main() {\n") {
		t.Fatalf("code newlines collapsed (structure destroyed): %q", out)
	}
	// The 4-space indentation of the body line MUST survive.
	if !strings.Contains(out, "\n    return 0") {
		t.Fatalf("code indentation lost: %q", out)
	}
	// No stray inline backticks injected by the <code> element inside <pre>.
	if strings.Contains(out, "`fn main") {
		t.Fatalf("stray inline backticks injected inside fenced block: %q", out)
	}
}

// WAVE2 BUG B: a GFM pipe table is shipped to the reader as a single paragraph
// of literal pipes ("<p>| Name | Age | | --- | ...</p>"): every row collapsed
// into one <p>, all structure lost. convertMarkdownToXHTML (and markdownToHTML)
// have no table handling at all.
func TestWave2_Table_ProducesTableElement(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	md := "| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |"
	xhtml := c.convertMarkdownToXHTML(md)
	if !strings.Contains(xhtml, "<table>") {
		t.Fatalf("table not converted to <table> (structure lost):\n%s", xhtml)
	}
	// The literal pipe-delimited rows must NOT survive as paragraph text.
	if strings.Contains(xhtml, "<p>| Name") {
		t.Fatalf("table shipped as literal pipe paragraph:\n%s", xhtml)
	}
	// Cell data must be present in cells.
	for _, want := range []string{"<th>Name</th>", "<th>Age</th>", "<td>Alice</td>", "<td>25</td>"} {
		if !strings.Contains(xhtml, want) {
			t.Fatalf("table cell %q missing:\n%s", want, xhtml)
		}
	}
}

// WAVE2 BUG B round-trip: the table survives md -> XHTML -> md as a pipe table
// with all cell data intact.
func TestWave2_Table_RoundTrips(t *testing.T) {
	out := roundTripChapter(t, "| Name | Age |\n| --- | --- |\n| Alice | 30 |")
	for _, want := range []string{"Name", "Age", "Alice", "30"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table cell %q lost on round-trip: %q", want, out)
		}
	}
	// Header row and data row MUST be on separate lines (not flattened into one
	// paragraph). A genuine table round-trip keeps the rows distinct.
	hdrLine := lineContaining(out, "Name")
	dataLine := lineContaining(out, "Alice")
	if hdrLine == dataLine {
		t.Fatalf("table rows flattened into one line (structure lost): %q", out)
	}
	if !strings.Contains(hdrLine, "|") || !strings.Contains(dataLine, "|") {
		t.Fatalf("table rows not pipe-delimited on round-trip: %q", out)
	}
}

// WAVE2 BUG C: a backslash-escaped asterisk ("\*literal\*") is wrongly treated
// as emphasis: convertInlineMarkdown emits "literal \<em>not italic\</em>",
// corrupting the text (stray backslashes) and injecting a phantom <em>. The
// user's literal asterisks never reach the reader.
func TestWave2_EscapedAsterisk_StaysLiteral(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	out := c.convertInlineMarkdown(`literal \*not italic\*`)
	if strings.Contains(out, "<em>") {
		t.Fatalf("escaped asterisk wrongly became emphasis: %q", out)
	}
	// The literal asterisks must reach the reader (escaped backslash consumed).
	if !strings.Contains(out, "*not italic*") {
		t.Fatalf("escaped asterisk content lost: %q", out)
	}
	// No stray backslashes left behind.
	if strings.Contains(out, `\`) {
		t.Fatalf("stray backslash left in output: %q", out)
	}
}
