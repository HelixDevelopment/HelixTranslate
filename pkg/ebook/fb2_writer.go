package ebook

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

// FB2Writer writes books to FictionBook 2.0 (FB2) format.
type FB2Writer struct{}

// NewFB2Writer creates a new FB2 writer.
func NewFB2Writer() *FB2Writer {
	return &FB2Writer{}
}

// Write writes a book to FB2 format.
func (w *FB2Writer) Write(book *Book, filename string) error {
	fb := fictionBook{
		XMLName: xml.Name{Local: "FictionBook"},
		Xmlns:   "http://www.gribuser.ru/xml/fictionbook/2.0",
		Description: description{
			TitleInfo: titleInfo{
				Genre:     "nonfiction",
				BookTitle: book.Metadata.Title,
				Lang:      book.Metadata.Language,
				Annotation: annotation{
					P: book.Metadata.Description,
				},
			},
			DocumentInfo: documentInfo{
				Author:    docAuthor{Nickname: "HelixTranslate"},
				ProgramUsed: "HelixTranslate",
				Date:      docDate{Value: time.Now().Format("2006-01-02"), Text: time.Now().Format("2006-01-02")},
			},
		},
		Body: body{
			Sections: make([]fbSection, 0, len(book.Chapters)),
		},
	}

	for _, author := range book.Metadata.Authors {
		parts := strings.SplitN(author, " ", 2)
		firstName := parts[0]
		lastName := ""
		if len(parts) > 1 {
			lastName = parts[1]
		}
		fb.Description.TitleInfo.Authors = append(fb.Description.TitleInfo.Authors, fbAuthor{
			FirstName: firstName,
			LastName:  lastName,
		})
	}

	for _, chapter := range book.Chapters {
		section := fbSection{
			Title: sectionTitle{P: chapter.Title},
		}
		for _, sec := range chapter.Sections {
			section.P = append(section.P, collectSectionParagraphs(sec)...)
		}
		fb.Body.Sections = append(fb.Body.Sections, section)
	}

	output, err := xml.MarshalIndent(fb, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal FB2: %w", err)
	}

	header := []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	data := append(header, output...)
	data = append(data, '\n')

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write FB2 file: %w", err)
	}

	return nil
}

// collectSectionParagraphs gathers the FB2 <p> paragraphs for a section AND all
// of its subsections at ANY depth, in document order. The previous writer only
// descended two levels (a section's content + its direct subsections' content),
// so any content nested deeper (a subsection's subsection and below) was silently
// dropped from the written FB2 file — real translated-text data loss. Recursing
// over the full Subsections tree (as the EPUB writer's formatSection already
// does) makes the write lossless for arbitrarily-nested content.
func collectSectionParagraphs(sec Section) []string {
	paragraphs := splitIntoParagraphs(sec.Content)
	for _, sub := range sec.Subsections {
		paragraphs = append(paragraphs, collectSectionParagraphs(sub)...)
	}
	return paragraphs
}

// splitIntoParagraphs splits section content into separate FB2 <p> paragraphs.
//
// Section content uses a blank line ("\n\n") as a paragraph separator. Each
// resulting paragraph becomes its own <p> element so the XML encoder emits real,
// well-formed <p>...</p> tags. The previous implementation injected raw
// "</p><p>" markup into a single chardata string, which the XML encoder then
// escaped to literal "&lt;/p&gt;&lt;p&gt;" text — producing one giant malformed
// paragraph with visible markup instead of separate paragraphs. XML special
// characters within each paragraph are escaped automatically by the encoder.
func splitIntoParagraphs(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	parts := strings.Split(text, "\n\n")
	paragraphs := make([]string, 0, len(parts))
	for _, part := range parts {
		// Collapse single newlines within a paragraph to spaces.
		p := strings.ReplaceAll(part, "\n", " ")
		p = strings.TrimSpace(p)
		if p != "" {
			paragraphs = append(paragraphs, p)
		}
	}
	return paragraphs
}

// FB2 XML structures.
type fictionBook struct {
	XMLName     xml.Name    `xml:"FictionBook"`
	Xmlns       string      `xml:"xmlns,attr"`
	Description description `xml:"description"`
	Body        body        `xml:"body"`
}

type description struct {
	TitleInfo    titleInfo    `xml:"title-info"`
	DocumentInfo documentInfo `xml:"document-info"`
}

type titleInfo struct {
	Genre      string      `xml:"genre"`
	Authors    []fbAuthor  `xml:"author"`
	BookTitle  string      `xml:"book-title"`
	Lang       string      `xml:"lang"`
	Annotation annotation  `xml:"annotation"`
}

type fbAuthor struct {
	FirstName string `xml:"first-name"`
	LastName  string `xml:"last-name"`
}

type annotation struct {
	P string `xml:"p"`
}

type documentInfo struct {
	Author      docAuthor `xml:"author"`
	ProgramUsed string    `xml:"program-used"`
	Date        docDate   `xml:"date"`
}

type docAuthor struct {
	Nickname string `xml:"nickname"`
}

type docDate struct {
	Value string `xml:"value,attr"`
	Text  string `xml:",chardata"`
}

type body struct {
	Sections []fbSection `xml:"section"`
}

type fbSection struct {
	Title sectionTitle `xml:"title"`
	P     []string     `xml:"p"`
}

type sectionTitle struct {
	P string `xml:"p"`
}
