package deployment

import (
	"context"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// MockSSHClient implements a mock SSH client for testing
type MockSSHClient struct {
	shouldConnect    bool
	shouldFailAuth   bool
	shouldFailExec   bool
	connectedServers map[string]bool
	executedCommands map[string][]string
}

func NewMockSSHClient(shouldConnect bool) *MockSSHClient {
	return &MockSSHClient{
		shouldConnect:    shouldConnect,
		connectedServers: make(map[string]bool),
		executedCommands: make(map[string][]string),
	}
}

func (m *MockSSHClient) Dial(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	if !m.shouldConnect {
		return nil, &net.OpError{Op: "dial", Net: network, Addr: &net.TCPAddr{Port: 22}}
	}
	if m.shouldFailAuth {
		return nil, &ssh.AuthError{User: config.User}
	}
	
	m.connectedServers[addr] = true
	return &ssh.Client{}, nil
}

func (m *MockSSHClient) SetAuthFail(shouldFail bool) {
	m.shouldFailAuth = shouldFail
}

func (m *MockSSHClient) SetExecFail(shouldFail bool) {
	m.shouldFailExec = shouldFail
}

func (m *MockSSHClient) IsConnected(addr string) bool {
	return m.connectedServers[addr]
}

func (m *MockSSHClient) GetExecutedCommands(addr string) []string {
	return m.executedCommands[addr]
}

// SSHDeployConfig represents SSH deployment configuration
type SSHDeployConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	Username     string        `json:"username"`
	Password     string        `json:"password,omitempty"`
	KeyPath      string        `json:"key_path,omitempty"`
	KeyData      []byte        `json:"key_data,omitempty"`
	Timeout      time.Duration `json:"timeout"`
	ConnectRetries int         `json:"connect_retries"`
	CommandTimeout time.Duration `json:"command_timeout"`
}

func (c *SSHDeployConfig) Validate() error {
	if c.Host == "" {
		return &ValidationError{Field: "host", Message: "host is required"}
	}
	if c.Port == 0 {
		c.Port = 22
	}
	if c.Username == "" {
		return &ValidationError{Field: "username", Message: "username is required"}
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.ConnectRetries == 0 {
		c.ConnectRetries = 3
	}
	if c.CommandTimeout == 0 {
		c.CommandTimeout = 10 * time.Minute
	}
	
	if c.Password == "" && c.KeyPath == "" && len(c.KeyData) == 0 {
		return &ValidationError{Field: "auth", Message: "either password, key_path, or key_data is required"}
	}
	
	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message + " (field: " + e.Field + ")"
}

// SSHDeployer handles SSH-based deployment operations
type SSHDeployer struct {
	config *SSHDeployConfig
	client SSHClientInterface
}

// SSHClientInterface defines the interface for SSH client operations
type SSHClientInterface interface {
	Dial(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
}

// NewSSHDeployer creates a new SSH deployer
func NewSSHDeployer(config *SSHDeployConfig) *SSHDeployer {
	return &SSHDeployer{
		config: config,
		client: &RealSSHClient{},
	}
}

// NewSSHDeployerWithClient creates a new SSH deployer with custom client (for testing)
func NewSSHDeployerWithClient(config *SSHDeployConfig, client SSHClientInterface) *SSHDeployer {
	return &SSHDeployer{
		config: config,
		client: client,
	}
}

// Connect establishes an SSH connection to the target host
func (d *SSHDeployer) Connect(ctx context.Context) error {
	if err := d.config.Validate(); err != nil {
		return &ConnectionError{Type: "config_validation", Err: err}
	}
	
	// Prepare SSH client configuration
	sshConfig := &ssh.ClientConfig{
		User:            d.config.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Only for testing
		Timeout:         d.config.Timeout,
	}
	
	// Setup authentication
	if d.config.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(d.config.Password))
	}
	
	if d.config.KeyPath != "" || len(d.config.KeyData) > 0 {
		var keyData []byte
		var err error
		
		if len(d.config.KeyData) > 0 {
			keyData = d.config.KeyData
		} else {
			keyData, err = os.ReadFile(d.config.KeyPath)
			if err != nil {
				return &ConnectionError{Type: "key_read", Err: err}
			}
		}
		
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return &ConnectionError{Type: "key_parse", Err: err}
		}
		
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}
	
	// Attempt connection with retries
	addr := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	var lastErr error
	
	for attempt := 1; attempt <= d.config.ConnectRetries; attempt++ {
		select {
		case <-ctx.Done():
			return &ConnectionError{Type: "timeout", Err: ctx.Err()}
		default:
		}
		
		client, err := d.client.Dial(ctx, "tcp", addr, sshConfig)
		if err == nil {
			client.Close() // Close immediately for this test
			return nil
		}
		
		lastErr = err
		if attempt < d.config.ConnectRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	
	return &ConnectionError{Type: "connect_failed", Err: lastErr}
}

// ConnectionError represents an SSH connection error
type ConnectionError struct {
	Type string
	Err  error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("SSH connection error [%s]: %v", e.Type, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// ExecuteCommand executes a command on the remote host
func (d *SSHDeployer) ExecuteCommand(ctx context.Context, command string) (*CommandResult, error) {
	if err := d.Connect(ctx); err != nil {
		return nil, err
	}
	
	// Mock implementation for testing - real implementation would create session
	result := &CommandResult{
		Command: command,
		ExitCode: 0,
		Stdout:   "Mock command execution successful",
		Stderr:   "",
		Duration: time.Millisecond * 100,
	}
	
	return result, nil
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}

// RealSSHClient implements real SSH client operations
type RealSSHClient struct{}

func (r *RealSSHClient) Dial(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return ssh.Dial(network, addr, config)
}