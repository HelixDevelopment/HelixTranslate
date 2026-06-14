package batch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
)

// MockTranslator is a mock translator for testing
type MockTranslator struct {
	translateFunc func(ctx context.Context, text string, contextStr string) (string, error)
}

func (mt *MockTranslator) GetName() string {
	return "mock-translator"
}

func (mt *MockTranslator) Translate(ctx context.Context, text string, contextStr string) (string, error) {
	if mt.translateFunc != nil {
		return mt.translateFunc(ctx, text, contextStr)
	}
	return "translated: " + text, nil
}

func (mt *MockTranslator) TranslateWithProgress(ctx context.Context, text string, contextStr string, eventBus *events.EventBus, sessionID string) (string, error) {
	return mt.Translate(ctx, text, contextStr)
}

func (mt *MockTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{
		Total:      1,
		Translated: 1,
		Cached:     0,
		Errors:     0,
	}
}

func TestNewBatchProcessor(t *testing.T) {
	options := &ProcessingOptions{
		InputType:   InputTypeString,
		InputString: "test",
		Translator:  &MockTranslator{},
	}

	processor := NewBatchProcessor(options)
	if processor == nil {
		t.Error("NewBatchProcessor returned nil")
	}
	if processor.options != options {
		t.Error("Processor options not set correctly")
	}
}

func TestBatchProcessor_ProcessString(t *testing.T) {
	t.Run("EmptyString", func(t *testing.T) {
		options := &ProcessingOptions{
			InputType:   InputTypeString,
			InputString: "",
			Translator:  &MockTranslator{},
		}

		processor := NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err == nil {
			t.Error("Expected error for empty string")
		}
		if results != nil {
			t.Error("Expected nil results for empty string")
		}
	})

	t.Run("ValidString", func(t *testing.T) {
		options := &ProcessingOptions{
			InputType:   InputTypeString,
			InputString: "Hello World",
			Translator:  &MockTranslator{},
		}

		processor := NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if !results[0].Success {
			t.Error("Expected success")
		}
		if results[0].InputPath != "<string>" {
			t.Errorf("Expected input path '<string>', got %s", results[0].InputPath)
		}
	})

	t.Run("StringWithOutputFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "output.txt")

		options := &ProcessingOptions{
			InputType:   InputTypeString,
			InputString: "Hello World",
			OutputPath:  outputFile,
			Translator:  &MockTranslator{},
		}

		processor := NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if !results[0].Success {
			t.Error("Expected success")
		}

		// Check file was created
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Error("Output file was not created")
		}

		// Check file contents
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}
		expected := "translated: Hello World"
		if string(content) != expected {
			t.Errorf("Expected content %q, got %q", expected, string(content))
		}
	})
}

func TestBatchProcessor_ProcessStdin(t *testing.T) {
	t.Run("ValidInput", func(t *testing.T) {
		input := "Hello from stdin"
		reader := strings.NewReader(input)

		options := &ProcessingOptions{
			InputType:   InputTypeStdin,
			InputReader: reader,
			Translator:  &MockTranslator{},
		}

		processor := NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if !results[0].Success {
			t.Error("Expected success")
		}
		if results[0].InputPath != "<stdin>" {
			t.Errorf("Expected input path '<stdin>', got %s", results[0].InputPath)
		}
	})

	t.Run("WithOutputFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "output.txt")
		input := "Hello from stdin"
		reader := strings.NewReader(input)

		options := &ProcessingOptions{
			InputType:   InputTypeStdin,
			InputReader: reader,
			OutputPath:  outputFile,
			Translator:  &MockTranslator{},
		}

		processor := NewBatchProcessor(options)
		_, err := processor.Process(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Check file was created and has correct content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}
		expected := "translated: Hello from stdin"
		if string(content) != expected {
			t.Errorf("Expected content %q, got %q", expected, string(content))
		}
	})
}

func TestBatchProcessor_ProcessFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test FB2 file
	inputFile := filepath.Join(tmpDir, "test.fb2")
	fb2Content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Test Book</book-title>
    </title-info>
  </description>
  <body>
    <section>
      <p>Hello World</p>
    </section>
  </body>
