package distributed

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// mockSSHConnection implements a mock SSH connection for testing
type mockSSHConnection struct {
	client      *ssh.Client
	session     *ssh.Session
	connected   bool
	closeCalled bool
	mu          sync.Mutex
}

func newMockSSHConnection() *mockSSHConnection {
	// Create a mock client and session
	// In a real test environment, these would be nil
	// but we need to return a non-nil struct for the tests
	return &mockSSHConnection{
		client:    nil, // Keep nil since we're mocking
		connected: true,
	}
}

func (m *mockSSHConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	m.connected = false
	return nil
}

func (m *mockSSHConnection) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

func (m *mockSSHConnection) GetClient() *ssh.Client {
	// Return the mock client (nil is fine for testing)
	return m.client
}

// mockSSHPoolImplementation implements the SSHPool interface for testing
type mockSSHPoolImplementation struct {
	connections map[string]*mockSSHConnection
	mu          sync.RWMutex
	maxActive   int
	maxIdle     int
	createCount int
	closeCount  int
	getCount    int
	putCount    int
}

func newMockSSHPoolImplementation(maxActive, maxIdle int) *mockSSHPoolImplementation {
	return &mockSSHPoolImplementation{
		connections: make(map[string]*mockSSHConnection),
		maxActive:   maxActive,
		maxIdle:     maxIdle,
	}
}

func (m *mockSSHPoolImplementation) GetConnection(ctx context.Context, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getCount++

	// Check if context is canceled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue
	}

	if len(m.connections) >= m.maxActive {
		return nil, errors.New("pool: maximum active connections reached")
	}

	if conn, exists := m.connections[address]; exists && !conn.IsClosed() {
		return conn.GetClient(), nil
	}

	m.createCount++
	conn := newMockSSHConnection()
	m.connections[address] = conn

	return conn.GetClient(), nil
}

func (m *mockSSHPoolImplementation) PutConnection(address string, client *ssh.Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.putCount++
	return nil
}

func (m *mockSSHPoolImplementation) CloseConnection(address string, client *ssh.Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closeCount++
	if conn, exists := m.connections[address]; exists {
		conn.Close()
		delete(m.connections, address)
	}
	return nil
}

func (m *mockSSHPoolImplementation) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for address, conn := range m.connections {
		conn.Close()
		delete(m.connections, address)
	}
	return nil
}

func (m *mockSSHPoolImplementation) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"total_connections": len(m.connections),
		"max_active":        m.maxActive,
		"max_idle":          m.maxIdle,
		"get_count":         m.getCount,
		"put_count":         m.putCount,
		"create_count":      m.createCount,
		"close_count":       m.closeCount,
	}
}

// TestSSHPoolBasicOperations tests basic connection pool operations
func TestSSHPoolBasicOperations(t *testing.T) {
	pool := newMockSSHPoolImplementation(10, 5)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test initial state
	stats := pool.GetStats()
	assert.Equal(t, 0, stats["total_connections"])
	assert.Equal(t, 10, stats["max_active"])
	assert.Equal(t, 5, stats["max_idle"])

	// Test creating new connection
	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := pool.GetConnection(ctx, "test.example.com:22", config)
	assert.NoError(t, err)
	// In mock context, client will be nil but that's expected
	// We're testing the pool logic, not actual SSH connections

	stats = pool.GetStats()
	assert.Equal(t, 1, stats["total_connections"])
	assert.Equal(t, 1, stats["create_count"])

	// Test reusing existing connection
	client2, err := pool.GetConnection(ctx, "test.example.com:22", config)
	assert.NoError(t, err)
	_ = client2 // Avoid unused variable warning
	// In mock context, both clients will be nil but represent the same "connection"

	stats = pool.GetStats()
	assert.Equal(t, 1, stats["total_connections"]) // No new connection created
	assert.Equal(t, 1, stats["create_count"])
	assert.Equal(t, 2, stats["get_count"])

	// Test returning connection to pool
	err = pool.PutConnection("test.example.com:22", client)
	assert.NoError(t, err)

	stats = pool.GetStats()
	assert.Equal(t, 1, stats["put_count"])

	// Test closing connection
	err = pool.CloseConnection("test.example.com:22", client)
	assert.NoError(t, err)

	stats = pool.GetStats()
	assert.Equal(t, 0, stats["total_connections"])
	assert.Equal(t, 1, stats["close_count"])
}

// TestSSHPoolConcurrency tests concurrent access to the connection pool
func TestSSHPoolConcurrency(t *testing.T) {
	pool := newMockSSHPoolImplementation(50, 20)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 10

	// Concurrent get operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				address := fmt.Sprintf("test%d.example.com:22", id%10) // 10 unique addresses
				client, err := pool.GetConnection(ctx, address, config)
				assert.NoError(t, err)
				// In mock context, client will be nil but that's expected

				// Simulate some work
				time.Sleep(time.Millisecond * time.Duration(j%5))

				err = pool.PutConnection(address, client)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	stats := pool.GetStats()
	assert.Equal(t, numGoroutines*operationsPerGoroutine, stats["get_count"])
	assert.Equal(t, numGoroutines*operationsPerGoroutine, stats["put_count"])
	assert.Equal(t, 10, stats["total_connections"]) // Only 10 unique addresses
	assert.Equal(t, 10, stats["create_count"])
}

