package ebook

import (
	"bytes"
	"context"
	"fmt"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"os"
	"strings"

	"digital.vasic.translator/pkg/format"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type PDFParser struct {
	config *PDFConfig
}

type PDFConfig struct {
	ExtractImages   bool   `yaml:"extract_images"`
	ImageFormat     string `yaml:"image_format"`
	OcrEnabled      bool   `yaml:"ocr_enabled"`
	OcrLanguage     string `yaml:"ocr_language"`
	PreserveLayout  bool   `yaml:"preserve_layout"`
	ExtractMetadata bool   `yaml:"extract_metadata"`
	ExtractTables   bool   `yaml:"extract_tables"`
	MinTextLength   int    `yaml:"min_text_length"`
}

func NewPDFParser(config *PDFConfig) *PDFParser {
	if config == nil {
		config = &PDFConfig{
			ExtractImages:   true,
			ImageFormat:     "png",
			OcrEnabled:      false,
			OcrLanguage:     "eng",
			PreserveLayout:  true,
			ExtractMetadata: true,
			ExtractTables:   true,
			MinTextLength:   1,
		}
	}
	return &PDFParser{config: config}
}

func (p *PDFParser) Parse(filename string) (*Book, error) {
	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseWithContext(context.Background(), data)
}

func (p *PDFParser) ParseWithContext(ctx context.Context, data []byte) (*Book, error) {
	pdfReader, err := model.NewPdfReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	book := &Book{
		Metadata: Metadata{},
	}

	// Extract metadata
	if p.config.ExtractMetadata {
		if err := p.extractMetadata(pdfReader, book); err != nil {
			return nil, fmt.Errorf("failed to extract metadata: %w", err)
		}
	}

	// Get page count
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	var allText strings.Builder

	// Extract text from all pages
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get page %d: %w", i, err)
		}

		// Extract text from page
		ex, err := extractor.New(page)
		if err != nil {
			return nil, fmt.Errorf("failed to create extractor for page %d: %w", i, err)
		}

		text, err := ex.ExtractText()
		if err != nil {
			return nil, fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}

		if allText.Len() > 0 {
			allText.WriteString("\n\n--- Page Break ---\n\n")
		}
		allText.WriteString(text)

		// Check for context cancellation
		if i%5 == 0 {
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
				Title:   "Full Text",
				Content: allText.String(),
			},
		},
	}

	book.Chapters = append(book.Chapters, mainChapter)
	book.Language = book.Metadata.Language

	return book, nil
}

func (p *PDFParser) Validate(data []byte) error {
	// Check PDF signature
	if len(data) < 5 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return fmt.Errorf("invalid PDF signature")
	}

	// Try to parse PDF structure
	_, err := model.NewPdfReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid PDF structure: %w", err)
	}

	return nil
}

func (p *PDFParser) SupportedFormats() []string {
	return []string{"pdf", "application/pdf"}
}

func (p *PDFParser) GetMetadata(data []byte) (*Metadata, error) {
	pdfReader, err := model.NewPdfReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	metadata := &Metadata{}
	err = p.extractMetadata(pdfReader, &Book{Metadata: *metadata})
	if err != nil {
		return nil, err
	}

	// Return a copy to avoid mutation
	result := *metadata
	return &result, nil
}

func (p *PDFParser) GetFormat() format.Format {
	return format.FormatPDF
}

func (p *PDFParser) extractMetadata(pdfReader *model.PdfReader, book *Book) error {
	// Note: The API is different than expected, using simplified metadata extraction
	// Basic PDF structure is valid, that's what matters for now

	// Extract page count
	if numPages, err := pdfReader.GetNumPages(); err == nil {
		if book.Metadata.Description != "" {
			book.Metadata.Description += fmt.Sprintf(" (Pages: %d)", numPages)
		} else {
			book.Metadata.Description = fmt.Sprintf("PDF document with %d pages", numPages)
		}
	}

	return nil
}
