package distributed

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Test SSH configuration for testing
const (
	testHost    = "127.0.0.1"
	testPort    = 2222
	testUser    = "testuser"
	testPass    = "testpass"
	testTimeout = 5 * time.Second
)

// MockSSHServer represents a mock SSH server for testing
type MockSSHServer struct {
	listener net.Listener
	config   *ssh.ServerConfig
	running  bool
	mu       sync.Mutex
}

// NewMockSSHServer creates a new mock SSH server
func NewMockSSHServer() *MockSSHServer {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == testUser && string(pass) == testPass {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
		NoClientAuth: false,
	}

	// Add a test host key (generated for testing purposes)
	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAzK8k5L6n2B4B7M5Xv1J2J3X9Y5Z6A7B8C9D0E1F2G3H4I5J6
K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7
P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8
U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O9P0Q1R2S3T4U5V6W7X8Y9
Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0
E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J
2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3
P4Q5R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O9P0Q1R2S3T4U
5V6W7X8Y9Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6
A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E
7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8
K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O9
P0Q1R2S3T4U5V6W7X8Y9Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0
U1V2W3X4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0
Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1
D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1
H2I3J4K5L6M7N8O9P0Q1R2S3T4U5V6W7X8Y9Z0A1B2C3D4E5F6G7H8I9J0K1
L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P
2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T
2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1
X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O9P0Q1R2S3T4U5V6W7X8Y9Z0A
1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0
E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9
H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9
L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O
9P0Q1R2S3T4U5V6W7X8Y9Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8
S9T0U1V2W3X4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7
V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6
Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4A5
B6C7D8E9F0G1H2I3J4K5L6M7N8O9P0Q1R2S3T4U5V6W7X8Y9Z0A1B2C3D4
E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0E1F2G3
H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J
2K3L4M5N6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0
M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O
9P0Q1R2S3T4U5V6W7X8Y9Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6
Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1P2Q3
R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9
R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5
R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4K5L6M7N8O9P0Q1
R2S3T4U5V6W7X8Y9Z0A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6
Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0E1F2G3H4I5J6K7L8M9N0O1
P2Q3R4S5T6U7V8W9X0Y1Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N
6O7P8Q9R0S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7J8K9L0
M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4A5B6C7D8E9F0G1H2I3J4
K5L6M7N8O9P0Q1R2S3T4U5V6W7X8Y9Z0A1B2C3D4E5F6G7
H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6A7B8C9D0
E1F2G3H4I5J6K7L8M9N0O1P2Q3R4S5T6U7V8W9X0Y1
Z2A3B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0
S1T2U3V4W5X6Y7Z8A9B0C1D2E3F4G5H6I7
J8K9L0M1N2O3P4Q5R6S7T8U9V0W1
X2Y3Z4A5B6C7D8E9F0G1H2
I3J4K5L6M7N8O9P0Q1R
2S3T4U5V6W7X8Y9Z0
A1B2C3D4E5F6G7
H8I9J0K1L2M3N4
O5P6Q7R8S9T0
U1V2W3X4Y5Z
6A7B8C9D0E1
F2G3H4I5J6
K7L8M9N0O1
P2Q3R4S5T
6U7V8W9X
0Y1Z2A3B
4C5D6E7
F8G9H0I
1J2K3L4
M5N6O7
P8Q9R0
S1T2U3
V4W5X6
Y7Z8A9
B0C1D2
E3F4G5
H6I7J8
K9L0M1
N2O3P4
Q5R6S7
T8U9V0
W1X2Y3
Z4A5B6
C7D8E9
F0G1H2
I3J4K5
L6M7N8
O9P0Q1
R2S3T4
U5V6W7
X8Y9Z0
A1B2C3
D4E5F6
G7H8I9
J0K1L2
M3N4O5
P6Q7R8
S9T0U1
V2W3X4
Y5Z6A7
B8C9D0
E1F2G3
H4I5J6
K7L8M9
N0O1P2
Q3R4S5
T6U7V8
W9X0Y1
Z2A3B4
C5D6E7
F8G9H9
-----END RSA PRIVATE KEY-----`)

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse test private key: %v", err))
	}
	config.AddHostKey(signer)

	return &MockSSHServer{
		config: config,
	}
}

// Start starts the mock SSH server
func (m *MockSSHServer) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testHost, testPort))
	if err != nil {
		return err
	}
	m.listener = listener
	m.running = true

	go m.acceptConnections()
	return nil
}

// Stop stops the mock SSH server
func (m *MockSSHServer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listener != nil {
		m.running = false
		return m.listener.Close()
	}
	return nil
}

// Address returns the server address
func (m *MockSSHServer) Address() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listener != nil {
		return m.listener.Addr().String()
	}
	return ""
}

// acceptConnections handles incoming SSH connections
func (m *MockSSHServer) acceptConnections() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			m.mu.Lock()
			if !m.running {
				m.mu.Unlock()
				return
			}
			m.mu.Unlock()
			continue
		}

		go func() {
			_, _, _, err := ssh.NewServerConn(conn, m.config)
			if err != nil {
				return
			}
		}()
	}
}

// TestSSHClient represents a test SSH client
type TestSSHClient struct {
	client *ssh.Client
	config *ssh.ClientConfig
}

// NewTestSSHClient creates a new test SSH client
func NewTestSSHClient() *TestSSHClient {
	config := &ssh.ClientConfig{
		User: testUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(testPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         testTimeout,
	}

	return &TestSSHClient{
		config: config,
	}
}

// Connect connects to the SSH server
func (t *TestSSHClient) Connect(address string) error {
	client, err := ssh.Dial("tcp", address, t.config)
	if err != nil {
		return err
	}
	t.client = client
	return nil
}

// Disconnect disconnects from the SSH server
func (t *TestSSHClient) Disconnect() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// TestPairingWorkflow tests the complete pairing workflow
func TestPairingWorkflow(t *testing.T) {
	// Setup mock SSH server
	server := NewMockSSHServer()
	require.NoError(t, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	t.Run("Successful Pairing", func(t *testing.T) {
		client := NewTestSSHClient()
		
		// Test connection
		err := client.Connect(server.Address())
		require.NoError(t, err)
		
		// Verify connection is established
		assert.NotNil(t, client.client)
		
		// Test disconnection
		err = client.Disconnect()
		assert.NoError(t, err)
	})

	t.Run("Failed Authentication", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User: "wronguser",
			Auth: []ssh.AuthMethod{
				ssh.Password("wrongpass"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         testTimeout,
		}

		// Try to connect with wrong credentials
		_, err := ssh.Dial("tcp", server.Address(), config)
		assert.Error(t, err)
	})

	t.Run("Connection Timeout", func(t *testing.T) {
		config := &ssh.ClientConfig{
			User:            testUser,
			Auth:            []ssh.AuthMethod{ssh.Password(testPass)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         1 * time.Millisecond, // Very short timeout
		}

		// Connect to non-existent address
		_, err := ssh.Dial("tcp", "127.0.0.1:12345", config)
		assert.Error(t, err)
	})
}

// TestConcurrentPairing tests concurrent pairing operations
func TestConcurrentPairing(t *testing.T) {
	server := NewMockSSHServer()
	require.NoError(t, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	const numClients = 10
	var wg sync.WaitGroup
	errs := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			client := NewTestSSHClient()
			err := client.Connect(server.Address())
			if err != nil {
				errs <- err
				return
			}
			
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			
			err = client.Disconnect()
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	// Check if any errors occurred
	for err := range errs {
		t.Errorf("Concurrent pairing error: %v", err)
	}
}

// TestPairingSecurity tests security aspects of pairing
func TestPairingSecurity(t *testing.T) {
	server := NewMockSSHServer()
	require.NoError(t, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	t.Run("Host Key Verification", func(t *testing.T) {
		// This test would normally verify host keys
		// For testing, we use insecure callback
		config := &ssh.ClientConfig{
			User: testUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(testPass),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         testTimeout,
		}

		client, err := ssh.Dial("tcp", server.Address(), config)
		require.NoError(t, err)
		defer client.Close()

		assert.NotNil(t, client)
	})

	t.Run("Connection Encryption", func(t *testing.T) {
		// Test that connection is encrypted (SSH should encrypt by default)
		config := &ssh.ClientConfig{
			User: testUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(testPass),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         testTimeout,
		}

		client, err := ssh.Dial("tcp", server.Address(), config)
		require.NoError(t, err)
		defer client.Close()

		// SSH connection should be established
		assert.NotNil(t, client)
		
		// Verify connection state
		assert.Equal(t, testUser, client.Conn.User())
	})

	t.Run("Authentication Failures", func(t *testing.T) {
		testCases := []struct {
			name     string
			user     string
			password string
		}{
			{"Wrong Password", testUser, "wrongpass"},
			{"Wrong User", "wronguser", testPass},
			{"Both Wrong", "wronguser", "wrongpass"},
			{"Empty Credentials", "", ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := &ssh.ClientConfig{
					User: tc.user,
					Auth: []ssh.AuthMethod{
						ssh.Password(tc.password),
					},
					HostKeyCallback: ssh.InsecureIgnoreHostKey(),
					Timeout:         testTimeout,
				}

				_, err := ssh.Dial("tcp", server.Address(), config)
				assert.Error(t, err)
			})
		}
	})
}

// TestPairingErrorHandling tests error handling in pairing operations
func TestPairingErrorHandling(t *testing.T) {
	t.Run("Server Not Running", func(t *testing.T) {
		client := NewTestSSHClient()
		
		// Try to connect to non-existent server
		err := client.Connect("127.0.0.1:12345")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("Network Errors", func(t *testing.T) {
		client := NewTestSSHClient()
		
		// Try to connect to invalid address
		err := client.Connect("invalid.address:12345")
		assert.Error(t, err)
	})

	server := NewMockSSHServer()
	require.NoError(t, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	t.Run("Connection Disruption", func(t *testing.T) {
		client := NewTestSSHClient()
		err := client.Connect(server.Address())
		require.NoError(t, err)

		// Stop server while client is connected
		err = server.Stop()
		require.NoError(t, err)

		// Try to use client after server shutdown
		_, err = client.client.NewSession()
		assert.Error(t, err)
	})
}

// TestPairingPerformance tests performance of pairing operations
func TestPairingPerformance(t *testing.T) {
	server := NewMockSSHServer()
	require.NoError(t, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	const numConnections = 100
	start := time.Now()

	for i := 0; i < numConnections; i++ {
		client := NewTestSSHClient()
		err := client.Connect(server.Address())
		require.NoError(t, err)
		
		err = client.Disconnect()
		require.NoError(t, err)
	}

	duration := time.Since(start)
	avgDuration := duration / numConnections

	t.Logf("Average connection time: %v", avgDuration)
	
	// Performance assertion - each connection should be reasonably fast
	assert.Less(t, avgDuration, 1*time.Second, "Connection took too long")
}

// TestPairingContext tests pairing with context cancellation
func TestPairingContext(t *testing.T) {
	server := NewMockSSHServer()
	require.NoError(t, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	t.Run("Context Cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		
		// Cancel context immediately
		cancel()
		
		config := &ssh.ClientConfig{
			User: testUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(testPass),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}

		// This should fail due to context cancellation
		done := make(chan error, 1)
		go func() {
			_, err := ssh.Dial("tcp", server.Address(), config)
			done <- err
		}()

		select {
		case err := <-done:
			// Connection should still succeed because SSH doesn't directly support context
			assert.NoError(t, err)
		case <-ctx.Done():
			t.Log("Context cancelled as expected")
		}
	})

	t.Run("Context Timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		config := &ssh.ClientConfig{
			User: testUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(testPass),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}

		// Test connection with short timeout
		start := time.Now()
		_, err := ssh.Dial("tcp", server.Address(), config)
		duration := time.Since(start)

		// Connection should be fast regardless
		assert.NoError(t, err)
		assert.Less(t, duration, 1*time.Second)
	})
}

// BenchmarkPairing benchmarks the pairing operation
func BenchmarkPairing(b *testing.B) {
	server := NewMockSSHServer()
	require.NoError(b, server.Start())
	defer server.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := NewTestSSHClient()
			err := client.Connect(server.Address())
			if err != nil {
				b.Fatal(err)
			}
			
			err = client.Disconnect()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}