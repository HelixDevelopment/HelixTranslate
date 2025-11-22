package distributed

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.translator/pkg/events"
)

// FallbackConfig holds fallback and recovery configuration
type FallbackConfig struct {
	// Graceful Degradation
	EnableGracefulDegradation bool
	DegradationThreshold      float64 // Percentage of failed requests before degrading

	// Retry Configuration
	MaxRetries       int
	RetryBackoffBase time.Duration
	RetryBackoffMax  time.Duration
	RetryJitter      bool

	// Timeout Configuration
	RequestTimeout     time.Duration
	ConnectionTimeout  time.Duration
	HealthCheckTimeout time.Duration

	// Recovery Configuration
	RecoveryCheckInterval    time.Duration
	RecoverySuccessThreshold int
	RecoveryWindow           time.Duration

	// Fallback Strategies
	EnableLocalFallback   bool
	EnableReducedQuality  bool
	EnableCachingFallback bool

	// Monitoring
	FailureTrackingWindow time.Duration
	AlertThreshold        float64
}

// DefaultFallbackConfig returns secure default fallback configuration
func DefaultFallbackConfig() *FallbackConfig {
	return &FallbackConfig{
		EnableGracefulDegradation: true,
		DegradationThreshold:      0.5, // 50% failure rate triggers degradation
		MaxRetries:                3,
		RetryBackoffBase:          100 * time.Millisecond,
		RetryBackoffMax:           30 * time.Second,
		RetryJitter:               true,
		RequestTimeout:            30 * time.Second,
		ConnectionTimeout:         10 * time.Second,
		HealthCheckTimeout:        5 * time.Second,
		RecoveryCheckInterval:     10 * time.Second,
		RecoverySuccessThreshold:  3,
		RecoveryWindow:            60 * time.Second,
		EnableLocalFallback:       true,
		EnableReducedQuality:      true,
		EnableCachingFallback:     true,
		FailureTrackingWindow:     5 * time.Minute,
		AlertThreshold:            0.8, // 80% failure rate triggers alerts
	}
}

// FallbackManager manages fallback and recovery strategies
type FallbackManager struct {
	config      *FallbackConfig
	performance *PerformanceConfig
	eventBus    *events.EventBus
	logger      Logger

	// State tracking
	failureCounts map[string]*FailureTracker
	recoveryState map[string]*RecoveryTracker
	degradedMode  bool

	mu sync.RWMutex
}

// FailureTracker tracks failures for a component
type FailureTracker struct {
	ComponentID   string
	Failures      int
	TotalRequests int
	LastFailure   time.Time
	WindowStart   time.Time
	mu            sync.Mutex
}

// RecoveryTracker tracks recovery progress
type RecoveryTracker struct {
	ComponentID          string
	ConsecutiveSuccesses int
	LastSuccess          time.Time
	InRecovery           bool
	mu                   sync.Mutex
}

// NewFallbackManager creates a new fallback manager
func NewFallbackManager(config *FallbackConfig, performance *PerformanceConfig, eventBus *events.EventBus, logger Logger) *FallbackManager {
	fm := &FallbackManager{
		config:        config,
		performance:   performance,
		eventBus:      eventBus,
		logger:        logger,
		failureCounts: make(map[string]*FailureTracker),
		recoveryState: make(map[string]*RecoveryTracker),
		degradedMode:  false,
	}

	// Start monitoring goroutines
	go fm.monitorFailures()
	go fm.monitorRecovery()

	return fm
}

// ExecuteWithFallback executes a function with comprehensive fallback strategies
func (fm *FallbackManager) ExecuteWithFallback(ctx context.Context, componentID string, operation func() error, fallbacks ...FallbackStrategy) error {
	// Track the operation
	startTime := time.Now()
	defer fm.trackOperation(componentID, startTime)

	// Try primary operation with retries
	err := fm.executeWithRetries(ctx, operation)
	if err == nil {
		fm.recordSuccess(componentID)
		return nil
	}

	fm.recordFailure(componentID, err)

	// Try fallback strategies
	for _, fallback := range fallbacks {
		if fm.shouldExecuteFallback(fallback) {
			fm.logger.Log("info", "Executing fallback strategy", map[string]interface{}{
				"component_id": componentID,
				"strategy":     fallback.Name,
				"error":        err.Error(),
			})

			fallbackErr := fm.executeWithRetries(ctx, fallback.Function)
			if fallbackErr == nil {
				fm.emitEvent(events.Event{
					Type:      "distributed_fallback_success",
					SessionID: "system",
					Message:   fmt.Sprintf("Fallback strategy '%s' succeeded for %s", fallback.Name, componentID),
					Data: map[string]interface{}{
						"component_id": componentID,
						"strategy":     fallback.Name,
						"duration":     time.Since(startTime),
					},
				})
				return nil
			}

			fm.logger.Log("warning", "Fallback strategy failed", map[string]interface{}{
				"component_id": componentID,
				"strategy":     fallback.Name,
				"error":        fallbackErr.Error(),
			})
		}
	}

	// All strategies failed
	fm.emitEvent(events.Event{
		Type:      "distributed_all_fallbacks_failed",
		SessionID: "system",
		Message:   fmt.Sprintf("All fallback strategies failed for %s", componentID),
		Data: map[string]interface{}{
			"component_id": componentID,
			"error":        err.Error(),
			"duration":     time.Since(startTime),
		},
	})

	return fmt.Errorf("all operations and fallbacks failed for %s: %w", componentID, err)
}

