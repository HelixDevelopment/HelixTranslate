package markdown

// §11.4.146 reproduce-first / §11.4.135 standing regression guards for the three
// deferred markdown-workflow defects:
//
//   D-INLINE-WS  : convertNode collapsed the whitespace separating adjacent inline
//                  elements ("<strong>bold</strong> tail" -> "**bold**tail").
//   D-CHAPTER-DROP: convertHTMLToMarkdown error path silently dropped a chapter
//                  (the EPUB->MD loop's `if err == nil` swallowed the error).
//   D-LIST-RT    : the reverse path (markdownToHTML / convertMarkdownToXHTML) had
//                  no list parsing, so EPUB->MD->EPUB lost every list.
//
// Each test asserts a concrete, user-visible outcome and would fail if the fix
// were reverted (mutation-proven §1.1).

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// D-INLINE-WS — inter-element whitespace must be preserved.
// ---------------------------------------------------------------------------

// TestInlineWhitespacePreserved is the permanent guard for the inline
// whitespace-collapse defect. Adjacent inline elements separated by a space in
// the source HTML MUST keep that separating space in the rendered markdown.
//
// Mutation proof (§1.1): reverting convertNode's text-node handling to a bare
// strings.TrimSpace(n.Data) makes every "with separating space" assertion FAIL.
func TestInlineWhitespacePreserved(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string // exact trimmed markdown
	}{
		{
			"strong-then-tail",
			"<p><strong>bold</strong> tail</p>",
			"**bold** tail",
		},
		{
			"lead-then-strong",
			"<p>lead <strong>bold</strong></p>",
			"lead **bold**",
		},
		{
			"em-between-words",
			"<p>a <em>b</em> c</p>",
			"a *b* c",
		},
		{
			"two-strongs-spaced",
			"<p><strong>one</strong> <strong>two</strong></p>",
			"**one** **two**",
		},
		{
			"code-then-word",
			"<p>run <code>cmd</code> now</p>",
			"run `cmd` now",
		},
		{
			"link-then-word",
			`<p>see <a href="u">here</a> please</p>`,
			"see [here](u) please",
		},
		{
			// No space in source -> no space introduced (must not over-correct).
			"adjacent-no-space",
			"<p><strong>a</strong><strong>b</strong></p>",
			"**a****b**",
		},
		{
			// Leading/trailing block whitespace must still be trimmed (no regression
			// of the block-whitespace behaviour the original TrimSpace guarded).
			"surrounding-block-ws-trimmed",
			"<p>   hello   world   </p>",
			"hello world",
		},
		{
			"multi-inline-mixed",
			"<p>The <strong>quick</strong> <em>brown</em> <code>fox</code> jumps</p>",
			"The **quick** *brown* `fox` jumps",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := htmlToMarkdown(t, tc.html)
			if got != tc.want {
				t.Fatalf("inline whitespace lost/corrupted\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// D-CHAPTER-DROP — a chapter conversion error must be observable, never silent.
// ---------------------------------------------------------------------------

// TestChapterConversionErrorSurfaced proves the EPUB->MD chapter loop no longer
// silently drops a chapter whose HTML fails to read/parse. A real
// ConvertEPUBToMarkdown run over an EPUB containing one unreadable chapter must
// return a non-nil error (the failure is observable), while still writing the
// chapters that DID convert (no data loss for the good chapters).
//
// Root cause: the chapter loop used `if err == nil && chapterMD != ""`, which
// dropped a failing chapter with no log, no error, no signal.
//
// Mutation proof (§1.1): restoring the silent `if err == nil` swallow (dropping
// the collected error) makes the "error surfaced" assertion FAIL.
func TestChapterConversionErrorSurfaced(t *testing.T) {
	dir := t.TempDir()
	epubPath := dir + "/with_bad_chapter.epub"
	outMD := dir + "/out.md"

	// The corrupt chapter carries a long, unique marker payload so the byte-flip
	// lands deep inside its STORED data region (never in the zip headers or the
	// central directory), guaranteeing a read-time CRC error for THAT entry only.
	corruptMarker := strings.Repeat("CORRUPTME", 64)
	buildEPUBWithChapters(t, epubPath, []epubChapter{
		{id: "chapter1", href: "chapter1.xhtml",
			content: "<html><body><h2>Good One</h2><p>Readable body.</p></body></html>",
			corrupt: false},
		{id: "chapter2", href: "chapter2.xhtml",
			content: "<html><body><h2>Bad</h2><p>" + corruptMarker + "</p></body></html>",
			corrupt: true, marker: corruptMarker},
		{id: "chapter3", href: "chapter3.xhtml",
			content: "<html><body><h2>Good Three</h2><p>Also readable.</p></body></html>",
			corrupt: false},
	})

	c := NewEPUBToMarkdownConverter(false, dir+"/imgs")
	err := c.ConvertEPUBToMarkdown(epubPath, outMD)
	if err == nil {
		t.Fatalf("ConvertEPUBToMarkdown must surface the failed chapter as an error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "chapter") {
		t.Fatalf("error must identify the failed chapter, got: %v", err)
	}

	// The good chapters MUST still have been written (best-effort, no data loss
	// for the chapters that converted cleanly).
	out := readFile(t, outMD)
	assertContainsAll(t, out, "Good One", "Readable body.", "Good Three", "Also readable.")
}

type epubChapter struct {
	id, href, content, marker string
	corrupt                   bool
}

// buildEPUBWithChapters writes a minimal valid EPUB whose spine lists every
// chapter. A chapter marked corrupt is STORED (no compression) and then a byte
// inside its unique marker payload is flipped, so reading that ONE entry back
// fails a CRC check (io.ReadAll -> zip: checksum error) while the rest of the
// archive (including the central directory) stays valid.
func buildEPUBWithChapters(t *testing.T, path string, chapters []epubChapter) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name, body string, method uint16) {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}

	add("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`, zip.Deflate)

	var manifest, spine strings.Builder
	for _, ch := range chapters {
		manifest.WriteString(`    <item id="` + ch.id + `" href="` + ch.href + `" media-type="application/xhtml+xml"/>` + "\n")
		spine.WriteString(`    <itemref idref="` + ch.id + `"/>` + "\n")
	}
	add("OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="BookID">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Bad Chapter Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="BookID">urn:uuid:test</dc:identifier>
  </metadata>
  <manifest>
`+manifest.String()+`  </manifest>
  <spine>
`+spine.String()+`  </spine>
</package>`, zip.Deflate)

	for _, ch := range chapters {
		// Corrupt chapters are STORED so the marker payload is literal bytes we
		// can locate and flip below.
		method := uint16(zip.Deflate)
		if ch.corrupt {
			method = zip.Store
		}
		add("OEBPS/"+ch.href, ch.content, method)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}

	raw := buf.Bytes()
	for _, ch := range chapters {
		if !ch.corrupt || ch.marker == "" {
			continue
		}
		mIdx := bytes.Index(raw, []byte(ch.marker))
		if mIdx < 0 {
			t.Fatalf("corrupt marker for %s not found in archive", ch.href)
		}
		// Flip a byte well inside the marker payload (data region), leaving the
		// central directory (at the end) untouched.
		raw[mIdx+len(ch.marker)/2] ^= 0xFF
	}

	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("write epub: %v", err)
	}
}

// ---------------------------------------------------------------------------
// D-LIST-RT — lists must round-trip through the reverse (MD->EPUB) path.
// ---------------------------------------------------------------------------

// TestReverseListParsing_MarkdownToHTML proves the named markdownToHTML renderer
// now emits <ul>/<ol>/<li> for markdown list syntax (it previously emitted none).
//
// Mutation proof (§1.1): removing the list branch makes the <ul>/<ol> assertions
// FAIL (list lines would be wrapped in <p> instead).
func TestReverseListParsing_MarkdownToHTML(t *testing.T) {
	c := NewMarkdownToEPUBConverter()

	t.Run("unordered", func(t *testing.T) {
		got := c.markdownToHTML("- alpha\n- beta\n- gamma\n")
		assertContainsAll(t, got, "<ul>", "</ul>",
			"<li>alpha</li>", "<li>beta</li>", "<li>gamma</li>")
		if strings.Contains(got, "<p>- alpha</p>") {
			t.Fatalf("unordered list item wrapped in <p> (not parsed): %q", got)
		}
	})

	t.Run("ordered", func(t *testing.T) {
		got := c.markdownToHTML("1. first\n2. second\n3. third\n")
		assertContainsAll(t, got, "<ol>", "</ol>",
			"<li>first</li>", "<li>second</li>", "<li>third</li>")
		if strings.Contains(got, "<p>1. first</p>") {
			t.Fatalf("ordered list item wrapped in <p> (not parsed): %q", got)
		}
	})

	t.Run("asterisk-unordered", func(t *testing.T) {
		got := c.markdownToHTML("* one\n* two\n")
		assertContainsAll(t, got, "<ul>", "<li>one</li>", "<li>two</li>")
	})

	t.Run("inline-markup-in-item", func(t *testing.T) {
		got := c.markdownToHTML("- a **bold** item\n")
		assertContainsAll(t, got, "<li>a <strong>bold</strong> item</li>")
	})

	t.Run("list-then-paragraph", func(t *testing.T) {
		got := c.markdownToHTML("- x\n- y\n\nAfter the list.\n")
		assertContainsAll(t, got, "<ul>", "<li>x</li>", "<li>y</li>", "</ul>",
			"<p>After the list.</p>")
	})

	t.Run("nested-unordered", func(t *testing.T) {
		got := c.markdownToHTML("- outer\n  - inner\n")
		// Both items present; nested <ul> rendered inside the outer item.
		assertContainsAll(t, got, "<li>outer", "<ul>", "<li>inner</li>")
	})
}

// TestReverseListParsing_XHTML proves the LIVE EPUB-chapter renderer
// (convertMarkdownToXHTML, used by createEPUB) also parses lists.
//
// Mutation proof (§1.1): removing the list branch makes the <ul>/<ol>
// assertions FAIL.
func TestReverseListParsing_XHTML(t *testing.T) {
	c := NewMarkdownToEPUBConverter()

	t.Run("unordered", func(t *testing.T) {
		got := c.convertMarkdownToXHTML("- alpha\n- beta\n")
		assertContainsAll(t, got, "<ul>", "<li>alpha</li>", "<li>beta</li>", "</ul>")
	})

	t.Run("ordered", func(t *testing.T) {
		got := c.convertMarkdownToXHTML("1. first\n2. second\n")
		assertContainsAll(t, got, "<ol>", "<li>first</li>", "<li>second</li>", "</ol>")
	})
}

// TestListFullRoundTrip is the end-to-end property guard: a markdown document
// containing both an unordered and an ordered list, converted to an EPUB on disk
// and back to markdown, MUST still contain both lists with their items.
//
// Mutation proof (§1.1): with no reverse-path list parsing, the intermediate
// EPUB chapters wrap list lines in <p>, so the round-tripped markdown loses the
// "- " / "1. " markers and this test FAILS.
func TestListFullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	srcMD := dir + "/src.md"
	epubPath := dir + "/out.epub"
	backMD := dir + "/back.md"

	const md = `---
title: List RT
language: en
---

# List RT

---

## Chapter One

Intro paragraph.

- apple
- banana
- cherry

Some prose between lists.

1. first
2. second
3. third

Closing paragraph.

---
`
	if err := writeFile(t, srcMD, md); err != nil {
		t.Fatalf("write src md: %v", err)
	}

	m2e := NewMarkdownToEPUBConverter()
	if err := m2e.ConvertMarkdownToEPUB(srcMD, epubPath); err != nil {
		t.Fatalf("ConvertMarkdownToEPUB: %v", err)
	}

	// Assert the produced EPUB chapter actually contains list elements.
	chapterXHTML := readEPUBEntryContaining(t, epubPath, "chapter1.xhtml")
	assertContainsAll(t, chapterXHTML, "<ul>", "<li>apple</li>", "<ol>", "<li>first</li>")

	// Now convert back to markdown and assert the lists survived.
	e2m := NewEPUBToMarkdownConverter(false, dir+"/imgs")
	if err := e2m.ConvertEPUBToMarkdown(epubPath, backMD); err != nil {
		t.Fatalf("ConvertEPUBToMarkdown: %v", err)
	}
	back := readFile(t, backMD)

	for _, want := range []string{"- apple", "- banana", "- cherry"} {
		if !strings.Contains(back, want) {
			t.Fatalf("unordered list item %q lost in round-trip; back markdown:\n%s", want, back)
		}
	}
	for _, want := range []string{"1. first", "2. second", "3. third"} {
		if !strings.Contains(back, want) {
			t.Fatalf("ordered list item %q lost in round-trip; back markdown:\n%s", want, back)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertContainsAll(t *testing.T, s string, subs ...string) {
	t.Helper()
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			t.Fatalf("missing %q in:\n%s", sub, s)
		}
	}
}

func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0644)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// readEPUBEntryContaining returns the content of the first zip entry whose name
// contains substr.
func readEPUBEntryContaining(t *testing.T, epubPath, substr string) string {
	t.Helper()
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if strings.Contains(f.Name, substr) {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open entry %s: %v", f.Name, err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("read entry %s: %v", f.Name, err)
			}
			return string(b)
		}
	}
	t.Fatalf("no zip entry containing %q in %s", substr, epubPath)
	return ""
}
