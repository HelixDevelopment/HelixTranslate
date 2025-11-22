package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"digital.vasic.translator/pkg/events"
)

// RemoteLLMInstance represents a remote LLM instance
type RemoteLLMInstance struct {
	ID        string
	WorkerID  string
	Provider  string
	Model     string
	Priority  int
	Available bool
	LastUsed  time.Time
	mu        sync.Mutex
}

// DistributedCoordinator manages distributed LLM instances across remote workers
type DistributedCoordinator struct {
	localCoordinator interface{} // Will be *coordination.MultiLLMCoordinator
	remoteInstances  []*RemoteLLMInstance
	sshPool          *SSHPool
	pairingManager   *PairingManager
	fallbackManager  *FallbackManager
	eventBus         *events.EventBus
	currentIndex     int
	maxRetries       int
	retryDelay       time.Duration
	mu               sync.RWMutex
}

// NewDistributedCoordinator creates a new distributed coordinator
func NewDistributedCoordinator(
	localCoordinator interface{},
	sshPool *SSHPool,
	pairingManager *PairingManager,
	fallbackManager *FallbackManager,
	eventBus *events.EventBus,
) *DistributedCoordinator {
	return &DistributedCoordinator{
		localCoordinator: localCoordinator,
		remoteInstances:  make([]*RemoteLLMInstance, 0),
		sshPool:          sshPool,
		pairingManager:   pairingManager,
		fallbackManager:  fallbackManager,
		eventBus:         eventBus,
		currentIndex:     0,
		maxRetries:       3,
		retryDelay:       2 * time.Second,
	}
}

// DiscoverRemoteInstances discovers LLM instances on paired remote workers
func (dc *DistributedCoordinator) DiscoverRemoteInstances(ctx context.Context) error {
	pairedServices := dc.pairingManager.GetPairedServices()

	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Clear existing remote instances
	dc.remoteInstances = make([]*RemoteLLMInstance, 0)

	instanceID := 1
	for workerID, service := range pairedServices {
		// Query remote service for available providers
		providers, err := dc.queryRemoteProviders(ctx, service)
		if err != nil {
			dc.emitWarning(fmt.Sprintf("Failed to query providers from worker %s: %v", workerID, err))
			continue
		}

		// Create instances based on provider priorities
		for provider, config := range providers {
			priority := config["priority"].(int)
			model := config["model"].(string)

			// Create multiple instances based on priority
			instanceCount := dc.getInstanceCountForPriority(priority, service.Capabilities.MaxConcurrent)

			for i := 0; i < instanceCount; i++ {
				instance := &RemoteLLMInstance{
					ID:        fmt.Sprintf("remote-%s-%d", provider, instanceID),
					WorkerID:  workerID,
					Provider:  provider,
					Model:     model,
					Priority:  priority,
					Available: true,
					LastUsed:  time.Time{},
				}

				dc.remoteInstances = append(dc.remoteInstances, instance)
				instanceID++
			}
		}
	}

	dc.emitEvent(events.Event{
		Type:      "distributed_instances_discovered",
		SessionID: "system",
		Message:   fmt.Sprintf("Discovered %d remote LLM instances across %d workers", len(dc.remoteInstances), len(pairedServices)),
		Data: map[string]interface{}{
			"remote_instances": len(dc.remoteInstances),
			"workers":          len(pairedServices),
		},
	})

	return nil
}