// FallbackStrategy represents a fallback strategy
type FallbackStrategy struct {
	Name     string
	Function func() error
	Priority int // Lower number = higher priority
}

// executeWithRetries executes a function with retry logic
func (fm *FallbackManager) executeWithRetries(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= fm.config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute operation with timeout
		opCtx, cancel := context.WithTimeout(ctx, fm.config.RequestTimeout)

		done := make(chan error, 1)
		go func() {
			done <- operation()
		}()

		select {
		case err := <-done:
			cancel()
			if err == nil {
				return nil
			}
			lastErr = err

		case <-opCtx.Done():
			cancel()
			lastErr = opCtx.Err()
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Calculate backoff delay
		if attempt < fm.config.MaxRetries {
			delay := fm.calculateBackoff(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// calculateBackoff calculates exponential backoff delay
func (fm *FallbackManager) calculateBackoff(attempt int) time.Duration {
	delay := time.Duration(attempt+1) * fm.config.RetryBackoffBase

	// Apply exponential backoff
	if attempt > 0 {
		multiplier := 1
		for i := 0; i < attempt-1; i++ {
			multiplier *= 2
		}
		delay = time.Duration(float64(delay) * float64(multiplier))
	}

	// Cap at maximum
	if delay > fm.config.RetryBackoffMax {
		delay = fm.config.RetryBackoffMax
	}

	// Add jitter if enabled
	if fm.config.RetryJitter {
		delay = time.Duration(float64(delay) * (0.5 + 0.5*float64(time.Now().UnixNano()%1000)/1000))
	}

	return delay
}

// shouldExecuteFallback determines if a fallback should be executed
func (fm *FallbackManager) shouldExecuteFallback(fallback FallbackStrategy) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	// In degraded mode, execute all fallbacks
	if fm.degradedMode {
		return true
	}

	// Check if fallback is enabled in config
	switch fallback.Name {
	case "local_fallback":
		return fm.config.EnableLocalFallback
	case "reduced_quality":
		return fm.config.EnableReducedQuality
	case "caching_fallback":
		return fm.config.EnableCachingFallback
	default:
		return true // Execute custom fallbacks
	}
}

// recordSuccess records a successful operation
func (fm *FallbackManager) recordSuccess(componentID string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Update failure tracker
	if tracker, exists := fm.failureCounts[componentID]; exists {
		tracker.mu.Lock()
		tracker.TotalRequests++
		tracker.mu.Unlock()
	}

	// Update recovery tracker
	if tracker, exists := fm.recoveryState[componentID]; exists {
		tracker.mu.Lock()
		tracker.ConsecutiveSuccesses++
		tracker.LastSuccess = time.Now()
		if tracker.ConsecutiveSuccesses >= fm.config.RecoverySuccessThreshold {
			tracker.InRecovery = false
		}
		tracker.mu.Unlock()
	}
}

// recordFailure records a failed operation
func (fm *FallbackManager) recordFailure(componentID string, err error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Update failure tracker
	tracker, exists := fm.failureCounts[componentID]
	if !exists {
		tracker = &FailureTracker{
			ComponentID: componentID,
			WindowStart: time.Now(),
		}
		fm.failureCounts[componentID] = tracker
	}

	tracker.mu.Lock()
	tracker.Failures++
	tracker.TotalRequests++
	tracker.LastFailure = time.Now()
	tracker.mu.Unlock()

	// Update recovery tracker
	recoveryTracker, exists := fm.recoveryState[componentID]
	if !exists {
		recoveryTracker = &RecoveryTracker{
			ComponentID: componentID,
		}
		fm.recoveryState[componentID] = recoveryTracker
	}

	recoveryTracker.mu.Lock()
	recoveryTracker.ConsecutiveSuccesses = 0
	recoveryTracker.InRecovery = true
	recoveryTracker.mu.Unlock()

	// Check if we should enter degraded mode
	failureRate := fm.getFailureRate(componentID)
	if fm.config.EnableGracefulDegradation && failureRate >= fm.config.DegradationThreshold && !fm.degradedMode {
		fm.enterDegradedMode(componentID, failureRate)
	}

	// Check if we should alert
	if failureRate >= fm.config.AlertThreshold {
		fm.emitAlert(componentID, failureRate, err)
	}
}

// getFailureRate calculates the failure rate for a component
func (fm *FallbackManager) getFailureRate(componentID string) float64 {
	tracker, exists := fm.failureCounts[componentID]
	if !exists || tracker.TotalRequests == 0 {
		return 0.0
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	// Reset window if it's too old
	if time.Since(tracker.WindowStart) > fm.config.FailureTrackingWindow {
		tracker.Failures = 1
		tracker.TotalRequests = 1
		tracker.WindowStart = time.Now()
		return 1.0
	}

	return float64(tracker.Failures) / float64(tracker.TotalRequests)
}

// enterDegradedMode enters graceful degradation mode
func (fm *FallbackManager) enterDegradedMode(componentID string, failureRate float64) {
	fm.degradedMode = true

	fm.emitEvent(events.Event{
		Type:      "distributed_degraded_mode_entered",
		SessionID: "system",
		Message:   fmt.Sprintf("Entered degraded mode due to high failure rate on %s", componentID),
		Data: map[string]interface{}{
			"component_id": componentID,
			"failure_rate": failureRate,
			"threshold":    fm.config.DegradationThreshold,
		},
	})

	fm.logger.Log("warning", "Entered degraded mode", map[string]interface{}{
		"component_id": componentID,
		"failure_rate": failureRate,
	})
}

// exitDegradedMode exits graceful degradation mode
func (fm *FallbackManager) exitDegradedMode() {
	fm.degradedMode = false

	fm.emitEvent(events.Event{
		Type:      "distributed_degraded_mode_exited",
		SessionID: "system",
		Message:   "Exited degraded mode - system recovered",
	})

	fm.logger.Log("info", "Exited degraded mode", nil)
}

// emitAlert emits an alert for high failure rates
func (fm *FallbackManager) emitAlert(componentID string, failureRate float64, err error) {
	fm.emitEvent(events.Event{
		Type:      "distributed_failure_alert",
		SessionID: "system",
		Message:   fmt.Sprintf("High failure rate alert for %s", componentID),
		Data: map[string]interface{}{
			"component_id": componentID,
			"failure_rate": failureRate,
			"threshold":    fm.config.AlertThreshold,
			"error":        err.Error(),
		},
	})
}

// monitorFailures monitors failure rates and manages degraded mode
func (fm *FallbackManager) monitorFailures() {
	ticker := time.NewTicker(fm.config.RecoveryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.mu.Lock()

			// Check if we should exit degraded mode
			if fm.degradedMode {
				shouldExit := true
				for componentID := range fm.failureCounts {
					if fm.getFailureRate(componentID) >= fm.config.DegradationThreshold {
						shouldExit = false
						break
					}
				}

				if shouldExit {
					fm.exitDegradedMode()
				}
			}

			fm.mu.Unlock()
		}
	}
}

// monitorRecovery monitors recovery progress
func (fm *FallbackManager) monitorRecovery() {
	ticker := time.NewTicker(fm.config.RecoveryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.mu.Lock()

			now := time.Now()
			for componentID, tracker := range fm.recoveryState {
				tracker.mu.Lock()

				// Reset recovery state if no recent successes
				if tracker.InRecovery && now.Sub(tracker.LastSuccess) > fm.config.RecoveryWindow {
					tracker.ConsecutiveSuccesses = 0
					tracker.InRecovery = false

					fm.logger.Log("info", "Recovery timeout expired", map[string]interface{}{
						"component_id": componentID,
					})
				}

				tracker.mu.Unlock()
			}

			fm.mu.Unlock()
		}
	}
}

// trackOperation tracks operation metrics
func (fm *FallbackManager) trackOperation(componentID string, startTime time.Time) {
	duration := time.Since(startTime)

	// Emit metrics event
	fm.emitEvent(events.Event{
		Type:      "distributed_operation_metrics",
		SessionID: "system",
		Message:   fmt.Sprintf("Operation completed for %s", componentID),
		Data: map[string]interface{}{
			"component_id": componentID,
			"duration_ms":  duration.Milliseconds(),
		},
	})
}

// GetStatus returns the current fallback system status
func (fm *FallbackManager) GetStatus() map[string]interface{} {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	components := make(map[string]interface{})

	for componentID, tracker := range fm.failureCounts {
		tracker.mu.Lock()
		failureRate := float64(tracker.Failures) / float64(tracker.TotalRequests)
		tracker.mu.Unlock()

		recoveryTracker := fm.recoveryState[componentID]
		inRecovery := false
		if recoveryTracker != nil {
			recoveryTracker.mu.Lock()
			inRecovery = recoveryTracker.InRecovery
			recoveryTracker.mu.Unlock()
		}

		components[componentID] = map[string]interface{}{
			"failure_rate":   failureRate,
			"total_requests": tracker.TotalRequests,
			"failures":       tracker.Failures,
			"in_recovery":    inRecovery,
		}
	}

	return map[string]interface{}{
		"degraded_mode": fm.degradedMode,
		"components":    components,
	}
}

// emitEvent emits an event if event bus is available
func (fm *FallbackManager) emitEvent(event events.Event) {
	if fm.eventBus != nil {
		fm.eventBus.Publish(event)
	}
}
