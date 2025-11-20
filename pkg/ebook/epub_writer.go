package ebook

import (
	"archive/zip"
	"fmt"
	"os"
	"strings"
	"time"
)

// EPUBWriter writes books to EPUB format
type EPUBWriter struct{}

// NewEPUBWriter creates a new EPUB writer
func NewEPUBWriter() *EPUBWriter {
	return &EPUBWriter{}
}

// Write writes a book to EPUB format
func (w *EPUBWriter) Write(book *Book, filename string) error {
	// Create EPUB file (ZIP)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Write mimetype (must be first, uncompressed)
	if err := w.writeMimetype(zipWriter); err != nil {
		return err
	}

	// Write META-INF/container.xml
	if err := w.writeContainer(zipWriter); err != nil {
		return err
	}

	// Write content.opf
	if err := w.writeContentOPF(zipWriter, book); err != nil {
		return err
	}

	// Write toc.ncx
	if err := w.writeTOC(zipWriter, book); err != nil {
		return err
	}

	// Write cover image if present
	if len(book.Metadata.Cover) > 0 {
		if err := w.writeCover(zipWriter, book.Metadata.Cover); err != nil {
			return err
		}
	}

	// Write chapters
	if err := w.writeChapters(zipWriter, book); err != nil {
		return err
	}

	return nil
}

// writeMimetype writes the mimetype file
func (w *EPUBWriter) writeMimetype(zw *zip.Writer) error {
	writer, err := zw.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte("application/epub+zip"))
	return err
}

// writeContainer writes META-INF/container.xml
func (w *EPUBWriter) writeContainer(zw *zip.Writer) error {
	writer, err := zw.Create("META-INF/container.xml")
	if err != nil {
		return err
	}

	container := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	_, err = writer.Write([]byte(container))
	return err
}

// writeContentOPF writes OEBPS/content.opf
func (w *EPUBWriter) writeContentOPF(zw *zip.Writer, book *Book) error {
	writer, err := zw.Create("OEBPS/content.opf")
	if err != nil {
		return err
	}

	// Build manifest and spine
	var manifest strings.Builder
	var spine strings.Builder

	// Add cover to manifest if present
	hasCover := len(book.Metadata.Cover) > 0
	if hasCover {
		manifest.WriteString(`    <item id="cover-image" href="cover.jpg" media-type="image/jpeg" properties="cover-image"/>` + "\n")
	}

	for i := range book.Chapters {
		id := fmt.Sprintf("chapter%d", i+1)
		href := fmt.Sprintf("chapter%d.xhtml", i+1)

		manifest.WriteString(fmt.Sprintf(`    <item id="%s" href="%s" media-type="application/xhtml+xml"/>%s`,
			id, href, "\n"))

		spine.WriteString(fmt.Sprintf(`    <itemref idref="%s"/>%s`, id, "\n"))
	}

	// Add NCX to manifest
	manifest.WriteString(`    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>` + "\n")

	authors := "Unknown"
	if len(book.Metadata.Authors) > 0 {
		authors = strings.Join(book.Metadata.Authors, ", ")
	}

	language := book.Metadata.Language
	if language == "" {
		language = "en"
	}

	// Use book date if available, otherwise current date
	date := book.Metadata.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Use book ISBN if available, otherwise generate UUID
	identifier := book.Metadata.ISBN
	if identifier == "" {
		identifier = "urn:uuid:" + generateUUID()
	}

	// Build metadata section
	var metadataSection strings.Builder
	metadataSection.WriteString(fmt.Sprintf("    <dc:title>%s</dc:title>\n", escapeXML(book.Metadata.Title)))
	metadataSection.WriteString(fmt.Sprintf("    <dc:creator>%s</dc:creator>\n", escapeXML(authors)))
	metadataSection.WriteString(fmt.Sprintf("    <dc:language>%s</dc:language>\n", language))
	metadataSection.WriteString(fmt.Sprintf("    <dc:identifier id=\"BookID\">%s</dc:identifier>\n", escapeXML(identifier)))
	metadataSection.WriteString(fmt.Sprintf("    <dc:date>%s</dc:date>\n", date))

	// Add description if available
	if book.Metadata.Description != "" {
		metadataSection.WriteString(fmt.Sprintf("    <dc:description>%s</dc:description>\n", escapeXML(book.Metadata.Description)))
	}

	// Add publisher if available
	if book.Metadata.Publisher != "" {
		metadataSection.WriteString(fmt.Sprintf("    <dc:publisher>%s</dc:publisher>\n", escapeXML(book.Metadata.Publisher)))
	}

	// Add cover meta tag if present
	if hasCover {
		metadataSection.WriteString(`    <meta name="cover" content="cover-image"/>` + "\n")
	}

	opf := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="BookID">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
%s  </metadata>
  <manifest>
%s  </manifest>
  <spine toc="ncx">
%s  </spine>
</package>`,
		metadataSection.String(),
		manifest.String(),
		spine.String())

	_, err = writer.Write([]byte(opf))
	return err
}

// writeTOC writes OEBPS/toc.ncx
func (w *EPUBWriter) writeTOC(zw *zip.Writer, book *Book) error {
	writer, err := zw.Create("OEBPS/toc.ncx")
	if err != nil {
		return err
	}

	var navMap strings.Builder
	for i, chapter := range book.Chapters {
		title := chapter.Title
		if title == "" {
			title = fmt.Sprintf("Chapter %d", i+1)
		}

		navMap.WriteString(fmt.Sprintf(`    <navPoint id="navPoint-%d" playOrder="%d">
      <navLabel>
        <text>%s</text>
      </navLabel>
      <content src="chapter%d.xhtml"/>
    </navPoint>
`, i+1, i+1, escapeXML(title), i+1))
	}

	ncx := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="urn:uuid:%s"/>
    <meta name="dtb:depth" content="1"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle>
    <text>%s</text>
  </docTitle>
  <navMap>
%s  </navMap>
</ncx>`,
		generateUUID(),
		escapeXML(book.Metadata.Title),
		navMap.String())

	_, err = writer.Write([]byte(ncx))
	return err
}

