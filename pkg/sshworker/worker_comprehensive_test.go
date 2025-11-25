package sshworker

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/version"
)

// Helper function for string contains check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestSSHWorkerConfig tests the SSHWorkerConfig structure
func TestSSHWorkerConfig(t *testing.T) {
	config := SSHWorkerConfig{
		Host:              "example.com",
		Username:          "testuser",
		Password:          "testpass",
		PrivateKeyPath:    "/path/to/key",
		Port:              2222,
		RemoteDir:         "/remote/dir",
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    10 * time.Second,
	}

	if config.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", config.Host)
	}

	if config.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", config.Username)
	}

	if config.Port != 2222 {
		t.Errorf("Expected port 2222, got %d", config.Port)
	}

	if config.RemoteDir != "/remote/dir" {
		t.Errorf("Expected remote dir '/remote/dir', got '%s'", config.RemoteDir)
	}
}

// TestSSHWorker_Structure tests the SSHWorker structure initialization
func TestSSHWorker_Structure(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.example.com",
		Username: "worker",
		Password: "secret",
		Port:     22,
		RemoteDir: "/app",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})

	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	// Test internal fields
	if worker.config.Host != "test.example.com" {
		t.Errorf("Expected config host 'test.example.com', got '%s'", worker.config.Host)
	}

	if worker.config.RemoteDir != "/app" {
		t.Errorf("Expected config remote dir '/app', got '%s'", worker.config.RemoteDir)
	}

	// Test initial state
	if worker.client != nil {
		t.Error("Client should be nil initially")
	}
}

