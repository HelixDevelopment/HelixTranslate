package markdown

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"digital.vasic.translator/pkg/ebook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEPUBToMarkdownConversion tests basic EPUB to Markdown conversion
func TestEPUBToMarkdownConversion(t *testing.T) {
	// Create a simple EPUB content for testing
	epubContent := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf">
	<metadata>
		<dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Test Book</dc:title>
		<dc:language>en</dc:language>
	</metadata>
	<manifest>
		<item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
	</manifest>
	<spine>
		<itemref idref="chapter1"/>
	</spine>
</package>`

	chapterContent := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
	<h1>Chapter 1</h1>
	<p>This is a test paragraph.</p>
	<p>This is another paragraph with <strong>bold text</strong>.</p>
	<ul>
		<li>Item 1</li>
		<li>Item 2</li>
		<li>Item 3</li>
	</ul>
</body>
</html>`

	// Test conversion
	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Mock EPUB reading
	epubReader := bytes.NewReader([]byte(epubContent))
	
	result, err := converter.Convert(ctx, epubReader)
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Verify Markdown content
	markdownContent := result.GetContent()
	assert.NotEmpty(t, markdownContent)
	assert.Contains(t, markdownContent, "# Chapter 1")
	assert.Contains(t, markdownContent, "This is a test paragraph")
	assert.Contains(t, markdownContent, "**bold text**")
	assert.Contains(t, markdownContent, "- Item 1")
}

// TestEPUBToMarkdownComplexContent tests conversion of complex EPUB content
func TestEPUBToMarkdownComplexContent(t *testing.T) {
	complexChapter := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Complex Chapter</title></head>
<body>
	<h1>Complex Chapter</h1>
	<h2>Section 1</h2>
	<p>Paragraph with <em>emphasis</em> and <strong>strong</strong> text.</p>
	<blockquote>
		<p>This is a blockquote with multiple lines.</p>
		<p>Second line of blockquote.</p>
	</blockquote>
	<h3>Subsection</h3>
	<ol>
		<li>First numbered item</li>
		<li>Second numbered item</li>
	</ol>
	<table>
		<tr><th>Header 1</th><th>Header 2</th></tr>
		<tr><td>Cell 1</td><td>Cell 2</td></tr>
		<tr><td>Cell 3</td><td>Cell 4</td></tr>
	</table>
	<p>Code example: <code>console.log('Hello')</code></p>
	<pre><code>function test() {
	return "Hello World";
}</code></pre>
</body>
</html>`

	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create test result with complex content
	testResult := &markdown.MarkdownResult{
		Content: converter.ParseHTMLToMarkdown(complexChapter),
		Metadata: map[string]interface{}{
			"title":    "Complex Chapter",
			"language": "en",
		},
		ConversionTime: time.Now(),
	}

	// Verify complex Markdown elements
	assert.Contains(t, testResult.Content, "## Section 1")
	assert.Contains(t, testResult.Content, "*emphasis*")
	assert.Contains(t, testResult.Content, "**strong**")
	assert.Contains(t, testResult.Content, "> This is a blockquote")
	assert.Contains(t, testResult.Content, "### Subsection")
	assert.Contains(t, testResult.Content, "1. First numbered item")
	assert.Contains(t, testResult.Content, "| Header 1 | Header 2 |")
	assert.Contains(t, testResult.Content, "`console.log('Hello')`")
	assert.Contains(t, testResult.Content, "```")
}

