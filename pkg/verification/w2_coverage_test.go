package verification

import (
	"strings"
	"sync"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/language"
)

func ebookMeta(title, desc string) *ebook.Metadata {
	return &ebook.Metadata{Title: title, Description: desc}
}

// W2 anti-bluff coverage slice (§11.4.27/§11.4): exercises pure-logic units that
// previously had 0% coverage — LLM-response note parsing, prompt construction,
// note-collection summaries, report generators, and language-detection branches.
// Every assertion checks a concrete output value; if the unit-under-test were
// replaced by a stub (e.g. `return nil` / `return ""`), the assertion fails.
// Anything requiring a live LLM call is NOT exercised here (deferred to the
// integration tier) — parseNote/parseNotes/createNotePrompt take/produce plain
// strings and never touch nt.translator, so a nil translator is safe.

// ---------------------------------------------------------------------------
// NoteTaker.parseNote — single-note parsing from an LLM-style response block.
// ---------------------------------------------------------------------------

func TestParseNote_FullBlock(t *testing.T) {
	nt := NewNoteTaker(nil, "prov1")
	block := strings.Join([]string{
		"NOTE: [CHARACTER]",
		"IMPORTANCE: [critical]",
		"TITLE: Protagonist voice",
		"CONTENT: The hero speaks formally.",
		"continued content line.",
		"EXAMPLES:",
		"He said: good day.",
		"She replied: indeed.",
		"IMPLICATIONS: Keep the register formal.",
		"more implication text.",
	}, "\n")

	note := nt.parseNote(block, 2, "sec-1", "Chapter 1")
	if note == nil {
		t.Fatal("parseNote returned nil for a complete, valid block")
	}
	if note.NoteType != NoteTypeCharacter {
		t.Errorf("NoteType = %q, want %q", note.NoteType, NoteTypeCharacter)
	}
	if note.Importance != ImportanceCritical {
		t.Errorf("Importance = %q, want %q", note.Importance, ImportanceCritical)
	}
	if note.Title != "Protagonist voice" {
		t.Errorf("Title = %q, want %q", note.Title, "Protagonist voice")
	}
	// Content must include the continuation line joined with a space.
	if !strings.Contains(note.Content, "The hero speaks formally.") ||
		!strings.Contains(note.Content, "continued content line.") {
		t.Errorf("Content missing parts: %q", note.Content)
	}
	if len(note.Examples) != 2 {
		t.Fatalf("Examples count = %d, want 2: %#v", len(note.Examples), note.Examples)
	}
	if note.Examples[0] != "He said: good day." || note.Examples[1] != "She replied: indeed." {
		t.Errorf("Examples = %#v", note.Examples)
	}
	if !strings.Contains(note.Implications, "Keep the register formal.") ||
		!strings.Contains(note.Implications, "more implication text.") {
		t.Errorf("Implications missing parts: %q", note.Implications)
	}
	// Identity fields wired from constructor + args.
	if note.PassNumber != 2 || note.SectionID != "sec-1" || note.Location != "Chapter 1" || note.Provider != "prov1" {
		t.Errorf("identity fields wrong: pass=%d sec=%q loc=%q prov=%q",
			note.PassNumber, note.SectionID, note.Location, note.Provider)
	}
}

func TestParseNote_DefaultsAndRejection(t *testing.T) {
	nt := NewNoteTaker(nil, "p")
	tests := []struct {
		name      string
		block     string
		wantNil   bool
		wantImp   ImportanceLevel
		wantType  NoteType
		wantTitle string
	}{
		{
			name:    "missing title -> rejected",
			block:   "NOTE: [tone]\nCONTENT: dark mood",
			wantNil: true,
		},
		{
			name:    "missing content -> rejected",
			block:   "NOTE: [tone]\nTITLE: Mood",
			wantNil: true,
		},
		{
			name:    "missing note type -> rejected",
			block:   "TITLE: Mood\nCONTENT: dark",
			wantNil: true,
		},
		{
			// IMPLICATIONS is optional (D5 fixed); Content is retained with or
			// without it. This case keeps IMPLICATIONS to also exercise that path.
			name:      "no importance -> defaults to medium",
			block:     "NOTE: [theme]\nTITLE: Loss\nCONTENT: recurring loss motif\nIMPLICATIONS: preserve motif",
			wantNil:   false,
			wantImp:   ImportanceMedium,
			wantType:  NoteTypeTheme,
			wantTitle: "Loss",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := nt.parseNote(tt.block, 1, "s", "loc")
			if tt.wantNil {
				if note != nil {
					t.Fatalf("expected nil for invalid block, got %#v", note)
				}
				return
			}
			if note == nil {
				t.Fatal("expected a note, got nil")
			}
			if note.Importance != tt.wantImp {
				t.Errorf("Importance = %q, want %q", note.Importance, tt.wantImp)
			}
			if note.NoteType != tt.wantType {
				t.Errorf("NoteType = %q, want %q", note.NoteType, tt.wantType)
			}
			if note.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", note.Title, tt.wantTitle)
			}
		})
	}
}

