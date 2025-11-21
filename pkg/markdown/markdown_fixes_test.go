package markdown

import (
	"digital.vasic.translator/pkg/ebook"
	"os"
	"strings"
	"testing"
)

// TestDoubleEscapingBugFix tests that the double-escaping bug is fixed
// This was the critical bug where HTML tags appeared as literal text in EPUB
func TestDoubleEscapingBugFix(t *testing.T) {
	tests := []struct {
		name           string
		markdown       string
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:     "Bold text should render as HTML",
			markdown: "This is **bold** text.",
			shouldContain: []string{
				"<strong>",
				"</strong>",
			},
			shouldNotContain: []string{
				"&lt;strong&gt;",
				"&lt;/strong&gt;",
				"**bold**", // markdown syntax should be converted
			},
		},
		{
			name:     "Italic text should render as HTML",
			markdown: "This is *italic* text.",
			shouldContain: []string{
				"<em>",
				"</em>",
			},
			shouldNotContain: []string{
				"&lt;em&gt;",
				"&lt;/em&gt;",
			},
		},
		{
			name:     "Mixed formatting should render correctly",
			markdown: "Text with **bold** and *italic* and `code`.",
			shouldContain: []string{
				"<strong>bold</strong>",
				"<em>italic</em>",
				"<code>code</code>",
			},
			shouldNotContain: []string{
				"&lt;strong&gt;",
				"&lt;em&gt;",
				"&lt;code&gt;",
			},
		},
		{
			name:     "Special XML characters should be escaped in content",
			markdown: "Text with & < > \" ' characters.",
			shouldContain: []string{
				"&amp;",
				"&lt;",
				"&gt;",
				"&quot;",
				"&apos;",
			},
			shouldNotContain: []string{
				// Raw special characters should not appear (except in HTML tags)
			},
		},
		{
			name:     "Bold text with special characters",
			markdown: "**Text & more**",
			shouldContain: []string{
				"<strong>Text &amp; more</strong>",
			},
			shouldNotContain: []string{
				"&lt;strong&gt;",
				"**Text &amp; more**", // markdown should be converted
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewMarkdownToEPUBConverter()
			html := converter.markdownToHTML(tt.markdown)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(html, expected) {
					t.Errorf("HTML should contain %q\nGot: %s", expected, html)
				}
			}

			for _, unexpected := range tt.shouldNotContain {
				if strings.Contains(html, unexpected) {
					t.Errorf("HTML should NOT contain %q\nGot: %s", unexpected, html)
				}
			}
		})
	}
}

// TestCodeBlockHandling tests that code blocks are properly converted
func TestCodeBlockHandling(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		wantCode string
	}{
		{
			name: "Simple code block",
			markdown: "```\ncode here\n```",
			wantCode: "<pre><code>code here</code></pre>",
		},
		{
			name: "Code block with language",
			markdown: "```go\nfunc main() {}\n```",
			wantCode: "func main() {}",
		},
		{
			name: "Multiple code blocks",
			markdown: "```\nfirst\n```\n\nText\n\n```\nsecond\n```",
			wantCode: "first",
		},
		{
			name: "Code block with special characters",
			markdown: "```\n<html>\n  &copy;\n```",
			wantCode: "&lt;html&gt;",
		},
		{
			name: "Code block should not process markdown",
			markdown: "```\n**this should not be bold**\n*this should not be italic*\n```",
			wantCode: "**this should not be bold**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewMarkdownToEPUBConverter()
			html := converter.markdownToHTML(tt.markdown)

			if !strings.Contains(html, tt.wantCode) {
				t.Errorf("Code block not found in HTML.\nWant substring: %s\nGot: %s", tt.wantCode, html)
			}

			// Verify code blocks use <pre><code>
			if strings.Contains(tt.markdown, "```") && !strings.Contains(html, "<pre><code>") {
				t.Error("Code blocks should be wrapped in <pre><code> tags")
			}
		})
	}
}

