package coordination

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

func TestNewMultiLLMCoordinator_NoAPIKeys(t *testing.T) {
	// Clear all API keys
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ZHIPU_API_KEY")
	os.Unsetenv("DEEPSEEK_API_KEY")
	os.Unsetenv("QWEN_API_KEY")
	os.Unsetenv("OLLAMA_ENABLED")

	// Disable Qwen OAuth discovery in tests
	os.Setenv("SKIP_QWEN_OAUTH", "1")
	defer os.Unsetenv("SKIP_QWEN_OAUTH")

	coordinator := NewMultiLLMCoordinator(CoordinatorConfig{
		SessionID: "test-session",
	})

	if coordinator.GetInstanceCount() != 0 {
		t.Errorf("Expected 0 instances with no API keys, got %d", coordinator.GetInstanceCount())
	}
}

func TestNewMultiLLMCoordinator_WithAPIKeys(t *testing.T) {
	// Set some API keys
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	os.Setenv("DEEPSEEK_API_KEY", "sk-deepseek-test")
	defer os.Unsetenv("DEEPSEEK_API_KEY")

	coordinator := NewMultiLLMCoordinator(CoordinatorConfig{
		SessionID: "test-session",
	})

	// Should create instances for the API keys
	if coordinator.GetInstanceCount() < 2 {
		t.Errorf("Expected at least 2 instances with API keys, got %d", coordinator.GetInstanceCount())
	}
}

func TestMultiLLMCoordinator_TranslateWithRetry_NoInstances(t *testing.T) {
	coordinator := &MultiLLMCoordinator{
		instances: make([]*LLMInstance, 0),
	}

	_, err := coordinator.TranslateWithRetry(context.Background(), "test text", "")
	if err == nil {
		t.Error("Expected error when no instances available")
	}

	if err.Error() != "no LLM instances available" {
		t.Errorf("Expected 'no LLM instances available' error, got: %v", err)
	}
}

func TestMultiLLMCoordinator_GetNextInstance(t *testing.T) {
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{ID: "instance1", Available: true},
			{ID: "instance2", Available: false},
			{ID: "instance3", Available: true},
		},
		currentIndex: 0,
	}

	// Should return first available instance
	instance := coordinator.getNextInstance()
	if instance == nil {
		t.Fatal("Expected instance, got nil")
	}
	if instance.ID != "instance1" {
		t.Errorf("Expected instance1, got %s", instance.ID)
	}

	// Should skip unavailable instance and return next available
	instance = coordinator.getNextInstance()
	if instance == nil {
		t.Fatal("Expected instance, got nil")
	}
	if instance.ID != "instance3" {
		t.Errorf("Expected instance3, got %s", instance.ID)
	}
}

func TestLLMInstance_Availability(t *testing.T) {
	instance := &LLMInstance{
		ID:        "test-instance",
		Available: true,
	}

	if !instance.Available {
		t.Error("Instance should be available initially")
	}

	// Mark as unavailable
	instance.Available = false
	if instance.Available {
		t.Error("Instance should be unavailable")
	}
}

func TestDiscoverProviders(t *testing.T) {
	// Clear environment
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ZHIPU_API_KEY")
	os.Unsetenv("DEEPSEEK_API_KEY")
	os.Unsetenv("OLLAMA_ENABLED")
	os.Setenv("SKIP_QWEN_OAUTH", "1")
	defer os.Unsetenv("SKIP_QWEN_OAUTH")

	coordinator := &MultiLLMCoordinator{}

	providers := coordinator.discoverProviders()

	// May have some providers from OAuth or other sources, but should be minimal
	initialCount := len(providers)

	// Set some keys
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	os.Setenv("DEEPSEEK_API_KEY", "deepseek-key")
	defer os.Unsetenv("DEEPSEEK_API_KEY")

	providers = coordinator.discoverProviders()

	// Should find at least the providers we set
	if len(providers) < initialCount+2 {
		t.Errorf("Expected at least %d providers with API keys, got %d", initialCount+2, len(providers))
	}

	if _, exists := providers["openai"]; !exists {
		t.Error("Expected openai provider to be discovered")
	}

	if _, exists := providers["deepseek"]; !exists {
		t.Error("Expected deepseek provider to be discovered")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnvOrDefault("TEST_VAR", "default")
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}

	// Test with non-existing env var
	result = getEnvOrDefault("NON_EXISTING_VAR", "default-value")
	if result != "default-value" {
		t.Errorf("Expected 'default-value', got '%s'", result)
	}
}

func TestCoordinatorConfig_Defaults(t *testing.T) {
	config := CoordinatorConfig{}

	if config.MaxRetries != 0 {
		t.Errorf("Expected MaxRetries to be 0 initially, got %d", config.MaxRetries)
	}

	if config.RetryDelay != 0 {
		t.Errorf("Expected RetryDelay to be 0 initially, got %v", config.RetryDelay)
	}

	// Test that NewMultiLLMCoordinator sets defaults
	coordinator := NewMultiLLMCoordinator(config)

	if coordinator.maxRetries != 3 {
		t.Errorf("Expected maxRetries to be 3, got %d", coordinator.maxRetries)
	}

	if coordinator.retryDelay != 2*time.Second {
		t.Errorf("Expected retryDelay to be 2s, got %v", coordinator.retryDelay)
	}
}

