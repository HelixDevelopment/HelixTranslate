package ebook

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"digital.vasic.translator/pkg/format"
)

// DOCXParser parses .docx (OOXML WordprocessingML) documents.
//
// It reads the OOXML package directly with the standard library (archive/zip +
// encoding/xml) — extracting paragraph text from word/document.xml and core
// metadata from docProps/core.xml. The previous implementation used
// github.com/unidoc/unioffice, which is LICENSE-GATED: document.Read returned
// "unioffice license required" for every real .docx, so DOCX input was entirely
// non-functional in this build. This stdlib parser makes DOCX input actually
// work and removes the gated dependency.
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
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.ParseWithContext(context.Background(), data)
}

func (p *DOCXParser) ParseWithContext(ctx context.Context, data []byte) (*Book, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX (not a valid OOXML zip): %w", err)
	}

	book := &Book{
		Metadata: Metadata{
			Title: "Document",
		},
	}

	if p.config.ExtractMetadata {
		// Metadata is best-effort: a document without docProps/core.xml is still
		// a valid document, so a metadata miss must not fail the whole parse.
		_ = p.extractMetadataFromZip(zr, book)
	}

	paragraphs, err := extractDOCXParagraphs(ctx, zr)
	if err != nil {
		return nil, err
	}

	// Preserve paragraph structure: join with the blank-line separator the rest
	// of the pipeline (formatSection / FB2 splitIntoParagraphs) treats as a
	// paragraph boundary.
	mainChapter := Chapter{
		Title: "Document Content",
		Sections: []Section{
			{
				Title:   "Main Content",
				Content: strings.Join(paragraphs, "\n\n"),
			},
		},
	}

	book.Chapters = append(book.Chapters, mainChapter)
	book.Language = book.Metadata.Language

	return book, nil
}

// extractDOCXParagraphs streams word/document.xml and returns one string per
// <w:p> paragraph (runs concatenated; <w:tab> -> "\t"; <w:br>/<w:cr> -> "\n").
// Matching is by element Local name so it is robust to namespace-prefix
// variations across producers.
func extractDOCXParagraphs(ctx context.Context, zr *zip.Reader) ([]string, error) {
	rc, err := openZipEntry(zr, "word/document.xml")
	if err != nil {
		return nil, fmt.Errorf("DOCX missing word/document.xml: %w", err)
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var paragraphs []string
	var cur strings.Builder
	inText, inPara := false, false
	count := 0

	for {
		if count%64 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
		count++

		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse word/document.xml: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inPara = true
				cur.Reset()
			case "t":
				inText = true
			case "tab":
				if inPara {
					cur.WriteByte('\t')
				}
			case "br", "cr":
				if inPara {
					cur.WriteByte('\n')
				}
			}
		case xml.CharData:
			if inText {
				cur.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				if inPara {
					paragraphs = append(paragraphs, cur.String())
					inPara = false
				}
			}
		}
	}

	return paragraphs, nil
}

// extractMetadataFromZip reads docProps/core.xml (Dublin Core + cp properties).
func (p *DOCXParser) extractMetadataFromZip(zr *zip.Reader, book *Book) error {
	rc, err := openZipEntry(zr, "docProps/core.xml")
	if err != nil {
		return err
	}
	defer rc.Close()

	dec := xml.NewDecoder(rc)
	var cur strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			cur.Reset()
		case xml.CharData:
			cur.Write(t)
		case xml.EndElement:
			val := strings.TrimSpace(cur.String())
			if val == "" {
				continue
			}
			switch t.Name.Local {
			case "title":
				book.Metadata.Title = val
			case "description", "subject":
				if book.Metadata.Description == "" {
					book.Metadata.Description = val
				}
			case "language":
				book.Metadata.Language = val
			case "creator":
				if len(book.Metadata.Authors) == 0 {
					book.Metadata.Authors = []string{val}
				}
			case "created":
				if tm, perr := time.Parse(time.RFC3339, val); perr == nil {
					book.Metadata.Date = tm.Format(time.RFC3339)
				} else {
					book.Metadata.Date = val
				}
			}
		}
	}
	return nil
}

func (p *DOCXParser) Validate(data []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid DOCX structure: %w", err)
	}
	rc, err := openZipEntry(zr, "word/document.xml")
	if err != nil {
		return fmt.Errorf("invalid DOCX: missing word/document.xml: %w", err)
	}
	_ = rc.Close()
	return nil
}

func (p *DOCXParser) SupportedFormats() []string {
	return []string{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
}

func (p *DOCXParser) GetMetadata(data []byte) (*Metadata, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX document: %w", err)
	}
	book := &Book{}
	_ = p.extractMetadataFromZip(zr, book)
	result := book.Metadata
	return &result, nil
}

func (p *DOCXParser) GetFormat() format.Format {
	return format.FormatDOCX
}

// openZipEntry opens a named entry from a zip reader, or returns an error if it
// is absent.
func openZipEntry(zr *zip.Reader, name string) (io.ReadCloser, error) {
	for _, f := range zr.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("entry %q not found in archive", name)
}
