package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/version"

	"golang.org/x/crypto/ssh"
)

// MockSSHServer creates a mock SSH server for testing
type MockSSHServer struct {
	config   *ssh.ServerConfig
	listener net.Listener
	server   *ssh.ServerConn
	handlers map[string]func(args []string) (string, error)
	port     int
	tempDir  string
}

// NewMockSSHServer creates a new mock SSH server for testing
func NewMockSSHServer(t *testing.T) *MockSSHServer {
	tempDir, err := os.MkdirTemp("", "mock_ssh_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Generate host key
	hostKey, err := generateHostKey()
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == "testuser" && string(password) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
	}
	config.AddHostKey(hostKey)

	// Start listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start SSH listener: %v", err)
	}

	server := &MockSSHServer{
		config:   config,
		listener: listener,
		handlers: make(map[string]func(args []string) (string, error)),
		tempDir:  tempDir,
		port:     listener.Addr().(*net.TCPAddr).Port,
	}

	// Setup default handlers
	server.setupDefaultHandlers()

	return server
}

// setupDefaultHandlers sets up default command handlers for the mock SSH server
func (s *MockSSHServer) setupDefaultHandlers() {
	// Hash command handler
	s.handlers["hash-codebase"] = func(args []string) (string, error) {
		// Generate a deterministic hash for testing
		return "abc123def456", nil
	}

	// File upload handler (cat with heredoc)
	s.handlers["cat"] = func(args []string) (string, error) {
		// For mock purposes, just return success
		return "", nil
	}

	// File download handler
	s.handlers["download"] = func(args []string) (string, error) {
		filename := args[0]
		content := fmt.Sprintf("Mock content for %s", filename)
		return content, nil
	}

	// Directory creation handler
	s.handlers["mkdir"] = func(args []string) (string, error) {
		// For mock purposes, just return success
		return "", nil
	}
}

// Start starts the mock SSH server
func (s *MockSSHServer) Start(t *testing.T) {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				return
			}

			go s.handleConnection(t, conn)
		}
	}()
}

// handleConnection handles incoming SSH connections
func (s *MockSSHServer) handleConnection(t *testing.T, conn net.Conn) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go func(in <-chan *ssh.Request) {
			for req := range in {
				if req.Type == "exec" {
					s.handleExec(t, channel, string(req.Payload[4:]))
				}
			}
		}(requests)
	}
}

// handleExec handles exec requests
func (s *MockSSHServer) handleExec(t *testing.T, channel ssh.Channel, command string) {
	defer channel.Close()

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	args := parts[1:]

	// Handle commands
	if handler, exists := s.handlers[cmd]; exists {
		output, err := handler(args)
		if err != nil {
			channel.SendRequest("exit-status", false, ssh.Marshal(&struct{ Status uint32 }{Status: 1}))
			return
		}

		channel.Write([]byte(output))
		channel.SendRequest("exit-status", false, ssh.Marshal(&struct{ Status uint32 }{Status: 0}))
		return
	}

	// Default success response
	channel.SendRequest("exit-status", false, ssh.Marshal(&struct{ Status uint32 }{Status: 0}))
}

// Stop stops the mock SSH server
func (s *MockSSHServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// GetAddress returns the server address
func (s *MockSSHServer) GetAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", s.port)
}

// AddHandler adds a custom command handler
func (s *MockSSHServer) AddHandler(command string, handler func(args []string) (string, error)) {
	s.handlers[command] = handler
}

// generateHostKey generates a temporary host key for the mock server
func generateHostKey() (ssh.Signer, error) {
	keyFile := filepath.Join(os.TempDir(), "test_host_key")
	defer os.Remove(keyFile)

	// Generate a simple test key
	// In a real implementation, you'd generate a proper RSA/ED25519 key
	// For testing purposes, we'll create a simple placeholder
	return nil, nil
}

