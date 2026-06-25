package llm

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// TranslationConfig holds translation configuration (use from parent package to avoid import cycle)
// Alias to parent package's TranslationConfig
type TranslationConfig = translator.TranslationConfig

// ConvertFromTranslatorConfig converts from parent package config to local config
func ConvertFromTranslatorConfig(config translator.TranslationConfig) TranslationConfig {
	return config
}

// Provider represents LLM provider types
type Provider string

const (
	ProviderOpenAI      Provider = "openai"
	ProviderAnthropic   Provider = "anthropic"
	ProviderZhipu       Provider = "zhipu"
	ProviderDeepSeek    Provider = "deepseek"
	ProviderQwen        Provider = "qwen"
	ProviderGemini      Provider = "gemini"
	ProviderMock        Provider = "mock"
	ProviderGroq        Provider = "groq"
	ProviderCohere      Provider = "cohere"
	ProviderMistral     Provider = "mistral"
	ProviderXAI         Provider = "xai"
	ProviderReplicate   Provider = "replicate"
	ProviderCerebras    Provider = "cerebras"
	ProviderCloudflare  Provider = "cloudflare"
	ProviderSiliconFlow Provider = "siliconflow"
	ProviderHyperbolic  Provider = "hyperbolic"
	ProviderTogetherAI  Provider = "togetherai"
	ProviderSambaNova   Provider = "sambanova"
	ProviderKimi        Provider = "kimi"
	ProviderNovita      Provider = "novita"
	ProviderNLPCloud    Provider = "nlpcloud"
	ProviderUpstage     Provider = "upstage"
	ProviderSarvam      Provider = "sarvam"
	ProviderModal       Provider = "modal"
	ProviderPublicAI    Provider = "publicai"
	ProviderNIA         Provider = "nia"
	ProviderVulavula    Provider = "vulavula"
)

