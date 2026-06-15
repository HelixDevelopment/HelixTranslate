package ebook

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"strings"
)

// PDFWriter writes books to PDF (.pdf) format.
//
// It renders the in-memory Book to a UTF-8 HTML5 document (reusing the same
// title→heading / blank-line→paragraph mapping as the FB2/DOCX/HTML writers)
// and hands that HTML to the already-installed `weasyprint` to produce a VALID
// PDF. This is the same translate→HTML→PDF pipeline the project already uses for
// its documentation exports (§11.4.65).
//
// Why weasyprint and not a pure-Go PDF emitter: this project's PRIMARY content
// is Serbian, including Serbian Cyrillic. The PDF Standard-14 base fonts
// (Helvetica/Times-Roman) are restricted to WinAnsiEncoding and CANNOT render
// Cyrillic — a hand-rolled minimal PDF using them would SILENTLY DROP the
// translated Cyrillic glyphs, a §11.4 PASS-bluff. weasyprint embeds the needed
// font glyphs automatically, so the translated text is real, visible, and
// text-extractable for every script the translator produces.
//
// When weasyprint is NOT available, Write returns a typed
// ErrWeasyPrintUnavailable (an honest, actionable error per §11.4.3/§11.4.6)
// rather than producing a broken/blank/misnamed artifact (§11.4.1).
//
// Design rationale + the deep multi-angle research (§11.4.150) backing the
// weasyprint-over-pure-Go choice, including the Cyrillic correctness constraint:
// docs/research/20260615_video_surfaced_fixes/pdf_output_writer.md
type PDFWriter struct {
	// binary is the weasyprint executable name/path; overridable for testing.
	binary string
}

// NewPDFWriter creates a new PDF writer backed by weasyprint.
func NewPDFWriter() *PDFWriter {
	return &PDFWriter{binary: "weasyprint"}
}

// ErrWeasyPrintUnavailable is returned by Write when the weasyprint executable
// cannot be found on PATH. It is an explicit, honest failure (§11.4.3/§11.4.6):
// the caller knows EXACTLY why no PDF was produced and what to install, instead
// of receiving a silently-broken or empty file.
var ErrWeasyPrintUnavailable = errors.New(
	"weasyprint not found on PATH: PDF output requires weasyprint " +
		"(install via `pip install weasyprint` or `brew install weasyprint`)")

// Write writes a book to .pdf format at filename.
//
// It renders the book to HTML then invokes weasyprint to produce the PDF. If
// weasyprint is unavailable it returns ErrWeasyPrintUnavailable and writes
// nothing — never a partial/blank PDF.
func (w *PDFWriter) Write(book *Book, filename string) error {
	binary := w.binary
	if binary == "" {
		binary = "weasyprint"
	}

	resolved, err := exec.LookPath(binary)
	if err != nil {
		return ErrWeasyPrintUnavailable
	}

	htmlDoc := buildPDFSourceHTML(book)

	// weasyprint reads HTML from stdin ("-") and writes the PDF to filename.
	cmd := exec.Command(resolved, "-", filename) //nolint:gosec // resolved is an absolute path from LookPath
	cmd.Stdin = strings.NewReader(htmlDoc)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Best-effort cleanup so a failed run never leaves a partial PDF behind.
		_ = os.Remove(filename)
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("weasyprint failed to produce PDF: %w: %s", err, msg)
		}
		return fmt.Errorf("weasyprint failed to produce PDF: %w", err)
	}

	// Verify the produced artifact is actually a PDF (defends against a
	// zero-exit-but-empty-output edge case — a §11.4 anti-bluff guard).
	if err := assertIsPDF(filename); err != nil {
		_ = os.Remove(filename)
		return err
	}
	return nil
}

// buildPDFSourceHTML renders the Book into a minimal, valid, UTF-8 HTML5 source
// document for weasyprint. The book title and every chapter/section title become
// heading elements; section content is split on blank lines into <p> paragraphs
// (mirroring the FB2/DOCX/HTML writers). ALL text is HTML-escaped so translated
// content can never inject markup, and the document is declared UTF-8 so every
// script (incl. Cyrillic) is carried losslessly to weasyprint.
func buildPDFSourceHTML(book *Book) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<title>")

	title := ""
	if book != nil {
		title = strings.TrimSpace(book.Metadata.Title)
	}
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title>\n</head>\n<body>\n")

	if title != "" {
		b.WriteString("<h1>")
		b.WriteString(html.EscapeString(title))
		b.WriteString("</h1>\n")
	}

	if book != nil {
		for _, chapter := range book.Chapters {
			if t := strings.TrimSpace(chapter.Title); t != "" {
				b.WriteString("<h2>")
				b.WriteString(html.EscapeString(t))
				b.WriteString("</h2>\n")
			}
			for _, sec := range chapter.Sections {
				writePDFSection(&b, sec)
			}
		}
	}

	b.WriteString("</body>\n</html>\n")
	return b.String()
}

// writePDFSection emits a section and ALL of its subsections at ANY depth, in
// document order — mirroring the FB2/DOCX writers' lossless recursion so
// deeply-nested translated text is never dropped from the PDF.
func writePDFSection(b *strings.Builder, sec Section) {
	if t := strings.TrimSpace(sec.Title); t != "" {
		b.WriteString("<h3>")
		b.WriteString(html.EscapeString(t))
		b.WriteString("</h3>\n")
	}
	for _, p := range splitIntoParagraphs(sec.Content) {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(p))
		b.WriteString("</p>\n")
	}
	for _, sub := range sec.Subsections {
		writePDFSection(b, sub)
	}
}

// assertIsPDF confirms filename exists and begins with the %PDF- magic header.
func assertIsPDF(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("PDF output not created: %w", err)
	}
	defer f.Close()

	header := make([]byte, 5)
	n, _ := f.Read(header)
	if n < 5 || string(header[:5]) != "%PDF-" {
		return fmt.Errorf("produced file is not a valid PDF (missing %%PDF- header)")
	}
	return nil
}
