package ebook

import (
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

	// Extract title
	if len(fb2Sec.Title.Paragraphs) > 0 {
		chapter.Title = fb2Sec.Title.Paragraphs[0].Text
	}

	// Create main section with paragraphs
	section := Section{
		Content: "",
	}

	for _, para := range fb2Sec.Paragraph {
		section.Content += para.Text + "\n\n"
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

// GetFormat returns the format
func (p *FB2Parser) GetFormat() format.Format {
	return format.FormatFB2
}
