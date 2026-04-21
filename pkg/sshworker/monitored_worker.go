package sshworker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/logger"
)

// MonitoredSSHWorker wraps SSHWorker with event monitoring capabilities
type MonitoredSSHWorker struct {
	*SSHWorker
	eventBus   *events.EventBus
	sessionID  string
	progress   map[string]*ProgressTracker
	progressMu sync.RWMutex
	logger     logger.Logger
}

// ProgressTracker tracks progress of long-running operations
type ProgressTracker struct {
	Operation  string                 `json:"operation"`
	Total      int                    `json:"total"`
	Completed  int                    `json:"completed"`
	Current    string                 `json:"current"`
	StartTime  time.Time              `json:"start_time"`
	LastUpdate time.Time              `json:"last_update"`
	Status     string                 `json:"status"`
	Details    map[string]interface{} `json:"details"`
	mu         sync.RWMutex
}

// NewMonitoredSSHWorker creates a new SSH worker with monitoring
func NewMonitoredSSHWorker(config SSHWorkerConfig, eventBus *events.EventBus, sessionID string, logger logger.Logger) (*MonitoredSSHWorker, error) {
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		return nil, err
	}

	return &MonitoredSSHWorker{
		SSHWorker: worker,
		eventBus:  eventBus,
		sessionID: sessionID,
		progress:  make(map[string]*ProgressTracker),
		logger:    logger,
	}, nil
}

// ExecuteCommandWithProgress executes a command with progress tracking
func (m *MonitoredSSHWorker) ExecuteCommandWithProgress(ctx context.Context, operation, command string) (*CommandResult, error) {
	// Create progress tracker
	tracker := &ProgressTracker{
		Operation: operation,
		Status:    "starting",
		StartTime: time.Now(),
		Details:   make(map[string]interface{}),
	}

	m.progressMu.Lock()
	m.progress[operation] = tracker
	m.progressMu.Unlock()

	// Emit start event
	m.emitEvent(events.EventTranslationStarted, fmt.Sprintf("Starting %s", operation), map[string]interface{}{
		"operation":  operation,
		"command":    command,
		"session_id": m.sessionID,
	})

	// Update status to running
	tracker.mu.Lock()
	tracker.Status = "running"
	tracker.Current = "Executing command"
	tracker.LastUpdate = time.Now()
	tracker.Details["command"] = command
	tracker.Details["session_id"] = m.sessionID
	tracker.mu.Unlock()

	m.emitEvent(events.EventTranslationProgress, fmt.Sprintf("%s in progress", operation), map[string]interface{}{
		"operation":  operation,
		"status":     "running",
		"progress":   tracker.GetProgress(),
		"session_id": m.sessionID,
	})

	// Execute the command
	result, err := m.SSHWorker.ExecuteCommand(ctx, command)

	// Update tracker with result
	tracker.mu.Lock()
	if err != nil {
		tracker.Status = "error"
		tracker.Current = fmt.Sprintf("Error: %v", err)
		tracker.Details["error"] = err.Error()
	} else if result.ExitCode != 0 {
		tracker.Status = "error"
		tracker.Current = fmt.Sprintf("Command failed with exit code %d", result.ExitCode)
		tracker.Details["error"] = result.Stderr
		tracker.Details["exit_code"] = result.ExitCode
	} else {
		tracker.Status = "completed"
		tracker.Current = "Command completed successfully"
		tracker.Completed = 100
	}
	tracker.LastUpdate = time.Now()
	tracker.mu.Unlock()

	// Emit completion event
	eventType := events.EventTranslationCompleted
	eventMessage := fmt.Sprintf("%s completed", operation)
	eventData := map[string]interface{}{
		"operation":  operation,
		"status":     tracker.Status,
		"progress":   tracker.GetProgress(),
		"session_id": m.sessionID,
	}

	if tracker.Status == "error" {
		eventType = events.EventTranslationError
		eventMessage = fmt.Sprintf("%s failed", operation)
		if err != nil {
			eventData["error"] = err.Error()
		} else {
			eventData["error"] = result.Stderr
		}
	}

	m.emitEvent(eventType, eventMessage, eventData)

	// Clean up progress tracker
	m.progressMu.Lock()
	delete(m.progress, operation)
	m.progressMu.Unlock()

	return result, err
}