// ValidModels defines valid model names for each provider
var ValidModels = map[Provider][]string{
	ProviderOpenAI:    {"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo", "gpt-4o"},
	ProviderAnthropic: {"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
	ProviderZhipu:     {"glm-4", "glm-3-turbo", "glm-4-plus", "glm-4-flash", "glm-4-air", "glm-4-airx", "glm-4-long", "glm-4-flashx"},
	// deepseek-v4-flash / deepseek-v4-pro: current models per the live DeepSeek
	// /models endpoint (verified 2026-06-14); v4-flash proven to translate. The
	// legacy deepseek-chat/coder remain accepted (still work — §11.4.122).
	ProviderDeepSeek:    {"deepseek-chat", "deepseek-coder", "deepseek-v4-flash", "deepseek-v4-pro"},
	ProviderQwen:        {"qwen-max", "qwen-plus", "qwen-turbo"},
	ProviderGemini:      {"gemini-pro", "gemini-pro-vision", "gemini-1.5-flash", "gemini-1.5-pro", "gemini-2.0-flash", "gemini-2.5-flash", "gemini-2.5-pro"},
	ProviderMock:        {"mock"},
	ProviderGroq:        {"llama-3.1-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "llama-3.3-70b-versatile", "gemma2-9b-it"},
	ProviderCohere:      {"command-r", "command-r-plus", "command"},
	ProviderMistral:     {"mistral-large-latest", "mistral-medium-latest", "mistral-small-latest"},
	ProviderXAI:         {"grok-beta", "grok-vision-beta"},
	ProviderReplicate:   {"meta/llama-2-70b-chat", "mistralai/mixtral-8x7b-instruct-v0.1"},
	ProviderCerebras:    {"llama3.1-70b", "llama3.1-8b"},
	ProviderCloudflare:  {"@cf/meta/llama-2-7b-chat-int8", "@cf/mistral/mistral-7b-instruct-v0.1"},
	ProviderSiliconFlow: {"deepseek-ai/DeepSeek-V2.5", "Qwen/Qwen2.5-72B-Instruct"},
	ProviderHyperbolic:  {"meta-llama/Meta-Llama-3.1-70B-Instruct", "meta-llama/Meta-Llama-3.1-8B-Instruct"},
	ProviderTogetherAI:  {"meta-llama/Llama-3.1-70B-Instruct-Turbo", "mistralai/Mixtral-8x7B-Instruct-v0.1"},
	ProviderSambaNova:   {"Meta-Llama-3.1-70B-Instruct", "Meta-Llama-3.1-8B-Instruct"},
	ProviderKimi:        {"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
	ProviderNovita:      {"meta-llama/llama-3.1-70b-instruct", "mistralai/mixtral-8x7b-instruct"},
	ProviderNLPCloud:    {"finetuned-llama-3-1-8b", "finetuned-llama-3-1-70b"},
	ProviderUpstage:     {"solar-pro", "solar-mini"},
	ProviderSarvam:      {"sarvam-1", "sarvam-2"},
	ProviderModal:       {"modal-llama-3-1-70b"},
	ProviderPublicAI:    {"publicai-llama-3-1-70b"},
	ProviderNIA:         {"nia-llama-3-1-70b"},
	ProviderVulavula:    {"vulavula-llama-3-1-70b"},
}

// IsKnownProvider reports whether name is a recognised LLM provider — i.e. one
// the NewLLMTranslatorWithConfig factory switch can build a client for. The set
// is derived from the ValidModels registry (every supported provider has a
// ValidModels entry), so this stays in lockstep with the factory without a
// second hand-maintained list. An explicitly client-supplied provider name that
// is NOT known is a client error: the caller asked for a provider the system
// does not offer, which MUST be rejected (HTTP 400) rather than silently routed
// to a different provider — a response-correctness defect. The empty string is
// NOT a known provider (it means "use the default", handled by the caller).
func IsKnownProvider(name string) bool {
	if name == "" {
		return false
	}
	_, ok := ValidModels[Provider(name)]
	return ok
}

// LLMTranslator implements LLM-based translation
type LLMTranslator struct {
	*BaseTranslator
	provider Provider
	client   LLMClient
}

// GetStats returns translation statistics (implements translator.Translator interface)
func (lt *LLMTranslator) GetStats() translator.TranslationStats {
	stats := lt.BaseTranslator.GetStats()
	return translator.TranslationStats{
		Total:      stats.Total,
		Translated: stats.Translated,
		Cached:     stats.Cached,
		Errors:     stats.Errors,
	}
}

// BaseTranslator provides common functionality (local copy to avoid import cycle)
type BaseTranslator struct {
	config TranslationConfig
	mu     sync.RWMutex // guards cache and stats for concurrent Translate callers
	stats  TranslationStats
	cache  map[string]string
}

// Config returns the translator's configured settings, including the resolved
// SourceLang / TargetLang language pair. Exposing it lets callers verify which
// language pair a constructed translator will actually translate into — the
// contract that the REST request's target_lang must reach (it previously could
// not, because the API hardcoded Russian→Serbian).
func (bt *BaseTranslator) Config() TranslationConfig {
	return bt.config
}

// TranslationStats tracks translation statistics (local copy to avoid import cycle)
type TranslationStats struct {
	Total      int
	Translated int
	Cached     int
	Errors     int
}

// NewBaseTranslator creates a new base translator
func NewBaseTranslator(config TranslationConfig) *BaseTranslator {
	return &BaseTranslator{
		config: config,
		stats:  TranslationStats{},
		cache:  make(map[string]string),
	}
}

// GetStats returns translation statistics
func (bt *BaseTranslator) GetStats() TranslationStats {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.stats
}

// CheckCache checks if translation is cached
func (bt *BaseTranslator) CheckCache(text string) (string, bool) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	if translated, ok := bt.cache[text]; ok {
		bt.stats.Cached++
		return translated, true
	}
	return "", false
}

// AddToCache adds a translation to cache
func (bt *BaseTranslator) AddToCache(original, translated string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.cache[original] = translated
}

// UpdateStats updates translation statistics
func (bt *BaseTranslator) UpdateStats(success bool) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.stats.Total++
	if success {
		bt.stats.Translated++
	} else {
		bt.stats.Errors++
	}
}

// EmitProgress emits a progress event
func EmitProgress(eventBus *events.EventBus, sessionID, message string, data map[string]interface{}) {
	if eventBus == nil {
		return
	}

	event := events.NewEvent(events.EventTranslationProgress, message, data)
	event.SessionID = sessionID
	eventBus.Publish(event)
}

// EmitError emits an error event
func EmitError(eventBus *events.EventBus, sessionID, message string, err error) {
	if eventBus == nil {
		return
	}

	data := map[string]interface{}{
		"error": err.Error(),
	}

	event := events.NewEvent(events.EventTranslationError, message, data)
	event.SessionID = sessionID
	eventBus.Publish(event)
}

// LLMClient interface for different LLM providers
type LLMClient interface {
	Translate(ctx context.Context, text string, prompt string) (string, error)
	GetProviderName() string
}

// NewLLMTranslator creates a new LLM translator
func NewLLMTranslator(config translator.TranslationConfig) (*LLMTranslator, error) {
	return NewLLMTranslatorWithConfig(ConvertFromTranslatorConfig(config))
}

// NewLLMTranslatorWithConfig creates a new LLM translator with local config
func NewLLMTranslatorWithConfig(config TranslationConfig) (*LLMTranslator, error) {
	provider := Provider(config.Provider)

	// Validate provider
	if provider == "" {
		return nil, fmt.Errorf("provider must be specified")
	}

	// Validate model if provided
	if config.Model != "" {
		if validModels, exists := ValidModels[provider]; exists {
			modelValid := false
			for _, validModel := range validModels {
				if config.Model == validModel {
					modelValid = true
					break
				}
			}
			if !modelValid {
				return nil, fmt.Errorf("model '%s' is not valid for provider '%s'. Valid models: %v",
					config.Model, provider, validModels)
			}
		}
	}

	client, err := newClientForProvider(provider, config)
	if err != nil {
		return nil, err
	}

	return &LLMTranslator{
		BaseTranslator: NewBaseTranslator(config),
		provider:       provider,
		client:         client,
	}, nil
}

// NewLLMTranslatorWithClient wraps an already-constructed LLMClient in an
// LLMTranslator carrying the given config, so the translator's prompt builder
// honors the configured SourceLang / TargetLang / Script. It is the seam the
// bridge's explicit-provider path uses: build a provider client (possibly the
// whitelist-immune generic OpenAI-compatible client for a verified model), then
// wrap it here so the REAL translation prompt — not the raw input text — reaches
// the model. The reported provider name comes from config.Provider.
func NewLLMTranslatorWithClient(config translator.TranslationConfig, client LLMClient) *LLMTranslator {
	local := ConvertFromTranslatorConfig(config)
	return &LLMTranslator{
		BaseTranslator: NewBaseTranslator(local),
		provider:       Provider(local.Provider),
		client:         client,
	}
}

// NewClientForProvider builds the raw per-provider LLMClient for an EXPLICIT
// provider selection, honoring config.Model / config.APIKey / config.BaseURL.
// It is the client-only counterpart of NewLLMTranslatorWithConfig (which wraps the
// same client in an LLMTranslator), exported so the bridge's explicit-provider
// path (`-provider X`) can build a native-wire client (e.g. anthropic) without a
// translator wrapper. Unsupported providers yield an honest error — never a silent
// fallback to a different provider (§11.4.69). No API key is logged (§11.4.10).
func NewClientForProvider(config translator.TranslationConfig) (LLMClient, error) {
	local := ConvertFromTranslatorConfig(config)
	provider := Provider(local.Provider)
	if provider == "" {
		return nil, fmt.Errorf("provider must be specified")
	}
	return newClientForProvider(provider, local)
}

// newClientForProvider is the single per-provider client construction switch,
// shared by NewLLMTranslatorWithConfig and NewClientForProvider so the provider
// set never drifts between the translator and client construction paths.
func newClientForProvider(provider Provider, config TranslationConfig) (LLMClient, error) {
	var client LLMClient
	var err error

	switch provider {
	case ProviderOpenAI:
		client, err = NewOpenAIClient(config)
	case ProviderAnthropic:
		client, err = NewAnthropicClient(config)
	case ProviderZhipu:
		client, err = NewZhipuClient(config)
	case ProviderDeepSeek:
		client, err = NewDeepSeekClient(config)
	case ProviderQwen:
		client, err = NewQwenClient(config)
	case ProviderGemini:
		client, err = NewGeminiClient(config)
	case ProviderGroq:
		client, err = NewGroqClient(config)
	case ProviderCohere:
		client, err = NewCohereClient(config)
	case ProviderMistral:
		client, err = NewMistralClient(config)
	case ProviderXAI:
		client, err = NewXAIClient(config)
	case ProviderReplicate:
		client, err = NewReplicateClient(config)
	case ProviderCerebras:
		client, err = NewCerebrasClient(config)
	case ProviderCloudflare:
		client, err = NewCloudflareClient(config)
	case ProviderSiliconFlow:
		client, err = NewSiliconFlowClient(config)
	case ProviderHyperbolic:
		client, err = NewHyperbolicClient(config)
	case ProviderTogetherAI:
		client, err = NewTogetherAIClient(config)
	case ProviderSambaNova:
		client, err = NewSambaNovaClient(config)
	case ProviderKimi:
		client, err = NewKimiClient(config)
	case ProviderNovita:
		client, err = NewNovitaClient(config)
	case ProviderNLPCloud:
		client, err = NewNLPCloudClient(config)
	case ProviderUpstage:
		client, err = NewUpstageClient(config)
	case ProviderSarvam:
		client, err = NewSarvamClient(config)
	case ProviderModal:
		client, err = NewModalClient(config)
	case ProviderPublicAI:
		client, err = NewPublicAIClient(config)
	case ProviderNIA:
		client, err = NewNIAClient(config)
	case ProviderVulavula:
		client, err = NewVulavulaClient(config)
	case ProviderMock:
		client = NewMockLLMClient()
		err = nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}
	return client, nil
}

// GetName returns the translator name
func (lt *LLMTranslator) GetName() string {
	return fmt.Sprintf("llm-%s", lt.provider)
}

// makeLLMCacheKey builds an INJECTIVE in-memory cache key from (text,
// contextStr). A plain "text:contextStr" join is NOT injective — ":" can appear
// inside the text, so ("a:b","c") and ("a","b:c") would both produce "a:b:c"
// and the cache would serve the wrong translation (same collision class as the
// storage Redis cache-key bug fixed earlier). Length-prefixing the text makes
// the key unambiguous: the leading "<len>:" tells a parser exactly where text
// ends, so no (text, contextStr) pair can alias another. This keeps the same
// memory profile as the previous full-text key (no hashing cost).
func makeLLMCacheKey(text, contextStr string) string {
	return fmt.Sprintf("%d:%s:%s", len(text), text, contextStr)
}

// Translate translates text using LLM with automatic retry and text splitting
func (lt *LLMTranslator) Translate(ctx context.Context, text string, contextStr string) (string, error) {
	if text == "" || strings.TrimSpace(text) == "" {
		return text, nil
	}

	// Check cache
	cacheKey := makeLLMCacheKey(text, contextStr)
	if cached, found := lt.CheckCache(cacheKey); found {
		return cached, nil
	}

	// Create translation prompt
	prompt := lt.createTranslationPrompt(text, contextStr)

	// Translate using LLM with smart retry
	result, err := lt.translateWithRetry(ctx, text, prompt, contextStr)
	if err != nil {
		lt.UpdateStats(false)
		return "", fmt.Errorf("LLM translation failed: %w", err)
	}

	// Enhance translation
	result = lt.enhanceTranslation(text, result)

	// Update stats
	lt.UpdateStats(true)

	// Cache result
	lt.AddToCache(cacheKey, result)

	return result, nil
}

// translateWithRetry attempts translation with automatic splitting on size errors
func (lt *LLMTranslator) translateWithRetry(ctx context.Context, text, prompt, contextStr string) (string, error) {
	// First attempt - try with full text
	result, err := lt.client.Translate(ctx, text, prompt)
	if err == nil {
		return result, nil
	}

	// Check if error is due to text size
	if !isTextSizeError(err) {
		return "", err
	}

	// Text is too large - split and translate in chunks
	fmt.Fprintf(os.Stderr, "[LLM_RETRY] Text too large (%d bytes), splitting into chunks\n", len(text))

	chunks := lt.splitText(text)
	if len(chunks) == 1 {
		// Cannot split further - text is too large even for one sentence
		return "", fmt.Errorf("text too large to translate even after splitting (min chunk: %d bytes): %w", len(chunks[0]), err)
	}

	fmt.Fprintf(os.Stderr, "[LLM_RETRY] Split into %d chunks, translating separately\n", len(chunks))

	// Translate each chunk
	var translatedChunks []string
	for i, chunk := range chunks {
		chunkPrompt := lt.createTranslationPrompt(chunk, fmt.Sprintf("%s (part %d/%d)", contextStr, i+1, len(chunks)))

		chunkResult, chunkErr := lt.client.Translate(ctx, chunk, chunkPrompt)
		if chunkErr != nil {
			return "", fmt.Errorf("failed to translate chunk %d/%d: %w", i+1, len(chunks), chunkErr)
		}

		translatedChunks = append(translatedChunks, chunkResult)
	}

	// Combine translated chunks
	result = strings.Join(translatedChunks, "")
	fmt.Fprintf(os.Stderr, "[LLM_RETRY] Successfully translated %d chunks\n", len(chunks))

	return result, nil
}

// isTextSizeError checks if error is due to text being too large
func isTextSizeError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Common size-related error patterns
	sizeErrorPatterns := []string{
		"max_tokens",
		"token limit",
		"too large",
		"too long",
		"maximum length",
		"context length",
		"exceeds",
		"invalid request",
	}

	for _, pattern := range sizeErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// splitText splits text into smaller chunks at sentence boundaries
func (lt *LLMTranslator) splitText(text string) []string {
	// Target chunk size (roughly 20KB to stay well under limits)
	const maxChunkSize = 20000

	// If text is small enough, return as-is
	if len(text) <= maxChunkSize {
		return []string{text}
	}

	var chunks []string
	var currentChunk strings.Builder

	// Split by paragraphs, but re-attach the "\n\n" delimiter to every paragraph
	// except the last so the units losslessly tile the input: concatenating all
	// units reproduces the original text exactly. This is the key invariant —
	// strings.Join(splitText(text), "") == text — which makes the per-chunk
	// translation reassembly (also a Join with "") preserve paragraph boundaries
	// instead of gluing the last paragraph of one chunk to the first of the next.
	// The previous implementation stripped "\n\n" via Split and re-added it
	// inconsistently (never around an oversized paragraph, and never across a
	// chunk boundary), dropping a separator at every chunk seam — structural
	// data loss in translated large chapters.
	paragraphs := strings.Split(text, "\n\n")

	for i, para := range paragraphs {
		unit := para
		if i < len(paragraphs)-1 {
			unit += "\n\n"
		}
		if unit == "" {
			// Only the final unit can be empty (text ended with a delimiter);
			// it carries no content, so skipping it keeps the tiling lossless.
			continue
		}

		// If a single unit is too large on its own, split it into sentences
		// (splitBySentences is lossless — its output concatenates back to the
		// unit, including the trailing "\n\n").
		if len(unit) > maxChunkSize {
			sentences := lt.splitBySentences(unit)
			for _, sentence := range sentences {
				if currentChunk.Len()+len(sentence) > maxChunkSize && currentChunk.Len() > 0 {
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
				}
				currentChunk.WriteString(sentence)
			}
			continue
		}

		// The unit already includes its "\n\n" delimiter, so no extra separator
		// is added here (that would double the breaks and break the round-trip).
		if currentChunk.Len()+len(unit) > maxChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}
		currentChunk.WriteString(unit)
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// splitBySentences splits text into sentences
func (lt *LLMTranslator) splitBySentences(text string) []string {
	var sentences []string
	var currentSentence strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		currentSentence.WriteRune(runes[i])

		// Check for sentence endings
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' || runes[i] == '…' {
			// Check if followed by space or end of text
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' {
				sentences = append(sentences, currentSentence.String())
				currentSentence.Reset()
			}
		}
	}

	// Add remaining text
	if currentSentence.Len() > 0 {
		sentences = append(sentences, currentSentence.String())
	}

	return sentences
}

