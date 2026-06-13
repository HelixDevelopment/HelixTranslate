package fb2

import (
	"strings"
	"testing"
)

// TestRegression_FB2ParagraphRoundTrip is the permanent guard for the FB2
// parse → write paragraph data-loss defect.
//
// The Paragraph struct stores its user-visible text in Text/FullText, both
// tagged `xml:"-"`, with a custom UnmarshalXML but NO matching MarshalXML. As a
// result the default struct marshaling dropped ALL paragraph content on write —
// every <p> came out empty (<p></p>) — so a parse → (translate) → write
// round-trip through this package silently lost every paragraph's text.
//
// Mutation guard: remove the CharData emission from Paragraph.MarshalXML and
// this test FAILs (the written output contains empty paragraphs and the
// re-parsed text is gone).
func TestRegression_FB2ParagraphRoundTrip(t *testing.T) {
	doc := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description><title-info><book-title>T</book-title><lang>ru</lang></title-info></description>
  <body><section>
    <p>Hello <emphasis>world</emphasis> translated text.</p>
    <p>Второй абзац с &amp; амперсандом.</p>
  </section></body>
</FictionBook>`

	p := NewParser()
	fb, err := p.ParseReader(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var buf strings.Builder
	if err := p.WriteToWriter(&buf, fb); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := buf.String()

	// The written FB2 MUST contain the paragraph text, not empty <p></p>.
	if strings.Contains(out, "<p></p>") {
		t.Errorf("written FB2 contains empty paragraph (data loss):\n%s", out)
	}
	for _, want := range []string{"Hello world translated text.", "Второй абзац с & амперсандом."} {
		if !strings.Contains(out, escapeForCheck(want)) {
			t.Errorf("written FB2 missing paragraph text %q:\n%s", want, out)
		}
	}

	// Full round-trip: re-parse the written output and confirm the text survives.
	fb2, err := p.ParseReader(strings.NewReader(out))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	paras := fb2.Body[0].Section[0].Paragraph
	want := []string{"Hello world translated text.", "Второй абзац с & амперсандом."}
	if len(paras) != len(want) {
		t.Fatalf("round-trip paragraph count = %d, want %d:\n%s", len(paras), len(want), out)
	}
	for i, w := range want {
		if got := paras[i].FullParagraphText(); got != w {
			t.Errorf("round-trip paragraph[%d] = %q, want %q", i, got, w)
		}
	}
}

// escapeForCheck mirrors the encoder's escaping of '&' so the raw-output
// substring check matches what the XML encoder emits.
func escapeForCheck(s string) string {
	return strings.ReplaceAll(s, "&", "&amp;")
}
