package ebook

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// DOCXWriter writes books to Office Open XML WordprocessingML (.docx) format.
//
// It produces a VALID .docx using only the Go standard library (archive/zip +
// encoding/xml) — no external process (pandoc) and no third-party dependency.
// The output carries the ISO/IEC 29500 minimal valid part set:
//
//	[Content_Types].xml
//	_rels/.rels
//	word/document.xml
//	word/_rels/document.xml.rels
//
// The body maps the in-memory Book to WordprocessingML block-level structures:
// the book title and every chapter/section title become heading paragraphs
// (bold + larger), and section content is split on blank lines into separate
// paragraphs (mirroring the FB2/HTML writers). All text is carried as <w:t>
// chardata via encoding/xml, so XML special characters in translated content are
// escaped automatically and can never inject markup.
//
// Design rationale + the deep multi-angle research (§11.4.150) backing the
// pure-Go-over-pandoc choice: docs/research/20260615_video_surfaced_fixes/docx_output_writer.md
type DOCXWriter struct{}

// NewDOCXWriter creates a new DOCX writer.
func NewDOCXWriter() *DOCXWriter {
	return &DOCXWriter{}
}

const (
	wordprocessingmlNS  = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	contentTypesPart    = "[Content_Types].xml"
	rootRelsPart        = "_rels/.rels"
	documentPart        = "word/document.xml"
	documentRelsPart    = "word/_rels/document.xml.rels"
	mainDocContentType  = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"
	officeDocumentRelNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument"
)

// Write writes a book to .docx format at filename.
func (w *DOCXWriter) Write(book *Book, filename string) error {
	docXML, err := buildDocumentXML(book)
	if err != nil {
		return fmt.Errorf("failed to build document.xml: %w", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	parts := []struct {
		name string
		data []byte
	}{
		{contentTypesPart, []byte(contentTypesXML)},
		{rootRelsPart, []byte(rootRelsXML)},
		{documentRelsPart, []byte(documentRelsXML)},
		{documentPart, docXML},
	}
	for _, p := range parts {
		f, err := zw.Create(p.name)
		if err != nil {
			return fmt.Errorf("failed to create zip part %q: %w", p.name, err)
		}
		if _, err := f.Write(p.data); err != nil {
			return fmt.Errorf("failed to write zip part %q: %w", p.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("failed to finalize docx zip: %w", err)
	}

	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write DOCX file: %w", err)
	}
	return nil
}

// buildDocumentXML marshals the Book into a WordprocessingML word/document.xml.
func buildDocumentXML(book *Book) ([]byte, error) {
	doc := wDocument{Xmlns: wordprocessingmlNS}

	if t := strings.TrimSpace(book.Metadata.Title); t != "" {
		doc.Body.Paragraphs = append(doc.Body.Paragraphs, headingParagraph(t))
	}

	for _, chapter := range book.Chapters {
		if t := strings.TrimSpace(chapter.Title); t != "" {
			doc.Body.Paragraphs = append(doc.Body.Paragraphs, headingParagraph(t))
		}
		for _, sec := range chapter.Sections {
			doc.Body.Paragraphs = append(doc.Body.Paragraphs, docxSectionParagraphs(sec)...)
		}
	}

	// A valid minimal document body must contain at least one block-level
	// element — emit an empty paragraph for a content-less book.
	if len(doc.Body.Paragraphs) == 0 {
		doc.Body.Paragraphs = append(doc.Body.Paragraphs, wParagraph{})
	}

	out, err := xml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	header := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	return append(header, out...), nil
}

// docxSectionParagraphs gathers the WordprocessingML paragraphs for a section and
// ALL of its subsections at ANY depth, in document order — mirroring the FB2
// writer's lossless recursion so deeply-nested translated text is never dropped.
func docxSectionParagraphs(sec Section) []wParagraph {
	var paras []wParagraph
	if t := strings.TrimSpace(sec.Title); t != "" {
		paras = append(paras, headingParagraph(t))
	}
	for _, p := range splitIntoParagraphs(sec.Content) {
		paras = append(paras, bodyParagraph(p))
	}
	for _, sub := range sec.Subsections {
		paras = append(paras, docxSectionParagraphs(sub)...)
	}
	return paras
}

// headingParagraph renders text as a bold, slightly larger paragraph (a simple
// heading style that needs no separate styles.xml part).
func headingParagraph(text string) wParagraph {
	return wParagraph{
		Runs: []wRun{{
			Props: &wRunProps{Bold: &xmlEmpty{}, Size: &wSize{Val: "32"}},
			Text:  wText{Space: "preserve", Value: text},
		}},
	}
}

// bodyParagraph renders text as a normal paragraph.
func bodyParagraph(text string) wParagraph {
	return wParagraph{
		Runs: []wRun{{Text: wText{Space: "preserve", Value: text}}},
	}
}

// WordprocessingML document.xml structures.
type wDocument struct {
	XMLName xml.Name `xml:"w:document"`
	Xmlns   string   `xml:"xmlns:w,attr"`
	Body    wBody    `xml:"w:body"`
}

type wBody struct {
	Paragraphs []wParagraph `xml:"w:p"`
}

type wParagraph struct {
	Runs []wRun `xml:"w:r"`
}

type wRun struct {
	Props *wRunProps `xml:"w:rPr,omitempty"`
	Text  wText      `xml:"w:t"`
}

type wRunProps struct {
	Bold *xmlEmpty `xml:"w:b,omitempty"`
	Size *wSize    `xml:"w:sz,omitempty"`
}

type wSize struct {
	Val string `xml:"w:val,attr"`
}

type wText struct {
	Space string `xml:"xml:space,attr,omitempty"`
	Value string `xml:",chardata"`
}

type xmlEmpty struct{}

// Static OOXML package parts.
const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="` + mainDocContentType + `"/>
</Types>`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="` + officeDocumentRelNS + `" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