// TranslateWithProgress translates and reports progress (implements translator.Translator interface)
func (lt *LLMTranslator) TranslateWithProgress(
	ctx context.Context,
	text string,
	contextStr string,
	eventBus *events.EventBus,
	sessionID string,
) (string, error) {
	translator.EmitProgress(eventBus, sessionID, "Starting LLM translation", map[string]interface{}{
		"provider":    string(lt.provider),
		"text_length": len(text),
	})

	result, err := lt.Translate(ctx, text, contextStr)

	if err != nil {
		// Log detailed error to stdout for debugging
		fmt.Fprintf(os.Stderr, "[LLM_ERROR] Translation failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "[LLM_ERROR] Text length: %d bytes, Context: %s\n", len(text), contextStr)
		translator.EmitError(eventBus, sessionID, "LLM translation failed", err)
		return "", err
	}

	translator.EmitProgress(eventBus, sessionID, "LLM translation completed", map[string]interface{}{
		"provider":          string(lt.provider),
		"original_length":   len(text),
		"translated_length": len(result),
	})

	return result, nil
}

// languageName maps an ISO-639-1 / short language code (or already-spelled
// language name) to a human-readable English language name used in the
// translation prompt. Unknown / empty codes return the empty string so the
// caller can decide on a sensible fallback (never emit a broken prompt).
func languageName(code string) string {
	c := strings.ToLower(strings.TrimSpace(code))
	if c == "" {
		return ""
	}
	names := map[string]string{
		"ru": "Russian", "russian": "Russian",
		"sr": "Serbian", "serbian": "Serbian",
		"en": "English", "english": "English",
		"fr": "French", "french": "French",
		"es": "Spanish", "spanish": "Spanish",
		"de": "German", "german": "German",
		"it": "Italian", "italian": "Italian",
		"pt": "Portuguese", "portuguese": "Portuguese",
		"nl": "Dutch", "dutch": "Dutch",
		"pl": "Polish", "polish": "Polish",
		"uk": "Ukrainian", "ukrainian": "Ukrainian",
		"bg": "Bulgarian", "bulgarian": "Bulgarian",
		"cs": "Czech", "czech": "Czech",
		"sk": "Slovak", "slovak": "Slovak",
		"sl": "Slovenian", "slovenian": "Slovenian",
		"hr": "Croatian", "croatian": "Croatian",
		"mk": "Macedonian", "macedonian": "Macedonian",
		"el": "Greek", "greek": "Greek",
		"tr": "Turkish", "turkish": "Turkish",
		"ro": "Romanian", "romanian": "Romanian",
		"hu": "Hungarian", "hungarian": "Hungarian",
		"sv": "Swedish", "swedish": "Swedish",
		"no": "Norwegian", "norwegian": "Norwegian",
		"da": "Danish", "danish": "Danish",
		"fi": "Finnish", "finnish": "Finnish",
		"ar": "Arabic", "arabic": "Arabic",
		"he": "Hebrew", "hebrew": "Hebrew",
		"zh": "Chinese", "chinese": "Chinese",
		"ja": "Japanese", "japanese": "Japanese",
		"ko": "Korean", "korean": "Korean",
		"hi": "Hindi", "hindi": "Hindi",
	}
	if name, ok := names[c]; ok {
		return name
	}
	return code // unknown but non-empty: pass through the operator-supplied label
}

// isRussianToSerbian reports whether the configured language pair is the
// project's primary Russian→Serbian path, which carries the rich Ekavica /
// Cyrillic literary guidance. An unset (empty) source AND target also resolve
// here so the historical default behaviour is preserved when no pair is
// configured (e.g. a zero-value LLMTranslator).
func isRussianToSerbian(sourceLang, targetLang string) bool {
	src := languageName(sourceLang)
	tgt := languageName(targetLang)
	if src == "" && tgt == "" {
		// No pair configured at all → preserve the legacy Russian→Serbian default.
		return true
	}
	// A PARTIALLY-configured pair must NOT inherit the missing side as Russian /
	// Serbian: when exactly one of source/target is set the empty side is unknown,
	// not "the default". Treating an empty source as Russian made an explicit
	// non-Russian-source → Serbian pair (e.g. SourceLang unset + TargetLang "sr",
	// common when the source language is auto-detected) emit the primary
	// "translate the following Russian text" Ekavica prompt — telling the LLM the
	// wrong source language. The RU→SR path requires BOTH sides explicit.
	return src == "Russian" && tgt == "Serbian"
}

// scriptInstruction returns a script-specific guideline line for the target
// language. Serbian Cyrillic↔Latin is fully supported via the configured
// Script value; for other targets the instruction is generic.
func scriptInstruction(targetName, script string) string {
	s := strings.ToLower(strings.TrimSpace(script))
	switch s {
	case "latin":
		return fmt.Sprintf("Write the %s translation using the Latin script.", targetName)
	case "cyrillic":
		return fmt.Sprintf("Write the %s translation using the Cyrillic script.", targetName)
	default:
		return fmt.Sprintf("Use the standard, natural writing system for %s.", targetName)
	}
}

// createTranslationPrompt creates the translation prompt honouring the
// configured SourceLang / TargetLang / Script. The primary Russian→Serbian
// path keeps its exact Ekavica + pure-Serbian-vocabulary + Cyrillic guidance;
// every other configured pair receives a correct generic professional-literary
// prompt for that pair and script.
// BuildContractPrompt exposes the translation prompt the translator will send to
// the LLM for the given text/context, built from the configured SourceLang /
// TargetLang / Script. It lets contract tests verify the resolved language pair
// reaches the prompt (e.g. that a Spanish target produces a Spanish prompt, not
// the hardcoded Serbian one) without performing a live LLM call.
func (lt *LLMTranslator) BuildContractPrompt(text string, contextStr string) string {
	return lt.createTranslationPrompt(text, contextStr)
}

func (lt *LLMTranslator) createTranslationPrompt(text string, contextStr string) string {
	context := contextStr
	if context == "" {
		context = "Literary text"
	}

	var sourceLang, targetLang, script string
	if lt.BaseTranslator != nil {
		sourceLang = lt.config.SourceLang
		targetLang = lt.config.TargetLang
		script = lt.config.Script
	}

	// Primary path — Russian → Serbian (also the no-config default). Preserve
	// the existing rich Ekavica guidance EXACTLY (§11.4.124 no-regression).
	if isRussianToSerbian(sourceLang, targetLang) {
		// Honour an explicit Latin-script override for the Serbian target while
		// keeping every other guideline identical to the historical prompt.
		scriptLine := "6. Use Serbian Cyrillic script (ћирилица)"
		if strings.EqualFold(strings.TrimSpace(script), "latin") {
			scriptLine = "6. Use Serbian Latin script (latinica)"
		}
		return fmt.Sprintf(`You are a professional literary translator specializing in Russian to Serbian translation.
Your task is to translate the following Russian text into natural, idiomatic Serbian.

Guidelines:
1. Preserve the literary style and tone
2. Use appropriate Serbian vocabulary and grammar
3. Maintain cultural nuances and idioms
4. Keep names of people and places unchanged unless they have standard Serbian equivalents
5. Preserve formatting, punctuation, and paragraph structure
%s
7. **CRITICAL**: Use ONLY Ekavica dialect (екавица) - the standard Serbian dialect used in Serbia
   - Use "е" instead of "ије/је": mleko (not mlijeko), dete (not dijete), pesma (not pjesma)
   - Ekavica examples: hteo (not htio), lepo (not lijepo), reka (not rijeka)
   - This is MANDATORY for all translations to Serbian
8. **CRITICAL**: Use ONLY pure Serbian vocabulary - avoid Croatian, Bosnian, or Montenegrin words
   - Use standard Serbian words preferred in Serbia, not regional variants
   - Example: use "avion" (not Croatian "zrakoplov"), "pozorište" (not Croatian "kazalište")
9. **CRITICAL**: Output ONLY the translated text itself. Do NOT add any notes,
   explanations, commentary, disclaimers, or remarks about the translation.

Context: %s

Russian text:
%s

Serbian translation (Ekavica only):`, scriptLine, context, text)
	}

	// Generic path — any other configured language pair.
	srcName := languageName(sourceLang)
	if srcName == "" {
		srcName = "the source language"
	}
	tgtName := languageName(targetLang)
	if tgtName == "" {
		tgtName = "the target language"
	}

	return fmt.Sprintf(`You are a professional literary translator specializing in %s to %s translation.
Your task is to translate the following %s text into natural, idiomatic %s.

Guidelines:
1. Preserve the literary style and tone
2. Use appropriate %s vocabulary and grammar
3. Maintain cultural nuances and idioms
4. Keep names of people and places unchanged unless they have standard %s equivalents
5. Preserve formatting, punctuation, and paragraph structure
6. %s
7. **CRITICAL**: Output ONLY the translated text itself. Do NOT add any notes,
   explanations, commentary, disclaimers, or remarks about the translation.

Context: %s

%s text:
%s

%s translation:`,
		srcName, tgtName,
		srcName, tgtName,
		tgtName,
		tgtName,
		scriptInstruction(tgtName, script),
		context,
		srcName,
		text,
		tgtName)
}

// commentaryLeadMarkers are case-insensitive English meta-commentary openers an
// instruct model appends AFTER the translation, separated by a blank line. They
// are deliberately specific (anti-bluff §11.4.6 — must not eat a genuine
// translated paragraph that merely begins with a common word). Captured from the
// live nezha llm-novita responses (qa-results/nezha_heavy_test_*/).
var commentaryLeadMarkers = []string{
	"this translation",
	"the translation",
	"note:",
	"translation note",
	"translator's note",
	"i've translated",
	"i have translated",
	"i translated",
	"in this translation",
	"please note",
}

// commentaryParenSignalWords are meta-commentary signal words the model uses
// INSIDE a trailing fully-enclosed parenthetical/bracketed aside to explain its
// translation choices (dialect, register, vocabulary, style, guidelines, etc.).
// A trailing ()/[] block containing any of these is a model aside, not real
// translated content. Kept narrow + English-meta-specific so a genuine
// in-content trailing parenthetical (real translated prose) is NOT stripped.
var commentaryParenSignalWords = []string{
	"using ",
	"dialect",
	"vocabulary",
	"guideline",
	"register",
	"as per",
	"as requested",
	"per the instruction",
	"i used",
	"i have used",
	"i chose",
	"i kept",
}

// stripTranslationCommentary removes a trailing model-commentary block. An
// instruct model that ignores the "translation only" instruction appends an
// explanation after a blank line, prefixed either by an enclosing marker
// ("(Note: ...)", "[Note: ...]") or by an English meta-phrase. Only the LAST
// blank-line-separated block(s) are eligible, and ONLY when they are clearly
// commentary — a genuine multi-paragraph translation has no such marker and is
// returned untouched.
func stripTranslationCommentary(s string) string {
	blocks := strings.Split(s, "\n\n")
	if len(blocks) < 2 {
		return s
	}
	for len(blocks) > 1 {
		last := strings.TrimSpace(blocks[len(blocks)-1])
		if last == "" {
			blocks = blocks[:len(blocks)-1]
			continue
		}
		if !isCommentaryBlock(last) {
			break
		}
		blocks = blocks[:len(blocks)-1]
	}
	return strings.TrimRight(strings.Join(blocks, "\n\n"), "\n")
}

// isCommentaryBlock reports whether a trailing block is model meta-commentary
// rather than translated content.
func isCommentaryBlock(block string) bool {
	t := strings.TrimSpace(block)
	if t == "" {
		return false
	}
	// Fully enclosed parenthetical / bracketed note, e.g. "(Note: ...)" /
	// "[Note: ...]" / "(Using Ekavica dialect and pure Serbian vocabulary as per
	// guidelines)". A trailing \n\n-separated block fully wrapped in ()/[] whose
	// inner text carries any meta-commentary signal word is the model explaining
	// its translation choices (dialect/register/vocabulary/style/guidelines), NOT
	// real translated content — strip it. The signal-word set is required (rather
	// than blanket-stripping every trailing parenthetical) so a genuine in-content
	// trailing parenthetical is preserved (see
	// TestEnhanceTranslation_StripCommentary_KeepsBenignTrailingParenthetical).
	if (strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")")) ||
		(strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]")) {
		inner := strings.ToLower(strings.TrimLeft(t, "(["))
		if strings.HasPrefix(inner, "note") || strings.Contains(inner, "translat") {
			return true
		}
		for _, w := range commentaryParenSignalWords {
			if strings.Contains(inner, w) {
				return true
			}
		}
	}
	lower := strings.ToLower(t)
	for _, m := range commentaryLeadMarkers {
		if strings.HasPrefix(lower, m) {
			return true
		}
	}
	return false
}

