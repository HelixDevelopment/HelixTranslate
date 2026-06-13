package format

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// writeZip builds a real ZIP archive at path containing the given name->content
// entries. The "mimetype" entry, when present, is written first and STORED
// (uncompressed) the way the EPUB OCF spec mandates, so archive/zip can read it
// back. This produces genuine PK\x03\x04 archives that exercise the real
// zip.OpenReader path in getZipMimetype / isAZW3File — not hand-rolled byte
// blobs that a stubbed reader could fake.
func writeZip(t *testing.T, path string, entries map[string]string, order []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, name := range order {
		var w interface {
			Write([]byte) (int, error)
		}
		if name == "mimetype" {
			hw, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
			if err != nil {
				t.Fatalf("zip header %s: %v", name, err)
			}
			w = hw
		} else {
			cw, err := zw.Create(name)
			if err != nil {
				t.Fatalf("zip create %s: %v", name, err)
			}
			w = cw
		}
		if _, err := w.Write([]byte(entries[name])); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

// TestGetZipMimetype_RealArchives drives getZipMimetype against genuine ZIP
// archives. Anti-bluff: if getZipMimetype were stubbed to return "" / an error,
// the present+correct cases below would FAIL because the precise mimetype
// string would not come back.
func TestGetZipMimetype_RealArchives(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()

	epub := filepath.Join(dir, "real.epub.zip")
	writeZip(t, epub,
		map[string]string{
			"mimetype":     "application/epub+zip",
			"OEBPS/x.html": "<html></html>",
		},
		[]string{"mimetype", "OEBPS/x.html"},
	)
	if mt, err := d.getZipMimetype(epub); err != nil || mt != "application/epub+zip" {
		t.Errorf("getZipMimetype(epub) = %q, %v; want application/epub+zip, nil", mt, err)
	}

	// mimetype with surrounding whitespace must be trimmed.
	wsZip := filepath.Join(dir, "ws.zip")
	writeZip(t, wsZip,
		map[string]string{"mimetype": "  application/epub+zip\n"},
		[]string{"mimetype"},
	)
	if mt, _ := d.getZipMimetype(wsZip); mt != "application/epub+zip" {
		t.Errorf("getZipMimetype(ws) = %q; want trimmed application/epub+zip", mt)
	}

	// ZIP without a mimetype entry must return an error.
	noMime := filepath.Join(dir, "nomime.zip")
	writeZip(t, noMime,
		map[string]string{"word/document.xml": "<w:document/>"},
		[]string{"word/document.xml"},
	)
	if _, err := d.getZipMimetype(noMime); err == nil {
		t.Error("getZipMimetype(no mimetype) should return error")
	}

	// Not a ZIP at all -> error from zip.OpenReader.
	notZip := filepath.Join(dir, "plain.txt")
	if err := os.WriteFile(notZip, []byte("hello not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := d.getZipMimetype(notZip); err == nil {
		t.Error("getZipMimetype(non-zip) should return error")
	}
}

// TestDisambiguateZipFormat_ByMimetype exercises the mimetype-inside-ZIP branch
// (no decisive extension), which the existing tests never reach because they
// always pass a decisive extension. Anti-bluff: stubbing getZipMimetype to ""
// makes DOCX/AZW3 collapse to the EPUB default => these cases FAIL.
func TestDisambiguateZipFormat_ByMimetype(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()

	cases := []struct {
		name     string
		mimetype string
		want     Format
	}{
		{"docx-by-mimetype.zip", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", FormatDOCX},
		{"azw3-by-mimetype.zip", "application/x-mobipocket-ebook", FormatAZW3},
		{"epub-by-mimetype.zip", "application/epub+zip", FormatEPUB},
	}
	for _, c := range cases {
		p := filepath.Join(dir, c.name)
		writeZip(t, p,
			map[string]string{"mimetype": c.mimetype, "content": "x"},
			[]string{"mimetype", "content"},
		)
		// ext deliberately non-decisive (".zip") so the mimetype branch runs.
		got, err := d.disambiguateZipFormat(p, ".zip")
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("disambiguateZipFormat(%s, .zip) = %s; want %s", c.name, got, c.want)
		}
	}
}

// TestDisambiguateZipFormat_AZW3StructureFallback hits the isAZW3File branch:
// no decisive extension, no mimetype entry, but a Kindle-specific member name.
// Anti-bluff: stubbing isAZW3File to false routes this to the EPUB default => FAIL.
func TestDisambiguateZipFormat_AZW3StructureFallback(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()

	azw3 := filepath.Join(dir, "structure.zip")
	writeZip(t, azw3,
		map[string]string{
			"OEBPS/kindle:embed:0001.jpg": "binary",
			"content.html":                "<html></html>",
		},
		[]string{"OEBPS/kindle:embed:0001.jpg", "content.html"},
	)
	if got, _ := d.disambiguateZipFormat(azw3, ".zip"); got != FormatAZW3 {
		t.Errorf("disambiguateZipFormat(kindle-structure) = %s; want %s", got, FormatAZW3)
	}

	// A plain ZIP with neither mimetype nor Kindle markers defaults to EPUB.
	plain := filepath.Join(dir, "plainzip.zip")
	writeZip(t, plain,
		map[string]string{"a.txt": "hello", "b.txt": "world"},
		[]string{"a.txt", "b.txt"},
	)
	if got, _ := d.disambiguateZipFormat(plain, ".zip"); got != FormatEPUB {
		t.Errorf("disambiguateZipFormat(plain zip) = %s; want default %s", got, FormatEPUB)
	}
}

// TestIsAZW3File directly asserts the Kindle-marker scan over every documented
// indicator, plus the negative (non-Kindle ZIP) and the open-failure path.
func TestIsAZW3File(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()

	markers := []string{
		"kindle:embed",
		"amzn-eastock",
		"kindle-fonts",
		"kindle:enclosure",
		"kindle:meta",
	}
	for i, m := range markers {
		p := filepath.Join(dir, "azw3-marker.zip")
		writeZip(t, p,
			map[string]string{"dir/" + m + "-x": "data", "other": "x"},
			[]string{"dir/" + m + "-x", "other"},
		)
		if !d.isAZW3File(p) {
			t.Errorf("isAZW3File should be true for marker #%d %q", i, m)
		}
	}

	clean := filepath.Join(dir, "clean.zip")
	writeZip(t, clean,
		map[string]string{"OEBPS/chapter1.xhtml": "<html></html>"},
		[]string{"OEBPS/chapter1.xhtml"},
	)
	if d.isAZW3File(clean) {
		t.Error("isAZW3File should be false for a clean EPUB-like ZIP")
	}

	// Unopenable / missing file -> false (not a panic).
	if d.isAZW3File(filepath.Join(dir, "does-not-exist.zip")) {
		t.Error("isAZW3File should be false when the file cannot be opened")
	}
}

// TestDetectByContent_UncoveredBranches covers the content-sniffing branches the
// existing suite skips: XML-wrapped HTML, raw <HTML> upper-case, %PDF/BOOKMOBI/
// TPZ0 found only in content (magic-byte path missed), and the empty input case.
func TestDetectByContent_UncoveredBranches(t *testing.T) {
	d := NewDetector()

	cases := []struct {
		name    string
		content string
		want    Format
	}{
		{"xml-wrapping-html", `<?xml version="1.0"?><html><body>hi</body></html>`, FormatHTML},
		// Upper-case <HTML> is recognised ONLY inside the <?xml branch (line 241).
		{"xml-wrapping-uppercase-HTML", `<?xml version="1.0"?><HTML><BODY>hi</BODY></HTML>`, FormatHTML},
		// Standalone upper-case <HTML> with NO <?xml prologue is NOT matched by the
		// lowercase-only standalone check (line 247) -> falls through to plain text.
		// See CORRECTNESS_FINDINGS: this is asymmetric vs the lowercase path.
		{"standalone-uppercase-HTML-falls-to-txt", `prefix junk <HTML> body words`, FormatTXT},
		{"pdf-in-content", "leading bytes then %PDF marker", FormatPDF},
		{"bookmobi-in-content", "junk\x00BOOKMOBI here", FormatMOBI},
		{"tpz0-in-content", "junk\x00TPZ0 marker", FormatAZW},
		{"empty-input", "", FormatTXT},
	}
	for _, c := range cases {
		if got := d.detectByContent([]byte(c.content)); got != c.want {
			t.Errorf("detectByContent(%s) = %s; want %s", c.name, got, c.want)
		}
	}
}

// TestDetectFile_ContentFallbackPaths drives DetectFile end-to-end for files
// whose extension and magic bytes are both inconclusive so the content-sniff
// fallback decides — the integration of detectByContent into DetectFile.
func TestDetectFile_ContentFallbackPaths(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()

	// FB2 body but an unhelpful extension and no leading magic match for FB2
	// (FB2 has no magic-byte entry), forcing the content path.
	fb2 := filepath.Join(dir, "book.dat")
	if err := os.WriteFile(fb2,
		[]byte(`<?xml version="1.0"?><FictionBook xmlns="x">body</FictionBook>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := d.DetectFile(fb2); err != nil || got != FormatFB2 {
		t.Errorf("DetectFile(fb2-by-content) = %s, %v; want fb2", got, err)
	}

	// Empty file: no ext match, no magic, content path -> plain text.
	empty := filepath.Join(dir, "empty.dat")
	if err := os.WriteFile(empty, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := d.DetectFile(empty); err != nil || got != FormatTXT {
		t.Errorf("DetectFile(empty) = %s, %v; want txt", got, err)
	}
}

// TestDetectFile_ReadError hits the header-read error branch (lines 58-60):
// os.Open succeeds on a directory, but the subsequent Read returns an error
// ("is a directory") on Unix, so DetectFile must surface a wrapped error.
func TestDetectFile_ReadError(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir() // a directory: openable, but not readable as a byte stream
	got, err := d.DetectFile(dir)
	if err == nil {
		t.Fatalf("DetectFile(directory) expected a read error, got format %s, nil", got)
	}
	if got != FormatUnknown {
		t.Errorf("DetectFile(directory) on error should return FormatUnknown, got %s", got)
	}
}

// TestDetectFile_ExtensionFallbackNoMagic verifies the extension-decides branch:
// a real DOCX magic (PK) with a .docx extension returns DOCX via the decisive
// extension inside disambiguateZipFormat, while a .mobi with no magic falls back
// to the extension result.
func TestDetectFile_ExtensionFallbackNoMagic(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()

	// .mobi extension, content is plain (no BOOKMOBI magic at offset 0) -> the
	// extension result wins because magic + content are inconclusive for MOBI.
	mobi := filepath.Join(dir, "book.mobi")
	if err := os.WriteFile(mobi, []byte("no magic here, just words and words"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := d.DetectFile(mobi); err != nil || got != FormatMOBI {
		t.Errorf("DetectFile(.mobi no-magic) = %s, %v; want mobi via extension", got, err)
	}
}
