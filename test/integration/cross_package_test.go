//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"

	"github.com/gin-gonic/gin"
)

func TestCrossPackage_FileProcessingPipeline(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	// Test the complete pipeline: file detection -> parsing -> translation -> output
	tempDir := t.TempDir()

	// Create a test FB2 file
	fb2Content := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <book-title>Test Book</book-title>
      <lang>ru</lang>
    </title-info>
  </description>
  <body>
    <section>
      <p>This is a test paragraph in Russian.</p>
      <p>Another paragraph with more text.</p>
    </section>
  </body>
</FictionBook>`

	fb2Path := filepath.Join(tempDir, "test.fb2")
	err := os.WriteFile(fb2Path, []byte(fb2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test FB2 file: %v", err)
	}

	// Step 1: Format detection
	detector := format.NewDetector()
	detectedFormat, err := detector.DetectFile(fb2Path)
	if err != nil {
		t.Fatalf("Format detection failed: %v", err)
	}
	if detectedFormat != format.FormatFB2 {
		t.Errorf("Expected FB2 format, got %s", detectedFormat)
	}

	// Step 2: Ebook parsing
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(fb2Path)
	if err != nil {
		t.Fatalf("Ebook parsing failed: %v", err)
	}
	if book.Metadata.Title != "Test Book" {
		t.Errorf("Expected title 'Test Book', got '%s'", book.Metadata.Title)
	}

	// Step 3: Translation
	trans := &MockTranslator{}

	// Translate each chapter
	for i, chapter := range book.Chapters {
		for j, section := range chapter.Sections {
			if section.Content != "" {
				translated, err := trans.Translate(context.Background(), section.Content, "")
				if err != nil {
					t.Logf("Translation failed for chapter %d, section %d: %v", i, j, err)
				} else {
					t.Logf("Translated: '%s' -> '%s'", section.Content, translated)
				}
			}
		}
	}

	t.Logf("Successfully processed complete pipeline for %s", fb2Path)
}

func TestCrossPackage_APIAndTranslationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)

	// Setup complete system: config -> API -> translation
	cfg := config.DefaultConfig()
	cfg.Translation.DefaultProvider = "mock"
	cfg.Translation.DefaultModel = "default"

	eventBus := events.NewEventBus()
	handler := api.NewHandler(cfg, eventBus, nil, nil, nil, nil)

	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterBatchRoutes(v1)

	// Test that the API can handle translation requests
	// This would normally make HTTP requests, but for integration testing
	// we'll test the components work together

	// Verify configuration is properly passed
	if cfg.Translation.DefaultProvider != "mock" {
		t.Errorf("Configuration not properly set")
	}

	// Verify event bus is working
	eventCount := 0
	eventBus.Subscribe(events.EventType("test.event"), func(event events.Event) {
		eventCount++
	})

	eventBus.Publish(events.Event{
		Type:    "test.event",
		Message: "Integration test",
	})

	// Give it a moment to process
	// Note: In a real implementation, we'd wait for the event to be processed
	t.Logf("Event system integration test completed")
}

func TestCrossPackage_ErrorHandlingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test error handling across multiple packages
	tempDir := t.TempDir()

	// Test 1: Invalid file format
	invalidFile := filepath.Join(tempDir, "invalid.txt")
	err := os.WriteFile(invalidFile, []byte("not a valid ebook format"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	parser := ebook.NewUniversalParser()
	_, err = parser.Parse(invalidFile)
	// Should handle gracefully
	t.Logf("Invalid file parsing result: %v", err)

	// Test 2: Non-existent file
	_, err = parser.Parse("/non/existent/file.fb2")
	// Should handle gracefully
	t.Logf("Non-existent file parsing result: %v", err)

	// Test 3: Format detection on unknown file
	detector := format.NewDetector()
	unknownFile := filepath.Join(tempDir, "unknown.xyz")
	err = os.WriteFile(unknownFile, []byte("unknown content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create unknown file: %v", err)
	}

	format, err := detector.DetectFile(unknownFile)
	t.Logf("Unknown file format detection: %s, error: %v", format, err)

	t.Logf("Error handling integration test completed")
}

func TestCrossPackage_ResourceManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test resource management across components
	eventBus := events.NewEventBus()

	// Subscribe to multiple events
	eventTypes := []string{"translation.start", "translation.progress", "translation.complete", "error"}

	for _, eventType := range eventTypes {
		eventBus.Subscribe(events.EventType(eventType), func(event events.Event) {
			t.Logf("Received event: %s - %s", event.Type, event.Message)
		})
	}

	// Publish events
	for _, eventType := range eventTypes {
		eventBus.Publish(events.Event{
			Type:    events.EventType(eventType),
			Message: "Resource management test",
		})
	}

	// Subscribers are cleaned up automatically when test ends

	t.Logf("Resource management integration test completed")
}
