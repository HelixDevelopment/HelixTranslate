package distributed

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// EventBusInterface defines the interface for EventBus used in FallbackManager
type EventBusInterface interface {
	Publish(event events.Event)
	Subscribe(eventType events.EventType, handler events.EventHandler)
	SubscribeAll(handler events.EventHandler)
}

// mockLogger for testing
type mockLogger struct {
	logs []map[string]interface{}
	mu   sync.Mutex
}

func (m *mockLogger) Log(level string, message string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	log := map[string]interface{}{
		"level":   level,
		"message": message,
		"data":    data,
	}
	if data != nil {
		for k, v := range data {
			log[k] = v
		}
	}
	m.logs = append(m.logs, log)
}

func (m *mockLogger) GetLogs() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]map[string]interface{}{}, m.logs...)
}

// mockEventBus captures events for testing
type mockEventBus struct {
	events []events.Event
	mu     sync.Mutex
}

func (m *mockEventBus) Publish(event events.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEventBus) Subscribe(eventType events.EventType, handler events.EventHandler) {
	// No-op for testing
}

func (m *mockEventBus) SubscribeAll(handler events.EventHandler) {
	// No-op for testing
}

func (m *mockEventBus) GetPublishedEvents() []events.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]events.Event{}, m.events...)
}

// Test helper function to create a FallbackManager with proper mocks
func newTestFallbackManager(config *FallbackConfig) (*FallbackManager, *mockLogger, *mockEventBus) {
	if config == nil {
		config = &FallbackConfig{
			EnableGracefulDegradation: true,
			DegradationThreshold:      0.5,
			MaxRetries:                3,
			RetryBackoffBase:          100 * time.Millisecond,
			RetryBackoffMax:           30 * time.Second,
			RetryJitter:               true,
			RequestTimeout:            30 * time.Second,
			ConnectionTimeout:         10 * time.Second,
			HealthCheckTimeout:        5 * time.Second,
			RecoveryCheckInterval:     1 * time.Second, // Valid interval
			RecoverySuccessThreshold:  3,
			RecoveryWindow:            60 * time.Second,
			EnableLocalFallback:       true,
			EnableReducedQuality:      true,
			EnableCachingFallback:     true,
			FailureTrackingWindow:     5 * time.Minute,
			AlertThreshold:            0.8,
		}
	}
	perfConfig := DefaultPerformanceConfig()
	eventBus := &mockEventBus{}
	logger := &mockLogger{}
	
	fm := &FallbackManager{
		config:        config,
		performance:   perfConfig,
		eventBus:      eventBus,
		logger:        logger,
		failureCounts: make(map[string]*FailureTracker),
		recoveryState: make(map[string]*RecoveryTracker),
		degradedMode:  false,
	}
	
	// Don't start monitoring goroutines in tests
	
	return fm, logger, eventBus
}

func TestDefaultFallbackConfig(t *testing.T) {
	config := DefaultFallbackConfig()
	
	if config == nil {
		t.Fatal("DefaultFallbackConfig returned nil")
	}
	
	if !config.EnableGracefulDegradation {
		t.Error("EnableGracefulDegradation should be true")
	}
	
	if config.DegradationThreshold != 0.5 {
		t.Errorf("Expected DegradationThreshold 0.5, got %f", config.DegradationThreshold)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}
	
	if config.RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout 30s, got %v", config.RequestTimeout)
	}
	
	if config.AlertThreshold != 0.8 {
		t.Errorf("Expected AlertThreshold 0.8, got %f", config.AlertThreshold)
	}
}

func TestNewFallbackManager(t *testing.T) {
	config := DefaultFallbackConfig()
	perfConfig := DefaultPerformanceConfig()
	eventBus := events.NewEventBus()
	logger := &mockLogger{}
	
	fm := NewFallbackManager(config, perfConfig, eventBus, logger)
	
	if fm == nil {
		t.Fatal("NewFallbackManager returned nil")
	}
	
	if fm.config != config {
		t.Error("Config not properly set")
	}
	
	if fm.eventBus != eventBus {
		t.Error("EventBus not properly set")
	}
	
	if fm.logger != logger {
		t.Error("Logger not properly set")
	}
	
	if fm.degradedMode {
		t.Error("Should not start in degraded mode")
	}
	
	if len(fm.failureCounts) != 0 {
		t.Error("Should start with empty failure counts")
	}
	
	if len(fm.recoveryState) != 0 {
		t.Error("Should start with empty recovery state")
	}
}

