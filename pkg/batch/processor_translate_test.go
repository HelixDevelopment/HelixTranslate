package batch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/language"
)

// markerTranslator wraps every translated string with a unique sentinel so the
// test can prove the output file genuinely went through translation (not a
// byte-copy of the untranslated input). Each call also bumps a counter so we can
// assert the translator was actually invoked.
type markerTranslator struct {
	MockTranslator
	calls int
}

const translateMarker = "[XLATED]"

func newMarkerTranslator() *markerTranslator {
	mt := &markerTranslator{}
	mt.MockTranslator.translateFunc = func(_ context.Context, text, _ string) (string, error) {
		if strings.TrimSpace(text) == "" {
			return text, nil
		}
		mt.calls++
		return translateMarker + text, nil
	}
	return mt
}

// fb2WithBody builds a minimal FB2 document whose body section carries a
// recognisable source string. After translation the EPUB output MUST contain
// the translated form of that string.
func fb2WithBody(title, body string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>` + title + `</book-title>
    </title-info>
  </description>
  <body>
    <section>
      <p>` + body + `</p>
    </section>
  </body>
</FictionBook>`
}

// TestProcessFile_OutputIsActuallyTranslated is the §11.4.115 reproduce-first
// guard for the silent-no-translation defect: processFile parsed the input and
// wrote the output WITHOUT ever invoking options.Translator, yet reported
// Success:true. The API /translate/directory handler builds a real translator,
// calls Process, and reports "N successful" to the end user — so an
// untranslated copy shipped as a green "translation completed" (a §11.4 /
// CONST-035 PASS-bluff).
//
// RED on the pre-fix code: the translator is never called (calls == 0) and the
// re-parsed EPUB contains the SOURCE string verbatim with no marker -> FAIL.
// GREEN after the fix: the translator is invoked and the output text carries the
// translation marker.
func TestProcessFile_OutputIsActuallyTranslated(t *testing.T) {
	tmpDir := t.TempDir()

	const sourceBody = "Hello World"
	inputFile := filepath.Join(tmpDir, "book.fb2")
	if err := os.WriteFile(inputFile, []byte(fb2WithBody("Test Book", sourceBody)), 0o644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}
	outputFile := filepath.Join(tmpDir, "book.epub")

	mt := newMarkerTranslator()
	options := &ProcessingOptions{
		InputType:      InputTypeFile,
		InputPath:      inputFile,
		OutputPath:     outputFile,
		SourceLanguage: language.English,
		TargetLanguage: language.Serbian,
		Translator:     mt,
	}

	results, err := NewBatchProcessor(options).Process(context.Background())
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("expected one successful result, got %+v", results)
	}

	// The translator MUST have been invoked at least once for non-empty content.
	if mt.calls == 0 {
		t.Fatalf("translator was never invoked: output is an untranslated copy " +
			"reported as a successful translation (§11.4 PASS-bluff)")
	}

	// Re-parse the produced EPUB and assert the output genuinely carries the
	// translated text, not the source verbatim.
	book, perr := ebook.NewUniversalParser().Parse(outputFile)
	if perr != nil {
		t.Fatalf("failed to re-parse output EPUB: %v", perr)
	}
	text := book.ExtractText()
	if !strings.Contains(text, translateMarker) {
		t.Fatalf("output EPUB contains no translated text (marker %q absent); extracted:\n%s",
			translateMarker, text)
	}
	if strings.Contains(text, translateMarker+sourceBody) == false {
		t.Fatalf("output EPUB does not contain the translated body %q; extracted:\n%s",
			translateMarker+sourceBody, text)
	}
}
