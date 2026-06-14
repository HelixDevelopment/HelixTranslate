package ebook

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"digital.vasic.translator/pkg/format"
	"github.com/ledongthuc/pdf"
)

// PDFParser extracts text from PDF documents.
//
// Text extraction uses github.com/ledongthuc/pdf (MIT). The previous
// implementation used github.com/unidoc/unipdf, whose text extractor is
// LICENSE-GATED: ExtractText returned "unipdf license code required" for every
// real PDF, so PDF input was non-functional (a §11.4 'ships but cannot be used'
// defect — old tests only fed invalid data and asserted failure, masking it).
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
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.ParseWithContext(context.Background(), data)
}

func (p *PDFParser) ParseWithContext(ctx context.Context, data []byte) (*Book, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	book := &Book{Metadata: Metadata{}}
	numPages := reader.NumPage()
	if p.config.ExtractMetadata {
		book.Metadata.Description = fmt.Sprintf("PDF document with %d pages", numPages)
	}

	allText, err := extractPDFText(ctx, reader, numPages)
	if err != nil {
		return nil, err
	}

	book.Chapters = append(book.Chapters, Chapter{
		Title: "Document Content",
		Sections: []Section{
			{Title: "Full Text", Content: allText},
		},
	})
	book.Language = book.Metadata.Language
	return book, nil
}

// extractPDFText pulls plain text from every page, separating pages with a blank
// line. ledongthuc/pdf can panic on a few malformed PDFs, so the per-page
// extraction is wrapped in a recover that converts a panic into an error (a
// malformed page must not crash the whole process — §11.4.1).
func extractPDFText(ctx context.Context, reader *pdf.Reader, numPages int) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PDF text extraction failed (malformed PDF): %v", r)
		}
	}()

	var sb strings.Builder
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		pageText, perr := page.GetPlainText(nil)
		if perr != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", i, perr)
		}
		if sb.Len() > 0 && strings.TrimSpace(pageText) != "" {
			sb.WriteString("\n\n")
		}
		sb.WriteString(pageText)

		if i%5 == 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
		}
	}
	return sb.String(), nil
}

func (p *PDFParser) Validate(data []byte) error {
	if len(data) < 5 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return fmt.Errorf("invalid PDF signature")
	}
	if _, err := pdf.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		return fmt.Errorf("invalid PDF structure: %w", err)
	}
	return nil
}

func (p *PDFParser) SupportedFormats() []string {
	return []string{"pdf", "application/pdf"}
}

func (p *PDFParser) GetMetadata(data []byte) (*Metadata, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF: %w", err)
	}
	md := &Metadata{Description: fmt.Sprintf("PDF document with %d pages", reader.NumPage())}
	return md, nil
}

func (p *PDFParser) GetFormat() format.Format {
	return format.FormatPDF
}
