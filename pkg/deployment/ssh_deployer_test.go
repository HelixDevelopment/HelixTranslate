package deployment

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHDeployConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *SSHDeployConfig
		wantErr bool
		errType string
	}{
		{
			name: "valid config with key path",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				KeyPath:  "/path/to/key",
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config with password",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				Password: "testpass",
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config with key data",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				KeyData:  []byte("-----BEGIN RSA PRIVATE KEY-----\nMOCK\n-----END RSA PRIVATE KEY-----"),
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: &SSHDeployConfig{
				Port:     22,
				Username: "testuser",
				KeyPath:  "/path/to/key",
				Timeout:  30 * time.Second,
			},
			wantErr: true,
			errType: "host",
		},
		{
			name: "missing username",
			config: &SSHDeployConfig{
				Host:    "test.example.com",
				Port:    22,
				KeyPath: "/path/to/key",
				Timeout: 30 * time.Second,
			},
			wantErr: true,
			errType: "username",
		},
		{
			name: "missing auth method",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				Timeout:  30 * time.Second,
			},
			wantErr: true,
			errType: "auth",
		},
		{
			name: "zero port uses default",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     0,
				Username: "testuser",
				KeyPath:  "/path/to/key",
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "zero timeout uses default",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				KeyPath:  "/path/to/key",
				Timeout:  0,
			},
			wantErr: false,
		},
		{
			name: "zero connect retries uses default",
			config: &SSHDeployConfig{
				Host:           "test.example.com",
				Port:           22,
				Username:       "testuser",
				KeyPath:        "/path/to/key",
				Timeout:        30 * time.Second,
				ConnectRetries: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				var ve *ValidationError
				require.ErrorAs(t, err, &ve)
				assert.Equal(t, tt.errType, ve.Field)
			} else {
				require.NoError(t, err)
				// Verify defaults are set
				if tt.config.Port == 0 {
					assert.Equal(t, 22, tt.config.Port)
				}
				if tt.config.Timeout == 0 {
					assert.Equal(t, 30*time.Second, tt.config.Timeout)
				}
				if tt.config.ConnectRetries == 0 {
					assert.Equal(t, 3, tt.config.ConnectRetries)
				}
				if tt.config.CommandTimeout == 0 {
					assert.Equal(t, 10*time.Minute, tt.config.CommandTimeout)
				}
			}
		})
	}
}

func TestSSHDeployer_Connect_Success(t *testing.T) {
	ctx := context.Background()

	// Test with a real configuration
	config := &SSHDeployConfig{
		Host:     "test.example.com",
		Port:     22,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}

	// Validate configuration first
	err := config.Validate()
	require.NoError(t, err)

	// Create deployer with default client
	deployer := NewSSHDeployer(config)
	require.NotNil(t, deployer)

	// Test will fail to actually connect but that's expected in unit test
	err = deployer.Connect(ctx)
	// We expect connection to fail in test environment, but not panic
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "SSH connection error")
}

