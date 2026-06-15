package main

import (
	"context"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/preparation"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// validateOutputFormat checks the -format flag against the closed set of
// supported output formats {epub, md}. Step 4 of the pipeline only knows how
// to emit those two formats; any other value would silently produce NO output
// file while the program still reported success. Rejecting unsupported values
// up front turns that false-success into an honest, actionable error.
func validateOutputFormat(format string) error {
	switch format {
	case "epub", "md":
		return nil
	default:
		return fmt.Errorf("unsupported output format %q (supported: epub, md)", format)
	}
}

func main() {
	// Command line flags
	inputFile := flag.String("input", "", "Input file (EPUB or Markdown)")
	outputFile := flag.String("output", "", "Output file (optional, auto-generated if not provided)")
	outputFormat := flag.String("format", "epub", "Output format (epub, md)")
	targetLang := flag.String("lang", "en", "Target language code (default: English)")
	provider := flag.String("provider", "deepseek", "LLM provider (deepseek, openai, anthropic)")
	model := flag.String("model", "", "LLM model (optional, uses provider default)")
	keepMarkdown := flag.Bool("keep-md", true, "Keep intermediate markdown files")
	enablePreparation := flag.Bool("prepare", false, "Enable preparation phase with multi-LLM analysis")
	preparationPasses := flag.Int("prep-passes", 2, "Number of preparation analysis passes")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: markdown-translator -input <file> [-output <output_file>] [-format <format>] [-lang <language>] [-provider <provider>] [-keep-md]")
		fmt.Println("\nSupported input formats: EPUB (.epub), Markdown (.md)")
		fmt.Println("Supported output formats: EPUB (epub), Markdown (md)")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate output format against the supported closed set before doing any
	// work; otherwise an unsupported value silently produces no output file
	// while the pipeline still reports success.
	if err := validateOutputFormat(*outputFormat); err != nil {
		log.Fatalf("%v", err)
	}

	// Validate input file
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		log.Fatalf("Input file does not exist: %s", *inputFile)
	}

	// Detect input file type
	inputExt := strings.ToLower(filepath.Ext(*inputFile))
	isMarkdownInput := (inputExt == ".md" || inputExt == ".markdown")

	// Generate output filename if not provided
	if *outputFile == "" {
		base := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))
		outputExt := "epub"
		if *outputFormat == "md" {
			outputExt = "md"
		}
		*outputFile = fmt.Sprintf("Books/%s_%s.%s", base, *targetLang, outputExt)
	}

	// Generate intermediate markdown filenames (save to Books directory)
	outputBase := strings.TrimSuffix(filepath.Base(*outputFile), filepath.Ext(*outputFile))
	sourceMD := filepath.Join("Books", outputBase+"_source.md")
	translatedMD := filepath.Join("Books", outputBase+"_translated.md")

	// If input is already markdown, use it directly as source
	if isMarkdownInput {
		sourceMD = *inputFile
	}

	// Ensure Books directory exists
	if err := os.MkdirAll("Books", 0755); err != nil {
		log.Fatalf("Failed to create Books directory: %v", err)
	}

	// Open the LLMsVerifier bridge ONCE (R-1a/R2). Every translator + preparation
	// component this binary builds now obtains its model(s) from the strongest
	// verified model(s) the bridge selects — NOT a local/hardcoded provider. With
	// no provider API keys present the bridge returns an honest hard error here and
	// we fail loudly; we NEVER silently fall back to a local runtime (§11.4.69).
	// The -provider/-model flags are advisory under R2 (the bridge selects the
	// verified model); they are retained for compatibility (removal is R-2/R-4).
	bridgeCtx, bridgeCancel := context.WithTimeout(context.Background(), bridgeOpenTimeout)
	b, err := bridge.Open(bridgeCtx, bridge.Options{})
	bridgeCancel()
	if err != nil {
		log.Fatalf("LLMsVerifier bridge unavailable (no local-runtime fallback): %v", err)
	}
	task := selection.TaskRequirements{TargetLang: *targetLang}

	// -provider/-model are advisory under R2 (the bridge selects the verified
	// model). Surface them so the operator sees they no longer drive selection,
	// and so the retained flags remain referenced (removal is R-2/R-4).
	if *model != "" {
		fmt.Printf("ℹ️  -provider=%s -model=%s are advisory; the bridge selects the strongest verified model.\n", *provider, *model)
	}

	fmt.Printf("🚀 Markdown-Based Translation Pipeline\n\n")
	fmt.Printf("Input:  %s (format: %s)\n", *inputFile, inputExt)
	fmt.Printf("Output: %s (format: %s)\n\n", *outputFile, *outputFormat)

	var stepNum int = 1
	totalSteps := 4
	if *enablePreparation {
		totalSteps++ // Add preparation step
	}
	if isMarkdownInput {
		totalSteps-- // Skip EPUB→MD conversion
	}
	if *outputFormat == "md" {
		totalSteps-- // Skip MD→EPUB conversion
	}

	// Step 1: EPUB → Markdown (skip if input is already markdown)
	if !isMarkdownInput {
		fmt.Printf("📖 Step %d/%d: Converting EPUB to Markdown...\n", stepNum, totalSteps)
		converter := markdown.NewEPUBToMarkdownConverter(false, "")
		if err := converter.ConvertEPUBToMarkdown(*inputFile, sourceMD); err != nil {
			log.Fatalf("Failed to convert EPUB to Markdown: %v", err)
		}
		fmt.Printf("✓ Source markdown saved: %s\n\n", sourceMD)
		stepNum++
	} else {
		fmt.Printf("ℹ️  Using markdown input directly: %s\n\n", sourceMD)
	}

	// Step 1.5: Preparation Phase (if enabled)
	var prepResult *preparation.PreparationResult
	if *enablePreparation {
		fmt.Printf("🔍 Step %d/%d: Content Analysis & Preparation...\n", stepNum, totalSteps)
		stepNum++

		// Parse the source book (either EPUB or reconstruct from markdown)
		var book *ebook.Book
		if !isMarkdownInput {
			parser := ebook.NewUniversalParser()
			var err error
			book, err = parser.Parse(*inputFile)
			if err != nil {
				log.Fatalf("Failed to parse book for preparation: %v", err)
			}
		} else {
			// Create minimal book structure from markdown for preparation
			book = &ebook.Book{
				Metadata: ebook.Metadata{
					Language: "ru", // Assume Russian source
				},
				Chapters: []ebook.Chapter{
					{
						Title: "Content",
						// Would need to read markdown content here
					},
				},
			}
		}

		// Configure preparation with multi-LLM analysis
		prepConfig := preparation.PreparationConfig{
			PassCount:          *preparationPasses,
			Providers:          []string{*provider}, // Use same provider for now
			AnalyzeContentType: true,
			AnalyzeCharacters:  true,
			AnalyzeTerminology: true,
			AnalyzeCulture:     true,
			AnalyzeChapters:    true,
			DetailLevel:        "comprehensive",
			SourceLanguage:     "ru",
			TargetLanguage:     *targetLang,
		}

		ctx := context.Background()
		// Source the preparation providers from the bridge's provider-diverse
		// verified ensemble (R-1a/R2) instead of the built-in per-provider
		// NewLLMTranslator construction.
		prepCoordinator, err := preparation.NewPreparationCoordinatorWithFactory(ctx, prepConfig, b.EnsembleFactory(task))
		if err != nil {
			log.Fatalf("Failed to create preparation coordinator: %v", err)
		}

		prepResult, err = prepCoordinator.PrepareBook(ctx, book)
		if err != nil {
			log.Printf("⚠️  Warning: Preparation failed: %v", err)
			fmt.Println("Continuing without preparation analysis...")
		} else {
			// Save preparation results
			prepJSON := filepath.Join("Books", outputBase+"_preparation.json")
			prepData, _ := json.MarshalIndent(prepResult, "", "  ")
			if err := os.WriteFile(prepJSON, prepData, 0644); err != nil {
				log.Printf("Warning: Failed to save preparation results: %v", err)
			} else {
				fmt.Printf("✓ Preparation complete (%d passes, %.2fs)\n",
					prepResult.PassCount, prepResult.TotalDuration.Seconds())
				fmt.Printf("  Analysis saved: %s\n", prepJSON)
				fmt.Printf("  Content type: %s\n", prepResult.FinalAnalysis.ContentType)
				fmt.Printf("  Genre: %s\n", prepResult.FinalAnalysis.Genre)
				fmt.Printf("  Characters: %d identified\n", len(prepResult.FinalAnalysis.Characters))
				fmt.Printf("  Untranslatable terms: %d identified\n", len(prepResult.FinalAnalysis.UntranslatableTerms))
				fmt.Printf("  Footnote guidance: %d items\n", len(prepResult.FinalAnalysis.FootnoteGuidance))
			}
		}
		fmt.Println()
	}

	// Step 2: Create translator.
	//
	// R-1d wiring: markdown.WorkflowConfig.LLMProvider is the markdown package's
	// dependency-injection seam for the LLM client. Its producer is the
	// LLMsVerifier bridge — bridge.BestClient sources the strongest verified
	// model's client (NO local runtime; honest hard error on no keys, §11.4.69).
	// The workflow's translate step consumes the SAME client through that field,
	// so the seam is genuinely wired (not an unset, nil-panicking field).
	fmt.Printf("🔧 Step %d/%d: Initializing translator...\n", stepNum, totalSteps)
	ctx := context.Background()
	workflowCfg, err := bridgeWorkflowConfig(ctx, b, task)
	if err != nil {
		log.Fatalf("Failed to create translator: %v", err)
	}
	fmt.Printf("✓ Using bridge source: %s\n\n", b.Source())
	stepNum++

	// Step 3: Translate Markdown — driven through WorkflowConfig.LLMProvider.
	fmt.Printf("🌍 Step %d/%d: Translating markdown content...\n", stepNum, totalSteps)
	mdTranslator := markdown.NewMarkdownTranslator(func(text string) (string, error) {
		return workflowCfg.LLMProvider.Translate(ctx, text, "")
	})

	if err := mdTranslator.TranslateMarkdownFile(sourceMD, translatedMD); err != nil {
		log.Fatalf("Failed to translate markdown: %v", err)
	}
	fmt.Printf("✓ Translated markdown saved: %s\n\n", translatedMD)
	stepNum++

	// Step 4: Markdown → EPUB (skip if output format is markdown)
	if *outputFormat == "epub" {
		fmt.Printf("📚 Step %d/%d: Converting translated markdown to EPUB...\n", stepNum, totalSteps)
		epubConverter := markdown.NewMarkdownToEPUBConverter()
		if err := epubConverter.ConvertMarkdownToEPUB(translatedMD, *outputFile); err != nil {
			log.Fatalf("Failed to convert markdown to EPUB: %v", err)
		}
		fmt.Printf("✓ Final EPUB created: %s\n\n", *outputFile)
	} else if *outputFormat == "md" {
		// Copy translated markdown to output file if different
		if translatedMD != *outputFile {
			content, err := os.ReadFile(translatedMD)
			if err != nil {
				log.Fatalf("Failed to read translated markdown: %v", err)
			}
			if err := os.WriteFile(*outputFile, content, 0644); err != nil {
				log.Fatalf("Failed to write output markdown: %v", err)
			}
		}
		fmt.Printf("✓ Final markdown created: %s\n\n", *outputFile)
	}

	// Cleanup markdown files if requested
	if !*keepMarkdown && *outputFormat == "epub" {
		fmt.Println("🧹 Cleaning up intermediate files...")
		if !isMarkdownInput {
			os.Remove(sourceMD)
		}
		os.Remove(translatedMD)
		fmt.Println("✓ Cleanup complete")
	}

	fmt.Println("✅ Translation complete!")
	fmt.Printf("\nFiles generated:\n")
	if *keepMarkdown || isMarkdownInput {
		if !isMarkdownInput {
			fmt.Printf("  - Source MD:      %s\n", sourceMD)
		}
		fmt.Printf("  - Translated MD:  %s\n", translatedMD)
	}
	if *outputFormat == "epub" {
		fmt.Printf("  - Final EPUB:     %s\n", *outputFile)
	} else {
		fmt.Printf("  - Final Markdown: %s\n", *outputFile)
	}
}

