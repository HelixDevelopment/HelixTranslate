package main

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/fb2"
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

const version = "1.0.0"

func main() {
	// Define CLI flags
	var (
		inputFile    string
		outputFile   string
		provider     string
		model        string
		apiKey       string
		baseURL      string
		scriptType   string
		showVersion  bool
		showHelp     bool
		createConfig string
	)

	flag.StringVar(&inputFile, "input", "", "Input FB2 file")
	flag.StringVar(&inputFile, "i", "", "Input FB2 file (shorthand)")
	flag.StringVar(&outputFile, "output", "", "Output FB2 file")
	flag.StringVar(&outputFile, "o", "", "Output FB2 file (shorthand)")
	flag.StringVar(&provider, "provider", "dictionary", "Translation provider (dictionary, openai, anthropic, zhipu, deepseek, ollama)")
	flag.StringVar(&provider, "p", "dictionary", "Translation provider (shorthand)")
	flag.StringVar(&model, "model", "", "LLM model name")
	flag.StringVar(&apiKey, "api-key", "", "API key for LLM provider")
	flag.StringVar(&baseURL, "base-url", "", "Base URL for LLM provider")
	flag.StringVar(&scriptType, "script", "cyrillic", "Output script type (cyrillic, latin)")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	flag.StringVar(&createConfig, "create-config", "", "Create config file template")

	flag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("Russian-Serbian FB2 Translator v%s\n", version)
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

	// Load API key from environment if not provided
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(provider)
	}

	// Generate output filename if not provided
	if outputFile == "" {
		outputFile = generateOutputFilename(inputFile, provider)
	}

	// Create event bus
	eventBus := events.NewEventBus()

	// Subscribe to events for CLI output
	eventBus.SubscribeAll(func(event events.Event) {
		fmt.Printf("[%s] %s\n", event.Type, event.Message)
	})

	// Run translation
	if err := translateFB2(inputFile, outputFile, provider, model, apiKey, baseURL, scriptType, eventBus); err != nil {
		fmt.Fprintf(os.Stderr, "Translation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Translation completed successfully!\n")
	fmt.Printf("Output file: %s\n", outputFile)
}

