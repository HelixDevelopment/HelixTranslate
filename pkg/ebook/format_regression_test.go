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
