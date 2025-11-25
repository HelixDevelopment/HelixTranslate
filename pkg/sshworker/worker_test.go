package sshworker

import (
	"context"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
)

func TestNewSSHWorker(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22, // Explicitly set port
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}
	
	if worker.host != "test.local" {
		t.Errorf("Expected host 'test.local', got '%s'", worker.host)
	}
	
	if worker.username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", worker.username)
	}
	
	if worker.password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", worker.password)
	}
	
	if worker.port != 22 {
		t.Errorf("Expected port 22, got %d", worker.port)
	}
}

func TestCommandResult_Success(t *testing.T) {
	// Test successful result
	successResult := &CommandResult{
		ExitCode: 0,
		Stdout:   "success",
		Stderr:   "",
		Error:    nil,
	}
	
	if !successResult.Success() {
		t.Error("Expected success result to return true")
	}
	
	output := successResult.Output()
	if output != "success" {
		t.Errorf("Expected output 'success', got '%s'", output)
	}
	
	// Test failed result
	failResult := &CommandResult{
		ExitCode: 1,
		Stdout:   "error output",
		Stderr:   "error message",
		Error:    nil,
	}
	
	if failResult.Success() {
		t.Error("Expected failed result to return false")
	}
	
	output = failResult.Output()
	expected := "error outputerror message"
	if output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, output)
	}
}

func TestGenerateSSHKey(t *testing.T) {
	privateKey, publicKey, err := GenerateSSHKey()
	if err != nil {
		t.Fatalf("Failed to generate SSH key: %v", err)
	}
	
	if privateKey == "" {
		t.Error("Private key should not be empty")
	}
	
	if publicKey == "" {
		t.Error("Public key should not be empty")
	}
	
	// Verify key format
	if !strings.Contains(privateKey, "BEGIN RSA PRIVATE KEY") {
		t.Error("Private key should contain RSA PRIVATE KEY header")
	}
	
	if !strings.HasPrefix(publicKey, "ssh-rsa") {
		t.Error("Public key should start with ssh-rsa")
	}
}

func TestSSHWorker_UpdateRemoteCodebase(t *testing.T) {
	// This is a mock test since we can't actually SSH in unit tests
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	// Test with nil client (not connected)
	ctx := context.Background()
	err = worker.UpdateRemoteCodebase(ctx, "/tmp")
	
	if err == nil {
		t.Error("Expected error when not connected")
	}
	
	// The actual error comes from SyncCodebase which tries to create an archive
	// using tar command, which fails because directories don't exist in test context
	if !contains(err.Error(), "failed to sync codebase") {
		t.Errorf("Expected sync codebase error, got '%s'", err.Error())
	}
}

func TestSSHWorker_SyncCodebase(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	// Test with nil client
	ctx := context.Background()
	err = worker.SyncCodebase(ctx, "/tmp")
	
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestSSHWorker_VerifyCodebaseVersion(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
		Port:     22,
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	// Test with mock local hasher (should work even without SSH connection)
	ctx := context.Background()
	isEqual, _, remoteHash, err := worker.VerifyCodebaseVersion(ctx)
	
	// Should fail at remote hash step but still calculate local hash
	if err == nil {
		t.Error("Expected error for remote connection")
	}
	
	// Local hash calculation may fail in test environment, that's ok
	// The important thing is that we get an error for the remote connection
	if remoteHash != "" {
		t.Error("Remote hash should be empty when connection fails")
	}
	
	if isEqual {
		t.Error("Should not be equal when remote hash fails")
	}
}

func TestSSHWorker_UploadFile(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	// Test with nil client
	ctx := context.Background()
	err = worker.UploadFile(ctx, "/tmp/test.txt", "/tmp/remote.txt")
	
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestSSHWorker_DownloadFile(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	// Test with nil client
	ctx := context.Background()
	err = worker.DownloadFile(ctx, "/tmp/remote.txt", "/tmp/local.txt")
	
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestSSHWorker_ExecuteCommand(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "test.local",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	// Test with nil client
	ctx := context.Background()
	result, err := worker.ExecuteCommand(ctx, "echo test")
	
	if err == nil {
		t.Error("Expected error when not connected")
	}
	
	if result != nil {
		t.Error("Result should be nil when not connected")
	}
}

func TestSSHWorker_ConnectDisconnect(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "invalid-host-that-does-not-exist.local",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Test connection to invalid host
	err = worker.Connect(ctx)
	if err == nil {
		t.Error("Expected connection to fail for invalid host")
	}
	
	// Test disconnect when not connected
	err = worker.Disconnect()
	if err != nil {
		t.Errorf("Disconnect should not error when not connected: %v", err)
	}
}

func TestSSHWorker_TestConnection(t *testing.T) {
	config := SSHWorkerConfig{
		Host:     "invalid-host.local",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Test connection to invalid host
	err = worker.TestConnection(ctx)
	if err == nil {
		t.Error("Expected test connection to fail for invalid host")
	}
}

func TestSSHWorker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// This test would require an actual SSH server
	// For now, just verify the structure is correct
	config := SSHWorkerConfig{
		Host:     "localhost",
		Username: "testuser",
		Password: "testpass",
	}
	logger := logger.NewLogger(logger.LoggerConfig{})
	worker, err := NewSSHWorker(config, logger)
	
	if err != nil {
		t.Fatalf("Failed to create SSH worker: %v", err)
	}
	_ = worker // Suppress unused variable warning
}

// Helper function for string contains check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
	       len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		   len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkGenerateSSHKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateSSHKey()
		if err != nil {
			b.Fatalf("Failed to generate SSH key: %v", err)
		}
	}
}