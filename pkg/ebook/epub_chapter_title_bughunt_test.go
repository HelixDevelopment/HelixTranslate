package ebook

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBugHunt_EPUB_ChapterTitleFromZipName is a §11.4.115 reproduce-first guard
// for a real EPUB round-trip data-loss defect:
//
//	The parser set chapter.Title to the INTERNAL zip entry name
//	(e.g. "OEBPS/chapter1.xhtml") instead of the chapter's real <title>/<h1>
//	text. In an EPUB -> translate -> EPUB flow the writer uses chapter.Title for
//	the <h1> heading AND the NCX navLabel, so every chapter's table-of-contents
//	entry and on-page heading became an internal filename — visible corruption
//	for the end user — while the real title was silently folded into body text
//	and lost as a title.
//
// RED on the pre-fix code: chapter.Title == "OEBPS/chapter1.xhtml".
// GREEN after the fix:     chapter.Title == "The Real Chapter Title".
func TestBugHunt_EPUB_ChapterTitleFromZipName(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "real.epub")
	f, err := os.Create(fn)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)

	mw, _ := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	_, _ = mw.Write([]byte("application/epub+zip"))

	cw, _ := zw.Create("META-INF/container.xml")
	_, _ = cw.Write([]byte(`<?xml version="1.0"?><container version="1.0" ` +
		`xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles>` +
		`<rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>` +
		`</rootfiles></container>`))

	ow, _ := zw.Create("OEBPS/content.opf")
	_, _ = ow.Write([]byte(`<?xml version="1.0"?><package ` +
		`xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="i">` +
		`<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">` +
		`<dc:title>Book</dc:title><dc:identifier id="i">x</dc:identifier></metadata>` +
		`<manifest><item id="c1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>` +
		`</manifest><spine><itemref idref="c1"/></spine></package>`))

	chw, _ := zw.Create("OEBPS/chapter1.xhtml")
	_, _ = chw.Write([]byte(`<?xml version="1.0"?>` +
		`<html xmlns="http://www.w3.org/1999/xhtml"><head>` +
		`<title>The Real Chapter Title</title></head>` +
		`<body><h1>The Real Chapter Title</h1><p>Body paragraph.</p></body></html>`))

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := NewEPUBParser().Parse(fn)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(out.Chapters))
	}

	got := out.Chapters[0].Title
	if got == "OEBPS/chapter1.xhtml" || strings.HasSuffix(got, ".xhtml") {
		t.Fatalf("DATA LOSS: chapter title is the internal zip entry name %q; "+
			"the real <title>/<h1> text was lost as the title and a round-trip "+
			"EPUB would show this filename as the chapter heading + TOC entry", got)
	}
	if got != "The Real Chapter Title" {
		t.Fatalf("chapter title = %q, want %q", got, "The Real Chapter Title")
	}

	// The real title text must still survive in the readable body content too
	// (no content regression).
	if !strings.Contains(out.Chapters[0].Sections[0].Content, "Body paragraph.") {
		t.Fatalf("body content lost: %q", out.Chapters[0].Sections[0].Content)
	}
}