// TestEPUBToMarkdownImages tests handling of images in EPUB
func TestEPUBToMarkdownImages(t *testing.T) {
	chapterWithImages := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter with Images</title></head>
<body>
	<h1>Chapter with Images</h1>
	<p>Text before image.</p>
	<img src="image1.jpg" alt="First image" title="Image title"/>
	<p>Text between images.</p>
	<img src="image2.png" alt="Second image" width="300" height="200"/>
	<p>Text after images.</p>
</body>
</html>`

	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := converter.ConvertImages(chapterWithImages)
	
	// Verify image handling
	assert.Contains(t, result, "![First image](image1.jpg \"Image title\")")
	assert.Contains(t, result, "![Second image](image2.png)")
}

// TestEPUBToMarkdownLinks tests handling of links in EPUB
func TestEPUBToMarkdownLinks(t *testing.T) {
	chapterWithLinks := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter with Links</title></head>
<body>
	<h1>Chapter with Links</h1>
	<p>This is a <a href="https://example.com">external link</a>.</p>
	<p>This is an <a href="chapter2.html">internal link</a>.</p>
	<p>This is a <a href="mailto:test@example.com">email link</a>.</p>
	<p>This is a <a href="https://example.com" title="Link title">link with title</a>.</p>
</body>
</html>`

	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := converter.ConvertLinks(chapterWithLinks)
	
	// Verify link handling
	assert.Contains(t, result, "[external link](https://example.com)")
	assert.Contains(t, result, "[internal link](chapter2.html)")
	assert.Contains(t, result, "[email link](mailto:test@example.com)")
	assert.Contains(t, result, "[link with title](https://example.com \"Link title\")")
}

// TestEPUBToMarkdownMetadata tests metadata extraction
func TestEPUBToMarkdownMetadata(t *testing.T) {
	epubMetadata := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf">
	<metadata>
		<dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">Test Book Title</dc:title>
		<dc:creator xmlns:dc="http://purl.org/dc/elements/1.1/">Test Author</dc:creator>
		<dc:language xmlns:dc="http://purl.org/dc/elements/1.1/">en</dc:language>
		<dc:date xmlns:dc="http://purl.org/dc/elements/1.1/">2023-01-01</dc:date>
		<dc:publisher xmlns:dc="http://purl.org/dc/elements/1.1/">Test Publisher</dc:publisher>
		<dc:description xmlns:dc="http://purl.org/dc/elements/1.1/">Test description</dc:description>
		<meta name="cover" content="cover-image"/>
	</metadata>
	<manifest>
		<item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
		<item id="cover-image" href="cover.jpg" media-type="image/jpeg"/>
	</manifest>
	<spine>
		<itemref idref="chapter1"/>
	</spine>
</package>`

	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadata := converter.ExtractMetadata(epubMetadata)
	
	// Verify metadata extraction
	assert.Equal(t, "Test Book Title", metadata["title"])
	assert.Equal(t, "Test Author", metadata["creator"])
	assert.Equal(t, "en", metadata["language"])
	assert.Equal(t, "2023-01-01", metadata["date"])
	assert.Equal(t, "Test Publisher", metadata["publisher"])
	assert.Equal(t, "Test description", metadata["description"])
	assert.Equal(t, "cover-image", metadata["cover"])
}

// TestEPUBToMarkdownPerformance tests conversion performance
func TestEPUBToMarkdownPerformance(t *testing.T) {
	// Generate large EPUB content
	var largeChapter bytes.Buffer
	largeChapter.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Large Chapter</title></head>
<body>`)

	for i := 0; i < 1000; i++ {
		largeChapter.WriteString(fmt.Sprintf(`<h2>Section %d</h2>`, i))
		largeChapter.WriteString(fmt.Sprintf(`<p>This is paragraph %d with some content.</p>`, i))
		largeChapter.WriteString(`<ul>`)
		for j := 0; j < 10; j++ {
			largeChapter.WriteString(fmt.Sprintf(`<li>List item %d-%d</li>`, i, j))
		}
		largeChapter.WriteString(`</ul>`)
	}

	largeChapter.WriteString(`</body></html>`)

	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	result := converter.ParseHTMLToMarkdown(largeChapter.String())
	duration := time.Since(start)

	// Performance assertions
	assert.NotEmpty(t, result)
	assert.Less(t, duration, 5*time.Second, "Conversion should complete within 5 seconds")
	t.Logf("Converted %d characters in %v", len(result), duration)
}