// TestParseNote_ContentWithoutImplicationsRetained is the D5 regression guard.
// A well-formed note (NOTE+TITLE+CONTENT) that omits the OPTIONAL IMPLICATIONS:
// marker MUST be retained with its Content — the validator treats Content as
// required and Implications as optional, so a CONTENT-only note is valid.
// Before the fix, parseNote only finalized note.Content inside the IMPLICATIONS:
// branch, so this note was silently dropped (returned nil) — a real loss of
// literary notes whenever the LLM omitted IMPLICATIONS.
func TestParseNote_ContentWithoutImplicationsRetained(t *testing.T) {
	nt := &NoteTaker{provider: "test"}
	block := "NOTE: [theme]\nTITLE: Loss\nCONTENT: recurring loss motif"
	note := nt.parseNote(block, 1, "s", "loc")
	if note == nil {
		t.Fatal("D5: a NOTE+TITLE+CONTENT note without IMPLICATIONS was dropped (Content not finalized)")
	}
	if note.Content != "recurring loss motif" {
		t.Errorf("D5: Content = %q, want %q", note.Content, "recurring loss motif")
	}
	if note.Implications != "" {
		t.Errorf("D5: Implications = %q, want empty (none provided)", note.Implications)
	}
}

// ---------------------------------------------------------------------------
// NoteTaker.parseNotes — splits a multi-note response on "---" and parses each.
// ---------------------------------------------------------------------------

func TestParseNotes_MultipleAndEmptySections(t *testing.T) {
	nt := NewNoteTaker(nil, "p")
	resp := strings.Join([]string{
		"NOTE: [tone]\nTITLE: A\nCONTENT: alpha\nIMPLICATIONS: keep",
		"---",
		"   ", // empty section after trim -> skipped
		"---",
		"NOTE: [theme]\nTITLE: B\nCONTENT: beta\nIMPLICATIONS: keep",
		"---",
		"NOTE: [style]\nTITLE: incomplete", // no content -> rejected, not appended
	}, "\n")

	notes := nt.parseNotes(resp, 1, "sec", "loc")
	if len(notes) != 2 {
		t.Fatalf("parseNotes returned %d notes, want 2 (empty + invalid sections dropped): %#v", len(notes), notes)
	}
	if notes[0].Title != "A" || notes[1].Title != "B" {
		t.Errorf("titles = %q, %q; want A, B", notes[0].Title, notes[1].Title)
	}
}

func TestParseNotes_EmptyResponse(t *testing.T) {
	nt := NewNoteTaker(nil, "p")
	notes := nt.parseNotes("", 1, "sec", "loc")
	if len(notes) != 0 {
		t.Fatalf("empty response should yield 0 notes, got %d", len(notes))
	}
	// Must be non-nil empty slice per implementation contract.
	if notes == nil {
		t.Fatal("parseNotes must return a non-nil slice")
	}
}

// ---------------------------------------------------------------------------
// NoteTaker.createNotePrompt — prompt assembly with/without previous notes.
// ---------------------------------------------------------------------------

func TestCreateNotePrompt(t *testing.T) {
	nt := NewNoteTaker(nil, "p")

	withoutPrev := nt.createNotePrompt("ORIG-TEXT", "TRANS-TEXT", nil)
	if !strings.Contains(withoutPrev, "ORIG-TEXT") || !strings.Contains(withoutPrev, "TRANS-TEXT") {
		t.Error("prompt must embed original and translated text")
	}
	if !strings.Contains(withoutPrev, "Response Format") {
		t.Error("prompt must contain the response-format instructions")
	}
	if strings.Contains(withoutPrev, "Previous Analysis") {
		t.Error("prompt must NOT mention previous analysis when none supplied")
	}

	prev := []*LiteraryNote{
		{NoteType: NoteTypeTone, Title: "Mood", Content: "somber"},
	}
	withPrev := nt.createNotePrompt("O", "T", prev)
	if !strings.Contains(withPrev, "Previous Analysis") {
		t.Error("prompt must include previous-analysis section when notes supplied")
	}
	if !strings.Contains(withPrev, "Mood") || !strings.Contains(withPrev, "somber") {
		t.Error("prompt must render the previous note's title and content")
	}
}

