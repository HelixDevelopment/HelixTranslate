package fb2

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// FB2 namespace constants
const (
	FB2Namespace   = "http://www.gribuser.ru/xml/fictionbook/2.0"
	XLinkNamespace = "http://www.w3.org/1999/xlink"
)

// FictionBook represents the root FB2 structure
type FictionBook struct {
	XMLName     xml.Name    `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook"`
	Description Description `xml:"description"`
	Body        []Body      `xml:"body"`
	Binary      []Binary    `xml:"binary,omitempty"`
}

// Description contains metadata
type Description struct {
	TitleInfo      TitleInfo      `xml:"title-info"`
	DocumentInfo   DocumentInfo   `xml:"document-info,omitempty"`
	PublishInfo    PublishInfo    `xml:"publish-info,omitempty"`
	CustomInfo     []CustomInfo   `xml:"custom-info,omitempty"`
	SrcTitleInfo   *TitleInfo     `xml:"src-title-info,omitempty"`
}

// TitleInfo contains book information
type TitleInfo struct {
	Genre      []string   `xml:"genre"`
	Author     []Author   `xml:"author"`
	BookTitle  string     `xml:"book-title"`
	Annotation Annotation `xml:"annotation,omitempty"`
	Keywords   string     `xml:"keywords,omitempty"`
	Date       Date       `xml:"date,omitempty"`
	Coverpage  Coverpage  `xml:"coverpage,omitempty"`
	Lang       string     `xml:"lang"`
	SrcLang    string     `xml:"src-lang,omitempty"`
	Translator []Author   `xml:"translator,omitempty"`
	Sequence   []Sequence `xml:"sequence,omitempty"`
}

// Author represents author information
type Author struct {
	FirstName  string `xml:"first-name,omitempty"`
	MiddleName string `xml:"middle-name,omitempty"`
	LastName   string `xml:"last-name,omitempty"`
	Nickname   string `xml:"nickname,omitempty"`
	HomePage   string `xml:"home-page,omitempty"`
	Email      string `xml:"email,omitempty"`
}

// Annotation represents book annotation
type Annotation struct {
	Paragraphs []Paragraph `xml:"p"`
}

// Date represents a date with optional value attribute
type Date struct {
	Value string `xml:"value,attr,omitempty"`
	Text  string `xml:",chardata"`
}

// Coverpage contains cover image reference
type Coverpage struct {
	Image Image `xml:"image"`
}

// Image represents an image reference
type Image struct {
	Href string `xml:"http://www.w3.org/1999/xlink href,attr"`
	Alt  string `xml:"alt,attr,omitempty"`
}

// Sequence represents a book series
type Sequence struct {
	Name   string `xml:"name,attr"`
	Number int    `xml:"number,attr,omitempty"`
}

// DocumentInfo contains document metadata
type DocumentInfo struct {
	Author      []Author `xml:"author"`
	ProgramUsed string   `xml:"program-used,omitempty"`
	Date        Date     `xml:"date"`
	SrcURL      []string `xml:"src-url,omitempty"`
	SrcOCR      string   `xml:"src-ocr,omitempty"`
	ID          string   `xml:"id"`
	Version     string   `xml:"version"`
}

// PublishInfo contains publishing information
type PublishInfo struct {
	BookName  string `xml:"book-name,omitempty"`
	Publisher string `xml:"publisher,omitempty"`
	City      string `xml:"city,omitempty"`
	Year      string `xml:"year,omitempty"`
	ISBN      string `xml:"isbn,omitempty"`
}

// CustomInfo contains custom metadata
type CustomInfo struct {
	InfoType string `xml:"info-type,attr"`
	Text     string `xml:",chardata"`
}

// Body represents the main content body
type Body struct {
	Name    string    `xml:"name,attr,omitempty"`
	Title   Title     `xml:"title,omitempty"`
	Section []Section `xml:"section"`
}

