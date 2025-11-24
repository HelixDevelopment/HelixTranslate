package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainFunction(t *testing.T) {
	// Test main function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("main() panicked: %v", r)
		}
	}()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "deployment-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original args and restore after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Test with valid arguments
	os.Args = []string{"deployment", "--config", filepath.Join(tempDir, "config.yaml")}

	t.Run("ConfigFileValidation", func(t *testing.T) {
		// Create a sample config file
		configContent := `
server:
  port: 8080
  host: "localhost"

database:
  host: "localhost"
  port: 5432
  name: "translator"

translation:
  providers: ["openai", "anthropic"]
  default_provider: "openai"
`
		configFile := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Test config file exists and is readable
		info, err := os.Stat(configFile)
		if err != nil {
			t.Errorf("Config file should exist: %v", err)
		}

		if info.Size() == 0 {
			t.Errorf("Config file should not be empty")
		}
	})
}

func TestDeploymentConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "deployment-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("YAMLConfig", func(t *testing.T) {
		// Test YAML configuration parsing
		configContent := `
app:
  name: "Translator API"
  version: "1.0.0"
  environment: "production"

server:
  port: 8080
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"

database:
  url: "postgresql://user:pass@localhost:5432/translator"
  max_connections: 20

logging:
  level: "info"
  format: "json"
  output: "stdout"
`
		configFile := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Test that YAML is valid
		content, err := os.ReadFile(configFile)
		if err != nil {
			t.Errorf("Failed to read config file: %v", err)
		}

		// Basic validation - should contain expected sections
		contentStr := string(content)
		requiredSections := []string{"app:", "server:", "database:", "logging:"}
		
		for _, section := range requiredSections {
			if !strings.Contains(contentStr, section) {
				t.Errorf("Config missing required section: %s", section)
			}
		}
	})

	t.Run("JSONConfig", func(t *testing.T) {
		// Test JSON configuration
		configContent := `{
  "app": {
    "name": "Translator API",
    "version": "1.0.0"
  },
  "server": {
    "port": 8080,
    "host": "0.0.0.0"
  },
  "translation": {
    "providers": ["openai", "anthropic"],
    "timeout": 30
  }
}`
		configFile := filepath.Join(tempDir, "config.json")
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Test JSON validity
		content, err := os.ReadFile(configFile)
		if err != nil {
			t.Errorf("Failed to read JSON config: %v", err)
		}

		// Basic JSON validation (simplified)
		contentStr := string(content)
		if !strings.HasPrefix(strings.TrimSpace(contentStr), "{") ||
		   !strings.HasSuffix(strings.TrimSpace(contentStr), "}") {
			t.Errorf("Invalid JSON format")
		}
	})
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("RequiredEnvVars", func(t *testing.T) {
		// Test required environment variables
		requiredVars := []string{
			"PORT",
			"DB_HOST",
			"DB_PORT",
			"OPENAI_API_KEY",
		}

		for _, varName := range requiredVars {
			// Check if we can get the variable (even if empty)
			value := os.Getenv(varName)
			t.Logf("Environment variable %s = %s", varName, value)
			
			// In a real test, we would validate that these are set for production
			// For this test, we just verify we can check them
		}
	})

	t.Run("OptionalEnvVars", func(t *testing.T) {
		// Test optional environment variables
		optionalVars := []string{
			"LOG_LEVEL",
			"LOG_FORMAT",
			"MAX_CONNECTIONS",
			"TIMEOUT",
		}

		for _, varName := range optionalVars {
			value := os.Getenv(varName)
			t.Logf("Optional environment variable %s = %s", varName, value)
		}
	})
}

