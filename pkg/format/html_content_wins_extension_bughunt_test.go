package format

import (
	"os"
	"path/filepath"
	"testing"
)

// §11.4.118 discovery + §11.4.115 RED-baseline polarity test.
//
// DEFECT (FACT, found 2026-06-16): DetectFile classified an unmistakable
// (X)HTML document carrying a generic .txt extension as FormatTXT instead of
// FormatHTML. Root cause: DetectFile returned the extension-derived format
// whenever the extension was a *known* one (.txt -> FormatTXT) BEFORE ever
// reaching detectByContent, which DOES correctly identify the markup as HTML.
// This is the exact sibling of the FB2-loses-to-generic-extension bug the
// isFB2Content() guard was added to fix: an HTML book mislabelled .txt would be
// handed to the plain-text parser and its markup mangled (heading/paragraph
// structure flattened, tags emitted as literal prose) — the same silent
// data-loss class.
//
// Polarity switch: RED_MODE=1 reproduces the defect on the pre-fix behaviour
// expectation (documents what was broken); the standing assertions (RED_MODE
// unset / "0") assert the FIXED behaviour — HTML content wins over a generic
// extension, while genuine plain text NEVER gets reclassified.

func writeProbeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func TestDetectFile_HTMLContentWinsGenericExtension(t *testing.T) {
	d := NewDetector()
	bom := []byte{0xEF, 0xBB, 0xBF}

	type tc struct {
		name string
		data []byte
		want Format
	}
	cases := []tc{
		{
			name: "xhtml_prolog.txt",
			data: []byte(`<?xml version="1.0" encoding="utf-8"?>` +
				`<html xmlns="http://www.w3.org/1999/xhtml"><head><title>T</title></head>` +
				`<body><h1>Chapter</h1><p>Body text.</p></body></html>`),
			want: FormatHTML,
		},
		{
			name: "doctype_html.txt",
			data: []byte("<!DOCTYPE html>\n<html><head></head><body><p>hi</p></body></html>"),
			want: FormatHTML,
		},
		{
			name: "root_html_noext",
			data: []byte("<html><head></head><body><p>plain html no doctype</p></body></html>"),
			want: FormatHTML,
		},
		{
			name: "bom_doctype_html.txt",
			data: append(append([]byte{}, bom...), []byte("<!DOCTYPE html><html><body>x</body></html>")...),
			want: FormatHTML,
		},
		// Negative cases — genuine plain text MUST stay TXT, even if it mentions
		// angle brackets / the word html in passing. The HTML-content guard must
		// be anchored, never a substring sniff that hijacks prose.
		{
			name: "plain_prose.txt",
			data: []byte("This is an ordinary paragraph. It talks about <html> tags in prose,\n" +
				"like <p> and </p>, but it is not a markup document.\n"),
			want: FormatTXT,
		},
		{
			name: "plain_prose_noext",
			data: []byte("Just words. The author once wrote html for a living. Nothing structural here.\n"),
			want: FormatTXT,
		},
	}

	red := os.Getenv("RED_MODE") == "1"
	for _, c := range cases {
		p := writeProbeFile(t, c.name, c.data)
		got, err := d.DetectFile(p)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if red {
			// Pre-fix reproduction: the .txt-extension HTML cases came back TXT.
			t.Logf("RED %-22s -> %s (want %s)", c.name, got, c.want)
			continue
		}
		if got != c.want {
			t.Errorf("%s: DetectFile = %s, want %s (HTML content must win a generic extension; plain text must stay TXT)",
				c.name, got, c.want)
		}
	}
}
