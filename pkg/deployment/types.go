package deployment

import (
	"fmt"
	"time"
)

// DeploymentPlan represents a complete deployment plan for the distributed system
type DeploymentPlan struct {
	Main    *DeploymentConfig   `json:"main"`
	Workers []*DeploymentConfig `json:"workers"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	InstanceID   string                 `json:"instance_id"`
	Status       string                 `json:"status"`
	Host         string                 `json:"host"`
	Port         int                    `json:"port"`
	ContainerID  string                 `json:"container_id"`
	LastSeen     time.Time              `json:"last_seen"`
	Health       *HealthStatus          `json:"health,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
}

// HealthStatus represents the health status of an instance
type HealthStatus struct {
	Status    string    `json:"status"`
	LastCheck time.Time `json:"last_check"`
	Response  string    `json:"response,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// NetworkService represents a discovered network service
type NetworkService struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Host         string                 `json:"host"`
	Port         int                    `json:"port"`
	Type         string                 `json:"type"`
	Protocol     string                 `json:"protocol"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	LastSeen     time.Time              `json:"last_seen"`
	TTL          time.Duration          `json:"ttl"`
}

// BroadcastMessage represents a service broadcast message
type BroadcastMessage struct {
	ServiceID    string                 `json:"service_id"`
	Type         string                 `json:"type"`
	Host         string                 `json:"host"`
	Port         int                    `json:"port"`
	Protocol     string                 `json:"protocol"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// APICommunicationLog represents a log entry for API communication
type APICommunicationLog struct {
	Timestamp    time.Time     `json:"timestamp"`
	SourceHost   string        `json:"source_host"`
	SourcePort   int           `json:"source_port"`
	TargetHost   string        `json:"target_host"`
	TargetPort   int           `json:"target_port"`
	Method       string        `json:"method"`
	URL          string        `json:"url"`
	StatusCode   int           `json:"status_code"`
	RequestSize  int64         `json:"request_size"`
	ResponseSize int64         `json:"response_size"`
	Duration     time.Duration `json:"duration"`
	UserAgent    string        `json:"user_agent,omitempty"`
	Error        string        `json:"error,omitempty"`
}
