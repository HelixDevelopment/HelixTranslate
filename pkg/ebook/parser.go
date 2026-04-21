package ebook

import (
	"digital.vasic.translator/pkg/format"
	"fmt"
)

// Book represents a universal ebook structure
type Book struct {
	Metadata Metadata
	Chapters []Chapter
	Format   format.Format
	Language string
}

// Metadata represents book metadata
type Metadata struct {
	Title       string
	Authors     []string
	Description string
	Publisher   string
	Language    string
	ISBN        string
	Date        string
	Cover       []byte
}

// Chapter represents a book chapter
type Chapter struct {
	Title    string
	Sections []Section
}

// Section represents a chapter section
type Section struct {
	Title       string
	Content     string
	Subsections []Section
}

// Parser interface for different ebook formats
type Parser interface {
	Parse(filename string) (*Book, error)
	GetFormat() format.Format
}

// UniversalParser handles multiple ebook formats
type UniversalParser struct {
	detector *format.Detector
	parsers  map[format.Format]Parser
}

// NewUniversalParser creates a new universal parser
func NewUniversalParser() *UniversalParser {
	up := &UniversalParser{
		detector: format.NewDetector(),
		parsers:  make(map[format.Format]Parser),
	}

	// Register format-specific parsers
	up.parsers[format.FormatFB2] = NewFB2Parser()
	up.parsers[format.FormatEPUB] = NewEPUBParser()
	up.parsers[format.FormatTXT] = NewTXTParser()
	up.parsers[format.FormatHTML] = NewHTMLParser()
	up.parsers[format.FormatPDF] = NewPDFParser(nil)
	up.parsers[format.FormatDOCX] = NewDOCXParser(nil)

	return up
}

// DebugParsers returns a map of registered parsers for debugging
func (up *UniversalParser) DebugParsers() map[string]string {
	result := make(map[string]string)
	for format, parser := range up.parsers {
		result[string(format)] = fmt.Sprintf("%T", parser)
	}
	return result
}

// Parse parses any supported ebook format
func (up *UniversalParser) Parse(filename string) (*Book, error) {
	// Detect format
	detectedFormat, err := up.detector.DetectFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to detect format: %w", err)
	}

	if detectedFormat == format.FormatUnknown {
		return nil, fmt.Errorf("unknown or unsupported format")
	}

	// Check if format is supported
	if !up.detector.IsSupported(detectedFormat) {
		return nil, fmt.Errorf("format %s is not yet supported", detectedFormat)
	}

	// Get appropriate parser
	parser, ok := up.parsers[detectedFormat]
	if !ok {
		return nil, fmt.Errorf("no parser available for format %s", detectedFormat)
	}

	// Parse the book
	book, err := parser.Parse(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", detectedFormat, err)
	}

	book.Format = detectedFormat
	return book, nil
}

// GetSupportedFormats returns list of supported formats
func (up *UniversalParser) GetSupportedFormats() []format.Format {
	return up.detector.GetSupportedFormats()
}

// ConvertBook converts a book from one format to another
func ConvertBook(book *Book, targetFormat format.Format) (*Book, error) {
	// The book structure is already universal
	// We just need to change the format marker
	converted := *book
	converted.Format = targetFormat
	return &converted, nil
}

// ExtractText extracts all text content from a book
func (book *Book) ExtractText() string {
	text := book.Metadata.Title + "\n\n"

	for _, chapter := range book.Chapters {
		text += chapter.Title + "\n"
		for _, section := range chapter.Sections {
			text += extractSectionText(&section)
		}
		text += "\n"
	}

	return text
}

// extractSectionText recursively extracts text from sections
func extractSectionText(section *Section) string {
	text := ""
	if section.Title != "" {
		text += section.Title + "\n"
	}
	text += section.Content + "\n"

	for _, subsection := range section.Subsections {
		text += extractSectionText(&subsection)
	}

	return text
}

// GetChapterCount returns the number of chapters
func (book *Book) GetChapterCount() int {
	return len(book.Chapters)
}

// GetWordCount estimates the word count
func (book *Book) GetWordCount() int {
	text := book.ExtractText()
	// Simple word count estimation
	words := 0
	inWord := false
	for _, ch := range text {
		if ch == ' ' || ch == '\n' || ch == '\t' {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}
	return words
}
