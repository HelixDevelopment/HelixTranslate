package markdown

import (
	"archive/zip"
	"bufio"
	"digital.vasic.translator/pkg/ebook"
	"fmt"
	"os"
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
	chapters, metadata, err := c.parseMarkdown(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse markdown: %w", err)
	}

	c.metadata = metadata

	// Create EPUB
	if err := c.createEPUB(chapters, epubPath); err != nil {
		return fmt.Errorf("failed to create EPUB: %w", err)
	}

	return nil
}

// parseMarkdown parses markdown content into chapters
func (c *MarkdownToEPUBConverter) parseMarkdown(content string) ([]ebook.Chapter, ebook.Metadata, error) {
	var metadata ebook.Metadata
	var chapters []ebook.Chapter
	var currentChapter *ebook.Chapter
	var currentContent strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontmatter := false
	frontmatterCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Handle frontmatter
		if line == "---" {
			frontmatterCount++
			if frontmatterCount == 1 {
				inFrontmatter = true
				continue
			} else if frontmatterCount >= 2 {
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			// Parse metadata
			c.parseFrontmatterLine(line, &metadata)
			continue
		}

		// Skip the main title and initial content before first chapter
		if strings.HasPrefix(line, "# ") && len(chapters) == 0 && currentChapter == nil {
			metadata.Title = strings.TrimPrefix(line, "# ")
			continue
		}

		// Chapter marker (## Chapter)
		if strings.HasPrefix(line, "## ") {
			// Save previous chapter
			if currentChapter != nil {
				currentChapter.Sections = []ebook.Section{
					{Content: strings.TrimSpace(currentContent.String())},
				}
				chapters = append(chapters, *currentChapter)
				currentContent.Reset()
			}

			// Start new chapter
			chapterTitle := strings.TrimPrefix(line, "## ")
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
		return nil, metadata, fmt.Errorf("error reading markdown: %w", err)
	}

	return chapters, metadata, nil
}

// parseFrontmatterLine parses a frontmatter YAML line
func (c *MarkdownToEPUBConverter) parseFrontmatterLine(line string, metadata *ebook.Metadata) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
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
	case "has_cover":
		// Cover presence is tracked but binary data is preserved separately
		// This flag just indicates the original had a cover
	}
}

// createEPUB creates an EPUB file from chapters
func (c *MarkdownToEPUBConverter) createEPUB(chapters []ebook.Chapter, outputPath string) error {
	// Create EPUB file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create EPUB file: %w", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Write mimetype (uncompressed, first file)
	mimetypeWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		return err
	}
	mimetypeWriter.Write([]byte("application/epub+zip"))

	// Write META-INF/container.xml
	if err := c.writeContainer(zipWriter); err != nil {
		return err
	}

	// Write content.opf
	if err := c.writeContentOPF(zipWriter, chapters); err != nil {
		return err
	}

	// Write toc.ncx
	if err := c.writeTOC(zipWriter, chapters); err != nil {
		return err
	}

	// Write chapter HTML files
	for idx, chapter := range chapters {
		if err := c.writeChapterHTML(zipWriter, chapter, idx+1); err != nil {
			return err
		}
	}

	return nil
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
	var currentParagraph strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Empty line ends paragraph
		if trimmed == "" {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			continue
		}

		// Headers
		if strings.HasPrefix(trimmed, "###") {
			if inParagraph {
				html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
				currentParagraph.Reset()
				inParagraph = false
			}
			text := strings.TrimPrefix(trimmed, "###")
			text = strings.TrimSpace(text)
			html.WriteString(fmt.Sprintf("  <h3>%s</h3>\n", c.escapeXML(text)))
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

	return html.String()
}

// convertInlineMarkdown converts inline markdown formatting to HTML
func (c *MarkdownToEPUBConverter) convertInlineMarkdown(text string) string {
	// Bold: **text** or __text__ (process first to avoid conflicts)
	text = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(text, "<strong>$1</strong>")
	text = regexp.MustCompile(`__(.+?)__`).ReplaceAllString(text, "<strong>$1</strong>")

	// Italic: *text* or _text_ (single stars/underscores only)
	// Process after bold to avoid matching ** or __
	text = regexp.MustCompile(`\*([^*]+?)\*`).ReplaceAllString(text, "<em>$1</em>")
	text = regexp.MustCompile(`_([^_]+?)_`).ReplaceAllString(text, "<em>$1</em>")

	// Code: `text`
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")

	return c.escapeXML(text)
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
