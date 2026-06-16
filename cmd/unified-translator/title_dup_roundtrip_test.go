package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
)

// TestBookToString_TitleAppearsOnce is the §11.4.115 round-trip guard for
// MINOR-W6-1 at the cmd flatten layer: parse an HTML doc whose <title> and <h1>
// are the SAME chapter title, then flatten via bookToString. The title must
// appear EXACTLY ONCE in the flattened translatable text (once from
// chapter.Title; ZERO leaked copies from Section.Content after the parser fix).
//
// Pre-fix (RED_MODE=1): the title appears >1 time (parser leaked it into Content,
// bookToString adds another from chapter.Title).
// Post-fix (RED_MODE=0, default standing guard): exactly 1.
//
// Root cause: docs/qa/minor_w6_1_rootcause_20260616-151123/FINDING.md.
func TestBookToString_TitleAppearsOnce(t *testing.T) {
	const title = "La Farola"
	const body = "The lighthouse guided the ships."
	red := os.Getenv("RED_MODE") == "1"

	htmlDoc := `<!DOCTYPE html><html><head><title>` + title + `</title></head>` +
		`<body><h1>` + title + `</h1><p>` + body + `</p></body></html>`

	path := filepath.Join(t.TempDir(), "title_dup.html")
	if err := os.WriteFile(path, []byte(htmlDoc), 0o600); err != nil {
		t.Fatal(err)
	}

	book, err := ebook.NewHTMLParser().Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	got := bookToString(book)
	n := strings.Count(got, title)

	if !strings.Contains(got, body) {
		t.Fatalf("body text lost from flattened output:\n%s", got)
	}

	if red {
		if n <= 1 {
			t.Fatalf("PRE-FIX expected the title to appear >1 time, got %d:\n%s", n, got)
		}
		return
	}
	if n != 1 {
		t.Fatalf("title appears %d time(s) in bookToString output (want 1):\n%s", n, got)
	}
}
