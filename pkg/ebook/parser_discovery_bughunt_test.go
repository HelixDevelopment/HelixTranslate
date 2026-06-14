package ebook

import (
	"strings"
	"testing"
)

// Bug A: EPUB manifest/spine hrefs are URI-encoded per the OCF spec, but zip
// entry names are literal. A chapter file whose name contains a space is
// referenced as "chapter%20one.xhtml"; naive opfDir+href concatenation never
// matches the literal zip entry "OEBPS/chapter one.xhtml" -> the chapter is
// SILENTLY DROPPED (data loss). RED reproduces the loss on current code.
func TestBugHunt_EPUB_URLEncodedHref_ChapterDropped(t *testing.T) {
	parser := NewEPUBParser()

	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

	// href is percent-encoded ("chapter%20one.xhtml") as required by the spec.
	contentOPF := `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>T</dc:title></metadata>
	<manifest>
		<item id="c1" href="chapter%20one.xhtml" media-type="application/xhtml+xml"/>
	</manifest>
	<spine><itemref idref="c1"/></spine>
</package>`

	chapterXHTML := `<html><body><p>UNIQUE_CHAPTER_BODY_MARKER</p></body></html>`

	files := []zipFile{
		{Name: "META-INF/container.xml", Content: containerXML},
		{Name: "OEBPS/content.opf", Content: contentOPF},
		// Literal zip entry name has a real space, not %20.
		{Name: "OEBPS/chapter one.xhtml", Content: chapterXHTML},
	}

	tmpFile, err := createTempZipFile(t, "test_urlenc.epub", files)
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	book, err := parser.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(book.Chapters) != 1 {
		t.Fatalf("chapter dropped: got %d chapters, want 1 (URL-encoded href not resolved)", len(book.Chapters))
	}
	if !strings.Contains(book.Chapters[0].Sections[0].Content, "UNIQUE_CHAPTER_BODY_MARKER") {
		t.Errorf("chapter body lost: %q", book.Chapters[0].Sections[0].Content)
	}
}

// Bug B: extractText hardcodes a substring replacement
// ("Nestedtexthere" -> "Nested text here"). Any real document whose text
// legitimately contains that substring is CORRUPTED. RED feeds a paragraph
// containing the literal token and asserts it survives unchanged.
func TestBugHunt_HTML_HardcodedStringCorruption(t *testing.T) {
	parser := NewHTMLParser()
	tmpFile := createTempHTMLFile(t, "test_hardcoded.html", `<html><body><p>Nestedtexthere</p></body></html>`)
	defer removeTempFile(t, tmpFile)

	book, err := parser.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	got := book.Chapters[0].Sections[0].Content
	if !strings.Contains(got, "Nestedtexthere") {
		t.Errorf("real content corrupted by hardcoded replacement: got %q, want it to contain %q", got, "Nestedtexthere")
	}
}

// Bug A (extend, §11.4.146): the same unresolved-href data loss hits the COVER
// image (also percent-encoded) AND OPF-relative "../" hrefs (OPF in a subdir,
// content one level up). Both were dropped by opfDir+href concatenation.
func TestBugHunt_EPUB_URLEncodedCover_AndRelativeHref(t *testing.T) {
	parser := NewEPUBParser()

	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	<rootfiles><rootfile full-path="OEBPS/sub/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

	// content href climbs out of the OPF subdir via "../"; cover is %20-encoded.
	contentOPF := `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
	<metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>T</dc:title>
		<meta name="cover" content="cov"/>
	</metadata>
	<manifest>
		<item id="c1" href="../text/ch1.xhtml" media-type="application/xhtml+xml"/>
		<item id="cov" href="my%20cover.jpg" media-type="image/jpeg"/>
	</manifest>
	<spine><itemref idref="c1"/></spine>
</package>`

	files := []zipFile{
		{Name: "META-INF/container.xml", Content: containerXML},
		{Name: "OEBPS/sub/content.opf", Content: contentOPF},
		{Name: "OEBPS/text/ch1.xhtml", Content: `<html><body><p>REL_HREF_BODY</p></body></html>`},
		{Name: "OEBPS/sub/my cover.jpg", Content: "COVERBYTES"},
	}

	tmpFile, err := createTempZipFile(t, "test_relcover.epub", files)
	if err != nil {
		t.Fatal(err)
	}
	defer removeTempFile(t, tmpFile)

	book, err := parser.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(book.Chapters) != 1 || !strings.Contains(book.Chapters[0].Sections[0].Content, "REL_HREF_BODY") {
		t.Errorf("relative '../' content href not resolved: chapters=%d", len(book.Chapters))
	}
	if string(book.Metadata.Cover) != "COVERBYTES" {
		t.Errorf("percent-encoded cover href not resolved: cover=%q", string(book.Metadata.Cover))
	}
}
