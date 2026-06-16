package main

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flag"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"digital.vasic.translator/pkg/verification"
	"digital.vasic.translator/pkg/version"
)

const (
	// appVersion is sourced from the single authoritative version.AppVersion (== VERSION file).
	appVersion = version.AppVersion
)

// UnifiedConfig holds the configuration for the unified translation system
type UnifiedConfig struct {
	// Input/Output
	InputFile  string
	OutputFile string

	// Translation Settings
	SourceLang string
	TargetLang string
	Script     string // cyrillic, latin

	// Provider Selection
	Provider string // openai, anthropic, zhipu, deepseek, qwen, gemini

	// API/Local LLM Configuration
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration

	// Execution Options
	Workers      int
	ChunkSize    int
	Concurrency  int
	VerifyOutput bool
	Verbose      bool

	// Multi-pass verification/polishing (pkg/verification engine)
	MultiPass      bool // run the multi-pass LLM verification/polishing engine
	MultiPassCount int  // number of polishing passes (default 1 — TEaR optimum)
	MultiPassDB    string

	// Monitoring
	EnableMonitoring bool
	MonitoringPort   int

	// LLMsVerifier Integration
	VerifierEnabled bool
	VerifierURL     string
	VerifierAPIKey  string
}

// TranslationSession tracks a translation session
type TranslationSession struct {
	ID        string
	Config    *UnifiedConfig
	StartTime time.Time
	EndTime   time.Time
	EventBus  *events.EventBus
	Logger    logger.Logger
	Files     []GeneratedFile
	Steps     []*TranslationStep
}

// GeneratedFile tracks files generated during translation
type GeneratedFile struct {
	Path         string
	Type         string // original_md, translated_md, epub
	Size         int64
	Verified     bool
	Verification string
}

// TranslationStep tracks each translation step
type TranslationStep struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Error     string
	Details   string
}

func main() {
	// Parse configuration
	config := parseFlags()

	// Initialize logger
	logLevel := logger.INFO
	if config.Verbose {
		logLevel = logger.DEBUG
	}

	logger := logger.NewLogger(logger.LoggerConfig{
		Level:  logLevel,
		Format: logger.FORMAT_TEXT,
	})

	// Initialize monitoring
	eventBus := events.NewEventBus()

	// Create translation session
	session := &TranslationSession{
		ID:        generateSessionID(),
		Config:    config,
		StartTime: time.Now(),
		EventBus:  eventBus,
		Logger:    logger,
		Files:     make([]GeneratedFile, 0),
		Steps:     make([]*TranslationStep, 0),
	}

	// Start monitoring server if requested
	if config.EnableMonitoring {
		go startMonitoringServer(config.MonitoringPort, eventBus)
		logger.Info("Monitoring server started", map[string]interface{}{
			"port":       config.MonitoringPort,
			"session_id": session.ID,
		})
	}

	// Execute translation
	err := executeTranslation(session)

	// Finalize session
	session.EndTime = time.Now()

	if err != nil {
		logger.Error("Translation failed", map[string]interface{}{
			"error":      err.Error(),
			"session_id": session.ID,
		})

		// Generate error report
		generateSessionReport(session, err)
		os.Exit(1)
	}

	logger.Info("Translation completed successfully", map[string]interface{}{
		"input":      config.InputFile,
		"output":     config.OutputFile,
		"provider":   config.Provider,
		"duration":   session.EndTime.Sub(session.StartTime).String(),
		"session_id": session.ID,
	})

	// Generate success report
	generateSessionReport(session, nil)
}

