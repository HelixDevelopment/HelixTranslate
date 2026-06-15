package coordination

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// LLMInstance represents a single LLM translator instance
type LLMInstance struct {
	ID         string
	Translator translator.Translator
	Provider   string
	Model      string
	Priority   int // Higher priority = more instances (10=API key, 5=OAuth, 1=free)
	Available  bool
	LastUsed   time.Time
	mu         sync.Mutex
}

// IsAvailable reports whether the instance is currently usable. It guards the
// read with instance.mu so concurrent callers (round-robin selection,
// consensus fan-out, rate-limit disabling, cooldown re-enable) never race on
// the Available field.
func (i *LLMInstance) IsAvailable() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.Available
}

// SetAvailable sets the availability flag under instance.mu.
func (i *LLMInstance) SetAvailable(available bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Available = available
}

// MarkUsed records the last-used timestamp under instance.mu.
func (i *LLMInstance) MarkUsed(t time.Time) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.LastUsed = t
}

// MultiLLMCoordinator manages multiple LLM instances for coordinated translation
type MultiLLMCoordinator struct {
	instances         []*LLMInstance
	currentIndex      int
	mu                sync.RWMutex
	maxRetries        int
	retryDelay        time.Duration
	eventBus          *events.EventBus
	sessionID         string
	disableLocalLLMs  bool
	preferDistributed bool
	distributedCoord  interface{} // *distributed.DistributedCoordinator
	translatorFactory EnsembleTranslatorFactory
}

// EnsembleTranslatorFactory supplies the coordinator's translators from an
// external source (e.g. the provider-diverse LLMsVerifier bridge) instead of the
// built-in per-provider NewLLMTranslator discovery. Each returned translator MUST
// carry a usable GetName()/provider identity. nil = use the built-in discovery
// (default, behaviour-preserving).
//
// This type is deliberately decoupled (§11.4.28): it references only
// translator.Translator + context, never the bridge or internal/verifier
// packages, so the coordination package stays project-not-aware and reusable.
type EnsembleTranslatorFactory func(ctx context.Context) ([]translator.Translator, error)

// CoordinatorConfig holds configuration for the coordinator
type CoordinatorConfig struct {
	MaxRetries        int
	RetryDelay        time.Duration
	EventBus          *events.EventBus
	SessionID         string
	DisableLocalLLMs  bool        // When true, only use distributed workers, no local LLM providers
	PreferDistributed bool        // When true, prefer distributed workers over local LLMs
	DistributedCoord  interface{} // Optional distributed coordinator for remote instances

	// TranslatorFactory, when non-nil, is the injectable seam that supplies the
	// ensemble translators from an external provider-diverse source instead of
	// the built-in NewLLMTranslator discovery loop. The zero value (nil) is the
	// behaviour-preserving default: discovery runs exactly as before.
	TranslatorFactory EnsembleTranslatorFactory
}

// NewMultiLLMCoordinator creates a new multi-LLM coordinator
func NewMultiLLMCoordinator(config CoordinatorConfig) *MultiLLMCoordinator {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 2 * time.Second
	}

	coordinator := &MultiLLMCoordinator{
		instances:         make([]*LLMInstance, 0),
		currentIndex:      0,
		maxRetries:        config.MaxRetries,
		retryDelay:        config.RetryDelay,
		eventBus:          config.EventBus,
		sessionID:         config.SessionID,
		disableLocalLLMs:  config.DisableLocalLLMs,
		preferDistributed: config.PreferDistributed,
		distributedCoord:  config.DistributedCoord,
		translatorFactory: config.TranslatorFactory,
	}

	// Auto-discover and initialize LLM instances
	coordinator.initializeLLMInstances()

	return coordinator
}

// NewMultiLLMCoordinatorWithFactory creates a coordinator whose ensemble
// translators come from the supplied external factory instead of the built-in
// per-provider discovery. It is a thin convenience over setting
// CoordinatorConfig.TranslatorFactory; passing a nil factory is byte-for-byte
// equivalent to NewMultiLLMCoordinator(config) (the default discovery path).
func NewMultiLLMCoordinatorWithFactory(config CoordinatorConfig, factory EnsembleTranslatorFactory) *MultiLLMCoordinator {
	config.TranslatorFactory = factory
	return NewMultiLLMCoordinator(config)
}

