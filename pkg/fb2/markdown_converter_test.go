package fb2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/logger"
)

func TestMarkdownConverter(t *testing.T) {
	// Create test logger
	testLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.DEBUG,
		Format: logger.FORMAT_TEXT,
	})

	// Create converter
	converter := NewMarkdownConverter(testLogger)

	// Test data - minimal FB2 structure
	testFB2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
	<description>
		<title-info>
			<genre>detective</genre>
			<author>
				<first-name>Test</first-name>
				<last-name>Author</last-name>
			</author>
			<book-title>Test Book</book-title>
			<annotation>
				<p>This is a test book for translation.</p>
			</annotation>
			<lang>ru</lang>
		</title-info>
	</description>
	<body>
		<section>
			<title>
				<p>Chapter 1</p>
			</title>
			<p>Это тестовый текст на русском языке.</p>
			<p>Он содержит различные элементы для проверки конвертации.</p>
		</section>
		<section>
			<title>
				<p>Chapter 2</p>
			</title>
			<epigraph>
				<p>Это эпиграф для тестирования.</p>
				<text-author>Test Author</text-author>
			</epigraph>
			<p>Продолжение тестового текста.</p>
			<poem>
				<title>
					<p>Test Poem</p>
				</title>
				<stanza>
					<v>Строка 1</v>
					<v>Строка 2</v>
					<v>Строка 3</v>
				</stanza>
			</poem>
		</section>
	</body>
</FictionBook>`

	// Create temporary files
	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "test.fb2")
	outputFile := filepath.Join(tempDir, "test.md")

	// Write test FB2
	if err := os.WriteFile(inputFile, []byte(testFB2Content), 0644); err != nil {
		t.Fatalf("Failed to write test FB2 file: %v", err)
	}

	// Convert FB2 to Markdown
	if err := converter.ConvertToMarkdown(inputFile, outputFile); err != nil {
		t.Fatalf("ConvertToMarkdown failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output markdown file was not created")
	}

	// Read and verify markdown content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output markdown: %v", err)
	}

	markdown := string(content)

	// Basic checks
	if !strings.Contains(markdown, "# Test Book") {
		t.Error("Book title not found in markdown")
	}

	if !strings.Contains(markdown, "Test Author") {
		t.Error("Author name not found in markdown")
	}

	if !strings.Contains(markdown, "## Chapter 1") {
		t.Error("Chapter 1 title not found in markdown")
	}

	if !strings.Contains(markdown, "## Chapter 2") {
		t.Error("Chapter 2 title not found in markdown")
	}

	if !strings.Contains(markdown, "Это тестовый текст на русском языке.") {
		t.Error("First paragraph not found in markdown")
	}

	// Check for epigraph
	if !strings.Contains(markdown, "> Это эпиграф для тестирования.") {
		t.Error("Epigraph not properly formatted in markdown")
	}

	if !strings.Contains(markdown, "> \u2014 Test Author") {
		t.Error("Epigraph author not properly formatted in markdown")
	}

	// Check for poem
	if !strings.Contains(markdown, "### Test Poem") {
		t.Error("Poem title not found in markdown")
	}

	if !strings.Contains(markdown, "    Строка 1") {
		t.Error("Poem verse not properly indented in markdown")
	}
}

func TestFormatAuthorName(t *testing.T) {
	tests := []struct {
		name     string
		author   Author
		expected string
	}{
		{
			name: "Full name",
			author: Author{
				FirstName: "Иван",
				LastName:  "Иванов",
			},
			expected: "Иван Иванов",
		},
		{
			name: "Only first name",
			author: Author{
				FirstName: "Петр",
			},
			expected: "Петр",
		},
		{
			name: "Only last name",
			author: Author{
				LastName: "Сидоров",
			},
			expected: "Сидоров",
		},
		{
			name: "Nickname only",
			author: Author{
				Nickname: "Writer123",
			},
			expected: "Writer123",
		},
		{
			name: "Empty author",
			author:   Author{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAuthorName(tt.author)
			if result != tt.expected {
				t.Errorf("formatAuthorName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractTextFromParagraph(t *testing.T) {
	tests := []struct {
		name     string
		paragraph Paragraph
		expected string
	}{
		{
			name: "Simple text",
			paragraph: Paragraph{
				Text: "Простой текст",
			},
			expected: "Простой текст",
		},
		{
			name: "Empty paragraph",
			paragraph: Paragraph{
				Text: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromParagraph(tt.paragraph)
			if result != tt.expected {
				t.Errorf("extractTextFromParagraph() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Integration test for full conversion workflow
func TestFullConversionWorkflow(t *testing.T) {
	// Skip if no test file available
	testFile := "../../test_book_small.fb2"
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test FB2 file not available, skipping integration test")
		return
	}

	testLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	converter := NewMarkdownConverter(testLogger)

	// Create temporary output
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.md")

	// Convert the test file
	if err := converter.ConvertToMarkdown(testFile, outputFile); err != nil {
		t.Fatalf("ConvertToMarkdown failed on test file: %v", err)
	}

	// Verify output exists and has content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	if len(content) == 0 {
		t.Error("Output file is empty")
	}

	// Check for markdown headers
	markdown := string(content)
	if !strings.Contains(markdown, "#") {
		t.Error("No markdown headers found in output")
	}
}