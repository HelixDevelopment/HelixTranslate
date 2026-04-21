package ebook

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"digital.vasic.translator/pkg/format"
	"github.com/unidoc/unioffice/document"
)

type DOCXParser struct {
	config *DOCXConfig
}

type DOCXConfig struct {
	ExtractImages      bool     `yaml:"extract_images"`
	ImageFormat        string   `yaml:"image_format"`
	ExtractTables      bool     `yaml:"extract_tables"`
	ExtractFootnotes   bool     `yaml:"extract_footnotes"`
	ExtractHeaders     bool     `yaml:"extract_headers"`
	ExtractFooters     bool     `yaml:"extract_footers"`
	ExtractComments    bool     `yaml:"extract_comments"`
	PreserveFormatting bool     `yaml:"preserve_formatting"`
	ExtractMetadata    bool     `yaml:"extract_metadata"`
	MinTextLength      int      `yaml:"min_text_length"`
	IgnoreStyles       []string `yaml:"ignore_styles"`
}

func NewDOCXParser(config *DOCXConfig) *DOCXParser {
	if config == nil {
		config = &DOCXConfig{
			ExtractImages:      true,
			ImageFormat:        "png",
			ExtractTables:      true,
			ExtractFootnotes:   true,
			ExtractHeaders:     true,
			ExtractFooters:     true,
			ExtractComments:    true,
			PreserveFormatting: false,
			ExtractMetadata:    true,
			MinTextLength:      1,
			IgnoreStyles:       []string{},
		}
	}
	return &DOCXParser{config: config}
}

func (p *DOCXParser) Parse(filename string) (*Book, error) {
	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseWithContext(context.Background(), data)
}

func (p *DOCXParser) ParseWithContext(ctx context.Context, data []byte) (*Book, error) {
	doc, err := document.Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX document: %w", err)
	}

	book := &Book{
		Metadata: Metadata{
			Title: "Document",
		},
	}

	// Extract metadata
	if p.config.ExtractMetadata {
		if err := p.extractMetadata(doc, book); err != nil {
			return nil, fmt.Errorf("failed to extract metadata: %w", err)
		}
	}

	// Extract content as plain text
	var allText strings.Builder

	// Simple paragraph extraction
	paragraphs := doc.Paragraphs()
	for i := 0; i < len(paragraphs); i++ {
		para := paragraphs[i]

		// Simple text extraction from paragraph
		runs := para.Runs()
		for j := 0; j < len(runs); j++ {
			run := runs[j]
			allText.WriteString(run.Text())
		}

		// Add paragraph separator
		if i < len(paragraphs)-1 {
			allText.WriteString("\n\n")
		}

		// Check for context cancellation
		if i%10 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
	}

	// Create main content as first chapter
	mainChapter := Chapter{
		Title: "Document Content",
		Sections: []Section{
			{
				Title:   "Main Content",
				Content: allText.String(),
			},
		},
	}

	book.Chapters = append(book.Chapters, mainChapter)
	book.Language = book.Metadata.Language

	return book, nil
}

func (p *DOCXParser) Validate(data []byte) error {
	_, err := document.Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid DOCX structure: %w", err)
	}

	return nil
}

func (p *DOCXParser) SupportedFormats() []string {
	return []string{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
}

func (p *DOCXParser) GetMetadata(data []byte) (*Metadata, error) {
	doc, err := document.Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX document: %w", err)
	}

	metadata := &Metadata{}
	err = p.extractMetadata(doc, &Book{Metadata: *metadata})
	if err != nil {
		return nil, err
	}

	// Return a copy to avoid mutation
	result := *metadata
	return &result, nil
}

func (p *DOCXParser) GetFormat() format.Format {
	return format.FormatDOCX
}

func (p *DOCXParser) extractMetadata(doc *document.Document, book *Book) error {
	// Extract core properties - simplified implementation
	props := doc.CoreProperties

	// Try to get title
	if props.Title() != "" {
		book.Metadata.Title = props.Title()
	}

	// Note: The API is different than expected, skip author extraction for now

	// Try to get description
	if props.Description() != "" {
		book.Metadata.Description = props.Description()
	}

	// Note: Skip language extraction due to API differences

	// Try to get creation date
	if !props.Created().IsZero() {
		book.Metadata.Date = props.Created().Format(time.RFC3339)
	}

	return nil
}
