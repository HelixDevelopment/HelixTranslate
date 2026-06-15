package ebook

import (
	"archive/zip"
	"digital.vasic.translator/pkg/format"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/url"
	"regexp"
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
		fullPath := resolveEPUBHref(opfDir, contentFile)
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
		coverPath := resolveEPUBHref(opfDir, coverHref)
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

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Try to parse with standard XML decoder
	type Container struct {
		Rootfiles struct {
			Rootfile []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfile"`
		} `xml:"rootfiles"`
	}

	var container Container
	if err := xml.Unmarshal(data, &container); err != nil {
		// If standard parsing fails, try to clean up the XML
		cleanData := p.CleanXMLData(data)
		if err := xml.Unmarshal(cleanData, &container); err != nil {
			return "", fmt.Errorf("failed to parse container.xml: %w", err)
		}
	}

	if len(container.Rootfiles.Rootfile) > 0 {
		return container.Rootfiles.Rootfile[0].FullPath, nil
	}

	return "", fmt.Errorf("no rootfile found in container.xml")
}

// CleanXMLData attempts to clean up malformed XML data
func (p *EPUBParser) CleanXMLData(data []byte) []byte {
	content := string(data)

	// Remove invalid characters that might cause XML parsing issues
	// Keep only valid XML characters (excluding control characters except \t, \n, \r)
	var cleaned strings.Builder
	for _, r := range content {
		if (r == 0x9) || (r == 0xA) || (r == 0xD) ||
			(r >= 0x20 && r <= 0xD7FF) ||
			(r >= 0xE000 && r <= 0xFFFD) ||
			(r >= 0x10000 && r <= 0x10FFFF) {
			cleaned.WriteRune(r)
		}
	}

	// Escape ONLY bare ampersands — a '&' that does not already begin a valid
	// XML entity reference (&amp; &lt; &gt; &quot; &apos; or a numeric &#...; /
	// &#x...;). The previous implementation blindly rewrote the 2-char prefixes
	// "&a"/"&l"/"&g"/"&q" into full entities, which corrupted ALREADY-VALID
	// entities (e.g. "&amp;" → "&amp;mp;" because "&a" matched inside it) and any
	// ordinary text containing those sequences ("Q&A", "AT&T", "rock & roll").
	// This pass is idempotent on valid markup and only fixes the
	// genuinely-unescaped ampersands that break the XML parse.
	cleanedStr := escapeBareAmpersands(cleaned.String())

	return []byte(cleanedStr)
}

// validEntityRe matches a complete, well-formed XML entity reference (named or
// numeric) anchored at the current position. Go's RE2 has no lookahead, so
// escapeBareAmpersands scans each '&' and tests whether a valid entity follows.
var validEntityRe = regexp.MustCompile(`^&(?:[a-zA-Z][a-zA-Z0-9]*|#[0-9]+|#[xX][0-9a-fA-F]+);`)

// escapeBareAmpersands replaces every '&' that does NOT start a well-formed XML
// entity reference with "&amp;", leaving already-valid entities untouched. This
// is the actual fix for "invalid character entity" parse failures without
// corrupting valid markup or ordinary text.
func escapeBareAmpersands(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '&' && validEntityRe.MatchString(s[i:]) {
			// Copy the whole valid entity verbatim.
			loc := validEntityRe.FindStringIndex(s[i:])
			b.WriteString(s[i : i+loc[1]])
			i += loc[1] - 1
			continue
		}
		if s[i] == '&' {
			b.WriteString("&amp;")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// resolveEPUBHref resolves a manifest/spine href (a URI reference, relative to
// the OPF file) to the literal zip entry name. EPUB OCF hrefs are percent-encoded
// per the spec (RFC 3986), so a chapter file whose zip entry name is
// "OEBPS/chapter one.xhtml" is referenced as "chapter%20one.xhtml". The previous
// implementation concatenated opfDir+href verbatim, so the encoded href never
// matched the literal (decoded) zip entry and the chapter/cover was SILENTLY
// DROPPED (data loss). This decodes the percent-encoding and normalises any
// "./" / "../" path segments relative to the OPF directory. Fragment (#...) and
// query (?...) parts, if present, are stripped — they never form part of a zip
// entry name. If decoding fails, fall back to the raw concatenation so a
// malformed href is no worse than before.
func resolveEPUBHref(opfDir, href string) string {
	// Strip any fragment / query that does not belong to the file path.
	if i := strings.IndexAny(href, "#?"); i != -1 {
		href = href[:i]
	}

	decoded, err := url.PathUnescape(href)
	if err != nil {
		decoded = href
	}

	full := opfDir + decoded
	return normalizeZipPath(full)
}

// normalizeZipPath collapses "./" and "x/../" segments in a forward-slash path
// without touching the leading structure, so an OPF-relative "../images/c.jpg"
// resolves to the correct zip entry. It deliberately does NOT use path.Clean for
// the whole string blindly — but path.Clean on a slash path is exactly the
// normalisation zip entries use, so we apply it and re-collapse the result.
func normalizeZipPath(p string) string {
	segs := strings.Split(p, "/")
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		switch s {
		case "", ".":
			// Drop empty (from "//") and current-dir segments, except keep a
			// trailing/leading meaningfully — handled by rejoin below.
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, s)
		}
	}
	return strings.Join(out, "/")
}

