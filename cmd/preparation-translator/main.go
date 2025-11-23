package main

import (
	"context"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/preparation"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	// Parse command-line flags
	inputPath := flag.String("input", "/tmp/markdown_e2e_source.md", "Input ebook path")
	outputPath := flag.String("output", "/tmp/prepared_translated.epub", "Output EPUB path")
	analysisPath := flag.String("analysis", "/tmp/preparation_analysis.json", "Preparation analysis output path")
	sourceLang := flag.String("source", "English", "Source language")
	targetLang := flag.String("target", "Spanish", "Target language")
	passCount := flag.Int("passes", 2, "Number of preparation passes")
	providers := flag.String("providers", "deepseek,zhipu", "Comma-separated list of LLM providers")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Validate input file exists
	if _, err := os.Stat(*inputPath); os.IsNotExist(err) {
		log.Fatalf("Input file does not exist: %s", *inputPath)
	}

	log.Printf("=== PREPARATION + TRANSLATION INTEGRATION TEST ===\n")
	log.Printf("Input: %s", *inputPath)
	log.Printf("Output: %s", *outputPath)
	log.Printf("Analysis: %s", *analysisPath)
	log.Printf("Languages: %s → %s", *sourceLang, *targetLang)
	log.Printf("Preparation passes: %d", *passCount)
	log.Printf("Providers: %s\n", *providers)

	// Parse ebook
	log.Printf("\n1. Parsing ebook...")
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(*inputPath)
	if err != nil {
		log.Fatalf("Failed to parse ebook: %v", err)
	}
	log.Printf("✅ Parsed ebook: %d chapters, %d words",
		book.GetChapterCount(), book.GetWordCount())

	// Setup languages
	sourceLanguage := language.Language{Code: "ru", Name: *sourceLang}
	targetLanguage := language.Language{Code: "sr", Name: *targetLang}

	// Setup preparation configuration
	log.Printf("\n2. Configuring preparation phase...")
	prepConfig := &preparation.PreparationConfig{
		PassCount:          *passCount,
		Providers:          []string{"deepseek", "zhipu"}, // Fixed for now
		AnalyzeContentType: true,
		AnalyzeCharacters:  true,
		AnalyzeTerminology: true,
		AnalyzeCulture:     true,
		AnalyzeChapters:    true,
		DetailLevel:        "comprehensive",
		SourceLanguage:     *sourceLang,
		TargetLanguage:     *targetLang,
	}

	// Create base translator (for translation phase)
	log.Printf("\n3. Creating translator...")
	translatorConfig := translator.TranslationConfig{
		SourceLang: sourceLanguage.Code,
		TargetLang: targetLanguage.Code,
		Provider:   "deepseek",
		Model:      "deepseek-chat",
	}

	baseTranslator, err := llm.NewLLMTranslator(translatorConfig)
	if err != nil {
		log.Fatalf("Failed to create translator: %v", err)
	}

	// Create preparation-aware translator
	log.Printf("\n4. Creating preparation-aware translator...")
	prepTranslator := preparation.NewPreparationAwareTranslator(
		baseTranslator,
		nil, // No language detector for test
		sourceLanguage,
		targetLanguage,
		prepConfig,
	)

	// Create event bus for progress tracking
	eventBus := events.NewEventBus()
	sessionID := "prep-test-session"

	// Subscribe to events with handler functions
	progressHandler := func(event events.Event) {
		log.Printf("📊 Progress: %s", event.Message)
		if data, ok := event.Data["phase"]; ok {
			if phase, ok := data.(string); ok && phase == "preparation" {
				// Log detailed preparation info
				if contentType, ok := event.Data["content_type"].(string); ok {
					log.Printf("   Content Type: %s", contentType)
				}
				if genre, ok := event.Data["genre"].(string); ok {
					log.Printf("   Genre: %s", genre)
				}
			}
		}
	}

	errorHandler := func(event events.Event) {
		log.Printf("❌ Error: %s", event.Message)
	}

	eventBus.Subscribe(events.EventTranslationProgress, progressHandler)
	eventBus.Subscribe(events.EventTranslationError, errorHandler)

	// Run preparation + translation
	ctx := context.Background()
	startTime := time.Now()

	log.Printf("\n5. Running preparation + translation pipeline...")
	err = prepTranslator.TranslateBook(ctx, book, eventBus, sessionID)
	if err != nil {
		log.Fatalf("Translation failed: %v", err)
	}

	duration := time.Since(startTime)
	log.Printf("\n✅ Translation complete in %.2f seconds", duration.Seconds())

	// Save preparation analysis
	log.Printf("\n6. Saving preparation analysis...")
	if err := prepTranslator.SavePreparationAnalysis(*analysisPath); err != nil {
		log.Printf("Warning: Failed to save analysis: %v", err)
	} else {
		log.Printf("✅ Analysis saved to: %s", *analysisPath)
	}

	// Print preparation summary
	if result := prepTranslator.GetPreparationResult(); result != nil {
		log.Printf("\n=== PREPARATION SUMMARY ===")
		log.Printf("Content Type: %s", result.FinalAnalysis.ContentType)
		log.Printf("Genre: %s", result.FinalAnalysis.Genre)
		log.Printf("Subgenres: %v", result.FinalAnalysis.Subgenres)
		log.Printf("Tone: %s", result.FinalAnalysis.Tone)
		log.Printf("Untranslatable Terms: %d", len(result.FinalAnalysis.UntranslatableTerms))
		log.Printf("Footnotes Needed: %d", len(result.FinalAnalysis.FootnoteGuidance))
		log.Printf("Characters: %d", len(result.FinalAnalysis.Characters))
		log.Printf("Cultural References: %d", len(result.FinalAnalysis.CulturalReferences))
		log.Printf("Key Themes: %d", len(result.FinalAnalysis.KeyThemes))
		log.Printf("Preparation Duration: %.2f seconds", result.TotalDuration.Seconds())
		log.Printf("Total Passes: %d", result.PassCount)
		log.Printf("Total Tokens: %d", result.TotalTokens)

		// Print some key themes
		if len(result.FinalAnalysis.KeyThemes) > 0 {
			log.Printf("\nKey Themes:")
			for i, theme := range result.FinalAnalysis.KeyThemes {
				if i >= 5 {
					log.Printf("  ... and %d more", len(result.FinalAnalysis.KeyThemes)-5)
					break
				}
				log.Printf("  - %s", theme)
			}
		}

		// Print some untranslatable terms
		if len(result.FinalAnalysis.UntranslatableTerms) > 0 {
			log.Printf("\nUntranslatable Terms (sample):")
			for i, term := range result.FinalAnalysis.UntranslatableTerms {
				if i >= 5 {
					log.Printf("  ... and %d more", len(result.FinalAnalysis.UntranslatableTerms)-5)
					break
				}
				log.Printf("  - %s: %s", term.Term, term.Reason)
			}
		}

		// Print characters
		if len(result.FinalAnalysis.Characters) > 0 {
			log.Printf("\nCharacters:")
			for _, char := range result.FinalAnalysis.Characters {
				log.Printf("  - %s (%s)", char.Name, char.Role)
				if char.SpeechPattern != "" {
					log.Printf("    Speech: %s", char.SpeechPattern)
				}
			}
		}
	}

	// Save translated book
	log.Printf("\n7. Saving translated book...")
	writer := ebook.NewEPUBWriter()
	if err := writer.Write(book, *outputPath); err != nil {
		log.Fatalf("Failed to write EPUB: %v", err)
	}
	log.Printf("✅ Translated book saved to: %s", *outputPath)

	// Final statistics
	log.Printf("\n=== FINAL STATISTICS ===")
	log.Printf("Total Duration: %.2f seconds", duration.Seconds())
	log.Printf("Input Chapters: %d", book.GetChapterCount())
	log.Printf("Output File: %s", *outputPath)
	log.Printf("Analysis File: %s", *analysisPath)

	// Check file sizes
	if info, err := os.Stat(*outputPath); err == nil {
		log.Printf("Output Size: %d bytes", info.Size())
	}
	if info, err := os.Stat(*analysisPath); err == nil {
		log.Printf("Analysis Size: %d bytes", info.Size())
	}

	log.Printf("\n✅ TEST COMPLETE - Preparation + Translation pipeline successful!")
}
