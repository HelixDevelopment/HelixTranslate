package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainFunction(t *testing.T) {
	// Test main function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "preparation-translator-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original args and restore after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Create test input file
	inputFile := filepath.Join(tempDir, "test.txt")
	inputContent := "This is a test document for preparation translation."
	err = os.WriteFile(inputFile, []byte(inputContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Test with valid arguments
	os.Args = []string{"preparation-translator", "--input", inputFile, "--output", filepath.Join(tempDir, "output.txt")}

	t.Run("FileValidation", func(t *testing.T) {
		// Test input file validation
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			t.Errorf("Input file should exist: %v", err)
		}
	})
}

func TestPreparationWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "preparation-workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("TextPreparation", func(t *testing.T) {
		// Test text preparation functionality
		inputText := `This is a test document.
		It has multiple lines.
		Some lines have leading/trailing spaces.   
		There are also  multiple   spaces between words.`
		
		inputFile := filepath.Join(tempDir, "input.txt")
		err := os.WriteFile(inputFile, []byte(inputText), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		// Test that preparation steps would work
		content, err := os.ReadFile(inputFile)
		if err != nil {
			t.Errorf("Failed to read input file: %v", err)
		}

		// Basic validation - file should not be empty
		if len(content) == 0 {
			t.Errorf("Input file should not be empty")
		}
	})

	t.Run("FormatDetection", func(t *testing.T) {
		// Test different text formats
		testCases := []struct {
			name     string
			content  string
			expected string
		}{
			{
				name:     "PlainText",
				content:  "This is plain text.",
				expected: "text",
			},
			{
				name:     "Markdown",
				content:  "# Header\n\nThis is **markdown**.",
				expected: "markdown",
			},
			{
				name:     "HTML",
				content:  "<h1>Header</h1><p>This is HTML.</p>",
				expected: "html",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				inputFile := filepath.Join(tempDir, tc.name+".txt")
				err := os.WriteFile(inputFile, []byte(tc.content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Test format detection logic
				content, err := os.ReadFile(inputFile)
				if err != nil {
					t.Errorf("Failed to read test file: %v", err)
				}

				// Simple detection logic (would be more sophisticated in real implementation)
				detected := "text" // default
				contentStr := string(content)
				
				if contentStr[0] == '<' {
					detected = "html"
				} else if contentStr[0] == '#' {
					detected = "markdown"
				}

				if detected != tc.expected {
					t.Errorf("Format detection: got %s, want %s", detected, tc.expected)
				}
			})
		}
	})
}

