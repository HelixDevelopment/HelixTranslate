package distributed

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"digital.vasic.translator/pkg/events"
)

// RemoteService represents a remote translator service
type RemoteService struct {
	WorkerID     string              `json:"worker_id"`
	Name         string              `json:"name"`
	Host         string              `json:"host"`
	Port         int                 `json:"port"`
	Protocol     string              `json:"protocol"` // http, https, http3
	Status       string              `json:"status"`   // online, offline, paired, updating, outdated
	Capabilities ServiceCapabilities `json:"capabilities"`
	Version      VersionInfo         `json:"version"`
	LastSeen     time.Time           `json:"last_seen"`
	PairedAt     *time.Time          `json:"paired_at,omitempty"`
}

// ServiceCapabilities represents what the remote service can do
type ServiceCapabilities struct {
	Providers         []string `json:"providers"`
	MaxConcurrent     int      `json:"max_concurrent"`
	SupportsBatch     bool     `json:"supports_batch"`
	SupportsWebSocket bool     `json:"supports_websocket"`
	LocalLLMs         []string `json:"local_llms,omitempty"` // ollama, llamacpp models
}

// VersionInfo represents version information for codebase and components
type VersionInfo struct {
	CodebaseVersion string            `json:"codebase_version"`
	BuildTime       string            `json:"build_time"`
	GitCommit       string            `json:"git_commit"`
	GoVersion       string            `json:"go_version"`
	Components      map[string]string `json:"components"` // component -> version
	LastUpdated     time.Time         `json:"last_updated"`
}

// PairingManager manages pairing with remote services
type PairingManager struct {
	services      map[string]*RemoteService
	sshPool       *SSHPool
	eventBus      *events.EventBus
	httpClient    *http.Client
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewPairingManager creates a new pairing manager
func NewPairingManager(sshPool *SSHPool, eventBus *events.EventBus) *PairingManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Create HTTP client with reasonable timeouts and TLS config for self-signed certs
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Accept self-signed certificates
			},
		},
	}

	manager := &PairingManager{
		services:      make(map[string]*RemoteService),
		sshPool:       sshPool,
		eventBus:      eventBus,
		httpClient:    httpClient,
		checkInterval: 30 * time.Second,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start health check routine
	go manager.healthCheckLoop()

	return manager
}

// DiscoverService discovers a remote service via SSH
func (pm *PairingManager) DiscoverService(ctx context.Context, workerID string) (*RemoteService, error) {
	conn, err := pm.sshPool.GetConnection(workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection: %w", err)
	}

	// Check if translator service is running
	cmd := "ps aux | grep -E '(translator|translator-server)' | grep -v grep || echo 'not running'"
	output, err := conn.ExecuteCommand(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to check service status: %w", err)
	}

	if strings.Contains(string(output), "not running") {
		return nil, fmt.Errorf("translator service not running on worker %s", workerID)
	}

	// Try to get service info via HTTP
	service, err := pm.queryServiceInfo(workerID)
	if err != nil {
		// Fallback: create basic service info
		config := conn.Config
		service = &RemoteService{
			WorkerID: workerID,
			Name:     config.Name,
			Host:     config.SSH.Host,
			Port:     8443, // Default port
			Protocol: "https",
			Status:   "online",
			Capabilities: ServiceCapabilities{
				Providers:     []string{"dictionary"}, // Basic assumption
				MaxConcurrent: config.MaxCapacity,
				SupportsBatch: true,
			},
			LastSeen: time.Now(),
		}
	}

	pm.services[workerID] = service
	return service, nil
}

// queryServiceInfo queries the remote service for its capabilities
func (pm *PairingManager) queryServiceInfo(workerID string) (*RemoteService, error) {
	conn, err := pm.sshPool.GetConnection(workerID)
	if err != nil {
		return nil, err
	}

	config := conn.Config

	// Try different ports and protocols
	endpoints := []struct {
		host  string
		port  int
		proto string
	}{
		{config.SSH.Host, 8443, "https"},
		{config.SSH.Host, 8080, "http"},
		{config.SSH.Host, 8443, "http"},
	}

	for _, endpoint := range endpoints {
		url := fmt.Sprintf("%s://%s:%d/api/v1/providers", endpoint.proto, endpoint.host, endpoint.port)

		resp, err := pm.httpClient.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Try to parse providers response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		var providers map[string]interface{}
		if err := json.Unmarshal(body, &providers); err != nil {
			continue
		}

		// Get health check info
		healthURL := fmt.Sprintf("%s://%s:%d/health", endpoint.proto, endpoint.host, endpoint.port)
		healthResp, err := pm.httpClient.Get(healthURL)
		if err != nil {
			continue
		}
		defer healthResp.Body.Close()

		var health map[string]interface{}
		if healthBody, err := io.ReadAll(healthResp.Body); err == nil {
			json.Unmarshal(healthBody, &health)
		}

		// Create service info
		service := &RemoteService{
			WorkerID: workerID,
			Name:     config.Name,
			Host:     endpoint.host,
			Port:     endpoint.port,
			Protocol: endpoint.proto,
			Status:   "online",
			Capabilities: ServiceCapabilities{
				MaxConcurrent: config.MaxCapacity,
				SupportsBatch: true,
			},
			LastSeen: time.Now(),
		}

		// Extract providers from response
		if providersList, ok := providers["providers"].([]interface{}); ok {
			for _, p := range providersList {
				if providerName, ok := p.(string); ok {
					service.Capabilities.Providers = append(service.Capabilities.Providers, providerName)
				}
			}
		}

		return service, nil
	}

	return nil, fmt.Errorf("could not reach service on worker %s", workerID)
}

