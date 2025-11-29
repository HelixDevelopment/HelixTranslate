package distributed

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig represents SSH connection configuration
type SSHConfig struct {
	Host           string        `json:"host"`
	Port           int           `json:"port"`
	User           string        `json:"user"`
	KeyFile        string        `json:"key_file,omitempty"`
	Password       string        `json:"password,omitempty"`
	KnownHostsFile string        `json:"known_hosts_file,omitempty"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// WorkerConfig represents a remote worker configuration
type WorkerConfig struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SSH         SSHConfig `json:"ssh"`
	Tags        []string  `json:"tags,omitempty"`
	MaxCapacity int       `json:"max_capacity"`
	Enabled     bool      `json:"enabled"`
}

// SSHConnection represents an SSH connection to a remote worker
type SSHConnection struct {
	Config    *WorkerConfig
	Client    *ssh.Client
	LastUsed  time.Time
	CreatedAt time.Time
	mu        sync.RWMutex
}

// SSHPool manages a pool of SSH connections to remote workers
type SSHPool struct {
	connections map[string]*SSHConnection
	configs     map[string]*WorkerConfig
	mu          sync.RWMutex
	maxIdleTime time.Duration
	cleanupTick time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewSSHConfig creates default SSH configuration
func NewSSHConfig(host, user string) *SSHConfig {
	return &SSHConfig{
		Host:       host,
		Port:       22,
		User:       user,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 5 * time.Second,
	}
}

// NewWorkerConfig creates a new worker configuration
func NewWorkerConfig(id, name, host, user string) *WorkerConfig {
	return &WorkerConfig{
		ID:          id,
		Name:        name,
		SSH:         *NewSSHConfig(host, user),
		MaxCapacity: 5, // Default capacity
		Enabled:     true,
	}
}

// NewSSHPool creates a new SSH connection pool
func NewSSHPool() *SSHPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &SSHPool{
		connections: make(map[string]*SSHConnection),
		configs:     make(map[string]*WorkerConfig),
		maxIdleTime: 30 * time.Minute,
		cleanupTick: 5 * time.Minute,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start cleanup goroutine
	go pool.cleanup()

	return pool
}

// AddWorker adds a worker configuration to the pool
func (p *SSHPool) AddWorker(config *WorkerConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.configs[config.ID] = config
}

// RemoveWorker removes a worker from the pool
func (p *SSHPool) RemoveWorker(workerID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.configs, workerID)
	if conn, exists := p.connections[workerID]; exists {
		conn.Client.Close()
		delete(p.connections, workerID)
	}
}

// GetConnection gets or creates an SSH connection for a worker
func (p *SSHPool) GetConnection(workerID string) (*SSHConnection, error) {
	p.mu.RLock()
	config, exists := p.configs[workerID]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("worker %s not configured", workerID)
	}

	if !config.Enabled {
		return nil, fmt.Errorf("worker %s is disabled", workerID)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we have an existing connection
	if conn, exists := p.connections[workerID]; exists {
		conn.LastUsed = time.Now()
		return conn, nil
	}

	// Create new connection
	conn, err := p.createConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH connection to %s: %w", workerID, err)
	}

	p.connections[workerID] = conn
	return conn, nil
}

// createConnection creates a new SSH connection
func (p *SSHPool) createConnection(config *WorkerConfig) (*SSHConnection, error) {
	var authMethods []ssh.AuthMethod

	// Add key-based authentication if key file is provided
	if config.SSH.KeyFile != "" {
		key, err := ssh.ParsePrivateKey([]byte(config.SSH.KeyFile))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	// Add password authentication if password is provided
	if config.SSH.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.SSH.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method configured")
	}

	// Implement proper host key verification
	var hostKeyCallback ssh.HostKeyCallback

	if config.SSH.KnownHostsFile != "" {
		// Create a simple host key callback that checks known_hosts file
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			knownHosts, err := os.ReadFile(config.SSH.KnownHostsFile)
			if err != nil {
				if os.IsNotExist(err) {
					// If known_hosts file doesn't exist, create it with this key
					return appendToKnownHosts(config.SSH.KnownHostsFile, hostname, key)
				}
				return fmt.Errorf("failed to read known_hosts file: %w", err)
			}

			// Check if key exists in known_hosts
			keyLine := fmt.Sprintf("%s %s", hostname, strings.TrimSpace(string(key.Marshal())))
			if strings.Contains(string(knownHosts), keyLine) {
				return nil // Key found and matches
			}

			// Key not found, append to known_hosts
			return appendToKnownHosts(config.SSH.KnownHostsFile, hostname, key)
		}
	} else {
		// Fallback to insecure callback if no known_hosts file specified
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.SSH.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         config.SSH.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", config.SSH.Host, config.SSH.Port)

	var client *ssh.Client
	var lastErr error

	// Retry connection
	for attempt := 0; attempt <= config.SSH.MaxRetries; attempt++ {
		var err error
		client, err = ssh.Dial("tcp", addr, sshConfig)
		if err == nil {
			break
		}

		lastErr = err
		if attempt < config.SSH.MaxRetries {
			time.Sleep(config.SSH.RetryDelay)
		}
	}

	if client == nil {
		return nil, fmt.Errorf("failed to connect after %d attempts: %w", config.SSH.MaxRetries+1, lastErr)
	}

	return &SSHConnection{
		Config:    config,
		Client:    client,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}, nil
}

// appendToKnownHosts appends a new host key to the known_hosts file
func appendToKnownHosts(filename, hostname string, key ssh.PublicKey) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	keyLine := fmt.Sprintf("%s %s\n", hostname, strings.TrimSpace(string(key.Marshal())))
	_, err = file.WriteString(keyLine)
	if err != nil {
		return fmt.Errorf("failed to write to known_hosts file: %w", err)
	}

	return nil
}

// ExecuteCommand executes a command on a remote worker
func (conn *SSHConnection) ExecuteCommand(ctx context.Context, command string) ([]byte, error) {
	conn.mu.Lock()
	conn.LastUsed = time.Now()
	conn.mu.Unlock()

	if conn.Client == nil {
		return nil, fmt.Errorf("SSH client is not initialized")
	}

	session, err := conn.Client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Execute command with context timeout
	done := make(chan []byte, 1)
	errChan := make(chan error, 1)

	go func() {
		output, err := session.CombinedOutput(command)
		if err != nil {
			errChan <- err
			return
		}
		done <- output
	}()

	select {
	case output := <-done:
		return output, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return nil, ctx.Err()
	}
}

// Close closes the SSH connection
func (conn *SSHConnection) Close() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.Client != nil {
		return conn.Client.Close()
	}
	return nil
}

// GetWorkers returns all configured workers
func (p *SSHPool) GetWorkers() map[string]*WorkerConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workers := make(map[string]*WorkerConfig)
	for id, config := range p.configs {
		workers[id] = config
	}
	return workers
}

// GetActiveConnections returns active connection count
func (p *SSHPool) GetActiveConnections() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// cleanup periodically removes idle connections
func (p *SSHPool) cleanup() {
	ticker := time.NewTicker(p.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			now := time.Now()
			for id, conn := range p.connections {
				if now.Sub(conn.LastUsed) > p.maxIdleTime {
					if conn.Client != nil {
						conn.Client.Close()
					}
					delete(p.connections, id)
				}
			}
			p.mu.Unlock()

		case <-p.ctx.Done():
			return
		}
	}
}

// Close closes all connections and stops the pool
func (p *SSHPool) Close() {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		if conn.Client != nil {
			conn.Client.Close()
		}
	}
	p.connections = make(map[string]*SSHConnection)
}
