//go:build integration
// +build integration

// Package integration contains end-to-end pipeline tests that exercise the REAL
// translation pipeline on REAL artifacts: a real input file is written to
// disk, parsed by the real pkg/format detector + pkg/ebook parsers, translated
// by the real pkg/translator.UniversalTranslator glue, and written by the real
// pkg/ebook output writers. Only the LLM provider is deterministic
// (a mock Translator or an httptest OpenAI-compatible server returning a known
// Serbian-Cyrillic string) — no real API keys, no real network (§11.4.27).
//
// Anti-bluff (§11.4 / §11.4.2 / §11.4.83): every PASS RE-OPENS the produced
// output artifact and asserts it contains the exact deterministic translated
// text. A pipeline that drops, empties, or fails to translate the content makes
// the test FAIL. The TestPipeline_DropsContent_DetectedByAssertion test proves
// the assertion is not a bluff: when translation is stubbed to drop content,
// the same artifact-content assertion FAILS.
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"archive/zip"
	"io"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// ---------------------------------------------------------------------------
// Deterministic test translator
// ---------------------------------------------------------------------------

// cyrillicMarker is embedded in every translated string so the artifact-content
// assertions are exact and unmistakably Cyrillic (proves real translated text
// in the target language reached the output file, per §11.4 anti-bluff).
const cyrillicMarker = "ПРЕВЕДЕНО"

// deterministicTranslator is a real implementation of translator.Translator
// (NOT a testify mock) that maps any non-empty source text to a deterministic
// Serbian-Cyrillic string. It lets the test drive the REAL UniversalTranslator
// book-walking glue while keeping the output exactly predictable.
type deterministicTranslator struct {
	dropContent bool // when true, returns "" — simulates a content-dropping bug
	calls       int
}

func (d *deterministicTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	d.calls++
	if d.dropContent {
		return "", nil
	}
	if strings.TrimSpace(text) == "" {
		return text, nil
	}
	// Deterministic Cyrillic translation that embeds the marker. The original
	// source is intentionally NOT echoed so the assertion proves the TRANSLATED
	// text (not the source) landed in the artifact.
	return cyrillicMarker + " текст", nil
}

func (d *deterministicTranslator) TranslateWithProgress(
	ctx context.Context, text, context string, _ *events.EventBus, _ string,
) (string, error) {
	return d.Translate(ctx, text, context)
}

func (d *deterministicTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{Total: d.calls, Translated: d.calls}
}

func (d *deterministicTranslator) GetName() string { return "deterministic-e2e" }

// ---------------------------------------------------------------------------
// Input fixtures (real on-disk files)
// ---------------------------------------------------------------------------

const sampleFB2 = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Test Book</book-title>
      <lang>ru</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>First Chapter</p></title>
      <p>This is the first paragraph in Russian.</p>
      <p>This is the second paragraph with more text.</p>
    </section>
  </body>
