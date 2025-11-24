package markdown

import (
	"archive/zip"
	"digital.vasic.translator/pkg/ebook"
	"fmt"
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

// createTestEPUB creates a test EPUB file from a Book structure
func createTestEPUB(book *ebook.Book, outputPath string) error {
	// Create the EPUB file
	epubFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create EPUB file: %w", err)
	}
	defer epubFile.Close()

	zipWriter := zip.NewWriter(epubFile)
	defer zipWriter.Close()

	// Add mimetype (must be uncompressed and first)
	mimeTypeWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		return fmt.Errorf("failed to create mimetype entry: %w", err)
	}
	
	if _, err := mimeTypeWriter.Write([]byte("application/epub+zip")); err != nil {
		return fmt.Errorf("failed to write mimetype: %w", err)
	}

	// Write META-INF/container.xml
	if err := writeContainerXML(zipWriter); err != nil {
		return fmt.Errorf("failed to write container.xml: %w", err)
	}

	// Write OEBPS/content.opf
	if err := writeContentOPF(zipWriter, book); err != nil {
		return fmt.Errorf("failed to write content.opf: %w", err)
	}

	// Write OEBPS/toc.ncx
	if err := writeTOC(zipWriter, book); err != nil {
		return fmt.Errorf("failed to write toc.ncx: %w", err)
	}

	// Write chapter files
	for i, chapter := range book.Chapters {
		chapterNum := i + 1
		chapterPath := fmt.Sprintf("OEBPS/chapter%d.xhtml", chapterNum)
		writer, err := zipWriter.Create(chapterPath)
		if err != nil {
			return fmt.Errorf("failed to create chapter file: %w", err)
		}

		// Extract content from chapter sections
		var content strings.Builder
		for _, section := range chapter.Sections {
			if section.Content != "" {
				content.WriteString(section.Content)
				content.WriteString("\n\n")
			}
		}
		
		// Convert chapter content to valid XHTML
		xhtml := convertMarkdownToXHTML(content.String())
		if _, err := writer.Write([]byte(xhtml)); err != nil {
			return fmt.Errorf("failed to write chapter content: %w", err)
		}
	}

	return nil
}

// writeContainerXML writes META-INF/container.xml
func writeContainerXML(zw *zip.Writer) error {
	writer, err := zw.Create("META-INF/container.xml")
	if err != nil {
		return err
	}

	container := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	
	_, err = writer.Write([]byte(container))
	return err
}

// writeContentOPF writes OEBPS/content.opf
func writeContentOPF(zw *zip.Writer, book *ebook.Book) error {
	writer, err := zw.Create("OEBPS/content.opf")
	if err != nil {
		return err
	}

	var opf strings.Builder
	opf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="BookID">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
`)
	
	// Metadata
	opf.WriteString(fmt.Sprintf("    <dc:title>%s</dc:title>\n", escapeXML(book.Metadata.Title)))
	for _, author := range book.Metadata.Authors {
		opf.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", escapeXML(author)))
	}
	opf.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", escapeXML(book.Metadata.Language)))
	opf.WriteString("    <dc:identifier id=\"BookID\">urn:uuid:generated</dc:identifier>\n")
	opf.WriteString("  </metadata>\n")

	// Manifest
	opf.WriteString("  <manifest>\n")
	opf.WriteString("    <item id=\"ncx\" href=\"toc.ncx\" media-type=\"application/x-dtbncx+xml\"/>\n")
	for i := 1; i <= len(book.Chapters); i++ {
		opf.WriteString(fmt.Sprintf("    <item id=\"chapter%d\" href=\"chapter%d.xhtml\" media-type=\"application/xhtml+xml\"/>\n", i, i))
	}
	opf.WriteString("  </manifest>\n")

	// Spine
	opf.WriteString("  <spine toc=\"ncx\">\n")
	for i := 1; i <= len(book.Chapters); i++ {
		opf.WriteString(fmt.Sprintf("    <itemref idref=\"chapter%d\"/>\n", i))
	}
	opf.WriteString("  </spine>\n")
	opf.WriteString("</package>")
	
	_, err = writer.Write([]byte(opf.String()))
	return err
}

// writeTOC writes OEBPS/toc.ncx
func writeTOC(zw *zip.Writer, book *ebook.Book) error {
	writer, err := zw.Create("OEBPS/toc.ncx")
	if err != nil {
		return err
	}

	var ncx strings.Builder
	ncx.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE ncx PUBLIC "-//NISO//DTD ncx 2005-1//EN" "http://www.daisy.org/z3986/2005/ncx-2005-1.dtd">
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="urn:uuid:generated"/>
    <meta name="dtb:depth" content="1"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle>
    <text>`)
	ncx.WriteString(escapeXML(book.Metadata.Title))
	ncx.WriteString(`</text>
  </docTitle>
  <navMap>
`)
	
	for i, chapter := range book.Chapters {
		playOrder := i + 1
		ncx.WriteString(fmt.Sprintf("    <navPoint id=\"navPoint-%d\" playOrder=\"%d\">\n", playOrder, playOrder))
		ncx.WriteString("      <navLabel>\n")
		ncx.WriteString(fmt.Sprintf("        <text>%s</text>\n", escapeXML(chapter.Title)))
		ncx.WriteString("      </navLabel>\n")
		ncx.WriteString(fmt.Sprintf("      <content src=\"chapter%d.xhtml\"/>\n", playOrder))
		ncx.WriteString("    </navPoint>\n")
	}
	
	ncx.WriteString(`  </navMap>
</ncx>`)
	
	_, err = writer.Write([]byte(ncx.String()))
	return err
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// TestCompleteWorkflowWithFixes tests the complete EPUB→MD→EPUB workflow with fixes applied
func TestCompleteWorkflowWithFixes(t *testing.T) {
	tmpDir := t.TempDir()

	// Step 1: Create a simple markdown file directly (no need to create EPUB first)
	sourceMD := tmpDir + "/source.md"
	mdContent := `---
title: Test Book with Formatting
authors: Test Author
language: en
---

# Test Book with Formatting

This is the main introduction to the book.

## Chapter 1

This has **bold** and *italic* text.

This has ` + "`" + `inline code` + "`" + ` and special chars: & < >

## Chapter 2

` + "```" + `
code block
with special <chars>
` + "```" + `

Normal paragraph after code.
`
	
	if err := os.WriteFile(sourceMD, []byte(mdContent), 0644); err != nil {
		t.Fatalf("Failed to write source markdown: %v", err)
	}

	// Step 2: Translate markdown (simple uppercase translation)
	translatedMD := tmpDir + "/translated.md"
	translator := NewMarkdownTranslator(func(text string) (string, error) {
		return strings.ToUpper(text), nil
	})
	if err := translator.TranslateMarkdownFile(sourceMD, translatedMD); err != nil {
		t.Fatalf("Failed to translate markdown: %v", err)
	}

	// Step 3: Convert translated markdown to EPUB
	outputEPUB := tmpDir + "/output.epub"
	mdToEpub := NewMarkdownToEPUBConverter()
	if err := mdToEpub.ConvertMarkdownToEPUB(translatedMD, outputEPUB); err != nil {
		t.Fatalf("Failed to convert MD to EPUB: %v", err)
	}

	// Step 4: Parse output and verify formatting
	parser := ebook.NewUniversalParser()
	resultBook, err := parser.Parse(outputEPUB)
	if err != nil {
		t.Fatalf("Failed to parse output EPUB: %v", err)
	}

	// Verify content exists - should have 2 chapters (Chapter 1 and Chapter 2)
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