// initializeLLMInstances discovers and initializes available LLM instances.
//
// When an external EnsembleTranslatorFactory has been injected it is the source
// of the ensemble and the built-in per-provider discovery is bypassed entirely.
// When no factory is injected (the default), behaviour is byte-for-byte
// identical to before: discoverProviders + the NewLLMTranslator loop run unchanged.
func (c *MultiLLMCoordinator) initializeLLMInstances() {
	if c.translatorFactory != nil {
		c.initializeFromFactory()
		return
	}

	// Check for available LLM providers based on API keys
	providers := c.discoverProviders()

	if len(providers) == 0 {
		if c.disableLocalLLMs {
			c.emitWarning("No LLM providers configured with API keys and local LLMs are disabled - distributed workers expected")
		} else {
			c.emitWarning("No LLM providers configured with API keys")
		}
		return
	}

	// Calculate total instances based on priority
	// API key (priority 10) gets 3 instances, OAuth (priority 5) gets 2, free (priority 1) gets 1
	getInstanceCount := func(priority int) int {
		switch {
		case priority >= 10:
			return 3 // API key providers
		case priority >= 5:
			return 2 // OAuth providers
		default:
			return 1 // Free/local providers
		}
	}

	totalInstances := 0
	for _, config := range providers {
		priority := config["priority"].(int)
		totalInstances += getInstanceCount(priority)
	}

	initMessage := fmt.Sprintf("Initializing %d LLM instances across %d providers", totalInstances, len(providers))
	if c.disableLocalLLMs {
		initMessage += " (local LLMs disabled)"
	}
	if c.preferDistributed {
		initMessage += " (preferring distributed workers)"
	}

	c.emitEvent(events.Event{
		Type:      "multi_llm_init",
		SessionID: c.sessionID,
		Message:   initMessage,
		Data: map[string]interface{}{
			"providers":          providers,
			"disable_local":      c.disableLocalLLMs,
			"prefer_distributed": c.preferDistributed,
		},
	})

	// Create multiple instances per provider based on priority
	// API-key providers get 3x instances, OAuth 2x, free/local 1x
	instanceID := 1
	for provider, config := range providers {
		priority := config["priority"].(int)
		instanceCount := getInstanceCount(priority)

		for i := 0; i < instanceCount; i++ {
			translatorConfig := translator.TranslationConfig{
				Provider: provider,
				Model:    config["model"].(string),
				APIKey:   config["api_key"].(string),
			}

			trans, err := llm.NewLLMTranslator(translatorConfig)
			if err != nil {
				c.emitWarning(fmt.Sprintf("Failed to initialize %s instance %d: %v", provider, i+1, err))
				continue
			}

			instance := &LLMInstance{
				ID:         fmt.Sprintf("%s-%d", provider, instanceID),
				Translator: trans,
				Provider:   provider,
				Model:      config["model"].(string),
				Priority:   priority,
				Available:  true,
				LastUsed:   time.Time{},
			}

			c.instances = append(c.instances, instance)
			instanceID++
		}
	}

	if len(c.instances) == 0 {
		c.emitWarning("No LLM instances could be initialized")
		return
	}

	c.emitEvent(events.Event{
		Type:      "multi_llm_ready",
		SessionID: c.sessionID,
		Message:   fmt.Sprintf("Multi-LLM coordinator ready with %d instances", len(c.instances)),
		Data: map[string]interface{}{
			"instance_count": len(c.instances),
			"providers":      c.getProviderList(),
		},
	})
}

// initializeFromFactory builds the ensemble from the injected
// EnsembleTranslatorFactory. Each returned translator becomes exactly one
// *LLMInstance: Provider/Model are taken from the translator's GetName() identity,
// Priority is the API-key tier (10), and Available starts true.
//
// The factory needs a context but initializeLLMInstances is reached from the
// no-ctx constructor; context.Background() is used here because instance
// construction has no caller deadline to honour (the per-translation deadline is
// carried by the ctx passed to TranslateWithRetry/TranslateWithConsensus).
func (c *MultiLLMCoordinator) initializeFromFactory() {
	translators, err := c.translatorFactory(context.Background())
	if err != nil {
		c.emitWarning(fmt.Sprintf("Ensemble translator factory failed: %v", err))
		return
	}

	if len(translators) == 0 {
		c.emitWarning("Ensemble translator factory returned no translators")
		return
	}

	instanceID := 1
	for _, trans := range translators {
		if trans == nil {
			c.emitWarning("Ensemble translator factory returned a nil translator; skipping")
			continue
		}

		name := trans.GetName()
		instance := &LLMInstance{
			ID:         fmt.Sprintf("%s-%d", name, instanceID),
			Translator: trans,
			Provider:   name,
			Model:      name,
			Priority:   10, // factory-supplied ensemble = API-key tier
			Available:  true,
			LastUsed:   time.Time{},
		}
		c.instances = append(c.instances, instance)
		instanceID++
	}

	if len(c.instances) == 0 {
		c.emitWarning("No LLM instances could be initialized from factory")
		return
	}

	c.emitEvent(events.Event{
		Type:      "multi_llm_ready",
		SessionID: c.sessionID,
		Message:   fmt.Sprintf("Multi-LLM coordinator ready with %d instances (factory)", len(c.instances)),
		Data: map[string]interface{}{
			"instance_count": len(c.instances),
			"providers":      c.getProviderList(),
			"source":         "factory",
		},
	})
}