// queryRemoteProviders queries a remote service for available providers
func (dc *DistributedCoordinator) queryRemoteProviders(ctx context.Context, service *RemoteService) (map[string]map[string]interface{}, error) {
	url := fmt.Sprintf("%s://%s:%d/api/v1/providers", service.Protocol, service.Host, service.Port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	providers := make(map[string]map[string]interface{})

	// Extract providers from response
	if providersList, ok := response["providers"].(map[string]interface{}); ok {
		for provider, config := range providersList {
			if configMap, ok := config.(map[string]interface{}); ok {
				providers[provider] = configMap
			}
		}
	}

	return providers, nil
}

// getInstanceCountForPriority determines how many instances to create based on priority and max concurrent
func (dc *DistributedCoordinator) getInstanceCountForPriority(priority int, maxConcurrent int) int {
	baseCount := 1

	switch {
	case priority >= 10: // API key providers
		baseCount = 3
	case priority >= 5: // OAuth providers
		baseCount = 2
	default: // Free/local providers
		baseCount = 1
	}

	// Don't exceed max concurrent capacity
	if baseCount > maxConcurrent {
		baseCount = maxConcurrent
	}

	return baseCount
}

// TranslateWithDistributedRetry translates using distributed instances with comprehensive fallback
func (dc *DistributedCoordinator) TranslateWithDistributedRetry(
	ctx context.Context,
	text string,
	contextHint string,
) (string, error) {

	var result string
	var resultMu sync.Mutex

	// Define fallback strategies
	fallbacks := []FallbackStrategy{
		{
			Name: "remote_instances",
			Function: func() error {
				translated, err := dc.translateWithRemoteInstances(ctx, text, contextHint)
				if err != nil {
					return err
				}
				resultMu.Lock()
				result = translated
				resultMu.Unlock()
				return nil
			},
			Priority: 1,
		},
		{
			Name: "local_coordinator",
			Function: func() error {
				// This would call the local coordinator's method
				// For now, return an error indicating local fallback not implemented
				return fmt.Errorf("local coordinator fallback not yet implemented")
			},
			Priority: 2,
		},
		{
			Name: "reduced_quality",
			Function: func() error {
				// Implement reduced quality fallback (e.g., dictionary-only translation)
				return fmt.Errorf("reduced quality fallback not yet implemented")
			},
			Priority: 3,
		},
	}

	// Use FallbackManager for comprehensive fallback handling
	componentID := "distributed_translator"
	err := dc.fallbackManager.ExecuteWithFallback(ctx, componentID, func() error {
		translated, err := dc.translateWithRemoteInstances(ctx, text, contextHint)
		if err != nil {
			return err
		}
		resultMu.Lock()
		result = translated
		resultMu.Unlock()
		return nil
	}, fallbacks...)

	resultMu.Lock()
	finalResult := result
	resultMu.Unlock()

	if err != nil {
		return "", err
	}

	return finalResult, nil
}

// translateWithRemoteInstances attempts translation using remote instances
func (dc *DistributedCoordinator) translateWithRemoteInstances(
	ctx context.Context,
	text string,
	contextHint string,
) (string, error) {

	if len(dc.remoteInstances) == 0 {
		return "", fmt.Errorf("no remote instances available")
	}

	var lastErr error
	triedInstances := make(map[string]bool)

	for attempt := 0; attempt < dc.maxRetries*len(dc.remoteInstances); attempt++ {
		instance := dc.getNextRemoteInstance()
		if instance == nil {
			break
		}

		if triedInstances[instance.ID] {
			continue
		}

		triedInstances[instance.ID] = true

		dc.emitEvent(events.Event{
			Type:      "distributed_translation_attempt",
			SessionID: "system",
			Message:   fmt.Sprintf("Attempting distributed translation with %s on worker %s", instance.ID, instance.WorkerID),
			Data: map[string]interface{}{
				"instance_id": instance.ID,
				"worker_id":   instance.WorkerID,
				"provider":    instance.Provider,
				"attempt":     attempt + 1,
			},
		})

		result, err := dc.translateWithRemoteInstance(ctx, instance, text, contextHint)
		if err == nil && result != "" {
			instance.LastUsed = time.Now()
			dc.emitEvent(events.Event{
				Type:      "distributed_translation_success",
				SessionID: "system",
				Message:   fmt.Sprintf("Distributed translation successful with %s", instance.ID),
				Data: map[string]interface{}{
					"instance_id": instance.ID,
					"worker_id":   instance.WorkerID,
				},
			})
			return result, nil
		}

		lastErr = err
		dc.emitWarning(fmt.Sprintf("Distributed translation failed with %s: %v", instance.ID, err))

		// Mark instance as temporarily unavailable
		if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "429") {
			instance.Available = false
			go dc.reenableRemoteInstance(instance, 30*time.Second)
		}

		if attempt < dc.maxRetries*len(dc.remoteInstances)-1 {
			time.Sleep(dc.retryDelay)
		}
	}

	return "", fmt.Errorf("distributed translation failed after %d attempts: %w", dc.maxRetries, lastErr)
}

// translateWithRemoteInstance translates using a specific remote instance
func (dc *DistributedCoordinator) translateWithRemoteInstance(
	ctx context.Context,
	instance *RemoteLLMInstance,
	text string,
	contextHint string,
) (string, error) {

	service, exists := dc.pairingManager.services[instance.WorkerID]
	if !exists {
		return "", fmt.Errorf("service %s not found", instance.WorkerID)
	}

	// Prepare translation request
	requestBody := map[string]interface{}{
		"text":         text,
		"context_hint": contextHint,
		"provider":     instance.Provider,
		"model":        instance.Model,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%d/api/v1/translate", service.Protocol, service.Host, service.Port)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if translated, ok := response["translated_text"].(string); ok && translated != "" {
		return translated, nil
	}

	return "", fmt.Errorf("invalid response format")
}

// getNextRemoteInstance gets the next available remote instance (round-robin)
func (dc *DistributedCoordinator) getNextRemoteInstance() *RemoteLLMInstance {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if len(dc.remoteInstances) == 0 {
		return nil
	}

	startIndex := dc.currentIndex
	for {
		instance := dc.remoteInstances[dc.currentIndex]
		dc.currentIndex = (dc.currentIndex + 1) % len(dc.remoteInstances)

		if instance.Available {
			return instance
		}

		if dc.currentIndex == startIndex {
			return nil
		}
	}
}

// reenableRemoteInstance re-enables a remote instance after delay
func (dc *DistributedCoordinator) reenableRemoteInstance(instance *RemoteLLMInstance, delay time.Duration) {
	time.Sleep(delay)
	instance.mu.Lock()
	instance.Available = true
	instance.mu.Unlock()

	dc.emitEvent(events.Event{
		Type:      "distributed_instance_reenabled",
		SessionID: "system",
		Message:   fmt.Sprintf("Remote instance %s re-enabled after cooldown", instance.ID),
		Data: map[string]interface{}{
			"instance_id": instance.ID,
			"worker_id":   instance.WorkerID,
		},
	})
}

// GetRemoteInstanceCount returns the number of remote instances
func (dc *DistributedCoordinator) GetRemoteInstanceCount() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.remoteInstances)
}

// emitEvent emits an event if event bus is available
func (dc *DistributedCoordinator) emitEvent(event events.Event) {
	if dc.eventBus != nil {
		dc.eventBus.Publish(event)
	}
}

// emitWarning emits a warning event
func (dc *DistributedCoordinator) emitWarning(message string) {
	if dc.eventBus != nil {
		dc.eventBus.Publish(events.Event{
			Type:      "distributed_warning",
			SessionID: "system",
			Message:   message,
		})
	}
}
