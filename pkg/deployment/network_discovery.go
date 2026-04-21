package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"digital.vasic.translator/internal/config"
)

// NetworkDiscoverer handles network discovery and service broadcasting
type NetworkDiscoverer struct {
	config        *config.Config
	logger        *log.Logger
	broadcastAddr string
	discoveryPort int
	services      map[string]*NetworkService
	broadcastConn *net.UDPConn
	discoveryConn *net.UDPConn
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewNetworkDiscoverer creates a new network discoverer
func NewNetworkDiscoverer(cfg *config.Config, logger *log.Logger) *NetworkDiscoverer {
	ctx, cancel := context.WithCancel(context.Background())

	return &NetworkDiscoverer{
		config:        cfg,
		logger:        logger,
		broadcastAddr: "255.255.255.255",
		discoveryPort: 9999,
		services:      make(map[string]*NetworkService),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// StartBroadcasting starts broadcasting service configurations
func (nd *NetworkDiscoverer) StartBroadcasting(ctx context.Context, instances map[string]*DeployedInstance) error {
	nd.logger.Println("Starting service broadcasting...")

	// Create UDP broadcast connection
	conn, err := nd.createBroadcastConnection()
	if err != nil {
		return fmt.Errorf("failed to create broadcast connection: %w", err)
	}
	nd.broadcastConn = conn

	nd.wg.Add(1)
	go nd.broadcastLoop(ctx, instances)

	nd.logger.Println("Service broadcasting started successfully")
	return nil
}

// StartDiscovery starts listening for service discovery broadcasts
func (nd *NetworkDiscoverer) StartDiscovery(ctx context.Context) error {
	nd.logger.Println("Starting service discovery...")

	// Create UDP discovery connection
	conn, err := nd.createDiscoveryConnection()
	if err != nil {
		return fmt.Errorf("failed to create discovery connection: %w", err)
	}
	nd.discoveryConn = conn

	nd.wg.Add(1)
	go nd.discoveryLoop(ctx)

	nd.logger.Println("Service discovery started successfully")
	return nil
}

// createBroadcastConnection creates a UDP connection for broadcasting
func (nd *NetworkDiscoverer) createBroadcastConnection() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", nd.broadcastAddr, nd.discoveryPort))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// createDiscoveryConnection creates a UDP connection for discovery listening
func (nd *NetworkDiscoverer) createDiscoveryConnection() (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", nd.discoveryPort))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// broadcastLoop continuously broadcasts service information
func (nd *NetworkDiscoverer) broadcastLoop(ctx context.Context, instances map[string]*DeployedInstance) {
	defer nd.wg.Done()
	ticker := time.NewTicker(30 * time.Second) // Broadcast every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-nd.ctx.Done():
			return
		case <-ticker.C:
			nd.broadcastServices(instances)
		}
	}
}

// broadcastServices broadcasts all known services
func (nd *NetworkDiscoverer) broadcastServices(instances map[string]*DeployedInstance) {
	for _, instance := range instances {
		message := BroadcastMessage{
			ServiceID: instance.ID,
			Type:      nd.determineServiceType(instance.ID),
			Host:      instance.Host,
			Port:      instance.Port,
			Protocol:  "https",
			Capabilities: map[string]interface{}{
				"container_id": instance.ContainerID,
				"status":       instance.Status,
			},
			Timestamp: time.Now(),
		}

		if err := nd.sendBroadcastMessage(message); err != nil {
			nd.logger.Printf("Failed to broadcast service %s: %v", instance.ID, err)
		}
	}
}

// sendBroadcastMessage sends a single broadcast message
func (nd *NetworkDiscoverer) sendBroadcastMessage(message BroadcastMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = nd.broadcastConn.Write(data)
	return err
}

// discoveryLoop listens for incoming discovery broadcasts
func (nd *NetworkDiscoverer) discoveryLoop(ctx context.Context) {
	defer nd.wg.Done()

	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		case <-nd.ctx.Done():
			return
		default:
			if err := nd.discoveryConn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				nd.logger.Printf("Failed to set read deadline: %v", err)
				continue
			}

			n, addr, err := nd.discoveryConn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout is expected, continue listening
				}
				nd.logger.Printf("Failed to read UDP message: %v", err)
				continue
			}

			if err := nd.handleDiscoveryMessage(buffer[:n], addr); err != nil {
				nd.logger.Printf("Failed to handle discovery message: %v", err)
			}
		}
	}
}

// handleDiscoveryMessage processes an incoming discovery message
func (nd *NetworkDiscoverer) handleDiscoveryMessage(data []byte, addr *net.UDPAddr) error {
	var message BroadcastMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("failed to unmarshal broadcast message: %w", err)
	}

	// Skip our own broadcasts
	if nd.isOwnService(message) {
		return nil
	}

	service := &NetworkService{
		ID:           message.ServiceID,
		Name:         message.ServiceID,
		Host:         message.Host,
		Port:         message.Port,
		Type:         message.Type,
		Protocol:     message.Protocol,
		Capabilities: message.Capabilities,
		LastSeen:     time.Now(),
		TTL:          90 * time.Second, // 90 seconds TTL
	}

	nd.mu.Lock()
	nd.services[message.ServiceID] = service
	nd.mu.Unlock()

	nd.logger.Printf("Discovered service: %s (%s:%d)", message.ServiceID, message.Host, message.Port)

	return nil
}

// isOwnService checks if a discovered service belongs to this host by comparing hostnames/IPs.
func (nd *NetworkDiscoverer) isOwnService(message BroadcastMessage) bool {
	hostname, err := os.Hostname()
	if err == nil && message.Host == hostname {
		return true
	}

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if message.Host == ipNet.IP.String() {
					return true
				}
			}
		}
	}

	return false
}

// determineServiceType determines the service type from instance ID
func (nd *NetworkDiscoverer) determineServiceType(instanceID string) string {
	if contains(instanceID, "main") || contains(instanceID, "coordinator") {
		return "coordinator"
	}
	if contains(instanceID, "worker") {
		return "worker"
	}
	return "unknown"
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetDiscoveredServices returns all discovered services
func (nd *NetworkDiscoverer) GetDiscoveredServices() map[string]*NetworkService {
	nd.mu.RLock()
	defer nd.mu.RUnlock()

	// Clean up expired services
	now := time.Now()
	for id, service := range nd.services {
		if now.Sub(service.LastSeen) > service.TTL {
			delete(nd.services, id)
		}
	}

	result := make(map[string]*NetworkService)
	for k, v := range nd.services {
		result[k] = v
	}
	return result
}

// QueryService queries a specific service for its capabilities
func (nd *NetworkDiscoverer) QueryService(ctx context.Context, service *NetworkService) (map[string]interface{}, error) {
	// This would implement HTTP queries to discovered services
	// For now, return the cached capabilities
	return service.Capabilities, nil
}

// Close shuts down the network discoverer
func (nd *NetworkDiscoverer) Close() error {
	nd.logger.Println("Shutting down network discoverer...")

	nd.cancel()

	if nd.broadcastConn != nil {
		nd.broadcastConn.Close()
	}

	if nd.discoveryConn != nil {
		nd.discoveryConn.Close()
	}

	nd.wg.Wait()
	return nil
}
