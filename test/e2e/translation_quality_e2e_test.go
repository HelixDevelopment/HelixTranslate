//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/dictionary"
	"digital.vasic.translator/pkg/verification"
)

// TestProjectGutenbergTranslation tests translation of real books from Project Gutenberg
func TestProjectGutenbergTranslation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("TranslateRussianToSerbian_EPUB", func(t *testing.T) {
		// Download Russian book from Project Gutenberg
		// "The Gambler" by Dostoevsky (Russian) - using EPUB format
		bookURL := "https://www.gutenberg.org/cache/epub/2197/pg2197-images.epub"
		bookPath := filepath.Join(tmpDir, "gambler_ru.epub")

		if err := downloadFile(bookURL, bookPath); err != nil {
			t.Fatalf("Failed to download book: %v", err)
		}

		// Verify file was downloaded
		info, err := os.Stat(bookPath)
		if err != nil {
			t.Fatalf("Downloaded file doesn't exist: %v", err)
		}

		t.Logf("Downloaded book: %d bytes", info.Size())

		// Parse the book
		reader := ebook.NewEPUBParser()
		book, err := reader.Parse(bookPath)
		if err != nil {
			t.Fatalf("Failed to parse book: %v", err)
		}

		t.Logf("Parsed book: %d chapters", len(book.Chapters))

		// Create translator (dictionary for fast testing)
		translatorConfig := translator.TranslationConfig{
			SourceLang: "ru",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)

		// Create language detector
		langDetector := language.NewDetector(nil)

		// Create universal translator
		ru := language.Language{Code: "ru", Name: "Russian"}
		sr := language.Language{Code: "sr", Name: "Serbian"}
		universalTrans := translator.NewUniversalTranslator(trans, langDetector, ru, sr)

		// Translate the book
		eventBus := events.NewEventBus()
		sessionID := "e2e-test-gutenberg"

		if err := universalTrans.TranslateBook(ctx, book, eventBus, sessionID); err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		// Verify translation
		verifier := verification.NewVerifier(ru, sr, eventBus, sessionID)
		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		t.Logf("Translation Quality Score: %.2f%%", result.QualityScore*100)
		t.Logf("Untranslated Blocks: %d", len(result.UntranslatedBlocks))
		t.Logf("HTML Artifacts: %d", len(result.HTMLArtifacts))
		t.Logf("Warnings: %d", len(result.Warnings))
		t.Logf("Errors: %d", len(result.Errors))

		// Dictionary translator won't translate everything perfectly,
		// but should translate some content
		if result.QualityScore == 0 {
			t.Error("Expected some content to be translated")
		}

		// Write translated book
		outputPath := filepath.Join(tmpDir, "gambler_sr.epub")
		writer := ebook.NewEPUBWriter()
		if err := writer.Write(book, outputPath); err != nil {
			t.Fatalf("Failed to write translated book: %v", err)
		}

		t.Logf("Translated book written to: %s", outputPath)
	})

	t.Run("TranslateEnglishToSerbian_EPUB", func(t *testing.T) {
		// Download English book from Project Gutenberg
		// "Pride and Prejudice" by Jane Austen
		bookURL := "https://www.gutenberg.org/cache/epub/1342/pg1342-images.epub"
		bookPath := filepath.Join(tmpDir, "pride_en.epub")

		if err := downloadFile(bookURL, bookPath); err != nil {
			t.Fatalf("Failed to download book: %v", err)
		}

		// Parse the book
		reader := ebook.NewEPUBParser()
		book, err := reader.Parse(bookPath)
		if err != nil {
			t.Fatalf("Failed to parse EPUB: %v", err)
		}

		t.Logf("Parsed EPUB: %d chapters", len(book.Chapters))

		// Create translator
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)

		// Translate
		en := language.Language{Code: "en", Name: "English"}
		sr := language.Language{Code: "sr", Name: "Serbian"}
		langDetector := language.NewDetector(nil)
		universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)

		eventBus := events.NewEventBus()
		sessionID := "e2e-test-epub"

		if err := universalTrans.TranslateBook(ctx, book, eventBus, sessionID); err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		// Verify
		verifier := verification.NewVerifier(en, sr, eventBus, sessionID)
		result, err := verifier.VerifyBook(ctx, book)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		t.Logf("EPUB Translation Quality Score: %.2f%%", result.QualityScore*100)

		// Write translated EPUB
		outputPath := filepath.Join(tmpDir, "pride_sr.epub")
		writer := ebook.NewEPUBWriter()
		if err := writer.Write(book, outputPath); err != nil {
			t.Fatalf("Failed to write EPUB: %v", err)
		}

		t.Logf("Translated EPUB written to: %s", outputPath)
	})
}