</FictionBook>`

const sampleTXT = "This is a plain text paragraph.\n\nThis is a second plain text paragraph that needs translating."

// writeFixture writes content to a temp file with the given name and returns the path.
func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", name, err)
	}
	return p
}

// runPipeline runs the REAL parse -> UniversalTranslator.TranslateBook -> write
// chain and returns the produced output artifact path.
func runPipeline(t *testing.T, inputPath, outputPath string, outFormat format.Format, trans translator.Translator) {
	t.Helper()

	// Step 1+2: real format detection + parsing.
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(inputPath)
	if err != nil {
		t.Fatalf("parse failed for %s: %v", inputPath, err)
	}
	if len(book.Chapters) == 0 {
		t.Fatalf("parsed book has no chapters: %s", inputPath)
	}

	// Step 3: REAL translation glue (walks metadata + every chapter/section).
	// Source language is set explicitly (Russian) so no LLM-backed language
	// detection is needed; a nil-LLMDetector detector is passed safely.
	universal := translator.NewUniversalTranslator(
		trans,
		language.NewDetector(nil),
		language.Russian,
		language.Serbian,
	)
	eventBus := events.NewEventBus()
	if err := universal.TranslateBook(context.Background(), book, eventBus, "e2e-session"); err != nil {
		t.Fatalf("TranslateBook failed: %v", err)
	}

	// Step 4: REAL output write.
	switch outFormat {
	case format.FormatEPUB:
		if err := ebook.NewEPUBWriter().Write(book, outputPath); err != nil {
			t.Fatalf("EPUB write failed: %v", err)
		}
	case format.FormatFB2:
		if err := ebook.NewFB2Writer().Write(book, outputPath); err != nil {
			t.Fatalf("FB2 write failed: %v", err)
		}
	default:
		t.Fatalf("unsupported output format in test: %s", outFormat)
	}
}

// ---------------------------------------------------------------------------
// Artifact readers (open the REAL produced file and extract its text)
// ---------------------------------------------------------------------------

// readEPUBChapters opens the produced EPUB zip and concatenates every
// OEBPS/chapterN.xhtml body — the bytes a reader app would render.
func readEPUBChapters(t *testing.T, epubPath string) string {
	t.Helper()
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("output EPUB is not a valid zip: %v", err)
	}
	defer r.Close()

	var sb strings.Builder
	chapterFiles := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "OEBPS/chapter") && strings.HasSuffix(f.Name, ".xhtml") {
			chapterFiles++
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open %s in EPUB: %v", f.Name, err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("failed to read %s in EPUB: %v", f.Name, err)
			}
			sb.Write(data)
		}
	}
	if chapterFiles == 0 {
		t.Fatalf("produced EPUB %s contains no OEBPS/chapter*.xhtml files", epubPath)
	}
	return sb.String()
}

// readFB2Text reads the produced FB2 file as text.
func readFB2Text(t *testing.T, fb2Path string) string {
	t.Helper()
	data, err := os.ReadFile(fb2Path)
	if err != nil {
		t.Fatalf("failed to read produced FB2: %v", err)
	}
	if !strings.Contains(string(data), "<FictionBook") {
		t.Fatalf("produced FB2 %s is not a valid FictionBook document", fb2Path)
	}
	return string(data)
}

// ---------------------------------------------------------------------------
// Tests: 2 input formats x 2 output formats, mock-translator backend
// ---------------------------------------------------------------------------

func TestPipeline_FB2_to_EPUB(t *testing.T) {
	dir := t.TempDir()
	in := writeFixture(t, dir, "book.fb2", sampleFB2)
	out := filepath.Join(dir, "out.epub")

	trans := &deterministicTranslator{}
	runPipeline(t, in, out, format.FormatEPUB, trans)

	if trans.calls == 0 {
		t.Fatalf("translator was never called — pipeline did not translate")
	}
	content := readEPUBChapters(t, out)
	if !strings.Contains(content, cyrillicMarker) {
		t.Fatalf("produced EPUB does not contain translated Cyrillic text %q.\nGot: %s", cyrillicMarker, content)
	}
}

func TestPipeline_FB2_to_FB2(t *testing.T) {
	dir := t.TempDir()
	in := writeFixture(t, dir, "book.fb2", sampleFB2)
	out := filepath.Join(dir, "out.fb2")

	trans := &deterministicTranslator{}
	runPipeline(t, in, out, format.FormatFB2, trans)

	if trans.calls == 0 {
		t.Fatalf("translator was never called — pipeline did not translate")
	}
	content := readFB2Text(t, out)
	if !strings.Contains(content, cyrillicMarker) {
		t.Fatalf("produced FB2 does not contain translated Cyrillic text %q.\nGot: %s", cyrillicMarker, content)
	}
}

func TestPipeline_TXT_to_EPUB(t *testing.T) {
	dir := t.TempDir()
	in := writeFixture(t, dir, "book.txt", sampleTXT)

	// Confirm the real detector classifies the input as TXT — proves the
	// detection seam is exercised, not bypassed.
	if f, err := format.NewDetector().DetectFile(in); err != nil || f != format.FormatTXT {
		t.Fatalf("expected TXT detection, got format=%s err=%v", f, err)
	}

	out := filepath.Join(dir, "out.epub")
	trans := &deterministicTranslator{}
	runPipeline(t, in, out, format.FormatEPUB, trans)

	if trans.calls == 0 {
		t.Fatalf("translator was never called — pipeline did not translate")
	}
	content := readEPUBChapters(t, out)
	if !strings.Contains(content, cyrillicMarker) {
		t.Fatalf("produced EPUB (from TXT) does not contain translated Cyrillic text %q.\nGot: %s", cyrillicMarker, content)
	}
}

func TestPipeline_TXT_to_FB2(t *testing.T) {
	dir := t.TempDir()
	in := writeFixture(t, dir, "book.txt", sampleTXT)
	out := filepath.Join(dir, "out.fb2")

	trans := &deterministicTranslator{}
	runPipeline(t, in, out, format.FormatFB2, trans)

	if trans.calls == 0 {
		t.Fatalf("translator was never called — pipeline did not translate")
	}
	content := readFB2Text(t, out)
	if !strings.Contains(content, cyrillicMarker) {
		t.Fatalf("produced FB2 (from TXT) does not contain translated Cyrillic text %q.\nGot: %s", cyrillicMarker, content)
	}
}

// ---------------------------------------------------------------------------
// Test: REAL LLM provider path against an httptest OpenAI-compatible server.
// Proves the production llm.NewLLMTranslator -> OpenAI client -> /chat/completions
// HTTP call path actually fires (§11.4 anti-bluff: real API call, not a mock
// object), with the network endpoint pointed at httptest (no real network).
// ---------------------------------------------------------------------------

func TestPipeline_FB2_to_EPUB_viaHTTPTestProvider(t *testing.T) {
	const httpCyrillic = "СЕРВЕР-ПРЕВОД"

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		hits++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": httpCyrillic}},
			},
		})
	}))
	defer srv.Close()

	cfg := translator.TranslationConfig{
		Provider:    "openai",
		Model:       "gpt-4o",
		APIKey:      "test-key-not-real",
		BaseURL:     srv.URL + "/v1",
		Temperature: 0.0,
		MaxTokens:   256,
		Timeout:     10 * time.Second,
	}
	trans, err := llm.NewLLMTranslator(cfg)
	if err != nil {
		t.Fatalf("failed to construct real LLM translator: %v", err)
	}

	dir := t.TempDir()
	in := writeFixture(t, dir, "book.fb2", sampleFB2)
	out := filepath.Join(dir, "out.epub")
	runPipeline(t, in, out, format.FormatEPUB, trans)

	if hits == 0 {
		t.Fatalf("httptest OpenAI server was never called — real provider HTTP path not exercised")
	}
	content := readEPUBChapters(t, out)
	if !strings.Contains(content, httpCyrillic) {
		t.Fatalf("produced EPUB does not contain server-returned translated text %q.\nGot: %s", httpCyrillic, content)
	}
}

// ---------------------------------------------------------------------------
// Anti-bluff mutation guard: when translation DROPS content, the artifact-content
// assertion MUST fail. This proves the assertions above genuinely depend on the
// translated text reaching the output file (they are not vacuous).
// ---------------------------------------------------------------------------

func TestPipeline_DropsContent_DetectedByAssertion(t *testing.T) {
	dir := t.TempDir()
	in := writeFixture(t, dir, "book.fb2", sampleFB2)
	out := filepath.Join(dir, "out.epub")

	// dropContent=true simulates a content-dropping translation bug.
	trans := &deterministicTranslator{dropContent: true}
	runPipeline(t, in, out, format.FormatEPUB, trans)

	content := readEPUBChapters(t, out)
	// The marker MUST be absent — and crucially the source text MUST also be
	// preserved-or-empty, never silently replaced by translated content. We
	// assert the bluff condition: no Cyrillic marker present.
	if strings.Contains(content, cyrillicMarker) {
		t.Fatalf("anti-bluff guard failed: dropped-content run still produced the Cyrillic marker")
	}

	// Now prove the POSITIVE assertion used by the other tests would FAIL here:
	// a sub-assertion that the marker IS present must not hold.
	markerPresent := strings.Contains(content, cyrillicMarker)
	if markerPresent {
		t.Fatalf("expected marker absent when content is dropped, but it was present")
	}
}
