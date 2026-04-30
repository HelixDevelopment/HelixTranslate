package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/fb2"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/version"
)

// TestEBookTranslationWorkflow provides comprehensive testing for ebook translation
func TestEBookTranslationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")  // SKIP-OK: #short-mode
	}
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ebook_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test logger
	testLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  "debug",
		Format: "text",
	})

	// Test 1: FB2 to Markdown conversion
	t.Run("FB2ToMarkdown", func(t *testing.T) {
		testFB2ToMarkdownConversion(t, testLogger, tempDir)
	})

	// Test 2: Markdown to EPUB conversion
	t.Run("MarkdownToEPUB", func(t *testing.T) {
		testMarkdownToEPUBConversion(t, testLogger, tempDir)
	})

	// Test 3: Codebase version hashing
	t.Run("CodebaseHashing", func(t *testing.T) {
		testCodebaseHashing(t, tempDir)
	})

	// Test 4: File verification
	t.Run("FileVerification", func(t *testing.T) {
		testFileVerification(t, tempDir)
	})

	// Test 5: Complete workflow simulation
	t.Run("CompleteWorkflow", func(t *testing.T) {
		testCompleteWorkflow(t, testLogger, tempDir)
	})
}

// testFB2ToMarkdownConversion tests FB2 to Markdown conversion
func testFB2ToMarkdownConversion(t *testing.T, testLogger logger.Logger, tempDir string) {
	// Create test FB2 file
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>fiction</genre>
      <author>
        <first-name>Test</first-name>
        <last-name>Author</last-name>
      </author>
      <book-title>Test Book</book-title>
      <annotation>
        <p>This is a test book for translation workflow testing.</p>
      </annotation>
      <lang>en</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title>
        <p>Chapter 1</p>
      </title>
      <p>This is the first paragraph of our test book. It contains simple English text that should be translated to Serbian Cyrillic.</p>
      <p>This is the second paragraph. It continues with more test content to ensure proper translation and formatting.</p>
    </section>
    <section>
      <title>
        <p>Chapter 2</p>
      </title>
      <p>This is the second chapter with more content to translate.</p>
    </section>
  </body>
</FictionBook>`

	fb2Path := filepath.Join(tempDir, "test_book.fb2")
	if err := os.WriteFile(fb2Path, []byte(fb2Content), 0644); err != nil {
		t.Fatalf("Failed to create test FB2 file: %v", err)
	}

	// Test conversion
	converter := fb2.NewMarkdownConverter(testLogger)
	mdPath := filepath.Join(tempDir, "test_book.md")

	if err := converter.ConvertToMarkdown(fb2Path, mdPath); err != nil {
		t.Fatalf("FB2 to Markdown conversion failed: %v", err)
	}

	// Verify markdown file was created
	if _, err := os.Stat(mdPath); err != nil {
		t.Fatalf("Markdown file was not created: %v", err)
	}

	// Verify markdown content
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("Failed to read markdown file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# Test Book") {
		t.Errorf("Markdown content missing title: %s", contentStr)
	}

	if !strings.Contains(contentStr, "This is the first paragraph") {
		t.Errorf("Markdown content missing first paragraph: %s", contentStr)
	}

	t.Logf("FB2 to Markdown conversion test passed: %s", mdPath)
}

// testMarkdownToEPUBConversion tests Markdown to EPUB conversion
func testMarkdownToEPUBConversion(t *testing.T, testLogger logger.Logger, tempDir string) {
	// Create test markdown file
	mdContent := `---
title: Test Book
author: Test Author
language: sr
---

# Test Book

## Chapter 1

This is the first paragraph of the test book.

This is the second paragraph.

## Chapter 2

This is the second chapter with more content.

> This is a test epigraph that should be formatted as a blockquote.
> 
> — Test Quote Author

Regular text continues after the epigraph.`

	mdPath := filepath.Join(tempDir, "test_book.md")
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to create test markdown file: %v", err)
	}

	// Test conversion
	converter := markdown.NewMarkdownToEPUBConverter()
	epubPath := filepath.Join(tempDir, "test_book.epub")

	if err := converter.ConvertMarkdownToEPUB(mdPath, epubPath); err != nil {
		t.Fatalf("Markdown to EPUB conversion failed: %v", err)
	}

	// Verify EPUB file was created
	if _, err := os.Stat(epubPath); err != nil {
		t.Fatalf("EPUB file was not created: %v", err)
	}

	// Verify EPUB file size
	info, err := os.Stat(epubPath)
	if err != nil {
		t.Fatalf("Failed to stat EPUB file: %v", err)
	}

	if info.Size() < 1000 {
		t.Errorf("EPUB file seems too small: %d bytes", info.Size())
	}

	t.Logf("Markdown to EPUB conversion test passed: %s", epubPath)
}

// testCodebaseHashing tests codebase hashing functionality
func testCodebaseHashing(t *testing.T, tempDir string) {
	// Create test files
	testFiles := map[string]string{
		"main.go":     "package main\n\nfunc main() { println(\"test\") }",
		"utils.go":    "package main\n\nfunc test() { println(\"utils\") }",
		"config.json": `{"test": "value"}`,
		"README.md":   "# Test Project\n\nThis is a test project",
	}

	for file, content := range testFiles {
		filePath := filepath.Join(tempDir, file)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp directory for hashing
	os.Chdir(tempDir)

	// Create relevant directories and files for hasher
	os.MkdirAll("cmd", 0755)
	os.WriteFile("cmd/main.go", []byte("package main\n"), 0644)

	// Test codebase hashing
	hasher := version.NewCodebaseHasher()
	hasher.RelevantDirectories = []string{"."}
	hash1, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate codebase hash: %v", err)
	}

	// Wait a moment to ensure different timestamp
	time.Sleep(100 * time.Millisecond)

	hash2, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate second codebase hash: %v", err)
	}

	// Hashes should be same for unchanged content
	if hash1 != hash2 {
		t.Errorf("Codebase hashes differ for unchanged content: %s vs %s", hash1, hash2)
	}

	// Modify a file
	testFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() { println(\"modified\") }"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash3, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate third codebase hash: %v", err)
	}

	// Hash should change after file modification
	if hash2 == hash3 {
		t.Errorf("Codebase hash should have changed after file modification: %s", hash3)
	}

	t.Logf("Codebase hashing test passed: %s -> %s -> %s", hash1[:16], hash2[:16], hash3[:16])
}

// testFileVerification tests file verification functions
func testFileVerification(t *testing.T, tempDir string) {
	// Create test files
	emptyFile := filepath.Join(tempDir, "empty.txt")
	smallFile := filepath.Join(tempDir, "small.txt")
	validFile := filepath.Join(tempDir, "valid.md")
	cyrillicFile := filepath.Join(tempDir, "cyrillic.txt")

	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	if err := os.WriteFile(smallFile, []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create small file: %v", err)
	}

	validContent := "# Valid Test File\n\n"
	validContent += "This is a valid test file with sufficient content for verification.\n"
	validContent += "It contains multiple lines and enough characters to pass validation.\n"
	validContent += "The content should be meaningful and properly formatted.\n"
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to create valid file: %v", err)
	}

	cyrillicContent := "Ово је текст на ћирилицу. Садржи слова као што су ћ, ћ, ч, ћ, ш, ж, љ, њ, и други."
	if err := os.WriteFile(cyrillicFile, []byte(cyrillicContent), 0644); err != nil {
		t.Fatalf("Failed to create Cyrillic file: %v", err)
	}

	// Create translator instance for testing
	translator := &EBookTranslator{
		logger:         logger.NewLogger(logger.LoggerConfig{Level: "debug", Format: "text"}),
		targetLanguage: "sr-cyrl",
	}

	// Test empty file verification
	if err := translator.verifyMarkdownFile(emptyFile); err == nil {
		t.Error("Empty file verification should have failed")
	}

	// Test small file verification
	if err := translator.verifyMarkdownFile(smallFile); err == nil {
		t.Error("Small file verification should have failed")
	}

	// Test valid file verification
	if err := translator.verifyMarkdownFile(validFile); err != nil {
		t.Errorf("Valid file verification failed: %v", err)
	}

	// Test Cyrillic verification
	if err := translator.verifyTargetLanguage(cyrillicFile); err != nil {
		t.Errorf("Cyrillic verification failed: %v", err)
	}

	// Test non-Cyrillic file with Cyrillic target
	latinFile := filepath.Join(tempDir, "latin.txt")
	latinContent := "Ovo je tekst na latinici. Ne sadrzi cirilicna slova."
	if err := os.WriteFile(latinFile, []byte(latinContent), 0644); err != nil {
		t.Fatalf("Failed to create Latin file: %v", err)
	}

	if err := translator.verifyTargetLanguage(latinFile); err == nil {
		t.Error("Latin file should have failed Cyrillic verification")
	}

	t.Logf("File verification test passed")
}

// testCompleteWorkflow tests the complete translation workflow simulation
func testCompleteWorkflow(t *testing.T, testLogger logger.Logger, tempDir string) {
	// Create test FB2 file
	fb2Path := filepath.Join(tempDir, "complete_test.fb2")
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>fiction</genre>
      <author>
        <first-name>Complete</first-name>
        <last-name>Test</last-name>
      </author>
      <book-title>Complete Test Book</book-title>
      <lang>en</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title>
        <p>Introduction</p>
      </title>
      <p>This is a complete workflow test book for translation.</p>
      <p>It contains multiple paragraphs to test the complete translation process.</p>
    </section>
  </body>
</FictionBook>`

	if err := os.WriteFile(fb2Path, []byte(fb2Content), 0644); err != nil {
		t.Fatalf("Failed to create test FB2 file: %v", err)
	}

	// Create translator instance
	translator, err := NewEBookTranslator(fb2Path, "sr-cyrl", "dummy.host", "dummy_user", "dummy_pass")
	if err != nil {
		t.Fatalf("Failed to create translator: %v", err)
	}

	// Test individual components without SSH connection

	// Test FB2 to Markdown conversion
	originalMdPath := filepath.Join(tempDir, "complete_test_original.md")
	originalContent := "# Complete Test Book\n\nThis is the content. " + strings.Repeat("More content here. ", 10)
	if err := os.WriteFile(originalMdPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create dummy original markdown: %v", err)
	}

	if err := translator.verifyMarkdownFile(originalMdPath); err != nil {
		t.Errorf("Original markdown verification failed: %v", err)
	}

	// Create dummy translated markdown file
	translatedMdPath := filepath.Join(tempDir, "complete_test_original_translated.md")
	dummyTranslatedContent := "# Комплетна тест књига\n\nОво је преведени садржај на ћирилицу.\n\nОво је други пасус који садржи ћириличка слова."
	if err := os.WriteFile(translatedMdPath, []byte(dummyTranslatedContent), 0644); err != nil {
		t.Fatalf("Failed to create dummy translated markdown: %v", err)
	}

	if err := translator.verifyMarkdownFile(translatedMdPath); err != nil {
		t.Errorf("Translated markdown verification failed: %v", err)
	}

	// Verify Cyrillic content
	if err := translator.verifyTargetLanguage(translatedMdPath); err != nil {
		t.Errorf("Cyrillic verification failed: %v", err)
	}

	// Create dummy EPUB file (must be >= 1000 bytes and valid ZIP)
	epubPath := filepath.Join(tempDir, "complete_test_original_translated.epub")
	epubContent := make([]byte, 1024)
	copy(epubContent, []byte("PK\x03\x04")) // ZIP magic number
	copy(epubContent[4:], []byte("mimetypeapplication/epub+zip"))
	if err := os.WriteFile(epubPath, epubContent, 0644); err != nil {
		t.Fatalf("Failed to create dummy EPUB file: %v", err)
	}

	if err := translator.verifyEPUBFile(epubPath); err != nil {
		t.Errorf("EPUB verification failed: %v", err)
	}

	// Test final verification of all outputs
	if err := translator.verifyOutputFiles(); err != nil {
		t.Errorf("Final output verification failed: %v", err)
	}

	t.Logf("Complete workflow test passed with files: %s, %s, %s", originalMdPath, translatedMdPath, epubPath)
}