// discoverProviders checks environment for available LLM API keys
// Priority levels: 10=API key (paid), 5=OAuth, 1=free/local
func (c *MultiLLMCoordinator) discoverProviders() map[string]map[string]interface{} {
	providers := make(map[string]map[string]interface{})

	// Check OpenAI (API key - high priority)
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		providers["openai"] = map[string]interface{}{
			"api_key":  apiKey,
			"model":    getEnvOrDefault("OPENAI_MODEL", "gpt-4"),
			"priority": 10, // API key = high priority
		}
	}

	// Check Anthropic (API key - high priority)
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		providers["anthropic"] = map[string]interface{}{
			"api_key":  apiKey,
			"model":    getEnvOrDefault("ANTHROPIC_MODEL", "claude-3-sonnet-20240229"),
			"priority": 10, // API key = high priority
		}
	}

	// Check Zhipu AI (API key - high priority)
	if apiKey := os.Getenv("ZHIPU_API_KEY"); apiKey != "" {
		providers["zhipu"] = map[string]interface{}{
			"api_key":  apiKey,
			"model":    getEnvOrDefault("ZHIPU_MODEL", "glm-4"),
			"priority": 10, // API key = high priority
		}
	}

	// Check DeepSeek (API key - high priority)
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		providers["deepseek"] = map[string]interface{}{
			"api_key":  apiKey,
			"model":    getEnvOrDefault("DEEPSEEK_MODEL", "deepseek-chat"),
			"priority": 10, // API key = high priority
		}
	}

	// Check Qwen (Alibaba Cloud)
	// Priority depends on authentication method
	if apiKey := os.Getenv("QWEN_API_KEY"); apiKey != "" {
		providers["qwen"] = map[string]interface{}{
			"api_key":  apiKey,
			"model":    getEnvOrDefault("QWEN_MODEL", "qwen-plus"),
			"priority": 10, // API key = high priority
		}
	} else if os.Getenv("SKIP_QWEN_OAUTH") == "" {
		// Check for OAuth credentials (skip in test environments)
		homeDir := os.Getenv("HOME")
		if homeDir != "" {
			qwenOAuthPaths := []string{
				homeDir + "/.translator/qwen_credentials.json",
				homeDir + "/.qwen/oauth_creds.json",
			}
			for _, path := range qwenOAuthPaths {
				if _, err := os.Stat(path); err == nil {
					providers["qwen"] = map[string]interface{}{
						"api_key":  "", // OAuth will be used
						"model":    getEnvOrDefault("QWEN_MODEL", "qwen-plus"),
						"priority": 5, // OAuth = medium priority
					}
					break
				}
			}
		}
	}

	// Local-runtime provider discovery removed per operator decision D2 — no local
	// runtime. The disableLocalLLMs gate is retained (still meaningful for the
	// remaining local-runtime work tracked under R-3/R-4); it currently gates no
	// built-in discovery here.

	return providers
}