// executeTranslation performs the translation using the specified provider
func executeTranslation(session *TranslationSession) error {
	config := session.Config
	ctx := context.Background()

	// Step 1: Parse input ebook
	step := addStep(session, "Input Parsing")
	ebookContent, format, err := parseInputFile(config.InputFile)
	if err != nil {
		return stepError(step, fmt.Sprintf("Failed to parse input file: %v", err))
	}
	step.Details = fmt.Sprintf("Parsed %s format, %d characters", format, len(ebookContent))
	stepComplete(step)

	// Step 2: Convert to markdown
	step = addStep(session, "Markdown Conversion")
	originalMarkdown, err := convertToMarkdown(ebookContent, format)
	if err != nil {
		return stepError(step, fmt.Sprintf("Failed to convert to markdown: %v", err))
	}

	// Save original markdown
	originalMDPath := generateOriginalMDPath(config.InputFile)
	if err := os.WriteFile(originalMDPath, []byte(originalMarkdown), 0644); err != nil {
		return stepError(step, fmt.Sprintf("Failed to save original markdown: %v", err))
	}

	addFile(session, originalMDPath, "original_md", int64(len(originalMarkdown)), true, "Saved successfully")
	step.Details = fmt.Sprintf("Converted to markdown, saved to %s", originalMDPath)
	stepComplete(step)

	// Step 3: Translate based on provider
	step = addStep(session, fmt.Sprintf("Translation (%s)", config.Provider))
	translatedMarkdown, err := executeProviderTranslation(ctx, config, session, originalMarkdown)
	if err != nil {
		return stepError(step, fmt.Sprintf("Translation failed: %v", err))
	}

	// Normalize to the requested target script (W10) — but ONLY when the target
	// language is Serbian. The script converter is Serbian Cyrillic<->Latin
	// specific; applying it to any other target (e.g. Spanish) transliterates the
	// already-correct translation into the wrong alphabet (Latin "Hola mundo" ->
	// Cyrillic "Хола мундо" = garbage). For non-Serbian targets the LLM output is
	// already in the correct script, so the conversion MUST be skipped (§11.4.6:
	// the -script flag is a Serbian-context control, not a universal transliterator).
	translatedMarkdown = applyTargetScript(translatedMarkdown, config.TargetLang, config.Script)

	// Step 3b (opt-in): multi-pass LLM verification/polishing. Wires the
	// previously-dormant pkg/verification engine (§11.4.124 — investigate then
	// wire-in-properly). Runs ONLY when -multipass is set so the default path is
	// unchanged (§11.4.1 no solve-A-create-B). The engine gates each pass on a
	// real per-provider LLM critique and preserves the original when no genuine
	// improvement is found (TEaR-style; see research artefact), so it never
	// silently degrades the translation.
	if config.MultiPass {
		mpStep := addStep(session, "Multi-pass Polishing")
		polished, polishErr := runMultiPass(ctx, config, session, originalMarkdown, translatedMarkdown)
		if polishErr != nil {
			// A multipass failure must not destroy the already-good base
			// translation: log it, keep the pre-polish text, mark the step.
			mpStep.Details = fmt.Sprintf("Multi-pass polishing failed, keeping base translation: %v", polishErr)
			stepComplete(mpStep)
		} else {
			translatedMarkdown = polished
			mpStep.Details = fmt.Sprintf("Polished over %d pass(es) with %s", config.MultiPassCount, config.Provider)
			stepComplete(mpStep)
		}
	}

	// Save translated markdown
	translatedMDPath := generateTranslatedMDPath(config.InputFile)
	if err := os.WriteFile(translatedMDPath, []byte(translatedMarkdown), 0644); err != nil {
		return stepError(step, fmt.Sprintf("Failed to save translated markdown: %v", err))
	}

	// Verify translation quality
	verified := verifyTranslation(translatedMarkdown, config.TargetLang, config.Script)
	addFile(session, translatedMDPath, "translated_md", int64(len(translatedMarkdown)), verified,
		map[bool]string{true: "Translation quality verified", false: "Translation needs review"}[verified])

	step.Details = fmt.Sprintf("Translated with %s, saved to %s", config.Provider, translatedMDPath)
	stepComplete(step)

	// Step 4: Convert to the requested output format (honors the -o extension)
	step = addStep(session, "Output Generation")
	outPath := config.OutputFile
	if err := generateOutput(translatedMarkdown, outPath, config.InputFile); err != nil {
		return stepError(step, fmt.Sprintf("output generation failed: %v", err))
	}

	// Verify the produced file. EPUB has a structural check; for the other
	// formats a non-empty file is the verification (the content is the translated
	// text written directly / via the FB2 writer).
	outFmt := strings.ToLower(strings.TrimPrefix(filepath.Ext(outPath), "."))
	if outFmt == "" {
		outFmt = "epub"
	}
	var outVerified bool
	if outFmt == "epub" {
		outVerified = verifyEPUB(outPath)
	} else {
		outVerified = getFileSize(outPath) > 0
	}
	outSize := getFileSize(outPath)
	addFile(session, outPath, outFmt, outSize, outVerified,
		map[bool]string{true: "Valid " + outFmt + " output", false: "Invalid " + outFmt + " output"}[outVerified])

	step.Details = fmt.Sprintf("Generated %s: %s", outFmt, outPath)
	stepComplete(step)

	return nil
}

// executeProviderTranslation handles translation based on the selected provider
func executeProviderTranslation(ctx context.Context, config *UnifiedConfig, session *TranslationSession, text string) (string, error) {
	switch config.Provider {
	case "ssh":
		// The SSH-local translation path (remote SSH worker running llama.cpp)
		// was removed in bridge phase-2 R-4 (operator-confirmed: "keep
		// distributed API, remove only SSH-local"). It is the project's only
		// former local-runtime path. provider=ssh MUST hard-error honestly
		// rather than silently routing to a DIFFERENT (API/bridge) provider —
		// a silent wrong-provider fallback is forbidden (§11.4.69).
		return "", fmt.Errorf("provider=ssh is no longer supported: the SSH-local translation path was removed (bridge phase-2 R-4); use an API provider (openai, anthropic, zhipu, deepseek, qwen, gemini)")
	default:
		// No local-runtime path: any provider routes to the API path, which
		// sources the strongest verified model via the bridge and hard-errors
		// honestly when no API key is set (§11.4.69). There is NEVER a silent
		// local-runtime fallback.
		return executeAPITranslation(ctx, config, session, text)
	}
}

