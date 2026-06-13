package batch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
)

// errReader is an io.Reader that always fails, to exercise the stdin read-error path.
type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

// validFB2 is a minimal parseable FictionBook used to drive the success path of processFile.
const validFB2 = `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title>T</book-title></title-info></description>
  <body><section><p>Hello World</p></section></body>
</FictionBook>`

// Process: the default branch for an unrecognised InputType must error, not panic or silently
// succeed. Anti-bluff: if the switch grew a silent default returning nil,nil this would FAIL.
func TestProcess_UnsupportedInputType(t *testing.T) {
	options := &ProcessingOptions{
		InputType:  InputType(999),
		Translator: &MockTranslator{},
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil {
		t.Fatal("expected error for unsupported input type, got nil")
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
	if !strings.Contains(err.Error(), "unsupported input type") {
		t.Errorf("expected 'unsupported input type' in error, got %q", err.Error())
	}
}

// Process (InputTypeFile) error path: a non-existent input file must surface a failed
// ProcessingResult AND a non-nil error. Asserts both the wrapper-error and the result row,
// which the file branch in Process builds explicitly.
func TestProcess_FileInputError(t *testing.T) {
	options := &ProcessingOptions{
		InputType:      InputTypeFile,
		InputPath:      filepath.Join(t.TempDir(), "does-not-exist.fb2"),
		OutputPath:     filepath.Join(t.TempDir(), "out.epub"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result row, got %d", len(results))
	}
	if results[0].Success {
		t.Error("expected Success=false for missing input file")
	}
	if results[0].Error == nil {
		t.Error("expected result.Error to be set for missing input file")
	}
	if results[0].InputPath != options.InputPath {
		t.Errorf("expected result InputPath %q, got %q", options.InputPath, results[0].InputPath)
	}
}

// processString translator-error path: when the translator fails, the returned result row must
// be a failure carrying the translator error and Process must return that error.
func TestProcessString_TranslatorError(t *testing.T) {
	wantErr := errors.New("boom-translate")
	options := &ProcessingOptions{
		InputType:   InputTypeString,
		InputString: "hello",
		Translator: &MockTranslator{translateFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", wantErr
		}},
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected translator error %v, got %v", wantErr, err)
	}
	if len(results) != 1 || results[0].Success {
		t.Fatalf("expected 1 failed result, got %+v", results)
	}
	if !errors.Is(results[0].Error, wantErr) {
		t.Errorf("expected result.Error to wrap %v, got %v", wantErr, results[0].Error)
	}
}

// processString output-write-error path: an unwritable output path (directory-as-file) makes
// os.WriteFile fail; Process must return that error and a failed result.
func TestProcessString_OutputWriteError(t *testing.T) {
	dir := t.TempDir() // a directory cannot be opened as a regular file for writing
	options := &ProcessingOptions{
		InputType:   InputTypeString,
		InputString: "hello",
		OutputPath:  dir,
		Translator:  &MockTranslator{},
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil {
		t.Fatal("expected write error when output path is a directory, got nil")
	}
	if len(results) != 1 || results[0].Success {
		t.Fatalf("expected 1 failed result, got %+v", results)
	}
	if results[0].Error == nil {
		t.Error("expected result.Error to be set on write failure")
	}
}

// processStdin reader-error path: a failing reader must produce a failed <stdin> result + error.
func TestProcessStdin_ReadError(t *testing.T) {
	options := &ProcessingOptions{
		InputType:   InputTypeStdin,
		InputReader: errReader{},
		Translator:  &MockTranslator{},
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
	if len(results) != 1 || results[0].Success {
		t.Fatalf("expected 1 failed result, got %+v", results)
	}
	if results[0].InputPath != "<stdin>" {
		t.Errorf("expected InputPath '<stdin>', got %q", results[0].InputPath)
	}
}

// processStdin translator-error path.
func TestProcessStdin_TranslatorError(t *testing.T) {
	wantErr := errors.New("boom-stdin")
	options := &ProcessingOptions{
		InputType:   InputTypeStdin,
		InputReader: strings.NewReader("data"),
		Translator: &MockTranslator{translateFunc: func(_ context.Context, _, _ string) (string, error) {
			return "", wantErr
		}},
	}
	_, err := NewBatchProcessor(options).Process(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected translator error %v, got %v", wantErr, err)
	}
}

// processStdin output-write-error path: directory-as-output makes WriteFile fail after a
// successful translate.
func TestProcessStdin_OutputWriteError(t *testing.T) {
	dir := t.TempDir()
	options := &ProcessingOptions{
		InputType:   InputTypeStdin,
		InputReader: strings.NewReader("data"),
		OutputPath:  dir,
		Translator:  &MockTranslator{},
	}
	_, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil {
		t.Fatal("expected write error when stdin output path is a directory, got nil")
	}
}

// processDirectory: empty input path.
func TestProcessDirectory_EmptyPath(t *testing.T) {
	options := &ProcessingOptions{InputType: InputTypeDirectory, InputPath: "", Translator: &MockTranslator{}}
	_, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil || !strings.Contains(err.Error(), "input directory path is empty") {
		t.Fatalf("expected empty-path error, got %v", err)
	}
}

// processDirectory: non-existent directory.
func TestProcessDirectory_NonexistentDir(t *testing.T) {
	options := &ProcessingOptions{
		InputType:  InputTypeDirectory,
		InputPath:  filepath.Join(t.TempDir(), "nope"),
		Translator: &MockTranslator{},
	}
	_, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil || !strings.Contains(err.Error(), "failed to access directory") {
		t.Fatalf("expected access error, got %v", err)
	}
}

// processDirectory: input path is a file, not a directory.
func TestProcessDirectory_NotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "afile.fb2")
	if err := os.WriteFile(f, []byte(validFB2), 0o644); err != nil {
		t.Fatal(err)
	}
	options := &ProcessingOptions{InputType: InputTypeDirectory, InputPath: f, Translator: &MockTranslator{}}
	_, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

// processDirectory: directory exists but holds no supported files.
func TestProcessDirectory_NoSupportedFiles(t *testing.T) {
	dir := t.TempDir() // empty dir => zero supported files
	options := &ProcessingOptions{InputType: InputTypeDirectory, InputPath: dir, Translator: &MockTranslator{}}
	_, err := NewBatchProcessor(options).Process(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no supported files found") {
		t.Fatalf("expected no-supported-files error, got %v", err)
	}
}

// processDirectory emits an EventTranslationStarted event with total_files; assert the event bus
// actually receives it (anti-bluff: a removed Publish call makes this FAIL).
func TestProcessDirectory_EmitsStartedEvent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.fb2"), []byte(validFB2), 0o644); err != nil {
		t.Fatal(err)
	}
	bus := events.NewEventBus()

	// Publish dispatches handlers asynchronously, so signal via a buffered channel and wait.
	gotEvent := make(chan events.Event, 4)
	bus.Subscribe(events.EventTranslationStarted, func(e events.Event) {
		gotEvent <- e
	})

	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      dir,
		OutputPath:     filepath.Join(t.TempDir(), "out"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		EventBus:       bus,
		SessionID:      "sess-1",
	}
	if _, err := NewBatchProcessor(options).Process(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case e := <-gotEvent:
		if e.SessionID != "sess-1" {
			t.Errorf("expected SessionID 'sess-1', got %q", e.SessionID)
		}
		if e.Data["total_files"] != 1 {
			t.Errorf("expected total_files=1 in event data, got %v", e.Data["total_files"])
		}
	case <-time.After(2 * time.Second):
		t.Error("expected EventTranslationStarted event, none received within 2s")
	}
}

// processFilesSequential: a file whose parse fails yields a failed result row but the batch
// continues (Process returns nil error for directory mode). Drive via a directory containing one
// unparseable file. A garbage .txt parses as TXT format then fails at the ebook parser.
func TestProcessFilesSequential_PerFileError(t *testing.T) {
	dir := t.TempDir()
	// Two files: one valid fb2 (succeeds), one fb2-named file with invalid XML (parse fails).
	if err := os.WriteFile(filepath.Join(dir, "good.fb2"), []byte(validFB2), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.fb2"), []byte("<FictionBook xmlns=\"http://www.gribuser.ru/xml/fictionbook/2.0\"><body><unclosed></FictionBook"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      dir,
		OutputPath:     filepath.Join(t.TempDir(), "out"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       false,
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err != nil {
		t.Fatalf("directory processing should not return top-level error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 result rows, got %d", len(results))
	}
	var successes, failures int
	for _, r := range results {
		if r.Success {
			successes++
		} else {
			failures++
			if r.Error == nil {
				t.Error("expected failed result to carry a non-nil Error")
			}
		}
	}
	if successes != 1 || failures != 1 {
		t.Errorf("expected 1 success + 1 failure, got %d/%d", successes, failures)
	}
}

// processFilesParallel: same mixed batch must produce correct per-index results with no data race.
// Run with -race to exercise concurrency safety of the results slice + semaphore.
func TestProcessFilesParallel_PerFileErrorAndRace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.fb2"), []byte(validFB2), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.fb2"), []byte("not xml at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      dir,
		OutputPath:     filepath.Join(t.TempDir(), "out"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       true,
		MaxConcurrency: 4,
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Every slot must be populated (no zero-value gaps) — proves each goroutine wrote its index.
	for i, r := range results {
		if r.InputPath == "" {
			t.Errorf("result[%d] has empty InputPath — a goroutine failed to write its slot", i)
		}
	}
}

// processFilesParallel default-concurrency branch: MaxConcurrency<=0 must fall back to a default
// worker count and still process every file.
func TestProcessFilesParallel_DefaultConcurrency(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.fb2", "b.fb2", "c.fb2"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(validFB2), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      dir,
		OutputPath:     filepath.Join(t.TempDir(), "out"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       true,
		MaxConcurrency: 0, // forces default
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("expected success for %s, got %v", r.InputPath, r.Error)
		}
	}
}

// computeOutputPath: explicit output path that is an existing regular file must be returned
// verbatim (the !isOutputDir branch), not have a language suffix appended.
func TestComputeOutputPath_ExplicitFile(t *testing.T) {
	tmp := t.TempDir()
	outFile := filepath.Join(tmp, "explicit.epub")
	if err := os.WriteFile(outFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := &ProcessingOptions{
		InputPath:      tmp,
		OutputPath:     outFile,
		TargetLanguage: language.Serbian,
		OutputFormat:   "epub",
	}
	got, err := NewBatchProcessor(options).computeOutputPath(filepath.Join(tmp, "in.fb2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != outFile {
		t.Errorf("expected explicit output file %q returned verbatim, got %q", outFile, got)
	}
}

// computeOutputPath: empty OutputFormat defaults to epub in the no-output-dir branch.
func TestComputeOutputPath_DefaultFormat(t *testing.T) {
	options := &ProcessingOptions{
		OutputPath:     "",
		TargetLanguage: language.Serbian,
		OutputFormat:   "", // should default to epub
	}
	got, err := NewBatchProcessor(options).computeOutputPath("/some/book.fb2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, "_sr.epub") {
		t.Errorf("expected default epub suffix, got %q", got)
	}
}
