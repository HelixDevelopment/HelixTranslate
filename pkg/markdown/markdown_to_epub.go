package markdown

import (
	"archive/zip"
	"bufio"
	"digital.vasic.translator/pkg/ebook"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MarkdownToEPUBConverter converts Markdown files to EPUB format
type MarkdownToEPUBConverter struct {
	metadata ebook.Metadata
}

// NewMarkdownToEPUBConverter creates a new converter
func NewMarkdownToEPUBConverter() *MarkdownToEPUBConverter {
	return &MarkdownToEPUBConverter{
		metadata: ebook.Metadata{},
	}
}

// ConvertMarkdownToEPUB converts a markdown file to EPUB
func (c *MarkdownToEPUBConverter) ConvertMarkdownToEPUB(mdPath, epubPath string) error {
	// Read markdown file
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown: %w", err)
	}

	// Parse markdown into chapters
	chapters, metadata, coverPath, err := c.parseMarkdown(string(content), filepath.Dir(mdPath))
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	// Load cover image if specified
	if coverPath != "" {
		coverData, err := os.ReadFile(coverPath)
		if err == nil {
			metadata.Cover = coverData
		}
	}

	c.metadata = metadata

	// Create EPUB
	if err := c.createEPUB(chapters, epubPath); err != nil {
		return fmt.Errorf("failed to create EPUB: %w", err)
	}

	return nil
}

// parseMarkdown parses markdown content into chapters
func (c *MarkdownToEPUBConverter) parseMarkdown(content string, mdDir string) ([]ebook.Chapter, ebook.Metadata, string, error) {
	var metadata ebook.Metadata
	var chapters []ebook.Chapter
	var currentChapter *ebook.Chapter
	var currentContent strings.Builder
	var coverPath string

	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontmatter := false
	frontmatterDone := false
	frontmatterCount := 0
	skipNextLines := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Handle frontmatter (only before it's done)
		if !frontmatterDone && line == "---" {
			frontmatterCount++
			if frontmatterCount == 1 {
				inFrontmatter = true
				continue
			} else if frontmatterCount >= 2 {
				inFrontmatter = false
				frontmatterDone = true
				// Skip next 5 lines (title, author, separator after frontmatter)
				skipNextLines = 5
				continue
			}
		}

		if inFrontmatter {
			// Parse metadata
			if cover := c.parseFrontmatterLine(line, &metadata); cover != "" {
				// Resolve cover path relative to markdown file
				coverPath = filepath.Join(mdDir, cover)
			}
			continue
		}

		// Skip lines after frontmatter (title, author, separator)
		if skipNextLines > 0 {
			skipNextLines--
			continue
		}

		// Chapter marker (# or ## followed by text)
		if (strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ")) &&
		   strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "##"), "#")) != "" {
			// Save previous chapter
			if currentChapter != nil {
				currentChapter.Sections = []ebook.Section{
					{Content: strings.TrimSpace(currentContent.String())},
				}
				chapters = append(chapters, *currentChapter)
				currentContent.Reset()
			}

			// Start new chapter
			chapterTitle := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "##"), "#"))
			currentChapter = &ebook.Chapter{
				Title:    chapterTitle,
				Sections: []ebook.Section{},
			}
			continue
		}

		// Horizontal rule (chapter separator) - also saves chapter
		if matched, _ := regexp.MatchString(`^[-*_]{3,}$`, strings.TrimSpace(line)); matched {
			if currentChapter != nil {
				currentChapter.Sections = []ebook.Section{
					{Content: strings.TrimSpace(currentContent.String())},
				}
				chapters = append(chapters, *currentChapter)
				currentChapter = nil
				currentContent.Reset()
			}
			continue
		}

		// Add content to current chapter
		if currentChapter != nil {
			currentContent.WriteString(line + "\n")
		}
	}

	// Save last chapter
	if currentChapter != nil {
		currentChapter.Sections = []ebook.Section{
			{Content: strings.TrimSpace(currentContent.String())},
		}
		chapters = append(chapters, *currentChapter)
	}

	if err := scanner.Err(); err != nil {
		return nil, metadata, "", fmt.Errorf("error reading markdown: %w", err)
	}

	return chapters, metadata, coverPath, nil
}