// bridgeOpenTimeout bounds the one-time LLMsVerifier bridge bootstrap (default
// 5m verify pass + 30s headroom, mirroring cmd/model-bridge's openBridge).
const bridgeOpenTimeout = 5*time.Minute + 30*time.Second

// bridgeWorkflowConfig produces a markdown.WorkflowConfig whose LLMProvider seam
// is sourced from the LLMsVerifier bridge (R-1d wiring). bridge.BestClient
// returns the strongest verified model's llm.LLMClient — NO local/hardcoded
// provider, NO local runtime: with no provider API keys present BestClient
// returns an honest hard error and this function propagates it, so there is
// NEVER a silent local-runtime (e.g. llama.cpp) fallback (§11.4.69). The former
// -provider/llamacpp arm is intentionally removed.
func bridgeWorkflowConfig(ctx context.Context, b *bridge.Bridge, task selection.TaskRequirements) (markdown.WorkflowConfig, error) {
	client, err := b.BestClient(ctx, task)
	if err != nil {
		return markdown.WorkflowConfig{}, err
	}
	return markdown.WorkflowConfig{
		ChunkSize:        2000,
		OverlapSize:      0,
		MaxConcurrency:   4,
		TranslationCache: map[string]string{},
		LLMProvider:      client,
	}, nil
}