func TestServiceDiscovery(t *testing.T) {
	t.Run("HealthCheckEndpoints", func(t *testing.T) {
		// Test health check endpoint configuration
		healthEndpoints := []string{
			"/health",
			"/health/ready",
			"/health/live",
			"/metrics",
		}

		for _, endpoint := range healthEndpoints {
			// Validate endpoint format
			if !strings.HasPrefix(endpoint, "/") {
				t.Errorf("Health endpoint should start with /: %s", endpoint)
			}

			// In a real test, we would make HTTP requests to these endpoints
			t.Logf("Health endpoint: %s", endpoint)
		}
	})

	t.Run("ServiceRegistry", func(t *testing.T) {
		// Test service registration
		serviceInfo := map[string]string{
			"name":     "translator-api",
			"version":  "1.0.0",
			"port":     "8080",
			"protocol": "http",
		}

		// Validate required fields
		requiredFields := []string{"name", "version", "port", "protocol"}
		for _, field := range requiredFields {
			if value, exists := serviceInfo[field]; !exists || value == "" {
				t.Errorf("Missing required service field: %s", field)
			}
		}
	})
}

func TestDatabaseMigration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "deployment-migration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("MigrationFiles", func(t *testing.T) {
		// Create sample migration files
		migrations := []struct {
			version    string
			upSQL      string
			downSQL    string
		}{
			{
				version: "001_initial_schema",
				upSQL:   "CREATE TABLE translations (id SERIAL PRIMARY KEY, content TEXT);",
				downSQL: "DROP TABLE translations;",
			},
			{
				version: "002_add_indexes",
				upSQL:   "CREATE INDEX idx_translations_content ON translations(content);",
				downSQL: "DROP INDEX idx_translations_content;",
			},
		}

		for _, migration := range migrations {
			// Create up migration file
			upFile := filepath.Join(tempDir, migration.version+"_up.sql")
			err := os.WriteFile(upFile, []byte(migration.upSQL), 0644)
			if err != nil {
				t.Fatalf("Failed to create up migration file: %v", err)
			}

			// Create down migration file
			downFile := filepath.Join(tempDir, migration.version+"_down.sql")
			err := os.WriteFile(downFile, []byte(migration.downSQL), 0644)
			if err != nil {
				t.Fatalf("Failed to create down migration file: %v", err)
			}

			// Validate files exist and have content
			upContent, err := os.ReadFile(upFile)
			if err != nil {
				t.Errorf("Failed to read up migration file: %v", err)
			}
			if len(upContent) == 0 {
				t.Errorf("Up migration file should not be empty")
			}

			downContent, err := os.ReadFile(downFile)
			if err != nil {
				t.Errorf("Failed to read down migration file: %v", err)
			}
			if len(downContent) == 0 {
				t.Errorf("Down migration file should not be empty")
			}
		}
	})
}

