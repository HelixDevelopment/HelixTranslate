package format

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Format represents an ebook format
type Format string

const (
	FormatFB2     Format = "fb2"
	FormatEPUB    Format = "epub"
	FormatPDF     Format = "pdf"
	FormatMOBI    Format = "mobi"
	FormatAZW     Format = "azw"
	FormatAZW3    Format = "azw3"
	FormatTXT     Format = "txt"
	FormatHTML    Format = "html"
	FormatDOCX    Format = "docx"
	FormatRTF     Format = "rtf"
	FormatUnknown Format = "unknown"
)

// Magic byte signatures for different formats
var magicBytes = map[Format][]byte{
	FormatPDF:  []byte("%PDF"),
	FormatEPUB: []byte("PK"), // EPUB is a ZIP file (DOCX and AZW3 also use PK but are handled by disambiguation)
	FormatMOBI: []byte("BOOKMOBI"),
	FormatAZW:  []byte("TPZ0"),
	// Note: AZW3 and DOCX removed since they also use PK magic bytes - handled by disambiguation
}

// Detector handles ebook format detection
type Detector struct{}

// NewDetector creates a new format detector
func NewDetector() *Detector {
	return &Detector{}
}

// DetectFile detects the format of a file
func (d *Detector) DetectFile(filename string) (Format, error) {
	file, err := os.Open(filename)
	if err != nil {
		return FormatUnknown, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes for magic byte detection
	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return FormatUnknown, fmt.Errorf("failed to read file header: %w", err)
	}
	header = header[:n]

	// Check file extension first
	ext := strings.ToLower(filepath.Ext(filename))
	formatByExt := d.detectByExtension(ext)

	// Check magic bytes
	formatByMagic := d.detectByMagicBytes(header)

	// Prioritize magic bytes over extension
	if formatByMagic != FormatUnknown {
		// Disambiguate ZIP-based formats (EPUB, DOCX both use PK magic bytes)
		if formatByMagic == FormatEPUB || formatByMagic == FormatDOCX {
			return d.disambiguateZipFormat(filename, ext)
		}
		return formatByMagic, nil
	}

	if formatByExt != FormatUnknown {
		return formatByExt, nil
	}

	// Try content-based detection
	return d.detectByContent(header), nil
}

// detectByExtension detects format by file extension
func (d *Detector) detectByExtension(ext string) Format {
	ext = strings.TrimPrefix(ext, ".")
	ext = strings.ToLower(ext)

	switch ext {
	case "fb2":
		return FormatFB2
	case "epub":
		return FormatEPUB
	case "pdf":
		return FormatPDF
	case "mobi", "prc":
		return FormatMOBI
	case "azw":
		return FormatAZW
	case "azw3":
		return FormatAZW3
	case "txt":
		return FormatTXT
	case "html", "htm":
		return FormatHTML
	case "docx":
		return FormatDOCX
	case "rtf":
		return FormatRTF
	default:
		return FormatUnknown
	}
}

// detectByMagicBytes detects format by magic bytes
func (d *Detector) detectByMagicBytes(data []byte) Format {
	for format, magic := range magicBytes {
		if bytes.HasPrefix(data, magic) {
			return format
		}
	}
	return FormatUnknown
}

// disambiguateZipFormat distinguishes between EPUB, DOCX, AZW3, and other ZIP formats
func (d *Detector) disambiguateZipFormat(filename string, ext string) (Format, error) {
	// Check extension first
	switch strings.ToLower(ext) {
	case ".epub":
		return FormatEPUB, nil
	case ".docx":
		return FormatDOCX, nil
	case ".azw3":
		return FormatAZW3, nil
	}

	// Check mimetype file inside ZIP
	mimetype, err := d.getZipMimetype(filename)
	if err == nil {
		switch mimetype {
		case "application/epub+zip":
			return FormatEPUB, nil
		case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			return FormatDOCX, nil
		case "application/x-mobipocket-ebook":
			return FormatAZW3, nil
		}
	}

	// Check for AZW3 specific structure ONLY if no specific mimetype found
	// This prevents EPUBs with application/epub+zip from being misidentified
	if d.isAZW3File(filename) {
		return FormatAZW3, nil
	}
	
	// Debug: Let's see what's happening
	// For ZIP files with PK magic but no specific AZW3 indicators, treat as EPUB
	// This catches cases where EPUB files are being incorrectly detected as AZW3

	// Default to EPUB for unknown ZIP formats
	return FormatEPUB, nil
}