// ---------------------------------------------------------------------------
// NoteCollection.Summary / GetByPass — aggregation logic.
// ---------------------------------------------------------------------------

func TestNoteCollection_SummaryAndGetByPass(t *testing.T) {
	nc := NewNoteCollection()
	nc.Add(&LiteraryNote{NoteType: NoteTypeTone, PassNumber: 1, SectionID: "s1", Importance: ImportanceCritical})
	nc.Add(&LiteraryNote{NoteType: NoteTypeTone, PassNumber: 1, SectionID: "s1", Importance: ImportanceLow})
	nc.Add(&LiteraryNote{NoteType: NoteTypeTheme, PassNumber: 2, SectionID: "s2", Importance: ImportanceCritical})

	pass1 := nc.GetByPass(1)
	if len(pass1) != 2 {
		t.Errorf("GetByPass(1) = %d notes, want 2", len(pass1))
	}
	if len(nc.GetByPass(99)) != 0 {
		t.Error("GetByPass for unknown pass must be empty")
	}

	summary := nc.Summary()
	if !strings.Contains(summary, "Total Notes: 3") {
		t.Errorf("summary total wrong: %q", summary)
	}
	if !strings.Contains(summary, "Critical Notes: 2") {
		t.Errorf("summary critical count wrong: %q", summary)
	}
	if !strings.Contains(summary, "tone: 2") {
		t.Errorf("summary by-type tone wrong: %q", summary)
	}
}

// ---------------------------------------------------------------------------
// TranslationNotes.Import — round-trips exported notes back into the store.
// ---------------------------------------------------------------------------

func TestTranslationNotes_Import(t *testing.T) {
	tn := NewTranslationNotes()
	// One with an explicit ID, one with empty ID (must be auto-assigned).
	data := []ExportedNote{
		{ID: "keep-1", Type: NoteTypeTone, Content: "c1"},
		{ID: "", Type: NoteTypeTheme, Content: "c2"},
	}
	if err := tn.Import(data); err != nil {
		t.Fatalf("Import error: %v", err)
	}

	got, ok := tn.GetNote("keep-1")
	if !ok || got.Content != "c1" || got.Type != NoteTypeTone {
		t.Errorf("imported note keep-1 wrong: ok=%v note=%#v", ok, got)
	}

	exported, err := tn.Export()
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}
	if len(exported) != 2 {
		t.Errorf("after import, Export count = %d, want 2", len(exported))
	}
	// The empty-ID note must have received an auto-generated id.
	for _, e := range exported {
		if e.ID == "" {
			t.Error("imported empty-ID note was not assigned an ID")
		}
	}
}

// ---------------------------------------------------------------------------
// PolishingReport.GenerateJSONReport / GenerateSummary — derived report output.
// ---------------------------------------------------------------------------

func buildFinalizedReport() *PolishingReport {
	cfg := PolishingConfig{
		Providers:        []string{"openai", "anthropic"},
		MinConsensus:     2,
		VerifySpirit:     true,
		VerifyLanguage:   true,
		VerifyContext:    true,
		VerifyVocabulary: true,
	}
	pr := NewPolishingReport(cfg)
	pr.AddSectionResult(&PolishingResult{
		Location:        "Chapter 1",
		SpiritScore:     0.9,
		LanguageScore:   0.8,
		ContextScore:    0.85,
		VocabularyScore: 0.95,
		Confidence:      0.9,
		Consensus:       2,
		Changes: []Change{
			{Location: "C1", Reason: "tone fix", Confidence: 0.9, Agreement: 2,
				Original: "old", Polished: "new"},
		},
		Issues: []Issue{
			{Type: "spirit", Severity: "critical", Description: "lost tone", Location: "C1"},
		},
	})
	pr.Finalize()
	return pr
}

