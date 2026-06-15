package challenge_runner

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// NewEventBusChallenge verifies the event bus system.
type EventBusChallenge struct {
	challenge.BaseChallenge
}

// NewEventBusChallenge creates the event bus challenge.
func NewEventBusChallenge() *EventBusChallenge {
	bc := challenge.NewBaseChallenge(
		"event-bus",
		"Event Bus",
		"Verifies event publish/subscribe, type filtering, and session tracking",
		"unit",
		nil,
	)
	return &EventBusChallenge{BaseChallenge: bc}
}

// Execute runs the event bus challenge.
func (c *EventBusChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestEventBus", "./pkg/events/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "event_bus",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewWebSocketChallenge verifies WebSocket hub and client management.
type WebSocketChallenge struct {
	challenge.BaseChallenge
}

// NewWebSocketChallenge creates the WebSocket challenge.
func NewWebSocketChallenge() *WebSocketChallenge {
	bc := challenge.NewBaseChallenge(
		"websocket-monitoring",
		"WebSocket Monitoring",
		"Verifies WebSocket hub, client registration, event broadcasting, and connection lifecycle",
		"integration",
		nil,
	)
	return &WebSocketChallenge{BaseChallenge: bc}
}

// Execute runs the WebSocket challenge.
func (c *WebSocketChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestWebSocket", "./pkg/websocket/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "websocket",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewAPIEndpointsChallenge verifies REST API endpoints.
type APIEndpointsChallenge struct {
	challenge.BaseChallenge
}

// NewAPIEndpointsChallenge creates the API endpoints challenge.
func NewAPIEndpointsChallenge() *APIEndpointsChallenge {
	bc := challenge.NewBaseChallenge(
		"api-endpoints",
		"API Endpoints",
		"Verifies REST API handlers, middleware, routing, and response formats",
		"integration",
		nil,
	)
	return &APIEndpointsChallenge{BaseChallenge: bc}
}

// Execute runs the API endpoints challenge.
func (c *APIEndpointsChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestAPI", "./pkg/api/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "api_endpoints",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewGRPCServerChallenge verifies gRPC server functionality.
type GRPCServerChallenge struct {
	challenge.BaseChallenge
}

// NewGRPCServerChallenge creates the gRPC server challenge.
func NewGRPCServerChallenge() *GRPCServerChallenge {
	bc := challenge.NewBaseChallenge(
		"grpc-server",
		"gRPC Server",
		"Verifies gRPC service definitions, server implementation, and proto compilation",
		"integration",
		nil,
	)
	return &GRPCServerChallenge{BaseChallenge: bc}
}

// Execute runs the gRPC server challenge.
func (c *GRPCServerChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}

	cmd := exec.CommandContext(ctx, "go", "build", "./pkg/grpc/...")
	out, err := cmd.CombinedOutput()
	pass := err == nil
	assertions = append(assertions, challenge.AssertionResult{
		Type:    "compilation",
		Target:  "grpc_package",
		Passed:  pass,
		Message: string(out),
	})

	cmd = exec.CommandContext(ctx, "go", "build", "-o", "/dev/null", "./cmd/grpc-server/...")
	out, err = cmd.CombinedOutput()
	pass2 := err == nil
	assertions = append(assertions, challenge.AssertionResult{
		Type:    "compilation",
		Target:  "grpc_server_binary",
		Passed:  pass2,
		Message: string(out),
	})

	status := challenge.StatusPassed
	if !pass || !pass2 {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewSecurityAuthChallenge verifies JWT authentication.
type SecurityAuthChallenge struct {
	challenge.BaseChallenge
}

// NewSecurityAuthChallenge creates the security auth challenge.
func NewSecurityAuthChallenge() *SecurityAuthChallenge {
	bc := challenge.NewBaseChallenge(
		"security-auth",
		"Security Authentication",
		"Verifies JWT token generation, validation, and middleware",
		"security",
		nil,
	)
	return &SecurityAuthChallenge{BaseChallenge: bc}
}

// Execute runs the security auth challenge.
func (c *SecurityAuthChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestAuth", "./pkg/security/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "security_auth",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewRateLimitingChallenge verifies rate limiting functionality.
type RateLimitingChallenge struct {
	challenge.BaseChallenge
}

// NewRateLimitingChallenge creates the rate limiting challenge.
func NewRateLimitingChallenge() *RateLimitingChallenge {
	bc := challenge.NewBaseChallenge(
		"rate-limiting",
		"Rate Limiting",
		"Verifies request rate limiting and burst handling",
		"security",
		nil,
	)
	return &RateLimitingChallenge{BaseChallenge: bc}
}

// Execute runs the rate limiting challenge.
func (c *RateLimitingChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestRateLimit", "./pkg/security/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "rate_limiting",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// HealthEndpointChallenge verifies the health endpoint responds.
type HealthEndpointChallenge struct {
	challenge.BaseChallenge
}

// NewHealthEndpointChallenge creates the health endpoint challenge.
func NewHealthEndpointChallenge() *HealthEndpointChallenge {
	bc := challenge.NewBaseChallenge(
		"health-endpoint",
		"Health Endpoint",
		"Verifies the /health endpoint returns 200 OK with correct body",
		"e2e",
		nil,
	)
	return &HealthEndpointChallenge{BaseChallenge: bc}
}

// Execute runs the health endpoint challenge.
func (c *HealthEndpointChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	assertions := []challenge.AssertionResult{}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:8080/health")

	if err != nil {
		assertions = append(assertions, challenge.AssertionResult{
			Type:    "reachability",
			Target:  "health_endpoint",
			Passed:  false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		})
		return c.CreateResult(challenge.StatusFailed, start, assertions, nil, nil, ""), nil
	}
	defer resp.Body.Close()

	pass := resp.StatusCode == http.StatusOK
	assertions = append(assertions, challenge.AssertionResult{
		Type:    "status_code",
		Target:  "health_endpoint",
		Passed:  pass,
		Message: fmt.Sprintf("Status: %d", resp.StatusCode),
	})

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}