// TestEPUBToMarkdownErrorHandling tests error scenarios
func TestEPUBToMarkdownErrorHandling(t *testing.T) {
	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("Invalid EPUB content", func(t *testing.T) {
		invalidContent := []byte("This is not valid EPUB content")
		reader := bytes.NewReader(invalidContent)
		
		result, err := converter.Convert(ctx, reader)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Empty EPUB content", func(t *testing.T) {
		emptyContent := []byte("")
		reader := bytes.NewReader(emptyContent)
		
		result, err := converter.Convert(ctx, reader)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Malformed XML content", func(t *testing.T) {
		malformedXML := `<?xml version="1.0" encoding="UTF-8"?>
<html>
<head><title>Malformed</title></head>
<body>
	<h1>Unclosed heading
	<p>Some content</p>
</body>
</html>`

		result := converter.ParseHTMLToMarkdown(malformedXML)
		// Should handle malformed XML gracefully
		assert.NotEmpty(t, result)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create a context that's already cancelled
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		invalidContent := []byte("Some content")
		reader := bytes.NewReader(invalidContent)
		
		result, err := converter.Convert(cancelledCtx, reader)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
		assert.Nil(t, result)
	})
}

// TestEPUBToMarkdownIntegration tests integration with ebook package
func TestEPUBToMarkdownIntegration(t *testing.T) {
	// Create a test EPUB file structure
	testEPUB := &ebook.EPUB{
		Metadata: ebook.Metadata{
			Title:    "Integration Test Book",
			Language: "en",
			Creator:  "Test Author",
		},
		Chapters: []*ebook.Chapter{
			{
				ID:    "chapter1",
				Title: "Test Chapter 1",
				Content: `<h1>Test Chapter 1</h1><p>This is test content.</p>`,
			},
			{
				ID:    "chapter2", 
				Title: "Test Chapter 2",
				Content: `<h1>Test Chapter 2</h1><p>More test content.</p>`,
			},
		},
	}

	converter := markdown.NewEPUBToMarkdownConverter()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Convert entire EPUB to Markdown
	result, err := converter.ConvertEPUB(ctx, testEPUB)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify conversion results
	assert.NotEmpty(t, result.Content)
	assert.Equal(t, "Integration Test Book", result.Metadata["title"])
	assert.Equal(t, "Test Author", result.Metadata["creator"])
	assert.Equal(t, "en", result.Metadata["language"])

	// Verify chapter content
	assert.Contains(t, result.Content, "# Test Chapter 1")
	assert.Contains(t, result.Content, "# Test Chapter 2")
	assert.Contains(t, result.Content, "This is test content")
	assert.Contains(t, result.Content, "More test content")
}

// BenchmarkEPUBToMarkdownConversion benchmarks conversion performance
func BenchmarkEPUBToMarkdownConversion(b *testing.B) {
	// Create test content
	chapterContent := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Benchmark Chapter</title></head>
<body>
	<h1>Benchmark Chapter</h1>
	<p>This is a benchmark paragraph with <strong>bold text</strong>.</p>
	<ul>
		<li>Item 1</li>
		<li>Item 2</li>
		<li>Item 3</li>
	</ul>
</body>
</html>`

	converter := markdown.NewEPUBToMarkdownConverter()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = converter.ParseHTMLToMarkdown(chapterContent)
	}
}

// BenchmarkEPUBToMarkdownLargeContent benchmarks large content conversion
func BenchmarkEPUBToMarkdownLargeContent(b *testing.B) {
	// Generate large test content
	var largeContent bytes.Buffer
	for i := 0; i < 100; i++ {
		largeContent.WriteString(fmt.Sprintf(`<h2>Section %d</h2><p>Paragraph content %d.</p>`, i, i))
	}

	converter := markdown.NewEPUBToMarkdownConverter()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = converter.ParseHTMLToMarkdown(largeContent.String())
	}
}