// Section represents a content section
type Section struct {
	ID        string      `xml:"id,attr,omitempty"`
	Title     Title       `xml:"title,omitempty"`
	Epigraph  []Epigraph  `xml:"epigraph,omitempty"`
	Section   []Section   `xml:"section,omitempty"`
	Paragraph []Paragraph `xml:"p,omitempty"`
	Poem      []Poem      `xml:"poem,omitempty"`
	Subtitle  []string    `xml:"subtitle,omitempty"`
	Cite      []Cite      `xml:"cite,omitempty"`
	EmptyLine []struct{}  `xml:"empty-line,omitempty"`
}

// Title represents a title
type Title struct {
	Paragraphs []Paragraph `xml:"p"`
	EmptyLine  []struct{}  `xml:"empty-line,omitempty"`
}

// Paragraph represents a text paragraph with mixed content
type Paragraph struct {
	ID      string        `xml:"id,attr,omitempty"`
	Style   string        `xml:"style,attr,omitempty"`
	Content []interface{} `xml:",any"`
	Text    string        `xml:",chardata"`
}

// Emphasis represents emphasized text
type Emphasis struct {
	Style string `xml:"style,attr,omitempty"`
	Text  string `xml:",chardata"`
}

// Strong represents strong text
type Strong struct {
	Text string `xml:",chardata"`
}

// Epigraph represents an epigraph
type Epigraph struct {
	Paragraph  []Paragraph `xml:"p"`
	Poem       []Poem      `xml:"poem,omitempty"`
	Cite       []Cite      `xml:"cite,omitempty"`
	TextAuthor []string    `xml:"text-author,omitempty"`
}

// Poem represents a poem
type Poem struct {
	Title    Title      `xml:"title,omitempty"`
	Epigraph []Epigraph `xml:"epigraph,omitempty"`
	Stanza   []Stanza   `xml:"stanza"`
}

// Stanza represents a poem stanza
type Stanza struct {
	Title   Title  `xml:"title,omitempty"`
	Subtitle string `xml:"subtitle,omitempty"`
	V       []V    `xml:"v"`
}

// V represents a verse line
type V struct {
	Text string `xml:",chardata"`
}

// Cite represents a citation
type Cite struct {
	Paragraph  []Paragraph `xml:"p"`
	Subtitle   []string    `xml:"subtitle,omitempty"`
	Poem       []Poem      `xml:"poem,omitempty"`
	EmptyLine  []struct{}  `xml:"empty-line,omitempty"`
	TextAuthor []string    `xml:"text-author,omitempty"`
}

// Binary represents embedded binary data (images)
type Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"`
}

// Parser handles FB2 parsing and writing
type Parser struct{}

// NewParser creates a new FB2 parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads and parses an FB2 file
func (p *Parser) Parse(filename string) (*FictionBook, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.ParseReader(file)
}

// ParseReader parses FB2 from an io.Reader
func (p *Parser) ParseReader(reader io.Reader) (*FictionBook, error) {
	var fb FictionBook
	decoder := xml.NewDecoder(reader)

	if err := decoder.Decode(&fb); err != nil {
		return nil, fmt.Errorf("failed to parse FB2: %w", err)
	}

	return &fb, nil
}

// Write writes an FB2 structure to a file
func (p *Parser) Write(filename string, fb *FictionBook) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return p.WriteToWriter(file, fb)
}

// WriteToWriter writes FB2 to an io.Writer
func (p *Parser) WriteToWriter(writer io.Writer, fb *FictionBook) error {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// Write XML header
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}

	if err := encoder.Encode(fb); err != nil {
		return fmt.Errorf("failed to encode FB2: %w", err)
	}

	return nil
}

// GetLanguage returns the document language
func (fb *FictionBook) GetLanguage() string {
	return fb.Description.TitleInfo.Lang
}

// SetLanguage sets the document language
func (fb *FictionBook) SetLanguage(lang string) {
	fb.Description.TitleInfo.Lang = lang
}

// GetTitle returns the book title
func (fb *FictionBook) GetTitle() string {
	return fb.Description.TitleInfo.BookTitle
}

// SetTitle sets the book title
func (fb *FictionBook) SetTitle(title string) {
	fb.Description.TitleInfo.BookTitle = title
}
