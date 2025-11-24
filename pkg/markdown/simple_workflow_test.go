package markdown

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/translator/llm"
)

func TestSimpleWorkflow_NewSimpleWorkflow(t *testing.T) {
	config := WorkflowConfig{
		ChunkSize:      1000,
		OverlapSize:    100,
		MaxConcurrency: 4,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	callback := func(current, total int, message string) {}

	workflow := NewSimpleWorkflow(config, testLogger, callback)

	if workflow.config.ChunkSize != 1000 {
		t.Errorf("Expected ChunkSize=1000, got %d", workflow.config.ChunkSize)
	}

	if workflow.config.OverlapSize != 100 {
		t.Errorf("Expected OverlapSize=100, got %d", workflow.config.OverlapSize)
	}

	if workflow.config.MaxConcurrency != 4 {
		t.Errorf("Expected MaxConcurrency=4, got %d", workflow.config.MaxConcurrency)
	}

	if workflow.logger == nil {
		t.Error("Logger not initialized")
	}

	if workflow.callback == nil {
		t.Error("Callback not set")
	}
}

func TestSimpleWorkflow_ConvertToMarkdown(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "workflow_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test EPUB file (simplified)
	inputPath := filepath.Join(tmpDir, "test.epub")
	outputPath := filepath.Join(tmpDir, "test.md")

	// Create a minimal EPUB structure
	os.MkdirAll(filepath.Join(tmpDir, "META-INF"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "OEBPS"), 0755)

	// Create mimetype file
	mimetypePath := filepath.Join(tmpDir, "mimetype")
	err = os.WriteFile(mimetypePath, []byte("application/epub+zip"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mimetype: %v", err)
	}

	// Create container.xml
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	err = os.WriteFile(filepath.Join(tmpDir, "META-INF", "container.xml"), []byte(containerXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create container.xml: %v", err)
	}

	// Create content.opf
	contentOPF := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata>
    <dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Test Book</dc:title>
    <dc:language xmlns:dc="http://purl.org/dc/elements/1.1/">en</dc:language>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`
	err = os.WriteFile(filepath.Join(tmpDir, "OEBPS", "content.opf"), []byte(contentOPF), 0644)
	if err != nil {
		t.Fatalf("Failed to create content.opf: %v", err)
	}

	// Create chapter1.xhtml
	chapterXHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
  <h1>Chapter 1</h1>
  <p>This is a test chapter with some content.</p>
  <p>Another paragraph for testing purposes.</p>
</body>
</html>`
	err = os.WriteFile(filepath.Join(tmpDir, "OEBPS", "chapter1.xhtml"), []byte(chapterXHTML), 0644)
	if err != nil {
		t.Fatalf("Failed to create chapter1.xhtml: %v", err)
	}

	// Create workflow
	config := WorkflowConfig{
		ChunkSize:      500,
		OverlapSize:    50,
		MaxConcurrency: 2,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	workflow := NewSimpleWorkflow(config, testLogger, nil)

	// Test conversion
	ctx := context.Background()
	err = workflow.ConvertToMarkdown(ctx, inputPath, outputPath)
	if err != nil {
		t.Fatalf("Failed to convert to markdown: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output markdown file was not created")
	}

	// Verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	if len(contentStr) == 0 {
		t.Error("Output file is empty")
	}

	// Should contain chapter content
	if !strings.Contains(contentStr, "Chapter 1") {
		t.Error("Output doesn't contain expected chapter title")
	}

	if !strings.Contains(contentStr, "test chapter") {
		t.Error("Output doesn't contain expected chapter content")
	}
}

func TestSimpleWorkflow_TranslateMarkdown(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "translate_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test markdown file
	inputPath := filepath.Join(tmpDir, "input.md")
	outputPath := filepath.Join(tmpDir, "output.md")

	markdownContent := `# Test Document

This is a test paragraph with some English text.

## Section 2

Another paragraph for translation testing.

* List item 1
* List item 2
* List item 3

This document contains various elements that should be translated.
`

	err = os.WriteFile(inputPath, []byte(markdownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create input markdown: %v", err)
	}

	// Create mock LLM client
	mockLLM := &MockLLMClient{
		responses: map[string]string{
			"This is a test paragraph with some English text.": "Ово је тестни параграф са енглеским текстом.",
			"Another paragraph for translation testing.":          "Још један параграф за тестирање превода.",
			"List item 1": "Ставка 1",
			"List item 2": "Ставка 2", 
			"List item 3": "Ставка 3",
			"This document contains various elements that should be translated.": "Овај документ садржи разне елементе који треба превести.",
		},
	}

	// Create workflow
	config := WorkflowConfig{
		ChunkSize:        200,
		OverlapSize:      20,
		MaxConcurrency:   2,
		TranslationCache: make(map[string]string),
		LLMProvider:      mockLLM,
	}

	progressCount := 0
	callback := func(current, total int, message string) {
		progressCount++
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	workflow := NewSimpleWorkflow(config, testLogger, callback)

	// Test translation
	ctx := context.Background()
	err = workflow.TranslateMarkdown(ctx, inputPath, outputPath, "en", "sr")
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Verify progress callback was called
	if progressCount == 0 {
		t.Error("Progress callback was not called")
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output translated file was not created")
	}

	// Verify translated content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	if len(contentStr) == 0 {
		t.Error("Output file is empty")
	}

	// Should contain translated text
	if !strings.Contains(contentStr, "Ово је тестни параграф") {
		t.Error("Output doesn't contain expected translated content")
	}

	if !strings.Contains(contentStr, "Још један параграф") {
		t.Error("Output doesn't contain expected translated content")
	}
}

func TestSimpleWorkflow_ConvertToEbook(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ebook_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test markdown file
	inputPath := filepath.Join(tmpDir, "input.md")
	outputPath := filepath.Join(tmpDir, "output.epub")

	markdownContent := `# Test Ebook

This is a test chapter for ebook conversion.

## Chapter 1

Content for chapter 1 with some text.

## Chapter 2

Content for chapter 2 with more text.

### Subsection

Some subsection content.
`

	err = os.WriteFile(inputPath, []byte(markdownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create input markdown: %v", err)
	}

	// Create workflow
	config := WorkflowConfig{
		ChunkSize:      500,
		OverlapSize:    50,
		MaxConcurrency: 2,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	workflow := NewSimpleWorkflow(config, testLogger, nil)

	// Test conversion
	ctx := context.Background()
	err = workflow.ConvertToEbook(ctx, inputPath, outputPath, "epub")
	if err != nil {
		t.Fatalf("Failed to convert to ebook: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Output ebook file was not created")
	}

	// Verify file size is reasonable
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("Output ebook file is empty")
	}

	// Basic EPUB validation - should contain ZIP signature
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	// Read first 4 bytes to check for ZIP signature
	signature := make([]byte, 4)
	_, err = file.Read(signature)
	if err != nil {
		t.Fatalf("Failed to read file signature: %v", err)
	}

	// ZIP files start with "PK" (0x50 0x4B)
	if signature[0] != 0x50 || signature[1] != 0x4B {
		t.Error("Output file is not a valid ZIP/EPUB file")
	}
}

func TestSimpleWorkflow_EndToEndWorkflow(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "e2e_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test EPUB file
	inputPath := filepath.Join(tmpDir, "input.epub")
	markdownPath := filepath.Join(tmpDir, "intermediate.md")
	translatedPath := filepath.Join(tmpDir, "translated.md")
	outputPath := filepath.Join(tmpDir, "output.epub")

	// Create a minimal EPUB structure (similar to previous test)
	os.MkdirAll(filepath.Join(tmpDir, "META-INF"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "OEBPS"), 0755)

	// mimetype
	err = os.WriteFile(filepath.Join(tmpDir, "mimetype"), []byte("application/epub+zip"), 0644)
	if err != nil {
		t.Fatalf("Failed to create mimetype: %v", err)
	}

	// container.xml
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	err = os.WriteFile(filepath.Join(tmpDir, "META-INF", "container.xml"), []byte(containerXML), 0644)
	if err != nil {
		t.Fatalf("Failed to create container.xml: %v", err)
	}

	// content.opf
	contentOPF := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata>
    <dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Test Book</dc:title>
    <dc:language xmlns:dc="http://purl.org/dc/elements/1.1/">en</dc:language>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`
	err = os.WriteFile(filepath.Join(tmpDir, "OEBPS", "content.opf"), []byte(contentOPF), 0644)
	if err != nil {
		t.Fatalf("Failed to create content.opf: %v", err)
	}

	// chapter1.xhtml
	chapterXHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
  <h1>Test Chapter</h1>
  <p>This is a test chapter for end-to-end workflow testing.</p>
  <p>It contains multiple paragraphs to test the translation pipeline.</p>
</body>
</html>`
	err = os.WriteFile(filepath.Join(tmpDir, "OEBPS", "chapter1.xhtml"), []byte(chapterXHTML), 0644)
	if err != nil {
		t.Fatalf("Failed to create chapter1.xhtml: %v", err)
	}

	// Create mock LLM client
	mockLLM := &MockLLMClient{
		responses: map[string]string{
			"This is a test chapter for end-to-end workflow testing.": "Ово је тестно поглавље за тестирање крај-до-крај радног тока.",
			"It contains multiple paragraphs to test the translation pipeline.": "Садржи више пасуса за тестирање линије превода.",
		},
	}

	// Create workflow
	config := WorkflowConfig{
		ChunkSize:        200,
		OverlapSize:      20,
		MaxConcurrency:   2,
		TranslationCache: make(map[string]string),
		LLMProvider:      mockLLM,
	}

	progressSteps := 0
	callback := func(current, total int, message string) {
		progressSteps++
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	workflow := NewSimpleWorkflow(config, testLogger, callback)

	ctx := context.Background()

	// Step 1: Convert EPUB to Markdown
	err = workflow.ConvertToMarkdown(ctx, inputPath, markdownPath)
	if err != nil {
		t.Fatalf("Failed to convert EPUB to markdown: %v", err)
	}

	// Verify intermediate markdown exists
	if _, err := os.Stat(markdownPath); os.IsNotExist(err) {
		t.Error("Intermediate markdown file was not created")
	}

	// Step 2: Translate Markdown
	err = workflow.TranslateMarkdown(ctx, markdownPath, translatedPath, "en", "sr")
	if err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Verify translated markdown exists
	if _, err := os.Stat(translatedPath); os.IsNotExist(err) {
		t.Error("Translated markdown file was not created")
	}

	// Step 3: Convert Markdown back to EPUB
	err = workflow.ConvertToEbook(ctx, translatedPath, outputPath, "epub")
	if err != nil {
		t.Fatalf("Failed to convert markdown to EPUB: %v", err)
	}

	// Verify final EPUB exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Final EPUB file was not created")
	}

	// Verify progress callback was called for all steps
	if progressSteps < 3 {
		t.Errorf("Expected at least 3 progress updates, got %d", progressSteps)
	}

	// Verify final EPUB is valid
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open final EPUB: %v", err)
	}
	defer file.Close()

	signature := make([]byte, 4)
	_, err = file.Read(signature)
	if err != nil {
		t.Fatalf("Failed to read EPUB signature: %v", err)
	}

	if signature[0] != 0x50 || signature[1] != 0x4B {
		t.Error("Final output is not a valid EPUB file")
	}
}

func TestSimpleWorkflow_ErrorHandling(t *testing.T) {
	// Create workflow
	config := WorkflowConfig{
		ChunkSize:      500,
		OverlapSize:    50,
		MaxConcurrency: 2,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	workflow := NewSimpleWorkflow(config, testLogger, nil)

	ctx := context.Background()

	// Test conversion with non-existent input file
	err := workflow.ConvertToMarkdown(ctx, "/nonexistent/input.epub", "/tmp/output.md")
	if err == nil {
		t.Error("Expected error when converting non-existent file")
	}

	// Test translation with non-existent input file
	err = workflow.TranslateMarkdown(ctx, "/nonexistent/input.md", "/tmp/output.md", "en", "sr")
	if err == nil {
		t.Error("Expected error when translating non-existent file")
	}

	// Test ebook conversion with non-existent input file
	err = workflow.ConvertToEbook(ctx, "/nonexistent/input.md", "/tmp/output.epub", "epub")
	if err == nil {
		t.Error("Expected error when converting non-existent file to ebook")
	}
}

func TestSimpleWorkflow_ContextCancellation(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cancel_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test markdown file
	inputPath := filepath.Join(tmpDir, "input.md")
	outputPath := filepath.Join(tmpDir, "output.md")

	markdownContent := `# Test Document

This is a test paragraph.

Another paragraph for testing.

This document contains multiple paragraphs.
`

	err = os.WriteFile(inputPath, []byte(markdownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create input markdown: %v", err)
	}

	// Create mock LLM client that delays
	mockLLM := &MockLLMClient{
		delay:      100 * time.Millisecond,
		responses: map[string]string{
			"This is a test paragraph.": "Ово је тестни параграф.",
			"Another paragraph for testing.": "Још један параграф за тестирање.",
		},
	}

	// Create workflow
	config := WorkflowConfig{
		ChunkSize:        100,
		OverlapSize:      10,
		MaxConcurrency:   2,
		TranslationCache: make(map[string]string),
		LLMProvider:      mockLLM,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "info"})
	workflow := NewSimpleWorkflow(config, testLogger, nil)

	// Create context that gets cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Test translation with cancelled context
	err = workflow.TranslateMarkdown(ctx, inputPath, outputPath, "en", "sr")
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context error, got: %v", err)
	}
}

// Mock LLM Client for testing
type MockLLMClient struct {
	responses map[string]string
	delay     time.Duration
}

func (m *MockLLMClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// Check context before processing
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if response, ok := m.responses[text]; ok {
		return response, nil
	}

	// Default response for unmapped text
	return "[TRANSLATED: " + text + "]", nil
}

func (m *MockLLMClient) GetProviderName() string {
	return "mock"
}

// Benchmark tests
func BenchmarkSimpleWorkflow_ConvertToMarkdown(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_markdown_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test EPUB
	inputPath := filepath.Join(tmpDir, "test.epub")
	// ... create minimal EPUB structure
	// (Implementation omitted for brevity)

	config := WorkflowConfig{
		ChunkSize:      1000,
		OverlapSize:    100,
		MaxConcurrency: 4,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "error"})
	workflow := NewSimpleWorkflow(config, testLogger, nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, fmt.Sprintf("output_%d.md", i))
		workflow.ConvertToMarkdown(ctx, inputPath, outputPath)
	}
}

func BenchmarkSimpleWorkflow_TranslateMarkdown(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench_translate_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test markdown
	inputPath := filepath.Join(tmpDir, "input.md")
	markdownContent := `# Test Document

This is a test paragraph.

Another paragraph for testing.

This document contains multiple paragraphs for benchmarking.
`
	err = os.WriteFile(inputPath, []byte(markdownContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create input markdown: %v", err)
	}

	mockLLM := &MockLLMClient{
		responses: map[string]string{
			"This is a test paragraph.": "Ово је тестни параграф.",
			"Another paragraph for testing.": "Још један параграф за тестирање.",
		},
	}

	config := WorkflowConfig{
		ChunkSize:        200,
		OverlapSize:      20,
		MaxConcurrency:   4,
		TranslationCache: make(map[string]string),
		LLMProvider:      mockLLM,
	}

	testLogger := logger.NewLogger(logger.Config{Level: "error"})
	workflow := NewSimpleWorkflow(config, testLogger, nil)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, fmt.Sprintf("output_%d.md", i))
		workflow.TranslateMarkdown(ctx, inputPath, outputPath, "en", "sr")
	}
}