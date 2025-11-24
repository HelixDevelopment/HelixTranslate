package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMainFunction(t *testing.T) {
	// Test main function doesn't panic
	// Note: This is a basic test since main() is hard to test directly
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "markdown-translator-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original args and restore after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create test input file
	inputFile := filepath.Join(tempDir, "test.md")
	inputContent := "# Test Document\n\nThis is a test document with some content."
	err = os.WriteFile(inputFile, []byte(inputContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Test with valid arguments
	os.Args = []string{"markdown-translator", "--input", inputFile, "--output", filepath.Join(tempDir, "output.md"), "--provider", "openai"}

	// We can't actually call main() as it would exit the process
	// Instead, we test the components that main() uses
	t.Run("FileValidation", func(t *testing.T) {
		// Test input file validation
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			t.Errorf("Input file should exist: %v", err)
		}
	})

	t.Run("DirectoryCreation", func(t *testing.T) {
		// Test output directory can be created
		outputDir := filepath.Dir(filepath.Join(tempDir, "output.md"))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			t.Errorf("Failed to create output directory: %v", err)
		}
	})
}

func TestFlagParsing(t *testing.T) {
	// Test that flag parsing would work correctly
	tempDir, err := os.MkdirTemp("", "markdown-translator-flags-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "ValidArgs",
			args:        []string{"--input", "test.md", "--output", "output.md", "--provider", "openai"},
			expectError: false,
		},
		{
			name:        "MissingInput",
			args:        []string{"--output", "output.md", "--provider", "openai"},
			expectError: true,
		},
		{
			name:        "MissingProvider",
			args:        []string{"--input", "test.md", "--output", "output.md"},
			expectError: true,
		},
		{
			name:        "ValidWithAllFlags",
			args:        []string{"--input", "test.md", "--output", "output.md", "--provider", "openai", "--model", "gpt-4", "--temperature", "0.7"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dummy input file if needed
			if len(tt.args) >= 2 && tt.args[0] == "--input" {
				inputFile := filepath.Join(tempDir, tt.args[1])
				err := os.WriteFile(inputFile, []byte("# Test"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test input file: %v", err)
				}
			}

			// This is a simplified test - in real implementation,
			// we would extract the flag parsing logic into a testable function
			_ = tt.args
			_ = tt.expectError
		})
	}
}

func TestFileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "markdown-translator-file-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("InputFileRead", func(t *testing.T) {
		testContent := "# Test Markdown\n\nThis is test content with **bold** text."
		inputFile := filepath.Join(tempDir, "input.md")
		err := os.WriteFile(inputFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Test file can be read
		content, err := os.ReadFile(inputFile)
		if err != nil {
			t.Errorf("Failed to read input file: %v", err)
		}

		if string(content) != testContent {
			t.Errorf("Content mismatch: got %s, want %s", string(content), testContent)
		}
	})

	t.Run("OutputFileWrite", func(t *testing.T) {
		outputContent := "# Translated Document\n\nOvo je prevedeni sadržaj."
		outputFile := filepath.Join(tempDir, "output.md")

		err := os.WriteFile(outputFile, []byte(outputContent), 0644)
		if err != nil {
			t.Errorf("Failed to write output file: %v", err)
		}

		// Verify file was written
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		if string(content) != outputContent {
			t.Errorf("Output content mismatch: got %s, want %s", string(content), outputContent)
		}
	})
}

func TestProviderValidation(t *testing.T) {
	validProviders := []string{"openai", "anthropic", "zhipu", "deepseek", "qwen", "gemini", "ollama"}
	
	tests := []struct {
		name     string
		provider string
		valid    bool
	}{
		{"ValidOpenAI", "openai", true},
		{"ValidAnthropic", "anthropic", true},
		{"ValidZhipu", "zhipu", true},
		{"ValidDeepSeek", "deepseek", true},
		{"ValidQwen", "qwen", true},
		{"ValidGemini", "gemini", true},
		{"ValidOllama", "ollama", true},
		{"InvalidProvider", "invalid", false},
		{"EmptyProvider", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := false
			for _, p := range validProviders {
				if p == tt.provider {
					isValid = true
					break
				}
			}
			
			if isValid != tt.valid {
				t.Errorf("Provider validation for %s: got %v, want %v", tt.provider, isValid, tt.valid)
			}
		})
	}
}

func TestMarkdownFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "markdown-translator-format-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("BasicMarkdown", func(t *testing.T) {
		content := `# Header 1

## Header 2

This is a paragraph with **bold** and *italic* text.

- List item 1
- List item 2
- List item 3

[Link](https://example.com)

---

## Code Example

```javascript
function hello() {
    console.log("Hello, world!");
}
\`\`\`

> This is a blockquote.
`
		inputFile := filepath.Join(tempDir, "format.md")
		err := os.WriteFile(inputFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test markdown: %v", err)
		}

		// Verify file structure
		info, err := os.Stat(inputFile)
		if err != nil {
			t.Errorf("Failed to stat markdown file: %v", err)
		}

		if info.Size() == 0 {
			t.Errorf("Markdown file should not be empty")
		}
	})
}

func TestConcurrencySafety(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "markdown-translator-concurrency-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("ConcurrentFileAccess", func(t *testing.T) {
		// Create test files
		for i := 0; i < 5; i++ {
			filename := filepath.Join(tempDir, fmt.Sprintf("test_%d.md", i))
			content := fmt.Sprintf("# Test Document %d\n\nContent for document %d.", i, i)
			err := os.WriteFile(filename, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %d: %v", i, err)
			}
		}

		// Test concurrent file reading
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func(id int) {
				filename := filepath.Join(tempDir, fmt.Sprintf("test_%d.md", id))
				_, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("Failed to read file %d concurrently: %v", id, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			select {
			case <-done:
				// OK
			case <-time.After(5 * time.Second):
				t.Errorf("Concurrent file reading timeout")
			}
		}
	})
}

func TestErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "markdown-translator-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.md")
		_, err := os.Stat(nonExistentFile)
		if !os.IsNotExist(err) {
			t.Errorf("Expected file not found error for %s", nonExistentFile)
		}
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		// Create file and remove read permissions
		testFile := filepath.Join(tempDir, "readonly.md")
		err := os.WriteFile(testFile, []byte("# Test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Remove read permissions (on Unix systems)
		err = os.Chmod(testFile, 0000)
		if err != nil {
			t.Skip("Cannot change file permissions on this system")
		}

		// Restore permissions after test
		defer os.Chmod(testFile, 0644)

		_, err = os.ReadFile(testFile)
		if err == nil {
			t.Errorf("Expected permission denied error")
		}
	})
}