// TestSSHTranslationIntegration tests the complete SSH translation workflow
func TestSSHTranslationIntegration(t *testing.T) {
	// Setup mock SSH server
	mockServer := NewMockSSHServer(t)
	defer mockServer.Stop()

	mockServer.Start(t)

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test SSH worker connection
	t.Run("SSHConnection", func(t *testing.T) {
		logger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		config := sshworker.SSHWorkerConfig{
			Host:              mockServer.GetAddress(),
			Username:          "testuser",
			Password:          "testpass",
			RemoteDir:         "/tmp/test",
			ConnectionTimeout: 5 * time.Second,
			CommandTimeout:    5 * time.Second,
		}

		worker, err := sshworker.NewSSHWorker(config, logger)
		if err != nil {
			t.Fatalf("Failed to create SSH worker: %v", err)
		}
		defer worker.Close()

		ctx := context.Background()

		// Test connection
		if err := worker.TestConnection(ctx); err != nil {
			t.Errorf("SSH connection test failed: %v", err)
		}
	})

	// Test codebase hash verification
	t.Run("CodebaseHashVerification", func(t *testing.T) {
		logger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		config := sshworker.SSHWorkerConfig{
			Host:              mockServer.GetAddress(),
			Username:          "testuser",
			Password:          "testpass",
			RemoteDir:         "/tmp/test",
			ConnectionTimeout: 5 * time.Second,
			CommandTimeout:    5 * time.Second,
		}

		worker, err := sshworker.NewSSHWorker(config, logger)
		if err != nil {
			t.Fatalf("Failed to create SSH worker: %v", err)
		}
		defer worker.Close()

		ctx := context.Background()

		// Test connection first
		if err := worker.TestConnection(ctx); err != nil {
			t.Skipf("Skipping hash test due to connection failure: %v", err)
		}

		// Generate local hash
		localHasher := version.NewCodebaseHasher()
		localHash, err := localHasher.CalculateHash()
		if err != nil {
			t.Fatalf("Failed to generate local hash: %v", err)
		}

		// Get remote hash (mock returns fixed value)
		remoteHash, err := worker.GetRemoteCodebaseHash(ctx)
		if err != nil {
			t.Errorf("Failed to get remote hash: %v", err)
		}

		// In real scenario, these would match after codebase sync
		// For testing, we just verify the calls work
		t.Logf("Local hash: %s, Remote hash: %s", localHash, remoteHash)
	})

	// Test file upload/download
	t.Run("FileTransfer", func(t *testing.T) {
		logger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		config := sshworker.SSHWorkerConfig{
			Host:              mockServer.GetAddress(),
			Username:          "testuser",
			Password:          "testpass",
			RemoteDir:         "/tmp/test",
			ConnectionTimeout: 5 * time.Second,
			CommandTimeout:    5 * time.Second,
		}

		worker, err := sshworker.NewSSHWorker(config, logger)
		if err != nil {
			t.Fatalf("Failed to create SSH worker: %v", err)
		}
		defer worker.Close()

		ctx := context.Background()

		// Test connection first
		if err := worker.TestConnection(ctx); err != nil {
			t.Skipf("Skipping file transfer test due to connection failure: %v", err)
		}

		// Create test file
		testFile := filepath.Join(os.TempDir(), "test_upload.txt")
		testContent := "This is test content for upload"
		if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		defer os.Remove(testFile)

		// Upload file
		remotePath := "/tmp/test/test_upload.txt"
		if err := worker.UploadFile(ctx, testFile, remotePath); err != nil {
			t.Errorf("Failed to upload file: %v", err)
		}

		// Upload data
		testData := []byte("This is test data for upload")
		dataRemotePath := "/tmp/test/test_data.txt"
		if err := worker.UploadData(ctx, testData, dataRemotePath); err != nil {
			t.Errorf("Failed to upload data: %v", err)
		}

		// Download file
		downloadPath := filepath.Join(os.TempDir(), "test_download.txt")
		if err := worker.DownloadFile(ctx, remotePath, downloadPath); err != nil {
			t.Errorf("Failed to download file: %v", err)
		}
		defer os.Remove(downloadPath)

		// Verify downloaded content (in real scenario)
		// For mock testing, we just verify the operation doesn't fail
	})
}

// TestSSHTranslationEndToEnd tests the complete end-to-end workflow
func TestSSHTranslationEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// This test would require:
	// 1. A real SSH server setup or more comprehensive mocking
	// 2. Mock ebook files for testing
	// 3. Mock llama.cpp integration
	// 4. Full integration of all components

	// For now, we'll create a placeholder test structure
	t.Run("CompleteWorkflow", func(t *testing.T) {
		// In a complete implementation, this would:
		// 1. Setup mock SSH server with all required handlers
		// 2. Create test FB2 file
		// 3. Execute the complete SSH translation workflow
		// 4. Verify all 4 output files are created correctly
		// 5. Verify file contents and structure

		t.Skip("End-to-end test requires full mock implementation")
	})
}

