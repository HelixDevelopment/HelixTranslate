package distributed

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

const (
	testHost = "127.0.0.1"
	testPort = 22222
	testUser = "testuser"
	testPass = "testpass"
	testKey  = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyruz6hPRslBNLw6lId3qpgcg5zm/luvJly79bJk0AeWxcqbF
nDL9cHBoQ6f7fRXcltI2t9C214lQRrAewzzU5G5UJI6uF/KNEkhOEm1O5CF+n68j
lnJXdthc0qj7tk2ILvA4BgcIed3RbZF/aZhqAB/HX/H516gLnS07JDSTgJYY9LqZ
z+lkvgyv/OrHGsXn5AEuYpJvNIjppT+dgr1yhSy0Z/nuR6HJwyC2kyIL6j/twQEz
HL0qIptTpgQ2LLCYS5/4VDvEjjcbypGJ9je5DBMOy1cL4ZcPB9YjeCNuemV0JGu6
epZZMTm+Uh5GDQkWgi5NPIb0V99su52hpeRzOQIDAQABAoIBAE+PP/jRlE6M8u1P
qwBSbX6Ad6omYIiiubcJ6sxOhzljYbLjvdMhs5IHmvNKHgilpq7Nikmyr76AFa/X
+AqYede3cG/0Sl/9gN024OScXwRqHJ4gBjBJaQeruym0xSty28nH3cSHyAzDPyfn
nH/dH2QzFHQTqv+14/DnyjjYJTalfZRk4NIseS/o3PHcYQNwyoH/ZiDV5zQZjMUu
tpkUvrqIk4wHzFvfOnrEA5IoIDinkh6TaPJoNjDAFB/uRURTYyDZPk+FoxSuC6Dl
q5Bsa0U+7LW6kCJY3hSbXz5q+c4t1I6z+J5hAuEF0p0npwJpyshd31KfzphpJeYe
WCt59+0CgYEA9T+qMJNTmw+SA1ubiDKbIGu4gZl0LtAzawAzzeJWp+E61Hd7Q6VQ
uAOOMyiq2DZhSEO/OQJcCb5b8iTkurbB7N7jf+VxvFWthZ6ySKvivqVdqqwap3Ok
jMe7rcu9ZbQmq4a5P82sYypUNEZ6poOOLJoAvUdLTpkDO7qWqBVoEP8CgYEA057n
DRq6wyh2L8iBDjH0wDc3n3mjJ0I8JNMtRxVVpKd5sU9j5acrEotwkVlyoSLYjpgg
pYxEtm8728aiyjf2vNJPchWp/Cx2WlmbRDdANS5rE7WTB+ufgqTSFmsXB3+v1zuT
98KvuB1JGZ5I6cgAYZ5vKRdVS8gUb9bZ0mXzw8cCgYEA1Tg+rPDJlVxaI9U3SZhF
ylAdH3/cxP56VaLdZzhLArYMwcAHSO6nWPSuYsgOkN/mgD92NwhYIJiBs+pjefl+
bIPz4rQGyCjtLeilNA1Mm1eGMeZjXgZqn4LfJuClj5CqtiHxWQllwOmCP9iutapW
p2xVDDq5vGHHr9wvM3849N0CgYEAyVHzRvE12XGFtgGOXP3DdJVTMkDaqP+HDhVk
jqpKNoEo8TiwtYqqHFNRPMWWmpr23/jznepqeBAsJvG6bpx8+7cr40Ge3AtEcMGs
R2I0kCNftHlZrgBHWFcKkk9Asl6T3zOLmfm5h3M81sVRYi5lxnieEb5j49stLhR8
Vn+tPoMCgYBI41kF9uNnjKhaWDvWMW8YPT9GFZB+ChQRfWA8elZwIO82xI1eWBKg
LQ4Ncu+Tf7o26pTKtO5ev1b7hmK1dhocT4VlvUgfHJ+SJPaOFYRtCSWZVx+J1AmY
X74l53OcOpcE1k0D5mUdafYjnZstW/pK7PQ3ZkiI8VvWUZCVadtYBg==
-----END RSA PRIVATE KEY-----`
	testTimeout = 5 * time.Second
)

// MockSSHServer represents a mock SSH server for testing
type MockSSHServer struct {
	config   *ssh.ServerConfig
	listener net.Listener
	running  bool
	mu       sync.RWMutex
}

// NewMockSSHServer creates a new mock SSH server
func NewMockSSHServer() *MockSSHServer {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == testUser && string(pass) == testPass {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyruz6hPRslBNLw6lId3qpgcg5zm/luvJly79bJk0AeWxcqbF
nDL9cHBoQ6f7fRXcltI2t9C214lQRrAewzzU5G5UJI6uF/KNEkhOEm1O5CF+n68j
lnJXdthc0qj7tk2ILvA4BgcIed3RbZF/aZhqAB/HX/H516gLnS07JDSTgJYY9LqZ
z+lkvgyv/OrHGsXn5AEuYpJvNIjppT+dgr1yhSy0Z/nuR6HJwyC2kyIL6j/twQEz
HL0qIptTpgQ2LLCYS5/4VDvEjjcbypGJ9je5DBMOy1cL4ZcPB9YjeCNuemV0JGu6
epZZMTm+Uh5GDQkWgi5NPIb0V99su52hpeRzOQIDAQABAoIBAE+PP/jRlE6M8u1P
qwBSbX6Ad6omYIiiubcJ6sxOhzljYbLjvdMhs5IHmvNKHgilpq7Nikmyr76AFa/X
+AqYede3cG/0Sl/9gN024OScXwRqHJ4gBjBJaQeruym0xSty28nH3cSHyAzDPyfn
nH/dH2QzFHQTqv+14/DnyjjYJTalfZRk4NIseS/o3PHcYQNwyoH/ZiDV5zQZjMUu
tpkUvrqIk4wHzFvfOnrEA5IoIDinkh6TaPJoNjDAFB/uRURTYyDZPk+FoxSuC6Dl
q5Bsa0U+7LW6kCJY3hSbXz5q+c4t1I6z+J5hAuEF0p0npwJpyshd31KfzphpJeYe
WCt59+0CgYEA9T+qMJNTmw+SA1ubiDKbIGu4gZl0LtAzawAzzeJWp+E61Hd7Q6VQ
uAOOMyiq2DZhSEO/OQJcCb5b8iTkurbB7N7jf+VxvFWthZ6ySKvivqVdqqwap3Ok
jMe7rcu9ZbQmq4a5P82sYypUNEZ6poOOLJoAvUdLTpkDO7qWqBVoEP8CgYEA057n
DRq6wyh2L8iBDjH0wDc3n3mjJ0I8JNMtRxVVpKd5sU9j5acrEotwkVlyoSLYjpgg
pYxEtm8728aiyjf2vNJPchWp/Cx2WlmbRDdANS5rE7WTB+ufgqTSFmsXB3+v1zuT
98KvuB1JGZ5I6cgAYZ5vKRdVS8gUb9bZ0mXzw8cCgYEA1Tg+rPDJlVxaI9U3SZhF
ylAdH3/cxP56VaLdZzhLArYMwcAHSO6nWPSuYsgOkN/mgD92NwhYIJiBs+pjefl+
bIPz4rQGyCjtLeilNA1Mm1eGMeZjXgZqn4LfJuClj5CqtiHxWQllwOmCP9iutapW
p2xVDDq5vGHHr9wvM3849N0CgYEAyVHzRvE12XGFtgGOXP3DdJVTMkDaqP+HDhVk
jqpKNoEo8TiwtYqqHFNRPMWWmpr23/jznepqeBAsJvG6bpx8+7cr40Ge3AtEcMGs
R2I0kCNftHlZrgBHWFcKkk9Asl6T3zOLmfm5h3M81sVRYi5lxnieEb5j49stLhR8
Vn+tPoMCgYBI41kF9uNnjKhaWDvWMW8YPT9GFZB+ChQRfWA8elZwIO82xI1eWBKg
LQ4Ncu+Tf7o26pTKtO5ev1b7hmK1dhocT4VlvUgfHJ+SJPaOFYRtCSWZVx+J1AmY
X74l53OcOpcE1k0D5mUdafYjnZstW/pK7PQ3ZkiI8VvWUZCVadtYBg==
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
		m.listener.Close()
	}
	m.running = false
	return nil
}

// Address returns the server address
func (m *MockSSHServer) Address() string {
	return fmt.Sprintf("%s:%d", testHost, testPort)
}

// IsRunning returns whether the server is running
func (m *MockSSHServer) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// acceptConnections accepts incoming connections
func (m *MockSSHServer) acceptConnections() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return // Server stopped
		}

		go m.handleConnection(conn)
	}
}

// handleConnection handles a single connection
func (m *MockSSHServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Upgrade TCP connection to SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, m.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Discard all global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		// Handle channel requests
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "shell":
					req.Reply(true, nil)
				case "exec":
					req.Reply(true, nil)
				}
			}
		}(requests)

		// Close channel when done
		channel.Close()
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
func (c *TestSSHClient) Connect(addr string) error {
	client, err := ssh.Dial("tcp", addr, c.config)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

// Disconnect disconnects from the SSH server
func (c *TestSSHClient) Disconnect() error {
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// IsConnected returns whether the client is connected
func (c *TestSSHClient) IsConnected() bool {
	return c.client != nil
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

	t.Run("Multiple Concurrent Connections", func(t *testing.T) {
		var wg sync.WaitGroup
		numConnections := 5

		for i := 0; i < numConnections; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := NewTestSSHClient()
				err := client.Connect(server.Address())
				assert.NoError(t, err)
				if err == nil {
					client.Disconnect()
				}
			}()
		}

		wg.Wait()
	})
}

// TestMockSSHServer tests the mock SSH server functionality
func TestMockSSHServer(t *testing.T) {
	t.Run("Server Lifecycle", func(t *testing.T) {
		server := NewMockSSHServer()

		// Server should not be running initially
		assert.False(t, server.IsRunning())

		// Start server
		err := server.Start()
		require.NoError(t, err)
		assert.True(t, server.IsRunning())

		// Stop server
		err = server.Stop()
		require.NoError(t, err)
		assert.False(t, server.IsRunning())
	})

	t.Run("Server Address", func(t *testing.T) {
		server := NewMockSSHServer()
		expectedAddr := fmt.Sprintf("%s:%d", testHost, testPort)
		assert.Equal(t, expectedAddr, server.Address())
	})
}

// TestTestSSHClient tests the test SSH client functionality
func TestTestSSHClient(t *testing.T) {
	t.Run("Client Lifecycle", func(t *testing.T) {
		client := NewTestSSHClient()

		// Client should not be connected initially
		assert.False(t, client.IsConnected())

		// Try to connect to non-existent server
		err := client.Connect("127.0.0.1:12345")
		assert.Error(t, err)
		assert.False(t, client.IsConnected())

		// Disconnect should not error even when not connected
		err = client.Disconnect()
		assert.NoError(t, err)
	})
}

// TestPairingManagerPairWithService tests the PairWithService function
func TestPairingManagerPairWithService(t *testing.T) {
	t.Run("PairWithService_NonExistentWorker", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Try to pair with a non-existent worker
		err := pm.PairWithService("non-existent-worker")

		// Should return error
		if err == nil {
			t.Error("Expected error when pairing with non-existent worker")
		}
		expectedError := "service non-existent-worker not discovered"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("PairWithService_DiscoveredWorker", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Add a discovered service
		service := &RemoteService{
			WorkerID: "test-worker",
			Name:     "Test Worker",
			Status:   "discovered",
		}
		pm.services["test-worker"] = service

		// Pair with the discovered worker
		err := pm.PairWithService("test-worker")

		// Should not error
		if err != nil {
			t.Errorf("Unexpected error when pairing with discovered worker: %v", err)
		}

		// Check that status changed to "paired"
		if service.Status != "paired" {
			t.Errorf("Expected status 'paired', got '%s'", service.Status)
		}

		// Check that PairedAt is set
		if service.PairedAt == nil {
			t.Error("Expected PairedAt to be set")
		}
	})
}

// TestPairingManagerUnpairService tests the UnpairService function
func TestPairingManagerUnpairService(t *testing.T) {
	t.Run("UnpairService_NonExistentWorker", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Try to unpair a non-existent worker
		err := pm.UnpairService("non-existent-worker")

		// Should return error
		if err == nil {
			t.Error("Expected error when unpairing non-existent worker")
		}
		expectedError := "service non-existent-worker not found"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("UnpairService_PairedWorker", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Add a paired service
		service := &RemoteService{
			WorkerID: "test-worker",
			Name:     "Test Worker",
			Status:   "paired",
		}
		pm.services["test-worker"] = service

		// Unpair the worker
		err := pm.UnpairService("test-worker")

		// Should not error
		if err != nil {
			t.Errorf("Unexpected error when unpairing worker: %v", err)
		}

		// Check that status changed to "online"
		if service.Status != "online" {
			t.Errorf("Expected status 'online', got '%s'", service.Status)
		}

		// Check that PairedAt is nil
		if service.PairedAt != nil {
			t.Error("Expected PairedAt to be nil")
		}
	})
}

// TestPairingManagerDiscoverService tests the DiscoverService function
func TestPairingManagerDiscoverService(t *testing.T) {
	t.Run("DiscoverService_NonExistentWorker", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Try to discover a non-existent worker
		_, err := pm.DiscoverService(context.Background(), "non-existent-worker")

		// Should return error
		if err == nil {
			t.Error("Expected error when discovering non-existent worker")
		}
		// Should contain error message about SSH connection
		if !strings.Contains(err.Error(), "SSH connection") {
			t.Errorf("Expected SSH connection error, got: %v", err)
		}
	})
}

// TestPairingManagerBasic tests PairingManager basic functions
func TestPairingManagerBasic(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()

		pm := NewPairingManager(sshPool, eventBus)

		if pm == nil {
			t.Error("Expected non-nil PairingManager")
		}

		if pm.services == nil {
			t.Error("Expected services map to be initialized")
		}

		if len(pm.services) != 0 {
			t.Error("Expected empty services map")
		}

		if pm.httpClient == nil {
			t.Error("Expected httpClient to be initialized")
		}

		if pm.checkInterval != 30*time.Second {
			t.Errorf("Expected checkInterval to be 30s, got %v", pm.checkInterval)
		}
	})

	t.Run("GetPairedServices_Empty", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		paired := pm.GetPairedServices()

		if paired == nil {
			t.Error("Expected non-nil paired services map")
		}

		if len(paired) != 0 {
			t.Error("Expected empty paired services map")
		}
	})

	t.Run("GetPairedServices_MixedStatuses", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Add test services with different statuses
		pm.services["worker1"] = &RemoteService{
			WorkerID: "worker1",
			Status:   "paired",
		}

		pm.services["worker2"] = &RemoteService{
			WorkerID: "worker2",
			Status:   "discovered",
		}

		pm.services["worker3"] = &RemoteService{
			WorkerID: "worker3",
			Status:   "paired",
		}

		paired := pm.GetPairedServices()

		if len(paired) != 2 {
			t.Errorf("Expected 2 paired services, got %d", len(paired))
		}

		if _, exists := paired["worker1"]; !exists {
			t.Error("Expected worker1 to be in paired services")
		}

		if _, exists := paired["worker2"]; exists {
			t.Error("Expected worker2 to NOT be in paired services")
		}

		if _, exists := paired["worker3"]; !exists {
			t.Error("Expected worker3 to be in paired services")
		}
	})

	t.Run("GetServiceStatus_Existing", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Add a test service
		pm.services["worker1"] = &RemoteService{
			WorkerID: "worker1",
			Status:   "paired",
		}

		status, err := pm.GetServiceStatus("worker1")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if status != "paired" {
			t.Errorf("Expected status 'paired', got '%s'", status)
		}
	})

	t.Run("GetServiceStatus_NonExistent", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		status, err := pm.GetServiceStatus("nonexistent")

		if err == nil {
			t.Error("Expected error for non-existent service")
		}

		if status != "unknown" {
			t.Errorf("Expected status 'unknown', got '%s'", status)
		}

		expectedError := "service nonexistent not found"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("CloseGracefully", func(t *testing.T) {
		sshPool := NewSSHPool()
		eventBus := events.NewEventBus()
		pm := NewPairingManager(sshPool, eventBus)

		// Should not panic when closing
		pm.Close()

		// Check that context is cancelled
		select {
		case <-pm.ctx.Done():
			// Expected - context should be cancelled
		default:
			t.Error("Expected context to be cancelled after Close")
		}
	})
}