// getZipMimetype reads the mimetype file from a ZIP archive
func (d *Detector) getZipMimetype(filename string) (string, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "mimetype" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}

			return strings.TrimSpace(string(data)), nil
		}
	}

	return "", fmt.Errorf("mimetype file not found")
}

// isAZW3File checks if the ZIP file has AZW3 specific structure
func (d *Detector) isAZW3File(filename string) bool {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return false
	}
	defer r.Close()

	// AZW3-specific indicators (only AZW3 has these)
	azw3Specific := []string{
		"kindle:embed",
		"amzn-eastock",
		"kindle-fonts",
		"kindle:enclosure",
		"kindle:meta",
	}

	hasSpecific := false
	
	for _, f := range r.File {
		// Check for AZW3-specific indicators
		for _, specific := range azw3Specific {
			if strings.Contains(f.Name, specific) {
				hasSpecific = true
				break
			}
		}
		if hasSpecific {
			break
		}
	}

	// Only consider it AZW3 if we find specific AZW3 indicators
	// EPUBs will never have these Kindle-specific files
	return hasSpecific
}

// detectByContent detects format by analyzing content
func (d *Detector) detectByContent(data []byte) Format {
	content := string(data)

	// Check for XML-based formats
	if strings.Contains(content, "<?xml") {
		if strings.Contains(content, "FictionBook") {
			return FormatFB2
		}
		if strings.Contains(content, "<html") || strings.Contains(content, "<HTML") {
			return FormatHTML
		}
	}

	// Check for HTML
	if strings.Contains(content, "<html") || strings.Contains(content, "<!DOCTYPE html") {
		return FormatHTML
	}

	// Check for RTF
	if strings.HasPrefix(content, "{\\rtf") {
		return FormatRTF
	}

	// Check for PDF content (if magic bytes were missed)
	if strings.Contains(content, "%PDF") {
		return FormatPDF
	}

	// Check for MOBI content (if magic bytes were missed)
	if strings.Contains(content, "BOOKMOBI") {
		return FormatMOBI
	}

	// Check for AZW content (if magic bytes were missed)
	if strings.Contains(content, "TPZ0") {
		return FormatAZW
	}

	// Default to plain text if mostly readable
	if d.isPlainText(data) {
		return FormatTXT
	}

	return FormatUnknown
}

// isPlainText checks if data is mostly plain text
func (d *Detector) isPlainText(data []byte) bool {
	if len(data) == 0 {
		return true // Empty data is considered plain text
	}
	printableCount := 0
	for _, b := range data {
		if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' || b >= 128 {
			printableCount++
		}
	}
	return float64(printableCount)/float64(len(data)) > 0.85
}

// IsSupported checks if a format is supported
func (d *Detector) IsSupported(format Format) bool {
	supported := []Format{
		FormatFB2,
		FormatEPUB,
		FormatTXT,
		FormatHTML,
	}

	for _, f := range supported {
		if f == format {
			return true
		}
	}
	return false
}

// GetSupportedFormats returns list of supported formats
func (d *Detector) GetSupportedFormats() []Format {
	return []Format{
		FormatFB2,
		FormatEPUB,
		FormatTXT,
		FormatHTML,
	}
}

// FormatToString converts Format to string
func (f Format) String() string {
	return string(f)
}

// ParseFormat parses a format string
func ParseFormat(s string) Format {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "fb2":
		return FormatFB2
	case "epub":
		return FormatEPUB
	case "pdf":
		return FormatPDF
	case "mobi":
		return FormatMOBI
	case "azw":
		return FormatAZW
	case "azw3":
		return FormatAZW3
	case "txt", "text":
		return FormatTXT
	case "html", "htm":
		return FormatHTML
	case "docx":
		return FormatDOCX
	case "rtf":
		return FormatRTF
	default:
		return FormatUnknown
	}
}
