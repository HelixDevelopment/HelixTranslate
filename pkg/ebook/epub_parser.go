package ebook

import (
	"archive/zip"
	"digital.vasic.translator/pkg/format"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// EPUBParser implements Parser for EPUB format
type EPUBParser struct{}

// NewEPUBParser creates a new EPUB parser
func NewEPUBParser() *EPUBParser {
	return &EPUBParser{}
}

// Parse parses an EPUB file into universal Book structure
func (p *EPUBParser) Parse(filename string) (*Book, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer r.Close()

	book := &Book{
		Metadata: Metadata{},
		Chapters: make([]Chapter, 0),
		Format:   format.FormatEPUB,
	}

	// Parse container.xml to find content.opf
	opfPath := ""
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			opfPath, err = p.parseContainer(f)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if opfPath == "" {
		return nil, fmt.Errorf("container.xml not found")
	}

	// Parse content.opf for metadata and spine
	var contentFiles []string
	var coverHref string
	for _, f := range r.File {
		if f.Name == opfPath {
			contentFiles, coverHref, err = p.parseOPF(f, book)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	// Extract content from HTML/XHTML files
	opfDir := ""
	if idx := strings.LastIndex(opfPath, "/"); idx != -1 {
		opfDir = opfPath[:idx+1]
	}

	for _, contentFile := range contentFiles {
		fullPath := opfDir + contentFile
		for _, f := range r.File {
			if f.Name == fullPath {
				chapter, err := p.parseContentFile(f)
				if err == nil && chapter != nil {
					book.Chapters = append(book.Chapters, *chapter)
				}
				break
			}
		}
	}

	// Extract cover image if found
	if coverHref != "" {
		coverPath := opfDir + coverHref
		for _, f := range r.File {
			if f.Name == coverPath {
				book.Metadata.Cover, _ = p.extractCoverImage(f)
				break
			}
		}
	}

	return book, nil
}

// parseContainer parses container.xml to find content.opf location
func (p *EPUBParser) parseContainer(f *zip.File) (string, error) {
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

	return "", fmt.Errorf("no rootfile found in container.xml")
}

// parseOPF parses content.opf for metadata and content files
func (p *EPUBParser) parseOPF(f *zip.File, book *Book) ([]string, string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, "", err
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
		return nil, "", err
	}

	// Extract all metadata fields
	if len(pkg.Metadata.Title) > 0 {
		book.Metadata.Title = pkg.Metadata.Title[0]
	}
	book.Metadata.Authors = pkg.Metadata.Creator
	book.Metadata.Language = pkg.Metadata.Language

	// Extract Description
	if len(pkg.Metadata.Description) > 0 {
		book.Metadata.Description = pkg.Metadata.Description[0]
	}

	// Extract Publisher
	if len(pkg.Metadata.Publisher) > 0 {
		book.Metadata.Publisher = pkg.Metadata.Publisher[0]
	}

	// Extract Date
	if len(pkg.Metadata.Date) > 0 {
		book.Metadata.Date = pkg.Metadata.Date[0]
	}

	// Extract ISBN from identifier
	for _, id := range pkg.Metadata.Identifier {
		if strings.Contains(strings.ToLower(id), "isbn") || len(id) >= 10 {
			book.Metadata.ISBN = id
			break
		}
	}

	// Build ID to href mapping
	idToHref := make(map[string]string)
	var coverHref string
	for _, item := range pkg.Manifest.Item {
		idToHref[item.ID] = item.Href

		// Detect cover image
		if strings.ToLower(item.ID) == "cover" ||
		   strings.ToLower(item.ID) == "cover-image" ||
		   strings.Contains(strings.ToLower(item.Properties), "cover-image") ||
		   strings.Contains(strings.ToLower(item.Href), "cover") {
			if strings.HasPrefix(item.MediaType, "image/") {
				coverHref = item.Href
			}
		}
	}

	// Also check for cover in meta tags
	for _, meta := range pkg.Metadata.Meta {
		if meta.Name == "cover" {
			if href, ok := idToHref[meta.Content]; ok {
				coverHref = href
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

	return contentFiles, coverHref, nil
}

// parseContentFile parses an HTML/XHTML content file
func (p *EPUBParser) parseContentFile(f *zip.File) (*Chapter, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	// Simple HTML text extraction
	content := string(data)

	// Remove tags (simple approach)
	content = removeHTMLTags(content)
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, nil
	}

	chapter := &Chapter{
		Title: f.Name,
		Sections: []Section{
			{
				Content: content,
			},
		},
	}

	return chapter, nil
}

// removeHTMLTags removes HTML tags from text
func removeHTMLTags(s string) string {
	// Simple tag removal
	inTag := false
	var result strings.Builder

	for _, ch := range s {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
			result.WriteRune(' ')
		} else if !inTag {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

// extractCoverImage extracts cover image bytes from a zip file
func (p *EPUBParser) extractCoverImage(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// GetFormat returns the format
func (p *EPUBParser) GetFormat() format.Format {
	return format.FormatEPUB
}
