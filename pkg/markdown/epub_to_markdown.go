package markdown

import (
	"archive/zip"
	"digital.vasic.translator/pkg/ebook"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// EPUBToMarkdownConverter converts EPUB files to Markdown format
type EPUBToMarkdownConverter struct {
	preserveImages bool
	imagesDir      string
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
	// Parse EPUB
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		return fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer r.Close()

	// Get metadata and content files
	metadata, contentFiles, opfDir, err := c.parseEPUBStructure(r)
	if err != nil {
		return fmt.Errorf("failed to parse EPUB structure: %w", err)
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
	if len(metadata.Cover) > 0 {
		mdContent.WriteString("has_cover: true\n")
		// Note: Cover binary data is not stored in markdown but preserved in the metadata structure
	}
	mdContent.WriteString("---\n\n")

	// Add main title
	mdContent.WriteString(fmt.Sprintf("# %s\n\n", metadata.Title))
	if len(metadata.Authors) > 0 {
		mdContent.WriteString(fmt.Sprintf("**By %s**\n\n", strings.Join(metadata.Authors, ", ")))
	}
	mdContent.WriteString("---\n\n")

	// Process each chapter
	for idx, contentFile := range contentFiles {
		fullPath := opfDir + contentFile
		for _, f := range r.File {
			if f.Name == fullPath {
				chapterMD, err := c.convertHTMLToMarkdown(f, idx+1)
				if err == nil && chapterMD != "" {
					mdContent.WriteString(chapterMD)
					mdContent.WriteString("\n\n---\n\n")
				}
				break
			}
		}
	}

	// Write markdown file
	if err := os.WriteFile(outputMDPath, []byte(mdContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown: %w", err)
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

	// Parse content.opf
	var metadata ebook.Metadata
	var contentFiles []string
	for _, f := range r.File {
		if f.Name == opfPath {
			var err error
			metadata, contentFiles, err = c.parseOPF(f)
			if err != nil {
				return nil, nil, "", err
			}
			break
		}
	}

	// Extract OPF directory path
	opfDir := ""
	if idx := strings.LastIndex(opfPath, "/"); idx != -1 {
		opfDir = opfPath[:idx+1]
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
func (c *EPUBToMarkdownConverter) parseOPF(f *zip.File) (ebook.Metadata, []string, error) {
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
		} `xml:"metadata"`
		Spine struct {
			Itemref []struct {
				Idref string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
		Manifest struct {
			Item []struct {
				ID   string `xml:"id,attr"`
				Href string `xml:"href,attr"`
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

	// Convert to markdown
	var mdBuilder strings.Builder
	c.convertNode(doc, &mdBuilder, 0)

	content := mdBuilder.String()
	content = strings.TrimSpace(content)

	if content == "" {
		return "", nil
	}

	return content, nil
}

// convertNode recursively converts HTML nodes to Markdown
func (c *EPUBToMarkdownConverter) convertNode(n *html.Node, md *strings.Builder, depth int) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
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
			md.WriteString("\n\n")
			c.convertChildren(n, md, depth)
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
			c.convertChildren(n, md, depth)
			md.WriteString("`")
		case "pre":
			md.WriteString("\n\n```\n")
			c.convertChildren(n, md, depth)
			md.WriteString("\n```\n\n")
		case "blockquote":
			md.WriteString("\n\n> ")
			c.convertChildren(n, md, depth)
			md.WriteString("\n\n")
		case "ul":
			md.WriteString("\n\n")
			c.convertChildren(n, md, depth+1)
			md.WriteString("\n\n")
		case "ol":
			md.WriteString("\n\n")
			c.convertChildren(n, md, depth+1)
			md.WriteString("\n\n")
		case "li":
			if depth > 0 {
				md.WriteString(strings.Repeat("  ", depth-1))
				md.WriteString("- ")
				c.convertChildren(n, md, depth)
				md.WriteString("\n")
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
			md.WriteString(fmt.Sprintf("![%s](%s)", alt, src))
		case "hr":
			md.WriteString("\n\n---\n\n")
		default:
			// For unknown elements, just process children
			c.convertChildren(n, md, depth)
		}
	}

	// Process siblings
	if n.NextSibling != nil {
		c.convertNode(n.NextSibling, md, depth)
	}
}

// convertChildren converts all child nodes
func (c *EPUBToMarkdownConverter) convertChildren(n *html.Node, md *strings.Builder, depth int) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.convertNode(child, md, depth)
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
