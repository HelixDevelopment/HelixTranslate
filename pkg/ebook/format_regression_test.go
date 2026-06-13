package ebook

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/fb2"
)

// TestRegression_FB2WriterSplitsParagraphs is the permanent guard for the FB2
// writer paragraph-split defect: content separated by blank lines must produce
// separate, well-formed <p> elements — never a single <p> with literal escaped
// "&lt;/p&gt;&lt;p&gt;" markup inside it.
//
// Mutation guard: restore the old escapeFB2Text "</p><p>" string injection and
// this test FAILs (escaped markup appears + round-trip yields 1 mangled para).
func TestRegression_FB2WriterSplitsParagraphs(t *testing.T) {
	w := NewFB2Writer()
	book := &Book{
		Metadata: Metadata{Title: "T", Language: "ru"},
		Chapters: []Chapter{{
			Title:    "C",
			Sections: []Section{{Content: "Первый абзац.\n\nВторой абзац.\n\nТретий абзац."}},
		}},
	}
	out := filepath.Join(t.TempDir(), "out.fb2")
	if err := w.Write(book, out); err != nil {
		t.Fatalf("Write: %v", err)
	}

	raw := readFile(t, out)
	if strings.Contains(raw, "&lt;/p&gt;") || strings.Contains(raw, "&lt;p&gt;") {
		t.Errorf("output contains literal escaped paragraph markup (broken split):\n%s", raw)
	}

	fb, err := fb2.NewParser().Parse(out)
	if err != nil {
		t.Fatalf("re-parse written FB2: %v", err)
	}
	paras := fb.Body[0].Section[0].Paragraph
	want := []string{"Первый абзац.", "Второй абзац.", "Третий абзац."}
	if len(paras) != len(want) {
		t.Fatalf("round-trip paragraph count = %d, want %d", len(paras), len(want))
	}
	for i, wantText := range want {
		if got := paras[i].FullParagraphText(); got != wantText {
			t.Errorf("paragraph[%d] = %q, want %q", i, got, wantText)
		}
	}
}

// TestRegression_EPUBSharedIdentifier is the permanent guard for the EPUB
// OPF/NCX identifier-mismatch defect: the NCX dtb:uid MUST equal the OPF
// package unique-identifier (EPUB 2 conformance).
//
// Mutation guard: revert writeTOC to generateUUID() for dtb:uid and this FAILs.
func TestRegression_EPUBSharedIdentifier(t *testing.T) {
	w := NewEPUBWriter()

	t.Run("generated uuid shared", func(t *testing.T) {
		book := &Book{Metadata: Metadata{Title: "T", Language: "en"}, Chapters: []Chapter{{Title: "C", Sections: []Section{{Content: "x"}}}}}
		assertEPUBIdentifiersMatch(t, w, book)
	})
	t.Run("isbn identifier shared", func(t *testing.T) {
		book := &Book{Metadata: Metadata{Title: "T", Language: "en", ISBN: "urn:isbn:9781234567897"}, Chapters: []Chapter{{Title: "C", Sections: []Section{{Content: "x"}}}}}
		assertEPUBIdentifiersMatch(t, w, book)
	})
}

func assertEPUBIdentifiersMatch(t *testing.T, w *EPUBWriter, book *Book) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "b.epub")
	if err := w.Write(book, out); err != nil {
		t.Fatalf("Write: %v", err)
	}
	opf := readZipEntry(t, out, "OEBPS/content.opf")
	ncx := readZipEntry(t, out, "OEBPS/toc.ncx")

	opfID := regexp.MustCompile(`<dc:identifier[^>]*>([^<]+)</dc:identifier>`).FindStringSubmatch(opf)
	ncxUID := regexp.MustCompile(`name="dtb:uid"\s+content="([^"]+)"`).FindStringSubmatch(ncx)
	if opfID == nil {
		t.Fatalf("no dc:identifier in OPF:\n%s", opf)
	}
	if ncxUID == nil {
		t.Fatalf("no dtb:uid in NCX:\n%s", ncx)
	}
	if opfID[1] != ncxUID[1] {
		t.Errorf("EPUB conformance: NCX dtb:uid %q != OPF dc:identifier %q", ncxUID[1], opfID[1])
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func readZipEntry(t *testing.T, path, name string) string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip %s: %v", path, err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open entry %s: %v", name, err)
			}
			defer rc.Close()
			b, _ := io.ReadAll(rc)
			return string(b)
		}
	}
	t.Fatalf("entry %s not found in %s", name, path)
	return ""
}

