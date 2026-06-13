package batch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/pkg/language"
)

// §11.4.146 STEP 1 — REPRODUCE-FIRST.
// Process(ctx) accepts a context but the sequential/parallel directory loops never honor
// cancellation: a caller that cancels mid-run (or before the run) still has every file
// parsed + written. These tests prove the leak by cancelling the context up front and
// asserting that processing stops with a context error and produces no successful results.
//
// On the PRE-FIX code both tests FAIL (every file is processed, top-level err is nil).

func makeDirWithFiles(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	for i := 0; i < n; i++ {
		name := filepath.Join(dir, "book"+string(rune('a'+i))+".fb2")
		if err := os.WriteFile(name, []byte(validFB2), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestProcessDirectory_Sequential_HonorsCancellation(t *testing.T) {
	dir := makeDirWithFiles(t, 3)
	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      dir,
		OutputPath:     filepath.Join(t.TempDir(), "out"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before Process runs

	results, err := NewBatchProcessor(options).Process(ctx)
	if err == nil {
		t.Fatalf("expected a context cancellation error, got nil (cancellation not honored); results=%d", len(results))
	}
	for _, r := range results {
		if r.Success {
			t.Errorf("file %s was processed despite cancelled context", r.InputPath)
		}
	}
}

func TestProcessDirectory_Parallel_HonorsCancellation(t *testing.T) {
	dir := makeDirWithFiles(t, 6)
	options := &ProcessingOptions{
		InputType:      InputTypeDirectory,
		InputPath:      dir,
		OutputPath:     filepath.Join(t.TempDir(), "out"),
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
		Parallel:       true,
		MaxConcurrency: 2,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, err := NewBatchProcessor(options).Process(ctx)
	if err == nil {
		t.Fatalf("expected a context cancellation error, got nil (cancellation not honored); results=%d", len(results))
	}
	for _, r := range results {
		if r.Success {
			t.Errorf("file %s was processed despite cancelled context", r.InputPath)
		}
	}
}
