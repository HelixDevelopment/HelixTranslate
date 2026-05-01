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
			section.P = append(section.P, escapeFB2Text(sec.Content))
			for _, sub := range sec.Subsections {
				section.P = append(section.P, escapeFB2Text(sub.Content))
			}
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

// escapeFB2Text normalizes text for FB2 paragraphs.
// Note: XML special characters are escaped automatically by the XML encoder.
func escapeFB2Text(text string) string {
	// Replace multiple newlines with paragraph breaks
	text = strings.ReplaceAll(text, "\n\n", "</p>\n<p>")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
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
