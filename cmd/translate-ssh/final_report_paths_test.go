package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// captureFinalReport runs printFinalReport and returns its stdout.
func captureFinalReport(progress *TranslationProgress) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(r)
		done <- buf.String()
	}()

	printFinalReport(progress)
	w.Close()
	os.Stdout = oldStdout
	return <-done
}

// TestPrintFinalReport_ExpectedFilesUseRealPaths is the RED-first reproduction
// (§11.4.115) of a real, user-impacting reporting defect:
//
// printFinalReport builds its "Expected Output Files" existence check with
//   fullPath := filepath.Join(inputDir, file)
// where `file` is, among others, the ABSOLUTE progress.InputFile and the
// ABSOLUTE progress.OutputFile. filepath.Join of an absolute base with an
// absolute second element produces a nonsense path
// (inputDir + "/" + outputFile), so the os.Stat existence check always looks at
// the wrong location.
//
// Consequence: after a SUCCESSFUL translation whose artifacts really exist at
// the operator-requested absolute paths, the final report prints
//   ✗ /home/.../srv/exports/novel_sr.epub (not found)
// telling the operator their output is missing when it is present at
// config.OutputFile. The report is a false negative about the user's artifact.
//
// The fix: resolve each expected path correctly (output EPUB at the exact
// configured OutputFile; sidecars next to the input file) so a file that
// genuinely exists is reported as found.
func TestPrintFinalReport_ExpectedFilesUseRealPaths(t *testing.T) {
	tmp := t.TempDir()
	inputDir := filepath.Join(tmp, "books")
	outDir := filepath.Join(tmp, "exports")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	inputFile := filepath.Join(inputDir, "novel.fb2")
	outputFile := filepath.Join(outDir, "novel_sr.epub") // different dir AND name

	// Create the real artifacts a successful run would have produced.
	mustWrite(t, inputFile, "<FictionBook/>")
	mustWrite(t, outputFile, "PK\x03\x04 real epub bytes")
	mustWrite(t, filepath.Join(inputDir, "novel_original.md"), "# original")
	mustWrite(t, filepath.Join(inputDir, "novel_translated.md"), "# translated")

	progress := &TranslationProgress{
		StartTime:      time.Now().Add(-time.Minute),
		CompletedSteps: 6,
		TotalSteps:     6,
		CurrentStep:    "Completed",
		InputFile:      inputFile,
		OutputFile:     outputFile,
	}

	out := captureFinalReport(progress)

	// The output EPUB genuinely exists at outputFile — the report MUST mark it found.
	if strings.Contains(out, "✗ "+outputFile) || !strings.Contains(out, "✓ "+outputFile) {
		t.Fatalf("output EPUB exists at %s but final report did not mark it found.\n--- report ---\n%s", outputFile, out)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