</FictionBook>`

	err := os.WriteFile(inputFile, []byte(fb2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "output.epub")

	options := &ProcessingOptions{
		InputType:      InputTypeFile,
		InputPath:      inputFile,
		OutputPath:     outputFile,
		SourceLanguage: language.Russian,
		TargetLanguage: language.Serbian,
		Translator:     &MockTranslator{},
	}

	processor := NewBatchProcessor(options)
	results, err := processor.Process(context.Background())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}
	if results[0].InputPath != inputFile {
		t.Errorf("Expected input path %s, got %s", inputFile, results[0].InputPath)
	}
	if results[0].OutputPath != outputFile {
		t.Errorf("Expected output path %s, got %s", outputFile, results[0].OutputPath)
	}

	// Check output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}
}

func TestBatchProcessor_ProcessDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	// Create input directory
	err := os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	// Create test files
	fb2Content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <body><section><p>Test</p></section></body>
</FictionBook>`

	file1 := filepath.Join(inputDir, "test1.fb2")
	file2 := filepath.Join(inputDir, "test2.fb2")
	file3 := filepath.Join(inputDir, "test.txt") // Unsupported format

	err = os.WriteFile(file1, []byte(fb2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	err = os.WriteFile(file2, []byte(fb2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}
	err = os.WriteFile(file3, []byte("text content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file 3: %v", err)
	}

	t.Run("SequentialProcessing", func(t *testing.T) {
		options := &ProcessingOptions{
			InputType:      InputTypeDirectory,
			InputPath:      inputDir,
			OutputPath:     outputDir,
			SourceLanguage: language.Russian,
			TargetLanguage: language.Serbian,
			Translator:     &MockTranslator{},
			Parallel:       false,
		}

		processor := NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 3 { // fb2 and epub files should be processed
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		for _, result := range results {
			if !result.Success {
				t.Errorf("Expected success for %s, got error: %v", result.InputPath, result.Error)
			}
		}
	})

	t.Run("ParallelProcessing", func(t *testing.T) {
		options := &ProcessingOptions{
			InputType:      InputTypeDirectory,
			InputPath:      inputDir,
			OutputPath:     outputDir,
			SourceLanguage: language.Russian,
			TargetLanguage: language.Serbian,
			Translator:     &MockTranslator{},
			Parallel:       true,
			MaxConcurrency: 2,
		}

		processor := NewBatchProcessor(options)
		results, err := processor.Process(context.Background())

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		for _, result := range results {
			if !result.Success {
				t.Errorf("Expected success for %s, got error: %v", result.InputPath, result.Error)
			}
		}
	})
}

func TestBatchProcessor_ComputeOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	inputDir := filepath.Join(tmpDir, "input")
	outputDir := filepath.Join(tmpDir, "output")

	err := os.MkdirAll(inputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	options := &ProcessingOptions{
		InputPath:      inputDir,
		OutputPath:     outputDir,
		TargetLanguage: language.Serbian,
		OutputFormat:   "epub",
	}

	processor := NewBatchProcessor(options)

	t.Run("FileInRootDirectory", func(t *testing.T) {
		inputFile := filepath.Join(inputDir, "test.fb2")
		outputPath, err := processor.computeOutputPath(inputFile)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Source extension is embedded in the auto-name so same-stem inputs
		// (e.g. test.fb2 + test.txt) do not collide onto one output file.
		expected := filepath.Join(outputDir, "test_fb2_sr.epub")
		if outputPath != expected {
			t.Logf("InputDir: %s, OutputDir: %s", inputDir, outputDir)
			t.Logf("InputFile: %s", inputFile)
			t.Errorf("Expected output path %s, got %s", expected, outputPath)
		}
	})

	t.Run("FileInSubdirectory", func(t *testing.T) {
		subDir := filepath.Join(inputDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		inputFile := filepath.Join(subDir, "test.fb2")
		outputPath, err := processor.computeOutputPath(inputFile)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := filepath.Join(outputDir, "subdir", "test_fb2_sr.epub")
		if outputPath != expected {
			t.Errorf("Expected output path %s, got %s", expected, outputPath)
		}
	})

	t.Run("NoOutputDirectory", func(t *testing.T) {
		options := &ProcessingOptions{
			InputPath:      inputDir,
			OutputPath:     "",
			TargetLanguage: language.Serbian,
			OutputFormat:   "epub",
		}

		processor := NewBatchProcessor(options)
		inputFile := filepath.Join(inputDir, "test.fb2")
		outputPath, err := processor.computeOutputPath(inputFile)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := filepath.Join(inputDir, "test_fb2_sr.epub")
		if outputPath != expected {
			t.Errorf("Expected output path %s, got %s", expected, outputPath)
		}
	})
}

func TestBatchProcessor_FindSupportedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	fb2File := filepath.Join(tmpDir, "test.fb2")
	epubFile := filepath.Join(tmpDir, "test.epub")
	txtFile := filepath.Join(tmpDir, "test.txt")
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	fb2InSubdir := filepath.Join(subDir, "subtest.fb2")

	err = os.WriteFile(fb2File, []byte("fb2 content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create FB2 file: %v", err)
	}
	err = os.WriteFile(epubFile, []byte("epub content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create EPUB file: %v", err)
	}
	err = os.WriteFile(txtFile, []byte("txt content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create TXT file: %v", err)
	}
	err = os.WriteFile(fb2InSubdir, []byte("fb2 in subdir"), 0644)
	if err != nil {
		t.Fatalf("Failed to create FB2 file in subdir: %v", err)
	}

	options := &ProcessingOptions{}
	processor := NewBatchProcessor(options)

	t.Run("NonRecursive", func(t *testing.T) {
		files, err := processor.findSupportedFiles(tmpDir, false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should find fb2, epub, and txt files in root directory only
		expectedFiles := 3 // fb2, epub, and txt
		if len(files) != expectedFiles {
			t.Errorf("Expected %d files, got %d: %v", expectedFiles, len(files), files)
		}
	})

	t.Run("Recursive", func(t *testing.T) {
		files, err := processor.findSupportedFiles(tmpDir, true)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should find fb2, epub, and txt files in root and subdirectories
		expectedFiles := 4 // fb2, epub, txt in root, fb2 in subdir
		if len(files) != expectedFiles {
			t.Errorf("Expected %d files, got %d: %v", expectedFiles, len(files), files)
		}
	})
}

func TestProcessingResult(t *testing.T) {
	result := ProcessingResult{
		InputPath:  "/input/test.fb2",
		OutputPath: "/output/test_sr.epub",
		Success:    true,
		Error:      nil,
	}

	if result.InputPath != "/input/test.fb2" {
		t.Errorf("Expected input path %s, got %s", "/input/test.fb2", result.InputPath)
	}
	if result.OutputPath != "/output/test_sr.epub" {
		t.Errorf("Expected output path %s, got %s", "/output/test_sr.epub", result.OutputPath)
	}
	if !result.Success {
		t.Error("Expected success to be true")
	}
	if result.Error != nil {
		t.Errorf("Expected error to be nil, got %v", result.Error)
	}
}