// TranslateWithRetry translates text with automatic retry and instance rotation
func (c *MultiLLMCoordinator) TranslateWithRetry(
	ctx context.Context,
	text string,
	contextHint string,
) (string, error) {
	if len(c.instances) == 0 {
		return "", fmt.Errorf("no LLM instances available")
	}

	var lastErr error
	// triedCount records how many times each instance has been attempted in THIS
	// call. The loop budget is maxRetries*len(instances), and an instance may be
	// retried up to maxRetries times: a transient (non-rate-limit) error on one
	// attempt must not permanently disqualify the instance, or a single-instance
	// configuration would never actually retry (the maxRetries multiplier would be
	// dead). Blocking only after maxRetries attempts preserves round-robin rotation
	// (other instances are still preferred first via getNextInstance) while honoring
	// the configured retry budget per instance.
	triedCount := make(map[string]int)

	for attempt := 0; attempt < c.maxRetries*len(c.instances); attempt++ {
		// Honor context cancellation: once the caller has given up, stop
		// rotating through instances and stop waiting between retries. Returning
		// the wrapped ctx error (rather than the last provider error) lets
		// callers distinguish "you cancelled" from "all providers failed".
		if ctxErr := ctx.Err(); ctxErr != nil {
			if lastErr != nil {
				return "", fmt.Errorf("translation aborted (%w) after %d attempts: %v", ctxErr, attempt, lastErr)
			}
			return "", fmt.Errorf("translation aborted: %w", ctxErr)
		}

		// Get next available instance
		instance := c.getNextInstance()
		if instance == nil {
			c.emitWarning("All LLM instances exhausted")
			break
		}

		// Skip only instances that have already exhausted their per-instance retry
		// budget; allow up to maxRetries attempts each so transient failures are
		// genuinely retried.
		if triedCount[instance.ID] >= c.maxRetries {
			continue
		}

		triedCount[instance.ID]++

		c.emitEvent(events.Event{
			Type:      "translation_attempt",
			SessionID: c.sessionID,
			Message:   fmt.Sprintf("Attempting translation with %s (Attempt %d)", instance.ID, attempt+1),
			Data: map[string]interface{}{
				"instance_id": instance.ID,
				"provider":    instance.Provider,
				"attempt":     attempt + 1,
			},
		})

		// Attempt translation
		translated, err := instance.Translator.Translate(ctx, text, contextHint)
		if err == nil && translated != "" {
			// Success!
			instance.MarkUsed(time.Now())
			c.emitEvent(events.Event{
				Type:      "translation_success",
				SessionID: c.sessionID,
				Message:   fmt.Sprintf("Translation successful with %s", instance.ID),
				Data: map[string]interface{}{
					"instance_id": instance.ID,
					"text_length": len(text),
				},
			})
			return translated, nil
		}

		lastErr = err
		c.emitWarning(fmt.Sprintf("Translation failed with %s: %v", instance.ID, err))

		// Mark instance as temporarily unavailable if rate limited.
		// Guard against a nil err: a provider may return ("", nil) (empty
		// result, no error), which is treated as a failure above but must not
		// cause a nil-pointer dereference on err.Error() here.
		if err != nil && (strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "429")) {
			instance.SetAvailable(false)
			go c.reenableInstanceAfterDelay(instance, 30*time.Second)
		}

		// Wait before retry, but abort the wait immediately if the caller
		// cancels the context (a bare time.Sleep would ignore cancellation).
		if attempt < c.maxRetries*len(c.instances)-1 && c.retryDelay > 0 {
			timer := time.NewTimer(c.retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", fmt.Errorf("translation aborted (%w) after %d attempts: %v", ctx.Err(), attempt+1, lastErr)
			case <-timer.C:
			}
		}
	}

	return "", fmt.Errorf("translation failed after %d attempts with %d instances: %w",
		c.maxRetries, len(c.instances), lastErr)
}