// executeAPITranslation uses API-based LLM providers
func executeAPITranslation(ctx context.Context, config *UnifiedConfig, session *TranslationSession, text string) (string, error) {
	session.Logger.Info("Starting API-based translation", map[string]interface{}{
		"provider": config.Provider,
		"model":    config.Model,
	})

	// trans is the translator.Translator surface every branch produces; the
	// caller (executeProviderTranslation) needs only TranslateWithProgress, which
	// the interface satisfies, so the concrete *llm.LLMTranslator return type is
	// no longer required.
	var trans translator.Translator
	var err error

	switch {
	case config.Provider == "mock":
		// The mock provider is a deliberate in-process test/demo seam (no network,
		// no API key) — it MUST NOT route through the bridge (which requires real
		// verified models). Build it directly via the llm factory, unchanged.
		trans, err = llm.NewLLMTranslator(translator.TranslationConfig{
			SourceLang: config.SourceLang,
			TargetLang: config.TargetLang,
			Provider:   "mock",
			Model:      config.Model,
		})
		if err != nil {
			return "", fmt.Errorf("failed to create mock translator: %w", err)
		}
	case config.VerifierEnabled:
		// CONST-034: explicit -use-verifier keeps the HTTP-client VerifiedFactory
		// path (operator-provided LLMsVerifier service via -verifier-url).
		var lt *llm.LLMTranslator
		lt, err = executeVerifiedTranslation(ctx, config, session)
		if err != nil {
			return "", fmt.Errorf("verified translation failed: %w", err)
		}
		trans = lt
	default:
		// R-1/R2 default: source the strongest verified model from the LLMsVerifier
		// bridge — NO local runtime, NO hardcoded provider. On no API keys the
		// bridge hard-errors honestly (§11.4.69); there is NEVER a silent local
		// llama.cpp fallback. The legacy direct-provider NewLLMTranslator
		// path is replaced by this bridge selection.
		trans, err = bridgeTranslator(ctx, config)
		if err != nil {
			return "", fmt.Errorf("bridge translation unavailable (no local-runtime fallback): %w", err)
		}
	}

	// Translate
	result, err := trans.TranslateWithProgress(ctx, text, "Ebook content", session.EventBus, session.ID)
	if err != nil {
		return "", fmt.Errorf("API translation failed: %w", err)
	}

	return result, nil
}

// bridgeOpener opens the underlying LLMsVerifier bridge. It is a package-level
// seam so tests can install a sentinel opener (e.g. a t.Fatal opener) to assert
// — env-independently — that a code path NEVER opens the bridge (the mock seam),
// without depending on the host's provider-key / network state (§11.4.3). The
// default opens the real bridge with default options.
var bridgeOpener = func(ctx context.Context) (*bridge.Bridge, error) {
	return bridge.Open(ctx, bridge.Options{})
}

// bridgeTranslator opens the LLMsVerifier bridge and returns the strongest
// verified translator for the run's language pair. It is the R-1/R2 default
// source for unified-translator's API path: no local runtime, honest hard error
// when no provider API key + no verified model is available (§11.4.69). The
// in-process bridge is bounded by config.VerifyTimeout-equivalent defaults.
func bridgeTranslator(ctx context.Context, config *UnifiedConfig) (translator.Translator, error) {
	b, err := bridgeOpener(ctx)
	if err != nil {
		return nil, err
	}
	task := selection.TaskRequirements{
		SourceLang: config.SourceLang,
		TargetLang: config.TargetLang,
	}
	tr, _, err := b.BestTranslator(ctx, task)
	if err != nil {
		return nil, err
	}
	return tr, nil
}

// executeVerifiedTranslation uses VerifiedFactory to select and translate with a verified model.
func executeVerifiedTranslation(ctx context.Context, config *UnifiedConfig, session *TranslationSession) (*llm.LLMTranslator, error) {
	vCfg := initVerifierConfig(config)
	factory := llm.NewVerifiedFactory(vCfg)
	factory.SetKeyResolver(func(providerID string) string {
		return resolveProviderAPIKey(config, providerID)
	})

	// Seed registry from local config for offline operation
	// Wire LLMsVerifier client as the single source of truth (SSOT).
	// The factory will fetch verified models from LLMsVerifier at runtime.
	client := verifier.NewClient(vCfg)
	factory.SetClient(client)

	task := selection.TaskRequirements{
		SourceLang: config.SourceLang,
		TargetLang: config.TargetLang,
	}

	trans, fallbackIDs, err := factory.CreateTranslatorWithFallback(ctx, task)
	if err != nil {
		return nil, err
	}

	session.Logger.Info("Verified model selected", map[string]interface{}{
		"provider":   trans.GetName(),
		"fallbacks":  fallbackIDs,
		"session_id": session.ID,
	})

	return trans, nil
}

// resolveProviderAPIKey returns the API key for a given provider ID.
func resolveProviderAPIKey(config *UnifiedConfig, providerID string) string {
	// If user explicitly selected this provider, use their API key
	if providerID == config.Provider && config.APIKey != "" {
		return config.APIKey
	}
	// Fall back to well-known environment variables
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
	if envVar, ok := envMap[providerID]; ok {
		return os.Getenv(envVar)
	}
	return ""
}

// requiresProviderKey reports whether the run will use a per-provider LLM client
// that needs Provider's API key, versus the default LLMsVerifier-bridge path that
// keys its own verified models. It mirrors executeAPITranslation's runtime switch
// EXACTLY (main.go ~line 330):
//   - mock provider: builds a direct mock translator — no key, no network.
//   - -use-verifier (VerifierEnabled): executeVerifiedTranslation builds a
//     per-provider client via VerifiedFactory + resolveProviderAPIKey — the ONLY
//     runtime path that consumes Provider's key.
//   - default (everything else, INCLUDING an explicit -provider): routes to the
//     LLMsVerifier bridge, which selects + keys its OWN verified models and
//     ignores config.Provider; it hard-errors honestly when none verify (§11.4.69).
//
// Crucially, an explicit `-provider X` (without -use-verifier) still hits the
// default bridge case — there is NO direct per-provider runtime path — so the gate
// must NOT fire on it. The sole trigger is VerifierEnabled. Keeping this a pure
// function makes the gate unit-testable without os.Exit.
func requiresProviderKey(config *UnifiedConfig) bool {
	if config.Provider == "mock" {
		return false
	}
	return config.VerifierEnabled
}

// Helper functions

