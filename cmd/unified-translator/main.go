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
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

const (
	appVersion = "3.0.0"
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
	Provider string // openai, anthropic, zhipu, deepseek, qwen, gemini, ollama, llamacpp, ssh

	// API/Local LLM Configuration
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration

	// SSH Worker Configuration (for provider=ssh)
	SSHHost     string
	SSHUser     string
	SSHPassword string
	SSHPort     int
	RemoteDir   string

	// Local Llama.cpp Configuration (for provider=llamacpp)
	LlamaBinary string
	LlamaModel  string
	ContextSize int

	// Execution Options
	Workers      int
	ChunkSize    int
	Concurrency  int
	VerifyOutput bool
	Verbose      bool

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

	// Normalize to the requested target script (W10). The LLM output may be in
	// the wrong script (e.g. Cyrillic when -script latin was requested); this
	// deterministically converts it so the saved markdown, the EPUB, and the
	// verification all see the script the user asked for.
	translatedMarkdown = normalizeScript(translatedMarkdown, config.Script)

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
		return executeSSHTranslation(ctx, config, session, text)
	case "llamacpp":
		return executeLlamaCppTranslation(ctx, config, session, text)
	default:
		return executeAPITranslation(ctx, config, session, text)
	}
}

// executeSSHTranslation uses SSH worker for translation
func executeSSHTranslation(ctx context.Context, config *UnifiedConfig, session *TranslationSession, text string) (string, error) {
	session.Logger.Info("Starting SSH translation", map[string]interface{}{
		"host": config.SSHHost,
		"user": config.SSHUser,
	})

	// Initialize SSH worker
	workerConfig := sshworker.SSHWorkerConfig{
		Host:              config.SSHHost,
		Port:              config.SSHPort,
		Username:          config.SSHUser,
		Password:          config.SSHPassword,
		RemoteDir:         config.RemoteDir,
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    30 * time.Minute,
	}

	worker, err := sshworker.NewSSHWorker(workerConfig, session.Logger)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH worker: %w", err)
	}
	defer worker.Close()

	if err := worker.Connect(ctx); err != nil {
		return "", fmt.Errorf("failed to connect to SSH worker: %w", err)
	}

	// Upload text to remote
	remoteTextPath := filepath.Join(config.RemoteDir, "input.md")
	if err := worker.UploadData(ctx, []byte(text), remoteTextPath); err != nil {
		return "", fmt.Errorf("failed to upload text to remote: %w", err)
	}

	// Execute translation using remote llama.cpp
	remoteOutputPath := filepath.Join(config.RemoteDir, "output.md")
	cmd := fmt.Sprintf("cd %s && /home/milosvasic/llama.cpp -m /home/milosvasic/models/tiny-llama-working.gguf -p 'Translate from Russian to Serbian Cyrillic: ' -f %s > %s",
		config.RemoteDir, remoteTextPath, remoteOutputPath)

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute remote translation: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("remote translation failed: %s", result.Stderr)
	}

	// Download result
	tempFile := filepath.Join(os.TempDir(), "translation_result_"+session.ID+".txt")
	err = worker.DownloadFile(ctx, remoteOutputPath, tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to download translation result: %w", err)
	}

	// Read the downloaded file
	translatedData, err := os.ReadFile(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to read downloaded translation result: %w", err)
	}

	// Clean up temp file
	os.Remove(tempFile)

	return string(translatedData), nil
}

// executeLlamaCppTranslation uses local llama.cpp for translation
func executeLlamaCppTranslation(ctx context.Context, config *UnifiedConfig, session *TranslationSession, text string) (string, error) {
	session.Logger.Info("Starting local llama.cpp translation", map[string]interface{}{
		"binary": config.LlamaBinary,
		"model":  config.LlamaModel,
	})

	// Create LLM translator
	llmConfig := translator.TranslationConfig{
		SourceLang:  config.SourceLang,
		TargetLang:  config.TargetLang,
		Provider:    "llamacpp",
		Model:       config.Model,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		Timeout:     config.Timeout,
		Options: map[string]interface{}{
			"binary_path":  config.LlamaBinary,
			"model_path":   config.LlamaModel,
			"context_size": config.ContextSize,
		},
	}

	llmTranslator, err := llm.NewLLMTranslator(llmConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create LLM translator: %w", err)
	}

	// Translate
	result, err := llmTranslator.TranslateWithProgress(ctx, text, "Ebook content", session.EventBus, session.ID)
	if err != nil {
		return "", fmt.Errorf("LLM translation failed: %w", err)
	}

	return result, nil
}