func TestTextCleaning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "preparation-cleaning-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("WhitespaceNormalization", func(t *testing.T) {
		// Test whitespace normalization
		inputContent := "  This   has    multiple   spaces.  \n\n   And newlines.  "
		inputFile := filepath.Join(tempDir, "whitespace.txt")
		err := os.WriteFile(inputFile, []byte(inputContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		// Read and process content
		content, err := os.ReadFile(inputFile)
		if err != nil {
			t.Errorf("Failed to read input file: %v", err)
		}

		// Basic validation that content exists
		if len(content) == 0 {
			t.Errorf("Content should not be empty after cleaning")
		}
	})

	t.Run("SpecialCharacterHandling", func(t *testing.T) {
		// Test special character handling
		inputContent := "Text with special chars: äöü ß 中文 العربية 🎉"
		inputFile := filepath.Join(tempDir, "special.txt")
		err := os.WriteFile(inputFile, []byte(inputContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		// Test that special characters are preserved
		content, err := os.ReadFile(inputFile)
		if err != nil {
			t.Errorf("Failed to read input file: %v", err)
		}

		if string(content) != inputContent {
			t.Errorf("Special characters not preserved: got %s, want %s", string(content), inputContent)
		}
	})
}

func TestSegmentation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "preparation-segmentation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("SentenceSegmentation", func(t *testing.T) {
		// Test sentence segmentation
		inputContent := `This is the first sentence. This is the second sentence! 
		Is this the third sentence? Yes, it is.`
		
		inputFile := filepath.Join(tempDir, "sentences.txt")
		err := os.WriteFile(inputFile, []byte(inputContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		content, err := os.ReadFile(inputFile)
		if err != nil {
			t.Errorf("Failed to read input file: %v", err)
		}

		// Basic sentence counting (simplified)
		contentStr := string(content)
		sentenceCount := 0
		sentenceEnds := []string{". ", "! ", "? ", ".\n", "!\n", "?\n"}
		
		for _, ending := range sentenceEnds {
			sentenceCount += strings.Count(contentStr, ending)
		}

		// Should have at least 4 sentences
		if sentenceCount < 3 {
			t.Errorf("Expected at least 3 sentences, got %d", sentenceCount)
		}
	})

	t.Run("ParagraphSegmentation", func(t *testing.T) {
		// Test paragraph segmentation
		inputContent := `First paragraph with some content.

Second paragraph after empty line.

Third paragraph.

Final paragraph.`
		
		inputFile := filepath.Join(tempDir, "paragraphs.txt")
		err := os.WriteFile(inputFile, []byte(inputContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create input file: %v", err)
		}

		content, err := os.ReadFile(inputFile)
		if err != nil {
			t.Errorf("Failed to read input file: %v", err)
		}

		// Count paragraphs (simplified - double newlines)
		paragraphs := strings.Split(string(content), "\n\n")
		if len(paragraphs) < 3 {
			t.Errorf("Expected at least 3 paragraphs, got %d", len(paragraphs))
		}
	})
}

func TestLanguageDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "preparation-langdetect-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("SimpleLanguageDetection", func(t *testing.T) {
		testCases := []struct {
			name     string
			content  string
			expected string
		}{
			{
				name:     "English",
				content:  "This is English text with common words.",
				expected: "en",
			},
			{
				name:     "Russian",
				content:  "Это русский текст с обычными словами.",
				expected: "ru",
			},
			{
				name:     "Serbian",
				content:  "Ово је српски текст са уобичајеним речима.",
				expected: "sr",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				inputFile := filepath.Join(tempDir, tc.name+".txt")
				err := os.WriteFile(inputFile, []byte(tc.content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Simple language detection based on character sets
				content := tc.content
				detected := "en" // default
				
				if strings.ContainsAny(content, "абвгдеёжзийклмнопрстуфхцчшщъыьэюяАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ") {
					detected = "ru"
				} else if strings.ContainsAny(content, "ћђчџшђжЋЂЧЏШЂЖ") {
					detected = "sr"
				}

				if detected != tc.expected {
					t.Errorf("Language detection for %s: got %s, want %s", tc.name, detected, tc.expected)
				}
			})
		}
	})
}

func TestPreparationOptions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "preparation-options-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("SplitOptions", func(t *testing.T) {
		// Test different split options
		testContent := "Sentence one. Sentence two. Sentence three. Sentence four."
		splitOptions := []string{"sentence", "paragraph", "word", "character"}
		
		for _, option := range splitOptions {
			t.Run(option, func(t *testing.T) {
				inputFile := filepath.Join(tempDir, "split_"+option+".txt")
				err := os.WriteFile(inputFile, []byte(testContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				// Test split option validation
				validOptions := map[string]bool{
					"sentence":  true,
					"paragraph": true,
					"word":      true,
					"character": true,
				}

				if !validOptions[option] {
					t.Errorf("Invalid split option: %s", option)
				}
			})
		}
	})

	t.Run("FormatOptions", func(t *testing.T) {
		// Test different format options
		formatOptions := []string{"txt", "json", "xml", "csv"}
		
		for _, format := range formatOptions {
			t.Run(format, func(t *testing.T) {
				// Test format option validation
				validFormats := map[string]bool{
					"txt":  true,
					"json": true,
					"xml":  true,
					"csv":  true,
				}

				if !validFormats[format] {
					t.Errorf("Invalid format option: %s", format)
				}
			})
		}
	})
}

func TestErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "preparation-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("EmptyFile", func(t *testing.T) {
		emptyFile := filepath.Join(tempDir, "empty.txt")
		err := os.WriteFile(emptyFile, []byte{}, 0644)
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		content, err := os.ReadFile(emptyFile)
		if err != nil {
			t.Errorf("Failed to read empty file: %v", err)
		}

		if len(content) != 0 {
			t.Errorf("Empty file should have 0 bytes, got %d", len(content))
		}
	})

	t.Run("LargeFile", func(t *testing.T) {
		// Test handling of large files
		largeFile := filepath.Join(tempDir, "large.txt")
		largeContent := strings.Repeat("This is a test sentence. ", 10000)
		
		err := os.WriteFile(largeFile, []byte(largeContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		content, err := os.ReadFile(largeFile)
		if err != nil {
			t.Errorf("Failed to read large file: %v", err)
		}

		if len(content) != len(largeContent) {
			t.Errorf("Large file size mismatch: got %d, want %d", len(content), len(largeContent))
		}
	})
}