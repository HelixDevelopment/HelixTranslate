package ebook

import (
	"digital.vasic.translator/pkg/format"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// HTMLParser implements Parser for HTML format
type HTMLParser struct{}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// Parse parses an HTML file into universal Book structure
func (p *HTMLParser) Parse(filename string) (*Book, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		return nil, err
	}

	book := &Book{
		Metadata: Metadata{
			Title: filename,
		},
		Chapters: make([]Chapter, 0),
		Format:   format.FormatHTML,
	}

	// Extract title
	title := p.findTitle(doc)
	if title != "" {
		book.Metadata.Title = title
	}

	// Extract content
	content := p.extractText(doc)

	// Drop the leading body heading (<h1>) when it equals the promoted chapter
	// title, so the title is carried EXACTLY ONCE (in chapter.Title) and not also
	// repeated as the first line of Content (MINOR-W6-1). No-op when the first
	// line differs from the title, so non-duplicating inputs are untouched.
	content = stripLeadingTitle(content, book.Metadata.Title)

	// Create single chapter
	chapter := Chapter{
		Title: book.Metadata.Title,
		Sections: []Section{
			{
				Content: content,
			},
		},
	}

	book.Chapters = append(book.Chapters, chapter)

	return book, nil
}

// stripLeadingTitle removes a single leading occurrence of title from the start
// of content when content begins with it, so a chapter title promoted into
// chapter.Title is not ALSO carried as the first line/prefix of Section.Content
// (the MINOR-W6-1 duplication). It is a strict no-op when:
//   - title is empty/whitespace, or
//   - content's leading text does NOT equal title (different first line/prefix).
//
// Only the leading occurrence is removed — interior repetitions of the title
// text inside the body are preserved. Both the HTML ("Title\n\nbody") and EPUB
// ("Title body") leading-title shapes are handled.
func stripLeadingTitle(content, title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return content
	}
	// Work against a left-trimmed copy so leading whitespace before the title
	// does not defeat the prefix check; only strip when the FIRST visible token
	// run equals the title.
	lead := strings.TrimLeft(content, " \t\r\n")
	if !strings.HasPrefix(lead, t) {
		return content
	}
	rest := lead[len(t):]
	// The title must be a complete leading unit, i.e. followed by whitespace or
	// end-of-content — never a prefix of a longer word (e.g. title "Cat" must not
	// strip "Caterpillar").
	if rest != "" {
		r := rest[0]
		if r != ' ' && r != '\t' && r != '\r' && r != '\n' {
			return content
		}
	}
	return strings.TrimLeft(rest, " \t\r\n")
}

// findTitle finds the title in HTML
func (p *HTMLParser) findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		if n.FirstChild != nil {
			return n.FirstChild.Data
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := p.findTitle(c); title != "" {
			return title
		}
	}

	return ""
}

// extractText extracts text content from HTML
func (p *HTMLParser) extractText(n *html.Node) string {
	return p.extractTextWithContext(n, false)
}

func (p *HTMLParser) extractTextWithContext(n *html.Node, inPre bool) string {
	if n.Type == html.TextNode {
		// For text nodes inside pre, preserve whitespace exactly
		if inPre {
			return n.Data
		}
		// Don't trim spaces yet, preserve them for processing
		return n.Data
	}

	var content strings.Builder

	// Check if this node is a pre element
	newInPre := inPre || (n.Type == html.ElementNode && n.Data == "pre")

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		// Skip script, style and head tags. <head> (incl. its <title>) is metadata,
		// not body content — harvesting it leaked the <title> text into Content,
		// duplicating the chapter title (MINOR-W6-1,
		// docs/qa/minor_w6_1_rootcause_20260616-151123/FINDING.md).
		if c.Type == html.ElementNode && (c.Data == "script" || c.Data == "style" || c.Data == "head") {
			continue
		}

		// A <br> is a void line-break element: it carries no child text, so
		// without explicit handling the words around it get glued together
		// ("line one<br>line two" -> "line oneline two"). Emit a newline.
		if c.Type == html.ElementNode && c.Data == "br" {
			content.WriteString("\n")
			continue
		}

		text := p.extractTextWithContext(c, newInPre)
		if text != "" {
			content.WriteString(text)

			// Add newlines after block elements if we have content
			if c.Type == html.ElementNode && isBlockElement(c.Data) {
				content.WriteString("\n\n")
			} else if c.Type == html.ElementNode && isCellElement(c.Data) {
				// Table cells / definition-list items are not full blocks but
				// MUST be separated, otherwise sibling cells glue into one
				// nonsense token ("cell A" + "cell B" -> "cell Acell B").
				content.WriteString("\n")
			}
		}
	}

	result := content.String()

	// Only normalize whitespace for nodes that are not in preformatted context themselves
	// and don't have any preformatted children
	if !newInPre && !p.hasPreformattedChild(n) {
		// Replace multiple spaces with single space
		result = strings.ReplaceAll(result, "  ", " ")
		result = strings.ReplaceAll(result, "  ", " ") // Do it twice for cases with 3+ spaces

		// Replace spaces before newlines
		result = strings.ReplaceAll(result, " \n\n", "\n\n")
		result = strings.ReplaceAll(result, " \n", "\n")

		// Clean up any remaining whitespace issues
		result = strings.TrimSpace(result)
	}

	return result
}

// hasPreformattedChild checks if node has any pre descendants
func (p *HTMLParser) hasPreformattedChild(n *html.Node) bool {
	if n.Type == html.ElementNode && n.Data == "pre" {
		return true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if p.hasPreformattedChild(c) {
			return true
		}
	}
	return false
}

// isInPreformattedContext checks if node is within a pre element
//
//nolint:unused
func (p *HTMLParser) isInPreformattedContext(n *html.Node) bool {
	for parent := n.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && parent.Data == "pre" {
			return true
		}
	}
	return false
}

// isBlockElement checks if HTML element is a block element
func isBlockElement(tag string) bool {
	blockElements := []string{
		"p", "div", "h1", "h2", "h3", "h4", "h5", "h6",
		"li", "section", "article", "header", "footer",
		"blockquote", "pre",
	}

	for _, elem := range blockElements {
		if tag == elem {
			return true
		}
	}
	return false
}

// isCellElement checks if an element is a table cell or definition-list item.
// These are not full block elements (they should not force a blank line), but
// their content MUST be separated from sibling cells/items to avoid gluing
// distinct values into a single token.
func isCellElement(tag string) bool {
	switch tag {
	case "td", "th", "dt", "dd", "tr", "caption":
		return true
	default:
		return false
	}
}

// GetFormat returns the format
func (p *HTMLParser) GetFormat() format.Format {
	return format.FormatHTML
}