func TestPolishingReport_GenerateJSONReport(t *testing.T) {
	pr := buildFinalizedReport()
	j := pr.GenerateJSONReport()

	summary, ok := j["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("JSON report missing summary map")
	}
	if summary["total_sections"].(int) != 1 {
		t.Errorf("total_sections = %v, want 1", summary["total_sections"])
	}
	if summary["total_changes"].(int) != 1 {
		t.Errorf("total_changes = %v, want 1", summary["total_changes"])
	}

	scores, ok := j["quality_scores"].(map[string]interface{})
	if !ok {
		t.Fatal("JSON report missing quality_scores map")
	}
	// Overall = mean of the four dimension scores.
	wantOverall := (0.9 + 0.8 + 0.85 + 0.95) / 4.0
	if got := scores["overall"].(float64); got < wantOverall-1e-9 || got > wantOverall+1e-9 {
		t.Errorf("overall score = %v, want %v", got, wantOverall)
	}

	cfgMap, ok := j["config"].(map[string]interface{})
	if !ok || cfgMap["min_consensus"].(int) != 2 {
		t.Errorf("config min_consensus wrong: %#v", cfgMap)
	}
}

func TestPolishingReport_GenerateSummary(t *testing.T) {
	pr := buildFinalizedReport()
	s := pr.GenerateSummary()

	if !strings.Contains(s, "POLISHING SUMMARY") {
		t.Errorf("summary header missing: %q", s)
	}
	if !strings.Contains(s, "Sections Verified: 1") {
		t.Errorf("summary sections wrong: %q", s)
	}
	if !strings.Contains(s, "Changes Made: 1") {
		t.Errorf("summary changes wrong: %q", s)
	}
	// Issue severity is title-cased in the breakdown.
	if !strings.Contains(s, "Critical: 1") {
		t.Errorf("summary issue breakdown missing Critical: %q", s)
	}
	// Overall grade for ~0.875 is "A-".
	if !strings.Contains(s, "A-") {
		t.Errorf("summary grade missing A-: %q", s)
	}
}

// ---------------------------------------------------------------------------
// getGrade — full threshold table.
// ---------------------------------------------------------------------------

