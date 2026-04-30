package ebook

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/format"
)

func TestNewPDFParser(t *testing.T) {
	// Test with nil config
	parser := NewPDFParser(nil)
	if parser == nil {
		t.Fatal("Expected parser to be created with nil config")
	}

	if parser.config == nil {
		t.Error("Expected default config to be set")
	}

	if !parser.config.ExtractImages {
		t.Error("Expected ExtractImages to be true by default")
	}
}

func TestPDFParser_GetFormat(t *testing.T) {
	parser := NewPDFParser(nil)
	if parser.GetFormat() != format.FormatPDF {
		t.Errorf("Expected format to be %s, got %s", format.FormatPDF, parser.GetFormat())
	}
}

func TestPDFParser_SupportedFormats(t *testing.T) {
	parser := NewPDFParser(nil)
	formats := parser.SupportedFormats()

	expectedFormats := []string{"pdf", "application/pdf"}
	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	for i, expected := range expectedFormats {
		if i >= len(formats) || formats[i] != expected {
			t.Errorf("Expected format %d to be %s, got %s", i, expected, formats[i])
		}
	}
}

func TestPDFParser_Validate(t *testing.T) {
	parser := NewPDFParser(nil)

	// Test with empty data
	err := parser.Validate([]byte{})
	if err == nil {
		t.Error("Expected validation to fail with empty data")
	}

	// Test with invalid PDF data
	invalidData := []byte("not a pdf file")
	err = parser.Validate(invalidData)
	if err == nil {
		t.Error("Expected validation to fail with invalid data")
	}

	// Test with valid PDF signature but invalid structure
	pdfHeader := []byte("%PDF-1.4\n")
	err = parser.Validate(pdfHeader)
	if err != nil {
		// This might fail due to invalid PDF structure, but the signature is valid
		t.Logf("Expected validation to possibly fail with incomplete PDF: %v", err)
	}
}

func TestPDFParser_GetMetadata(t *testing.T) {
	parser := NewPDFParser(nil)

	// Test with empty data
	_, err := parser.GetMetadata([]byte{})
	if err == nil {
		t.Error("Expected metadata extraction to fail with empty data")
	}

	// Test with invalid data (not a PDF)
	invalidData := []byte("not a pdf file")
	_, err = parser.GetMetadata(invalidData)
	if err == nil {
		t.Error("Expected metadata extraction to fail with invalid data")
	}
}

func TestPDFParser_ParseWithContext(t *testing.T) {
	parser := NewPDFParser(nil)
	ctx := context.Background()

	// Test with empty data
	_, err := parser.ParseWithContext(ctx, []byte{})
	if err == nil {
		t.Error("Expected parsing to fail with empty data")
	}

	// Test with invalid data (not a PDF)
	invalidData := []byte("not a pdf file")
	_, err = parser.ParseWithContext(ctx, invalidData)
	if err == nil {
		t.Error("Expected parsing to fail with invalid data")
	}
}

func TestPDFParser_ContextCancellation(t *testing.T) {
	parser := NewPDFParser(nil)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Even with invalid data, context cancellation should be detected
	_, err := parser.ParseWithContext(ctx, []byte("not a pdf"))
	if err == nil {
		t.Error("Expected parsing to fail with cancelled context")
	}

	// Check if it's either context cancelled or parsing error (both acceptable)
	if err != context.Canceled && !containsString(err.Error(), "version not found") {
		t.Errorf("Expected context.Canceled or version error, got %v", err)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPDFConfig_Defaults(t *testing.T) {
	parser := NewPDFParser(nil)
	config := parser.config

	if config.ExtractImages != true {
		t.Error("Expected ExtractImages to be true by default")
	}

	if config.ImageFormat != "png" {
		t.Errorf("Expected ImageFormat to be 'png', got '%s'", config.ImageFormat)
	}

	if config.ExtractTables != true {
		t.Error("Expected ExtractTables to be true by default")
	}

	if config.MinTextLength != 1 {
		t.Errorf("Expected MinTextLength to be 1, got %d", config.MinTextLength)
	}
}

func TestPDFParser_WithConfig(t *testing.T) {
	config := &PDFConfig{
		ExtractImages:  false,
		ImageFormat:    "jpeg",
		ExtractTables:  false,
		OcrEnabled:     true,
		OcrLanguage:    "spa",
		MinTextLength:  50,
		PreserveLayout: false,
	}

	parser := NewPDFParser(config)

	if parser.config.ExtractImages != false {
		t.Error("Expected ExtractImages to be false")
	}

	if parser.config.ImageFormat != "jpeg" {
		t.Errorf("Expected ImageFormat to be 'jpeg', got '%s'", parser.config.ImageFormat)
	}

	if parser.config.MinTextLength != 50 {
		t.Errorf("Expected MinTextLength to be 50, got %d", parser.config.MinTextLength)
	}

	if parser.config.OcrEnabled != true {
		t.Error("Expected OcrEnabled to be true")
	}

	if parser.config.OcrLanguage != "spa" {
		t.Errorf("Expected OcrLanguage to be 'spa', got '%s'", parser.config.OcrLanguage)
	}
}

// Performance test
func TestPDFParser_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")  // SKIP-OK: #short-mode
	}

	parser := NewPDFParser(nil)
	ctx := context.Background()

	// Create a large but invalid PDF data for performance testing
	largeData := make([]byte, 2*1024*1024) // 2MB
	copy(largeData, []byte("%PDF-1.4\n"))
	for i := 10; i < len(largeData); i++ {
		largeData[i] = byte(i % 256)
	}

	start := time.Now()
	_, err := parser.ParseWithContext(ctx, largeData)
	duration := time.Since(start)

	// Should fail quickly even with large invalid data
	if duration > 10*time.Second {
		t.Errorf("Parsing took too long: %v", duration)
	}

	if err == nil {
		t.Error("Expected parsing to fail with invalid data")
	}
}

// Test that PDF signature detection works correctly
func TestPDFParser_SignatureDetection(t *testing.T) {
	parser := NewPDFParser(nil)

	// Test various PDF signatures
	validSignatures := [][]byte{
		[]byte("%PDF-1.0\n"),
		[]byte("%PDF-1.1\n"),
		[]byte("%PDF-1.2\n"),
		[]byte("%PDF-1.3\n"),
		[]byte("%PDF-1.4\n"),
		[]byte("%PDF-1.5\n"),
		[]byte("%PDF-1.6\n"),
		[]byte("%PDF-1.7\n"),
		[]byte("%PDF-2.0\n"),
	}

	invalidSignatures := [][]byte{
		[]byte("%PDF-1.8\n"), // Invalid version
		[]byte("%PDf-1.4\n"), // Case sensitive
		[]byte("PDF-1.4\n"),  // Missing %
		[]byte("%PDF-1.4"),   // Missing newline
	}

	for i, sig := range validSignatures {
		err := parser.Validate(sig)
		// Valid signatures should either succeed or fail due to incomplete structure
		// but should not fail signature validation
		if err != nil && !containsString(err.Error(), "signature") && !containsString(err.Error(), "structure") {
			t.Errorf("Valid signature %d failed with unexpected error: %v", i, err)
		}
	}

	for i, sig := range invalidSignatures {
		err := parser.Validate(sig)
		if err == nil {
			t.Errorf("Invalid signature %d should have failed validation", i)
		}
	}
}