// TranslateWithConsensus uses multiple instances to translate and picks best result
func (c *MultiLLMCoordinator) TranslateWithConsensus(
	ctx context.Context,
	text string,
	contextHint string,
	requiredAgreement int,
) (string, error) {
	if len(c.instances) < requiredAgreement {
		requiredAgreement = len(c.instances)
	}

	if requiredAgreement == 0 {
		return c.TranslateWithRetry(ctx, text, contextHint)
	}

	// Collect translations from multiple instances
	type result struct {
		translation string
		instance    string
		err         error
	}

	resultsChan := make(chan result, requiredAgreement)
	instancesUsed := 0

	// Fan out over AVAILABLE instances until we have requiredAgreement of them.
	// Scanning the whole slice (not just the first requiredAgreement entries)
	// ensures that when leading instances are unavailable, healthy instances
	// further down still participate in the consensus instead of silently
	// shrinking the vote (or falling back to retry while healthy instances sit
	// idle).
	for i := 0; i < len(c.instances) && instancesUsed < requiredAgreement; i++ {
		instance := c.instances[i]
		if !instance.IsAvailable() {
			continue
		}

		instancesUsed++
		go func(inst *LLMInstance) {
			translated, err := inst.Translator.Translate(ctx, text, contextHint)
			resultsChan <- result{
				translation: translated,
				instance:    inst.ID,
				err:         err,
			}
		}(instance)
	}

	// Collect results
	translations := make(map[string]int)
	var firstSuccess string

	for i := 0; i < instancesUsed; i++ {
		res := <-resultsChan
		if res.err == nil && res.translation != "" {
			if firstSuccess == "" {
				firstSuccess = res.translation
			}
			translations[res.translation]++
		}
	}

	// Find consensus deterministically.
	//
	// The winner is chosen by a STABLE, documented rule so that identical
	// per-instance outputs always yield the identical consensus result
	// (§11.4.50 determinism): (1) the translation with the highest vote count
	// wins; (2) on an EXACT TIE in count, the lexicographically-smallest
	// translation wins.
	//
	// Iterating the `translations` map directly would let Go's randomized
	// map-iteration order decide a tie, so the same inputs could produce
	// different consensus outputs across runs. Sorting the candidate keys
	// first removes that non-determinism entirely.
	maxCount := 0
	bestTranslation := firstSuccess

	candidates := make([]string, 0, len(translations))
	for translation := range translations {
		candidates = append(candidates, translation)
	}
	sort.Strings(candidates)

	for _, translation := range candidates {
		// Strict `>` combined with the lexicographic pre-sort means the first
		// candidate reaching a given max count keeps it: ties are broken in
		// favour of the lexicographically-smallest translation.
		if translations[translation] > maxCount {
			maxCount = translations[translation]
			bestTranslation = translation
		}
	}

	if bestTranslation != "" {
		c.emitEvent(events.Event{
			Type:      "consensus_reached",
			SessionID: c.sessionID,
			Message:   fmt.Sprintf("Consensus reached with %d/%d agreement", maxCount, instancesUsed),
			Data: map[string]interface{}{
				"agreement_count": maxCount,
				"total_instances": instancesUsed,
			},
		})
		return bestTranslation, nil
	}

	// Fallback to retry mechanism
	return c.TranslateWithRetry(ctx, text, contextHint)
}

// getNextInstance gets the next available instance (round-robin)
func (c *MultiLLMCoordinator) getNextInstance() *LLMInstance {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.instances) == 0 {
		return nil
	}

	// Try to find available instance
	startIndex := c.currentIndex
	for {
		instance := c.instances[c.currentIndex]
		c.currentIndex = (c.currentIndex + 1) % len(c.instances)

		if instance.IsAvailable() {
			return instance
		}

		// If we've checked all instances, return nil
		if c.currentIndex == startIndex {
			return nil
		}
	}
}

// reenableInstanceAfterDelay re-enables an instance after a delay
func (c *MultiLLMCoordinator) reenableInstanceAfterDelay(instance *LLMInstance, delay time.Duration) {
	time.Sleep(delay)
	instance.SetAvailable(true)

	c.emitEvent(events.Event{
		Type:      "instance_reenabled",
		SessionID: c.sessionID,
		Message:   fmt.Sprintf("Instance %s re-enabled after cooldown", instance.ID),
		Data: map[string]interface{}{
			"instance_id": instance.ID,
		},
	})
}

// GetInstanceCount returns the number of active instances
func (c *MultiLLMCoordinator) GetInstanceCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.instances)
}

// getProviderList returns list of unique providers
func (c *MultiLLMCoordinator) getProviderList() []string {
	providers := make(map[string]bool)
	for _, instance := range c.instances {
		providers[instance.Provider] = true
	}

	list := make([]string, 0, len(providers))
	for provider := range providers {
		list = append(list, provider)
	}
	return list
}

// emitEvent emits an event if event bus is available
func (c *MultiLLMCoordinator) emitEvent(event events.Event) {
	if c.eventBus != nil {
		c.eventBus.Publish(event)
	}
}

// emitWarning emits a warning event
func (c *MultiLLMCoordinator) emitWarning(message string) {
	if c.eventBus != nil {
		c.eventBus.Publish(events.Event{
			Type:      "multi_llm_warning",
			SessionID: c.sessionID,
			Message:   message,
		})
	}
}

// getEnvOrDefault gets environment variable or returns default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
