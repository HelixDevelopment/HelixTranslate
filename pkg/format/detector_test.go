package format

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

func TestDetectFileFB2(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a test FB2 file
	fb2Content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Test</book-title>
    </title-info>
  </description>
</FictionBook>`

	filename := filepath.Join(tempDir, "test.fb2")
	err := os.WriteFile(filename, []byte(fb2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatFB2 {
		t.Errorf("Expected FormatFB2, got %s", format)
	}
}

func TestDetectFileEPUB(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a test EPUB file (ZIP with PK magic bytes)
	epubContent := "PK\x03\x04\x14\x00\x00\x00\x00\x00\x8d\x8f\x8bN\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0e\x00\x00\x00mimetypeapplication/epub+zip"

	filename := filepath.Join(tempDir, "test.epub")
	err := os.WriteFile(filename, []byte(epubContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatEPUB {
		t.Errorf("Expected FormatEPUB, got %s", format)
	}
}

func TestDetectFilePDF(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a test PDF file
	pdfContent := "%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n"

	filename := filepath.Join(tempDir, "test.pdf")
	err := os.WriteFile(filename, []byte(pdfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatPDF {
		t.Errorf("Expected FormatPDF, got %s", format)
	}
}

func TestDetectFileTXT(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a test TXT file
	txtContent := "This is a plain text file.\nIt contains readable text.\n"

	filename := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(filename, []byte(txtContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatTXT {
		t.Errorf("Expected FormatTXT, got %s", format)
	}
}

func TestDetectFileHTML(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a test HTML file
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><p>This is HTML content.</p></body>
</html>`

	filename := filepath.Join(tempDir, "test.html")
	err := os.WriteFile(filename, []byte(htmlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatHTML {
		t.Errorf("Expected FormatHTML, got %s", format)
	}
}

func TestDetectFileRTF(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a test RTF file
	rtfContent := `{\rtf1\ansi\ansicpg1252\deff0\deflang1033{\fonttbl{\f0\fnil\fcharset0 Arial;}}\viewkind4\uc1\pard\f0\fs20 This is RTF content.\par}`

	filename := filepath.Join(tempDir, "test.rtf")
	err := os.WriteFile(filename, []byte(rtfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatRTF {
		t.Errorf("Expected FormatRTF, got %s", format)
	}
}

func TestDetectFileUnknown(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a file with unknown content
	unknownContent := "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\x0C\x0D\x0E\x0F"

	filename := filepath.Join(tempDir, "test.unknown")
	err := os.WriteFile(filename, []byte(unknownContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	if format != FormatUnknown {
		t.Errorf("Expected FormatUnknown, got %s", format)
	}
}

func TestDetectFileNonexistent(t *testing.T) {
	detector := NewDetector()

	_, err := detector.DetectFile("nonexistent.file")
	if err == nil {
		t.Error("DetectFile() should have failed with nonexistent file")
	}
}

func TestDetectByExtension(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		ext      string
		expected Format
	}{
		{".fb2", FormatFB2},
		{".epub", FormatEPUB},
		{".pdf", FormatPDF},
		{".mobi", FormatMOBI},
		{".azw", FormatAZW},
		{".azw3", FormatAZW3},
		{".txt", FormatTXT},
		{".html", FormatHTML},
		{".htm", FormatHTML},
		{".docx", FormatDOCX},
		{".rtf", FormatRTF},
		{".unknown", FormatUnknown},
		{"", FormatUnknown},
	}

	for _, test := range tests {
		result := detector.detectByExtension(test.ext)
		if result != test.expected {
			t.Errorf("detectByExtension(%s) = %s, expected %s", test.ext, result, test.expected)
		}
	}
}

func TestDetectByMagicBytes(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		data     []byte
		expected Format
	}{
		{[]byte("%PDF"), FormatPDF},
		{[]byte("PK\x03\x04"), FormatEPUB}, // EPUB uses PK (ZIP format)
		{[]byte("BOOKMOBI"), FormatMOBI},
		{[]byte("unknown"), FormatUnknown},
		{[]byte(""), FormatUnknown},
	}

	for _, test := range tests {
		result := detector.detectByMagicBytes(test.data)
		if result != test.expected {
			t.Errorf("detectByMagicBytes(%q) = %s, expected %s", test.data, result, test.expected)
		}
	}
}

func TestDisambiguateZipFormat(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a mock ZIP file
	zipContent := "PK\x03\x04\x14\x00\x00\x00\x00\x00\x8d\x8f\x8bN\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0e\x00\x00\x00mimetypeapplication/epub+zip"

	tests := []struct {
		filename string
		ext      string
		expected Format
	}{
		{filepath.Join(tempDir, "test.epub"), ".epub", FormatEPUB},
		{filepath.Join(tempDir, "test.docx"), ".docx", FormatDOCX},
		{filepath.Join(tempDir, "test.unknown"), ".zip", FormatEPUB}, // Default fallback
	}

	for _, test := range tests {
		// Create the file
		err := os.WriteFile(test.filename, []byte(zipContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result, err := detector.disambiguateZipFormat(test.filename, test.ext)
		if err != nil {
			t.Errorf("disambiguateZipFormat() failed: %v", err)
		}
		if result != test.expected {
			t.Errorf("disambiguateZipFormat(%s, %s) = %s, expected %s", test.filename, test.ext, result, test.expected)
		}
	}
}

func TestDetectByContent(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		content  string
		expected Format
	}{
		{`<?xml version="1.0"?><FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">`, FormatFB2},
		{`<!DOCTYPE html><html><body>`, FormatHTML},
		{`<html><body>`, FormatHTML},
		{`{\rtf1\ansi`, FormatRTF},
		{`This is plain text content.`, FormatTXT},
		{`Unknown content type.`, FormatTXT},              // Plain text content
		{string([]byte{0x00, 0x01, 0x02}), FormatUnknown}, // Binary data
	}

	for _, test := range tests {
		result := detector.detectByContent([]byte(test.content))
		if result != test.expected {
			t.Errorf("detectByContent(%q) = %s, expected %s", test.content, result, test.expected)
		}
	}
}

func TestIsPlainText(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte("This is plain text."), true},
		{[]byte("Текст на русском."), true},         // UTF-8 text
		{[]byte("\x00\x01\x02\x03\x04\x05"), false}, // Binary data
		{[]byte("Mixed text\x00binary"), true},      // Mostly readable text
		{[]byte(""), true},                          // Empty is considered plain text
	}

	for _, test := range tests {
		result := detector.isPlainText(test.data)
		if result != test.expected {
			t.Errorf("isPlainText(%q) = %t, expected %t", test.data, result, test.expected)
		}
	}
}

func TestIsSupported(t *testing.T) {
	detector := NewDetector()

	supportedFormats := []Format{
		FormatFB2,
		FormatEPUB,
		FormatTXT,
		FormatHTML,
		FormatDOCX, // §11.4.120: now supported via the stdlib OOXML parser
		FormatPDF,  // §11.4.120: now supported via the MIT ledongthuc/pdf text extractor
	}

	unsupportedFormats := []Format{
		FormatMOBI,
		FormatAZW,
		FormatAZW3,
		FormatRTF,
		FormatUnknown,
	}

	for _, format := range supportedFormats {
		if !detector.IsSupported(format) {
			t.Errorf("IsSupported(%s) should return true", format)
		}
	}

	for _, format := range unsupportedFormats {
		if detector.IsSupported(format) {
			t.Errorf("IsSupported(%s) should return false", format)
		}
	}
}

func TestGetSupportedFormats(t *testing.T) {
	detector := NewDetector()

	supported := detector.GetSupportedFormats()
	expected := []Format{FormatFB2, FormatEPUB, FormatTXT, FormatHTML, FormatDOCX, FormatPDF}

	if len(supported) != len(expected) {
		t.Errorf("GetSupportedFormats() returned %d formats, expected %d", len(supported), len(expected))
	}

	for i, format := range expected {
		if i >= len(supported) || supported[i] != format {
			t.Errorf("GetSupportedFormats()[%d] = %s, expected %s", i, supported[i], format)
		}
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		format   Format
		expected string
	}{
		{FormatFB2, "fb2"},
		{FormatEPUB, "epub"},
		{FormatPDF, "pdf"},
		{FormatTXT, "txt"},
		{FormatUnknown, "unknown"},
	}

	for _, test := range tests {
		if test.format.String() != test.expected {
			t.Errorf("Format(%s).String() = %s, expected %s", test.format, test.format.String(), test.expected)
		}
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
	}{
		{"fb2", FormatFB2},
		{"epub", FormatEPUB},
		{"pdf", FormatPDF},
		{"mobi", FormatMOBI},
		{"azw", FormatAZW},
		{"azw3", FormatAZW3},
		{"txt", FormatTXT},
		{"text", FormatTXT},
		{"html", FormatHTML},
		{"htm", FormatHTML},
		{"docx", FormatDOCX},
		{"rtf", FormatRTF},
		{"FB2", FormatFB2}, // Case insensitive
		{"EPUB", FormatEPUB},
		{"unknown", FormatUnknown},
		{"", FormatUnknown},
		{"invalid", FormatUnknown},
	}

	for _, test := range tests {
		result := ParseFormat(test.input)
		if result != test.expected {
			t.Errorf("ParseFormat(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestPriorityMagicBytesOverExtension(t *testing.T) {
	detector := NewDetector()
	tempDir := t.TempDir()

	// Create a file with PDF magic bytes but .txt extension
	pdfContent := "%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj\n"

	filename := filepath.Join(tempDir, "fake.txt") // Wrong extension
	err := os.WriteFile(filename, []byte(pdfContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	format, err := detector.DetectFile(filename)
	if err != nil {
		t.Fatalf("DetectFile() failed: %v", err)
	}

	// Should detect PDF by magic bytes, not TXT by extension
	if format != FormatPDF {
		t.Errorf("Expected FormatPDF (magic bytes priority), got %s", format)
	}
}
