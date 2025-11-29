package distributed

import (
	"context"
	"errors"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

func TestFallbackManager_executeWithRetries_EdgeCases(t *testing.T) {
	t.Run("executeWithRetries_ImmediateSuccess", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		fm := NewFallbackManager(DefaultFallbackConfig(), DefaultPerformanceConfig(), eventBus, logger)
		
		// Operation that succeeds immediately
		called := 0
		operation := func() error {
			called++
			return nil
		}
		
		err := fm.executeWithRetries(context.Background(), operation)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if called != 1 {
			t.Errorf("Expected operation to be called once, was called %d times", called)
		}
	})
	
	t.Run("executeWithRetries_ContextCancellationDuringRetries", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.MaxRetries = 5
		config.RequestTimeout = 100 * time.Millisecond
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Operation that always fails
		called := 0
		operation := func() error {
			called++
			return errors.New("operation failed")
		}
		
		// Create a context that will be cancelled after a short delay
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()
		
		err := fm.executeWithRetries(ctx, operation)
		if err == nil {
			t.Error("Expected error due to context cancellation")
		}
		
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
		
		if called < 1 {
			t.Error("Expected operation to be called at least once")
		}
	})
	
	t.Run("executeWithRetries_EventualSuccess", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.MaxRetries = 3
		config.RequestTimeout = 50 * time.Millisecond
		config.RetryBackoffBase = 10 * time.Millisecond
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Operation that fails twice then succeeds
		called := 0
		operation := func() error {
			called++
			if called < 3 {
				return errors.New("operation failed")
			}
			return nil
		}
		
		err := fm.executeWithRetries(context.Background(), operation)
		if err != nil {
			t.Errorf("Expected no error after retries, got %v", err)
		}
		
		if called != 3 {
			t.Errorf("Expected operation to be called 3 times, was called %d times", called)
		}
	})
	
	t.Run("executeWithRetries_AlwaysFails", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.MaxRetries = 2
		config.RequestTimeout = 50 * time.Millisecond
		config.RetryBackoffBase = 10 * time.Millisecond
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Operation that always fails
		called := 0
		operation := func() error {
			called++
			return errors.New("operation failed")
		}
		
		err := fm.executeWithRetries(context.Background(), operation)
		if err == nil {
			t.Error("Expected error after all retries")
		}
		
		if called != 3 { // Initial attempt + 2 retries
			t.Errorf("Expected operation to be called 3 times, was called %d times", called)
		}
	})
}

func TestFallbackManager_monitorFailures(t *testing.T) {
	t.Run("monitorFailures_FailureRateCalculation", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.RecoveryCheckInterval = 50 * time.Millisecond
		config.DegradationThreshold = 0.5
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Manually add failures
		fm.recordFailure("test-component", errors.New("test error 1"))
		fm.recordFailure("test-component", errors.New("test error 2"))
		
		// Add a success
		fm.recordSuccess("test-component")
		
		// Check failure rate
		failureRate := fm.getFailureRate("test-component")
		if failureRate < 0.66 || failureRate > 0.67 { // 2 failures out of 3 total
			t.Errorf("Expected failure rate around 0.667, got %f", failureRate)
		}
	})
	
	t.Run("monitorFailures_DegradedModeEntry", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.DegradationThreshold = 0.3 // Lower threshold for easier testing
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Record enough failures to exceed threshold
		for i := 0; i < 10; i++ {
			fm.recordFailure("test-component", errors.New("test error"))
		}
		
		// Check if degraded mode was entered
		if !fm.degradedMode {
			t.Error("Expected degraded mode to be entered")
		}
	})
	
	t.Run("monitorFailures_DegradedModeExit", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.DegradationThreshold = 0.3
		config.RecoveryCheckInterval = 50 * time.Millisecond
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Enter degraded mode
		for i := 0; i < 10; i++ {
			fm.recordFailure("test-component", errors.New("test error"))
		}
		
		// Verify degraded mode
		if !fm.degradedMode {
			t.Error("Expected degraded mode to be entered")
		}
		
		// Add successes to reduce failure rate
		for i := 0; i < 20; i++ {
			fm.recordSuccess("test-component")
		}
		
		// Manually trigger the exit condition check
		shouldExit := true
		for componentID := range fm.failureCounts {
			if fm.getFailureRate(componentID) >= fm.config.DegradationThreshold {
				shouldExit = false
				break
			}
		}
		
		if shouldExit && fm.degradedMode {
			// Call exitDegradedMode manually to test it
			fm.exitDegradedMode()
			
			if fm.degradedMode {
				t.Error("Expected degraded mode to be exited")
			}
		}
	})
}

func TestFallbackManager_calculateBackoff(t *testing.T) {
	t.Run("calculateBackoff_WithoutJitter", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.RetryBackoffBase = 100 * time.Millisecond
		config.RetryBackoffMax = 5 * time.Second
		config.RetryJitter = false
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Test different attempts
		testCases := []struct {
			attempt  int
			expected time.Duration
		}{
			{0, 100 * time.Millisecond}, // (0+1) * 100ms = 100ms
			{1, 200 * time.Millisecond}, // (1+1) * 100ms = 200ms
			{2, 600 * time.Millisecond}, // (2+1) * 100ms * 2^(2-1) = 300ms * 2 = 600ms
			{3, 1600 * time.Millisecond}, // (3+1) * 100ms * 2^(3-1) = 400ms * 4 = 1600ms
		}
		
		for _, tc := range testCases {
			delay := fm.calculateBackoff(tc.attempt)
			if delay != tc.expected {
				t.Errorf("Attempt %d: expected delay %v, got %v", tc.attempt, tc.expected, delay)
			}
		}
	})
	
	t.Run("calculateBackoff_WithMaxCap", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		config := DefaultFallbackConfig()
		config.RetryBackoffBase = 1 * time.Second
		config.RetryBackoffMax = 3 * time.Second
		config.RetryJitter = false
		fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)
		
		// Test attempt that would exceed max
		delay := fm.calculateBackoff(10) // This should produce a large delay
		if delay > config.RetryBackoffMax {
			t.Errorf("Expected delay to be capped at %v, got %v", config.RetryBackoffMax, delay)
		}
	})
}

func TestFallbackManager_getFailureRate(t *testing.T) {
	t.Run("getFailureRate_EmptyHistory", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		fm := NewFallbackManager(DefaultFallbackConfig(), DefaultPerformanceConfig(), eventBus, logger)
		
		// Check failure rate for component with no history
		rate := fm.getFailureRate("non-existent-component")
		if rate != 0 {
			t.Errorf("Expected failure rate 0 for non-existent component, got %f", rate)
		}
	})
	
	t.Run("getFailureRate_MixedHistory", func(t *testing.T) {
		eventBus := events.NewEventBus()
		logger := &mockLogger{}
		fm := NewFallbackManager(DefaultFallbackConfig(), DefaultPerformanceConfig(), eventBus, logger)
		
		// Record mixed history
		fm.recordSuccess("test-component")
		fm.recordFailure("test-component", errors.New("test error 1"))
		fm.recordSuccess("test-component")
		fm.recordFailure("test-component", errors.New("test error 2"))
		fm.recordFailure("test-component", errors.New("test error 3"))
		
		// Check failure rate (3 failures out of 5 total)
		rate := fm.getFailureRate("test-component")
		if rate < 0.59 || rate > 0.61 { // Allow for small floating point differences
			t.Errorf("Expected failure rate around 0.6, got %f", rate)
		}
	})
}