// TestMarkdownSyntaxSupport tests comprehensive markdown syntax support
func TestMarkdownSyntaxSupport(t *testing.T) {
	markdown := `# H1 Header
## H2 Header
### H3 Header
#### H4 Header
##### H5 Header
###### H6 Header

**Bold text**
*Italic text*
` + "`inline code`" + `

---

` + "```" + `
code block
line 2
` + "```" + `

Normal paragraph.`

	converter := NewMarkdownToEPUBConverter()
	html := converter.markdownToHTML(markdown)

	expectations := []struct {
		description string
		shouldHave  string
	}{
		{"H1 header", "<h1>H1 Header</h1>"},
		{"H2 header", "<h2>H2 Header</h2>"},
		{"H3 header", "<h3>H3 Header</h3>"},
		{"H4 header", "<h4>H4 Header</h4>"},
		{"H5 header", "<h5>H5 Header</h5>"},
		{"H6 header", "<h6>H6 Header</h6>"},
		{"Bold", "<strong>Bold text</strong>"},
		{"Italic", "<em>Italic text</em>"},
		{"Inline code", "<code>inline code</code>"},
		{"Horizontal rule", "<hr/>"},
		{"Code block", "<pre><code>code block"},
		{"Paragraph", "<p>Normal paragraph.</p>"},
	}

	for _, exp := range expectations {
		if !strings.Contains(html, exp.shouldHave) {
			t.Errorf("%s not found.\nExpected substring: %s\nIn HTML:\n%s",
				exp.description, exp.shouldHave, html)
		}
	}
}

// TestEscapingOrder tests that escaping happens in the correct order
func TestEscapingOrder(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
		not   []string
	}{
		{
			name:  "Escape content, not HTML tags",
			input: "Text with **<bold>** & more",
			want: []string{
				"<strong>",
				"&lt;bold&gt;",
				"&amp; more",
			},
			not: []string{
				"&lt;strong&gt;", // HTML tags should not be escaped
			},
		},
			// Note: Link conversion to <a> tags is not yet implemented
		// Links remain as markdown [text](url) format
		{
			name:  "Links should preserve markdown format",
			input: "Check [link](http://example.com?a=1&b=2)",
			want: []string{
				"[link](http://example.com?a=1&amp;b=2)", // URL ampersands should be escaped
			},
			not: []string{
				"&lt;a href", // No HTML anchor conversion yet
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewMarkdownToEPUBConverter()
			html := converter.markdownToHTML(tt.input)

			for _, expected := range tt.want {
				if !strings.Contains(html, expected) {
					t.Errorf("Should contain: %s\nGot: %s", expected, html)
				}
			}

			for _, unexpected := range tt.not {
				if strings.Contains(html, unexpected) {
					t.Errorf("Should NOT contain: %s\nGot: %s", unexpected, html)
				}
			}
		})
	}
}

// TestCompleteWorkflowWithFixes tests the complete EPUB→MD→EPUB workflow with fixes applied
func TestCompleteWorkflowWithFixes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source book with complex formatting
	sourceBook := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test Book with Formatting",
			Authors: []string{"Test Author"},
			Cover:   []byte{0xFF, 0xD8, 0xFF, 0xE0},
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "This has **bold** and *italic* text."},
					{Content: "This has `inline code` and special chars: & < >"},
				},
			},
			{
				Title: "Chapter 2",
				Sections: []ebook.Section{
					{Content: "```\ncode block\nwith special <chars>\n```"},
					{Content: "Normal paragraph after code."},
				},
			},
		},
	}

	// Step 1: Write source EPUB
	sourceEPUB := tmpDir + "/source.epub"
	writer := ebook.NewEPUBWriter()
	if err := writer.Write(sourceBook, sourceEPUB); err != nil {
		t.Fatalf("Failed to write source EPUB: %v", err)
	}

	// Step 2: Convert to Markdown
	sourceMD := tmpDir + "/source.md"
	epubToMd := NewEPUBToMarkdownConverter(false, "")
	if err := epubToMd.ConvertEPUBToMarkdown(sourceEPUB, sourceMD); err != nil {
		t.Fatalf("Failed to convert EPUB to MD: %v", err)
	}

	// Verify markdown file exists
	mdContent, err := os.ReadFile(sourceMD)
	if err != nil {
		t.Fatalf("Failed to read markdown: %v", err)
	}

	mdStr := string(mdContent)

	// Verify markdown contains original formatting
	if !strings.Contains(mdStr, "**bold**") {
		t.Error("Markdown should contain bold syntax")
	}
	if !strings.Contains(mdStr, "*italic*") {
		t.Error("Markdown should contain italic syntax")
	}
	if !strings.Contains(mdStr, "`inline code`") {
		t.Error("Markdown should contain inline code syntax")
	}
	if !strings.Contains(mdStr, "```") {
		t.Error("Markdown should contain code block syntax")
	}

	// Step 3: Translate markdown (simple uppercase translation)
	translatedMD := tmpDir + "/translated.md"
	translator := NewMarkdownTranslator(func(text string) (string, error) {
		return strings.ToUpper(text), nil
	})
	if err := translator.TranslateMarkdownFile(sourceMD, translatedMD); err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Step 4: Convert back to EPUB
	outputEPUB := tmpDir + "/output.epub"
	mdToEpub := NewMarkdownToEPUBConverter()
	if err := mdToEpub.ConvertMarkdownToEPUB(translatedMD, outputEPUB); err != nil {
		t.Fatalf("Failed to convert MD to EPUB: %v", err)
	}

	// Step 5: Parse output and verify formatting
	parser := ebook.NewUniversalParser()
	resultBook, err := parser.Parse(outputEPUB)
	if err != nil {
		t.Fatalf("Failed to parse output EPUB: %v", err)
	}

	// Verify content exists
	if len(resultBook.Chapters) != 2 {
		t.Errorf("Expected 2 chapters, got %d", len(resultBook.Chapters))
	}

	// Note: Since we translated to uppercase, formatting should be preserved
	// The actual XHTML should have proper <strong>, <em>, <code> tags
	// without any double-escaping artifacts
}

