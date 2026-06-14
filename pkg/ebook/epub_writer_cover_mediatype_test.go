package ebook

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/format"
)

// BUG (FACT, reproduced): the EPUB writer hardcodes EVERY cover image as
// "OEBPS/cover.jpg" with manifest media-type "image/jpeg" (epub_writer.go
// writeCover + the cover manifest item + the cover meta), regardless of the
// actual image format carried in book.Metadata.Cover. The EPUB parser
// (epub_parser.go extractCoverImage) reads the cover bytes VERBATIM from the
// source — and PNG covers (cover.png) are ubiquitous in real EPUBs. On a
// round-trip a PNG cover therefore ships as a file named cover.jpg whose
// manifest media-type claims image/jpeg while the bytes are PNG. EPUB readers
// that trust the declared media-type (or the .jpg extension) cannot decode the
// image, so the cover renders broken/blank for the end user — silent cover
// corruption.
//
// This test feeds a real PNG-signature cover and asserts the produced EPUB
// declares + names the cover according to the ACTUAL format (PNG), and never
// mislabels it as JPEG.

// pngHeader is a minimal valid PNG magic signature + IHDR start. The first 8
// bytes are the canonical PNG signature used for format sniffing.
var pngHeader = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 'I', 'H', 'D', 'R'}

func coverZipEntry(t *testing.T, filename, entry string) (string, bool) {
	t.Helper()
	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == entry {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open entry %s: %v", entry, err)
			}
			data, _ := io.ReadAll(rc)
			rc.Close()
			return string(data), true
		}
	}
	return "", false
}

func coverZipHasEntry(t *testing.T, filename, entry string) bool {
	t.Helper()
	_, ok := coverZipEntry(t, filename, entry)
	return ok
}

func TestBugHunt_EPUBWriter_PNGCoverNotMislabeledAsJPEG(t *testing.T) {
	book := &Book{
		Metadata: Metadata{
			Title: "PNG Cover Book",
			Cover: pngHeader,
		},
		Chapters: []Chapter{{Title: "Ch1", Sections: []Section{{Content: "hi"}}}},
		Format:   format.FormatEPUB,
	}

	tmp, err := os.CreateTemp("", "png_cover*.epub")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := NewEPUBWriter().Write(book, tmp.Name()); err != nil {
		t.Fatalf("write: %v", err)
	}

	opf, ok := coverZipEntry(t, tmp.Name(), "OEBPS/content.opf")
	if !ok {
		t.Fatal("content.opf missing")
	}

	// The manifest MUST NOT declare a PNG cover as image/jpeg.
	if strings.Contains(opf, `media-type="image/jpeg"`) {
		t.Errorf("PNG cover mislabeled as image/jpeg in OPF manifest:\n%s", opf)
	}
	// The manifest MUST declare the correct media-type for PNG.
	if !strings.Contains(opf, `media-type="image/png"`) {
		t.Errorf("OPF manifest does not declare image/png for a PNG cover:\n%s", opf)
	}

	// The cover file MUST be stored with a PNG-correct name, and a .jpg copy of
	// PNG bytes MUST NOT exist (extension contradicting content).
	if !coverZipHasEntry(t, tmp.Name(), "OEBPS/cover.png") {
		t.Errorf("PNG cover not stored as OEBPS/cover.png")
	}
	if coverZipHasEntry(t, tmp.Name(), "OEBPS/cover.jpg") {
		t.Errorf("PNG bytes stored under contradicting OEBPS/cover.jpg name")
	}

	// The cover manifest href MUST match the stored file (no dangling reference).
	if !strings.Contains(opf, `href="cover.png"`) {
		t.Errorf("OPF cover item href does not point at cover.png:\n%s", opf)
	}
}

// Guard: JPEG covers must still work and be labeled image/jpeg as before (the
// fix must not regress the common JPEG path).
func TestBugHunt_EPUBWriter_JPEGCoverStillJPEG(t *testing.T) {
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	book := &Book{
		Metadata: Metadata{Title: "JPEG Cover Book", Cover: jpegHeader},
		Chapters: []Chapter{{Title: "Ch1", Sections: []Section{{Content: "hi"}}}},
		Format:   format.FormatEPUB,
	}
	tmp, err := os.CreateTemp("", "jpeg_cover*.epub")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if err := NewEPUBWriter().Write(book, tmp.Name()); err != nil {
		t.Fatalf("write: %v", err)
	}
	opf, _ := coverZipEntry(t, tmp.Name(), "OEBPS/content.opf")
	if !strings.Contains(opf, `media-type="image/jpeg"`) {
		t.Errorf("JPEG cover not labeled image/jpeg:\n%s", opf)
	}
	if !coverZipHasEntry(t, tmp.Name(), "OEBPS/cover.jpg") {
		t.Errorf("JPEG cover not stored as OEBPS/cover.jpg")
	}
	if !strings.Contains(opf, `href="cover.jpg"`) {
		t.Errorf("OPF cover item href does not point at cover.jpg:\n%s", opf)
	}
}