func parseFlags() *UnifiedConfig {
	config := &UnifiedConfig{
		SourceLang:     "ru",
		TargetLang:     "sr",
		Script:         "cyrillic",
		Provider:       "openai",
		Model:          "gpt-4",
		Temperature:    0.3,
		MaxTokens:      4096,
		Timeout:        30 * time.Second,
		Workers:        1,
		ChunkSize:      2000,
		Concurrency:    4,
		VerifyOutput:   true,
		MonitoringPort: 8080,
	}

	flag.StringVar(&config.InputFile, "input", "", "Input ebook file")
	flag.StringVar(&config.InputFile, "i", "", "Input ebook file (shorthand)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file (auto-detected if not specified)")
	flag.StringVar(&config.OutputFile, "o", "", "Output file (shorthand)")

	flag.StringVar(&config.SourceLang, "source-lang", "ru", "Source language (default: ru)")
	flag.StringVar(&config.TargetLang, "target-lang", "sr", "Target language (default: sr)")
	flag.StringVar(&config.Script, "script", "cyrillic", "Target script: cyrillic, latin (default: cyrillic)")

	flag.StringVar(&config.Provider, "provider", "openai", "Translation provider: openai, anthropic, zhipu, deepseek, qwen, gemini")
	flag.StringVar(&config.Model, "model", "gpt-4", "Model name")
	flag.StringVar(&config.APIKey, "api-key", "", "API key for provider")
	flag.StringVar(&config.BaseURL, "base-url", "", "Base URL for provider (if needed)")
	flag.Float64Var(&config.Temperature, "temperature", 0.3, "LLM temperature")
	flag.IntVar(&config.MaxTokens, "max-tokens", 4096, "Maximum tokens")
	flag.DurationVar(&config.Timeout, "timeout", 30*time.Second, "Request timeout")

	// Execution options
	flag.IntVar(&config.Workers, "workers", 1, "Number of parallel workers")
	flag.IntVar(&config.ChunkSize, "chunk-size", 2000, "Text chunk size")
	flag.IntVar(&config.Concurrency, "concurrency", 4, "Maximum concurrent operations")
	flag.BoolVar(&config.VerifyOutput, "verify", true, "Verify translated output")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	// Multi-pass LLM verification/polishing (opt-in; costs real LLM calls).
	// Separate from -verify (which stays the lightweight heuristic check) so
	// existing behaviour is preserved. Default passes = 1 (TEaR best-practice
	// optimum; later passes regress — see docs/research/.../verification_multipass_wiring.md).
	flag.BoolVar(&config.MultiPass, "multipass", false,
		"Run the multi-pass LLM verification/polishing engine after translation (opt-in, uses extra LLM calls)")
	flag.IntVar(&config.MultiPassCount, "multipass-passes", 1, "Number of multi-pass polishing passes (default 1)")
	flag.StringVar(&config.MultiPassDB, "multipass-db", "", "Optional SQLite path to persist multi-pass polishing reports")

	// Monitoring options
	flag.BoolVar(&config.EnableMonitoring, "monitoring", false, "Enable web monitoring")
	flag.IntVar(&config.MonitoringPort, "monitoring-port", 8080, "Monitoring server port")

	// LLMsVerifier options
	flag.BoolVar(&config.VerifierEnabled, "use-verifier", false, "Use LLMsVerifier as single source of truth for model selection")
	flag.StringVar(&config.VerifierURL, "verifier-url", "http://localhost:8080", "LLMsVerifier API URL")
	flag.StringVar(&config.VerifierAPIKey, "verifier-api-key", os.Getenv("LLMSVERIFIER_API_KEY"), "LLMsVerifier API key")

	versionFlag := flag.Bool("version", false, "Show version information")
	help := flag.Bool("help", false, "Show help information")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Unified Translator v%s\n", appVersion)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Validate required arguments
	if config.InputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Input file is required\n")
		printHelp()
		os.Exit(1)
	}

	// Auto-detect output file if not specified
	if config.OutputFile == "" {
		config.OutputFile = generateOutputFilename(config.InputFile, config.TargetLang)
	}

	// Provider-specific validation
	switch config.Provider {
	case "ssh":
		// SSH-local translation was removed (bridge phase-2 R-4). Reject early
		// with an honest error rather than silently falling back to an API
		// provider (§11.4.69 — never a silent wrong-provider fallback).
		fmt.Fprintf(os.Stderr, "Error: provider=ssh is no longer supported (SSH-local path removed in bridge phase-2 R-4); use an API provider (openai, anthropic, zhipu, deepseek, qwen, gemini)\n")
		os.Exit(1)
	case "mock":
		// No API key required for the mock offline test seam.
	default:
		// Fall back to the provider's well-known env var (e.g. DEEPSEEK_API_KEY)
		// when -api-key was not passed. The gate previously checked ONLY the flag,
		// so env-var keys — which resolveProviderAPIKey already supports and the
		// docs advertise — never satisfied it. Populate config.APIKey so the whole
		// pipeline (provider client construction) uses the resolved key too.
		if config.APIKey == "" {
			config.APIKey = resolveProviderAPIKey(config, config.Provider)
		}
		// Require a per-provider key ONLY for runs that actually use a
		// per-provider client. The default-translate path routes to the
		// LLMsVerifier bridge (see executeAPITranslation's default switch case),
		// which selects + keys its own verified models and hard-errors honestly
		// (§11.4.69) when none are available — it never consumes Provider's key.
		// A bare default run (Provider left at "openai", no -use-verifier) must
		// therefore NOT be blocked by this gate. Found by §11.4.153 video wave 2;
		// reproduce-first RED → GREEN per §11.4.115/§11.4.146.
		if requiresProviderKey(config) && config.APIKey == "" {
			fmt.Fprintf(os.Stderr, "Error: API key required for provider=%s (pass -api-key or set the provider's *_API_KEY env var)\n", config.Provider)
			os.Exit(1)
		}
	}

	// LLMsVerifier model validation (if enabled)
	if config.VerifierEnabled {
		if err := validateWithVerifier(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: LLMsVerifier validation failed: %v\n", err)
			os.Exit(1)
		}
	}

	return config
}

