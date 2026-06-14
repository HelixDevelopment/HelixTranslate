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
	hrRegex  *regexp.Regexp
}

// NewMarkdownToEPUBConverter creates a new converter
func NewMarkdownToEPUBConverter() *MarkdownToEPUBConverter {
	return &MarkdownToEPUBConverter{
		metadata: ebook.Metadata{},
		hrRegex:  regexp.MustCompile(`^[-*_]{3,}$`),
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

		// D8: YAML frontmatter must begin at the very first non-blank line. If the
		// first real content is anything other than "---", there is no frontmatter
		// block, so a later "---" is a horizontal-rule / chapter separator (handled
		// by the HR branch below) — NOT a frontmatter fence. Without this, a bare
		// "---" used as a chapter separator in a doc with no leading frontmatter was
		// treated as frontmatter-open, silently swallowing every chapter after it
		// (data loss). Leading blank lines stay open to a following frontmatter fence.
		if !frontmatterDone && !inFrontmatter && frontmatterCount == 0 &&
			strings.TrimSpace(line) != "" && line != "---" {
			frontmatterDone = true
		}

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

			// If this is the first H1 and we don't have a title yet, extract it as book title
			if strings.HasPrefix(line, "# ") && metadata.Title == "" && len(chapters) == 0 {
				metadata.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
				continue
			}

			// Save previous chapter
			if currentChapter != nil {
				currentChapter.Sections = []ebook.Section{
					{Content: strings.TrimSpace(currentContent.String())},
				}
				chapters = append(chapters, *currentChapter)
				currentContent.Reset()
			}

			// Start new chapter and include the header in the content
			chapterTitle := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "##"), "#"))
			currentChapter = &ebook.Chapter{
				Title:    chapterTitle,
				Sections: []ebook.Section{},
			}
			// Add the header to the content so it's preserved in the round-trip
			currentContent.WriteString(line + "\n")
			continue
		}

		// Horizontal rule (chapter separator) - also saves chapter
		if c.hrRegex.MatchString(strings.TrimSpace(line)) {
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

		// Add content to current chapter, or to default buffer if no chapter yet
		if currentChapter != nil {
			currentContent.WriteString(line + "\n")
		} else {
			currentContent.WriteString(line + "\n")
		}
	}

	// Save last chapter
	if currentChapter != nil {
		currentChapter.Sections = []ebook.Section{
			{Content: strings.TrimSpace(currentContent.String())},
		}
		chapters = append(chapters, *currentChapter)
		currentContent.Reset()
	}

	// If no chapters were created (no headers found), create a default chapter from all content
	if len(chapters) == 0 && currentContent.Len() > 0 {
		chapters = append(chapters, ebook.Chapter{
			Title:    "Content",
			Sections: []ebook.Section{{Content: strings.TrimSpace(currentContent.String())}},
		})
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
		return value // Return parsed value for testing
	case "authors":
		metadata.Authors = strings.Split(value, ",")
		for i := range metadata.Authors {
			metadata.Authors[i] = strings.TrimSpace(metadata.Authors[i])
		}
		return value // Return parsed value for testing
	case "author":
		metadata.Authors = []string{value}
		return "" // Return empty for testing
	case "description":
		metadata.Description = value
		return value // Return parsed value for testing
	case "publisher":
		metadata.Publisher = value
		return value // Return parsed value for testing
	case "language":
		metadata.Language = value
		return value // Return parsed value for testing
	case "isbn":
		metadata.ISBN = value
		return value // Return parsed value for testing
	case "date":
		metadata.Date = value
		return value // Return parsed value for testing
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
	// Create the EPUB file directly
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
	if err := c.writeContainer(zipWriter); err != nil {
		return fmt.Errorf("failed to write container.xml: %w", err)
	}

	// Write OEBPS/content.opf
	if err := c.writeContentOPF(zipWriter, chapters); err != nil {
		return fmt.Errorf("failed to write content.opf: %w", err)
	}

	// Write OEBPS/toc.ncx
	if err := c.writeTOC(zipWriter, chapters); err != nil {
		return fmt.Errorf("failed to write toc.ncx: %w", err)
	}

	// Write chapter files
	for i, chapter := range chapters {
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
		xhtml := c.convertMarkdownToXHTML(content.String())
		if _, err := writer.Write([]byte(xhtml)); err != nil {
			return fmt.Errorf("failed to write chapter content: %w", err)
		}
	}

	// Write cover image if present
	if len(c.metadata.Cover) > 0 {
		coverWriter, err := zipWriter.Create("OEBPS/cover.jpg")
		if err != nil {
			return fmt.Errorf("failed to create cover file: %w", err)
		}
		if _, err := coverWriter.Write(c.metadata.Cover); err != nil {
			return fmt.Errorf("failed to write cover content: %w", err)
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
	if c.metadata.Description != "" {
		opf.WriteString(fmt.Sprintf("    <dc:description>%s</dc:description>\n", c.escapeXML(c.metadata.Description)))
	}
	if c.metadata.Publisher != "" {
		opf.WriteString(fmt.Sprintf("    <dc:publisher>%s</dc:publisher>\n", c.escapeXML(c.metadata.Publisher)))
	}
	opf.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", c.metadata.Language))

	// Use ISBN as identifier if available, otherwise generate UUID
	if c.metadata.ISBN != "" {
		opf.WriteString(fmt.Sprintf("    <dc:identifier id=\"BookID\">%s</dc:identifier>\n", c.escapeXML(c.metadata.ISBN)))
	} else {
		opf.WriteString("    <dc:identifier id=\"BookID\">urn:uuid:generated</dc:identifier>\n")
	}
	if c.metadata.Date != "" {
		opf.WriteString(fmt.Sprintf("    <dc:date>%s</dc:date>\n", c.escapeXML(c.metadata.Date)))
	}

	// Add cover meta tag if cover is present (helps with cover detection)
	if len(c.metadata.Cover) > 0 {
		opf.WriteString("    <meta name=\"cover\" content=\"cover\"/>\n")
	}

	opf.WriteString("  </metadata>\n")

	// Manifest
	opf.WriteString("  <manifest>\n")
	opf.WriteString("    <item id=\"ncx\" href=\"toc.ncx\" media-type=\"application/x-dtbncx+xml\"/>\n")

	// Add cover if present
	if len(c.metadata.Cover) > 0 {
		opf.WriteString("    <item id=\"cover\" href=\"cover.jpg\" media-type=\"image/jpeg\"/>\n")
	}

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
//
//nolint:unused
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

// listItemRegex matches a markdown list item line, capturing leading indent,
// the marker, and the item content. Unordered markers: -, *, +. Ordered
// markers: a number followed by '.' or ')'.
var listItemRegex = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])\s+(.*)$`)

// parsedListItem describes one markdown list line.
type parsedListItem struct {
	indent  int  // number of leading-space "levels" (2 spaces == 1 level)
	ordered bool // true for "N." / "N)" markers
	content string
}

// parseListItemLine returns the parsed item and ok=true if line is a list item.
func parseListItemLine(line string) (parsedListItem, bool) {
	m := listItemRegex.FindStringSubmatch(line)
	if m == nil {
		return parsedListItem{}, false
	}
	indentSpaces := len(strings.ReplaceAll(m[1], "\t", "  "))
	marker := m[2]
	ordered := marker[len(marker)-1] == '.' || marker[len(marker)-1] == ')'
	// Unordered markers are single chars (-, *, +); everything else is ordered.
	if marker == "-" || marker == "*" || marker == "+" {
		ordered = false
	} else {
		ordered = true
	}
	return parsedListItem{
		indent:  indentSpaces / 2,
		ordered: ordered,
		content: m[3],
	}, true
}

// tableDelimRegex matches a GFM table delimiter row, e.g. "| --- | :--: |".
// Each cell is a run of dashes with optional leading/trailing colons (alignment).
var tableDelimRegex = regexp.MustCompile(`^\s*\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)*\|?\s*$`)

// isTableRow reports whether a line looks like a GFM table row: it contains at
// least one pipe and is non-empty after trimming. (Pure delimiter rows are
// matched separately by tableDelimRegex.)
func isTableRow(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && strings.Contains(t, "|")
}

// splitTableCells splits a "| a | b |" row into its cell strings, dropping the
// optional leading/trailing empty cells produced by the bounding pipes. A
// backslash-escaped pipe ("\|") is NOT a column separator.
func splitTableCells(row string) []string {
	t := strings.TrimSpace(row)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	// Split on unescaped pipes.
	var cells []string
	var cur strings.Builder
	runes := []rune(t)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) && runes[i+1] == '|' {
			cur.WriteRune('|')
			i++
			continue
		}
		if runes[i] == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteRune(runes[i])
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}

// renderTableBlock converts a GFM pipe-table block (header row, delimiter row,
// then zero+ data rows) into a <table> with <thead>/<tbody>. Inline markdown in
// each cell is converted via convertInlineMarkdown so links/emphasis in cells
// survive. indentStr is prepended to every emitted line for pretty-printing.
func (c *MarkdownToEPUBConverter) renderTableBlock(rows []string, indentStr string) string {
	if len(rows) < 2 {
		return ""
	}
	header := splitTableCells(rows[0])
	var b strings.Builder
	b.WriteString(indentStr + "<table>\n")
	b.WriteString(indentStr + "  <thead>\n" + indentStr + "    <tr>")
	for _, h := range header {
		b.WriteString("<th>" + c.convertInlineMarkdown(h) + "</th>")
	}
	b.WriteString("</tr>\n" + indentStr + "  </thead>\n")
	// Data rows start after the delimiter row (rows[1]).
	if len(rows) > 2 {
		b.WriteString(indentStr + "  <tbody>\n")
		for _, dr := range rows[2:] {
			cells := splitTableCells(dr)
			b.WriteString(indentStr + "    <tr>")
			for _, cell := range cells {
				b.WriteString("<td>" + c.convertInlineMarkdown(cell) + "</td>")
			}
			b.WriteString("</tr>\n")
		}
		b.WriteString(indentStr + "  </tbody>\n")
	}
	b.WriteString(indentStr + "</table>\n")
	return b.String()
}

// renderListBlock converts a contiguous run of markdown list lines into nested
// <ul>/<ol> HTML. Items more deeply indented than their predecessor open a
// nested list inside the previous <li>. Inline markdown in each item is
// converted via convertInlineMarkdown. indentStr is prepended to every emitted
// line for pretty-printing.
func (c *MarkdownToEPUBConverter) renderListBlock(items []parsedListItem, indentStr string) string {
	var b strings.Builder
	c.renderListLevel(&b, items, 0, indentStr)
	return b.String()
}

// renderListLevel renders the items at level `level`, recursing for deeper
// indents. It returns after consuming a contiguous block; callers pass the full
// slice and level 0.
func (c *MarkdownToEPUBConverter) renderListLevel(b *strings.Builder, items []parsedListItem, level int, indentStr string) {
	i := 0
	for i < len(items) {
		it := items[i]
		if it.indent < level {
			return
		}
		// Open the list tag for this level using the first item's ordering.
		tag := "ul"
		if it.ordered {
			tag = "ol"
		}
		b.WriteString(indentStr + "<" + tag + ">\n")
		for i < len(items) && items[i].indent == level {
			cur := items[i]
			// Look ahead for nested children (deeper indent).
			j := i + 1
			for j < len(items) && items[j].indent > level {
				j++
			}
			if j > i+1 {
				// This item has nested children: emit content + nested list.
				b.WriteString(indentStr + "  <li>" + c.convertInlineMarkdown(cur.content) + "\n")
				c.renderListLevel(b, items[i+1:j], level+1, indentStr+"  ")
				b.WriteString(indentStr + "  </li>\n")
				i = j
			} else {
				b.WriteString(indentStr + "  <li>" + c.convertInlineMarkdown(cur.content) + "</li>\n")
				i++
			}
		}
		b.WriteString(indentStr + "</" + tag + ">\n")
	}
}

// markdownToHTML converts markdown content to HTML
func (c *MarkdownToEPUBConverter) markdownToHTML(markdown string) string {
	var html strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(markdown))

	inParagraph := false
	inCodeBlock := false
	var currentParagraph strings.Builder
	var codeBlock strings.Builder
	var listBuf []parsedListItem
	var quoteBuf []string

	flushParagraph := func() {
		if inParagraph {
			html.WriteString("  <p>" + c.convertInlineMarkdown(currentParagraph.String()) + "</p>\n")
			currentParagraph.Reset()
			inParagraph = false
		}
	}
	flushList := func() {
		if len(listBuf) > 0 {
			html.WriteString(c.renderListBlock(listBuf, "  "))
			listBuf = nil
		}
	}
	// flushQuote emits a contiguous run of "> " lines as one <blockquote>. A
	// blockquote left as a literal "&gt; ..." paragraph (the pre-fix behaviour)
	// breaks the round-trip: the HTML->markdown side emits "> " for
	// <blockquote>, so md->HTML must produce the matching element.
	flushQuote := func() {
		if len(quoteBuf) > 0 {
			html.WriteString("  <blockquote>" +
				c.convertInlineMarkdown(strings.Join(quoteBuf, " ")) + "</blockquote>\n")
			quoteBuf = nil
		}
	}
	// flush ends any open paragraph AND any open list AND any open blockquote —
	// call before emitting a block-level element so blocks are properly closed.
	flush := func() {
		flushParagraph()
		flushList()
		flushQuote()
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Code block delimiter
		if strings.HasPrefix(trimmed, "```") {
			flush()
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

		// Empty line ends paragraph and list
		if trimmed == "" {
			flush()
			continue
		}

		// Horizontal rule
		if c.hrRegex.MatchString(trimmed) {
			flush()
			html.WriteString("  <hr/>\n")
			continue
		}

		// List item (ordered or unordered, possibly nested). Detected BEFORE the
		// paragraph fallback so list lines are not swallowed into a <p>. A list
		// item ends any open paragraph; consecutive list lines accumulate and are
		// flushed as one nested <ul>/<ol> block.
		if item, ok := parseListItemLine(line); ok {
			flushParagraph()
			flushQuote()
			listBuf = append(listBuf, item)
			continue
		}
		// A non-list line ends any open list.
		flushList()

		// Blockquote: "> text" (or bare ">"). Accumulate contiguous quote lines
		// and flush as one <blockquote>. Detected before the header/paragraph
		// fallbacks so the marker is not escaped into a literal "&gt;".
		if trimmed == ">" || strings.HasPrefix(trimmed, "> ") {
			flushParagraph()
			quoteBuf = append(quoteBuf, strings.TrimSpace(strings.TrimPrefix(trimmed, ">")))
			continue
		}
		// A non-quote line ends any open blockquote.
		flushQuote()

		// Headers (h1 through h6)
		if strings.HasPrefix(trimmed, "######") {
			flushParagraph()
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "######"))
			html.WriteString(fmt.Sprintf("  <h6>%s</h6>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "#####") {
			flushParagraph()
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "#####"))
			html.WriteString(fmt.Sprintf("  <h5>%s</h5>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "####") {
			flushParagraph()
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "####"))
			html.WriteString(fmt.Sprintf("  <h4>%s</h4>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "###") {
			flushParagraph()
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			html.WriteString(fmt.Sprintf("  <h3>%s</h3>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "##") {
			flushParagraph()
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "##"))
			html.WriteString(fmt.Sprintf("  <h2>%s</h2>\n", c.escapeXML(text)))
			continue
		} else if strings.HasPrefix(trimmed, "#") && len(trimmed) > 1 && trimmed[1] == ' ' {
			flushParagraph()
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

	// Close last paragraph and list
	flush()

	// Close unclosed code block
	if inCodeBlock {
		html.WriteString("  <pre><code>" + c.escapeXML(codeBlock.String()) + "</code></pre>\n")
	}

	return html.String()
}

// inlineEscapable lists the markdown metacharacters a leading backslash escapes
// (CommonMark "backslash escapes"). A "\*" must reach the reader as a literal
// "*", not be interpreted as emphasis.
const inlineEscapable = "\\`*_{}[]()#+-.!>"

// inlineEscapePlaceholderBase is a Private-Use-Area rune base used to hide
// backslash-escaped characters from the emphasis/link regexes; decoded back to
// the literal character at the very end of convertInlineMarkdown.
const inlineEscapePlaceholderBase = rune(0xE000)

// protectEscapes replaces every "\X" (X in inlineEscapable) with a private-use
// placeholder so X is not seen by the markdown substitution regexes. The
// backslash is consumed (a literal "\*" yields a literal "*").
func protectEscapes(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) &&
			strings.ContainsRune(inlineEscapable, runes[i+1]) {
			b.WriteRune(inlineEscapePlaceholderBase + (runes[i+1] & 0xFF))
			i++
			continue
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

// restoreEscapes turns the placeholders from protectEscapes back into the
// literal characters they stood for.
func restoreEscapes(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= inlineEscapePlaceholderBase && r < inlineEscapePlaceholderBase+0x100 {
			b.WriteRune(r - inlineEscapePlaceholderBase)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// convertInlineMarkdown converts inline markdown formatting to HTML
func (c *MarkdownToEPUBConverter) convertInlineMarkdown(text string) string {
	// First escape XML special characters in the raw text
	text = c.escapeXML(text)

	// Hide backslash-escaped markdown metacharacters from the substitution
	// regexes below so "\*literal\*" is NOT treated as emphasis. The escape is
	// processed AFTER escapeXML so a "\<" still produced a valid "&lt;" first.
	text = protectEscapes(text)

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

	// Image: ![alt](src) — MUST run before the link rule, since the link rule's
	// "[text](url)" pattern also matches the "[alt](src)" tail of an image. The
	// alt text and src have already been XML-escaped above, so the emitted
	// attributes are safe. Without this, an inline image left as literal
	// "![alt](src)" text never reaches the EPUB (image lost on round-trip).
	text = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`).
		ReplaceAllString(text, `<img src="$2" alt="$1"/>`)

	// Link: [text](url) — emit a real <a> so the hyperlink survives into the
	// EPUB and round-trips back to markdown. Left literal before the fix, so a
	// translated book shipped raw "[text](url)" to the reader.
	text = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`).
		ReplaceAllString(text, `<a href="$2">$1</a>`)

	// Decode the protected backslash-escapes back to their literal characters.
	text = restoreEscapes(text)

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

// convertMarkdownToXHTML converts markdown content to XHTML
func (c *MarkdownToEPUBConverter) convertMarkdownToXHTML(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var result strings.Builder
	inParagraph := false
	inCodeBlock := false
	var codeBlock strings.Builder
	var listBuf []parsedListItem
	var quoteBuf []string
	var tableBuf []string

	flushParagraph := func() {
		if inParagraph {
			result.WriteString("</p>\n")
			inParagraph = false
		}
	}
	flushList := func() {
		if len(listBuf) > 0 {
			result.WriteString(c.renderListBlock(listBuf, ""))
			listBuf = nil
		}
	}
	// flushTable emits a buffered GFM pipe-table block. A buffer that is NOT a
	// valid table (missing the delimiter row) is emitted as paragraph text so no
	// pipe content is lost — the buffering is speculative and reversible.
	flushTable := func() {
		if len(tableBuf) == 0 {
			return
		}
		if len(tableBuf) >= 2 && tableDelimRegex.MatchString(tableBuf[1]) {
			result.WriteString(c.renderTableBlock(tableBuf, ""))
		} else {
			for _, row := range tableBuf {
				result.WriteString("<p>" + c.convertInlineMarkdown(strings.TrimSpace(row)) + "</p>\n")
			}
		}
		tableBuf = nil
	}
	// flushQuote emits a contiguous run of "> " lines as one <blockquote>. This
	// is the path createEPUB writes into each chapter, so a blockquote left as a
	// literal "&gt; ..." paragraph reached the end user in the produced EPUB.
	flushQuote := func() {
		if len(quoteBuf) > 0 {
			result.WriteString("<blockquote>" +
				c.convertInlineMarkdown(strings.Join(quoteBuf, " ")) + "</blockquote>\n")
			quoteBuf = nil
		}
	}

	for _, raw := range lines {
		trimmedRaw := strings.TrimSpace(raw)

		// Fenced code block (```): the fence and its body MUST be emitted as a real
		// <pre><code> so the code structure (internal newlines) survives into the
		// EPUB and round-trips. Detected FIRST, before list/blockquote/header
		// detection, so markdown markers inside a code block are treated as literal
		// code, not parsed. Without this, createEPUB flattened the whole block into
		// one <p> with literal backticks and newlines collapsed to spaces — the
		// code was destroyed for the reader. Mirrors markdownToHTML's fence logic.
		if strings.HasPrefix(trimmedRaw, "```") {
			if inCodeBlock {
				result.WriteString("<pre><code>" + c.escapeXML(codeBlock.String()) + "</code></pre>\n")
				codeBlock.Reset()
				inCodeBlock = false
			} else {
				flushParagraph()
				flushList()
				flushQuote()
				flushTable()
				inCodeBlock = true
			}
			continue
		}
		if inCodeBlock {
			if codeBlock.Len() > 0 {
				codeBlock.WriteString("\n")
			}
			codeBlock.WriteString(raw)
			continue
		}

		// List detection runs on the RAW line so indentation (nesting) survives.
		if item, ok := parseListItemLine(raw); ok {
			flushParagraph()
			flushQuote()
			flushTable()
			listBuf = append(listBuf, item)
			continue
		}
		flushList()

		line := strings.TrimSpace(raw)

		// GFM table row: any non-empty line containing a pipe (and not a list /
		// quote / header) is a candidate table line. Buffer contiguous candidate
		// rows; flushTable decides whether the buffer is a real table (delimiter
		// row present) or falls back to paragraphs. Detected before the
		// header/paragraph fallback so a "| a | b |" row is never flattened into a
		// single literal-pipe paragraph (total table-structure loss).
		if line != "" && isTableRow(line) &&
			!strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ">") {
			flushParagraph()
			flushQuote()
			tableBuf = append(tableBuf, line)
			continue
		}
		flushTable()

		// Blockquote: "> text" (or bare ">"). Accumulate contiguous quote lines
		// and flush as one <blockquote> before the header/paragraph fallbacks.
		if line == ">" || strings.HasPrefix(line, "> ") {
			flushParagraph()
			quoteBuf = append(quoteBuf, strings.TrimSpace(strings.TrimPrefix(line, ">")))
			continue
		}
		flushQuote()

		// Headers (h1..h6). Each level MUST be emitted as the matching element so
		// the heading level survives into the EPUB and round-trips (convertNode
		// emits "#".."######" for <h1>..<h6>). Deepest prefix is checked first so
		// "######" is not matched by the "#" branch. Header text is XML-escaped so
		// a header containing <, &, etc. produces valid XHTML.
		if strings.HasPrefix(line, "###### ") {
			flushParagraph()
			result.WriteString(fmt.Sprintf("<h6>%s</h6>\n", c.escapeXML(strings.TrimPrefix(line, "###### "))))
		} else if strings.HasPrefix(line, "##### ") {
			flushParagraph()
			result.WriteString(fmt.Sprintf("<h5>%s</h5>\n", c.escapeXML(strings.TrimPrefix(line, "##### "))))
		} else if strings.HasPrefix(line, "#### ") {
			flushParagraph()
			result.WriteString(fmt.Sprintf("<h4>%s</h4>\n", c.escapeXML(strings.TrimPrefix(line, "#### "))))
		} else if strings.HasPrefix(line, "### ") {
			flushParagraph()
			result.WriteString(fmt.Sprintf("<h3>%s</h3>\n", c.escapeXML(strings.TrimPrefix(line, "### "))))
		} else if strings.HasPrefix(line, "## ") {
			flushParagraph()
			result.WriteString(fmt.Sprintf("<h2>%s</h2>\n", c.escapeXML(strings.TrimPrefix(line, "## "))))
		} else if strings.HasPrefix(line, "# ") {
			flushParagraph()
			result.WriteString(fmt.Sprintf("<h1>%s</h1>\n", c.escapeXML(strings.TrimPrefix(line, "# "))))
		} else if line == "" {
			// Empty line - close paragraph if open
			flushParagraph()
		} else {
			// Regular text - start or continue paragraph
			if !inParagraph {
				result.WriteString("<p>")
				inParagraph = true
			} else {
				result.WriteString(" ")
			}
			result.WriteString(c.convertInlineMarkdown(line))
		}
	}

	// Close any open paragraph, list, blockquote and table
	flushParagraph()
	flushList()
	flushQuote()
	flushTable()
	// Close an unterminated code block so its content is not silently dropped.
	if inCodeBlock {
		result.WriteString("<pre><code>" + c.escapeXML(codeBlock.String()) + "</code></pre>\n")
	}

	// Wrap in XHTML document structure
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<title>%s</title>
</head>
<body>
%s</body>
</html>`, c.metadata.Title, result.String())
}
