package translator

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"errors"
	"sync"
)

var (
	// ErrNoLLMInstances is returned when no LLM instances are available
	ErrNoLLMInstances = errors.New("no LLM instances available")

	// ErrInvalidProvider is returned when an invalid provider is specified
	ErrInvalidProvider = errors.New("invalid translation provider")
)

// TranslationResult holds the result of a translation
type TranslationResult struct {
	OriginalText   string
	TranslatedText string
	Provider       string
	Cached         bool
	Error          error
}

// TranslationStats tracks translation statistics
type TranslationStats struct {
	Total      int
	Translated int
	Cached     int
	Errors     int
}

// Translator interface defines translation methods
type Translator interface {
	// Translate translates text with optional context
	Translate(ctx context.Context, text string, context string) (string, error)

	// TranslateWithProgress translates and reports progress via events
	TranslateWithProgress(ctx context.Context, text string, context string, eventBus *events.EventBus, sessionID string) (string, error)

	// GetStats returns translation statistics
	GetStats() TranslationStats

	// GetName returns the translator name
	GetName() string
}

// BaseTranslator provides common functionality
type BaseTranslator struct {
	config TranslationConfig
	// mu guards cache and stats for concurrent Translate callers. The
	// translation pipeline runs with -concurrency / -workers, so CheckCache,
	// AddToCache, GetStats and UpdateStats are invoked from multiple
	// goroutines; without this lock concurrent access to the cache map is a
	// runtime "fatal error: concurrent map writes" crash and the stats
	// counters race.
	mu    sync.RWMutex
	stats TranslationStats
	cache map[string]string
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
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return "nil error"
		}(),
	}

	event := events.NewEvent(events.EventTranslationError, message, data)
	event.SessionID = sessionID
	eventBus.Publish(event)
}