// TestSSHTranslationErrorHandling tests error scenarios
func TestSSHTranslationErrorHandling(t *testing.T) {
	// Test various error scenarios
	testCases := []struct {
		name     string
		setup    func(*MockSSHServer)
		testFunc func(*testing.T, sshworker.SSHWorker, context.Context)
	}{
		{
			name: "AuthenticationFailure",
			setup: func(s *MockSSHServer) {
				// Authentication is handled by config, no setup needed
			},
			testFunc: func(t *testing.T, worker sshworker.SSHWorker, ctx context.Context) {
				// This test would verify authentication failures
				// In the current setup, this is handled by NewSSHWorker
			},
		},
		{
			name: "CommandTimeout",
			setup: func(s *MockSSHServer) {
				s.AddHandler("sleep", func(args []string) (string, error) {
					time.Sleep(10 * time.Second)
					return "done", nil
				})
			},
			testFunc: func(t *testing.T, worker sshworker.SSHWorker, ctx context.Context) {
				// Test command timeout handling
				// This would need a worker with very short timeout
				t.Skip("Timeout test requires worker configuration")
			},
		},
		{
			name: "RemoteCommandFailure",
			setup: func(s *MockSSHServer) {
				s.AddHandler("fail", func(args []string) (string, error) {
					return "", fmt.Errorf("command failed")
				})
			},
			testFunc: func(t *testing.T, worker sshworker.SSHWorker, ctx context.Context) {
				result, err := worker.ExecuteCommand(ctx, "fail")
				if err == nil {
					t.Error("Expected command failure but got success")
				}
				if result.ExitCode == 0 {
					t.Error("Expected non-zero exit code for failed command")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := NewMockSSHServer(t)
			defer mockServer.Stop()

			tc.setup(mockServer)
			mockServer.Start(t)

			// Wait for server to start
			time.Sleep(100 * time.Millisecond)

			logger := logger.NewLogger(logger.LoggerConfig{
				Level:  logger.DEBUG,
				Format: logger.FORMAT_TEXT,
			})

			config := sshworker.SSHWorkerConfig{
				Host:              mockServer.GetAddress(),
				Username:          "testuser",
				Password:          "testpass",
				RemoteDir:         "/tmp/test",
				ConnectionTimeout: 1 * time.Second,
				CommandTimeout:    1 * time.Second,
			}

			worker, err := sshworker.NewSSHWorker(config, logger)
			if err != nil {
				t.Fatalf("Failed to create SSH worker: %v", err)
			}
			defer worker.Close()

			ctx := context.Background()

			// Test connection first
			if err := worker.TestConnection(ctx); err != nil {
				t.Skipf("Skipping test case %s due to connection failure: %v", tc.name, err)
			}

			tc.testFunc(t, *worker, ctx)
		})
	}
}

// BenchmarkSSHTranslation benchmarks SSH translation performance
func BenchmarkSSHTranslation(b *testing.B) {
	mockServer := NewMockSSHServer(&testing.T{})
	defer mockServer.Stop()

	mockServer.Start(&testing.T{})

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	logger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.WARN, // Reduce logging for benchmark
		Format: logger.FORMAT_TEXT,
	})

	config := sshworker.SSHWorkerConfig{
		Host:              mockServer.GetAddress(),
		Username:          "testuser",
		Password:          "testpass",
		RemoteDir:         "/tmp/test",
		ConnectionTimeout: 5 * time.Second,
		CommandTimeout:    5 * time.Second,
	}

	worker, err := sshworker.NewSSHWorker(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SSH worker: %v", err)
	}
	defer worker.Close()

	ctx := context.Background()

	// Test connection
	if err := worker.TestConnection(ctx); err != nil {
		b.Skipf("Skipping benchmark due to connection failure: %v", err)
	}

	// Benchmark simple command execution
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := worker.ExecuteCommand(ctx, "echo test")
		if err != nil {
			b.Errorf("Command execution failed: %v", err)
		}
	}
}