// MonitorLongRunningCommand monitors a long-running command with periodic progress updates
func (m *MonitoredSSHWorker) MonitorLongRunningCommand(ctx context.Context, operation, command string, progressCheckInterval time.Duration) (*CommandResult, error) {
	// Create progress tracker
	tracker := &ProgressTracker{
		Operation: operation,
		Total:     100,
		Status:    "starting",
		StartTime: time.Now(),
		Details: map[string]interface{}{
			"command":    command,
			"session_id": m.sessionID,
		},
	}

	m.progressMu.Lock()
	m.progress[operation] = tracker
	m.progressMu.Unlock()

	// Start command in background
	resultChan := make(chan *CommandResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := m.SSHWorker.ExecuteCommand(ctx, command)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Monitor progress
	ticker := time.NewTicker(progressCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			tracker.mu.Lock()
			tracker.Status = "cancelled"
			tracker.Current = "Operation cancelled"
			tracker.LastUpdate = time.Now()
			tracker.mu.Unlock()

			m.emitEvent(events.EventTranslationError, fmt.Sprintf("%s cancelled", operation), map[string]interface{}{
				"operation":  operation,
				"session_id": m.sessionID,
			})
			return nil, ctx.Err()

		case result := <-resultChan:
			// Command completed successfully
			tracker.mu.Lock()
			tracker.Status = "completed"
			tracker.Completed = 100
			tracker.Current = "Command completed successfully"
			tracker.LastUpdate = time.Now()
			tracker.Details["exit_code"] = result.ExitCode
			tracker.Details["stdout"] = result.Stdout
			if result.Stderr != "" {
				tracker.Details["stderr"] = result.Stderr
			}
			tracker.mu.Unlock()

			m.emitEvent(events.EventTranslationCompleted, fmt.Sprintf("%s completed", operation), map[string]interface{}{
				"operation":  operation,
				"progress":   tracker.GetProgress(),
				"session_id": m.sessionID,
			})

			m.progressMu.Lock()
			delete(m.progress, operation)
			m.progressMu.Unlock()
			return result, nil

		case err := <-errorChan:
			// Command failed
			tracker.mu.Lock()
			tracker.Status = "error"
			tracker.Current = fmt.Sprintf("Error: %v", err)
			tracker.LastUpdate = time.Now()
			tracker.Details["error"] = err.Error()
			tracker.mu.Unlock()

			m.emitEvent(events.EventTranslationError, fmt.Sprintf("%s failed", operation), map[string]interface{}{
				"operation":  operation,
				"error":      err.Error(),
				"session_id": m.sessionID,
			})

			m.progressMu.Lock()
			delete(m.progress, operation)
			m.progressMu.Unlock()
			return nil, err

		case <-ticker.C:
			// Update progress
			tracker.mu.Lock()
			elapsed := time.Since(tracker.StartTime)
			progress := min(95, int(elapsed.Seconds()/10)) // Simple progress based on time
			tracker.Completed = progress
			tracker.Current = fmt.Sprintf("Running... (%v elapsed)", elapsed.Round(time.Second))
			tracker.LastUpdate = time.Now()
			tracker.mu.Unlock()

			m.emitEvent(events.EventTranslationProgress, fmt.Sprintf("%s in progress", operation), map[string]interface{}{
				"operation":  operation,
				"progress":   tracker.GetProgress(),
				"session_id": m.sessionID,
			})
		}
	}
}

// GetProgress returns current progress for all operations
func (m *MonitoredSSHWorker) GetProgress() map[string]*ProgressTracker {
	m.progressMu.RLock()
	defer m.progressMu.RUnlock()

	progress := make(map[string]*ProgressTracker)
	for op, tracker := range m.progress {
		progress[op] = tracker.GetCopy()
	}
	return progress
}

// GetProgressTracker returns progress for a specific operation
func (m *MonitoredSSHWorker) GetProgressTracker(operation string) *ProgressTracker {
	m.progressMu.RLock()
	defer m.progressMu.RUnlock()

	if tracker, ok := m.progress[operation]; ok {
		return tracker.GetCopy()
	}
	return nil
}

// emitEvent emits an event to the event bus
func (m *MonitoredSSHWorker) emitEvent(eventType events.EventType, message string, data map[string]interface{}) {
	if m.eventBus != nil {
		event := events.NewEvent(eventType, message, data)
		event.SessionID = m.sessionID
		m.eventBus.Publish(event)
	}
}

// GetProgress returns the current progress
func (pt *ProgressTracker) GetProgress() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return map[string]interface{}{
		"operation":   pt.Operation,
		"total":       pt.Total,
		"completed":   pt.Completed,
		"current":     pt.Current,
		"start_time":  pt.StartTime,
		"last_update": pt.LastUpdate,
		"status":      pt.Status,
		"details":     pt.Details,
		"elapsed":     time.Since(pt.StartTime).String(),
		"percentage":  pt.getPercentage(),
	}
}

// GetCopy returns a copy of the progress tracker
func (pt *ProgressTracker) GetCopy() *ProgressTracker {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// Deep copy details
	detailsCopy := make(map[string]interface{})
	for k, v := range pt.Details {
		detailsCopy[k] = v
	}

	return &ProgressTracker{
		Operation:  pt.Operation,
		Total:      pt.Total,
		Completed:  pt.Completed,
		Current:    pt.Current,
		StartTime:  pt.StartTime,
		LastUpdate: pt.LastUpdate,
		Status:     pt.Status,
		Details:    detailsCopy,
	}
}

// getPercentage returns completion percentage
func (pt *ProgressTracker) getPercentage() float64 {
	if pt.Total == 0 {
		return 0
	}
	return float64(pt.Completed) / float64(pt.Total) * 100
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
