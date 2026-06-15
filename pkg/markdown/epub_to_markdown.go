package markdown

import (
	"archive/zip"
	"digital.vasic.translator/pkg/ebook"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// EPUBToMarkdownConverter converts EPUB files to Markdown format
type EPUBToMarkdownConverter struct {
	preserveImages bool
	imagesDir      string
	// inCode is >0 while converting the descendants of a <code> element, where
	// the text is literal-by-fence (wrapped in backticks) and MUST NOT have its
	// markdown metacharacters backslash-escaped — escaping them would corrupt the
	// code content (`file_name` would ship as `file\_name`).
	inCode int
}

// NewEPUBToMarkdownConverter creates a new converter
func NewEPUBToMarkdownConverter(preserveImages bool, imagesDir string) *EPUBToMarkdownConverter {
	return &EPUBToMarkdownConverter{
		preserveImages: preserveImages,
		imagesDir:      imagesDir,
	}
}

// ConvertEPUBToMarkdown converts an EPUB file to Markdown
func (c *EPUBToMarkdownConverter) ConvertEPUBToMarkdown(epubPath, outputMDPath string) error {
	// Set up images directory next to markdown file
	if c.imagesDir == "" {
		mdDir := filepath.Dir(outputMDPath)
		c.imagesDir = filepath.Join(mdDir, "Images")
	}

	// Create Images directory
	if err := os.MkdirAll(c.imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	// Parse EPUB using universal parser to get metadata including cover
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(epubPath)
	if err != nil {
		// If format detection fails (e.g., detects as AZW3 but it's actually EPUB),
		// try parsing directly as EPUB
		if strings.Contains(err.Error(), "azw3") {
			epubParser := ebook.NewEPUBParser()
			book, err = epubParser.Parse(epubPath)
		}
		if err != nil {
			return fmt.Errorf("failed to parse EPUB: %w", err)
		}
	}
	metadata := book.Metadata

	// Open EPUB again to get content files structure
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer r.Close()

	// Get content files structure
	parsedMetadata, contentFiles, opfDir, err := c.parseEPUBStructure(r)
	if err != nil {
		return fmt.Errorf("failed to parse EPUB structure: %w", err)
	}

	// Merge metadata from parseEPUBStructure but preserve cover from UniversalParser
	if parsedMetadata != nil {
		originalCover := metadata.Cover // Preserve cover from UniversalParser
		metadata = *parsedMetadata
		if len(originalCover) > 0 {
			metadata.Cover = originalCover // Use cover from UniversalParser if it has one
		}
	}

	// Extract cover image if present
	var coverFilename string
	if len(metadata.Cover) > 0 {
		coverFilename = "cover.jpg"
		coverPath := filepath.Join(c.imagesDir, coverFilename)
		if err := os.WriteFile(coverPath, metadata.Cover, 0644); err != nil {
			return fmt.Errorf("failed to write cover image: %w", err)
		}
	}

	// Extract all images from EPUB
	if err := c.extractImages(r, opfDir); err != nil {
		return fmt.Errorf("failed to extract images: %w", err)
	}

	// Create markdown content
	var mdContent strings.Builder

	// Add title and metadata (YAML frontmatter)
	mdContent.WriteString("---\n")
	mdContent.WriteString(fmt.Sprintf("title: %s\n", metadata.Title))
	if len(metadata.Authors) > 0 {
		mdContent.WriteString(fmt.Sprintf("authors: %s\n", strings.Join(metadata.Authors, ", ")))
	}
	if metadata.Description != "" {
		// Escape multi-line descriptions
		desc := strings.ReplaceAll(metadata.Description, "\n", " ")
		mdContent.WriteString(fmt.Sprintf("description: %s\n", desc))
	}
	if metadata.Publisher != "" {
		mdContent.WriteString(fmt.Sprintf("publisher: %s\n", metadata.Publisher))
	}
	mdContent.WriteString(fmt.Sprintf("language: %s\n", metadata.Language))
	if metadata.ISBN != "" {
		mdContent.WriteString(fmt.Sprintf("isbn: %s\n", metadata.ISBN))
	}
	if metadata.Date != "" {
		mdContent.WriteString(fmt.Sprintf("date: %s\n", metadata.Date))
	}
	if coverFilename != "" {
		mdContent.WriteString(fmt.Sprintf("cover: Images/%s\n", coverFilename))
	}
	mdContent.WriteString("---\n\n")

	// Add main title
	mdContent.WriteString(fmt.Sprintf("# %s\n\n", metadata.Title))
	if len(metadata.Authors) > 0 {
		mdContent.WriteString(fmt.Sprintf("**By %s**\n\n", strings.Join(metadata.Authors, ", ")))
	}
	mdContent.WriteString("---\n\n")

	// Process each chapter. A chapter that fails to convert MUST NOT be dropped
	// silently (§11.4 / §11.4.6): the good chapters are still written
	// (best-effort, no data loss for the chapters that converted), and every
	// failure is collected and surfaced as an error from this function so the
	// caller knows a chapter was lost rather than discovering a truncated book
	// later.
	var chapterErrs []error
	for idx, contentFile := range contentFiles {
		fullPath := opfDir + contentFile
		found := false
		for _, f := range r.File {
			if f.Name == fullPath {
				found = true
				chapterMD, err := c.convertHTMLToMarkdown(f, idx+1)
				if err != nil {
					chapterErrs = append(chapterErrs,
						fmt.Errorf("chapter %d (%s): %w", idx+1, contentFile, err))
					break
				}
				if chapterMD != "" {
					mdContent.WriteString(chapterMD)
					mdContent.WriteString("\n\n---\n\n")
				}
				break
			}
		}
		if !found {
			chapterErrs = append(chapterErrs,
				fmt.Errorf("chapter %d (%s): content file not found in EPUB", idx+1, contentFile))
		}
	}

	// Write markdown file (write the chapters that succeeded before reporting).
	if err := os.WriteFile(outputMDPath, []byte(mdContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
	}

	if len(chapterErrs) > 0 {
		return fmt.Errorf("failed to convert %d chapter(s): %w",
			len(chapterErrs), errors.Join(chapterErrs...))
	}

	return nil
}

// parseEPUBStructure extracts metadata and content file paths from EPUB
func (c *EPUBToMarkdownConverter) parseEPUBStructure(r *zip.ReadCloser) (*ebook.Metadata, []string, string, error) {
	// Find container.xml to locate content.opf
	opfPath := ""
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			var err error
			opfPath, err = c.parseContainer(f)
			if err != nil {
				return nil, nil, "", err
			}
			break
		}
	}

	if opfPath == "" {
		return nil, nil, "", fmt.Errorf("container.xml not found")
	}

	// Extract OPF directory path
	opfDir := ""
	if idx := strings.LastIndex(opfPath, "/"); idx != -1 {
		opfDir = opfPath[:idx+1]
	}

	// Parse content.opf
	var metadata ebook.Metadata
	var contentFiles []string
	for _, f := range r.File {
		if f.Name == opfPath {
			var err error
			metadata, contentFiles, err = c.parseOPF(f, r, opfDir)
			if err != nil {
				return nil, nil, "", err
			}
			break
		}
	}

	return &metadata, contentFiles, opfDir, nil
}

// parseContainer parses container.xml
func (c *EPUBToMarkdownConverter) parseContainer(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	type Container struct {
		Rootfiles struct {
			Rootfile []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfile"`
		} `xml:"rootfiles"`
	}

	var container Container
	if err := xml.NewDecoder(rc).Decode(&container); err != nil {
		return "", err
	}

	if len(container.Rootfiles.Rootfile) > 0 {
		return container.Rootfiles.Rootfile[0].FullPath, nil
	}

	return "", fmt.Errorf("no rootfile found")
}

// parseOPF parses content.opf for metadata and spine
func (c *EPUBToMarkdownConverter) parseOPF(f *zip.File, r *zip.ReadCloser, opfDir string) (ebook.Metadata, []string, error) {
	rc, err := f.Open()
	if err != nil {
		return ebook.Metadata{}, nil, err
	}
	defer rc.Close()

	type Package struct {
		Metadata struct {
			Title       []string `xml:"title"`
			Creator     []string `xml:"creator"`
			Language    string   `xml:"language"`
			Description []string `xml:"description"`
			Publisher   []string `xml:"publisher"`
			Date        []string `xml:"date"`
			Identifier  []string `xml:"identifier"`
			Meta        []struct {
				Name    string `xml:"name,attr"`
				Content string `xml:"content,attr"`
			} `xml:"meta"`
		} `xml:"metadata"`
		Spine struct {
			Itemref []struct {
				Idref string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
		Manifest struct {
			Item []struct {
				ID         string `xml:"id,attr"`
				Href       string `xml:"href,attr"`
				MediaType  string `xml:"media-type,attr"`
				Properties string `xml:"properties,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
	}

	var pkg Package
	if err := xml.NewDecoder(rc).Decode(&pkg); err != nil {
		return ebook.Metadata{}, nil, err
	}

	metadata := ebook.Metadata{
		Language: pkg.Metadata.Language,
		Authors:  pkg.Metadata.Creator,
	}
	if len(pkg.Metadata.Title) > 0 {
		metadata.Title = pkg.Metadata.Title[0]
	}
	if len(pkg.Metadata.Description) > 0 {
		metadata.Description = pkg.Metadata.Description[0]
	}
	if len(pkg.Metadata.Publisher) > 0 {
		metadata.Publisher = pkg.Metadata.Publisher[0]
	}
	if len(pkg.Metadata.Date) > 0 {
		metadata.Date = pkg.Metadata.Date[0]
	}
	// Extract ISBN from identifier
	for _, id := range pkg.Metadata.Identifier {
		if strings.Contains(strings.ToLower(id), "isbn") || len(id) >= 10 {
			metadata.ISBN = id
			break
		}
	}

	// Build ID to href mapping
	idToHref := make(map[string]string)
	for _, item := range pkg.Manifest.Item {
		idToHref[item.ID] = item.Href
	}

	// Extract cover if present
	var coverHref string
	for _, item := range pkg.Manifest.Item {
		// Use same comprehensive cover detection as EPUBParser
		if strings.ToLower(item.ID) == "cover" ||
			strings.ToLower(item.ID) == "cover-image" ||
			strings.Contains(strings.ToLower(item.Properties), "cover-image") ||
			strings.Contains(strings.ToLower(item.Href), "cover") {
			if strings.HasPrefix(item.MediaType, "image/") {
				coverHref = item.Href
				break
			}
		}
	}

	// Also check for cover in meta tags (same as EPUBParser)
	for _, meta := range pkg.Metadata.Meta {
		if meta.Name == "cover" {
			if href, ok := idToHref[meta.Content]; ok {
				coverHref = href
				break
			}
		}
	}

	if coverHref != "" {
		// Find cover file in zip
		for _, f := range r.File {
			if f.Name == opfDir+"/"+coverHref || f.Name == coverHref {
				rc, err := f.Open()
				if err == nil {
					defer rc.Close()
					coverData, err := io.ReadAll(rc)
					if err == nil {
						metadata.Cover = coverData
					}
				}
				break
			}
		}
	}

	// Get content files in spine order
	var contentFiles []string
	for _, itemref := range pkg.Spine.Itemref {
		if href, ok := idToHref[itemref.Idref]; ok {
			contentFiles = append(contentFiles, href)
		}
	}

	return metadata, contentFiles, nil
}

// convertHTMLToMarkdown converts an HTML/XHTML file to Markdown
func (c *EPUBToMarkdownConverter) convertHTMLToMarkdown(f *zip.File, chapterNum int) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the body element
	body := c.findBody(doc)
	if body == nil {
		return "", nil
	}

	// Convert body content to markdown
	var mdBuilder strings.Builder
	c.convertChildren(body, &mdBuilder, 0)

	content := mdBuilder.String()
	content = strings.TrimSpace(content)

	if content == "" {
		return "", nil
	}

	return content, nil
}

// findBody recursively searches for the body element
func (c *EPUBToMarkdownConverter) findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if body := c.findBody(child); body != nil {
			return body
		}
	}
	return nil
}

// convertNode recursively converts HTML nodes to Markdown
func (c *EPUBToMarkdownConverter) convertNode(n *html.Node, md *strings.Builder, depth int) {
	if n.Type == html.TextNode {
		// Collapse internal whitespace runs to a single space but PRESERVE a
		// single leading/trailing space when the source had one, so the space
		// separating adjacent inline elements (e.g. "<strong>x</strong> y") is
		// not lost. Leading/trailing whitespace at block boundaries is cleaned
		// up later by the block markers + the chapter-level TrimSpace in
		// convertHTMLToMarkdown, so this does not reintroduce block-edge bugs.
		text := collapseInlineWhitespace(n.Data)
		if text != "" {
			// Outside a <code> span, the EPUB source text is plain prose whose
			// markdown metacharacters (`*`, `_`, `[`, etc.) are LITERAL. They must
			// be backslash-escaped so the markdown->EPUB return trip (which honours
			// CommonMark backslash escapes via protectEscapes/restoreEscapes) does
			// NOT reinterpret them as emphasis/links and permanently corrupt the
			// user-visible text (e.g. "C_3 and C_4" -> "C<em>3 and C</em>4").
			// Inside <code> the content is literal-by-fence (backtick-wrapped) and
			// must be emitted verbatim.
			if c.inCode == 0 {
				text = escapeMarkdownText(text)
			}
			md.WriteString(text)
		}
		return
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1":
			md.WriteString("\n\n# ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "h2":
			md.WriteString("\n\n## ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "h3":
			md.WriteString("\n\n### ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "h4":
			md.WriteString("\n\n#### ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "h5":
			md.WriteString("\n\n##### ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "h6":
			md.WriteString("\n\n###### ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "p":
			// Emit the paragraph's inline content into a sub-builder first so a
			// LINE-LEADING markdown block marker (a prose paragraph whose text
			// legitimately begins with "1. ", "- ", "> ", "#") can be backslash-
			// escaped before it reaches the markdown. Without this, the MD->EPUB
			// block scanner (convertMarkdownToXHTML) sees the leading marker and
			// silently converts the prose paragraph into an <ol>/<ul>/<blockquote>/
			// heading, dropping the marker chars ("<p>1. text</p>" -> "<ol><li>text
			// </li></ol>", the "1." lost). Block-level partner of the inline
			// escapeMarkdownText; the MD->EPUB protectEscapes/restoreEscapes restores
			// the literal marker so the round-trip yields the exact original text.
			var pb strings.Builder
			c.convertChildren(n, &pb, depth)
			md.WriteString("\n\n")
			md.WriteString(escapeLeadingBlockMarker(pb.String()))
			md.WriteString("\n\n")
		case "br":
			md.WriteString("  \n")
		case "strong", "b":
			md.WriteString("**")
			c.convertChildren(n, md, depth)
			md.WriteString("**")
		case "em", "i":
			md.WriteString("*")
			c.convertChildren(n, md, depth)
			md.WriteString("*")
		case "code":
			md.WriteString("`")
			c.inCode++
			c.convertChildren(n, md, depth)
			c.inCode--
			md.WriteString("`")
		case "pre":
			// A <pre> holds preformatted text whose newlines and indentation are
			// significant. Recursing through convertChildren would (a) run every
			// text node through collapseInlineWhitespace — flattening newlines to
			// spaces and stripping leading indentation — and (b) wrap any inner
			// <code> in stray inline backticks. Both DESTROY the code block on the
			// round-trip. Instead, gather the descendant text VERBATIM so the
			// fenced block survives exactly as written.
			md.WriteString("\n\n```\n")
			md.WriteString(strings.TrimRight(rawText(n), "\n"))
			md.WriteString("\n```\n\n")
		case "blockquote":
			md.WriteString("\n\n> ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "ul":
			md.WriteString("\n\n")
			c.convertListItems(n, md, depth+1, false)
			md.WriteString("\n\n")
		case "ol":
			md.WriteString("\n\n")
			c.convertListItems(n, md, depth+1, true)
			md.WriteString("\n\n")
		case "li":
			// A bare <li> reached here is outside any <ul>/<ol> (orphan) — the
			// real list items are emitted by convertListItems. Keep the legacy
			// behaviour: render only the inline content, no marker.
			if depth > 0 {
				c.convertChildren(n, md, depth)
			}
		case "a":
			href := c.getAttribute(n, "href")
			md.WriteString("[")
			c.convertChildren(n, md, depth)
			md.WriteString("](")
			md.WriteString(href)
			md.WriteString(")")
		case "img":
			src := c.getAttribute(n, "src")
			alt := c.getAttribute(n, "alt")
			// Convert image src to Images/ reference
			imgFilename := filepath.Base(src)
			md.WriteString(fmt.Sprintf("![%s](Images/%s)", alt, imgFilename))
		case "table":
			// Render an HTML table back to a GFM pipe table so the table survives
			// the EPUB->markdown direction (md->XHTML emits <table> for a pipe
			// table; the reverse MUST reproduce the pipe table or the structure is
			// lost). Without this, the default branch flattened every cell's text
			// into one run with no row/column structure.
			md.WriteString("\n\n")
			c.convertTable(n, md, depth)
			md.WriteString("\n\n")
		case "hr":
			md.WriteString("\n\n---\n\n")
		default:
			// For unknown elements, just process children
			c.convertChildren(n, md, depth)
		}
	}
	// Note: Sibling processing is handled by convertChildren loop
}

// convertChildren converts all child nodes
func (c *EPUBToMarkdownConverter) convertChildren(n *html.Node, md *strings.Builder, depth int) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.convertNode(child, md, depth)
	}
}

// convertListItems emits the direct <li> children of a <ul>/<ol> with the
// correct marker. Unordered lists use "- "; ordered lists use a monotonic
// "1. ", "2. ", ... counter so the numbering survives the HTML->Markdown
// conversion (a plain "- " for <ol> silently loses the user's ordering).
// Non-<li> children (text, nested elements directly under the list) are passed
// through unchanged so structure is preserved.
func (c *EPUBToMarkdownConverter) convertListItems(n *html.Node, md *strings.Builder, depth int, ordered bool) {
	indent := ""
	if depth > 0 {
		indent = strings.Repeat("  ", depth-1)
	}
	itemNum := 0
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "li" {
			itemNum++
			md.WriteString(indent)
			if ordered {
				md.WriteString(fmt.Sprintf("%d. ", itemNum))
			} else {
				md.WriteString("- ")
			}
			c.convertChildren(child, md, depth)
			md.WriteString("\n")
			continue
		}
		c.convertNode(child, md, depth)
	}
}

// rawText returns the concatenated text of all descendant text nodes of n
// WITHOUT any whitespace collapsing — used for <pre> content where newlines and
// indentation are significant and must survive verbatim.
func rawText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return b.String()
}

// convertTable renders an HTML <table> as a GFM pipe table. It gathers every
// <tr> (whether under <thead>/<tbody>/<tfoot> or directly under <table>), treats
// the first row as the header, emits a delimiter row, then the data rows. Cell
// inline content is converted via convertChildren so links/emphasis in cells
// survive; a literal "|" inside a cell is backslash-escaped so it does not get
// mistaken for a column separator on the next parse.
func (c *EPUBToMarkdownConverter) convertTable(table *html.Node, md *strings.Builder, depth int) {
	var rows [][]string
	var collectRows func(n *html.Node)
	collectRows = func(n *html.Node) {
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != html.ElementNode {
				continue
			}
			switch child.Data {
			case "thead", "tbody", "tfoot":
				collectRows(child)
			case "tr":
				var cells []string
				for cell := child.FirstChild; cell != nil; cell = cell.NextSibling {
					if cell.Type == html.ElementNode && (cell.Data == "td" || cell.Data == "th") {
						var cb strings.Builder
						c.convertChildren(cell, &cb, depth)
						text := strings.TrimSpace(cb.String())
						text = strings.ReplaceAll(text, "|", "\\|")
						cells = append(cells, text)
					}
				}
				rows = append(rows, cells)
			}
		}
	}
	collectRows(table)

	if len(rows) == 0 {
		return
	}
	writeRow := func(cells []string) {
		md.WriteString("| ")
		md.WriteString(strings.Join(cells, " | "))
		md.WriteString(" |\n")
	}
	writeRow(rows[0])
	// Delimiter row matching the header's column count.
	delims := make([]string, len(rows[0]))
	for i := range delims {
		delims[i] = "---"
	}
	writeRow(delims)
	for _, r := range rows[1:] {
		writeRow(r)
	}
}

// collapseInlineWhitespace collapses every run of whitespace in an HTML text
// node to a single ASCII space. A leading and/or trailing space is preserved
// IFF the source text had leading/trailing whitespace — this keeps the space
// that separates adjacent inline elements ("<strong>x</strong> y") while still
// normalising the multiple-space / newline noise that HTML source carries. A
// text node that is entirely whitespace collapses to a single space (a genuine
// inline separator); an empty string stays empty.
func collapseInlineWhitespace(s string) string {
	if s == "" {
		return ""
	}
	hasLead := isASCIISpace(s[0])
	hasTrail := isASCIISpace(s[len(s)-1])

	fields := strings.Fields(s)
	if len(fields) == 0 {
		// All whitespace: a single separating space.
		return " "
	}
	collapsed := strings.Join(fields, " ")
	if hasLead {
		collapsed = " " + collapsed
	}
	if hasTrail {
		collapsed += " "
	}
	return collapsed
}

// markdownInlineMeta is the set of inline markdown metacharacters that the
// markdown->EPUB direction interprets (emphasis `*`/`_`, inline code backtick,
// link/image brackets and parens) and that its protectEscapes/restoreEscapes
// pair will faithfully restore from a leading backslash. Escaping these in the
// EPUB->markdown plain-text output keeps literal prose ("C_3", "a*b", "[x](y)")
// from being re-interpreted as formatting on the return trip.
const markdownInlineMeta = "`*_[]()"

// escapeMarkdownText backslash-escapes the inline markdown metacharacters in a
// plain-text run so a markdown->EPUB re-parse treats them as literals. The
// backslash itself is escaped first so an already-present "\" is preserved
// rather than being consumed as an escape of the following character.
func escapeMarkdownText(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		if r == '\\' || strings.ContainsRune(markdownInlineMeta, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// leadingBlockMarkerRegex matches a LINE-LEADING markdown block marker on the
// first line of a paragraph's emitted inline content: an unordered bullet
// ("-"/"*"/"+"), an ordered marker ("N." / "N)"), a blockquote (">"), or an ATX
// heading ("#".."######"). Group 1 = leading whitespace (preserved), group 2 =
// the marker token to backslash-escape, group 3 = the required trailing
// whitespace + rest (preserved). The trailing-whitespace requirement (\s, or
// end-of-line for a bare ">") mirrors exactly what the MD->EPUB block scanner
// (convertMarkdownToXHTML / parseListItemLine) treats as a block marker, so a
// mid-word "."/"-"/">" or a marker with no following space is NOT matched and
// therefore NOT over-escaped.
var leadingBlockMarkerRegex = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)]|>|#{1,6})(\s.*|>?$)`)

// escapeLeadingBlockMarker backslash-escapes a LINE-LEADING markdown block marker
// in a paragraph's emitted content so the MD->EPUB block scanner treats it as
// literal prose rather than a list/heading/blockquote. Only the first line is
// considered (a paragraph emits a single logical block); the MD->EPUB
// protectEscapes/restoreEscapes pair restores the exact literal on the return trip.
func escapeLeadingBlockMarker(content string) string {
	nl := strings.IndexByte(content, '\n')
	first := content
	rest := ""
	if nl >= 0 {
		first = content[:nl]
		rest = content[nl:]
	}

	m := leadingBlockMarkerRegex.FindStringSubmatch(first)
	if m == nil {
		return content
	}
	lead := m[1]
	marker := m[2]
	tail := m[3]

	// Insert the backslash before the first metacharacter of the marker. For
	// ordered markers ("N." / "N)") the metacharacter is the final "."/")"; for
	// the single-char markers it is the whole marker.
	var escaped string
	last := marker[len(marker)-1]
	if last == '.' || last == ')' {
		escaped = marker[:len(marker)-1] + `\` + string(last)
	} else {
		escaped = `\` + marker
	}
	return lead + escaped + tail + rest
}

// isASCIISpace reports whether b is one of the HTML whitespace bytes.
func isASCIISpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

// getAttribute gets an attribute value from a node
func (c *EPUBToMarkdownConverter) getAttribute(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// extractImages extracts all images from EPUB to Images directory
func (c *EPUBToMarkdownConverter) extractImages(r *zip.ReadCloser, opfDir string) error {
	for _, f := range r.File {
		// Check if file is an image
		if strings.HasPrefix(f.Name, opfDir) && isImageFile(f.Name) {
			// Extract filename from path
			filename := filepath.Base(f.Name)

			// Skip cover.jpg (already extracted separately)
			if filename == "cover.jpg" {
				continue
			}

			// Read image data
			rc, err := f.Open()
			if err != nil {
				continue
			}

			imgData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}

			// Write to Images directory
			imgPath := filepath.Join(c.imagesDir, filename)
			if err := os.WriteFile(imgPath, imgData, 0644); err != nil {
				return fmt.Errorf("failed to write image %s: %w", filename, err)
			}
		}
	}
	return nil
}

// isImageFile checks if filename has an image extension
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".svg" || ext == ".webp"
}

// ConvertBookToMarkdown converts a Book struct to markdown and saves it
func ConvertBookToMarkdown(book *ebook.Book, outputPath string) error {
	var mdContent strings.Builder

	// Add frontmatter
	mdContent.WriteString("---\n")
	mdContent.WriteString(fmt.Sprintf("title: %s\n", book.Metadata.Title))
	if len(book.Metadata.Authors) > 0 {
		mdContent.WriteString(fmt.Sprintf("authors: %s\n", strings.Join(book.Metadata.Authors, ", ")))
	}
	mdContent.WriteString(fmt.Sprintf("language: %s\n", book.Metadata.Language))
	mdContent.WriteString("---\n\n")

	// Add main title
	mdContent.WriteString(fmt.Sprintf("# %s\n\n", book.Metadata.Title))
	if len(book.Metadata.Authors) > 0 {
		mdContent.WriteString(fmt.Sprintf("**By %s**\n\n", strings.Join(book.Metadata.Authors, ", ")))
	}
	mdContent.WriteString("---\n\n")

	// Add chapters
	for idx, chapter := range book.Chapters {
		mdContent.WriteString(fmt.Sprintf("## Chapter %d\n\n", idx+1))
		if chapter.Title != "" {
			mdContent.WriteString(fmt.Sprintf("### %s\n\n", chapter.Title))
		}

		for _, section := range chapter.Sections {
			mdContent.WriteString(section.Content)
			mdContent.WriteString("\n\n")
		}

		mdContent.WriteString("---\n\n")
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(mdContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	return nil
}
