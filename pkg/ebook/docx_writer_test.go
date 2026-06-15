package ebook

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readZipEntry(t, path, name) string is defined in format_regression_test.go and
// reused here. zipEntryNames is DOCX-test-local.

// §11.4.115 RED-baseline-on-the-broken-artifact + polarity switch.
//
// RED_MODE=1 (env unset/"1") reproduces the pre-impl defect: DOCX output is
// unimplemented, so NewDOCXWriter() does not exist and this file does not even
// compile against the pre-fix tree — the strongest possible RED (the whole
// package fails to build, proving the capability is genuinely absent).
//
// RED_MODE=0 is the standing GREEN regression guard: the writer produces a
// VALID .docx (correct OOXML part set + the translated chapter text inside
// word/document.xml). The mutation gate (break the writer) flips this RED again.
//
// Validity is asserted by REAL artifact inspection (unzip the produced .docx,
// assert [Content_Types].xml + word/document.xml present + the chapter text
// appears in document.xml) — never a metadata-only PASS (§11.4 / §11.4.107).

func zipEntryNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	require.NoError(t, err)
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

func TestDOCXWriter_ProducesValidDocx(t *testing.T) {
	book := &Book{
		Metadata: Metadata{
			Title:    "Mi Pequeño Libro",
			Authors:  []string{"Jane Translator"},
			Language: "es",
		},
		Chapters: []Chapter{
			{
				Title: "Capítulo Uno",
				Sections: []Section{
					{Title: "Sección 1.1", Content: "El zorro rápido salta sobre el perro perezoso.\n\nSegundo párrafo aquí."},
				},
			},
			{
				Title: "Capítulo Dos",
				Sections: []Section{
					{Content: "Contenido del segundo capítulo con caracteres especiales <&>."},
				},
			},
		},
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "book_es.docx")

	w := NewDOCXWriter()
	require.NoError(t, w.Write(book, out))

	// File exists and is non-empty.
	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0), "docx must be non-empty")

	// Required OOXML parts present (real zip inspection).
	names := zipEntryNames(t, out)
	for _, required := range []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"word/document.xml",
		"word/_rels/document.xml.rels",
	} {
		assert.Contains(t, names, required, "valid .docx must contain part %q", required)
	}

	// Content_Types declares the main document part content type.
	ct := readZipEntry(t, out, "[Content_Types].xml")
	assert.Contains(t, ct,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml",
		"[Content_Types].xml must declare the WordprocessingML main document content type")

	// document.xml carries the WordprocessingML namespace + body + the translated text.
	docStr := readZipEntry(t, out, "word/document.xml")
	assert.Contains(t, docStr, "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
		"document.xml must declare the w: namespace")
	assert.Contains(t, docStr, "<w:body>")
	assert.Contains(t, docStr, "<w:t")

	// The translated chapter title + body text MUST appear in document.xml
	// (the anti-bluff core: a valid-but-empty docx is not a translation).
	for _, want := range []string{
		"Mi Pequeño Libro",
		"Capítulo Uno",
		"El zorro rápido salta sobre el perro perezoso.",
		"Segundo párrafo aquí.",
		"Capítulo Dos",
		"Contenido del segundo capítulo con caracteres especiales",
	} {
		assert.Contains(t, docStr, want, "document.xml must contain translated text %q", want)
	}

	// XML special chars in content are escaped (no raw '<&>' injection).
	assert.NotContains(t, docStr, "especiales <&>",
		"XML special characters in content must be escaped, not raw")
	assert.Contains(t, docStr, "especiales &lt;&amp;&gt;",
		"content special chars must be XML-escaped inside <w:t>")
}

func TestDOCXWriter_EmptyBookStillValid(t *testing.T) {
	book := &Book{Metadata: Metadata{Title: "Empty"}}
	dir := t.TempDir()
	out := filepath.Join(dir, "empty.docx")
	w := NewDOCXWriter()
	require.NoError(t, w.Write(book, out))
	names := zipEntryNames(t, out)
	assert.Contains(t, names, "word/document.xml")
	doc := readZipEntry(t, out, "word/document.xml")
	assert.Contains(t, doc, "<w:body>")
	// A valid minimal document body must contain at least one block-level element.
	assert.True(t, strings.Contains(doc, "<w:p>") || strings.Contains(doc, "<w:p/>"),
		"minimal valid document.xml body must contain a paragraph")
}
