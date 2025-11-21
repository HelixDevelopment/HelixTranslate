package llm

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"fmt"
	"os"
	"strings"
)

// Provider represents LLM provider types
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderZhipu     Provider = "zhipu"
	ProviderDeepSeek  Provider = "deepseek"
	ProviderQwen      Provider = "qwen"
	ProviderOllama    Provider = "ollama"
	ProviderLlamaCpp  Provider = "llamacpp"
)

// LLMTranslator implements LLM-based translation
type LLMTranslator struct {
	*translator.BaseTranslator
	provider Provider
	client   LLMClient
}

// LLMClient interface for different LLM providers
type LLMClient interface {
	Translate(ctx context.Context, text string, prompt string) (string, error)
	GetProviderName() string
}

// NewLLMTranslator creates a new LLM translator
func NewLLMTranslator(config translator.TranslationConfig) (*LLMTranslator, error) {
	provider := Provider(config.Provider)

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
	case ProviderOllama:
		client, err = NewOllamaClient(config)
	case ProviderLlamaCpp:
		client, err = NewLlamaCppClient(config)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	return &LLMTranslator{
		BaseTranslator: translator.NewBaseTranslator(config),
		provider:       provider,
		client:         client,
	}, nil
}

// GetName returns the translator name
func (lt *LLMTranslator) GetName() string {
	return fmt.Sprintf("llm-%s", lt.provider)
}

// Translate translates text using LLM with automatic retry and text splitting
func (lt *LLMTranslator) Translate(ctx context.Context, text string, contextStr string) (string, error) {
	if text == "" || strings.TrimSpace(text) == "" {
		return text, nil
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s:%s", text, contextStr)
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

	// Split by paragraphs first
	paragraphs := strings.Split(text, "\n\n")

	for _, para := range paragraphs {
		// If single paragraph is too large, split by sentences
		if len(para) > maxChunkSize {
			sentences := lt.splitBySentences(para)
			for _, sentence := range sentences {
				if currentChunk.Len()+len(sentence) > maxChunkSize && currentChunk.Len() > 0 {
					// Current chunk is full, start new chunk
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
				}
				currentChunk.WriteString(sentence)
			}
		} else {
			// Add paragraph to current chunk
			if currentChunk.Len()+len(para)+2 > maxChunkSize && currentChunk.Len() > 0 {
				// Current chunk is full, start new chunk
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
		}
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

// TranslateWithProgress translates and reports progress
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

// createTranslationPrompt creates the translation prompt
func (lt *LLMTranslator) createTranslationPrompt(text string, contextStr string) string {
	context := contextStr
	if context == "" {
		context = "Literary text"
	}

	return fmt.Sprintf(`You are a professional literary translator specializing in Russian to Serbian translation.
Your task is to translate the following Russian text into natural, idiomatic Serbian.

Guidelines:
1. Preserve the literary style and tone
2. Use appropriate Serbian vocabulary and grammar
3. Maintain cultural nuances and idioms
4. Keep names of people and places unchanged unless they have standard Serbian equivalents
5. Preserve formatting, punctuation, and paragraph structure
6. Use Serbian Cyrillic script (ћирилица)
7. **CRITICAL**: Use ONLY Ekavica dialect (екавица) - the standard Serbian dialect used in Serbia
   - Use "е" instead of "ије/је": mleko (not mlijeko), dete (not dijete), pesma (not pjesma)
   - Ekavica examples: hteo (not htio), lepo (not lijepo), reka (not rijeka)
   - This is MANDATORY for all translations to Serbian
8. **CRITICAL**: Use ONLY pure Serbian vocabulary - avoid Croatian, Bosnian, or Montenegrin words
   - Use standard Serbian words preferred in Serbia, not regional variants
   - Example: use "avion" (not Croatian "zrakoplov"), "pozorište" (not Croatian "kazalište")

Context: %s

Russian text:
%s

Serbian translation (Ekavica only):`, context, text)
}

// enhanceTranslation post-processes the translation
func (lt *LLMTranslator) enhanceTranslation(original, translated string) string {
	enhanced := translated

	// Fix common punctuation issues
	enhanced = strings.ReplaceAll(enhanced, "\u201c", "\"")
	enhanced = strings.ReplaceAll(enhanced, "\u201d", "\"")
	enhanced = strings.ReplaceAll(enhanced, "\u2018", "'")

	// Preserve paragraph structure
	if strings.HasSuffix(original, "\n") && !strings.HasSuffix(enhanced, "\n") {
		enhanced += "\n"
	}

	// Fix sentence capitalization
	if len(enhanced) > 0 && len(original) > 0 {
		if isLower(rune(enhanced[0])) && isUpper(rune(original[0])) {
			runes := []rune(enhanced)
			runes[0] = toUpper(runes[0])
			enhanced = string(runes)
		}
	}

	return enhanced
}

// Helper functions
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