func TestGetGrade_Thresholds(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{0.99, "A+"}, {0.95, "A+"},
		{0.92, "A"}, {0.90, "A"},
		{0.87, "A-"}, {0.85, "A-"},
		{0.82, "B+"}, {0.80, "B+"},
		{0.77, "B"}, {0.75, "B"},
		{0.72, "B-"}, {0.70, "B-"},
		{0.67, "C+"}, {0.65, "C+"},
		{0.62, "C"}, {0.60, "C"},
		{0.50, "D"}, {0.0, "D"},
	}
	for _, c := range cases {
		if got := getGrade(c.score); got != c.want {
			t.Errorf("getGrade(%.2f) = %q, want %q", c.score, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// truncateForDisplay — boundary behaviour.
// ---------------------------------------------------------------------------

func TestTruncateForDisplay(t *testing.T) {
	if got := truncateForDisplay("short", 10); got != "short" {
		t.Errorf("under-limit should be unchanged, got %q", got)
	}
	if got := truncateForDisplay("exactly10!", 10); got != "exactly10!" {
		t.Errorf("at-limit should be unchanged, got %q", got)
	}
	got := truncateForDisplay("abcdefghijkl", 5)
	if got != "abcde..." {
		t.Errorf("over-limit truncation = %q, want %q", got, "abcde...")
	}
}

// ---------------------------------------------------------------------------
// isValidForLanguage — script-validity per target language.
// ---------------------------------------------------------------------------

func TestIsValidForLanguage(t *testing.T) {
	cases := []struct {
		word string
		lang string
		want bool
	}{
		{"привет", "ru", true},   // Cyrillic word, Cyrillic target -> valid
		{"hello", "ru", false},   // Latin word, Cyrillic target -> invalid
		{"hello", "en", true},    // Latin word, Latin target -> valid
		{"привет", "en", false},  // Cyrillic word, Latin target -> invalid
		{"漢字", "zh", true},       // Han, Chinese target -> valid
		{"hello", "zh", true},    // Latin allowed (continue) for CJK target
		{"12345", "en", false},   // no letters -> hasLetters false
		{"こんにちは", "ja", true},    // Hiragana, Japanese -> valid
		{"안녕", "ko", true},       // Hangul, Korean -> valid
		{"привет", "zh", false},  // Cyrillic in CJK target -> invalid
	}
	for _, c := range cases {
		if got := isValidForLanguage(c.word, c.lang); got != c.want {
			t.Errorf("isValidForLanguage(%q, %q) = %v, want %v", c.word, c.lang, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Verifier.isSourceLanguage — script-mismatch detection branches.
// ---------------------------------------------------------------------------

func newTestVerifier(src, tgt string) *Verifier {
	return NewVerifier(
		language.Language{Code: src, Name: src},
		language.Language{Code: tgt, Name: tgt},
		nil, "w2-test",
	)
}

func TestIsSourceLanguage_Branches(t *testing.T) {
	t.Run("empty is never source", func(t *testing.T) {
		v := newTestVerifier("ru", "sr")
		if v.isSourceLanguage("") {
			t.Error("empty string must not be flagged as source language")
		}
	})

	t.Run("russian-only char ru->sr conclusive", func(t *testing.T) {
		v := newTestVerifier("ru", "sr")
		// Contains 'ы' which is Russian-only — conclusive even though short.
		if !v.isSourceLanguage("мы") {
			t.Error("Russian-only char must mark text as source for ru->sr")
		}
	})

	t.Run("too few letters -> not source", func(t *testing.T) {
		v := newTestVerifier("ru", "en")
		if v.isSourceLanguage("да") { // <10 letters, no ru-only char
			t.Error("text with <10 letters must not be flagged")
		}
	})

	t.Run("cyrillic source, latin target, cyrillic text -> source", func(t *testing.T) {
		v := newTestVerifier("ru", "en")
		// >=10 Cyrillic letters, target Latin -> untranslated.
		if !v.isSourceLanguage("приветствие миру") {
			t.Error("long Cyrillic text for ru->en target should be flagged as source")
		}
	})

	t.Run("cyrillic source, latin target, latin text -> not source", func(t *testing.T) {
		v := newTestVerifier("ru", "en")
		if v.isSourceLanguage("hello there world") {
			t.Error("translated Latin text must not be flagged")
		}
	})

	t.Run("latin source, cyrillic target, latin text -> source", func(t *testing.T) {
		v := newTestVerifier("en", "ru")
		if !v.isSourceLanguage("the quick brown fox jumps") {
			t.Error("untranslated Latin text for en->ru should be flagged as source")
		}
	})
}

// ---------------------------------------------------------------------------
// Verifier.verifyMetadata — untranslated title/description detection.
// ---------------------------------------------------------------------------

func TestVerifyMetadata_Branches(t *testing.T) {
	v := newTestVerifier("ru", "en")

	t.Run("untranslated title produces error block", func(t *testing.T) {
		res := &VerificationResult{}
		// Long Cyrillic title -> flagged.
		err := v.verifyMetadata(ebookMeta("приветствие большому миру здесь", ""), res)
		if err != nil {
			t.Fatalf("verifyMetadata error: %v", err)
		}
		if len(res.UntranslatedBlocks) != 1 || res.UntranslatedBlocks[0].Location != "Book Title" {
			t.Errorf("expected 1 title block, got %#v", res.UntranslatedBlocks)
		}
		if len(res.Errors) != 1 {
			t.Errorf("expected 1 error, got %#v", res.Errors)
		}
	})

	t.Run("untranslated description produces warning block", func(t *testing.T) {
		res := &VerificationResult{}
		err := v.verifyMetadata(ebookMeta("", "приветствие большому миру здесь снова"), res)
		if err != nil {
			t.Fatalf("verifyMetadata error: %v", err)
		}
		if len(res.UntranslatedBlocks) != 1 || res.UntranslatedBlocks[0].Location != "Book Description" {
			t.Errorf("expected 1 description block, got %#v", res.UntranslatedBlocks)
		}
		if len(res.Warnings) != 1 {
			t.Errorf("expected 1 warning, got %#v", res.Warnings)
		}
	})

	t.Run("translated metadata produces no blocks", func(t *testing.T) {
		res := &VerificationResult{}
		err := v.verifyMetadata(ebookMeta("a translated english title here", "fully translated description text"), res)
		if err != nil {
			t.Fatalf("verifyMetadata error: %v", err)
		}
		if len(res.UntranslatedBlocks) != 0 {
			t.Errorf("translated metadata should yield no blocks, got %#v", res.UntranslatedBlocks)
		}
	})
}

// ---------------------------------------------------------------------------
// Concurrency: TranslationNotes is mutex-guarded; exercise it under -race.
// ---------------------------------------------------------------------------

func TestTranslationNotes_ConcurrentAccess(t *testing.T) {
	tn := NewTranslationNotes()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := tn.AddNote(NoteTypeTone, "concurrent content", nil)
			if err != nil {
				t.Errorf("AddNote error: %v", err)
				return
			}
			_, _ = tn.GetNote(id)
			_ = tn.GetStatistics()
			_ = tn.FilterNotes(NoteFilter{})
		}()
	}
	wg.Wait()

	stats := tn.GetStatistics()
	if stats.Total != 20 {
		t.Errorf("after 20 concurrent adds, Total = %d, want 20", stats.Total)
	}
}
