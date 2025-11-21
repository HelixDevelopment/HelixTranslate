package main

import (
	"context"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Command line flags
	inputFile := flag.String("input", "", "Input file (EPUB or Markdown)")
	outputFile := flag.String("output", "", "Output file (optional, auto-generated if not provided)")
	outputFormat := flag.String("format", "epub", "Output format (epub, md)")
	targetLang := flag.String("lang", "sr", "Target language code (default: Serbian)")
	provider := flag.String("provider", "deepseek", "LLM provider (deepseek, openai, anthropic)")
	model := flag.String("model", "", "LLM model (optional, uses provider default)")
	keepMarkdown := flag.Bool("keep-md", true, "Keep intermediate markdown files")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Usage: markdown-translator -input <file> [-output <output_file>] [-format <format>] [-lang <language>] [-provider <provider>] [-keep-md]")
		fmt.Println("\nSupported input formats: EPUB (.epub), Markdown (.md)")
		fmt.Println("Supported output formats: EPUB (epub), Markdown (md)")
		flag.PrintDefaults()
		os.Exit(1)
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

	fmt.Printf("🚀 Markdown-Based Translation Pipeline\n\n")
	fmt.Printf("Input:  %s (format: %s)\n", *inputFile, inputExt)
	fmt.Printf("Output: %s (format: %s)\n\n", *outputFile, *outputFormat)

	var stepNum int = 1
	totalSteps := 4
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

	// Step 2: Create translator
	fmt.Printf("🔧 Step %d/%d: Initializing translator...\n", stepNum, totalSteps)
	llmTranslator, err := createTranslator(*provider, *model, *targetLang)
	if err != nil {
		log.Fatalf("Failed to create translator: %v", err)
	}
	fmt.Printf("✓ Using provider: %s\n\n", *provider)
	stepNum++

	// Step 3: Translate Markdown
	fmt.Printf("🌍 Step %d/%d: Translating markdown content...\n", stepNum, totalSteps)
	ctx := context.Background()
	mdTranslator := markdown.NewMarkdownTranslator(func(text string) (string, error) {
		return llmTranslator.Translate(ctx, text, "")
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
		fmt.Println("✓ Cleanup complete\n")
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

// createTranslator creates an LLM translator based on provider
func createTranslator(provider, model, targetLang string) (translator.Translator, error) {
	// Get API keys from environment
	var apiKey string
	var defaultModel string

	switch provider {
	case "deepseek":
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
		defaultModel = "deepseek-chat"
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
		defaultModel = "gpt-4"
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		defaultModel = "claude-3-sonnet-20240229"
	case "zhipu":
		apiKey = os.Getenv("ZHIPU_API_KEY")
		defaultModel = "glm-4"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not set for provider %s (check environment variables)", provider)
	}

	// Use default model if not specified
	if model == "" {
		model = defaultModel
	}

	// Create translator config
	config := translator.TranslationConfig{
		SourceLang: "",
		TargetLang: targetLang,
		Provider:   provider,
		Model:      model,
		APIKey:     apiKey,
	}

	// Create LLM translator (it handles all providers internally)
	return llm.NewLLMTranslator(config)
}