// parseFrontmatterLine parses a frontmatter YAML line and returns cover path if present
func (c *MarkdownToEPUBConverter) parseFrontmatterLine(line string, metadata *ebook.Metadata) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	switch key {
	case "title":
		metadata.Title = value
	case "authors":
		metadata.Authors = strings.Split(value, ",")
		for i := range metadata.Authors {
			metadata.Authors[i] = strings.TrimSpace(metadata.Authors[i])
		}
	case "description":
		metadata.Description = value
	case "publisher":
		metadata.Publisher = value
	case "language":
		metadata.Language = value
	case "isbn":
		metadata.ISBN = value
	case "date":
		metadata.Date = value
	case "cover":
		// Return the cover path for loading the cover image
		return value
	case "has_cover":
		// Cover presence is tracked but binary data is preserved separately
		// This flag just indicates the original had a cover
	}
	return ""
}

// createEPUB creates an EPUB file from chapters using the enhanced EPUBWriter
func (c *MarkdownToEPUBConverter) createEPUB(chapters []ebook.Chapter, outputPath string) error {
	// Create Book structure with all metadata
	book := &ebook.Book{
		Metadata: c.metadata,
		Chapters: chapters,
	}

	// Use the enhanced EPUBWriter which handles all metadata properly
	writer := ebook.NewEPUBWriter()
	return writer.Write(book, outputPath)
}

// writeContainer writes META-INF/container.xml
func (c *MarkdownToEPUBConverter) writeContainer(zw *zip.Writer) error {
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
func (c *MarkdownToEPUBConverter) writeContentOPF(zw *zip.Writer, chapters []ebook.Chapter) error {
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
	opf.WriteString(fmt.Sprintf("    <dc:title>%s</dc:title>\n", c.escapeXML(c.metadata.Title)))
	for _, author := range c.metadata.Authors {
		opf.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", c.escapeXML(author)))
	}
	opf.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", c.metadata.Language))
	opf.WriteString("    <dc:identifier id=\"BookID\">urn:uuid:generated</dc:identifier>\n")
	opf.WriteString("  </metadata>\n")

	// Manifest
	opf.WriteString("  <manifest>\n")
	opf.WriteString("    <item id=\"ncx\" href=\"toc.ncx\" media-type=\"application/x-dtbncx+xml\"/>\n")
	for i := 1; i <= len(chapters); i++ {
		opf.WriteString(fmt.Sprintf("    <item id=\"chapter%d\" href=\"chapter%d.xhtml\" media-type=\"application/xhtml+xml\"/>\n", i, i))
	}
	opf.WriteString("  </manifest>\n")

	// Spine
	opf.WriteString("  <spine toc=\"ncx\">\n")
	for i := 1; i <= len(chapters); i++ {
		opf.WriteString(fmt.Sprintf("    <itemref idref=\"chapter%d\"/>\n", i))
	}
	opf.WriteString("  </spine>\n")
	opf.WriteString("</package>")

	_, err = writer.Write([]byte(opf.String()))
	return err
}

// writeTOC writes OEBPS/toc.ncx
func (c *MarkdownToEPUBConverter) writeTOC(zw *zip.Writer, chapters []ebook.Chapter) error {
	writer, err := zw.Create("OEBPS/toc.ncx")
	if err != nil {
		return err
	}

	var toc strings.Builder
	toc.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="urn:uuid:generated"/>
    <meta name="dtb:depth" content="1"/>
  </head>
  <docTitle>
`)
	toc.WriteString(fmt.Sprintf("    <text>%s</text>\n", c.escapeXML(c.metadata.Title)))
	toc.WriteString("  </docTitle>\n  <navMap>\n")

	for idx, chapter := range chapters {
		toc.WriteString(fmt.Sprintf("    <navPoint id=\"chapter%d\" playOrder=\"%d\">\n", idx+1, idx+1))
		toc.WriteString(fmt.Sprintf("      <navLabel><text>%s</text></navLabel>\n", c.escapeXML(chapter.Title)))
		toc.WriteString(fmt.Sprintf("      <content src=\"chapter%d.xhtml\"/>\n", idx+1))
		toc.WriteString("    </navPoint>\n")
	}

	toc.WriteString("  </navMap>\n</ncx>")

	_, err = writer.Write([]byte(toc.String()))
	return err
}

// writeChapterHTML writes a chapter as XHTML
func (c *MarkdownToEPUBConverter) writeChapterHTML(zw *zip.Writer, chapter ebook.Chapter, chapterNum int) error {
	writer, err := zw.Create(fmt.Sprintf("OEBPS/chapter%d.xhtml", chapterNum))
	if err != nil {
		return err
	}

	var html strings.Builder
	html.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
`)
	html.WriteString(fmt.Sprintf("  <title>%s</title>\n", c.escapeXML(chapter.Title)))
	html.WriteString("  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\"/>\n")
	html.WriteString("</head>\n<body>\n")
	html.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", c.escapeXML(chapter.Title)))

	// Convert markdown content to HTML
	for _, section := range chapter.Sections {
		htmlContent := c.markdownToHTML(section.Content)
		html.WriteString(htmlContent)
	}

	html.WriteString("</body>\n</html>")

	_, err = writer.Write([]byte(html.String()))
	return err
}