// TestSSHWorker_Connect_InvalidPort tests connection with invalid port
func TestSSHWorker_Connect_InvalidPort(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "example.com",
		Username: "testuser",
		Password: "testpass",
		Port:     0, // Invalid port
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx := context.Background()
	err = worker.Connect(ctx)
	if err == nil {
		t.Error("Expected error for invalid port")
	}

	expectedError := "invalid port number: 0 (must be between 1 and 65535)"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestSSHWorker_Connect_NoAuth tests connection with no authentication method
func TestSSHWorker_Connect_NoAuth(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "example.com",
		Username: "testuser",
		Port:     22,
		// No password or private key
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx := context.Background()
	err = worker.Connect(ctx)
	if err == nil {
		t.Error("Expected error for no authentication method")
	}

	expectedError := "no authentication method available"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestSSHWorker_UploadData tests the UploadData method
func TestSSHWorker_UploadData(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
		RemoteDir: "/tmp",
		ConnectionTimeout: 1 * time.Second, // Short timeout for faster test
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx := context.Background()
	testData := []byte("test data content")
	
	err = worker.UploadData(ctx, testData, "/tmp/test.txt")
	if err == nil {
		t.Error("Expected error when not connected")
	}

	// Should fail at connection establishment due to unresolvable hostname
	if !contains(err.Error(), "failed to connect") && !contains(err.Error(), "no such host") {
		t.Errorf("Expected connection error, got '%s'", err.Error())
	}
}

// TestSSHWorker_GetRemoteCodebaseHash tests the GetRemoteCodebaseHash method
func TestSSHWorker_GetRemoteCodebaseHash(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
		RemoteDir: "/tmp",
		ConnectionTimeout: 1 * time.Second, // Short timeout for faster test
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx := context.Background()
	hash, err := worker.GetRemoteCodebaseHash(ctx)
	
	if err == nil {
		t.Error("Expected error when not connected")
	}
	
	if hash != "" {
		t.Error("Hash should be empty when connection fails")
	}

	// Should fail at connection establishment due to unresolvable hostname
	if !contains(err.Error(), "failed to connect") && !contains(err.Error(), "no such host") {
		t.Errorf("Expected connection error, got '%s'", err.Error())
	}
}

// TestSSHWorker_UploadEssentialFiles tests the UploadEssentialFiles method
func TestSSHWorker_UploadEssentialFiles(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
		RemoteDir: "/tmp",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx := context.Background()
	err = worker.UploadEssentialFiles(ctx)
	
	if err == nil {
		t.Error("Expected error when not connected")
	}

	// Should fail at the first command execution
	expectedError := "failed to setup remote directory: SSH client not connected"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

// TestCommandResult_ErrorHandling tests CommandResult error handling
func TestCommandResult_ErrorHandling(t *testing.T) {
	// Test with actual error
	testError := &CommandResult{
		ExitCode: 1,
		Stdout:   "output",
		Stderr:   "error",
		Error:    &testCustomError{msg: "custom error"},
	}

	if testError.Success() {
		t.Error("Expected false for failed result with error")
	}

	output := testError.Output()
	expected := "outputerror"
	if output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}
}

// Custom error type for testing
type testCustomError struct {
	msg string
}

func (e *testCustomError) Error() string {
	return e.msg
}

// TestSSHWorker_Close tests the Close method
func TestSSHWorker_Close(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	// Close when client is nil should not error
	err = worker.Close()
	if err != nil {
		t.Errorf("Close should not error when client is nil: %v", err)
	}
}

// TestCodebaseHasher_Integration tests the codebase hasher integration
func TestCodebaseHasher_Integration(t *testing.T) {
	// Test that the version package can create a hasher
	hasher := version.NewCodebaseHasher()
	if hasher == nil {
		t.Error("Failed to create codebase hasher")
	}

	// Test hash calculation in the correct directory context
	// Change to the project root for testing
	originalDir, _ := os.Getwd()
	defer func() {
		os.Chdir(originalDir)
	}()

	// Try to change to project root (may already be there)
	if _, err := os.Stat("go.mod"); err == nil {
		// We're at project root, proceed with hash calculation
		hash, err := hasher.CalculateHash()
		if err != nil {
			t.Logf("Hash calculation failed (expected in test): %v", err)
			// Don't fail the test as this depends on the environment
			return
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// Test version comparison
		same := hasher.CompareVersions(hash, hash)
		if !same {
			t.Error("Hash should be equal to itself")
		}

		different := hasher.CompareVersions(hash, "different_hash")
		if different {
			t.Error("Hash should not be equal to different hash")
		}
	} else {
		t.Skip("Skipping hash test - not in project root")
	}
}

// TestEnvironmentVariables tests SSH worker behavior with environment variables
func TestEnvironmentVariables(t *testing.T) {
	// Save original value
	originalKeyPath := os.Getenv("SSH_PRIVATE_KEY_PATH")
	defer func() {
		if originalKeyPath != "" {
			os.Setenv("SSH_PRIVATE_KEY_PATH", originalKeyPath)
		} else {
			os.Unsetenv("SSH_PRIVATE_KEY_PATH")
		}
	}()

	// Test with non-existent key path
	os.Setenv("SSH_PRIVATE_KEY_PATH", "/non/existent/key/path")
	
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx := context.Background()
	err = worker.Connect(ctx)
	// Should fallback to password auth
	if err != nil && err.Error() == "no authentication method available" {
		t.Error("Should fallback to password auth when key file doesn't exist")
	}
}

// TestSSHWorker_Timeouts tests connection timeout handling
func TestSSHWorker_Timeouts(t *testing.T) {
	config := SSHWorkerConfig{
		Host:              "10.255.255.1", // Non-routable IP
		Username:          "testuser",
		Password:          "testpass",
		Port:              22,
		ConnectionTimeout: 1 * time.Second, // Very short timeout
		CommandTimeout:    1 * time.Second,
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err = worker.Connect(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected connection to fail to non-routable IP")
	}

	if elapsed > 5*time.Second {
		t.Errorf("Connection took too long: %v", elapsed)
	}

	// Test should timeout quickly due to non-routable IP
	if elapsed > 3*time.Second {
		t.Logf("Warning: Connection took longer than expected: %v", elapsed)
	}
}

// Benchmark tests for performance monitoring
func BenchmarkNewSSHWorker(b *testing.B) {
	config := SSHWorkerConfig{
		Host:     "benchmark.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
	}
	logger := logger.NewLogger(logger.LoggerConfig{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewSSHWorker(config, logger)
		if err != nil {
			b.Fatalf("Failed to create SSH worker: %v", err)
		}
	}
}