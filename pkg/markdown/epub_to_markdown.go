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
		return fmt.Errorf("failed to parse EPUB: %w", err)
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
	
	// Use metadata from parseEPUBStructure as it might have cover info
	if parsedMetadata != nil {
		metadata = *parsedMetadata
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

	// Extract cover if present
	var coverHref string
	for _, item := range pkg.Manifest.Item {
		if item.ID == "cover" && strings.HasSuffix(item.Href, ".jpg") {
			coverHref = item.Href
			break
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
			// Convert image src to Images/ reference
			imgFilename := filepath.Base(src)
			md.WriteString(fmt.Sprintf("![%s](Images/%s)", alt, imgFilename))
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
