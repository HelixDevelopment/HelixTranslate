package ebook

import (
	"strings"

	"digital.vasic.translator/pkg/fb2"
	"digital.vasic.translator/pkg/format"
)

// FB2Parser implements Parser for FB2 format
type FB2Parser struct{}

// NewFB2Parser creates a new FB2 parser
func NewFB2Parser() *FB2Parser {
	return &FB2Parser{}
}

// Parse parses an FB2 file into universal Book structure
func (p *FB2Parser) Parse(filename string) (*Book, error) {
	parser := fb2.NewParser()
	fb2Book, err := parser.Parse(filename)
	if err != nil {
		return nil, err
	}

	book := &Book{
		Metadata: Metadata{
			Title:    fb2Book.GetTitle(),
			Language: fb2Book.GetLanguage(),
		},
		Chapters: make([]Chapter, 0),
		Format:   format.FormatFB2,
	}

	// Extract authors
	for _, author := range fb2Book.Description.TitleInfo.Author {
		authorName := author.FirstName
		if author.MiddleName != "" {
			authorName += " " + author.MiddleName
		}
		if author.LastName != "" {
			authorName += " " + author.LastName
		}
		if authorName != "" {
			book.Metadata.Authors = append(book.Metadata.Authors, authorName)
		}
	}

	// Convert FB2 body sections to chapters
	for _, body := range fb2Book.Body {
		for _, fb2Section := range body.Section {
			chapter := convertFB2Section(&fb2Section)
			book.Chapters = append(book.Chapters, chapter)
		}
	}

	return book, nil
}

// convertFB2Section converts FB2 section to universal Chapter
func convertFB2Section(fb2Sec *fb2.Section) Chapter {
	chapter := Chapter{
		Sections: make([]Section, 0),
	}

	// Extract title — use FullParagraphText so inline-element text in the
	// title (e.g. <emphasis> inside a chapter heading) is not dropped, and join
	// ALL title <p> lines. FB2 <title> is a sequence of <p> lines (multi-line
	// headings are common); reading only Paragraphs[0] silently dropped every
	// title line after the first.
	if len(fb2Sec.Title.Paragraphs) > 0 {
		titleLines := make([]string, 0, len(fb2Sec.Title.Paragraphs))
		for _, para := range fb2Sec.Title.Paragraphs {
			if t := strings.TrimSpace(para.FullParagraphText()); t != "" {
				titleLines = append(titleLines, t)
			}
		}
		chapter.Title = strings.Join(titleLines, "\n")
	}

	// Create main section with paragraphs
	section := Section{
		Content: "",
	}

	for _, para := range fb2Sec.Paragraph {
		// Use FullParagraphText so emphasized/strong/linked inline words and
		// tail text are preserved in the chapter content (not dropped).
		section.Content += para.FullParagraphText() + "\n\n"
	}

	// FB2 sections may carry <subtitle>, <epigraph>, <poem>, and <cite> as direct
	// children. The fb2 parser populates these, but they were previously never
	// converted into the universal Book — every word of a section-level subtitle,
	// epigraph, poem verse, or citation was silently dropped before translation
	// and output. Append their user-visible text so the FB2 pipeline is lossless.
	for _, sub := range fb2Sec.Subtitle {
		if t := strings.TrimSpace(string(sub)); t != "" {
			section.Content += t + "\n\n"
		}
	}
	for _, epi := range fb2Sec.Epigraph {
		section.Content += flattenFB2Epigraph(epi)
	}
	for _, poem := range fb2Sec.Poem {
		section.Content += flattenFB2Poem(poem)
	}
	for _, cite := range fb2Sec.Cite {
		section.Content += flattenFB2Cite(cite)
	}

	chapter.Sections = append(chapter.Sections, section)

	// Convert subsections
	for _, subSec := range fb2Sec.Section {
		subChapter := convertFB2Section(&subSec)
		// Create subsections from the sub-chapter
		if len(subChapter.Sections) > 0 {
			for _, subSection := range subChapter.Sections {
				// Use the sub-chapter's title for the subsection
				subSection.Title = subChapter.Title
				// Add to the first section's subsections
				chapter.Sections[0].Subsections = append(chapter.Sections[0].Subsections, subSection)
			}
		}
	}

	return chapter
}

// flattenFB2Epigraph returns the user-visible text of an FB2 <epigraph>
// (paragraphs, nested poems/cites, and the text-author) in document order so no
// content is dropped when an epigraph appears directly under a section.
func flattenFB2Epigraph(epi fb2.Epigraph) string {
	var b strings.Builder
	for _, para := range epi.Paragraph {
		appendParagraph(&b, para)
	}
	for _, poem := range epi.Poem {
		b.WriteString(flattenFB2Poem(poem))
	}
	for _, cite := range epi.Cite {
		b.WriteString(flattenFB2Cite(cite))
	}
	for _, ta := range epi.TextAuthor {
		appendInline(&b, ta)
	}
	return b.String()
}

// flattenFB2Poem returns the user-visible text of an FB2 <poem> (title, nested
// epigraphs, and every stanza verse line) in document order.
func flattenFB2Poem(poem fb2.Poem) string {
	var b strings.Builder
	for _, para := range poem.Title.Paragraphs {
		appendParagraph(&b, para)
	}
	for _, epi := range poem.Epigraph {
		b.WriteString(flattenFB2Epigraph(epi))
	}
	for _, stanza := range poem.Stanza {
		for _, para := range stanza.Title.Paragraphs {
			appendParagraph(&b, para)
		}
		appendInline(&b, stanza.Subtitle)
		for _, v := range stanza.V {
			if t := strings.TrimSpace(v.Text); t != "" {
				b.WriteString(t + "\n\n")
			}
		}
	}
	return b.String()
}

// flattenFB2Cite returns the user-visible text of an FB2 <cite> (paragraphs,
// subtitles, nested poems, and the text-author) in document order.
func flattenFB2Cite(cite fb2.Cite) string {
	var b strings.Builder
	for _, para := range cite.Paragraph {
		appendParagraph(&b, para)
	}
	for _, sub := range cite.Subtitle {
		appendInline(&b, sub)
	}
	for _, poem := range cite.Poem {
		b.WriteString(flattenFB2Poem(poem))
	}
	for _, ta := range cite.TextAuthor {
		appendInline(&b, ta)
	}
	return b.String()
}

func appendParagraph(b *strings.Builder, para fb2.Paragraph) {
	if t := strings.TrimSpace(para.FullParagraphText()); t != "" {
		b.WriteString(t + "\n\n")
	}
}

func appendInline(b *strings.Builder, s fb2.InlineText) {
	if t := strings.TrimSpace(string(s)); t != "" {
		b.WriteString(t + "\n\n")
	}
}

// GetFormat returns the format
func (p *FB2Parser) GetFormat() format.Format {
	return format.FormatFB2
}
