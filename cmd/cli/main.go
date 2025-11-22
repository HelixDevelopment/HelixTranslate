package main

import (
	"context"
	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/dictionary"
	"digital.vasic.translator/pkg/translator/llm"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const version = "2.0.0"

func main() {
	// Define CLI flags
	var (
		inputFile         string
		outputFile        string
		outputFormat      string
		provider          string
		model             string
		apiKey            string
		baseURL           string
		scriptType        string
		locale            string
		targetLanguage    string
		sourceLanguage    string
		showVersion       bool
		showHelp          bool
		createConfig      string
		detectLang        bool
		disableLocalLLMs  bool
		preferDistributed bool
	)

	flag.StringVar(&inputFile, "input", "", "Input ebook file (any format: FB2, EPUB, TXT, HTML)")
	flag.StringVar(&inputFile, "i", "", "Input ebook file (shorthand)")
	flag.StringVar(&outputFile, "output", "", "Output file")
	flag.StringVar(&outputFile, "o", "", "Output file (shorthand)")
	flag.StringVar(&outputFormat, "format", "epub", "Output format (epub, fb2, txt)")
	flag.StringVar(&outputFormat, "f", "epub", "Output format (shorthand)")
	flag.StringVar(&provider, "provider", "dictionary", "Translation provider")
	flag.StringVar(&provider, "p", "dictionary", "Translation provider (shorthand)")
	flag.StringVar(&model, "model", "", "LLM model name")
	flag.StringVar(&apiKey, "api-key", "", "API key for LLM provider")
	flag.StringVar(&baseURL, "base-url", "", "Base URL for LLM provider")
	flag.StringVar(&scriptType, "script", "cyrillic", "Output script (cyrillic, latin)")
	flag.StringVar(&locale, "locale", "", "Target language locale (e.g., sr, de, DE)")
	flag.StringVar(&targetLanguage, "language", "", "Target language name (e.g., Serbian, German)")
	flag.StringVar(&sourceLanguage, "source", "", "Source language (optional, auto-detected if not specified)")
	flag.BoolVar(&detectLang, "detect", false, "Detect source language and exit")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	flag.StringVar(&createConfig, "create-config", "", "Create config file template")
	flag.BoolVar(&disableLocalLLMs, "disable-local-llms", false, "Disable local LLM providers, use only distributed workers")
	flag.BoolVar(&preferDistributed, "prefer-distributed", false, "Prefer distributed workers over local LLMs when available")

	flag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("Universal Ebook Translator v%s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp || (inputFile == "" && createConfig == "") {
		printHelp()
		os.Exit(0)
	}

	// Handle config creation
	if createConfig != "" {
		if err := createConfigFile(createConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config file created: %s\n", createConfig)
		os.Exit(0)
	}

	// Parse target language from locale or language flag
	var targetLang language.Language
	var err error

	if locale != "" {
		targetLang, err = language.ParseLanguage(locale)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid locale '%s': %v\n", locale, err)
			fmt.Fprintf(os.Stderr, "Supported languages: %s\n", getSupportedLanguagesString())
			os.Exit(1)
		}
	} else if targetLanguage != "" {
		targetLang, err = language.ParseLanguage(targetLanguage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid language '%s': %v\n", targetLanguage, err)
			fmt.Fprintf(os.Stderr, "Supported languages: %s\n", getSupportedLanguagesString())
			os.Exit(1)
		}
	} else {
		// Default to Serbian
		targetLang = language.Serbian
	}

	// Parse source language if specified
	var sourceLang language.Language
	if sourceLanguage != "" {
		sourceLang, err = language.ParseLanguage(sourceLanguage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid source language '%s': %v\n", sourceLanguage, err)
			os.Exit(1)
		}
	}

	// Load API key from environment if not provided
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(provider)
	}

	// Create event bus
	eventBus := events.NewEventBus()

	// Subscribe to events for CLI output
	eventBus.SubscribeAll(func(event events.Event) {
		fmt.Printf("[%s] %s\n", event.Type, event.Message)
	})

	// Parse input file
	fmt.Printf("Universal Ebook Translator v%s\n\n", version)
	fmt.Printf("Input file: %s\n", inputFile)

	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse ebook: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Detected format: %s\n", book.Format)
	fmt.Printf("Title: %s\n", book.Metadata.Title)
	fmt.Printf("Chapters: %d\n", book.GetChapterCount())

	// Detect language if requested
	if detectLang {
		langDetector := language.NewDetector(nil)
		sample := book.ExtractText()
		if len(sample) > 2000 {
			sample = sample[:2000]
		}

		detectedLang, err := langDetector.Detect(context.Background(), sample)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Language detection failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nDetected language: %s (%s)\n", detectedLang.Name, detectedLang.Code)
		os.Exit(0)
	}

	fmt.Printf("Target language: %s (%s)\n", targetLang.Name, targetLang.Code)

	// Generate output filename if not provided
	if outputFile == "" {
		outputFile = generateOutputFilename(inputFile, targetLang.Code, outputFormat)
	}

	// Run translation
	if err := translateEbook(
		book,
		outputFile,
		outputFormat,
		provider,
		model,
		apiKey,
		baseURL,
		scriptType,
		sourceLang,
		targetLang,
		eventBus,
		disableLocalLLMs,
		preferDistributed,
	); err != nil {
		fmt.Fprintf(os.Stderr, "Translation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Translation completed successfully!\n")
	fmt.Printf("Output file: %s\n", outputFile)
	fmt.Printf("Output format: %s\n", outputFormat)
}

func translateEbook(
	book *ebook.Book,
	outputFile, outputFormat, providerName, model, apiKey, baseURL, scriptType string,
	sourceLang, targetLang language.Language,
	eventBus *events.EventBus,
	disableLocalLLMs, preferDistributed bool,
) error {
	ctx := context.Background()

	// Create translator
	config := translator.TranslationConfig{
		SourceLang: sourceLang.Code,
		TargetLang: targetLang.Code,
		Provider:   providerName,
		Model:      model,
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Options:    make(map[string]interface{}),
	}

	var trans translator.Translator
	var err error
	sessionID := "cli-session"

	// Try multi-LLM first if provider is "multi-llm" or not specified
	if providerName == "multi-llm" || providerName == "" {
		multiTrans, multiErr := coordination.NewMultiLLMTranslatorWrapperWithConfig(config, eventBus, sessionID, disableLocalLLMs, preferDistributed)
		if multiErr == nil {
			trans = multiTrans
			fmt.Printf("Using translator: multi-llm-coordinator (%d instances)\n\n", multiTrans.Coordinator.GetInstanceCount())
		} else if providerName == "multi-llm" {
			// User explicitly requested multi-llm but it failed
			return fmt.Errorf("failed to create multi-LLM translator: %w", multiErr)
		}
		// Otherwise fall through to single translator
	}

	// Fall back to single translator
	if trans == nil {
		if providerName == "dictionary" {
			trans = dictionary.NewDictionaryTranslator(config)
		} else {
			trans, err = llm.NewLLMTranslator(config)
			if err != nil {
				return fmt.Errorf("failed to create translator: %w", err)
			}
		}
		fmt.Printf("Using translator: %s\n\n", trans.GetName())
	}

	// Create language detector with LLM support if API key available
	var llmDetector language.LLMDetector
	if apiKey != "" {
		llmDetector = language.NewSimpleLLMDetector(providerName, apiKey)
	}
	langDetector := language.NewDetector(llmDetector)

	// Create universal translator
	universalTrans := translator.NewUniversalTranslator(
		trans,
		langDetector,
		sourceLang,
		targetLang,
	)

	// Translate the book
	if err := universalTrans.TranslateBook(ctx, book, eventBus, sessionID); err != nil {
		return fmt.Errorf("translation failed: %w", err)
	}

	// Convert script if needed
	if scriptType == "latin" && targetLang.Code == "sr" {
		fmt.Printf("Converting to Latin script...\n")
		converter := script.NewConverter()
		convertBookToLatin(book, converter)
	}

	// Write output in requested format
	fmt.Printf("Writing output file...\n")
	outFormat := format.ParseFormat(outputFormat)

	switch outFormat {
	case format.FormatEPUB:
		writer := ebook.NewEPUBWriter()
		if err := writer.Write(book, outputFile); err != nil {
			return fmt.Errorf("failed to write EPUB: %w", err)
		}

	case format.FormatFB2:
		// Convert to FB2 and write
		// For now, we'll use EPUB as primary format
		return fmt.Errorf("FB2 output format not yet implemented")

	case format.FormatTXT:
		// Write as plain text
		if err := writeAsText(book, outputFile); err != nil {
			return fmt.Errorf("failed to write TXT: %w", err)
		}

	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	// Print statistics
	stats := trans.GetStats()
	fmt.Printf("\nTranslation Statistics:\n")
	fmt.Printf("  Total: %d\n", stats.Total)
	fmt.Printf("  Translated: %d\n", stats.Translated)
	fmt.Printf("  Cached: %d\n", stats.Cached)
	fmt.Printf("  Errors: %d\n", stats.Errors)

	return nil
}

func convertBookToLatin(book *ebook.Book, converter *script.Converter) {
	// Convert metadata
	book.Metadata.Title = converter.ToLatin(book.Metadata.Title)
	book.Metadata.Description = converter.ToLatin(book.Metadata.Description)

	for i := range book.Metadata.Authors {
		book.Metadata.Authors[i] = converter.ToLatin(book.Metadata.Authors[i])
	}

	// Convert chapters
	for i := range book.Chapters {
		convertChapterToLatin(&book.Chapters[i], converter)
	}
}

func convertChapterToLatin(chapter *ebook.Chapter, converter *script.Converter) {
	chapter.Title = converter.ToLatin(chapter.Title)

	for i := range chapter.Sections {
		convertSectionToLatin(&chapter.Sections[i], converter)
	}
}

func convertSectionToLatin(section *ebook.Section, converter *script.Converter) {
	section.Title = converter.ToLatin(section.Title)
	section.Content = converter.ToLatin(section.Content)

	for i := range section.Subsections {
		convertSectionToLatin(&section.Subsections[i], converter)
	}
}

func writeAsText(book *ebook.Book, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	text := book.ExtractText()
	_, err = file.WriteString(text)
	return err
}

func getAPIKeyFromEnv(provider string) string {
	envMappings := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"zhipu":     "ZHIPU_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
		"qwen":      "QWEN_API_KEY",
	}

	if envVar, ok := envMappings[provider]; ok {
		return os.Getenv(envVar)
	}

	return ""
}

func generateOutputFilename(inputFile, targetLang, outputFormat string) string {
	ext := filepath.Ext(inputFile)
	base := strings.TrimSuffix(inputFile, ext)

	// Add target language
	outputExt := "." + outputFormat
	return fmt.Sprintf("%s_%s%s", base, targetLang, outputExt)
}

func createConfigFile(filename string) error {
	config := `{
  "provider": "openai",
  "model": "gpt-4",
  "temperature": 0.3,
  "max_tokens": 4000,
  "target_language": "sr",
  "output_format": "epub",
  "script": "cyrillic"
}
`
	return os.WriteFile(filename, []byte(config), 0644)
}

func getSupportedLanguagesString() string {
	langs := language.GetSupportedLanguages()
	var names []string
	for _, lang := range langs {
		names = append(names, fmt.Sprintf("%s (%s)", lang.Name, lang.Code))
	}
	return strings.Join(names, ", ")
}

func printHelp() {
	fmt.Printf(`Universal Ebook Translator v%s

Translate ebooks between any languages with support for multiple formats.

Usage:
  translator [options] -input <file>

Options:
  -i, -input <file>       Input ebook file (any format: FB2, EPUB, TXT, HTML)
  -o, -output <file>      Output file (auto-generated if not specified)
  -f, -format <format>    Output format (epub, fb2, txt) [default: epub]

  -locale <code>          Target language locale (e.g., sr, de, fr, es)
  -language <name>        Target language name (e.g., Serbian, German, French)
                          (case-insensitive, default: Serbian)
  -source <lang>          Source language (optional, auto-detected)
  -detect                 Detect source language and exit

  -p, -provider <name>    Translation provider (dictionary, openai, anthropic,
                          zhipu, deepseek, qwen, ollama, llamacpp) [default: dictionary]
  -model <name>           LLM model name (e.g., gpt-4, claude-3-sonnet)
  -api-key <key>          API key for LLM provider
  -base-url <url>         Base URL for LLM provider

  -script <type>          Output script for Serbian (cyrillic, latin)
                          [default: cyrillic]

   -create-config <file>   Create a config file template
   -disable-local-llms     Disable local LLM providers (Ollama), use only API providers
   -prefer-distributed     Prefer distributed workers over local LLMs (when available)
   -v, -version            Show version
   -h, -help               Show this help

Supported Input Formats:
  FB2, EPUB, TXT, HTML

Supported Output Formats:
  EPUB (default), TXT

Supported Languages:
  %s

Environment Variables:
  OPENAI_API_KEY          OpenAI API key
  ANTHROPIC_API_KEY       Anthropic API key
  ZHIPU_API_KEY           Zhipu AI API key
  DEEPSEEK_API_KEY        DeepSeek API key
  QWEN_API_KEY            Qwen (Alibaba Cloud) API key

Examples:
  # Translate any ebook to Serbian (auto-detect source language)
  translator -input book.epub

  # Translate to German
  translator -input book.fb2 -locale de
  translator -input book.epub -language German

  # Translate Russian to French with OpenAI
  export OPENAI_API_KEY="your-key"
  translator -input book_ru.epub -locale fr -provider openai -model gpt-4

  # Detect language only
  translator -input mystery_book.epub -detect

  # Latin script output (for Serbian)
  translator -input book.fb2 -script latin

  # Output as plain text
  translator -input book.epub -locale de -format txt

  # Local Ollama translation
  translator -input book.txt -locale es -provider ollama -model llama3:8b

`, version, getSupportedLanguagesString())
}