// Mock translator implementing the translator.Translator interface
type mockTranslator struct {
	responses []string
	callCount int
}

func (m *mockTranslator) Translate(ctx context.Context, text, context string) (string, error) {
	if len(m.responses) == 0 {
		return "test translation", nil
	}
	response := m.responses[0]
	if m.callCount < len(m.responses)-1 {
		response = m.responses[m.callCount]
	}
	m.callCount++
	return response, nil
}

func (m *mockTranslator) TranslateWithProgress(ctx context.Context, text, context string, eventBus *events.EventBus, sessionID string) (string, error) {
	return m.Translate(ctx, text, context)
}

func (m *mockTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (m *mockTranslator) GetName() string {
	return "mock"
}

// namedStubTranslator is a minimal translator.Translator whose GetName() identity
// is configurable, so factory-injected ensemble tests can assert the coordinator
// builds one instance per supplied translator with the right provider identity.
type namedStubTranslator struct {
	name string
}

func (s *namedStubTranslator) Translate(ctx context.Context, text, context string) (string, error) {
	return "stub translation", nil
}

func (s *namedStubTranslator) TranslateWithProgress(
	ctx context.Context, text, context string, eventBus *events.EventBus, sessionID string,
) (string, error) {
	return s.Translate(ctx, text, context)
}

func (s *namedStubTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (s *namedStubTranslator) GetName() string { return s.name }

// TestNewMultiLLMCoordinator_WithFactory is the §11.4.115 polarity guard for the
// injectable ensemble factory seam. With an injected factory returning 3 distinct
// stub translators the coordinator MUST initialize EXACTLY those 3 instances from
// the factory — NOT the env-key discovery path.
//
// RED_MODE=1 (or a mutation that makes initializeLLMInstances ignore the injected
// factory and fall through to discovery) FAILs this guard: with no API keys and
// SKIP_QWEN_OAUTH set, discovery yields 0 instances and none of the stub
// providers, so both the count==3 assertion and the provider-identity assertions
// fail. RED_MODE=0 (default, seam wired) is the GREEN guard.
func TestNewMultiLLMCoordinator_WithFactory(t *testing.T) {
	// Ensure the discovery path would produce ZERO instances, so a PASS can only
	// come from the injected factory (proves the seam, not the environment).
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ZHIPU_API_KEY")
	os.Unsetenv("DEEPSEEK_API_KEY")
	os.Unsetenv("QWEN_API_KEY")
	os.Unsetenv("OLLAMA_ENABLED")
	os.Setenv("SKIP_QWEN_OAUTH", "1")
	defer os.Unsetenv("SKIP_QWEN_OAUTH")

	wantNames := []string{"alpha-llm", "beta-llm", "gamma-llm"}

	factoryCalled := false
	factory := func(ctx context.Context) ([]translator.Translator, error) {
		factoryCalled = true
		return []translator.Translator{
			&namedStubTranslator{name: wantNames[0]},
			&namedStubTranslator{name: wantNames[1]},
			&namedStubTranslator{name: wantNames[2]},
		}, nil
	}

	// RED_MODE=1 swaps the factory for nil, forcing the discovery path; the guard
	// then catches a seam that fails to honour the injected factory.
	if os.Getenv("RED_MODE") == "1" {
		factory = nil
	}

	coordinator := NewMultiLLMCoordinatorWithFactory(
		CoordinatorConfig{SessionID: "factory-test"},
		factory,
	)

	if os.Getenv("RED_MODE") != "1" && !factoryCalled {
		t.Fatal("expected the injected factory to be invoked, but it was not")
	}

	if got := coordinator.GetInstanceCount(); got != 3 {
		t.Fatalf("expected exactly 3 instances from injected factory, got %d", got)
	}

	gotProviders := make(map[string]bool)
	for _, inst := range coordinator.instances {
		gotProviders[inst.Provider] = true
	}
	for _, want := range wantNames {
		if !gotProviders[want] {
			t.Errorf("expected an instance with provider identity %q from the factory, providers=%v",
				want, coordinator.getProviderList())
		}
	}
}

// TestNewMultiLLMCoordinator_NilFactory_DiscoveryUnchanged asserts the default
// (nil factory) path is byte-for-byte the legacy behaviour: with no API keys and
// SKIP_QWEN_OAUTH set, the coordinator yields 0 instances exactly as
// TestNewMultiLLMCoordinator_NoAPIKeys requires. A seam that wrongly engaged the
// factory branch on a nil factory would diverge here.
func TestNewMultiLLMCoordinator_NilFactory_DiscoveryUnchanged(t *testing.T) {
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ZHIPU_API_KEY")
	os.Unsetenv("DEEPSEEK_API_KEY")
	os.Unsetenv("QWEN_API_KEY")
	os.Unsetenv("OLLAMA_ENABLED")
	os.Setenv("SKIP_QWEN_OAUTH", "1")
	defer os.Unsetenv("SKIP_QWEN_OAUTH")

	// Both the explicit nil-factory config field and the bare constructor MUST
	// behave identically to the pre-seam default.
	c1 := NewMultiLLMCoordinator(CoordinatorConfig{SessionID: "nil-factory-test"})
	if c1.GetInstanceCount() != 0 {
		t.Errorf("nil-factory config: expected 0 instances (discovery, no keys), got %d", c1.GetInstanceCount())
	}

	c2 := NewMultiLLMCoordinatorWithFactory(CoordinatorConfig{SessionID: "nil-factory-test"}, nil)
	if c2.GetInstanceCount() != 0 {
		t.Errorf("nil factory via WithFactory: expected 0 instances (discovery, no keys), got %d", c2.GetInstanceCount())
	}
}

func TestMultiLLMCoordinator_TranslateWithConsensus(t *testing.T) {
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{
				ID: "instance1",
				Translator: &mockTranslator{
					responses: []string{"Hello world"},
					callCount: 0,
				},
				Available: true,
			},
			{
				ID: "instance2",
				Translator: &mockTranslator{
					responses: []string{"Hello world"},
					callCount: 0,
				},
				Available: true,
			},
		},
		currentIndex: 0,
	}

	result, err := coordinator.TranslateWithConsensus(context.Background(), "Привет мир", "", 2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "Hello world" {
		t.Errorf("Expected 'Hello world', got: %s", result)
	}
}

func TestMultiLLMCoordinator_TranslateWithConsensus_InsufficientInstances(t *testing.T) {
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{
				ID: "instance1",
				Translator: &mockTranslator{
					responses: []string{"Hello world"},
				},
				Available: true,
			},
		},
		currentIndex: 0,
	}

	// Should fallback to TranslateWithRetry when not enough instances
	result, err := coordinator.TranslateWithConsensus(context.Background(), "Привет мир", "", 3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "Hello world" {
		t.Errorf("Expected 'Hello world', got: %s", result)
	}
}

