package markdown

import (
	"context"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/translator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTranslator implements a mock translator for testing
type MockTranslator struct {
	mock.Mock
}

func (m *MockTranslator) Translate(ctx context.Context, text string, sourceLang string, targetLang string) (string, error) {
	args := m.Called(ctx, text, sourceLang, targetLang)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) GetProvider() string {
	args := m.Called()
	return args.String(0)
}

// TestMarkdownTranslatorBasicFunctionality tests basic Markdown translation
func TestMarkdownTranslatorBasicFunctionality(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	// Create Markdown translator
	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test basic Markdown translation
	input := "# Hello World\n" +
		"This is a paragraph with **bold text** and *italic text*.\n" +
		"## Subsection\n" +
		"- List item 1\n" +
		"- List item 2\n" +
		"> This is a quote\n" +
		"`This is code`\n"

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify translation preserves Markdown structure
	assert.Contains(t, result, "#") // Headers preserved
	assert.Contains(t, result, "**") // Bold preserved
	assert.Contains(t, result, "*") // Italic preserved
	assert.Contains(t, result, "-") // Lists preserved
	assert.Contains(t, result, ">") // Quotes preserved
	assert.Contains(t, result, "`") // Code preserved

	// Verify mock was called
	mockTrans.AssertExpectations(t)
}

// TestMarkdownTranslatorCodeBlocks tests handling of code blocks
func TestMarkdownTranslatorCodeBlocks(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with code blocks
	input := "# Code Test\n" +
		"Here is some inline code: `" + "`console.log('Hello')`" + "`\n" +
		"\n" +
		"Here is a code block:\n" +
		"```javascript\n" +
		"function greet() {\n" +
		"    console.log('Hello world');\n" +
		"}\n" +
		"```\n"

More text after code block.`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify code blocks are preserved (not translated)
	assert.Contains(t, result, "`console.log('Hello')`") // Inline code preserved
	assert.Contains(t, result, "```javascript") // Code block marker preserved
	assert.Contains(t, result, "function greet()") // Code content preserved
}

// TestMarkdownTranslatorTables tests handling of Markdown tables
func TestMarkdownTranslatorTables(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with tables
	input := `# Table Test
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Value 1  | Value 2  | Value 3  |
| Value 4  | Value 5  | Value 6  |
| Value 7  | Value 8  | Value 9  |

Text after table.`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify table structure is preserved
	assert.Contains(t, result, "|") // Table structure preserved
	assert.Contains(t, result, "---") // Table separators preserved
	assert.Contains(t, result, "Column") // Headers preserved
}

// TestMarkdownTranslatorLinks tests handling of Markdown links
func TestMarkdownTranslatorLinks(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with various link types
	input := `# Links Test
Here is a [regular link](https://example.com).
Here is a [link with title](https://example.com "Link Title").
Here is a [reference link][ref].
Here is an <https://example.com/auto-link>.
Here is an email: test@example.com
Here is an image: ![Alt text](image.jpg "Image title")

[ref]: https://example.com/reference`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify links are preserved
	assert.Contains(t, result, "[regular link](https://example.com)") // Regular link preserved
	assert.Contains(t, result, "[link with title](https://example.com \"Link Title\")") // Link with title preserved
	assert.Contains(t, result, "[reference link][ref]") // Reference link preserved
	assert.Contains(t, result, "<https://example.com/auto-link>") // Auto link preserved
	assert.Contains(t, result, "test@example.com") // Email preserved
	assert.Contains(t, result, "![Alt text](image.jpg \"Image title\")") // Image link preserved
	assert.Contains(t, result, "[ref]: https://example.com/reference") // Reference definition preserved
}

// TestMarkdownTranslatorHeaders tests handling of headers
func TestMarkdownTranslatorHeaders(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, mock.AnythingOfType("string"), "en", "ru").Return("Переведено", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with all header levels
	input := `# H1 Header
## H2 Header  
### H3 Header
#### H4 Header
##### H5 Header
###### H6 Header
Some content here.`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify all header levels are preserved
	assert.Contains(t, result, "# ") // H1 preserved
	assert.Contains(t, result, "## ") // H2 preserved
	assert.Contains(t, result, "### ") // H3 preserved
	assert.Contains(t, result, "#### ") // H4 preserved
	assert.Contains(t, result, "##### ") // H5 preserved
	assert.Contains(t, result, "###### ") // H6 preserved
}

// TestMarkdownTranslatorInlineFormatting tests inline formatting
func TestMarkdownTranslatorInlineFormatting(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with various inline formatting
	input := `# Inline Formatting Test
This text has **bold** and *italic* and ***bold italic*** formatting.
This has ~~strikethrough~~ text.
This has <ins>underline</ins> text.
This has ` + "`" + `monospace` + "`" + ` text.
This has superscript^2^ and subscript~2~ text.`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify inline formatting is preserved
	assert.Contains(t, result, "**") // Bold preserved
	assert.Contains(t, result, "*") // Italic preserved
	assert.Contains(t, result, "***") // Bold italic preserved
	assert.Contains(t, result, "~~") // Strikethrough preserved
	assert.Contains(t, result, "`") // Monospace preserved
}

// TestMarkdownTranslatorLists tests handling of lists
func TestMarkdownTranslatorLists(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with various list types
	input := `# Lists Test
## Unordered List
- Item 1
- Item 2
  - Nested item 2.1
  - Nested item 2.2
- Item 3

## Ordered List
1. First item
2. Second item
   1. Nested item 2.1
   2. Nested item 2.2
3. Third item

## Mixed List
- Mixed item 1
2. Mixed item 2
- Mixed item 3`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify list structures are preserved
	assert.Contains(t, result, "- ") // Unordered list preserved
	assert.Contains(t, result, "1.") // Ordered list preserved
	assert.Contains(t, result, "  - ") // Nested unordered list preserved
	assert.Contains(t, result, "   1.") // Nested ordered list preserved
}

// TestMarkdownTranslatorBlockquotes tests handling of blockquotes
func TestMarkdownTranslatorBlockquotes(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("Привет мир", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with blockquotes
	input := `# Blockquote Test
> This is a simple blockquote.
> This is the second line of the same blockquote.

> This is a nested blockquote.
>> This is a doubly nested blockquote.

> This blockquote contains **bold text** and *italic text*.
> It also contains ` + "`" + `inline code` + "`" + `.

Regular text after blockquote.`

	result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// Verify blockquote structure is preserved
	assert.Contains(t, result, "> ") // Basic blockquote preserved
	assert.Contains(t, result, ">> ") // Nested blockquote preserved
}

// TestMarkdownTranslatorPerformance tests translation performance
func TestMarkdownTranslatorPerformance(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, mock.AnythingOfType("string"), "en", "ru").Return("Переведено", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate large Markdown content
	var largeContent strings.Builder
	for i := 0; i < 1000; i++ {
		largeContent.WriteString(fmt.Sprintf("## Section %d\n", i))
		largeContent.WriteString(fmt.Sprintf("This is paragraph %d with **bold** and *italic* text.\n", i))
		largeContent.WriteString(fmt.Sprintf("- List item %d-1\n- List item %d-2\n", i, i))
	}

	start := time.Now()
	result, err := mdTrans.TranslateMarkdown(ctx, largeContent.String(), "en", "ru")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Less(t, duration, 10*time.Second, "Translation should complete within 10 seconds")
	
	t.Logf("Translated %d characters in %v", len(result), duration)
}

// TestMarkdownTranslatorErrorHandling tests error scenarios
func TestMarkdownTranslatorErrorHandling(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("Translation error", func(t *testing.T) {
		mockTrans.On("Translate", mock.Anything, "Hello world", "en", "ru").Return("", assert.AnError)
		mockTrans.On("GetProvider").Return("mock_provider")

		input := `# Test
Hello world`

		result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("Empty input", func(t *testing.T) {
		input := ""

		result, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		input := `# Test
Hello world`

		result, err := mdTrans.TranslateMarkdown(cancelledCtx, input, "en", "ru")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
		assert.Empty(t, result)
	})
}

// TestMarkdownTranslatorBatch tests batch translation functionality
func TestMarkdownTranslatorBatch(t *testing.T) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, mock.AnythingOfType("string"), "en", "ru").Return("Переведено", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test batch translation
	inputs := []string{
		"# Document 1\nThis is the first document.",
		"# Document 2\nThis is the second document.",
		"# Document 3\nThis is the third document.",
	}

	results, err := mdTrans.TranslateMarkdownBatch(ctx, inputs, "en", "ru")
	require.NoError(t, err)
	assert.Len(t, results, len(inputs))

	for i, result := range results {
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "#") // Headers preserved
	}
}

// BenchmarkMarkdownTranslatorTranslation benchmarks translation performance
func BenchmarkMarkdownTranslatorTranslation(b *testing.B) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, mock.AnythingOfType("string"), "en", "ru").Return("Переведено", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := `# Test Document
This is a test paragraph with **bold** and *italic* text.
## Section
- List item 1
- List item 2
> This is a quote
\`inline code\``

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := mdTrans.TranslateMarkdown(ctx, input, "en", "ru")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarkdownTranslatorLargeContent benchmarks large content translation
func BenchmarkMarkdownTranslatorLargeContent(b *testing.B) {
	// Create mock translator
	mockTrans := new(MockTranslator)
	mockTrans.On("Translate", mock.Anything, mock.AnythingOfType("string"), "en", "ru").Return("Переведено", nil)
	mockTrans.On("GetProvider").Return("mock_provider")

	mdTrans := markdown.NewMarkdownTranslator(markdown.Config{
		Translator: mockTrans,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Generate large content
	var largeContent strings.Builder
	for i := 0; i < 100; i++ {
		largeContent.WriteString(fmt.Sprintf("## Section %d\nParagraph %d with **bold** text.\n", i, i))
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := mdTrans.TranslateMarkdown(ctx, largeContent.String(), "en", "ru")
		if err != nil {
			b.Fatal(err)
		}
	}
}