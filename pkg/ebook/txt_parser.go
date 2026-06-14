package ebook

import (
	"bufio"
	"digital.vasic.translator/pkg/format"
	"os"
	"strings"
)

// maxTXTLineBytes bounds the single-line size the TXT scanner accepts. Set well
// above the 64 KiB default so a chapter-on-one-line ebook parses without the
// bufio.ErrTooLong total-content-loss failure, while still capping pathological
// input (a binary file with no newlines) so the parser cannot be driven to
// unbounded memory.
const maxTXTLineBytes = 64 * 1024 * 1024 // 64 MiB

// TXTParser implements Parser for plain text format
type TXTParser struct{}

// NewTXTParser creates a new TXT parser
func NewTXTParser() *TXTParser {
	return &TXTParser{}
}

// Parse parses a plain text file into universal Book structure
func (p *TXTParser) Parse(filename string) (*Book, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	book := &Book{
		Metadata: Metadata{
			Title: filename,
		},
		Chapters: make([]Chapter, 0),
		Format:   format.FormatTXT,
	}

	// Read content. bufio.Scanner's default token buffer is capped at
	// bufio.MaxScanTokenSize (64 KiB); a TXT ebook whose chapter is emitted as a
	// single physical line (common in stripped exports) exceeds that, making
	// Scan() abort with bufio.ErrTooLong and losing the ENTIRE book. Lift the
	// scanner's max-token limit so no line length can truncate content, while
	// keeping the exact line-by-line "<line>\n" output shape (and bufio's
	// \r\n -> \n normalisation) the rest of the pipeline expects.
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTXTLineBytes)
	var content strings.Builder

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Create single chapter with all content
	chapter := Chapter{
		Title: "Content",
		Sections: []Section{
			{
				Content: content.String(),
			},
		},
	}

	book.Chapters = append(book.Chapters, chapter)

	return book, nil
}

// GetFormat returns the format
func (p *TXTParser) GetFormat() format.Format {
	return format.FormatTXT
}