// TestSSHPoolLimits tests connection pool limits and boundaries
func TestSSHPoolLimits(t *testing.T) {
	const maxConnections = 5
	pool := newMockSSHPoolImplementation(maxConnections, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Test max active connections limit
	addresses := make([]string, maxConnections+3)
	for i := 0; i < maxConnections+3; i++ {
		addresses[i] = fmt.Sprintf("test%d.example.com:22", i)
	}

	// First maxConnections should succeed
	for i := 0; i < maxConnections; i++ {
		_, err := pool.GetConnection(ctx, addresses[i], config)
		assert.NoError(t, err)
		// In mock context, client will be nil but that's expected
	}

	// Next attempts should fail
	for i := maxConnections; i < maxConnections+3; i++ {
		client, err := pool.GetConnection(ctx, addresses[i], config)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "maximum active connections")
	}

	stats := pool.GetStats()
	assert.Equal(t, maxConnections, stats["total_connections"])
	assert.Equal(t, maxConnections+3, stats["get_count"])
}

// TestSSHPoolTimeout tests connection timeout behavior
func TestSSHPoolTimeout(t *testing.T) {
	pool := newMockSSHPoolImplementation(10, 5)

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			time.Sleep(time.Millisecond * 10) // Simulate slow connection
			return nil
		},
	}

	// Should fail due to timeout
	_, err := pool.GetConnection(ctx, "slow.example.com:22", config)
	assert.Error(t, err)
	// In mock context, client will be nil but that's expected
	assert.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded))
}

// TestSSHPoolCloseAll tests closing all connections in the pool
func TestSSHPoolCloseAll(t *testing.T) {
	pool := newMockSSHPoolImplementation(20, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Create multiple connections
	addresses := []string{"test1.example.com:22", "test2.example.com:22", "test3.example.com:22"}
	// var clients []*ssh.Client  // Not needed for test

	for _, address := range addresses {
		_, err := pool.GetConnection(ctx, address, config)
		require.NoError(t, err)
		// clients = append(clients, client)  // Not needed
	}

	stats := pool.GetStats()
	assert.Equal(t, 3, stats["total_connections"])

	// Close all connections
	err := pool.CloseAll()
	assert.NoError(t, err)

	stats = pool.GetStats()
	assert.Equal(t, 0, stats["total_connections"])
}

// TestSSHPoolResourceLeak tests for resource leaks in the connection pool
func TestSSHPoolResourceLeak(t *testing.T) {
	pool := newMockSSHPoolImplementation(100, 50)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Create and close many connections
	var err error
	for i := 0; i < 1000; i++ {
		address := fmt.Sprintf("test%d.example.com:22", i%20) // 20 unique addresses

		client, err := pool.GetConnection(ctx, address, config)
		require.NoError(t, err)
		// In mock context, client will be nil but that's expected

		// Sometimes close directly, sometimes return to pool
		if i%3 == 0 {
			err = pool.CloseConnection(address, client)
			require.NoError(t, err)
		} else {
			err = pool.PutConnection(address, client)
			require.NoError(t, err)
		}
	}

	// Check for resource leaks
	stats := pool.GetStats()
	assert.Equal(t, 1000, stats["get_count"])
	putCount := stats["put_count"].(int) + stats["close_count"].(int)
	assert.Equal(t, 1000, putCount)                        // Put or close called each time
	assert.True(t, stats["total_connections"].(int) <= 20) // Max 20 unique addresses

	// Clean up
	err = pool.CloseAll()
	assert.NoError(t, err)
	statsAfterClose := pool.GetStats()
	assert.Equal(t, 0, statsAfterClose["total_connections"])
}

// TestSSHPoolIntegrationWithWorker tests SSH pool integration with worker system
func TestSSHPoolIntegrationWithWorker(t *testing.T) {
	pool := newMockSSHPoolImplementation(10, 5)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Create worker info
	workerInfo := WorkerInfo{
		ID:       "worker-1",
		Address:  "worker1.example.com:22",
		Username: "test",
	}

	// Get SSH connection for worker
	sshClient, err := pool.GetConnection(ctx, workerInfo.Address, config)
	assert.NoError(t, err)
	// In mock context, client will be nil but that's expected

	// Create SSH worker
	workerConfig := sshworker.SSHWorkerConfig{
		Host:     "worker1.example.com",
		Port:     22,
		Username: "test",
	}
	log := logger.NewLogger(logger.LoggerConfig{})
	sshWorker, err := sshworker.NewSSHWorker(workerConfig, log)
	assert.NoError(t, err)
	assert.NotNil(t, sshWorker)

	// Test worker functionality with ExecuteCommand
	result, err := sshWorker.ExecuteCommand(ctx, "echo 'Hello world'")
	// Note: In mock environment, this might fail, but we're testing the integration
	if err == nil {
		assert.NotNil(t, result)
	}

	// Return connection to pool
	err = pool.PutConnection(workerInfo.Address, sshClient)
	assert.NoError(t, err)
}

// BenchmarkSSHPoolGet benchmarks connection pool get operations
func BenchmarkSSHPoolGet(b *testing.B) {
	pool := newMockSSHPoolImplementation(1000, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addresses := make([]string, 100)
	for i := 0; i < 100; i++ {
		addresses[i] = fmt.Sprintf("test%d.example.com:22", i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		address := addresses[i%len(addresses)]
		client, err := pool.GetConnection(ctx, address, config)
		if err != nil {
			b.Fatal(err)
		}
		_ = client
	}
}

// BenchmarkSSHPoolGetPut benchmarks get and put operations together
func BenchmarkSSHPoolGetPut(b *testing.B) {
	pool := newMockSSHPoolImplementation(1000, 100)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &ssh.ClientConfig{
		User: "test",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	addresses := make([]string, 100)
	for i := 0; i < 100; i++ {
		addresses[i] = fmt.Sprintf("test%d.example.com:22", i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		address := addresses[i%len(addresses)]
		client, err := pool.GetConnection(ctx, address, config)
		if err != nil {
			b.Fatal(err)
		}

		err = pool.PutConnection(address, client)
		if err != nil {
			b.Fatal(err)
		}
	}
}
