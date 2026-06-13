package verification

import (
	"sync"
	"testing"
	"unicode/utf8"
)

// --- Defect 1: byte-boundary truncation corrupts multibyte UTF-8 (Cyrillic) ---
//
// truncate() and truncateForDisplay() slice by byte index. For this
// translation system the source/original text is routinely Cyrillic
// (Russian/Serbian/Bulgarian/Ukrainian), where each letter is a 2-byte
// UTF-8 sequence. Slicing at an odd byte boundary splits a rune and emits
// the U+FFFD replacement byte sequence — invalid/corrupt UTF-8 lands in
// the verification report and in UntranslatedBlock.OriginalText.
//
// RED: with maxLen chosen to split a Cyrillic rune, the pre-fix code
// returns a string that is NOT valid UTF-8.

func TestTruncate_PreservesValidUTF8_Cyrillic(t *testing.T) {
	// "Привет" — 6 Cyrillic letters, 12 bytes (2 bytes each).
	in := "Привет"
	if len(in) != 12 {
		t.Fatalf("precondition: expected 12 bytes, got %d", len(in))
	}
	// maxLen=5 lands in the middle of the 3rd rune (bytes 4-5) -> byte slice
	// text[:5] cuts a rune in half.
	out := truncate(in, 5)
	if !utf8.ValidString(out) {
		t.Fatalf("truncate produced invalid UTF-8 from Cyrillic input: %q (bytes=%v)", out, []byte(out))
	}
}

func TestTruncateForDisplay_PreservesValidUTF8_Cyrillic(t *testing.T) {
	in := "Привет" // 12 bytes
	out := truncateForDisplay(in, 5)
	if !utf8.ValidString(out) {
		t.Fatalf("truncateForDisplay produced invalid UTF-8 from Cyrillic input: %q (bytes=%v)", out, []byte(out))
	}
}

// --- Defect 2: TranslationNotes.Import has no mutex lock (data race) ---
//
// Every other mutator (AddNote/UpdateNote/DeleteNote) takes tn.mu.Lock();
// Import mutates tn.notes and tn.nextID with no lock at all. Concurrent
// Import + AddNote is a data race on the map and the counter.
//
// RED: run with `-race`; the pre-fix code reports a DATA RACE.

func TestImport_ConcurrentWithAddNote_NoRace(t *testing.T) {
	tn := NewTranslationNotes()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = tn.Import([]ExportedNote{
				{Type: NoteTypeCharacter, Content: "imported content"},
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = tn.AddNote(NoteTypeTone, "added content", nil)
		}()
	}
	wg.Wait()
}

// --- EXTEND: truncate case-space (boundaries / scripts / no-op paths) ---

func TestTruncate_CaseSpace(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		maxLen int
		// when truncation actually happens (len(in) > maxLen), result must be
		// valid UTF-8 and must NOT exceed maxLen content bytes (+ "...").
		truncates bool
	}{
		{"empty", "", 0, false},
		{"empty_nonzero_max", "", 10, false},
		{"ascii_under", "hello", 100, false},
		{"ascii_exact", "hello", 5, false},
		{"ascii_over", "hello world", 5, true},
		{"cyrillic_split_at_odd", "Привет мир", 5, true},  // 5 splits rune 3
		{"cyrillic_split_at_7", "Привет мир", 7, true},    // 7 splits rune 4
		{"cyrillic_on_boundary", "Привет", 4, true},       // 4 is a clean boundary (2 runes)
		{"emoji_4byte_split", "a😀b", 2, true},            // 😀 is 4 bytes starting at byte 1
		{"mixed_latin_cyrillic", "abПривет", 3, true},     // 3 splits first Cyrillic rune
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := truncate(c.in, c.maxLen)
			if !utf8.ValidString(out) {
				t.Fatalf("truncate(%q, %d) = %q is not valid UTF-8 (bytes=%v)", c.in, c.maxLen, out, []byte(out))
			}
			if !c.truncates {
				if out != c.in {
					t.Fatalf("truncate(%q, %d): expected no-op, got %q", c.in, c.maxLen, out)
				}
				return
			}
			// content portion must be <= maxLen bytes
			content := out[:len(out)-len("...")]
			if len(content) > c.maxLen {
				t.Fatalf("truncate(%q, %d): content %q exceeds maxLen (%d bytes)", c.in, c.maxLen, content, len(content))
			}
			// content must be a valid prefix of the input (no corruption/reorder)
			if !hasPrefix(c.in, content) {
				t.Fatalf("truncate(%q, %d): content %q is not a prefix of input", c.in, c.maxLen, content)
			}
		})
	}
}

func TestTruncateForDisplay_CaseSpace(t *testing.T) {
	// truncateForDisplay shares the same rune-safety contract.
	for _, in := range []string{"", "ascii text here", "Привет мир, как дела", "a😀b日本語"} {
		for _, maxLen := range []int{0, 1, 2, 3, 5, 7, 11, 1000} {
			out := truncateForDisplay(in, maxLen)
			if !utf8.ValidString(out) {
				t.Fatalf("truncateForDisplay(%q, %d) = %q is not valid UTF-8", in, maxLen, out)
			}
		}
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// --- EXTEND: Import concurrency case-space (Import||Import, Import||read) ---

func TestImport_ConcurrentWithReadsAndImports_NoRace(t *testing.T) {
	tn := NewTranslationNotes()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = tn.Import([]ExportedNote{{Type: NoteTypeCharacter, Content: "c1"}})
		}()
		go func() {
			defer wg.Done()
			_ = tn.Import([]ExportedNote{{ID: "fixed_id", Type: NoteTypeTone, Content: "c2"}})
		}()
		go func() {
			defer wg.Done()
			_ = tn.GetStatistics()
		}()
	}
	wg.Wait()
}

// --- EXTEND: Import still functionally correct after the lock fix ---

func TestImport_AssignsIDsAndStoresNotes(t *testing.T) {
	tn := NewTranslationNotes()
	err := tn.Import([]ExportedNote{
		{ID: "explicit_1", Type: NoteTypeCharacter, Content: "a"},
		{Type: NoteTypeTone, Content: "b"}, // empty ID -> generated
	})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if got, ok := tn.GetNote("explicit_1"); !ok || got.Content != "a" {
		t.Fatalf("explicit-ID note not stored correctly: ok=%v note=%+v", ok, got)
	}
	stats := tn.GetStatistics()
	if stats.Total != 2 {
		t.Fatalf("expected 2 notes after Import, got %d", stats.Total)
	}
}