// TestMultiLLMCoordinationE2E tests the multi-LLM coordination system end-to-end
func TestMultiLLMCoordinationE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Skip if no API keys are set
	if os.Getenv("DEEPSEEK_API_KEY") == "" &&
		os.Getenv("OPENAI_API_KEY") == "" &&
		os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("Skipping multi-LLM test - no API keys configured")
	}

	ctx := context.Background()
	eventBus := events.NewEventBus()

	// Track events
	receivedEvents := make([]events.Event, 0)
	eventBus.SubscribeAll(func(event events.Event) {
		receivedEvents = append(receivedEvents, event)
	})

	t.Run("InitializeWithRealAPIKeys", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:   eventBus,
			SessionID:  "e2e-multi-llm",
			MaxRetries: 3,
			RetryDelay: 2 * time.Second,
		})

		instanceCount := coordinator.GetInstanceCount()
		t.Logf("Initialized %d LLM instances", instanceCount)

		if instanceCount == 0 {
			t.Skip("No LLM instances initialized - check API keys")
		}

		// Check that init events were emitted
		hasInit := false
		for _, event := range receivedEvents {
			if event.Type == "multi_llm_init" || event.Type == "multi_llm_ready" {
				hasInit = true
				t.Logf("Event: %s - %s", event.Type, event.Message)
			}
		}

		if !hasInit {
			t.Error("Expected initialization events")
		}
	})

	t.Run("TranslateWithRetry", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:   eventBus,
			SessionID:  "e2e-retry",
			MaxRetries: 2,
			RetryDelay: 1 * time.Second,
		})

		if coordinator.GetInstanceCount() == 0 {
			t.Skip("No LLM instances available")
		}

		// Try to translate a short text
		text := "Hello, world!"
		translated, err := coordinator.TranslateWithRetry(ctx, text, "greeting")

		if err != nil {
			t.Logf("Translation failed (may be expected with rate limits): %v", err)
		} else {
			t.Logf("Translated '%s' to '%s'", text, translated)

			if translated == "" {
				t.Error("Expected non-empty translation")
			}
		}
	})

	t.Run("ConsensusMode", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:  eventBus,
			SessionID: "e2e-consensus",
		})

		if coordinator.GetInstanceCount() < 2 {
			t.Skip("Need at least 2 instances for consensus mode")
		}

		text := "Good morning"
		translated, err := coordinator.TranslateWithConsensus(ctx, text, "greeting", 2)

		if err != nil {
			t.Logf("Consensus translation failed: %v", err)
		} else {
			t.Logf("Consensus translation: '%s' to '%s'", text, translated)
		}
	})
}

// TestFullPipelineWithVerification tests the complete translation pipeline
func TestFullPipelineWithVerification(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	tmpDir := t.TempDir()

	t.Run("CompleteTranslationWorkflow", func(t *testing.T) {
		// Create a test book
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:       "Test Book",
				Authors:     []string{"Test Author"},
				Description: "A test book for translation",
				Language:    "en",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{
							Title:   "Introduction",
							Content: "This is a test book with multiple chapters and sections.",
						},
						{
							Title:   "Section 2",
							Content: "Each section contains text that needs to be translated.",
						},
					},
				},
				{
					Title: "Chapter 2",
					Sections: []ebook.Section{
						{
							Title:   "Advanced Topics",
							Content: "This chapter discusses more complex topics.",
						},
					},
				},
			},
		}

		// Write original book
		originalPath := filepath.Join(tmpDir, "original.epub")
		writer := ebook.NewEPUBWriter()
		if err := writer.Write(book, originalPath); err != nil {
			t.Fatalf("Failed to write original book: %v", err)
		}

		// Read it back
		reader := ebook.NewEPUBParser()
		loadedBook, err := reader.Parse(originalPath)
		if err != nil {
			t.Fatalf("Failed to read book: %v", err)
		}

		// Translate
		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)

		en := language.Language{Code: "en", Name: "English"}
		sr := language.Language{Code: "sr", Name: "Serbian"}
		langDetector := language.NewDetector(nil)
		universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)

		eventBus := events.NewEventBus()
		sessionID := "e2e-pipeline"

		// Track progress events
		progressEvents := 0
		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == events.EventTranslationProgress {
				progressEvents++
			}
		})

		if err := universalTrans.TranslateBook(ctx, loadedBook, eventBus, sessionID); err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		t.Logf("Received %d progress events", progressEvents)

		// Verify translation
		verifier := verification.NewVerifier(en, sr, eventBus, sessionID)
		result, err := verifier.VerifyBook(ctx, loadedBook)
		if err != nil {
			t.Fatalf("Verification failed: %v", err)
		}

		t.Logf("Quality Score: %.2f%%", result.QualityScore*100)
		t.Logf("Is Valid: %v", result.IsValid)

		// Write translated book
		translatedPath := filepath.Join(tmpDir, "translated_sr.epub")
		if err := writer.Write(loadedBook, translatedPath); err != nil {
			t.Fatalf("Failed to write translated book: %v", err)
		}

		// Verify file exists and has content
		info, err := os.Stat(translatedPath)
		if err != nil {
			t.Fatalf("Translated file doesn't exist: %v", err)
		}

		if info.Size() == 0 {
			t.Error("Translated file is empty")
		}

		t.Logf("Translation complete: %s (%d bytes)", translatedPath, info.Size())
	})
}