// PairWithService pairs with a discovered remote service
func (pm *PairingManager) PairWithService(workerID string) error {
	service, exists := pm.services[workerID]
	if !exists {
		return fmt.Errorf("service %s not discovered", workerID)
	}

	now := time.Now()
	service.Status = "paired"
	service.PairedAt = &now

	// Emit pairing event
	pm.emitEvent(events.Event{
		Type:      "distributed_worker_paired",
		SessionID: "system",
		Message:   fmt.Sprintf("Successfully paired with remote worker %s", workerID),
		Data: map[string]interface{}{
			"worker_id":    workerID,
			"worker_name":  service.Name,
			"host":         service.Host,
			"capabilities": service.Capabilities,
		},
	})

	return nil
}

// UnpairService unpairs from a remote service
func (pm *PairingManager) UnpairService(workerID string) error {
	service, exists := pm.services[workerID]
	if !exists {
		return fmt.Errorf("service %s not found", workerID)
	}

	service.Status = "online"
	service.PairedAt = nil

	// Emit unpairing event
	pm.emitEvent(events.Event{
		Type:      "distributed_worker_unpaired",
		SessionID: "system",
		Message:   fmt.Sprintf("Unpaired from remote worker %s", workerID),
		Data: map[string]interface{}{
			"worker_id":   workerID,
			"worker_name": service.Name,
		},
	})

	return nil
}

// GetPairedServices returns all paired services
func (pm *PairingManager) GetPairedServices() map[string]*RemoteService {
	paired := make(map[string]*RemoteService)

	for id, service := range pm.services {
		if service.Status == "paired" {
			paired[id] = service
		}
	}

	return paired
}

// GetServiceStatus returns the status of a service
func (pm *PairingManager) GetServiceStatus(workerID string) (string, error) {
	service, exists := pm.services[workerID]
	if !exists {
		return "unknown", fmt.Errorf("service %s not found", workerID)
	}

	return service.Status, nil
}

// healthCheckLoop periodically checks health of paired services
func (pm *PairingManager) healthCheckLoop() {
	ticker := time.NewTicker(pm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.performHealthChecks()

		case <-pm.ctx.Done():
			return
		}
	}
}

// performHealthChecks checks health of all known services
func (pm *PairingManager) performHealthChecks() {
	for workerID, service := range pm.services {
		go pm.checkServiceHealth(workerID, service)
	}
}

// checkServiceHealth checks the health of a single service
func (pm *PairingManager) checkServiceHealth(workerID string, service *RemoteService) {
	url := fmt.Sprintf("%s://%s:%d/health", service.Protocol, service.Host, service.Port)

	resp, err := pm.httpClient.Get(url)
	if err != nil {
		// Service is unreachable
		if service.Status != "offline" {
			service.Status = "offline"
			pm.emitEvent(events.Event{
				Type:      "distributed_worker_offline",
				SessionID: "system",
				Message:   fmt.Sprintf("Remote worker %s went offline", workerID),
				Data: map[string]interface{}{
					"worker_id": workerID,
					"error":     err.Error(),
				},
			})
		}
		return
	}
	defer resp.Body.Close()

	// Service is reachable. Preserve the "paired" status — a successful health
	// check must NOT demote a paired worker to "online", otherwise
	// GetPairedServices() (which filters by Status == "paired") would silently
	// drop it after the first health-check tick, losing work distribution.
	wasOffline := service.Status == "offline"
	if service.Status != "paired" {
		service.Status = "online"
	}
	service.LastSeen = time.Now()

	if wasOffline {
		pm.emitEvent(events.Event{
			Type:      "distributed_worker_online",
			SessionID: "system",
			Message:   fmt.Sprintf("Remote worker %s came back online", workerID),
			Data: map[string]interface{}{
				"worker_id": workerID,
			},
		})
	}
}

// emitEvent emits an event if event bus is available
func (pm *PairingManager) emitEvent(event events.Event) {
	if pm.eventBus != nil {
		pm.eventBus.Publish(event)
	}
}

// Close stops the pairing manager
func (pm *PairingManager) Close() {
	pm.cancel()
}
