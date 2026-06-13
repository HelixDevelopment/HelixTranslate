package markdown

import (
	"strings"
	"testing"
)

// TestConvertOrderedList_Numbering is the permanent regression guard for the
// <ol> numbering data-loss defect: ordered-list items must render with a
// monotonic "1. ", "2. ", ... counter (not the "- " dash used for <ul>), so the
// user's ordering survives the EPUB->Markdown conversion.
//
// Mutation proof (§1.1): reverting convertListItems to always write "- "
// regardless of the `ordered` flag makes the "1. " / "2. " assertions FAIL.
func TestConvertOrderedList_Numbering(t *testing.T) {
	t.Run("simple-ordered", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ol><li>first</li><li>second</li><li>third</li></ol>")
		for i, want := range []string{"1. first", "2. second", "3. third"} {
			if !strings.Contains(got, want) {
				t.Fatalf("item %d: got %q, want substring %q", i+1, got, want)
			}
		}
		if strings.Contains(got, "- first") {
			t.Fatalf("ordered list must not emit dash markers, got %q", got)
		}
	})

	t.Run("unordered-still-dashes", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ul><li>alpha</li><li>beta</li></ul>")
		if !strings.Contains(got, "- alpha") || !strings.Contains(got, "- beta") {
			t.Fatalf("unordered list lost dash markers, got %q", got)
		}
		if strings.Contains(got, "1. alpha") {
			t.Fatalf("unordered list must not be numbered, got %q", got)
		}
	})

	t.Run("ordered-counter-resets-per-list", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ol><li>a</li><li>b</li></ol><ol><li>c</li><li>d</li></ol>")
		// Each <ol> restarts at 1.
		if strings.Count(got, "1. ") != 2 {
			t.Fatalf("expected two independent '1. ' starts, got %q", got)
		}
		if !strings.Contains(got, "1. c") || !strings.Contains(got, "2. d") {
			t.Fatalf("second ol did not restart numbering, got %q", got)
		}
	})

	t.Run("single-item-ordered", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ol><li>only</li></ol>")
		if !strings.Contains(got, "1. only") {
			t.Fatalf("single ordered item = %q, want '1. only'", got)
		}
	})

	t.Run("ordered-with-inline-formatting", func(t *testing.T) {
		// Numbering must survive and inline markup must convert. (Note: the space
		// between </strong> and the following text node is collapsed by
		// convertNode's per-text-node TrimSpace — a separate, pre-existing
		// inline-whitespace defect NOT in scope for the ordered-list fix; this
		// guard asserts only the numbering + markup that the fix owns.)
		got := htmlToMarkdown(t, "<ol><li>plain</li><li><strong>bold</strong> tail</li></ol>")
		if !strings.Contains(got, "1. plain") {
			t.Fatalf("ordered list lost first-item numbering: %q", got)
		}
		if !strings.Contains(got, "2. **bold**") {
			t.Fatalf("ordered list lost numbering/markup on item 2: %q", got)
		}
	})

	t.Run("nested-unordered-inside-ordered", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ol><li>outer<ul><li>inner</li></ul></li></ol>")
		if !strings.Contains(got, "1. outer") {
			t.Fatalf("ordered outer item lost numbering, got %q", got)
		}
		if !strings.Contains(got, "  - inner") {
			t.Fatalf("nested unordered item lost indented dash, got %q", got)
		}
	})

	t.Run("nested-ordered-inside-ordered", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ol><li>outer<ol><li>sub-one</li><li>sub-two</li></ol></li></ol>")
		if !strings.Contains(got, "1. outer") {
			t.Fatalf("outer ordered item lost numbering, got %q", got)
		}
		if !strings.Contains(got, "  1. sub-one") || !strings.Contains(got, "  2. sub-two") {
			t.Fatalf("nested ordered list lost indented numbering, got %q", got)
		}
	})

	t.Run("empty-ordered-list", func(t *testing.T) {
		got := htmlToMarkdown(t, "<ol></ol>")
		if strings.Contains(got, "1.") {
			t.Fatalf("empty ordered list should emit no items, got %q", got)
		}
	})

	t.Run("orphan-li-still-no-marker", func(t *testing.T) {
		got := htmlToMarkdown(t, "<li>orphan</li>")
		if strings.Contains(got, "- ") || strings.Contains(got, "1. ") {
			t.Fatalf("orphan <li> must not emit any marker, got %q", got)
		}
	})
}
