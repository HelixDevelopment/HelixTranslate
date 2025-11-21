package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/batch"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/dictionary"
)

func TestBatchProcessor(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	t.Run("ProcessString", func(t *testing.T) {
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeString,
			InputString:    "Hello world",
			TargetLanguage: language.Serbian,
			Translator:     trans,
		}

		processor := batch.NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Fatalf("ProcessString failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if !results[0].Success {
			t.Errorf("Expected success, got failure: %v", results[0].Error)
		}
	})

	t.Run("ProcessStringWithOutput", func(t *testing.T) {
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		outputPath := filepath.Join(tmpDir, "output.txt")

		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeString,
			InputString:    "Test translation",
			OutputPath:     outputPath,
			TargetLanguage: language.Serbian,
			Translator:     trans,
		}

		processor := batch.NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Fatalf("ProcessStringWithOutput failed: %v", err)
		}

		if len(results) != 1 || !results[0].Success {
			t.Errorf("Expected successful processing")
		}

		// Verify output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Output file not created: %s", outputPath)
		}
	})

	t.Run("ProcessStdin", func(t *testing.T) {
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		inputReader := strings.NewReader("stdin test input")

		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeStdin,
			InputReader:    inputReader,
			TargetLanguage: language.Serbian,
			Translator:     trans,
		}

		processor := batch.NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Fatalf("ProcessStdin failed: %v", err)
		}

		if len(results) != 1 || !results[0].Success {
			t.Errorf("Expected successful processing")
		}
	})

	t.Run("FindSupportedFiles", func(t *testing.T) {
		// Create test directory structure
		testDir := filepath.Join(tmpDir, "testfiles")
		os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)

		// Create test files
		os.WriteFile(filepath.Join(testDir, "book1.epub"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(testDir, "book2.fb2"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(testDir, "text.txt"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(testDir, "doc.html"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(testDir, "invalid.xyz"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(testDir, "subdir", "book3.epub"), []byte("test"), 0644)

		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)

		// Test non-recursive
		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeDirectory,
			InputPath:      testDir,
			OutputPath:     filepath.Join(tmpDir, "output"),
			TargetLanguage: language.Serbian,
			Translator:     trans,
			Recursive:      false,
		}

		processor := batch.NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		// Should find 4 files (excluding .xyz and subdirectory)
		if len(results) < 4 {
			t.Errorf("Expected at least 4 files non-recursively, got %d", len(results))
		}

		// Test recursive
		options.Recursive = true
		processor = batch.NewBatchProcessor(options)
		results, err = processor.Process(context.Background())

		if err != nil {
			t.Fatalf("Recursive process failed: %v", err)
		}

		// Should find 5 files (including subdirectory)
		if len(results) < 5 {
			t.Errorf("Expected at least 5 files recursively, got %d", len(results))
		}
	})

	t.Run("ComputeOutputPath", func(t *testing.T) {
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		inputDir := filepath.Join(tmpDir, "input")
		outputDir := filepath.Join(tmpDir, "output_structured")
		os.MkdirAll(filepath.Join(inputDir, "subdir"), 0755)
		os.MkdirAll(outputDir, 0755)

		// Create test file
		inputFile := filepath.Join(inputDir, "subdir", "book.epub")
		os.WriteFile(inputFile, []byte("test"), 0644)

		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeDirectory,
			InputPath:      inputDir,
			OutputPath:     outputDir,
			OutputFormat:   "epub",
			TargetLanguage: language.Serbian,
			Translator:     trans,
			Recursive:      true,
		}

		processor := batch.NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Fatalf("Process failed: %v", err)
		}

		if len(results) == 0 {
			t.Fatal("Expected at least one result")
		}

		// Verify output path preserves structure
		result := results[0]
		if !strings.Contains(result.OutputPath, "subdir") {
			t.Errorf("Output path should preserve directory structure: %s", result.OutputPath)
		}

		if !strings.Contains(result.OutputPath, "_sr.epub") {
			t.Errorf("Output path should contain language suffix: %s", result.OutputPath)
		}
	})

	t.Run("ParallelProcessing", func(t *testing.T) {
		// Create multiple test files
		testDir := filepath.Join(tmpDir, "parallel_test")
		os.MkdirAll(testDir, 0755)

		for i := 0; i < 5; i++ {
			filename := filepath.Join(testDir, "book"+string(rune('0'+i))+".txt")
			os.WriteFile(filename, []byte("test content"), 0644)
		}

		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeDirectory,
			InputPath:      testDir,
			OutputPath:     filepath.Join(tmpDir, "parallel_output"),
			TargetLanguage: language.Serbian,
			Translator:     trans,
			Parallel:       true,
			MaxConcurrency: 3,
		}

		processor := batch.NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Fatalf("Parallel processing failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected results from parallel processing")
		}

		// Verify all results are present (even if some failed)
		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}

		t.Logf("Parallel processing: %d/%d successful", successCount, len(results))
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.MkdirAll(emptyDir, 0755)

		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		options := &batch.ProcessingOptions{
			InputType:      batch.InputTypeDirectory,
			InputPath:      emptyDir,
			OutputPath:     filepath.Join(tmpDir, "empty_output"),
			TargetLanguage: language.Serbian,
			Translator:     trans,
		}

		processor := batch.NewBatchProcessor(options)
		_, err := processor.Process(context.Background())

		if err == nil {
			t.Error("Expected error for empty directory")
		}

		if !strings.Contains(err.Error(), "no supported files") {
			t.Errorf("Expected 'no supported files' error, got: %v", err)
		}
	})

	t.Run("InvalidInputType", func(t *testing.T) {
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)
		options := &batch.ProcessingOptions{
			InputType:      batch.InputType(999), // Invalid type
			TargetLanguage: language.Serbian,
			Translator:     trans,
		}

		processor := batch.NewBatchProcessor(options)
		_, err := processor.Process(context.Background())

		if err == nil {
			t.Error("Expected error for invalid input type")
		}

		if !strings.Contains(err.Error(), "unsupported input type") {
			t.Errorf("Expected 'unsupported input type' error, got: %v", err)
		}
	})
}

func TestInputType(t *testing.T) {
	t.Run("InputTypeConstants", func(t *testing.T) {
		if batch.InputTypeFile != 0 {
			t.Error("InputTypeFile should be 0")
		}
		if batch.InputTypeString != 1 {
			t.Error("InputTypeString should be 1")
		}
		if batch.InputTypeStdin != 2 {
			t.Error("InputTypeStdin should be 2")
		}
		if batch.InputTypeDirectory != 3 {
			t.Error("InputTypeDirectory should be 3")
		}
	})
}
