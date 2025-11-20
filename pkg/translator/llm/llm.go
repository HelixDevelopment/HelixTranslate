package llm

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"fmt"
	"strings"
)

// Provider represents LLM provider types
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderZhipu     Provider = "zhipu"
	ProviderDeepSeek  Provider = "deepseek"
	ProviderOllama    Provider = "ollama"
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
	case ProviderOllama:
		client, err = NewOllamaClient(config)
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

// Translate translates text using LLM
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

	// Translate using LLM
	result, err := lt.client.Translate(ctx, text, prompt)
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

Context: %s

Russian text:
%s

Serbian translation:`, context, text)
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