func printHelp() {
	fmt.Printf(`Unified Translator v%s - Multi-Provider Ebook Translation Tool

Usage:
  unified-translator -input <file> -provider <provider> [options]

Providers:
  openai      - OpenAI GPT models (requires API key)
  anthropic   - Anthropic Claude models (requires API key)
  zhipu       - Zhipu AI models (requires API key)
  deepseek    - DeepSeek models (requires API key)
  qwen        - Qwen models (requires API key)
  gemini      - Google Gemini models (requires API key)

Basic Options:
  -i, -input <file>        Input ebook file (FB2, EPUB, PDF, DOCX, TXT, HTML)
  -o, -output <file>       Output file (auto-detected if not specified)
  -source-lang <lang>       Source language (default: ru)
  -target-lang <lang>       Target language (default: sr)
  -script <script>          Target script: cyrillic, latin (default: cyrillic)

Provider Configuration:
  -provider <provider>      Translation provider (default: openai)
  -model <model>            Model name (default: gpt-4)
  -api-key <key>            API key for provider
  -base-url <url>           Base URL for provider (if needed)
  -temperature <value>      LLM temperature (default: 0.3)
  -max-tokens <num>         Maximum tokens (default: 4096)
  -timeout <duration>       Request timeout (default: 30s)

Execution Options:
  -workers <num>            Parallel workers (default: 1)
  -chunk-size <size>         Text chunk size (default: 2000)
  -concurrency <num>         Concurrent operations (default: 4)
  -verify                   Verify output (default: true)
  -multipass                Run multi-pass LLM verification/polishing after translation (opt-in)
  -multipass-passes N       Number of multi-pass polishing passes (default: 1)
  -multipass-db PATH        Optional SQLite path to persist multi-pass polishing reports
  -verbose                  Enable verbose logging

Monitoring:
  -monitoring               Enable web monitoring
  -monitoring-port <port>   Monitoring server port (default: 8080)

LLMsVerifier (CONST-034):
  -use-verifier             Use LLMsVerifier as single source of truth for model selection
  -verifier-url <url>       LLMsVerifier API URL (default: http://localhost:8080)
  -verifier-api-key <key>   LLMsVerifier API key

Other:
  -version                  Show version information
  -help                     Show this help

Examples:
  # Translate with OpenAI
  unified-translator -i book.fb2 -provider openai -api-key YOUR_KEY

  # Translate with monitoring
  unified-translator -i book.fb2 -provider openai -api-key YOUR_KEY -monitoring

Translation Flow:
  1. Parse input ebook (FB2, EPUB, TXT, HTML, DOCX, PDF)
  2. Convert to markdown format
  3. Translate using selected provider
  4. Write output in the format set by the -o extension
     (.epub default; .fb2, .html/.htm, .txt, .md, .docx honored)
  5. Verify and document results

Generated Files:
  - <name>_original.md      Original content in markdown
  - <name>_translated.md    Translated content in markdown  
  - <name>_sr.epub        Final EPUB in Serbian Cyrillic
  - <name>_session_report.md  Translation session report

Monitoring Dashboard:
  When -monitoring is enabled, access the web dashboard at:
  http://localhost:8080/session?id=<session_id>
`, appVersion)
}

func generateSessionID() string {
	return fmt.Sprintf("tx-%d", time.Now().UnixNano())
}

// generateOutputFilename derives the auto-generated output path from the input
// file AND the requested target language. The language tag MUST reflect
// targetLang — hardcoding "_sr" silently mislabelled non-Serbian translations
// (e.g. a French translation written to book_sr.epub), a §11.4 wrong-output
// defect. An empty targetLang falls back to "sr" (the CLI's default) so the
// behaviour is deterministic (§11.4.6, no guessing).
func generateOutputFilename(inputFile, targetLang string) string {
	base := filepath.Base(inputFile)
	// Trim the actual extension (case-preserving) so UPPERCASE/mixed-case
	// extensions (e.g. "Story.EPUB", "Tale.Fb2") are stripped just like
	// lowercase ones. Lowercasing the ext before TrimSuffix would not match
	// the original-case basename and leave the extension embedded in the name.
	baseName := strings.TrimSuffix(base, filepath.Ext(base))

	lang := strings.TrimSpace(targetLang)
	if lang == "" {
		lang = "sr"
	}

	return filepath.Join(filepath.Dir(inputFile), baseName+"_"+lang+".epub")
}

func generateOriginalMDPath(inputFile string) string {
	base := filepath.Base(inputFile)
	baseName := strings.TrimSuffix(base, filepath.Ext(base))

	return filepath.Join(filepath.Dir(inputFile), baseName+"_original.md")
}

func generateTranslatedMDPath(inputFile string) string {
	base := filepath.Base(inputFile)
	baseName := strings.TrimSuffix(base, filepath.Ext(base))

	return filepath.Join(filepath.Dir(inputFile), baseName+"_translated.md")
}