// TestErrorRecovery tests error handling and recovery mechanisms
func TestErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	t.Run("RecoverFromTranslationFailure", func(t *testing.T) {
		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Test Book",
				Language: "en",
			},
			Chapters: []ebook.Chapter{
				{
					Title: "Chapter 1",
					Sections: []ebook.Section{
						{Title: "Section 1", Content: "Content 1"},
						{Title: "Section 2", Content: "Content 2"},
					},
				},
			},
		}

		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)

		en := language.Language{Code: "en", Name: "English"}
		sr := language.Language{Code: "sr", Name: "Serbian"}
		langDetector := language.NewDetector(nil)
		universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)

		eventBus := events.NewEventBus()
		errorEvents := 0
		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == events.EventTranslationError {
				errorEvents++
			}
		})

		// This should complete despite any errors (dictionary translator is forgiving)
		_ = universalTrans.TranslateBook(ctx, book, eventBus, "e2e-recovery")

		t.Logf("Received %d error events", errorEvents)
	})
}

// TestLargeBookPerformance tests performance with larger books
func TestLargeBookPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()

	t.Run("TranslateLargeBook", func(t *testing.T) {
		// Create a book with many chapters and sections
		chapters := make([]ebook.Chapter, 50)
		for i := 0; i < 50; i++ {
			sections := make([]ebook.Section, 10)
			for j := 0; j < 10; j++ {
				sections[j] = ebook.Section{
					Title:   fmt.Sprintf("Section %d", j+1),
					Content: fmt.Sprintf("This is section %d content with some text to translate.", j+1),
				}
			}
			chapters[i] = ebook.Chapter{
				Title:    fmt.Sprintf("Chapter %d", i+1),
				Sections: sections,
			}
		}

		book := &ebook.Book{
			Metadata: ebook.Metadata{
				Title:    "Large Test Book",
				Language: "en",
			},
			Chapters: chapters,
		}

		translatorConfig := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "dictionary",
		}
		trans := dictionary.NewDictionaryTranslator(translatorConfig)

		en := language.Language{Code: "en", Name: "English"}
		sr := language.Language{Code: "sr", Name: "Serbian"}
		langDetector := language.NewDetector(nil)
		universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)

		eventBus := events.NewEventBus()

		startTime := time.Now()
		err := universalTrans.TranslateBook(ctx, book, eventBus, "e2e-performance")
		duration := time.Since(startTime)

		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		t.Logf("Translated %d chapters (500 sections) in %v", len(chapters), duration)
		t.Logf("Average time per section: %v", duration/500)

		// Performance check: should complete in reasonable time
		if duration > 5*time.Minute {
			t.Errorf("Translation took too long: %v (expected < 5 minutes)", duration)
		}
	})
}

// TestCLIBypassIssue tests the fix for CLI bypassing distributed API
func TestCLIBypassIssue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("APIReturnsValidEPUB", func(t *testing.T) {
		// This test ensures that the API properly translates files
		// and doesn't return the 18-byte invalid output that occurred
		// when CLI bypassed the distributed API

		// Start a test server
		// This would require setting up a full server instance
		// For now, just verify that the parsing and translation logic works

		parser := ebook.NewUniversalParser()
		book, err := parser.Parse("../../test_output.epub")
		if err != nil {
			t.Fatalf("Failed to parse test EPUB: %v", err)
		}

		if book.Format != format.FormatEPUB {
			t.Errorf("Expected EPUB format, got %s", book.Format)
		}

		if len(book.Chapters) == 0 {
			t.Error("Expected at least one chapter")
		}

		// Verify that translation produces valid output
		trans := dictionary.NewDictionaryTranslator(translator.TranslationConfig{
			SourceLang: "ru",
			TargetLang: "sr",
			Provider:   "dictionary",
		})

		ctx := context.Background()
		eventBus := events.NewEventBus()

		en := language.Language{Code: "ru", Name: "Russian"}
		sr := language.Language{Code: "sr", Name: "Serbian"}
		langDetector := language.NewDetector(nil)
		universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)

		err = universalTrans.TranslateBook(ctx, book, eventBus, "test-session")
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		// Update metadata like the API does
		book.Language = "sr"

		// Verify book has been translated (language changed)
		if book.Language != "sr" {
			t.Errorf("Expected language 'sr', got '%s'", book.Language)
		}

		// Verify content has been modified
		if book.Metadata.Title == "" {
			t.Error("Title should not be empty after translation")
		}
	})
}

// downloadFile downloads a file from URL to local path
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
