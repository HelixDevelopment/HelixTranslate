package ebook

import (
	"context"
	"testing"
	"time"

	"digital.vasic.translator/pkg/format"
)

func TestNewDOCXParser(t *testing.T) {
	// Test with nil config
	parser := NewDOCXParser(nil)
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

func TestDOCXParser_GetFormat(t *testing.T) {
	parser := NewDOCXParser(nil)
	if parser.GetFormat() != format.FormatDOCX {
		t.Errorf("Expected format to be %s, got %s", format.FormatDOCX, parser.GetFormat())
	}
}

func TestDOCXParser_SupportedFormats(t *testing.T) {
	parser := NewDOCXParser(nil)
	formats := parser.SupportedFormats()

	expectedFormats := []string{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	for i, expected := range expectedFormats {
		if i >= len(formats) || formats[i] != expected {
			t.Errorf("Expected format %d to be %s, got %s", i, expected, formats[i])
		}
	}
}

func TestDOCXParser_Validate(t *testing.T) {
	parser := NewDOCXParser(nil)

	// Test with empty data
	err := parser.Validate([]byte{})
	if err == nil {
		t.Error("Expected validation to fail with empty data")
	}

	// Test with invalid data (not a DOCX)
	invalidData := []byte("not a docx file")
	err = parser.Validate(invalidData)
	if err == nil {
		t.Error("Expected validation to fail with invalid data")
	}
}

func TestDOCXParser_GetMetadata(t *testing.T) {
	parser := NewDOCXParser(nil)

	// Test with empty data
	_, err := parser.GetMetadata([]byte{})
	if err == nil {
		t.Error("Expected metadata extraction to fail with empty data")
	}

	// Test with invalid data (not a DOCX)
	invalidData := []byte("not a docx file")
	_, err = parser.GetMetadata(invalidData)
	if err == nil {
		t.Error("Expected metadata extraction to fail with invalid data")
	}
}

func TestDOCXParser_ParseWithContext(t *testing.T) {
	parser := NewDOCXParser(nil)
	ctx := context.Background()

	// Test with empty data
	_, err := parser.ParseWithContext(ctx, []byte{})
	if err == nil {
		t.Error("Expected parsing to fail with empty data")
	}

	// Test with invalid data (not a DOCX)
	invalidData := []byte("not a docx file")
	_, err = parser.ParseWithContext(ctx, invalidData)
	if err == nil {
		t.Error("Expected parsing to fail with invalid data")
	}
}

func TestDOCXParser_ContextCancellation(t *testing.T) {
	parser := NewDOCXParser(nil)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Use a VALID .docx so parsing reaches the streaming loop where context
	// cancellation is checked (the stdlib parser checks ctx during
	// word/document.xml token streaming). §11.4.120: the old assertion accepted a
	// "license" error from the now-removed unioffice dependency; the correct
	// post-rewrite behavior is a clean context.Canceled.
	data := buildTestDOCX(t, sampleDocumentXML, sampleCoreXML)
	_, err := parser.ParseWithContext(ctx, data)
	if err == nil {
		t.Error("Expected parsing to fail with cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestDOCXConfig_Defaults(t *testing.T) {
	parser := NewDOCXParser(nil)
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

func TestDOCXParser_WithConfig(t *testing.T) {
	config := &DOCXConfig{
		ExtractImages: false,
		ImageFormat:   "jpeg",
		ExtractTables: false,
		MinTextLength: 10,
		IgnoreStyles:  []string{"Title", "Heading"},
	}

	parser := NewDOCXParser(config)

	if parser.config.ExtractImages != false {
		t.Error("Expected ExtractImages to be false")
	}

	if parser.config.ImageFormat != "jpeg" {
		t.Errorf("Expected ImageFormat to be 'jpeg', got '%s'", parser.config.ImageFormat)
	}

	if parser.config.MinTextLength != 10 {
		t.Errorf("Expected MinTextLength to be 10, got %d", parser.config.MinTextLength)
	}

	if len(parser.config.IgnoreStyles) != 2 {
		t.Errorf("Expected 2 ignored styles, got %d", len(parser.config.IgnoreStyles))
	}
}

// Performance test
func TestDOCXParser_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")  // SKIP-OK: #short-mode
	}

	parser := NewDOCXParser(nil)
	ctx := context.Background()

	// Create a large but invalid DOCX data for performance testing
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	start := time.Now()
	_, err := parser.ParseWithContext(ctx, largeData)
	duration := time.Since(start)

	// Should fail quickly even with large invalid data
	if duration > 5*time.Second {
		t.Errorf("Parsing took too long: %v", duration)
	}

	if err == nil {
		t.Error("Expected parsing to fail with invalid data")
	}
}