func addStep(session *TranslationSession, name string) *TranslationStep {
	step := &TranslationStep{
		Name:      name,
		StartTime: time.Now(),
		Success:   false,
	}
	// Append the pointer (not a value): a subsequent append may reallocate the
	// backing array, but the *TranslationStep we returned keeps pointing at the
	// same heap object, so a step held across a later addStep stays live.
	session.Steps = append(session.Steps, step)
	return step
}

func stepComplete(step *TranslationStep) {
	step.EndTime = time.Now()
	step.Success = true
}

func stepError(step *TranslationStep, err string) error {
	step.EndTime = time.Now()
	step.Success = false
	step.Error = err
	return fmt.Errorf("%s", err)
}

func addFile(session *TranslationSession, path, fileType string, size int64, verified bool, verification string) {
	session.Files = append(session.Files, GeneratedFile{
		Path:         path,
		Type:         fileType,
		Size:         size,
		Verified:     verified,
		Verification: verification,
	})
}

// Placeholder functions - these need to be implemented based on existing functionality
func parseInputFile(filePath string) (string, string, error) {
	// Use existing ebook parser
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(filePath)
	if err != nil {
		return "", "", err
	}
	return bookToString(book), book.Format.String(), nil
}

func convertToMarkdown(content, format string) (string, error) {
	// parseInputFile has ALREADY parsed the source format (FB2/EPUB/HTML/DOCX/TXT/
	// ...) into extracted text via bookToString — `content` IS that extracted text,
	// not the original file bytes. The previous fb2/epub special-cases re-wrote this
	// extracted text into a temp .fb2/.epub and re-parsed it AS that format, which
	// always failed ("failed to parse FB2: EOF" / "zip: not a valid zip file") and
	// broke FB2 and EPUB INPUT end-to-end (only TXT/HTML/DOCX, which hit the default
	// passthrough, worked). All formats now use the already-extracted content
	// uniformly. (format is retained in the signature for call-site clarity.)
	_ = format
	return content, nil
}

func bookToString(book *ebook.Book) string {
	var result strings.Builder
	for _, chapter := range book.Chapters {
		result.WriteString(chapter.Title)
		result.WriteString("\n\n")
		for _, section := range chapter.Sections {
			result.WriteString(section.Content)
			result.WriteString("\n\n")
		}
	}
	return result.String()
}

// normalizeScript deterministically converts translated text to the requested
// target script. `-script latin` always yields Latin output (no Cyrillic
// codepoints survive); `-script cyrillic` always yields Cyrillic. Conversion is
// done directly via the script converter (ToLatin/ToCyrillic) rather than the
// auto-detect Convert(), so the result is deterministic regardless of how the
// LLM mixed scripts. Both directions are idempotent on already-target text
// (target-script codepoints are not in the source mapping table, so they pass
// through untouched). An empty or unrecognised script value passes the text
// through unchanged (W10 — §11.4.6: no guessing, explicit closed-set handling).
func normalizeScript(text, targetScript string) string {
	conv := script.NewConverter()
	switch targetScript {
	case string(script.Latin):
		return conv.ToLatin(text)
	case string(script.Cyrillic):
		return conv.ToCyrillic(text)
	default:
		return text
	}
}

// isSerbianTarget reports whether the target language is Serbian — the only
// language for which the Serbian Cyrillic<->Latin script conversion is
// meaningful. Handles the common flag forms (code/name, either script,
// Cyrillic spelling). §11.4.6: explicit closed-set, no guessing.
func isSerbianTarget(targetLang string) bool {
	t := strings.ToLower(strings.TrimSpace(targetLang))
	switch t {
	case "sr", "srp", "serbian", "srpski", "српски", "serbian cyrillic", "serbian latin":
		return true
	}
	return strings.Contains(t, "serb")
}

// applyTargetScript runs the Serbian script normalization ONLY for a Serbian
// target language; for every other target it returns the text unchanged. This
// prevents the Serbian Cyrillic<->Latin converter from mangling a correct
// translation in another language (e.g. transliterating Latin Spanish into
// Serbian Cyrillic).
func applyTargetScript(text, targetLang, targetScript string) string {
	if !isSerbianTarget(targetLang) {
		return text
	}
	return normalizeScript(text, targetScript)
}

