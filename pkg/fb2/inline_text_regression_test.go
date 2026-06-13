package fb2

import (
	"strings"
	"testing"
)

// TestRegression_FB2InlineAndTailTextPreserved is the permanent regression
// guard for the FB2 mixed-content text-loss defect (inline-element text and
// tail text were dropped, producing broken sentences with gaps where
// <emphasis>/<strong>/<a> words used to be).
//
// Mutation guard: revert Paragraph to chardata-only extraction (drop the
// inline-element text) and this test FAILs — the inline words disappear.
func TestRegression_FB2InlineAndTailTextPreserved(t *testing.T) {
	parser := NewParser()
	cases := []struct {
		name string
		xml  string
		want string
	}{
		{
			name: "inline emphasis and strong with tail",
			xml:  `<p>Before <emphasis>EMPH</emphasis> middle <strong>STRONG</strong> tail.</p>`,
			want: "Before EMPH middle STRONG tail.",
		},
		{
			name: "nested inline formatting",
			xml:  `<p>A <emphasis>e<strong>S</strong>e</emphasis> B</p>`,
			want: "A eSe B",
		},
		{
			name: "link element text preserved",
			xml:  `<p>see <a l:href="#n1">note one</a> now</p>`,
			want: "see note one now",
		},
		{
			name: "leading inline element then tail",
			xml:  `<p><emphasis>Start</emphasis> then rest.</p>`,
			want: "Start then rest.",
		},
		{
			name: "cyrillic inline content",
			xml:  `<p>Текст с <emphasis>выделением</emphasis> внутри.</p>`,
			want: "Текст с выделением внутри.",
		},
		{
			name: "plain paragraph unchanged",
			xml:  `<p>Just plain text.</p>`,
			want: "Just plain text.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description><title-info><book-title>T</book-title><lang>ru</lang></title-info></description>
  <body><section>` + tc.xml + `</section></body>
</FictionBook>`
			fb, err := parser.ParseReader(strings.NewReader(doc))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(fb.Body) == 0 || len(fb.Body[0].Section) == 0 || len(fb.Body[0].Section[0].Paragraph) == 0 {
				t.Fatalf("no paragraph parsed from %q", tc.xml)
			}
			got := fb.Body[0].Section[0].Paragraph[0].FullParagraphText()
			if got != tc.want {
				t.Errorf("FullParagraphText() = %q, want %q (inline/tail text must be preserved)", got, tc.want)
			}
			// The same defect also corrupts markdown conversion.
			gotMD := extractTextFromParagraph(fb.Body[0].Section[0].Paragraph[0])
			if gotMD != tc.want {
				t.Errorf("extractTextFromParagraph() = %q, want %q", gotMD, tc.want)
			}
		})
	}
}