// writeChapters writes chapter XHTML files
func (w *EPUBWriter) writeChapters(zw *zip.Writer, book *Book) error {
	for i, chapter := range book.Chapters {
		filename := fmt.Sprintf("OEBPS/chapter%d.xhtml", i+1)
		writer, err := zw.Create(filename)
		if err != nil {
			return err
		}

		title := chapter.Title
		if title == "" {
			title = fmt.Sprintf("Chapter %d", i+1)
		}

		var content strings.Builder
		for _, section := range chapter.Sections {
			content.WriteString(w.formatSection(&section))
		}

		xhtml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>%s</title>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
</head>
<body>
  <h1>%s</h1>
%s
</body>
</html>`,
			escapeXML(title),
			escapeXML(title),
			content.String())

		if _, err := writer.Write([]byte(xhtml)); err != nil {
			return err
		}
	}

	return nil
}

// formatSection formats a section as HTML
func (w *EPUBWriter) formatSection(section *Section) string {
	var sb strings.Builder

	if section.Title != "" {
		sb.WriteString(fmt.Sprintf("  <h2>%s</h2>\n", escapeXML(section.Title)))
	}

	// Split content into paragraphs
	paragraphs := strings.Split(section.Content, "\n\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			sb.WriteString(fmt.Sprintf("  <p>%s</p>\n", escapeXML(para)))
		}
	}

	// Process subsections
	for _, subsection := range section.Subsections {
		sb.WriteString(w.formatSection(&subsection))
	}

	return sb.String()
}

// writeCover writes the cover image file
func (w *EPUBWriter) writeCover(zw *zip.Writer, coverData []byte) error {
	writer, err := zw.Create("OEBPS/cover.jpg")
	if err != nil {
		return err
	}

	_, err = writer.Write(coverData)
	return err
}

// escapeXML escapes XML special characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// generateUUID generates a simple UUID
func generateUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