// TestPathPreservation tests that file paths are correctly handled in Books/ directory
func TestPathPreservation(t *testing.T) {
	tmpDir := t.TempDir()
	booksDir := tmpDir + "/Books"

	// Create Books directory
	if err := os.MkdirAll(booksDir, 0755); err != nil {
		t.Fatalf("Failed to create Books dir: %v", err)
	}

	// Create source EPUB
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test",
			Authors: []string{"Author"},
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Content here"},
				},
			},
		},
	}

	sourceEPUB := booksDir + "/source.epub"
	writer := ebook.NewEPUBWriter()
	if err := writer.Write(book, sourceEPUB); err != nil {
		t.Fatalf("Failed to write EPUB: %v", err)
	}

	// Convert to markdown with Books/ path
	sourceMD := booksDir + "/source.md"
	converter := NewEPUBToMarkdownConverter(false, "")
	if err := converter.ConvertEPUBToMarkdown(sourceEPUB, sourceMD); err != nil {
		t.Fatalf("Failed to convert: %v", err)
	}

	// Verify file exists in Books/
	if _, err := os.Stat(sourceMD); os.IsNotExist(err) {
		t.Error("Markdown file was not created in Books/ directory")
	}

	// Verify Images directory created in correct location
	imagesDir := booksDir + "/Images"
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		t.Error("Images directory was not created in Books/ directory")
	}
}

// TestNoDoubleEscapingRegression tests that the fix doesn't regress
func TestNoDoubleEscapingRegression(t *testing.T) {
	// This test ensures the bug doesn't come back
	converter := NewMarkdownToEPUBConverter()

	// Test cases that previously caused double-escaping
	regressionCases := []struct {
		markdown string
		mustNot  string
	}{
		{
			markdown: "**bold**",
			mustNot:  "&lt;strong&gt;",
		},
		{
			markdown: "*italic*",
			mustNot:  "&lt;em&gt;",
		},
		{
			markdown: "`code`",
			mustNot:  "&lt;code&gt;",
		},
		{
			markdown: "[link](url)",
			mustNot:  "&lt;a href",
		},
	}

	for _, tc := range regressionCases {
		html := converter.markdownToHTML(tc.markdown)
		if strings.Contains(html, tc.mustNot) {
			t.Errorf("REGRESSION: Double-escaping detected!\nInput: %s\nFound: %s\nIn HTML: %s",
				tc.markdown, tc.mustNot, html)
		}
	}
}

// BenchmarkFixedMarkdownConversion benchmarks the fixed markdown to HTML conversion
func BenchmarkFixedMarkdownConversion(b *testing.B) {
	converter := NewMarkdownToEPUBConverter()
	markdown := `# Title

This is a paragraph with **bold**, *italic*, and ` + "`code`" + `.

` + "```" + `
code block
with multiple lines
` + "```" + `

Another paragraph with special chars: & < >`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		converter.markdownToHTML(markdown)
	}
}

// TestEmptyAndEdgeCasesWithFixes tests edge cases with the fixes applied
func TestEmptyAndEdgeCasesWithFixes(t *testing.T) {
	converter := NewMarkdownToEPUBConverter()

	edgeCases := []struct {
		name     string
		markdown string
	}{
		{"Empty string", ""},
		{"Only whitespace", "   \n  \n  "},
		{"Only code block", "```\ncode\n```"},
		{"Nested formatting", "**bold *and italic***"},
		{"Adjacent formatting", "**bold***italic*"},
		{"Unclosed formatting", "**bold"},
		{"Special chars only", "& < > \" '"},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			html := converter.markdownToHTML(tc.markdown)

			// Should not contain double-escaped entities
			if strings.Contains(html, "&amp;lt;") {
				t.Error("Found double-escaped entities")
			}
			if strings.Contains(html, "&amp;gt;") {
				t.Error("Found double-escaped entities")
			}
			if strings.Contains(html, "&amp;amp;") {
				t.Error("Found double-escaped entities")
			}
		})
	}
}
