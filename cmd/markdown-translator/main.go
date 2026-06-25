package main

import (
	"context"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/preparation"
	"digital.vasic.translator/pkg/translator/llm"
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
	apiKey := flag.String("api-key", "", "API key for the provider (overrides <PROVIDER>_API_KEY env)")
	baseURL := flag.String("base-url", "", "Base URL override for the provider")
	flag.Parse()

	// Detect whether -provider / -model were passed EXPLICITLY (vs. their defaults).
	// An explicit -provider must route to THAT provider's client (honoring -model/
	// -api-key/-base-url) and hard-error on misconfiguration — never the bridge's
	// global strongest-verified selection (§11.4.69: no silent wrong-provider).
	var providerExplicit, modelExplicit bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "provider":
			providerExplicit = true
		case "model":
			modelExplicit = true
		}
	})

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

	// When -provider is NOT set explicitly, it stays advisory under R2 (the bridge
	// selects the strongest verified model). When it IS set explicitly, it is
	// HONORED: the run routes to that provider's client (see bridgeWorkflowConfig).
	if !providerExplicit && *model != "" {
		fmt.Printf("ℹ️  -provider=%s -model=%s are advisory; the bridge selects the strongest verified model.\n", *provider, *model)
	}
	if providerExplicit {
		fmt.Printf("ℹ️  -provider=%s is honored (explicit); routing to that provider.\n", *provider)
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
	explicitModel := ""
	if modelExplicit {
		explicitModel = *model
	}
	workflowCfg, err := bridgeWorkflowConfig(ctx, b, task, providerSelection{
		explicit:   providerExplicit,
		providerID: *provider,
		model:      explicitModel,
		apiKey:     *apiKey,
		baseURL:    *baseURL,
	})
	if err != nil {
		log.Fatalf("Failed to create translator: %v", err)
	}
	fmt.Printf("✓ Using bridge source: %s\n\n", b.Source())
	stepNum++

	// Step 3: Translate Markdown — driven through WorkflowConfig.LLMProvider.
	fmt.Printf("🌍 Step %d/%d: Translating markdown content...\n", stepNum, totalSteps)
	mdTranslator := markdown.NewMarkdownTranslator(func(text string) (string, error) {
		// workflowCfg.LLMProvider is the raw OpenAI-compatible client (bridge.BestClient),
		// which sends ONLY its 2nd arg as the user message and ignores the 1st. Passing
		// the block content in the 1st arg with an empty 2nd arg sent an EMPTY user
		// message → provider boilerplate stored as the translation → real chapter text
		// lost (docs/qa/bug_markdown_empty_payload_rootcause_20260616-152640/FINDING.md).
		// Build a real translation instruction embedding the block text and pass it as
		// the 2nd arg, mirroring pkg/markdown/simple_workflow.go's working pattern.
		prompt := fmt.Sprintf(
			"Translate the following text to %s.\n"+
				"Provide ONLY the translation without any explanations, notes, or additional text.\n"+
				"Maintain the original formatting, line breaks, and structure.\n\n"+
				"Source text:\n%s\n\nTranslation:",
			*targetLang, text,
		)
		return workflowCfg.LLMProvider.Translate(ctx, text, prompt)
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
// providerSelection carries the operator's provider routing choice for
// bridgeWorkflowConfig: when explicit is true the run is routed to providerID's
// client (honoring model/apiKey/baseURL); otherwise the bridge selects the
// strongest verified model (the R2 advisory default).
type providerSelection struct {
	explicit   bool
	providerID string
	model      string
	apiKey     string
	baseURL    string
}

func bridgeWorkflowConfig(ctx context.Context, b *bridge.Bridge, task selection.TaskRequirements, sel providerSelection) (markdown.WorkflowConfig, error) {
	var client llm.LLMClient
	var err error
	if sel.explicit {
		// EXPLICIT -provider X: build THAT provider's client (no silent fallback).
		client, err = b.ClientForProvider(ctx, bridge.ProviderRequest{
			ProviderID: sel.providerID,
			Model:      sel.model,
			APIKey:     sel.apiKey,
			BaseURL:    sel.baseURL,
			Task:       task,
		})
	} else {
		// R2 advisory default: strongest verified model across all providers.
		client, err = b.BestClient(ctx, task)
	}
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
