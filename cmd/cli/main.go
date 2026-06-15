package main

import (
	"context"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/translator"
	versionpkg "digital.vasic.translator/pkg/version"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// bridgeOpenTimeout bounds the one-time LLMsVerifier bridge bootstrap (default
// 5m verify pass + 30s headroom, mirroring cmd/model-bridge's openBridge).
const bridgeOpenTimeout = 5*time.Minute + 30*time.Second

// version is sourced from the single authoritative version.AppVersion (== VERSION file).
const version = versionpkg.AppVersion

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
		configFile        string
		showVersion       bool
		showHelp          bool
		createConfig      string
		detectLang        bool
		disableLocalLLMs  bool
		preferDistributed bool
		hashCodebase      bool
	)

	flag.StringVar(&inputFile, "input", "", "Input ebook file (any format: FB2, EPUB, TXT, HTML, PDF, DOCX)")
	flag.StringVar(&inputFile, "i", "", "Input ebook file (any format: FB2, EPUB, TXT, HTML, PDF, DOCX)")
	flag.StringVar(&outputFile, "output", "", "Output file")
	flag.StringVar(&outputFile, "o", "", "Output file (shorthand)")
	flag.StringVar(&outputFormat, "format", "epub", "Output format (epub, fb2, txt)")
	flag.StringVar(&outputFormat, "f", "epub", "Output format (shorthand)")
	flag.StringVar(&provider, "provider", "openai", "Translation provider")
	flag.StringVar(&provider, "p", "openai", "Translation provider (shorthand)")
	flag.StringVar(&model, "model", "", "LLM model name")
	flag.StringVar(&apiKey, "api-key", "", "API key for LLM provider")
	flag.StringVar(&baseURL, "base-url", "", "Base URL for LLM provider")
	flag.StringVar(&scriptType, "script", "default", "Output script (default, cyrillic, latin, arabic, etc.)")
	flag.StringVar(&locale, "locale", "", "Target language locale (e.g., sr, de, DE)")
	flag.StringVar(&targetLanguage, "language", "", "Target language name (e.g., English, Spanish, French)")
	flag.StringVar(&sourceLanguage, "source", "", "Source language (optional, auto-detected if not specified)")
	flag.BoolVar(&detectLang, "detect", false, "Detect source language and exit")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	flag.StringVar(&createConfig, "create-config", "", "Create config file template")
	flag.BoolVar(&disableLocalLLMs, "disable-local-llms", false, "Disable local LLM providers, use only distributed workers")
	flag.BoolVar(&preferDistributed, "prefer-distributed", false, "Prefer distributed workers over local LLMs when available")
	flag.StringVar(&configFile, "config", "", "Configuration file path")
	flag.StringVar(&configFile, "c", "", "Configuration file path (shorthand)")
	flag.BoolVar(&hashCodebase, "hash-codebase", false, "Calculate codebase hash and exit")

	flag.Parse()

	// Track which flags the user explicitly set so config defaults never
	// silently override an explicit CLI choice (e.g. -provider openai must win
	// over config.DefaultProvider). flag.Visit only reports flags actually set.
	explicitProvider := false
	explicitAPIKey := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "provider" || f.Name == "p" {
			explicitProvider = true
		}
		if f.Name == "api-key" {
			explicitAPIKey = true
		}
	})

	// Handle version
	if showVersion {
		fmt.Printf("Universal Ebook Translator v%s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp || (inputFile == "" && createConfig == "" && !hashCodebase) {
		printHelp()
		os.Exit(0)
	}

	// Handle hash-codebase calculation
	if hashCodebase {
		hasher := versionpkg.NewCodebaseHasher()
		hash, err := hasher.CalculateHash()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating codebase hash: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(hash)
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
		// Default to English (widely used target language)
		targetLang = language.English
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

	// Load configuration if specified
	var appConfig *config.Config
	if configFile != "" {
		var err error
		appConfig, err = config.LoadConfig(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded configuration from: %s\n", configFile)
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

	// Sanitize the metadata title BEFORE any translation. Some parsers (notably
	// the TXT parser) fall back to using the raw input file PATH as the book
	// title. That path would then be sent through the translator as if it were
	// real title text and leak into the output's <dc:title> metadata field — a
	// user-visible defect (file path shown as the book title). Replace a
	// path-derived title with a clean, human-readable title derived from the
	// file's base name so only genuine title text is ever translated.
	book.Metadata.Title = sanitizeBookTitle(book.Metadata.Title, inputFile)

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
		appConfig,
		sourceLang,
		targetLang,
		eventBus,
		disableLocalLLMs,
		preferDistributed,
		explicitProvider,
		explicitAPIKey,
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
	appConfig *config.Config,
	sourceLang, targetLang language.Language,
	eventBus *events.EventBus,
	disableLocalLLMs, preferDistributed, explicitProvider, explicitAPIKey bool,
) error {
	ctx := context.Background()

	// Load configuration if specified
	if appConfig != nil {
		fmt.Printf("Using loaded configuration\n")

		// Override CLI parameters with config values if not explicitly set.
		// The CLI flag wins over the config default (see resolveProvider).
		originalProvider := providerName
		providerName = resolveProvider(providerName, explicitProvider, appConfig.Translation.DefaultProvider)
		if model == "" && appConfig.Translation.DefaultModel != "" {
			model = appConfig.Translation.DefaultModel
		}

		// If config resolution switched us to a DIFFERENT provider than the one
		// the env key was pre-loaded for (main loads getAPIKeyFromEnv for the
		// CLI-default provider BEFORE config is read), the pre-loaded key belongs
		// to the wrong provider and MUST NOT be treated as user-supplied. Drop it
		// so the per-provider config key / per-provider env key path below can
		// resolve the correct credential. An explicit -api-key always wins and is
		// never dropped.
		if !explicitAPIKey && providerName != originalProvider {
			apiKey = ""
		}

		// Load provider-specific config
		if providerConfig, ok := appConfig.Translation.Providers[providerName]; ok {
			if apiKey == "" && providerConfig.APIKey != "" {
				apiKey = providerConfig.APIKey
			}
			if baseURL == "" && providerConfig.BaseURL != "" {
				baseURL = providerConfig.BaseURL
			}
			if model == "" && providerConfig.Model != "" {
				model = providerConfig.Model
			}
		}

		// Last-resort: if still no key and the provider changed, consult the
		// environment variable for the RESOLVED provider (not the CLI default).
		if apiKey == "" && !explicitAPIKey && providerName != originalProvider {
			apiKey = getAPIKeyFromEnv(providerName)
		}
	}

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

	// Open the LLMsVerifier bridge ONCE (R-1a/R2): every translator built below —
	// the multi-LLM ensemble AND the single fallback — now obtains its model(s)
	// from the strongest verified model(s) the bridge selects, NOT a local/
	// hardcoded provider. With no provider API keys present bridge.Open returns an
	// honest hard error and we fail loudly; we NEVER silently fall back to a local
	// runtime (§11.4.69). The -provider/-model/-api-key flags remain compiling and
	// advisory under R2 (the bridge selects the verified model); -disable-local-llms
	// / -prefer-distributed still thread through the coordinator (removal is R-2/R-4).
	bridgeCtx, bridgeCancel := context.WithTimeout(context.Background(), bridgeOpenTimeout)
	b, bridgeErr := bridge.Open(bridgeCtx, bridge.Options{})
	bridgeCancel()
	if bridgeErr != nil {
		return fmt.Errorf("LLMsVerifier bridge unavailable (no local-runtime fallback): %w", bridgeErr)
	}
	task := selection.TaskRequirements{SourceLang: sourceLang.Code, TargetLang: targetLang.Code}

	// Try multi-LLM first if provider is "multi-llm", "distributed" or not specified
	if providerName == "multi-llm" || providerName == "distributed" || providerName == "" {
		// For distributed provider, prefer distributed workers if enabled
		if providerName == "distributed" && appConfig != nil && appConfig.Distributed.Enabled {
			preferDistributed = true
			fmt.Printf("Distributed translation enabled, preferring remote workers\n")
		}

		// Source the coordinator's ensemble from the bridge's provider-diverse
		// verified translators instead of the built-in per-provider discovery.
		multiTrans, multiErr := coordination.NewMultiLLMTranslatorWrapperWithFactory(
			config, eventBus, sessionID, disableLocalLLMs, preferDistributed, b.EnsembleFactory(task))
		if multiErr == nil {
			trans = multiTrans
			fmt.Printf("Using translator: multi-llm-coordinator (%d instances, bridge=%s)\n\n", multiTrans.Coordinator.GetInstanceCount(), b.Source())
		} else if providerName == "multi-llm" || providerName == "distributed" {
			// User explicitly requested multi-llm or distributed but it failed
			return fmt.Errorf("failed to create multi-LLM translator: %w", multiErr)
		}
		// Otherwise fall through to single translator
	}

	// Fall back to single translator — the STRONGEST verified model the bridge
	// selects (R-1a/R2), not llm.NewLLMTranslator on a local/hardcoded provider.
	if trans == nil {
		trans, err = b.BestTranslatorFunc(task)(ctx)
		if err != nil {
			return fmt.Errorf("failed to obtain verified translator from bridge: %w", err)
		}
		fmt.Printf("Using translator: %s (bridge=%s)\n\n", trans.GetName(), b.Source())
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

	// Write output in requested format. writeOutput guarantees that a write
	// failure never leaves a partial/empty file at the user's output path and
	// never truncates a pre-existing file (it writes to a sibling temp file and
	// atomically renames into place only on full success).
	fmt.Printf("Writing output file...\n")
	if err := writeOutput(book, outputFile, outputFormat); err != nil {
		return err
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

// writeOutput writes the book to outputFile in the requested format with an
// all-or-nothing guarantee (§11.4 / §11.4.1 — a failed run must never leave a
// partial/empty output the user mistakes for success):
//
//   - The format is validated BEFORE any filesystem write, so an unsupported
//     format never touches the output path.
//   - The content is written to a temporary sibling file (same directory, so the
//     final os.Rename is atomic on POSIX). A mid-write failure removes that temp
//     file and leaves the user's pre-existing output (if any) untouched.
//   - Only on FULL success is the temp file atomically renamed into place,
//     replacing any stale prior file with complete, fresh content.
//
// Net effect: the caller-visible output path holds either the user's prior file
// (on failure) or a complete fresh file (on success) — never a partial/empty
// half-written artefact.
func writeOutput(book *ebook.Book, outputFile, outputFormat string) error {
	outFormat := format.ParseFormat(outputFormat)

	// Validate the format up front so an unsupported format never creates or
	// truncates anything at the output path.
	switch outFormat {
	case format.FormatEPUB, format.FormatFB2, format.FormatTXT:
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	dir := filepath.Dir(outputFile)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(outputFile)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary output file: %w", err)
	}
	tmpName := tmp.Name()
	// We only need the path; the format writers open the file themselves.
	_ = tmp.Close()

	// Best-effort removal of the temp file on every failure path. On success the
	// rename consumes it, so this os.Remove becomes a no-op.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	switch outFormat {
	case format.FormatEPUB:
		if err := ebook.NewEPUBWriter().Write(book, tmpName); err != nil {
			return fmt.Errorf("failed to write EPUB: %w", err)
		}
	case format.FormatFB2:
		if err := ebook.NewFB2Writer().Write(book, tmpName); err != nil {
			return fmt.Errorf("failed to write FB2: %w", err)
		}
	case format.FormatTXT:
		if err := writeAsText(book, tmpName); err != nil {
			return fmt.Errorf("failed to write TXT: %w", err)
		}
	}

	// Atomically move the fully-written temp file into place.
	if err := os.Rename(tmpName, outputFile); err != nil {
		return fmt.Errorf("failed to finalize output file %q: %w", outputFile, err)
	}
	committed = true
	return nil
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

// resolveProvider applies config-default precedence for the translation
// provider. The CLI flag wins: the config DefaultProvider is only adopted when
// the user did NOT explicitly pass -provider/-p. Previously this used
// `providerName == "openai"` as a proxy for "unset", which silently overrode an
// explicit `-provider openai` with the config's DefaultProvider.
func resolveProvider(cliProvider string, explicitlySet bool, configDefault string) string {
	if !explicitlySet && configDefault != "" {
		return configDefault
	}
	return cliProvider
}

func getAPIKeyFromEnv(provider string) string {
	envMappings := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"anthropic": "ANTHROPIC_API_KEY",
		"zhipu":     "ZHIPU_API_KEY",
		"deepseek":  "DEEPSEEK_API_KEY",
		"qwen":      "QWEN_API_KEY",
		"gemini":    "GEMINI_API_KEY",
	}

	if envVar, ok := envMappings[provider]; ok {
		return os.Getenv(envVar)
	}

	return ""
}

// sanitizeBookTitle returns a clean, human-readable book title, replacing a
// parser-supplied title that is actually the input file PATH with a title
// derived from the file's base name. Genuine titles (anything that is not the
// raw input path) are returned unchanged.
//
// The TXT parser uses the input filename as the metadata title; for an input
// like "/Volumes/T7/books/my_book.txt" that path would otherwise be translated
// and leak into the output <dc:title>. We detect that case (title equals the
// input path, or is empty) and substitute a derived title: base name, extension
// stripped, '_'/'-' turned into spaces, trimmed.
func sanitizeBookTitle(title, inputFile string) string {
	cleaned := titleFromPath(inputFile)

	if title == "" {
		return cleaned
	}

	// Title leaked the file path if it matches the input path exactly, the
	// cleaned absolute form of it, or just its base name (path-as-title).
	base := filepath.Base(inputFile)
	if title == inputFile || title == base || pathsEqual(title, inputFile) {
		return cleaned
	}

	// A title that still looks like a filesystem path (contains a separator and
	// a known ebook extension) is treated as a leaked path as well.
	if looksLikePath(title) {
		return cleaned
	}

	return title
}

// titleFromPath derives a human-readable title from a file path: base name,
// extension removed, separators normalised to spaces, collapsed and trimmed.
func titleFromPath(inputFile string) string {
	base := filepath.Base(inputFile)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.Join(strings.Fields(base), " ")
	return strings.TrimSpace(base)
}

// pathsEqual reports whether two path strings refer to the same file, comparing
// cleaned absolute forms when resolvable (falls back to a cleaned compare).
func pathsEqual(a, b string) bool {
	ca, cb := filepath.Clean(a), filepath.Clean(b)
	if ca == cb {
		return true
	}
	if aa, err := filepath.Abs(a); err == nil {
		if ab, err := filepath.Abs(b); err == nil {
			return aa == ab
		}
	}
	return false
}

// looksLikePath reports whether a string is a filesystem path rather than a
// genuine book title: it contains a path separator AND ends with a known ebook
// file extension.
func looksLikePath(s string) bool {
	if !strings.ContainsRune(s, filepath.Separator) && !strings.ContainsRune(s, '/') {
		return false
	}
	switch strings.ToLower(filepath.Ext(s)) {
	case ".txt", ".epub", ".fb2", ".html", ".htm", ".pdf", ".docx":
		return true
	}
	return false
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
  -i, -input <file>       Input ebook file (any format: FB2, EPUB, TXT, HTML, PDF, DOCX)
  -o, -output <file>      Output file (auto-generated if not specified)
  -f, -format <format>    Output format (epub, fb2, txt) [default: epub]

  -locale <code>          Target language locale (e.g., sr, de, fr, es)
  -language <name>        Target language name (e.g., English, Spanish, French)
                          (case-insensitive, default: English)
  -source <lang>          Source language (optional, auto-detected)
  -detect                 Detect source language and exit

  -p, -provider <name>    Translation provider (openai, anthropic,
                          zhipu, deepseek, qwen, ollama, llamacpp) [default: openai]
  -model <name>           LLM model name (e.g., gpt-4, claude-3-sonnet)
  -api-key <key>          API key for LLM provider
  -base-url <url>         Base URL for LLM provider

  -script <type>          Output script (default, cyrillic, latin, arabic, etc.)
                          [default: default]

   -c, -config <file>      Configuration file path
   -create-config <file>   Create a config file template
   -disable-local-llms     Disable local LLM providers (Ollama), use only API providers
   -prefer-distributed     Prefer distributed workers over local LLMs (when available)
   -v, -version            Show version
   -h, -help               Show this help

Supported Input Formats:
  FB2, EPUB, TXT, HTML, PDF, DOCX

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
