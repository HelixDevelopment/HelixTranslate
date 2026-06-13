package markdown

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// mdToHTMLToMD performs the real round-trip used by the EPUB<->Markdown
// workflow: markdown -> XHTML (markdownToHTML) -> markdown (convertNode).
// Data dropped here is data the end user loses when round-tripping a book.
func mdToHTMLToMD(t *testing.T, md string) (htmlOut, mdOut string) {
	t.Helper()
	c := NewMarkdownToEPUBConverter()
	htmlOut = c.markdownToHTML(md)
	full := "<html><body>" + htmlOut + "</body></html>"
	doc, err := html.Parse(strings.NewReader(full))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	conv := NewEPUBToMarkdownConverter(false, "")
	body := conv.findBody(doc)
	var b strings.Builder
	conv.convertChildren(body, &b, 0)
	mdOut = strings.TrimSpace(b.String())
	return
}

// BUG 1: markdownToHTML emits a real <a> for inline links so the hyperlink
// survives into the EPUB (and round-trips). Before the fix, "[t](url)" was
// left as literal text inside a <p>, so the reader saw raw "[t](url)" and a
// round-trip produced no <a> at all (hyperlink lost).
func TestRoundTrip_InlineLink_ProducesAnchor(t *testing.T) {
	md := "See [the docs](http://example.com/x) now."
	htmlOut, mdOut := mdToHTMLToMD(t, md)
	if !strings.Contains(htmlOut, `<a href="http://example.com/x">the docs</a>`) {
		t.Fatalf("link not converted to <a> in HTML:\n%s", htmlOut)
	}
	if !strings.Contains(mdOut, "[the docs](http://example.com/x)") {
		t.Fatalf("link lost across round-trip: got %q", mdOut)
	}
}

// BUG 2: markdownToHTML emits a real <img> for inline images. Before the fix
// "![alt](pic.png)" stayed literal text, so the image never reached the EPUB.
func TestRoundTrip_InlineImage_ProducesImg(t *testing.T) {
	md := "![a cat](cat.png)"
	htmlOut, mdOut := mdToHTMLToMD(t, md)
	if !strings.Contains(htmlOut, `<img src="cat.png" alt="a cat"`) {
		t.Fatalf("image not converted to <img> in HTML:\n%s", htmlOut)
	}
	if !strings.Contains(mdOut, "![a cat](Images/cat.png)") {
		t.Fatalf("image lost across round-trip: got %q", mdOut)
	}
}

// BUG 3: a markdown blockquote line ("> ...") becomes a <blockquote> in HTML,
// not a <p> with a literal "&gt;". The HTML->md side already emits "> " for
// <blockquote>; before the fix md->HTML produced "<p>&gt; quoted</p>" so the
// round-trip mangled the quote into a paragraph beginning with a literal ">".
func TestRoundTrip_Blockquote_ProducesBlockquote(t *testing.T) {
	md := "> a wise quote"
	htmlOut, mdOut := mdToHTMLToMD(t, md)
	if !strings.Contains(htmlOut, "<blockquote>") {
		t.Fatalf("blockquote not converted to <blockquote> in HTML:\n%s", htmlOut)
	}
	if strings.Contains(htmlOut, "&gt; a wise quote") {
		t.Fatalf("blockquote emitted as literal '&gt;' paragraph:\n%s", htmlOut)
	}
	if !strings.HasPrefix(mdOut, ">") {
		t.Fatalf("blockquote marker lost across round-trip: got %q", mdOut)
	}
}

// KNOWN-LOSSY (tracked follow-up): an ordered list whose markdown starts at a
// number other than 1 is renumbered from 1 on the EPUB->markdown side, because
// the generated <ol> carries no "start" attribute and convertListItems always
// counts from 1. This is documented loss, NOT silent: this test pins the exact
// current behaviour so a future "preserve <ol start>" fix has a regression
// anchor and any accidental change to numbering is caught. The numbering loss
// is cosmetic (item order/content is preserved); the data — the list items —
// round-trips intact.
func TestRoundTrip_OrderedListStart_KnownLossy(t *testing.T) {
	_, mdOut := mdToHTMLToMD(t, "3. third\n4. fourth")
	// Content (the data) survives:
	if !strings.Contains(mdOut, "third") || !strings.Contains(mdOut, "fourth") {
		t.Fatalf("ordered-list ITEM CONTENT lost (real data loss): %q", mdOut)
	}
	// Numbering is reset to 1.. — documented loss. If a future fix preserves the
	// original start, update this test (it is the regression anchor).
	if !strings.HasPrefix(strings.TrimSpace(mdOut), "1. third") {
		t.Fatalf("KNOWN-LOSSY contract changed: expected renumber-from-1, got %q", mdOut)
	}
}

// BUG 4: convertMarkdownToXHTML is the path createEPUB actually writes into each
// EPUB chapter. Before the fix it emitted a blockquote line as
// "<p>&gt; a quote</p>" — the reader saw a literal ">" and the blockquote
// element was lost in the produced EPUB (and would not round-trip).
func TestXHTML_Blockquote_ProducesBlockquote(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	out := c.convertMarkdownToXHTML("> a wise quote")
	if !strings.Contains(out, "<blockquote>") {
		t.Fatalf("chapter XHTML did not emit <blockquote>:\n%s", out)
	}
	if strings.Contains(out, "&gt; a wise quote") {
		t.Fatalf("chapter XHTML emitted blockquote as literal '&gt;':\n%s", out)
	}
}

// Behavioral: a realistic chapter mixing a link, an image and a blockquote
// survives the PRODUCTION EPUB chapter path (convertMarkdownToXHTML) AND
// round-trips back through the HTML->markdown converter with all three
// structural elements intact. Fails if any of the three fixes is removed.
func TestRoundTrip_RealChapter_LinkImageQuote(t *testing.T) {
	md := "Intro with [a link](http://e.com/p).\n\n" +
		"![cover art](art.png)\n\n" +
		"> a quoted insight"
	c := NewMarkdownToEPUBConverter()
	xhtml := c.convertMarkdownToXHTML(md)

	for _, want := range []string{
		`<a href="http://e.com/p">a link</a>`,
		`<img src="art.png" alt="cover art"/>`,
		`<blockquote>a quoted insight</blockquote>`,
	} {
		if !strings.Contains(xhtml, want) {
			t.Fatalf("production chapter XHTML missing %q:\n%s", want, xhtml)
		}
	}

	// Round-trip the produced XHTML back to markdown.
	doc, err := html.Parse(strings.NewReader(xhtml))
	if err != nil {
		t.Fatalf("parse produced xhtml: %v", err)
	}
	conv := NewEPUBToMarkdownConverter(false, "")
	body := conv.findBody(doc)
	var b strings.Builder
	conv.convertChildren(body, &b, 0)
	out := b.String()
	for _, want := range []string{"[a link](http://e.com/p)", "![cover art](Images/art.png)", "> a quoted insight"} {
		if !strings.Contains(out, want) {
			t.Fatalf("round-trip lost %q:\n%s", want, out)
		}
	}
}