// enhanceTranslation post-processes the translation
func (lt *LLMTranslator) enhanceTranslation(original, translated string) string {
	// Strip any trailing model meta-commentary the LLM appended despite the
	// "translation only" prompt instruction (§11.4.135 regression guard:
	// TestEnhanceTranslation_StripsModelCommentary).
	enhanced := stripTranslationCommentary(translated)

	// Fix common punctuation issues
	enhanced = strings.ReplaceAll(enhanced, "\u201c", "\"")
	enhanced = strings.ReplaceAll(enhanced, "\u201d", "\"")
	enhanced = strings.ReplaceAll(enhanced, "\u2018", "'")

	// Preserve paragraph structure
	if strings.HasSuffix(original, "\n") && !strings.HasSuffix(enhanced, "\n") {
		enhanced += "\n"
	}

	// Fix sentence capitalization.
	//
	// The first character must be decoded as a *rune*, not indexed as a byte:
	// the translator's primary target scripts (Serbian/Russian Cyrillic, accented
	// Latin) have multibyte first characters, so the previous `enhanced[0]` /
	// `original[0]` byte-indexing fed a UTF-8 lead byte (e.g. 0xD0) to ASCII-only
	// helpers, making this correction silently dead for exactly the languages
	// this tool produces. unicode.IsLower / IsUpper / ToUpper operate on the
	// decoded rune and capitalize Cyrillic/accented letters correctly.
	if len(enhanced) > 0 && len(original) > 0 {
		eFirst, _ := utf8.DecodeRuneInString(enhanced)
		oFirst, _ := utf8.DecodeRuneInString(original)
		if eFirst != utf8.RuneError && oFirst != utf8.RuneError &&
			unicode.IsLower(eFirst) && unicode.IsUpper(oFirst) {
			runes := []rune(enhanced)
			runes[0] = unicode.ToUpper(runes[0])
			enhanced = string(runes)
		}
	}

	return enhanced
}

// Helper functions.
//
// NOTE: these ASCII-only helpers are retained for their existing unit tests;
// enhanceTranslation itself uses the unicode package so capitalization works for
// the multibyte (Cyrillic / accented Latin) scripts this translator targets.
func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func toUpper(r rune) rune {
	if isLower(r) {
		return r - 32
	}
	return r
}