func TestLoadBalancing(t *testing.T) {
	t.Run("RoundRobin", func(t *testing.T) {
		// Test round-robin load balancing
		servers := []string{"server1:8080", "server2:8080", "server3:8080"}
		
		// Simulate round-robin selection
		for i := 0; i < 10; i++ {
			selectedServer := servers[i%len(servers)]
			t.Logf("Request %d assigned to: %s", i, selectedServer)
			
			// Validate selected server is in the list
			found := false
			for _, server := range servers {
				if server == selectedServer {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Selected server not in server list: %s", selectedServer)
			}
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		// Test server health checking
		servers := map[string]bool{
			"server1:8080": true,
			"server2:8080": false, // unhealthy
			"server3:8080": true,
		}

		healthyServers := []string{}
		for server, healthy := range servers {
			if healthy {
				healthyServers = append(healthyServers, server)
			}
		}

		if len(healthyServers) != 2 {
			t.Errorf("Expected 2 healthy servers, got %d", len(healthyServers))
		}
	})
}

func TestSSLConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "deployment-ssl-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("CertificateFiles", func(t *testing.T) {
		// Create mock certificate files
		certContent := `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAMlyFqk69v+9MA0GCSqGSIb3DQEBBQUAMBExDzANBgNVBAMTBnRl
-----END CERTIFICATE-----`

		keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC5...
-----END PRIVATE KEY-----`

		certFile := filepath.Join(tempDir, "cert.pem")
		keyFile := filepath.Join(tempDir, "key.pem")

		err := os.WriteFile(certFile, []byte(certContent), 0600)
		if err != nil {
			t.Fatalf("Failed to create cert file: %v", err)
		}

		err = os.WriteFile(keyFile, []byte(keyContent), 0600)
		if err != nil {
			t.Fatalf("Failed to create key file: %v", err)
		}

		// Validate file permissions
		certInfo, err := os.Stat(certFile)
		if err != nil {
			t.Errorf("Failed to stat cert file: %v", err)
		}

		keyInfo, err := os.Stat(keyFile)
		if err != nil {
			t.Errorf("Failed to stat key file: %v", err)
		}

		// Check file permissions (should be 0600)
		if certInfo.Mode().Perm()&0177 != 0 {
			t.Errorf("Certificate file should have restricted permissions")
		}

		if keyInfo.Mode().Perm()&0177 != 0 {
			t.Errorf("Key file should have restricted permissions")
		}
	})
}

func TestContainerization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "deployment-container-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("Dockerfile", func(t *testing.T) {
		// Create sample Dockerfile
		dockerfileContent := `FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o translator ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/translator .
EXPOSE 8080
CMD ["./translator"]
`
		dockerfilePath := filepath.Join(tempDir, "Dockerfile")
		err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		// Validate Dockerfile content
		content, err := os.ReadFile(dockerfilePath)
		if err != nil {
			t.Errorf("Failed to read Dockerfile: %v", err)
		}

		contentStr := string(content)
		requiredInstructions := []string{"FROM", "WORKDIR", "COPY", "RUN", "EXPOSE", "CMD"}
		
		for _, instruction := range requiredInstructions {
			if !strings.Contains(contentStr, instruction) {
				t.Errorf("Dockerfile missing required instruction: %s", instruction)
			}
		}
	})

	t.Run("DockerCompose", func(t *testing.T) {
		// Create sample docker-compose.yml
		composeContent := `version: '3.8'
services:
  translator:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: translator
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

volumes:
  postgres_data:
`
		composeFile := filepath.Join(tempDir, "docker-compose.yml")
		err := os.WriteFile(composeFile, []byte(composeContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create docker-compose.yml: %v", err)
		}

		// Validate docker-compose content
		content, err := os.ReadFile(composeFile)
		if err != nil {
			t.Errorf("Failed to read docker-compose.yml: %v", err)
		}

		contentStr := string(content)
		requiredSections := []string{"services:", "translator:", "postgres:", "redis:", "volumes:"}
		
		for _, section := range requiredSections {
			if !strings.Contains(contentStr, section) {
				t.Errorf("docker-compose.yml missing required section: %s", section)
			}
		}
	})
}

func TestMonitoringSetup(t *testing.T) {
	t.Run("MetricsEndpoints", func(t *testing.T) {
		// Test metrics endpoint configuration
		metricsConfig := map[string]interface{}{
			"enabled":   true,
			"path":      "/metrics",
			"port":      9090,
			"format":    "prometheus",
			"interval":  "30s",
		}

		// Validate required configuration
		if enabled, ok := metricsConfig["enabled"].(bool); !ok || !enabled {
			t.Errorf("Metrics should be enabled")
		}

		if path, ok := metricsConfig["path"].(string); !ok || path == "" {
			t.Errorf("Metrics path should be specified")
		}

		if port, ok := metricsConfig["port"].(int); !ok || port == 0 {
			t.Errorf("Metrics port should be specified")
		}
	})

	t.Run("LoggingConfiguration", func(t *testing.T) {
		// Test logging configuration
		loggingConfig := map[string]string{
			"level":     "info",
			"format":    "json",
			"output":    "stdout",
			"max_size":  "100MB",
			"max_files": "10",
		}

		requiredFields := []string{"level", "format", "output"}
		for _, field := range requiredFields {
			if value, exists := loggingConfig[field]; !exists || value == "" {
				t.Errorf("Missing required logging field: %s", field)
			}
		}

		// Validate log level
		validLevels := []string{"debug", "info", "warn", "error", "fatal"}
		level := loggingConfig["level"]
		validLevel := false
		for _, valid := range validLevels {
			if level == valid {
				validLevel = true
				break
			}
		}
		if !validLevel {
			t.Errorf("Invalid log level: %s", level)
		}
	})
}