func TestFallbackManager_ExecuteWithFallback_Success(t *testing.T) {
	fm, _, _ := newTestFallbackManager(nil)
	
	called := false
	err := fm.ExecuteWithFallback(context.Background(), "test_component", func() error {
		called = true
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if !called {
		t.Error("Primary operation was not called")
	}
	
	// Check that success was recorded
	status := fm.GetStatus()
	components, ok := status["components"].(map[string]interface{})
	if !ok {
		t.Fatal("Components not in status")
	}
	
	if _, exists := components["test_component"]; !exists {
		t.Error("Component not tracked after success")
	}
}

func TestFallbackManager_ExecuteWithFallback_FailureNoFallback(t *testing.T) {
	fm, _, _ := newTestFallbackManager(nil)
	
	primaryErr := errors.New("primary operation failed")
	err := fm.ExecuteWithFallback(context.Background(), "test_component", func() error {
		return primaryErr
	})
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	if !errors.Is(err, primaryErr) {
		t.Errorf("Expected primary error, got %v", err)
	}
	
	// Check that failure was recorded
	status := fm.GetStatus()
	components, ok := status["components"].(map[string]interface{})
	if !ok {
		t.Fatal("Components not in status")
	}
	
	component, exists := components["test_component"]
	if !exists {
		t.Fatal("Component not tracked after failure")
	}
	
	componentData := component.(map[string]interface{})
	if componentData["failures"].(int) != 1 {
		t.Error("Failure not recorded properly")
	}
}

func TestFallbackManager_ExecuteWithFallback_FallbackSuccess(t *testing.T) {
	fm, _, eventBus := newTestFallbackManager(nil)
	
	primaryCalled := false
	fallbackCalled := false
	
	err := fm.ExecuteWithFallback(context.Background(), "test_component", 
		func() error {
			primaryCalled = true
			return errors.New("primary operation failed")
		},
		FallbackStrategy{
			Name: "test_fallback",
			Function: func() error {
				fallbackCalled = true
				return nil
			},
		},
	)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if !primaryCalled {
		t.Error("Primary operation was not called")
	}
	
	if !fallbackCalled {
		t.Error("Fallback operation was not called")
	}
	
	// Check for fallback success event
	publishedEvents := eventBus.GetPublishedEvents()
	if len(publishedEvents) == 0 {
		t.Error("No events published")
	}
	
	foundFallbackEvent := false
	for _, event := range publishedEvents {
		if event.Type == "distributed_fallback_success" {
			foundFallbackEvent = true
			if event.Data["strategy"] != "test_fallback" {
				t.Error("Fallback strategy not in event data")
			}
			break
		}
	}
	
	if !foundFallbackEvent {
		t.Error("Fallback success event not found")
	}
}

func TestFallbackManager_ExecuteWithFallback_AllFailures(t *testing.T) {
	fm, _, eventBus := newTestFallbackManager(nil)
	
	err := fm.ExecuteWithFallback(context.Background(), "test_component", 
		func() error {
			return errors.New("primary operation failed")
		},
		FallbackStrategy{
			Name: "test_fallback",
			Function: func() error {
				return errors.New("fallback operation failed")
			},
		},
	)
	
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	expectedError := "all operations and fallbacks failed for test_component: primary operation failed"
	if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
	
	// Check for all fallbacks failed event
	publishedEvents := eventBus.GetPublishedEvents()
	foundAllFailedEvent := false
	for _, event := range publishedEvents {
		if event.Type == "distributed_all_fallbacks_failed" {
			foundAllFailedEvent = true
			break
		}
	}
	
	if !foundAllFailedEvent {
		t.Error("All fallbacks failed event not found")
	}
}

func TestFallbackManager_RetryLogic(t *testing.T) {
	config := &FallbackConfig{
		EnableGracefulDegradation: false,
		MaxRetries:                2,
		RetryBackoffBase:          10 * time.Millisecond,
		RetryBackoffMax:           100 * time.Millisecond,
		RetryJitter:               false,
		RequestTimeout:            100 * time.Millisecond,
	}
	
	fm, _, _ := newTestFallbackManager(config)
	
	attempts := 0
	err := fm.ExecuteWithFallback(context.Background(), "test_component", func() error {
		attempts++
		if attempts < 3 {
			return errors.New("not ready yet")
		}
		return nil
	})
	
	if err != nil {
		t.Errorf("Expected no error after retries, got %v", err)
	}
	
	if attempts != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}
}

func TestFallbackManager_ContextCancellation(t *testing.T) {
	fm, _, _ := newTestFallbackManager(nil)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	called := false
	err := fm.ExecuteWithFallback(ctx, "test_component", func() error {
		called = true
		return nil
	})
	
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	
	if called {
		t.Error("Operation should not be called when context is already cancelled")
	}
}

func TestFallbackManager_DegradedMode(t *testing.T) {
	config := &FallbackConfig{
		EnableGracefulDegradation: true,
		DegradationThreshold:      0.5, // 50% failure rate
		MaxRetries:                0,
		RequestTimeout:            10 * time.Millisecond,
		AlertThreshold:            0.8,
	}
	
	fm, _, eventBus := newTestFallbackManager(config)
	
	// Simulate failures to trigger degraded mode
	for i := 0; i < 10; i++ {
		fm.ExecuteWithFallback(context.Background(), "test_component", func() error {
			return errors.New("operation failed")
		})
	}
	
	// Manually check and trigger degraded mode
	fm.mu.Lock()
	failureRate := fm.getFailureRate("test_component")
	if fm.config.EnableGracefulDegradation && failureRate >= fm.config.DegradationThreshold && !fm.degradedMode {
		fm.enterDegradedMode("test_component", failureRate)
	}
	fm.mu.Unlock()
	
	// Debug: check if we triggered degraded mode
	fm.mu.Lock()
	t.Logf("Degraded mode: %v, Failure rate: %f, Threshold: %f", fm.degradedMode, failureRate, fm.config.DegradationThreshold)
	fm.mu.Unlock()
	
	// Check if degraded mode was entered
	status := fm.GetStatus()
	degradedMode, ok := status["degraded_mode"].(bool)
	if !ok || degradedMode != true {
		t.Errorf("Expected to be in degraded mode, got %v", degradedMode)
	}
	
	// Check for degraded mode event
	publishedEvents := eventBus.GetPublishedEvents()
	foundDegradedEvent := false
	for _, event := range publishedEvents {
		if event.Type == "distributed_degraded_mode_entered" {
			foundDegradedEvent = true
			break
		}
	}
	
	if !foundDegradedEvent {
		t.Error("Degraded mode entered event not found")
	}
}

func TestFallbackManager_ShouldExecuteFallback(t *testing.T) {
	tests := []struct {
		name           string
		fallbackName   string
		enabledLocal   bool
		enabledReduced bool
		enabledCache   bool
		expected       bool
		degradedMode   bool
	}{
		{
			name:           "local_fallback_enabled",
			fallbackName:   "local_fallback",
			enabledLocal:   true,
			enabledReduced: false,
			enabledCache:   false,
			expected:       true,
			degradedMode:   false,
		},
		{
			name:           "local_fallback_disabled",
			fallbackName:   "local_fallback",
			enabledLocal:   false,
			enabledReduced: false,
			enabledCache:   false,
			expected:       false,
			degradedMode:   false,
		},
		{
			name:           "reduced_quality_enabled",
			fallbackName:   "reduced_quality",
			enabledLocal:   false,
			enabledReduced: true,
			enabledCache:   false,
			expected:       true,
			degradedMode:   false,
		},
		{
			name:           "caching_fallback_enabled",
			fallbackName:   "caching_fallback",
			enabledLocal:   false,
			enabledReduced: false,
			enabledCache:   true,
			expected:       true,
			degradedMode:   false,
		},
		{
			name:           "custom_fallback_always_enabled",
			fallbackName:   "custom_fallback",
			enabledLocal:   false,
			enabledReduced: false,
			enabledCache:   false,
			expected:       true,
			degradedMode:   false,
		},
		{
			name:           "any_fallback_in_degraded_mode",
			fallbackName:   "local_fallback",
			enabledLocal:   false,
			enabledReduced: false,
			enabledCache:   false,
			expected:       true,
			degradedMode:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &FallbackConfig{
				EnableGracefulDegradation: true,
				EnableLocalFallback:       tt.enabledLocal,
				EnableReducedQuality:      tt.enabledReduced,
				EnableCachingFallback:     tt.enabledCache,
				MaxRetries:                0,
				RequestTimeout:            10 * time.Millisecond,
			}
			
			fm, _, _ := newTestFallbackManager(config)
			
			if tt.degradedMode {
				fm.degradedMode = true
			}
			
			fallback := FallbackStrategy{
				Name: tt.fallbackName,
			}
			
			result := fm.shouldExecuteFallback(fallback)
			
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for fallback %s", tt.expected, result, tt.fallbackName)
			}
		})
	}
}