// TestRegression_EPUBCleanXMLDataNoCorruption is the permanent guard for the
// CleanXMLData entity-corruption defect: the fallback XML cleaner must escape
// genuinely-bare ampersands WITHOUT corrupting already-valid entities or text.
//
// The previous implementation blindly ran ReplaceAll("&a","&amp;") (and "&l",
// "&g", "&q"), which rewrote the "&a" inside an existing "&amp;" → "&amp;mp;"
// and mangled ordinary text like "Q&A" / "AT&T".
//
// Mutation guard: restore any blind 2-char ReplaceAll and a "valid entity
// untouched" case FAILs.
func TestRegression_EPUBCleanXMLDataNoCorruption(t *testing.T) {
	p := NewEPUBParser()
	cases := []struct{ name, in, want string }{
		{"valid amp untouched", `<dc:title>Tom &amp; Jerry</dc:title>`, `<dc:title>Tom &amp; Jerry</dc:title>`},
		{"valid entities untouched", `&lt;a&gt; &quot;x&quot; &apos;y&apos;`, `&lt;a&gt; &quot;x&quot; &apos;y&apos;`},
		{"numeric entity untouched", `caf&#233; &#x41;`, `caf&#233; &#x41;`},
		{"bare amp escaped once", `Q&A and rock & roll`, `Q&amp;A and rock &amp; roll`},
		{"AT&T escaped once", `AT&T`, `AT&amp;T`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(p.CleanXMLData([]byte(tc.in)))
			if got != tc.want {
				t.Errorf("CleanXMLData(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRegression_EPUBRemoveHTMLTagsSelfClosing is the permanent guard for the
// removeHTMLTags defect: self-closing/void tags (<br/>, <img .../>, <hr/>),
// HTML comments and doctype declarations must all be stripped from extracted
// chapter text — never leak through as literal markup.
//
// The previous opening-tag regex `<[a-zA-Z][^>/]*>` excluded the '/' character,
// so it never matched <br/> etc., leaving "Line1<br/>Line2" intact.
//
// Mutation guard: restore the opening-only/closing-only regex pair and the
// self-closing cases FAIL.
func TestRegression_EPUBRemoveHTMLTagsSelfClosing(t *testing.T) {
	cases := []struct {
		name, in       string
		mustNotContain []string
		mustContain    []string
	}{
		{"br void tag", "Line1<br/>Line2", []string{"<br", "/>"}, []string{"Line1", "Line2"}},
		{"img void tag", `Para<img src="x.png"/>two`, []string{"<img", "/>", "x.png"}, []string{"Para", "two"}},
		{"hr void tag", "A<hr/>B", []string{"<hr", "/>"}, []string{"A", "B"}},
		{"html comment", "X<!-- secret -->Y", []string{"<!--", "secret", "-->"}, []string{"X", "Y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := removeHTMLTags(tc.in)
			for _, bad := range tc.mustNotContain {
				if strings.Contains(got, bad) {
					t.Errorf("removeHTMLTags(%q) leaked %q: got %q", tc.in, bad, got)
				}
			}
			for _, good := range tc.mustContain {
				if !strings.Contains(got, good) {
					t.Errorf("removeHTMLTags(%q) dropped %q: got %q", tc.in, good, got)
				}
			}
		})
	}
}
