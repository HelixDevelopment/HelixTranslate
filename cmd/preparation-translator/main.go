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
	"strings"
	"time"
)

// resolveLanguageCodes maps the human-readable -source / -target flag values
// (e.g. "English", "Spanish", or an ISO code like "en") to the language.Language
// structs (with the correct ISO Code) that drive BOTH the translator's
// translation-direction prompt AND the output book's metadata language tag.
//
// The codes were previously hardcoded to "ru" (source) and "sr" (target),
// so the -source/-target flags only affected the human-readable Name passed to
// the preparation analyzer while the translator ALWAYS built a Russian→Serbian
// prompt and the output EPUB was ALWAYS tagged language "sr" — a wrong-output
// defect (§11.4): a user requesting English→Spanish silently got a
// Russian→Serbian prompt and an "sr"-tagged book. Unknown values fall back to a
// Language whose Code == the trimmed input (no guessing — §11.4.6) so an
// unrecognised language still threads a deterministic, honest code through the
// pipeline rather than a hardcoded wrong one.
func resolveLanguageCodes(source, target string) (language.Language, language.Language) {
	resolve := func(raw, fallbackName string) language.Language {
		if lang, err := language.ParseLanguage(raw); err == nil {
			return lang
		}
		trimmed := strings.TrimSpace(raw)
		return language.Language{Code: trimmed, Name: fallbackName}
	}
	return resolve(source, source), resolve(target, target)
}

// resolveAPIKey returns the API key for the given provider, mirroring
// cmd/unified-translator's resolveProviderAPIKey behaviour: an explicitly
// supplied -api-key flag value wins, otherwise the provider's well-known
// environment variable (e.g. DEEPSEEK_API_KEY) is used. Previously this binary
// built a TranslationConfig with NO APIKey, exposed no -api-key flag, and never
// read any *_API_KEY env var, so llm.NewLLMTranslator → NewDeepSeekClient always
// failed with "DeepSeek API key is required" and the pre-translation analysis
// CLI produced no analysis JSON for any provider — a dead-feature defect (§11.4).
// An empty return (unknown provider or no key anywhere) is honest (§11.4.6): the
// downstream client surfaces the precise "<provider> API key is required" error
// rather than this function guessing a value.
func resolveAPIKey(provider, flagVal string) string {
	if strings.TrimSpace(flagVal) != "" {
		return flagVal
	}
	envMap := map[string]string{
		"openai":     "OPENAI_API_KEY",
		"anthropic":  "ANTHROPIC_API_KEY",
		"deepseek":   "DEEPSEEK_API_KEY",
		"zhipu":      "ZHIPU_API_KEY",
		"qwen":       "QWEN_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"groq":       "GROQ_API_KEY",
		"mistral":    "MISTRAL_API_KEY",
		"xai":        "XAI_API_KEY",
		"cohere":     "COHERE_API_KEY",
		"togetherai": "TOGETHER_API_KEY",
	}
	if envVar, ok := envMap[provider]; ok {
		return os.Getenv(envVar)
	}
	return ""
}

// parseProviders splits a comma-separated providers flag into a clean slice.
// It trims surrounding whitespace from each entry and drops empty entries
// (so "deepseek, ,zhipu" -> ["deepseek","zhipu"]). When the result is empty
// (flag empty or only separators/whitespace), it returns the supplied
// fallback so the pipeline always has at least one provider to work with.
func parseProviders(raw string, fallback []string) []string {
	out := make([]string, 0)
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func main() {
	// Parse command-line flags
	inputPath := flag.String("input", "/tmp/markdown_e2e_source.md", "Input ebook path")
	outputPath := flag.String("output", "/tmp/prepared_translated.epub", "Output EPUB path")
	analysisPath := flag.String("analysis", "/tmp/preparation_analysis.json", "Preparation analysis output path")
	sourceLang := flag.String("source", "English", "Source language")
	targetLang := flag.String("target", "Spanish", "Target language")
	passCount := flag.Int("passes", 2, "Number of preparation passes")
	providers := flag.String("providers", "deepseek,zhipu", "Comma-separated list of LLM providers")
	apiKey := flag.String("api-key", "", "API key for the translation provider (falls back to the provider's *_API_KEY env var, e.g. DEEPSEEK_API_KEY)")
	flag.Parse()

	// Honor the -providers flag instead of silently ignoring it.
	providerList := parseProviders(*providers, []string{"deepseek", "zhipu"})

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

	// Setup languages — resolve the -source/-target flag values to their real
	// ISO codes so the translator and the output metadata follow the requested
	// language pair instead of a hardcoded Russian→Serbian one.
	sourceLanguage, targetLanguage := resolveLanguageCodes(*sourceLang, *targetLang)

	// Setup preparation configuration
	log.Printf("\n2. Configuring preparation phase...")
	prepConfig := &preparation.PreparationConfig{
		PassCount:          *passCount,
		Providers:          providerList,
		AnalyzeContentType: true,
		AnalyzeCharacters:  true,
		AnalyzeTerminology: true,
		AnalyzeCulture:     true,
		AnalyzeChapters:    true,
		DetailLevel:        "comprehensive",
		SourceLanguage:     *sourceLang,
		TargetLanguage:     *targetLang,
		// Without an API key the preparation coordinator's per-provider LLM
		// clients fail ("<provider> API key is required") and no analysis JSON
		// is produced. Resolve from -api-key, falling back to the first listed
		// provider's *_API_KEY env var; the coordinator additionally falls back
		// to each provider's own env var when this shared key is empty.
		APIKey: resolveAPIKey(providerList[0], *apiKey),
	}

	// Create base translator (for translation phase)
	log.Printf("\n3. Creating translator...")
	const translationProvider = "deepseek"
	translatorConfig := translator.TranslationConfig{
		SourceLang: sourceLanguage.Code,
		TargetLang: targetLanguage.Code,
		Provider:   translationProvider,
		Model:      "deepseek-chat",
		// Resolve the API key from the -api-key flag, falling back to the
		// provider's well-known env var (DEEPSEEK_API_KEY). Without this the
		// DeepSeek client always failed with "DeepSeek API key is required".
		APIKey: resolveAPIKey(translationProvider, *apiKey),
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
