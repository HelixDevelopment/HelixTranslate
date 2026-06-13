package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/format"
)

// ---------------------------------------------------------------------------
// Output-path derivation (generateOutputFilename / generateOriginalMDPath /
// generateTranslatedMDPath). These compute the user-visible file names the CLI
// writes to. A stub returning "" or a constant would fail every case below.
// ---------------------------------------------------------------------------

func TestGenerateOutputFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"fb2 in subdir", "/books/war.fb2", "/books/war_sr.epub"},
		{"lowercase ext stripped", "/books/story.epub", "/books/story_sr.epub"},
		// KNOWN QUIRK (reported as finding): the ext is lowercased before
		// TrimSuffix, so an UPPERCASE extension is NOT stripped. We pin the
		// actual behavior so a future fix flips this case deliberately.
		{"uppercase ext not stripped (quirk)", "/books/Story.EPUB", "/books/Story.EPUB_sr.epub"},
		{"relative path", "book.txt", "book_sr.epub"},
		{"no extension", "/data/plain", "/data/plain_sr.epub"},
		{"dotted basename", "/d/my.book.v2.fb2", "/d/my.book.v2_sr.epub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateOutputFilename(tt.input); got != tt.want {
				t.Fatalf("generateOutputFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateOriginalMDPath(t *testing.T) {
	if got := generateOriginalMDPath("/books/war.fb2"); got != "/books/war_original.md" {
		t.Fatalf("got %q, want /books/war_original.md", got)
	}
	if got := generateOriginalMDPath("novel.epub"); got != "novel_original.md" {
		t.Fatalf("got %q, want novel_original.md", got)
	}
}

func TestGenerateTranslatedMDPath(t *testing.T) {
	if got := generateTranslatedMDPath("/x/y/book.txt"); got != "/x/y/book_translated.md" {
		t.Fatalf("got %q, want /x/y/book_translated.md", got)
	}
}

// The three path helpers must produce DISTINCT files for the same input so the
// pipeline does not overwrite original with translated, etc.
func TestPathHelpersDistinct(t *testing.T) {
	in := "/d/book.fb2"
	out := generateOutputFilename(in)
	orig := generateOriginalMDPath(in)
	tr := generateTranslatedMDPath(in)
	if out == orig || out == tr || orig == tr {
		t.Fatalf("path helpers collided: out=%q orig=%q tr=%q", out, orig, tr)
	}
	for _, p := range []string{out, orig, tr} {
		if filepath.Dir(p) != "/d" {
			t.Fatalf("helper changed directory: %q", p)
		}
	}
}

// ---------------------------------------------------------------------------
// generateSessionID — must be unique-ish and carry the tx- prefix the report
// and SSH temp-file naming rely on.
// ---------------------------------------------------------------------------

func TestGenerateSessionID(t *testing.T) {
	id := generateSessionID()
	if !strings.HasPrefix(id, "tx-") {
		t.Fatalf("session ID missing tx- prefix: %q", id)
	}
	// Two calls separated in time must differ (UnixNano-based).
	id1 := generateSessionID()
	time.Sleep(2 * time.Nanosecond)
	id2 := generateSessionID()
	if id1 == id2 {
		t.Fatalf("session IDs not unique: %q == %q", id1, id2)
	}
}

// ---------------------------------------------------------------------------
// verifyTranslation — the gate deciding whether translated output is "verified"
// in the session report. Real behavior: Serbian-Cyrillic target requires an
// actual Serbian-Cyrillic codepoint present; other targets require non-blank.
// ---------------------------------------------------------------------------

func TestVerifyTranslation(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		targetLang string
		script     string
		want       bool
	}{
		{"sr cyrillic with serbian char", "Љубав", "sr", "cyrillic", true},
		{"sr cyrillic plain ascii fails", "Hello world", "sr", "cyrillic", false},
		{"sr cyrillic empty fails", "", "sr", "cyrillic", false},
		{"sr cyrillic non-serbian cyrillic-only fails", "Ы", "sr", "cyrillic", false},
		{"non-sr target non-blank passes", "anything", "en", "latin", true},
		{"non-sr target blank fails", "   \n\t ", "en", "latin", false},
		{"sr latin (not cyrillic branch) non-blank passes", "Ljubav", "sr", "latin", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verifyTranslation(tt.text, tt.targetLang, tt.script); got != tt.want {
				t.Fatalf("verifyTranslation(%q,%q,%q) = %v, want %v",
					tt.text, tt.targetLang, tt.script, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// bookToString — flattens the parsed Book into the text fed to markdown
// conversion. Must include chapter titles AND section content, in order.
// ---------------------------------------------------------------------------

func TestBookToString(t *testing.T) {
	book := &ebook.Book{
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter One",
				Sections: []ebook.Section{
					{Content: "First section body."},
					{Content: "Second section body."},
				},
			},
			{
				Title:    "Chapter Two",
				Sections: []ebook.Section{{Content: "Lone section."}},
			},
		},
	}
	got := bookToString(book)

	for _, must := range []string{"Chapter One", "First section body.", "Second section body.", "Chapter Two", "Lone section."} {
		if !strings.Contains(got, must) {
			t.Fatalf("bookToString output missing %q\nfull output:\n%s", must, got)
		}
	}
	// Ordering: Chapter One must precede Chapter Two; section content must follow
	// its chapter title.
	if strings.Index(got, "Chapter One") > strings.Index(got, "Chapter Two") {
		t.Fatal("chapters out of order")
	}
	if strings.Index(got, "Chapter One") > strings.Index(got, "First section body.") {
		t.Fatal("section content placed before its chapter title")
	}
}

func TestBookToStringEmpty(t *testing.T) {
	if got := bookToString(&ebook.Book{}); got != "" {
		t.Fatalf("empty book should yield empty string, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// convertToMarkdown default branch — unknown/text formats pass through verbatim.
// (fb2/epub branches require real parser files → integration tier.)
// ---------------------------------------------------------------------------

func TestConvertToMarkdownPassthrough(t *testing.T) {
	const content = "Plain text body\nwith two lines."
	for _, fmtName := range []string{"txt", "html", "pdf", "docx", "unknown"} {
		out, err := convertToMarkdown(content, fmtName)
		if err != nil {
			t.Fatalf("convertToMarkdown(%q) unexpected error: %v", fmtName, err)
		}
		if out != content {
			t.Fatalf("convertToMarkdown(%q) altered passthrough content: got %q", fmtName, out)
		}
	}
}

// ---------------------------------------------------------------------------
// Session-step lifecycle: addStep / stepComplete / stepError.
// ---------------------------------------------------------------------------

func newTestSession() *TranslationSession {
	return &TranslationSession{
		Files: make([]GeneratedFile, 0),
		Steps: make([]*TranslationStep, 0),
	}
}

func TestAddStepAppendsAndReturnsLive(t *testing.T) {
	s := newTestSession()
	step := addStep(s, "Parsing")
	if len(s.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(s.Steps))
	}
	if step.Name != "Parsing" || step.Success {
		t.Fatalf("new step wrong defaults: %+v", *step)
	}
	// Returned pointer must alias the slot in the session (live mutation).
	step.Details = "mutated"
	if s.Steps[0].Details != "mutated" {
		t.Fatal("addStep did not return a pointer aliasing the session slice slot")
	}
}

func TestStepComplete(t *testing.T) {
	s := newTestSession()
	step := addStep(s, "X")
	stepComplete(step)
	if !s.Steps[0].Success {
		t.Fatal("stepComplete did not mark success")
	}
	if s.Steps[0].EndTime.IsZero() {
		t.Fatal("stepComplete did not set EndTime")
	}
}

func TestStepError(t *testing.T) {
	s := newTestSession()
	step := addStep(s, "Y")
	err := stepError(step, "boom")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("stepError returned %v, want error 'boom'", err)
	}
	if s.Steps[0].Success {
		t.Fatal("stepError must not mark success")
	}
	if s.Steps[0].Error != "boom" {
		t.Fatalf("stepError did not record message, got %q", s.Steps[0].Error)
	}
	if s.Steps[0].EndTime.IsZero() {
		t.Fatal("stepError did not set EndTime")
	}
}

// TestMultipleStepsRetainIndependentState is a regression guard for the
// slice-realloc pointer-aliasing fix: addStep returns a *TranslationStep that
// must stay valid across a SECOND addStep (which can reallocate the backing
// array). Completing step A AFTER step B was appended must still mutate A.
// Reverting Steps to []TranslationStep makes this FAIL (stale pointer).
func TestMultipleStepsRetainIndependentState(t *testing.T) {
	s := newTestSession()
	a := addStep(s, "A")
	b := addStep(s, "B")
	stepComplete(a) // mutate the FIRST step after the second was appended
	stepError(b, "b-failed")
	if !s.Steps[0].Success {
		t.Fatal("step A should be success")
	}
	if s.Steps[1].Success {
		t.Fatal("step B should be failed")
	}
	if s.Steps[1].Error != "b-failed" {
		t.Fatalf("step B error wrong: %q", s.Steps[1].Error)
	}
}

// ---------------------------------------------------------------------------
// addFile — records generated files for the report.
// ---------------------------------------------------------------------------

func TestAddFile(t *testing.T) {
	s := newTestSession()
	addFile(s, "/out/book.epub", "epub", 123, true, "ok")
	addFile(s, "/out/book.md", "translated_md", 45, false, "needs review")
	if len(s.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(s.Files))
	}
	f0 := s.Files[0]
	if f0.Path != "/out/book.epub" || f0.Type != "epub" || f0.Size != 123 || !f0.Verified || f0.Verification != "ok" {
		t.Fatalf("file 0 fields wrong: %+v", f0)
	}
	if s.Files[1].Verified {
		t.Fatal("file 1 should be unverified")
	}
}

// ---------------------------------------------------------------------------
// initVerifierConfig — assembles the verifier.Config from UnifiedConfig flags.
// ---------------------------------------------------------------------------

func TestInitVerifierConfig(t *testing.T) {
	uc := &UnifiedConfig{
		VerifierURL:    "http://verifier.local:9000",
		VerifierAPIKey: "vk-123",
	}
	vc := initVerifierConfig(uc)
	if vc.APIURL != "http://verifier.local:9000" {
		t.Fatalf("APIURL = %q", vc.APIURL)
	}
	if vc.APIKey != "vk-123" {
		t.Fatalf("APIKey = %q", vc.APIKey)
	}
	if vc.CacheTTL != time.Hour {
		t.Fatalf("CacheTTL = %v, want 1h", vc.CacheTTL)
	}
	if vc.MinScoreThreshold != 0.0 {
		t.Fatalf("MinScoreThreshold = %v, want 0", vc.MinScoreThreshold)
	}
}

// ---------------------------------------------------------------------------
// getFileSize / verifyEPUB — filesystem helpers, exercised with real temp files
// (deterministic, no network).
// ---------------------------------------------------------------------------

func TestGetFileSize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.bin")
	payload := []byte("0123456789") // 10 bytes
	if err := os.WriteFile(p, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := getFileSize(p); got != int64(len(payload)) {
		t.Fatalf("getFileSize = %d, want %d", got, len(payload))
	}
	if got := getFileSize(filepath.Join(dir, "nope")); got != 0 {
		t.Fatalf("missing file size = %d, want 0", got)
	}
}

func TestVerifyEPUB(t *testing.T) {
	dir := t.TempDir()

	// Valid: starts with PK and contains the epub mimetype marker.
	valid := filepath.Join(dir, "ok.epub")
	if err := os.WriteFile(valid, []byte("PKmimetypeapplication/epub+zip rest"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !verifyEPUB(valid) {
		t.Fatal("verifyEPUB should accept a PK + epub-mimetype file")
	}

	// Has marker but wrong magic (not PK) → reject.
	noMagic := filepath.Join(dir, "nomagic.epub")
	if err := os.WriteFile(noMagic, []byte("XXapplication/epub+zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if verifyEPUB(noMagic) {
		t.Fatal("verifyEPUB must reject a file without the PK magic")
	}

	// PK magic but no epub mimetype marker → reject.
	noMarker := filepath.Join(dir, "nomarker.epub")
	if err := os.WriteFile(noMarker, []byte("PK plain zip without marker"), 0o644); err != nil {
		t.Fatal(err)
	}
	if verifyEPUB(noMarker) {
		t.Fatal("verifyEPUB must reject a zip lacking the epub mimetype marker")
	}

	// Missing file → reject.
	if verifyEPUB(filepath.Join(dir, "absent.epub")) {
		t.Fatal("verifyEPUB must reject a missing file")
	}
}

// Cross-check that format.Format stringification used by parseInputFile is sane
// (guards the "format" string fed into convertToMarkdown's switch).
func TestFormatStringsKnown(t *testing.T) {
	// FB2 and EPUB string forms must match the cases convertToMarkdown switches on.
	if format.FormatFB2.String() != "fb2" {
		t.Fatalf("FB2 string = %q, want fb2", format.FormatFB2.String())
	}
	if format.FormatEPUB.String() != "epub" {
		t.Fatalf("EPUB string = %q, want epub", format.FormatEPUB.String())
	}
}