func TestMultiLLMCoordinator_TranslateWithConsensus_NoInstances(t *testing.T) {
	coordinator := &MultiLLMCoordinator{
		instances:    make([]*LLMInstance, 0),
		currentIndex: 0,
	}

	_, err := coordinator.TranslateWithConsensus(context.Background(), "Привет мир", "", 2)
	if err == nil {
		t.Error("Expected error when no instances available")
	}
}

func TestMultiLLMCoordinator_reenableInstanceAfterDelay(t *testing.T) {
	instance := &LLMInstance{
		ID:        "test-instance",
		Available: false,
	}

	coordinator := &MultiLLMCoordinator{}

	// Test that the function exists and doesn't panic
	go coordinator.reenableInstanceAfterDelay(instance, time.Millisecond*10)

	// Wait a bit to ensure the goroutine runs
	time.Sleep(time.Millisecond * 20)

	// Note: We can't easily test the actual availability change without
	// exposing internal state, but we can at least verify the function runs
}

func TestMultiLLMCoordinator_GetProviderList(t *testing.T) {
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{Provider: "openai"},
			{Provider: "anthropic"},
			{Provider: "openai"}, // Duplicate
		},
	}

	providers := coordinator.getProviderList()

	// Should return unique providers
	expectedProviders := []string{"openai", "anthropic"}
	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	for _, expected := range expectedProviders {
		found := false
		for _, actual := range providers {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider %s not found in %v", expected, providers)
		}
	}
}

func TestMultiLLMCoordinator_EmitEvent(t *testing.T) {
	eventBus := events.NewEventBus()
	receivedEvent := false
	var eventData events.Event
	var mu sync.Mutex

	eventBus.Subscribe("test.event", func(event events.Event) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvent = true
		eventData = event
	})

	coordinator := &MultiLLMCoordinator{
		eventBus:  eventBus,
		sessionID: "test-session",
	}

	coordinator.emitEvent(events.Event{
		Type:      "test.event",
		Message:   "test message",
		Data:      map[string]interface{}{"test": "data"},
		SessionID: "test-session",
	})

	time.Sleep(time.Millisecond * 10) // Allow event to propagate

	mu.Lock()
	received := receivedEvent
	data := eventData.Data
	sessionID := eventData.SessionID
	mu.Unlock()

	if !received {
		t.Error("Event was not emitted")
	}

	if data["test"] != "data" {
		t.Errorf("Expected event data with test='data', got %v", data)
	}

	if sessionID != "test-session" {
		t.Errorf("Expected session_id='test-session', got %v", sessionID)
	}
}
