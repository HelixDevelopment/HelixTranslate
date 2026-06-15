package grpc

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
)

// readEPUBText unzips an EPUB and returns the concatenated text of every XHTML
// chapter file (tags stripped). This is the end-user-visible content: what a
// reader app would actually render.
func readEPUBText(t *testing.T, epubPath string) string {
	t.Helper()
	zr, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("output is not a valid EPUB (zip open failed): %v", err)
	}
	defer zr.Close()

	var sb strings.Builder
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if !strings.HasSuffix(name, ".xhtml") && !strings.HasSuffix(name, ".html") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("failed to open %s inside EPUB: %v", f.Name, err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		// crude tag strip
		raw := string(data)
		for {
			lt := strings.IndexByte(raw, '<')
			if lt < 0 {
				sb.WriteString(raw)
				break
			}
			sb.WriteString(raw[:lt])
			gt := strings.IndexByte(raw[lt:], '>')
			if gt < 0 {
				break
			}
			raw = raw[lt+gt+1:]
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildRealEPUB writes a genuine EPUB (zip) fixture to disk and returns its path.
func buildRealEPUB(t *testing.T, dir string) string {
	t.Helper()
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:    "Sample Source Book",
			Authors:  []string{"Original Author"},
			Language: "ru",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "First Chapter",
				Sections: []ebook.Section{
					{
						Title:   "Opening",
						Content: "ZEBRAPHRASE one unique source sentence.\n\nSecond paragraph here.",
					},
				},
			},
		},
		Format: format.FormatEPUB,
	}
	epubPath := filepath.Join(dir, "input.epub")
	if err := ebook.NewEPUBWriter().Write(book, epubPath); err != nil {
		t.Fatalf("failed to write input EPUB fixture: %v", err)
	}
	return epubPath
}

// TestCoreTranslator_EPUBInput_ProducesTranslatedContent drives the REAL gRPC
// core translation pipeline (offline "mock" provider, no network) with a REAL
// EPUB *input* file and asserts the job COMPLETES and the output EPUB genuinely
// contains translated text.
//
// REPRODUCE-FIRST (§11.4.115): on the pre-fix code, convertToMarkdown writes the
// plain-text ExtractText() output into a .epub temp file and re-parses it as a
// zip — which always fails — so an EPUB-input job errors out and never produces
// translated content. This test FAILs (RED) on that code.
func TestCoreTranslator_EPUBInput_ProducesTranslatedContent(t *testing.T) {
	dir := t.TempDir()
	inputEPUB := buildRealEPUB(t, dir)
	outputEPUB := filepath.Join(dir, "output_translated.epub")

	// sanity: the fixture really is a valid EPUB the parser accepts
	if _, err := ebook.NewUniversalParser().Parse(inputEPUB); err != nil {
		t.Fatalf("precondition: fixture EPUB is not parseable: %v", err)
	}

	log := logger.NewLogger(logger.LoggerConfig{Level: "error"})
	ct := NewCoreTranslator(log)

	req := &proto.TranslationRequest{
		SessionId:  "epub-input-test",
		InputFile:  inputEPUB,
		OutputFile: outputEPUB,
		SourceLang: "ru",
		TargetLang: "en",
		Script:     "latin",
		ProviderConfig: &proto.ProviderConfig{
			Type:           "mock",
			Model:          "mock",
			TimeoutSeconds: 30,
		},
		Options: &proto.TranslationOptions{},
	}

	resp, err := ct.Translate(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("EPUB-input translation job FAILED (bug): %v", err)
	}
	if resp == nil || resp.Status != "completed" {
		gotStatus := "<nil>"
		if resp != nil {
			gotStatus = resp.Status
		}
		t.Fatalf("expected job status 'completed', got %q", gotStatus)
	}

	// The output file must exist and be a real EPUB.
	if _, err := os.Stat(outputEPUB); err != nil {
		t.Fatalf("output EPUB not created: %v", err)
	}

	text := readEPUBText(t, outputEPUB)

	// The mock provider transforms "X" -> "translated: X". Genuine translated
	// content MUST be present in the output a reader would see.
	if !strings.Contains(text, "translated:") {
		t.Fatalf("output EPUB does not contain translated content (mock transform marker 'translated:' missing).\nOutput text:\n%s", text)
	}
	// The unique source phrase must have been carried through translation, not lost.
	if !strings.Contains(text, "ZEBRAPHRASE") {
		t.Fatalf("output EPUB lost the source content (unique phrase 'ZEBRAPHRASE' missing).\nOutput text:\n%s", text)
	}
}