// parseOPF parses content.opf for metadata and content files
func (p *EPUBParser) parseOPF(f *zip.File, book *Book) ([]string, string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}

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
	if err := xml.Unmarshal(data, &pkg); err != nil {
		// If standard parsing fails, try to clean up the XML
		cleanData := p.CleanXMLData(data)
		if err := xml.Unmarshal(cleanData, &pkg); err != nil {
			return nil, "", fmt.Errorf("failed to parse content.opf: %w", err)
		}
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

	// Simple HTML text extraction - remove head/title sections first
	content := string(data)

	// Extract the chapter's real title BEFORE the head is stripped. Prefer the
	// document <title>, falling back to the first <h1> heading. The previous
	// implementation used the zip entry name (e.g. "OEBPS/chapter1.xhtml") as the
	// chapter title, so an EPUB -> translate -> EPUB round-trip emitted that
	// internal filename as the chapter <h1> heading AND the NCX/TOC navLabel
	// (writeChapters/writeTOC both use chapter.Title) — visible corruption — while
	// the real title was lost as a title (it only survived folded into body text).
	chapterTitle := extractChapterTitle(content)
	if chapterTitle == "" {
		chapterTitle = f.Name
	}

	// Remove entire head section including title. The `(?s)` flag is REQUIRED so
	// `.` matches newlines: real EPUB XHTML almost always spans the <head> over
	// several lines (<title>, charset <meta>, stylesheet <link> on separate
	// lines). Without `(?s)` a multi-line head is never matched, so the <title>
	// TEXT survives tag-stripping and leaks into the extracted chapter content.
	headRe := regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	content = headRe.ReplaceAllString(content, " ")

	// Remove tags from remaining content
	content = removeHTMLTags(content)

	// Decode HTML/XML character references. removeHTMLTags only strips TAGS; it
	// leaves entities (&amp; &lt; &gt; &quot; &apos; and numeric &#NNN; / &#xHH;)
	// verbatim. Without this pass those entities survive as literal markup in the
	// chapter text — the translator then ships "Tom &amp; Jerry" / "caf&#233;" to
	// the reader instead of "Tom & Jerry" / "café". html.UnescapeString decodes
	// every named + numeric reference and is a no-op on entity-free text.
	content = html.UnescapeString(content)

	// Clean up multiple spaces
	spaceRe := regexp.MustCompile(` {2,}`)
	content = spaceRe.ReplaceAllString(content, " ")

	content = strings.TrimSpace(content)

	if content == "" {
		return nil, nil
	}

	chapter := &Chapter{
		Title: chapterTitle,
		Sections: []Section{
			{
				Content: content,
			},
		},
	}

	return chapter, nil
}

// titleTagRe / h1TagRe extract the chapter's human-readable title from the raw
// XHTML. `(?is)` so `.` matches newlines (multi-line <head>/<h1>) and matching is
// case-insensitive.
var (
	titleTagRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	h1TagRe    = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
)

// extractChapterTitle returns the chapter's real title text — the <title>
// element if present and non-empty, otherwise the first <h1> heading. Any inner
// markup (e.g. <span> inside the heading) is stripped and entities decoded, and
// whitespace is collapsed, so the returned title is clean reader-facing text.
// Returns "" when neither element yields text, letting the caller fall back to a
// filename.
func extractChapterTitle(raw string) string {
	for _, re := range []*regexp.Regexp{titleTagRe, h1TagRe} {
		m := re.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		t := removeHTMLTags(m[1])
		t = html.UnescapeString(t)
		t = strings.Join(strings.Fields(t), " ")
		if t != "" {
			return t
		}
	}
	return ""
}

// htmlTagRe matches any HTML/XML tag form: opening (<p>), closing (</p>),
// self-closing / void (<br/>, <img .../>, <hr/>), comments (<!-- ... -->),
// doctype / declarations (<!DOCTYPE ...>), and processing instructions
// (<?xml ...?>). Each is replaced with a single space so adjacent words are not
// glued together.
var htmlTagRe = regexp.MustCompile(`(?s)<!--.*?-->|<[^>]*>`)

// removeHTMLTags removes HTML tags from text, replacing each with a single space.
//
// The previous implementation used `<[a-zA-Z][^>/]*>` for opening tags and
// `</[^>]*>` for closing tags. That pattern EXCLUDED self-closing / void tags
// (`<br/>`, `<img .../>`, `<hr/>`) — the most common tags in EPUB content — so
// they leaked through as literal markup into the extracted chapter text, and
// HTML comments / doctype declarations also survived. This single pass removes
// every tag form (and HTML comments), each replaced with a space to preserve
// word boundaries.
func removeHTMLTags(s string) string {
	return htmlTagRe.ReplaceAllString(s, " ")
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