// markdownToHTML converts markdown content to HTML
func (c *MarkdownToEPUBConverter) markdownToHTML(markdown string) string {
	var html strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(markdown))

	inParagraph := false
	inCodeBlock := false
	var currentParagraph strings.Builder
	var codeBlock strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Code block delimiter
		if strings.HasPrefix(trimmed, "```") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}

			if inCodeBlock {
				// End code block
				html.WriteString("  <pre><code>" + c.escapeXML(codeBlock.String()) + "</code></pre>\n")
				codeBlock.Reset()
				inCodeBlock = false
			} else {
				// Start code block
				inCodeBlock = true
			}
			continue
		}

		// Inside code block
		if inCodeBlock {
			if codeBlock.Len() > 0 {
				codeBlock.WriteString("\n")
			}
			codeBlock.WriteString(line)
			continue
		}

		// Empty line ends paragraph
		if trimmed == "" {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			continue
		}

		// Horizontal rule
		if matched, _ := regexp.MatchString(`^[-*_]{3,}$`, trimmed); matched {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			html.WriteString("  <hr/>\n")
			continue
		}

		// Headers (h1 through h6)
		if strings.HasPrefix(trimmed, "######") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "######"))
			html.WriteString(fmt.Sprintf("  <h6>%s</h6>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "#####") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "#####"))
			html.WriteString(fmt.Sprintf("  <h5>%s</h5>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "####") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "####"))
			html.WriteString(fmt.Sprintf("  <h4>%s</h4>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "###") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			html.WriteString(fmt.Sprintf("  <h3>%s</h3>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "##") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "##"))
			html.WriteString(fmt.Sprintf("  <h2>%s</h2>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "#") && len(trimmed) > 1 && trimmed[1] == ' ' {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			html.WriteString(fmt.Sprintf("  <h1>%s</h1>\n", c.escapeXML(text)))
			continue
		}

		// Regular paragraph content
		if !inParagraph {
			inParagraph = true
		} else {
			currentParagraph.WriteString(" ")
		}
		currentParagraph.WriteString(trimmed)
	}

	// Close last paragraph
	if inParagraph {
		html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
	}

	// Close unclosed code block
	if inCodeBlock {
		html.WriteString("  <pre><code>" + c.escapeXML(codeBlock.String()) + "</code></pre>\n")
	}

	return html.String()
}

// convertInlineMarkdown converts inline markdown formatting to HTML
func (c *MarkdownToEPUBConverter) convertInlineMarkdown(text string) string {
	// First escape XML special characters in the raw text
	text = c.escapeXML(text)

	// Now convert markdown to HTML (HTML tags won't be escaped)
	// Bold: **text** or __text__ (process first to avoid conflicts)
	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "<strong>$1</strong>")
	text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "<strong>$1</strong>")

	// Italic: *text* or _text_ (single stars/underscores only)
	// Process after bold to avoid matching ** or __
	text = regexp.MustCompile(`\*([^*]+?)\*`).ReplaceAllString(text, "<em>$1</em>")
	text = regexp.MustCompile(`_([^_]+?)_`).ReplaceAllString(text, "<em>$1</em>")

	// Code: `text`
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")

	return text
}

// escapeXML escapes special XML characters
func (c *MarkdownToEPUBConverter) escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
