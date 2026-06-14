package markdown

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// BUG (reproduce-first, §11.4.115): inline code spans in convertInlineMarkdown
// are converted LAST — after the bold/italic/link/image regexes have already
// run over the whole string. So markdown metacharacters INSIDE a `code` span
// (underscores in an identifier, asterisks, [..](..) link syntax shown as
// literal code) are mistaken for emphasis/links and rewritten. An identifier
// like `file_name_v2` ships to the reader as <code>file<em>name</em>v2</code>
// and ROUND-TRIPS back to markdown as `file*name*v2` — permanent corruption of
// the code content (and a silent _→* mutation). Inline code must be opaque.
func TestInlineCode_UnderscoresNotEmphasis(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	out := c.convertInlineMarkdown("The `file_name_v2` variable.")
	if strings.Contains(out, "<em>") {
		t.Fatalf("underscores inside inline code became <em> (code content corrupted): %q", out)
	}
	if !strings.Contains(out, "<code>file_name_v2</code>") {
		t.Fatalf("inline code content not preserved verbatim: %q", out)
	}
}

func TestInlineCode_AsterisksNotBold(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	out := c.convertInlineMarkdown("Glob `**/*.go` matches files.")
	if strings.Contains(out, "<strong>") {
		t.Fatalf("asterisks inside inline code became <strong>: %q", out)
	}
	if !strings.Contains(out, "<code>**/*.go</code>") {
		t.Fatalf("inline code content not preserved verbatim: %q", out)
	}
}

func TestInlineCode_LinkSyntaxNotLink(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	out := c.convertInlineMarkdown("Write `[text](url)` for a link.")
	if strings.Contains(out, "<a ") || strings.Contains(out, "<img ") {
		t.Fatalf("link/image syntax inside inline code was interpreted: %q", out)
	}
	if !strings.Contains(out, "<code>[text](url)</code>") {
		t.Fatalf("inline code content not preserved verbatim: %q", out)
	}
}

// Round-trip on the PRODUCTION EPUB chapter path (convertMarkdownToXHTML ->
// convertNode). A Cyrillic sentence with an embedded code identifier containing
// underscores must survive intact. This is the end-user data-loss assertion.
func TestInlineCode_CyrillicRoundTrip_Preserved(t *testing.T) {
	c := NewMarkdownToEPUBConverter()
	md := "Используйте `имя_файла_v2` для настройки."
	xhtml := c.convertMarkdownToXHTML(md)
	if !strings.ContainsRune(xhtml, 'И') || !strings.Contains(xhtml, "имя_файла_v2") {
		t.Fatalf("cyrillic or code content lost in XHTML: %q", xhtml)
	}
	if strings.Contains(xhtml, "<em>") {
		t.Fatalf("code identifier underscores became <em> in production XHTML: %q", xhtml)
	}
	if !utf8Valid(xhtml) {
		t.Fatalf("produced XHTML is not valid UTF-8")
	}

	doc, err := html.Parse(strings.NewReader(xhtml))
	if err != nil {
		t.Fatalf("parse produced xhtml: %v", err)
	}
	conv := NewEPUBToMarkdownConverter(false, "")
	body := conv.findBody(doc)
	var b strings.Builder
	conv.convertChildren(body, &b, 0)
	out := strings.TrimSpace(b.String())
	if !strings.Contains(out, "`имя_файла_v2`") {
		t.Fatalf("code identifier corrupted on round-trip (expected `имя_файла_v2`): got %q", out)
	}
}

func utf8Valid(s string) bool {
	return strings.ToValidUTF8(s, "�") == s
}