// runMultiPass runs the pkg/verification multi-pass LLM polishing engine over an
// already-translated markdown document and returns the polished markdown.
//
// The engine operates on *ebook.Book; the unified-translator pipeline works in
// markdown, so we wrap the original and translated markdown each as a single-
// chapter / single-section Book, run MultiPassPolisher.PolishBook (real per-pass
// LLM critique + consensus), and read the polished section content back out.
//
// Safety / guardrails (see docs/research/.../verification_multipass_wiring.md):
//   - Pass count defaults to 1 and is clamped to >=1 (TEaR optimum; later passes
//     regress).
//   - The engine preserves the existing translation when no genuine improvement
//     is found (POLISHED_TEXT "UNCHANGED" / default-to-current), so it cannot
//     silently degrade the base translation.
//   - If the polished result is empty (e.g. provider returned nothing useful),
//     the original translation is returned unchanged rather than wiped.
func runMultiPass(
	ctx context.Context,
	config *UnifiedConfig,
	session *TranslationSession,
	originalMarkdown, translatedMarkdown string,
) (string, error) {
	passes := config.MultiPassCount
	if passes < 1 {
		passes = 1
	}

	// Per-provider TranslationConfig the polisher uses to build its LLM clients.
	tc := translator.TranslationConfig{
		SourceLang:  config.SourceLang,
		TargetLang:  config.TargetLang,
		Provider:    config.Provider,
		Model:       config.Model,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		Timeout:     config.Timeout,
		APIKey:      config.APIKey,
		BaseURL:     config.BaseURL,
		Script:      config.Script,
	}

	mpConfig := verification.MultiPassConfig{
		PassCount:    passes,
		MinConsensus: 1,
		// Verify across all four quality dimensions the engine supports.
		VerifySpirit:     true,
		VerifyLanguage:   true,
		VerifyContext:    true,
		VerifyVocabulary: true,
		// Note-taking is an extra LLM round per section; keep it off for the CLI
		// path to bound cost — the polishing critique itself is the value here.
		EnableNoteTaking: false,
		DatabasePath:     config.MultiPassDB,
		TranslationConfigs: map[string]translator.TranslationConfig{
			config.Provider: tc,
		},
	}

	polisher, err := verification.NewMultiPassPolisher(mpConfig, session.EventBus, session.ID)
	if err != nil {
		return translatedMarkdown, fmt.Errorf("create multi-pass polisher: %w", err)
	}
	defer polisher.Close()

	origBook := markdownToSingleSectionBook(originalMarkdown)
	transBook := markdownToSingleSectionBook(translatedMarkdown)

	result, err := polisher.PolishBook(ctx, origBook, transBook)
	if err != nil {
		return translatedMarkdown, fmt.Errorf("polish book: %w", err)
	}

	polished := extractSingleSectionContent(result.FinalBook)
	if strings.TrimSpace(polished) == "" {
		// Never return empty over a real translation — keep the base text.
		return translatedMarkdown, nil
	}
	return polished, nil
}

// markdownToSingleSectionBook wraps a markdown document as a minimal one-chapter,
// one-section ebook.Book so the verification engine (which polishes per-section)
// can process it. The whole document is the single section's content.
func markdownToSingleSectionBook(md string) *ebook.Book {
	return &ebook.Book{
		Metadata: ebook.Metadata{Title: "document"},
		Chapters: []ebook.Chapter{
			{
				Title: "Content",
				Sections: []ebook.Section{
					{Title: "Content", Content: md},
				},
			},
		},
	}
}

// extractSingleSectionContent reads the polished content back out of the single
// section produced by markdownToSingleSectionBook. Returns "" if the shape is
// unexpected (caller falls back to the base translation).
func extractSingleSectionContent(book *ebook.Book) string {
	if book == nil || len(book.Chapters) == 0 || len(book.Chapters[0].Sections) == 0 {
		return ""
	}
	return book.Chapters[0].Sections[0].Content
}

func verifyTranslation(text, targetLang, script string) bool {
	// Basic verification - check for target script characters
	if targetLang == "sr" && script == "cyrillic" {
		serbianCyrillic := "љњертзуиопшђжасдфгхјклчћџ"
		for _, char := range text {
			if strings.ContainsRune(serbianCyrillic, char) {
				return true
			}
		}
		return false
	}
	return len(strings.TrimSpace(text)) > 0
}

// generateOutput writes the translated content to outputPath in the format
// indicated by the output file extension.
//
// Previously the CLI ALWAYS produced an EPUB regardless of the -o extension, so
// `-o book.txt` / `-o book.fb2` wrote EPUB bytes into a misnamed file — a silent
// wrong-output defect (§11.4: the user asked for one format and silently got
// another). The output format is now honored. An unsupported extension is an
// explicit, honest error (§11.4.6) rather than a misnamed EPUB.
//
// Supported: .epub (default / no extension), .fb2, .html, .htm, .txt, .md, .docx, .pdf.
// PDF output requires weasyprint (honest typed error when absent — see pkg/ebook/pdf_writer.go).
func generateOutput(content, outputPath, inputFile string) error {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(outputPath), "."))
	switch ext {
	case "", "epub":
		return generateEPUB(content, outputPath, inputFile)
	case "txt", "md":
		// The translated content is plain (markdown) text — write it directly.
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s output: %w", ext, err)
		}
		return nil
	case "fb2":
		book := &ebook.Book{
			Metadata: ebook.Metadata{Title: titleFromInput(inputFile)},
			Chapters: []ebook.Chapter{{
				Title:    titleFromInput(inputFile),
				Sections: []ebook.Section{{Content: content}},
			}},
		}
		return ebook.NewFB2Writer().Write(book, outputPath)
	case "html", "htm":
		return generateHTML(content, outputPath, titleFromInput(inputFile))
	case "docx":
		book := &ebook.Book{
			Metadata: ebook.Metadata{Title: titleFromInput(inputFile)},
			Chapters: []ebook.Chapter{{
				Title:    titleFromInput(inputFile),
				Sections: []ebook.Section{{Content: content}},
			}},
		}
		return ebook.NewDOCXWriter().Write(book, outputPath)
	case "pdf":
		book := &ebook.Book{
			Metadata: ebook.Metadata{Title: titleFromInput(inputFile)},
			Chapters: []ebook.Chapter{{
				Title:    titleFromInput(inputFile),
				Sections: []ebook.Section{{Content: content}},
			}},
		}
		return ebook.NewPDFWriter().Write(book, outputPath)
	default:
		return fmt.Errorf("unsupported output format %q (supported: .epub, .fb2, .html, .txt, .md, .docx, .pdf)", ext)
	}
}

// titleFromInput derives a document title from the input filename (basename
// without extension), falling back to a generic title.
func titleFromInput(inputFile string) string {
	base := filepath.Base(inputFile)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.TrimSpace(title) == "" {
		return "Translated Document"
	}
	return title
}

