package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/pkg/language"
)

// TestComputeOutputPath_SameStemDifferentExt_NoCollision is a REPRODUCE-FIRST
// (§11.4.115 / §11.4.146) guard for a data-loss collision the prior
// non-existent-dir fix did NOT cover.
//
// Root cause (FACT): computeOutputPath derives the output name from the input's
// stem with its extension STRIPPED (base = strings.TrimSuffix(path, ext)) and
// appends only the language + output format. Two supported inputs that share a
// stem but differ in extension (e.g. a library holding "book.fb2" AND
// "book.epub" — both supported formats found by findSupportedFiles) therefore
// map onto ONE output path. In a directory batch the second file silently
// overwrites the first (sequential) or both goroutines write the same file
// concurrently (parallel) — translated output is lost/corrupted.
//
// This collides in BOTH the OutputPath=="" (write-next-to-input) branch and the
// explicit-directory branch. The fix preserves the source extension in the
// output name so distinct sources never collide.
func TestComputeOutputPath_SameStemDifferentExt_NoCollision(t *testing.T) {
	t.Run("empty output (next to input)", func(t *testing.T) {
		bp := NewBatchProcessor(&ProcessingOptions{
			OutputPath:     "",
			TargetLanguage: language.Serbian,
			OutputFormat:   "epub",
		})
		o1, err := bp.computeOutputPath("/lib/book.fb2")
		if err != nil {
			t.Fatal(err)
		}
		o2, err := bp.computeOutputPath("/lib/book.epub")
		if err != nil {
			t.Fatal(err)
		}
		if o1 == o2 {
			t.Fatalf("COLLISION (data loss): book.fb2 and book.epub both map to %q", o1)
		}
	})

	t.Run("explicit directory", func(t *testing.T) {
		tmp := t.TempDir()
		inDir := filepath.Join(tmp, "in")
		if err := os.MkdirAll(inDir, 0o755); err != nil {
			t.Fatal(err)
		}
		outDir := filepath.Join(tmp, "out") // nonexistent => directory target
		bp := NewBatchProcessor(&ProcessingOptions{
			InputPath:      inDir,
			OutputPath:     outDir,
			TargetLanguage: language.Serbian,
			OutputFormat:   "epub",
		})
		o1, err := bp.computeOutputPath(filepath.Join(inDir, "book.fb2"))
		if err != nil {
			t.Fatal(err)
		}
		o2, err := bp.computeOutputPath(filepath.Join(inDir, "book.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if o1 == o2 {
			t.Fatalf("COLLISION (data loss): book.fb2 and book.txt both map to %q", o1)
		}
		if filepath.Dir(o1) != outDir || filepath.Dir(o2) != outDir {
			t.Errorf("outputs must live under %q, got %q and %q", outDir, o1, o2)
		}
	})
}

// TestProcessDirectory_SameStem_AllFilesWritten drives the full directory
// pipeline (user-visible behaviour) with two same-stem inputs and asserts each
// produces its own distinct on-disk output — proving no translation is lost.
func TestProcessDirectory_SameStem_AllFilesWritten(t *testing.T) {
	tmp := t.TempDir()
	inDir := filepath.Join(tmp, "in")
	if err := os.MkdirAll(inDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Two supported inputs sharing the stem "book".
	if err := os.WriteFile(filepath.Join(inDir, "book.fb2"), []byte(validFB2), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inDir, "book.txt"), []byte("Plain text body for translation."), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(tmp, "out")

	results, err := NewBatchProcessor(&ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      inDir,
		OutputPath:     outDir,
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       false,
	}).Process(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	seen := map[string]bool{}
	for _, r := range results {
		if !r.Success {
			t.Errorf("file %s failed: %v", r.InputPath, r.Error)
			continue
		}
		if seen[r.OutputPath] {
			t.Fatalf("COLLISION: output %q reused by more than one input (data loss)", r.OutputPath)
		}
		seen[r.OutputPath] = true
		if _, statErr := os.Stat(r.OutputPath); statErr != nil {
			t.Errorf("expected output file %q to exist: %v", r.OutputPath, statErr)
		}
	}
	if len(seen) != 2 {
		t.Fatalf("expected 2 distinct output files on disk, got %d", len(seen))
	}
}

// TestComputeOutputPath_SingleFileExplicit_Unchanged guards against regression:
// the explicit single-file output path must still be returned verbatim (no
// extension-discriminator mangling of a user-chosen file name).
func TestComputeOutputPath_SingleFileExplicit_Unchanged(t *testing.T) {
	tmp := t.TempDir()
	outFile := filepath.Join(tmp, "explicit.epub")
	if err := os.WriteFile(outFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	bp := NewBatchProcessor(&ProcessingOptions{
		InputPath:      tmp,
		OutputPath:     outFile,
		TargetLanguage: language.Serbian,
		OutputFormat:   "epub",
	})
	got, err := bp.computeOutputPath(filepath.Join(tmp, "in.fb2"))
	if err != nil {
		t.Fatal(err)
	}
	if got != outFile {
		t.Errorf("explicit single-file output must be verbatim %q, got %q", outFile, got)
	}
}