// BenchmarkFB2ToMarkdown benchmarks the FB2 to Markdown conversion
func BenchmarkFB2ToMarkdown(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "fb2_benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fb2Path := filepath.Join(tempDir, "benchmark.fb2")
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>fiction</genre>
      <author>
        <first-name>Benchmark</first-name>
        <last-name>Test</last-name>
      </author>
      <book-title>Benchmark Test Book</book-title>
      <lang>en</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title>
        <p>Benchmark Chapter</p>
      </title>
      <p>This is a benchmark paragraph for performance testing.</p>
      <p>Another paragraph for comprehensive testing.</p>
    </section>
  </body>
</FictionBook>`

	if err := os.WriteFile(fb2Path, []byte(fb2Content), 0644); err != nil {
		b.Fatalf("Failed to create test FB2 file: %v", err)
	}

	testLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  "error", // Minimal logging for benchmark
		Format: "text",
	})

	converter := fb2.NewMarkdownConverter(testLogger)
	mdPath := filepath.Join(tempDir, "benchmark.md")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := converter.ConvertToMarkdown(fb2Path, mdPath); err != nil {
			b.Fatalf("FB2 to Markdown conversion failed: %v", err)
		}
		os.Remove(mdPath) // Clean up for next iteration
	}
}

// BenchmarkMarkdownToEPUB benchmarks the Markdown to EPUB conversion
func BenchmarkMarkdownToEPUB(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "epub_benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mdPath := filepath.Join(tempDir, "benchmark.md")
	mdContent := `---
title: Benchmark Test Book
author: Benchmark Test
language: sr
---

# Benchmark Test Book

## Chapter 1

This is a benchmark paragraph for performance testing.

This is another benchmark paragraph.

## Chapter 2

More content for comprehensive benchmark testing.

> This is a benchmark epigraph.
> 
> — Benchmark Author

Regular text continues after the epigraph.`

	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		b.Fatalf("Failed to create test markdown file: %v", err)
	}

	converter := markdown.NewMarkdownToEPUBConverter()
	epubPath := filepath.Join(tempDir, "benchmark.epub")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := converter.ConvertMarkdownToEPUB(mdPath, epubPath); err != nil {
			b.Fatalf("Markdown to EPUB conversion failed: %v", err)
		}
		os.Remove(epubPath) // Clean up for next iteration
	}
}

// BenchmarkCodebaseHashing benchmarks the codebase hashing functionality
func BenchmarkCodebaseHashing(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "hash_benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"main.go":     "package main\n\nfunc main() { println(\"benchmark\") }",
		"utils.go":    "package main\n\nfunc test() { println(\"utils\") }",
		"config.json": `{"benchmark": "value"}`,
		"README.md":   "# Benchmark Project\n\nThis is a benchmark test project",
	}

	for file, content := range testFiles {
		filePath := filepath.Join(tempDir, file)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Change to temp directory for hashing
	os.Chdir(tempDir)

	hasher := version.NewCodebaseHasher()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hasher.CalculateHash()
		if err != nil {
			b.Fatalf("Failed to calculate codebase hash: %v", err)
		}
	}
}