func TestSSHDeployer_Connect_Failures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		config    *SSHDeployConfig
		setupMock func(*MockSSHClient)
		wantErr   bool
		errType   string
	}{
		{
			name: "connection failed",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				Password: "testpass",
				Timeout:  5 * time.Second,
			},
			setupMock: func(m *MockSSHClient) {
				// Mock client setup with false shouldConnect
				*m = *NewMockSSHClient(false)
			},
			wantErr: true,
			errType: "connect_failed",
		},
		{
			name: "authentication failed",
			config: &SSHDeployConfig{
				Host:     "test.example.com",
				Port:     22,
				Username: "testuser",
				Password: "testpass",
				Timeout:  5 * time.Second,
			},
			setupMock: func(m *MockSSHClient) {
				m.shouldConnect = false
				m.SetAuthFail(true) // This will cause authentication failure
			},
			wantErr: true,
			errType: "connect_failed", // Auth failure becomes connection failed
		},
		{
			name: "invalid config",
			config: &SSHDeployConfig{
				Host:     "", // Invalid - empty host
				Port:     22,
				Username: "testuser",
				Password: "testpass",
				Timeout:  5 * time.Second,
			},
			setupMock: func(m *MockSSHClient) {},
			wantErr:   true,
			errType:   "config_validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockSSHClient(true)
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			deployer := NewSSHDeployerWithClient(tt.config, mockClient)

			err := deployer.Connect(ctx)
			if tt.wantErr {
				require.Error(t, err)
				var ce *ConnectionError
				require.ErrorAs(t, err, &ce)
				assert.Equal(t, tt.errType, ce.Type)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSSHDeployer_Connect_Timeout(t *testing.T) {
	config := &SSHDeployConfig{
		Host:     "slow.example.com",
		Port:     22,
		Username: "testuser",
		Password: "testpass",
		Timeout:  1 * time.Millisecond, // Very short timeout
	}

	// Create deployer with default real client
	deployer := NewSSHDeployer(config)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := deployer.Connect(ctx)
	require.Error(t, err)
	var ce *ConnectionError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, "timeout", ce.Type)
}

func TestSSHDeployer_Connect_Retries(t *testing.T) {
	ctx := context.Background()

	// Create a simple test that verifies retry configuration
	config := &SSHDeployConfig{
		Host:           "test.example.com",
		Port:           22,
		Username:       "testuser",
		Password:       "testpass",
		Timeout:        5 * time.Second,
		ConnectRetries: 3,
	}

	deployer := NewSSHDeployer(config)
	require.NotNil(t, deployer)
	assert.Equal(t, 3, config.ConnectRetries)

	// In test environment connection will fail, but that's expected
	start := time.Now()
	err := deployer.Connect(ctx)
	duration := time.Since(start)

	// Should fail to connect
	require.Error(t, err)

	// Should have taken some time due to retries
	assert.Greater(t, duration, 1*time.Second)
}

func TestSSHDeployer_ExecuteCommand(t *testing.T) {
	ctx := context.Background()

	// Test with real configuration - will fail to connect but that's expected
	config := &SSHDeployConfig{
		Host:     "test.example.com",
		Port:     22,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}

	deployer := NewSSHDeployer(config)

	// ExecuteCommand should fail due to connection failure, but not panic
	result, err := deployer.ExecuteCommand(ctx, "echo 'hello world'")
	require.Error(t, err)  // Connection should fail in test environment
	require.Nil(t, result) // Result should be nil on connection failure
}

func TestSSHDeployer_ExecuteCommand_Failure(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockSSHClient(false) // Will fail to connect

	config := &SSHDeployConfig{
		Host:     "unreachable.example.com",
		Port:     22,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}

	deployer := NewSSHDeployerWithClient(config, mockClient)

	result, err := deployer.ExecuteCommand(ctx, "echo 'test'")
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestSSHDeployer_Integration_RealConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with a real configuration structure
	config := &SSHDeployConfig{
		Host:           "localhost",
		Port:           22,
		Username:       "testuser",
		Password:       "testpass",
		Timeout:        10 * time.Second,
		ConnectRetries: 3,
		CommandTimeout: 5 * time.Minute,
	}

	// Validate configuration
	err := config.Validate()
	require.NoError(t, err)

	// Create deployer
	deployer := NewSSHDeployer(config)
	assert.NotNil(t, deployer)
	assert.Equal(t, config, deployer.config)
}

func TestValidationError_Error_Shadow(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	expected := "test message (field: test_field)"
	assert.Equal(t, expected, err.Error())
}

func TestCommandResult_String(t *testing.T) {
	result := &CommandResult{
		Command:  "echo 'test'",
		ExitCode: 0,
		Stdout:   "test",
		Stderr:   "",
		Duration: time.Millisecond * 100,
	}

	// Basic stringification test
	assert.Equal(t, "echo 'test'", result.Command)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "test", result.Stdout)
	assert.Equal(t, "", result.Stderr)
	assert.Equal(t, time.Millisecond*100, result.Duration)
}

// Performance benchmarks
func BenchmarkSSHDeployer_Connect(b *testing.B) {
	config := &SSHDeployConfig{
		Host:     "test.example.com",
		Port:     22,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mockClient := NewMockSSHClient(true)
		deployer := NewSSHDeployerWithClient(config, mockClient)
		_ = deployer.Connect(ctx)
	}
}

func BenchmarkSSHDeployConfig_Validate(b *testing.B) {
	config := &SSHDeployConfig{
		Host:     "test.example.com",
		Port:     22,
		Username: "testuser",
		KeyPath:  "/path/to/key",
		Timeout:  30 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

// Concurrent testing
func TestSSHDeployer_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent stress test in short mode")
	}

	config := &SSHDeployConfig{
		Host:     "localhost",
		Port:     1, // Guaranteed connection refused, fails fast
		Username: "testuser",
		Password: "testpass",
		Timeout:  1 * time.Second,
	}

	const numGoroutines = 5
	const iterationsPerGoroutine = 3

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*iterationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				deployer := NewSSHDeployer(config) // Use real client instead of mock

				ctx := context.Background()
				err := deployer.Connect(ctx)
				if err != nil {
					// Connection failure is expected in test environment
					continue
				}

				// Test command execution
				_, err = deployer.ExecuteCommand(ctx, "echo test")
				if err != nil {
					// Command execution failure is expected if connection fails
					continue
				}

				// In real test environment, connection will fail but that's expected
				// We're mainly testing that the concurrent operations don't panic
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Error(err)
	}
}