// executeAPITranslation uses API-based LLM providers
func executeAPITranslation(ctx context.Context, config *UnifiedConfig, session *TranslationSession, text string) (string, error) {
	session.Logger.Info("Starting API-based translation", map[string]interface{}{
		"provider": config.Provider,
		"model":    config.Model,
	})

	var llmTranslator *llm.LLMTranslator
	var err error

	if config.VerifierEnabled {
		// CONST-034: Use VerifiedFactory as single source of truth
		llmTranslator, err = executeVerifiedTranslation(ctx, config, session)
		if err != nil {
			return "", fmt.Errorf("verified translation failed: %w", err)
		}
	} else {
		// Legacy direct provider path
		llmConfig := translator.TranslationConfig{
			SourceLang:  config.SourceLang,
			TargetLang:  config.TargetLang,
			Provider:    config.Provider,
			Model:       config.Model,
			Temperature: config.Temperature,
			MaxTokens:   config.MaxTokens,
			Timeout:     config.Timeout,
			APIKey:      config.APIKey,
			BaseURL:     config.BaseURL,
		}

		llmTranslator, err = llm.NewLLMTranslator(llmConfig)
		if err != nil {
			return "", fmt.Errorf("failed to create LLM translator: %w", err)
		}
	}

	// Translate
	result, err := llmTranslator.TranslateWithProgress(ctx, text, "Ebook content", session.EventBus, session.ID)
	if err != nil {
		return "", fmt.Errorf("API translation failed: %w", err)
	}

	return result, nil
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
		SSHPort:        22,
		RemoteDir:      "/tmp/translator",
		Workers:        1,
		ChunkSize:      2000,
		Concurrency:    4,
		VerifyOutput:   true,
		MonitoringPort: 8080,
		ContextSize:    2048,
	}

	flag.StringVar(&config.InputFile, "input", "", "Input ebook file")
	flag.StringVar(&config.InputFile, "i", "", "Input ebook file (shorthand)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file (auto-detected if not specified)")
	flag.StringVar(&config.OutputFile, "o", "", "Output file (shorthand)")

	flag.StringVar(&config.SourceLang, "source-lang", "ru", "Source language (default: ru)")
	flag.StringVar(&config.TargetLang, "target-lang", "sr", "Target language (default: sr)")
	flag.StringVar(&config.Script, "script", "cyrillic", "Target script: cyrillic, latin (default: cyrillic)")

	flag.StringVar(&config.Provider, "provider", "openai", "Translation provider: openai, anthropic, zhipu, deepseek, qwen, gemini, ollama, llamacpp, ssh")
	flag.StringVar(&config.Model, "model", "gpt-4", "Model name")
	flag.StringVar(&config.APIKey, "api-key", "", "API key for provider")
	flag.StringVar(&config.BaseURL, "base-url", "", "Base URL for provider (if needed)")
	flag.Float64Var(&config.Temperature, "temperature", 0.3, "LLM temperature")
	flag.IntVar(&config.MaxTokens, "max-tokens", 4096, "Maximum tokens")
	flag.DurationVar(&config.Timeout, "timeout", 30*time.Second, "Request timeout")

	// SSH options
	flag.StringVar(&config.SSHHost, "ssh-host", "", "SSH host (for provider=ssh)")
	flag.StringVar(&config.SSHUser, "ssh-user", "", "SSH username (for provider=ssh)")
	flag.StringVar(&config.SSHPassword, "ssh-password", "", "SSH password (for provider=ssh)")
	flag.IntVar(&config.SSHPort, "ssh-port", 22, "SSH port (default: 22)")
	flag.StringVar(&config.RemoteDir, "remote-dir", "/tmp/translator", "Remote directory (default: /tmp/translator)")

	// Llama.cpp options
	flag.StringVar(&config.LlamaBinary, "llama-binary", "/usr/local/bin/llama.cpp", "Path to llama.cpp binary")
	flag.StringVar(&config.LlamaModel, "llama-model", "", "Path to llama.cpp model")
	flag.IntVar(&config.ContextSize, "context-size", 2048, "LLM context size")

	// Execution options
	flag.IntVar(&config.Workers, "workers", 1, "Number of parallel workers")
	flag.IntVar(&config.ChunkSize, "chunk-size", 2000, "Text chunk size")
	flag.IntVar(&config.Concurrency, "concurrency", 4, "Maximum concurrent operations")
	flag.BoolVar(&config.VerifyOutput, "verify", true, "Verify translated output")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

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
		config.OutputFile = generateOutputFilename(config.InputFile)
	}

	// Provider-specific validation
	switch config.Provider {
	case "ssh":
		if config.SSHHost == "" || config.SSHUser == "" || config.SSHPassword == "" {
			fmt.Fprintf(os.Stderr, "Error: SSH host, user, and password required for provider=ssh\n")
			os.Exit(1)
		}
	case "llamacpp":
		if config.LlamaModel == "" {
			fmt.Fprintf(os.Stderr, "Error: llama-model path required for provider=llamacpp\n")
			os.Exit(1)
		}
	case "mock", "ollama":
		// No API key required for mock or local Ollama
	default:
		// Fall back to the provider's well-known env var (e.g. DEEPSEEK_API_KEY)
		// when -api-key was not passed. The gate previously checked ONLY the flag,
		// so env-var keys — which resolveProviderAPIKey already supports and the
		// docs advertise — never satisfied it. Populate config.APIKey so the whole
		// pipeline (provider client construction) uses the resolved key too.
		if config.APIKey == "" {
			config.APIKey = resolveProviderAPIKey(config, config.Provider)
		}
		if config.APIKey == "" {
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
  ollama      - Local Ollama models
  llamacpp    - Local llama.cpp models
  ssh         - Remote SSH worker with llama.cpp

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

SSH Configuration (provider=ssh):
  -ssh-host <host>          SSH host
  -ssh-user <user>          SSH username
  -ssh-password <pass>      SSH password
  -ssh-port <port>          SSH port (default: 22)
  -remote-dir <dir>         Remote directory (default: /tmp/translator)

Llama.cpp Configuration (provider=llamacpp):
  -llama-binary <path>      Path to llama.cpp binary
  -llama-model <path>       Path to llama.cpp model
  -context-size <size>      LLM context size (default: 2048)

Execution Options:
  -workers <num>            Parallel workers (default: 1)
  -chunk-size <size>         Text chunk size (default: 2000)
  -concurrency <num>         Concurrent operations (default: 4)
  -verify                   Verify output (default: true)
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

  # Translate with local llama.cpp
  unified-translator -i book.fb2 -provider llamacpp -llama-model ./model.gguf

  # Translate via SSH worker
  unified-translator -i book.fb2 -provider ssh -ssh-host worker.local -ssh-user user -ssh-password pass

  # Translate with monitoring
  unified-translator -i book.fb2 -provider openai -api-key YOUR_KEY -monitoring

Translation Flow:
  1. Parse input ebook (FB2, EPUB, PDF, etc.)
  2. Convert to markdown format
  3. Translate using selected provider
  4. Convert translated markdown to EPUB
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

func generateOutputFilename(inputFile string) string {
	base := filepath.Base(inputFile)
	// Trim the actual extension (case-preserving) so UPPERCASE/mixed-case
	// extensions (e.g. "Story.EPUB", "Tale.Fb2") are stripped just like
	// lowercase ones. Lowercasing the ext before TrimSuffix would not match
	// the original-case basename and leave the extension embedded in the name.
	baseName := strings.TrimSuffix(base, filepath.Ext(base))

	return filepath.Join(filepath.Dir(inputFile), baseName+"_sr.epub")
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
// Supported: .epub (default / no extension), .fb2, .txt, .md.
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
	default:
		return fmt.Errorf("unsupported output format %q (supported: .epub, .fb2, .html, .txt, .md)", ext)
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