func translateFB2(inputFile, outputFile, providerName, model, apiKey, baseURL, scriptType string, eventBus *events.EventBus) error {
	ctx := context.Background()

	// Parse input FB2
	fmt.Printf("Parsing FB2 file: %s\n", inputFile)
	parser := fb2.NewParser()
	book, err := parser.Parse(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse FB2: %w", err)
	}

	// Create translator
	config := translator.TranslationConfig{
		SourceLang: "ru",
		TargetLang: "sr",
		Provider:   providerName,
		Model:      model,
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Options:    make(map[string]interface{}),
	}

	var trans translator.Translator
	if providerName == "dictionary" {
		trans = dictionary.NewDictionaryTranslator(config)
	} else {
		trans, err = llm.NewLLMTranslator(config)
		if err != nil {
			return fmt.Errorf("failed to create translator: %w", err)
		}
	}

	fmt.Printf("Using translator: %s\n", trans.GetName())

	// Translate book title
	sessionID := "cli-session"
	if book.Description.TitleInfo.BookTitle != "" {
		fmt.Printf("Translating title...\n")
		translated, err := trans.TranslateWithProgress(
			ctx,
			book.Description.TitleInfo.BookTitle,
			"Book title",
			eventBus,
			sessionID,
		)
		if err == nil {
			book.Description.TitleInfo.BookTitle = translated
		}
	}

	// Translate body sections
	fmt.Printf("Translating content...\n")
	totalSections := 0
	for _, body := range book.Body {
		totalSections += len(body.Section)
	}

	currentSection := 0
	for i := range book.Body {
		for j := range book.Body[i].Section {
			currentSection++
			fmt.Printf("Translating section %d/%d...\n", currentSection, totalSections)

			if err := translateSection(ctx, &book.Body[i].Section[j], trans, eventBus, sessionID); err != nil {
				return fmt.Errorf("failed to translate section: %w", err)
			}
		}
	}

	// Convert script if needed
	if scriptType == "latin" {
		fmt.Printf("Converting to Latin script...\n")
		converter := script.NewConverter()
		book.Description.TitleInfo.BookTitle = converter.ToLatin(book.Description.TitleInfo.BookTitle)
		for i := range book.Body {
			for j := range book.Body[i].Section {
				convertSectionToLatin(&book.Body[i].Section[j], converter)
			}
		}
	}

	// Update metadata
	book.SetLanguage("sr")

	// Write output
	fmt.Printf("Writing output file...\n")
	if err := parser.Write(outputFile, book); err != nil {
		return fmt.Errorf("failed to write FB2: %w", err)
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

func translateSection(ctx context.Context, section *fb2.Section, trans translator.Translator, eventBus *events.EventBus, sessionID string) error {
	// Translate title
	for i := range section.Title.Paragraphs {
		if section.Title.Paragraphs[i].Text != "" {
			translated, err := trans.TranslateWithProgress(
				ctx,
				section.Title.Paragraphs[i].Text,
				"Section title",
				eventBus,
				sessionID,
			)
			if err == nil {
				section.Title.Paragraphs[i].Text = translated
			}
		}
	}

	// Translate paragraphs
	for i := range section.Paragraph {
		if section.Paragraph[i].Text != "" {
			translated, err := trans.TranslateWithProgress(
				ctx,
				section.Paragraph[i].Text,
				"Paragraph",
				eventBus,
				sessionID,
			)
			if err == nil {
				section.Paragraph[i].Text = translated
			}
		}
	}

	// Recursively translate subsections
	for i := range section.Section {
		if err := translateSection(ctx, &section.Section[i], trans, eventBus, sessionID); err != nil {
			return err
		}
	}

	return nil
}

func convertSectionToLatin(section *fb2.Section, converter *script.Converter) {
	// Convert title
	for i := range section.Title.Paragraphs {
		section.Title.Paragraphs[i].Text = converter.ToLatin(section.Title.Paragraphs[i].Text)
	}

	// Convert paragraphs
	for i := range section.Paragraph {
		section.Paragraph[i].Text = converter.ToLatin(section.Paragraph[i].Text)
	}

	// Recursively convert subsections
	for i := range section.Section {
		convertSectionToLatin(&section.Section[i], converter)
	}
}

func getAPIKeyFromEnv(provider string) string {
	envMappings := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"zhipu":     "ZHIPU_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
	}

	if envVar, ok := envMappings[provider]; ok {
		return os.Getenv(envVar)
	}

	return ""
}

func generateOutputFilename(inputFile, provider string) string {
	ext := filepath.Ext(inputFile)
	base := strings.TrimSuffix(inputFile, ext)
	return fmt.Sprintf("%s_sr_%s%s", base, provider, ext)
}

func createConfigFile(filename string) error {
	config := `{
  "provider": "openai",
  "model": "gpt-4",
  "temperature": 0.3,
  "max_tokens": 4000,
  "script": "cyrillic"
}
`
	return os.WriteFile(filename, []byte(config), 0644)
}

func printHelp() {
	fmt.Printf(`Russian-Serbian FB2 Translator v%s

Usage:
  translator [options] -input <file.fb2>

Options:
  -i, -input <file>       Input FB2 file (required)
  -o, -output <file>      Output FB2 file (optional, auto-generated if not provided)
  -p, -provider <name>    Translation provider (default: dictionary)
                          Options: dictionary, openai, anthropic, zhipu, deepseek, ollama
  -model <name>           LLM model name (e.g., gpt-4, claude-3-sonnet-20240229)
  -api-key <key>          API key for LLM provider (or use environment variables)
  -base-url <url>         Base URL for LLM provider (optional)
  -script <type>          Output script (cyrillic or latin, default: cyrillic)
  -create-config <file>   Create a config file template
  -v, -version            Show version
  -h, -help               Show this help

Environment Variables:
  OPENAI_API_KEY          OpenAI API key
  ANTHROPIC_API_KEY       Anthropic API key
  ZHIPU_API_KEY           Zhipu AI API key
  DEEPSEEK_API_KEY        DeepSeek API key

Examples:
  # Basic dictionary translation
  translator -input book.fb2

  # LLM translation with OpenAI
  export OPENAI_API_KEY="your-key"
  translator -input book.fb2 -provider openai -model gpt-4

  # LLM translation with Anthropic Claude
  export ANTHROPIC_API_KEY="your-key"
  translator -input book.fb2 -provider anthropic

  # Latin script output
  translator -input book.fb2 -provider deepseek -script latin

  # Local Ollama translation
  translator -input book.fb2 -provider ollama -model llama3:8b

`, version)
}
