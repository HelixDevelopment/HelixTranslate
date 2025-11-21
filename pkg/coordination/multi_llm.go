package coordination

import (
	"context"
	"fmt"
	"os"
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
	Priority   int       // Higher priority = more instances (10=API key, 5=OAuth, 1=free)
	Available  bool
	LastUsed   time.Time
	mu         sync.Mutex
}

// MultiLLMCoordinator manages multiple LLM instances for coordinated translation
type MultiLLMCoordinator struct {
	instances       []*LLMInstance
	currentIndex    int
	mu              sync.RWMutex
	maxRetries      int
	retryDelay      time.Duration
	eventBus        *events.EventBus
	sessionID       string
}

// CoordinatorConfig holds configuration for the coordinator
type CoordinatorConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	EventBus    *events.EventBus
	SessionID   string
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
		instances:    make([]*LLMInstance, 0),
		currentIndex: 0,
		maxRetries:   config.MaxRetries,
		retryDelay:   config.RetryDelay,
		eventBus:     config.EventBus,
		sessionID:    config.SessionID,
	}

	// Auto-discover and initialize LLM instances
	coordinator.initializeLLMInstances()

	return coordinator
}

// initializeLLMInstances discovers and initializes available LLM instances
func (c *MultiLLMCoordinator) initializeLLMInstances() {
	// Check for available LLM providers based on API keys
	providers := c.discoverProviders()

	if len(providers) == 0 {
		c.emitWarning("No LLM providers configured with API keys")
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

	c.emitEvent(events.Event{
		Type:      "multi_llm_init",
		SessionID: c.sessionID,
		Message:   fmt.Sprintf("Initializing %d LLM instances across %d providers (prioritizing API key providers)", totalInstances, len(providers)),
		Data: map[string]interface{}{
			"providers": providers,
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

	// Check Ollama (local, no API key needed - lowest priority)
	if os.Getenv("OLLAMA_ENABLED") == "true" {
		providers["ollama"] = map[string]interface{}{
			"api_key":  "",
			"model":    getEnvOrDefault("OLLAMA_MODEL", "llama3:8b"),
			"priority": 1, // Free/local = low priority
		}
	}

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
	triedInstances := make(map[string]bool)

	for attempt := 0; attempt < c.maxRetries*len(c.instances); attempt++ {
		// Get next available instance
		instance := c.getNextInstance()
		if instance == nil {
			c.emitWarning("All LLM instances exhausted")
			break
		}

		// Skip if already tried this instance
		if triedInstances[instance.ID] {
			continue
		}

		triedInstances[instance.ID] = true

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
			instance.LastUsed = time.Now()
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

		// Mark instance as temporarily unavailable if rate limited
		if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "429") {
			instance.Available = false
			go c.reenableInstanceAfterDelay(instance, 30*time.Second)
		}

		// Wait before retry
		if attempt < c.maxRetries*len(c.instances)-1 {
			time.Sleep(c.retryDelay)
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

	for i := 0; i < requiredAgreement && i < len(c.instances); i++ {
		instance := c.instances[i]
		if !instance.Available {
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

	// Find consensus
	maxCount := 0
	bestTranslation := firstSuccess

	for translation, count := range translations {
		if count > maxCount {
			maxCount = count
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

		if instance.Available {
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
	instance.mu.Lock()
	instance.Available = true
	instance.mu.Unlock()

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
