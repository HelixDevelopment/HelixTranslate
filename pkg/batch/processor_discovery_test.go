package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/pkg/language"
)

// TestComputeOutputPath_NonexistentDirNoCollision is a REPRODUCE-FIRST (§11.4.115/§11.4.146)
// guard for the output-path collision bug.
//
// Root cause (FACT): computeOutputPath stats OutputPath. When OutputPath does not exist yet
// (the common "-output /some/new/dir" case), os.Stat returns IsNotExist, isOutputDir is false,
// and the function returns bp.options.OutputPath VERBATIM for EVERY input file. In directory
// batch mode that maps N distinct inputs onto ONE output path — every file silently overwrites
// the previous one (data loss). A non-existent output target that ends with no recognised
// ebook extension is unambiguously a destination DIRECTORY, not a single output file.
func TestComputeOutputPath_NonexistentDirNoCollision(t *testing.T) {
	tmp := t.TempDir()
	inputDir := filepath.Join(tmp, "in")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// OutputPath intentionally does NOT exist and has no ebook extension => a directory target.
	outDir := filepath.Join(tmp, "out")

	options := &ProcessingOptions{
		InputPath:      inputDir,
		OutputPath:     outDir,
		TargetLanguage: language.Serbian,
		OutputFormat:   "epub",
	}
	bp := NewBatchProcessor(options)

	in1 := filepath.Join(inputDir, "alpha.fb2")
	in2 := filepath.Join(inputDir, "beta.fb2")

	out1, err := bp.computeOutputPath(in1)
	if err != nil {
		t.Fatalf("computeOutputPath(alpha) error: %v", err)
	}
	out2, err := bp.computeOutputPath(in2)
	if err != nil {
		t.Fatalf("computeOutputPath(beta) error: %v", err)
	}

	if out1 == out2 {
		t.Fatalf("COLLISION: two distinct inputs map to the same output path %q — "+
			"the second file would overwrite the first (data loss)", out1)
	}
	// Each output must live under the requested directory, not be the bare directory path.
	if filepath.Dir(out1) != outDir || filepath.Dir(out2) != outDir {
		t.Errorf("expected outputs under %q, got %q and %q", outDir, out1, out2)
	}
}

// TestProcessDirectory_NonexistentOutputDir_AllFilesWritten drives the full directory pipeline
// (the user-visible behaviour) with a non-existent output directory and asserts that every input
// produces its own distinct, on-disk output file.
func TestProcessDirectory_NonexistentOutputDir_AllFilesWritten(t *testing.T) {
	tmp := t.TempDir()
	inputDir := filepath.Join(tmp, "in")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"alpha.fb2", "beta.fb2", "gamma.fb2"} {
		if err := os.WriteFile(filepath.Join(inputDir, n), []byte(validFB2), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outDir := filepath.Join(tmp, "out") // does not exist yet

	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      inputDir,
		OutputPath:     outDir,
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       false,
	}
	results, err := NewBatchProcessor(options).Process(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	seen := map[string]bool{}
	for _, r := range results {
		if !r.Success {
			t.Errorf("file %s failed: %v", r.InputPath, r.Error)
		}
		if seen[r.OutputPath] {
			t.Fatalf("COLLISION: output path %q used by more than one input (data loss)", r.OutputPath)
		}
		seen[r.OutputPath] = true
	}

	// Every distinct output path must actually exist on disk.
	for outPath := range seen {
		if _, statErr := os.Stat(outPath); statErr != nil {
			t.Errorf("expected output file %q to exist: %v", outPath, statErr)
		}
	}
}
