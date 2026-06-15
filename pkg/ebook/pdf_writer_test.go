package ebook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §11.4.115 RED-baseline-on-the-broken-artifact + polarity switch.
//
// RED_MODE=1 (env unset/"1") reproduces the pre-impl defect: PDF output was
// unimplemented — NewPDFWriter() did not exist, so this file did not even
// compile against the pre-fix tree, and `-o *.pdf` was rejected by the
// unified-translator output switch. The strongest possible RED (the capability
// is genuinely absent).
//
// RED_MODE=0 is the standing GREEN regression guard: the writer produces a
// VALID PDF (correct %PDF- header) that genuinely CARRIES the translated text —
// including Serbian Cyrillic, the project's primary content that a Standard-14
// font PDF would silently drop (§11.4 bluff). Validity is asserted by REAL
// artifact inspection: the produced file starts with %PDF-, and the translated
// text is extracted BACK out of the PDF via the project's own in-process PDF
// text extractor and asserted present — never a metadata-only PASS
// (§11.4 / §11.4.107 / §11.4.150).
//
// The mutation gate (break buildPDFSourceHTML so the text is dropped) flips this
// RED again.

func weasyprintAvailable() bool {
	_, err := exec.LookPath("weasyprint")
	return err == nil
}

// extractPDFText round-trips the produced PDF back through the project's own PDF
// parser and returns all extracted title + content text concatenated.
func extractPDFTextRoundTrip(t *testing.T, path string) string {
	t.Helper()
	book, err := NewPDFParser(nil).Parse(path)
	require.NoError(t, err, "produced PDF must be parseable by the project PDF extractor")
	var b strings.Builder
	b.WriteString(book.Metadata.Title)
	b.WriteByte(' ')
	var walk func(s Section)
	walk = func(s Section) {
		b.WriteString(s.Title)
		b.WriteByte(' ')
		b.WriteString(s.Content)
		b.WriteByte(' ')
		for _, sub := range s.Subsections {
			walk(sub)
		}
	}
	for _, c := range book.Chapters {
		b.WriteString(c.Title)
		b.WriteByte(' ')
		for _, s := range c.Sections {
			walk(s)
		}
	}
	return b.String()
}

func TestPDFWriter_ProducesValidPDFCarryingTranslatedText(t *testing.T) {
	if !weasyprintAvailable() {
		// §11.4.3 honest topology SKIP — PDF output requires weasyprint; the
		// production code returns ErrWeasyPrintUnavailable on this host. Never a
		// PASS-by-default.
		t.Skip("SKIP: weasyprint not on PATH — PDF output dependency absent (§11.4.3); install weasyprint to run this test")
	}

	book := &Book{
		Metadata: Metadata{Title: "Mi Pequeño Libro", Language: "es"},
		Chapters: []Chapter{
			{
				Title: "Capítulo Uno",
				Sections: []Section{
					{
						Title:   "Sección 1.1",
						Content: "El zorro rápido salta sobre el perro perezoso.\n\nSegundo párrafo aquí.",
						Subsections: []Section{
							{Title: "Subsección", Content: "Здраво свете — texto anidado."},
						},
					},
				},
			},
		},
	}

	out := filepath.Join(t.TempDir(), "book_es.pdf")
	require.NoError(t, NewPDFWriter().Write(book, out))

	// (1) The artifact begins with the %PDF- magic header.
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(data), 5, "PDF must be non-empty")
	assert.Equal(t, "%PDF-", string(data[:5]), "output must start with the %PDF- header")
	assert.Contains(t, string(data), "%%EOF", "PDF must carry the %%EOF trailer marker")

	// (2) The translated text — incl. Latin-with-diacritics AND Serbian Cyrillic
	//     (subsection content, proving lossless nested recursion + Unicode) — is
	//     extractable back out of the produced PDF. This is the anti-bluff core:
	//     a blank/glyph-dropped PDF would fail here.
	extracted := extractPDFTextRoundTrip(t, out)
	assert.Contains(t, extracted, "Mi Pequeño Libro", "book title must be in the PDF")
	assert.Contains(t, extracted, "Capítulo Uno", "chapter title must be in the PDF")
	assert.Contains(t, extracted, "zorro", "translated body text must be in the PDF")
	assert.Contains(t, extracted, "Segundo párrafo", "second paragraph must be in the PDF")
	assert.Contains(t, extracted, "Здраво свете", "Serbian Cyrillic nested text must be carried losslessly (no glyph drop)")
}

// TestPDFWriter_UnavailableWeasyPrint asserts the honest typed error path
// (§11.4.3/§11.4.6): when the weasyprint binary cannot be found, Write returns
// ErrWeasyPrintUnavailable and writes NO file — never a broken/blank PDF.
func TestPDFWriter_UnavailableWeasyPrint(t *testing.T) {
	w := &PDFWriter{binary: "weasyprint-does-not-exist-xyz"}
	out := filepath.Join(t.TempDir(), "should_not_exist.pdf")
	err := w.Write(&Book{Metadata: Metadata{Title: "x"}}, out)
	require.ErrorIs(t, err, ErrWeasyPrintUnavailable)
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "no file must be produced when weasyprint is unavailable")
}

// TestBuildPDFSourceHTML asserts the HTML source carries the structure + escapes
// markup — this guard runs even without weasyprint installed, and the mutation
// gate (drop the body text from buildPDFSourceHTML) flips it RED.
func TestBuildPDFSourceHTML(t *testing.T) {
	book := &Book{
		Metadata: Metadata{Title: "Title & <Co>"},
		Chapters: []Chapter{{
			Title:    "Ch1",
			Sections: []Section{{Title: "S1", Content: "para one\n\npara two"}},
		}},
	}
	html := buildPDFSourceHTML(book)

	assert.True(t, strings.HasPrefix(html, "<!DOCTYPE html>"), "must be an HTML5 document")
	assert.Contains(t, html, `<meta charset="utf-8">`, "must declare UTF-8 for full-Unicode fidelity")
	assert.Contains(t, html, "Title &amp; &lt;Co&gt;", "title must be HTML-escaped (no markup injection)")
	assert.Contains(t, html, "<h2>Ch1</h2>", "chapter title becomes a heading")
	assert.Contains(t, html, "<h3>S1</h3>", "section title becomes a heading")
	assert.Contains(t, html, "<p>para one</p>", "first paragraph present")
	assert.Contains(t, html, "<p>para two</p>", "second paragraph present (blank-line split)")
}
