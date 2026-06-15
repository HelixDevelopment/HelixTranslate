package markdown

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// htmlToMDToHTML runs the real EPUB<->Markdown round-trip starting from the
// EPUB side: an XHTML fragment (as found inside an EPUB chapter) is converted
// EPUB->Markdown (convertChildren) and then Markdown->EPUB (markdownToHTML).
// Literal prose text containing markdown metacharacters MUST survive this trip
// unchanged — otherwise the user's book content is permanently corrupted.
func htmlToMDToHTML(t *testing.T, bodyHTML string) (mdOut, htmlOut string) {
	t.Helper()
	full := "<html><body>" + bodyHTML + "</body></html>"
	doc, err := html.Parse(strings.NewReader(full))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	conv := NewEPUBToMarkdownConverter(false, "")
	body := conv.findBody(doc)
	var b strings.Builder
	conv.convertChildren(body, &b, 0)
	mdOut = strings.TrimSpace(b.String())
	c := NewMarkdownToEPUBConverter()
	htmlOut = c.markdownToHTML(mdOut)
	return
}

// REGRESSION: literal underscores in EPUB prose ("C_3 and C_4") must NOT be
// re-interpreted as emphasis on the markdown->EPUB return trip. Before the fix,
// EPUB->markdown emitted the underscores raw and the return trip produced
// "C<em>3 and C</em>4" — permanent corruption of the user-visible text.
func TestEPUBToMD_LiteralUnderscores_SurviveRoundTrip(t *testing.T) {
	_, htmlOut := htmlToMDToHTML(t, "<p>variables C_3 and C_4 are coupled</p>")
	if strings.Contains(htmlOut, "<em>") || strings.Contains(htmlOut, "<strong>") {
		t.Fatalf("literal underscores became emphasis on round-trip: %q", htmlOut)
	}
	if !strings.Contains(htmlOut, "C_3 and C_4") {
		t.Fatalf("literal text not preserved: %q", htmlOut)
	}
}

// REGRESSION: literal asterisks in EPUB prose must NOT become emphasis.
func TestEPUBToMD_LiteralAsterisks_SurviveRoundTrip(t *testing.T) {
	_, htmlOut := htmlToMDToHTML(t, "<p>compute 2 *important* result and a*b*c</p>")
	if strings.Contains(htmlOut, "<em>") || strings.Contains(htmlOut, "<strong>") {
		t.Fatalf("literal asterisks became emphasis on round-trip: %q", htmlOut)
	}
	if !strings.Contains(htmlOut, "*important*") {
		t.Fatalf("literal text not preserved: %q", htmlOut)
	}
}

// REGRESSION: literal bracket/paren prose ("[draft](TODO)") must NOT become a
// fabricated hyperlink on the round-trip.
func TestEPUBToMD_LiteralBrackets_SurviveRoundTrip(t *testing.T) {
	_, htmlOut := htmlToMDToHTML(t, "<p>see [draft](TODO) note</p>")
	if strings.Contains(htmlOut, "<a ") {
		t.Fatalf("literal bracket text became a link on round-trip: %q", htmlOut)
	}
	if !strings.Contains(htmlOut, "[draft](TODO)") {
		t.Fatalf("literal text not preserved: %q", htmlOut)
	}
}

// REGRESSION (no over-escape): content inside a <code> span is literal-by-fence
// and MUST be emitted verbatim — the metacharacter escaping that protects prose
// must NOT touch code content ("file_name_v2" stays "file_name_v2", not
// "file\_name\_v2", and never becomes emphasis).
func TestEPUBToMD_InlineCode_NotOverEscaped(t *testing.T) {
	mdOut, htmlOut := htmlToMDToHTML(t, "<p>run <code>file_name_v2</code> now</p>")
	if strings.Contains(mdOut, `file\_name`) {
		t.Fatalf("code content over-escaped in markdown: %q", mdOut)
	}
	if !strings.Contains(htmlOut, "<code>file_name_v2</code>") {
		t.Fatalf("code content corrupted on round-trip: %q", htmlOut)
	}
}

// REGRESSION (no over-escape): genuine markdown emphasis / links authored as
// real markup ("*x*", "[t](u)") must STILL round-trip into real <em>/<a> — the
// prose-escaping fix must not suppress legitimate formatting (it operates on the
// EPUB text branch, while real emphasis arrives as <em>/<a> elements).
func TestEPUBToMD_RealFormatting_StillRoundTrips(t *testing.T) {
	_, htmlOut := htmlToMDToHTML(t, "<p>see <em>real</em> and <a href=\"u\">t</a></p>")
	if !strings.Contains(htmlOut, "<em>real</em>") {
		t.Fatalf("real emphasis lost on round-trip: %q", htmlOut)
	}
	if !strings.Contains(htmlOut, `<a href="u">t</a>`) {
		t.Fatalf("real link lost on round-trip: %q", htmlOut)
	}
}