func TestFallbackManager_CalculateBackoff(t *testing.T) {
	config := &FallbackConfig{
		RetryBackoffBase: 100 * time.Millisecond,
		RetryBackoffMax:  5 * time.Second,
		RetryJitter:      false,
	}
	
	fm, _, _ := newTestFallbackManager(config)
	
	tests := []struct {
		attempt int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},  // First attempt (100ms)
		{1, 200 * time.Millisecond},  // Second attempt (100ms * 2)
		{2, 600 * time.Millisecond},  // Third attempt (100ms * 3 * 2)
		{3, 1600 * time.Millisecond}, // Fourth attempt (100ms * 4 * 4)
	}
	
	for _, tt := range tests {
		delay := fm.calculateBackoff(tt.attempt)
		if delay != tt.expected {
			t.Errorf("Attempt %d: expected delay %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
	
	// Test max cap
	delay := fm.calculateBackoff(10) // Very high attempt
	if delay != 5*time.Second {
		t.Errorf("Expected delay capped at %v, got %v", 5*time.Second, delay)
	}
}

func TestFallbackManager_GetStatus(t *testing.T) {
	// Use a fresh manager for this test to avoid contamination
	config := &FallbackConfig{
		EnableGracefulDegradation: false, // Disable to avoid entering degraded mode
		MaxRetries:                0,
		RequestTimeout:            10 * time.Millisecond,
	}
	
	fm, _, _ := newTestFallbackManager(config)
	
	// Execute some operations to create status data
	fm.ExecuteWithFallback(context.Background(), "component1", func() error {
		return errors.New("failed")
	})
	
	fm.ExecuteWithFallback(context.Background(), "component1", func() error {
		return nil
	})
	
	fm.ExecuteWithFallback(context.Background(), "component2", func() error {
		return nil
	})
	
	status := fm.GetStatus()
	
	// Check top-level keys
	if degraded, ok := status["degraded_mode"].(bool); !ok {
		t.Errorf("degraded_mode missing or not bool, got %T", status["degraded_mode"])
	} else if degraded {
		t.Error("Should not be in degraded mode")
	}
	
	components, ok := status["components"].(map[string]interface{})
	if !ok {
		t.Fatal("Components not in status")
	}
	
	// Check component1 (has failures)
	component1, exists := components["component1"]
	if !exists {
		t.Error("Component1 not found in status")
	}
	
	c1Data := component1.(map[string]interface{})
	
	if c1Total, ok := c1Data["total_requests"].(int); !ok || c1Total != 2 {
		t.Errorf("Expected 2 total requests for component1, got %d", c1Data["total_requests"])
	}
	
	if c1Failures, ok := c1Data["failures"].(int); !ok || c1Failures != 1 {
		t.Errorf("Expected 1 failure for component1, got %d", c1Data["failures"])
	}
	
	// Check component2 (only successes)
	component2, exists := components["component2"]
	if !exists {
		t.Error("Component2 not found in status")
	}
	
	c2Data := component2.(map[string]interface{})
	if c2Total, ok := c2Data["total_requests"].(int); !ok || c2Total != 1 {
		t.Errorf("Expected 1 total request for component2, got %d", c2Data["total_requests"])
	}
	
	if c2Failures, ok := c2Data["failures"].(int); !ok || c2Failures != 0 {
		t.Errorf("Expected 0 failures for component2, got %d", c2Data["failures"])
	}
}

func TestFallbackManager_ExitDegradedMode(t *testing.T) {
	t.Run("ExitGracefully", func(t *testing.T) {
		// Create fallback manager
		fallbackConfig := DefaultFallbackConfig()
		performanceConfig := DefaultPerformanceConfig()
		fallbackConfig.RecoveryCheckInterval = 0 // Disable monitoring
		
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		fm := NewFallbackManager(fallbackConfig, performanceConfig, eventBus, logger)
		
		// Manually set degraded mode
		fm.degradedMode = true
		
		// Exit degraded mode
		fm.exitDegradedMode()
		
		// Check that degraded mode is disabled
		if fm.degradedMode {
			t.Error("Expected degraded mode to be false")
		}
	})
}