// generateHTML writes the translated content as a minimal, valid, well-formed
// HTML5 document. Blank-line-separated blocks become <p> paragraphs; all text
// (title + body) is HTML-escaped so translated content can never inject markup.
func generateHTML(content, outputPath, title string) error {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title>\n</head>\n<body>\n")
	for _, block := range strings.Split(content, "\n\n") {
		// Collapse single newlines within a paragraph to spaces.
		p := strings.TrimSpace(strings.ReplaceAll(block, "\n", " "))
		if p == "" {
			continue
		}
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(p))
		b.WriteString("</p>\n")
	}
	b.WriteString("</body>\n</html>\n")
	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("failed to write html output: %w", err)
	}
	return nil
}

func generateEPUB(content, outputPath, inputFile string) error {
	// Create temporary markdown file
	tmpFile := filepath.Join(os.TempDir(), "content.md")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp markdown: %w", err)
	}
	defer os.Remove(tmpFile)

	// Use existing EPUB generator
	generator := markdown.NewMarkdownToEPUBConverter()
	return generator.ConvertMarkdownToEPUB(tmpFile, outputPath)
}

func verifyEPUB(path string) bool {
	// Basic EPUB verification
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	content := string(buffer[:n])
	return strings.Contains(content, "application/epub+zip") && string(buffer[:2]) == "PK"
}

func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func startMonitoringServer(port int, eventBus *events.EventBus) {
	// This would start the monitoring server - implement based on existing code
	fmt.Printf("Monitoring server available on port %d\n", port)
}

func generateSessionReport(session *TranslationSession, err error) {
	reportPath := strings.TrimSuffix(session.Config.OutputFile, filepath.Ext(session.Config.OutputFile)) + "_session_report.md"

	file, err2 := os.Create(reportPath)
	if err2 != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Fprintf(writer, "# Translation Session Report\n\n")
	fmt.Fprintf(writer, "**Session ID:** %s\n", session.ID)
	fmt.Fprintf(writer, "**Start Time:** %s\n", session.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "**End Time:** %s\n", session.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "**Duration:** %s\n", session.EndTime.Sub(session.StartTime).String())
	fmt.Fprintf(writer, "**Provider:** %s\n", session.Config.Provider)
	fmt.Fprintf(writer, "**Input:** %s\n", session.Config.InputFile)
	fmt.Fprintf(writer, "**Output:** %s\n\n", session.Config.OutputFile)

	if err != nil {
		fmt.Fprintf(writer, "## Error\n\n%s\n\n", err.Error())
	} else {
		fmt.Fprintf(writer, "## Status\n\n✅ Translation completed successfully\n\n")
	}

	fmt.Fprintf(writer, "## Steps\n\n")
	for i, step := range session.Steps {
		status := "✅ Success"
		if !step.Success {
			status = "❌ Failed"
		}
		fmt.Fprintf(writer, "### Step %d: %s %s\n", i+1, step.Name, status)
		fmt.Fprintf(writer, "- **Duration:** %s\n", step.EndTime.Sub(step.StartTime).String())
		if step.Details != "" {
			fmt.Fprintf(writer, "- **Details:** %s\n", step.Details)
		}
		if step.Error != "" {
			fmt.Fprintf(writer, "- **Error:** %s\n", step.Error)
		}
		fmt.Fprintf(writer, "\n")
	}

	fmt.Fprintf(writer, "## Generated Files\n\n")
	for _, file := range session.Files {
		status := "✅ Verified"
		if !file.Verified {
			status = "⚠️ Issue"
		}
		fmt.Fprintf(writer, "### %s %s\n", filepath.Base(file.Path), status)
		fmt.Fprintf(writer, "- **Path:** %s\n", file.Path)
		fmt.Fprintf(writer, "- **Type:** %s\n", file.Type)
		fmt.Fprintf(writer, "- **Size:** %d bytes\n", file.Size)
		fmt.Fprintf(writer, "- **Verification:** %s\n\n", file.Verification)
	}

	fmt.Printf("Session report generated: %s\n", reportPath)
}

// validateWithVerifier validates the selected provider and model against LLMsVerifier.
func validateWithVerifier(cfg *UnifiedConfig) error {
	vCfg := &verifier.Config{
		APIURL:            cfg.VerifierURL,
		APIKey:            cfg.VerifierAPIKey,
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0.0,
	}
	client := verifier.NewClient(vCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("LLMsVerifier unreachable: %w", err)
	}

	model, err := client.GetModel(ctx, cfg.Model)
	if err != nil {
		return fmt.Errorf("model %s not verified: %w", cfg.Model, err)
	}

	if model.VerificationStatus != "verified" {
		return fmt.Errorf("model %s has status %s, expected verified", cfg.Model, model.VerificationStatus)
	}
	if !model.CanSeeCode {
		return fmt.Errorf("model %s failed code visibility check", cfg.Model)
	}
	if !model.AffirmativeResponse {
		return fmt.Errorf("model %s has no affirmative response", cfg.Model)
	}

	fmt.Printf("Model %s verified by LLMsVerifier (score: %.2f)\n", cfg.Model, model.OverallScore)
	return nil
}

// initVerifierConfig loads LLMsVerifier configuration from global config if available.
func initVerifierConfig(uc *UnifiedConfig) *verifier.Config {
	return &verifier.Config{
		APIURL:            uc.VerifierURL,
		APIKey:            uc.VerifierAPIKey,
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0.0,
	}
}
