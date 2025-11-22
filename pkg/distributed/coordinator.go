package distributed

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"digital.vasic.translator/pkg/deployment"
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
	versionManager   *VersionManager
	eventBus         *events.EventBus
	apiLogger        *deployment.APICommunicationLogger
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
	versionManager *VersionManager,
	eventBus *events.EventBus,
	apiLogger *deployment.APICommunicationLogger,
) *DistributedCoordinator {
	return &DistributedCoordinator{
		localCoordinator: localCoordinator,
		remoteInstances:  make([]*RemoteLLMInstance, 0),
		sshPool:          sshPool,
		pairingManager:   pairingManager,
		fallbackManager:  fallbackManager,
		versionManager:   versionManager,
		eventBus:         eventBus,
		apiLogger:        apiLogger,
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

		// Create instances based on provider capabilities
		for provider, config := range providers {
			// Determine priority based on provider type
			priority := dc.getPriorityForProvider(provider)

			// Get first model from models array, or use provider name as fallback
			model := provider // default
			if models, ok := config["models"].([]interface{}); ok && len(models) > 0 {
				if firstModel, ok := models[0].(string); ok {
					model = firstModel
				}
			}

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

	// Log outgoing request if logger is available
	var logEntry *deployment.APICommunicationLog
	if dc.apiLogger != nil {
		logEntry = dc.apiLogger.LogRequest(service.Host, 8443, service.Host, service.Port, "GET", "/api/v1/providers", 0)
	}

	// Use HTTP client that accepts self-signed certificates
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		// Log failed response if logger is available
		if dc.apiLogger != nil && logEntry != nil {
			dc.apiLogger.LogResponse(logEntry, 0, 0, duration, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Log failed response if logger is available
		if dc.apiLogger != nil && logEntry != nil {
			dc.apiLogger.LogResponse(logEntry, resp.StatusCode, 0, duration, err)
		}
		return nil, err
	}

	// Log successful response if logger is available
	if dc.apiLogger != nil && logEntry != nil {
		dc.apiLogger.LogResponse(logEntry, resp.StatusCode, int64(len(body)), duration, nil)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	providers := make(map[string]map[string]interface{})

	// Extract providers from response - handle both array and map formats
	if providersList, ok := response["providers"].([]interface{}); ok {
		// Array format: [{"name": "ollama", "models": [...], ...}, ...]
		for _, item := range providersList {
			if providerMap, ok := item.(map[string]interface{}); ok {
				if name, ok := providerMap["name"].(string); ok {
					providers[name] = providerMap
				}
			}
		}
	} else if providersMap, ok := response["providers"].(map[string]interface{}); ok {
		// Map format: {"ollama": {...}, ...}
		for provider, config := range providersMap {
			if configMap, ok := config.(map[string]interface{}); ok {
				providers[provider] = configMap
			}
		}
	}

	return providers, nil
}

// getPriorityForProvider determines priority based on provider type
func (dc *DistributedCoordinator) getPriorityForProvider(provider string) int {
	switch provider {
	case "openai", "anthropic", "zhipu", "deepseek":
		return 10 // API key providers - highest priority
	case "ollama", "llamacpp":
		return 5 // Local LLM providers - medium priority
	default:
		return 1 // Default priority
	}
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

		// Validate worker version before attempting translation
		if err := dc.validateWorkerForWork(ctx, instance.WorkerID); err != nil {
			dc.emitWarning(fmt.Sprintf("Worker %s validation failed: %v", instance.WorkerID, err))
			continue
		}

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
		dc.emitWarning(fmt.Sprintf("Distributed translation attempt %d failed: %v", attempt+1, err))
	}

	return "", fmt.Errorf("all distributed translation attempts failed, last error: %w", lastErr)
}

// validateWorkerForWork validates that a worker is ready for work
func (dc *DistributedCoordinator) validateWorkerForWork(ctx context.Context, workerID string) error {
	if dc.versionManager == nil {
		// Version manager not available, skip validation
		return nil
	}

	// Get the service for this worker
	services := dc.pairingManager.GetPairedServices()
	service, exists := services[workerID]
	if !exists {
		return fmt.Errorf("worker %s not found in paired services", workerID)
	}

	// Validate worker version and health
	return dc.versionManager.ValidateWorkerForWork(ctx, service)
}

// getNextRemoteInstance returns the next remote instance in round-robin fashion
func (dc *DistributedCoordinator) getNextRemoteInstance() *RemoteLLMInstance {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if len(dc.remoteInstances) == 0 {
		return nil
	}

	// Use round-robin selection
	instance := dc.remoteInstances[dc.currentIndex]
	dc.currentIndex = (dc.currentIndex + 1) % len(dc.remoteInstances)

	return instance
}

// translateWithRemoteInstance performs translation using a specific remote instance
func (dc *DistributedCoordinator) translateWithRemoteInstance(
	ctx context.Context,
	instance *RemoteLLMInstance,
	text string,
	contextHint string,
) (string, error) {
	// Get the service for this worker
	services := dc.pairingManager.GetPairedServices()
	service, exists := services[instance.WorkerID]
	if !exists {
		return "", fmt.Errorf("service not found for worker %s", instance.WorkerID)
	}

	// Prepare translation request
	translateURL := fmt.Sprintf("%s://%s:%d/api/v1/translate", service.Protocol, service.Host, service.Port)

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

	req, err := http.NewRequestWithContext(ctx, "POST", translateURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Log outgoing request
	var logEntry *deployment.APICommunicationLog
	if dc.apiLogger != nil {
		logEntry = dc.apiLogger.LogRequest(service.Host, 8443, service.Host, service.Port, "POST", "/api/v1/translate", int64(len(jsonData)))
	}

	// Use HTTP client that accepts self-signed certificates
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		// Log failed response
		if dc.apiLogger != nil && logEntry != nil {
			dc.apiLogger.LogResponse(logEntry, 0, 0, duration, err)
		}
		return "", fmt.Errorf("translation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Log failed response
		if dc.apiLogger != nil && logEntry != nil {
			dc.apiLogger.LogResponse(logEntry, resp.StatusCode, 0, duration, err)
		}
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Log successful response
	if dc.apiLogger != nil && logEntry != nil {
		dc.apiLogger.LogResponse(logEntry, resp.StatusCode, int64(len(body)), duration, nil)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("translation failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract translated text
	translated, ok := response["translated_text"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format: missing translated_text")
	}

	return translated, nil
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
