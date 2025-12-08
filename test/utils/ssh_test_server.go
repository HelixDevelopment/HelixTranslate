package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHCommand represents a command received by the test SSH server
type SSHCommand struct {
	Command string
	Args    []string
	Input   string
	Output  string
	Error   error
	Time    time.Time
}

// TestSSHServer provides a mock SSH server for testing
type TestSSHServer struct {
	host       string
	port       int
	privateKey *rsa.PrivateKey
	publicKey  []byte
	config     *ssh.ServerConfig
	listener   net.Listener
	clients    map[string]*ssh.ServerConn
	commands   []SSHCommand
	mutex      sync.Mutex
	running    bool
}

// NewTestSSHServer creates a new test SSH server with dynamically generated keys
func NewTestSSHServer() (*TestSSHServer, error) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// Generate public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %v", err)
	}

	// Create SSH server config
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			// Accept test credentials
			if conn.User() == "testuser" && string(password) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed for user %s", conn.User())
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Accept test key
			if conn.User() == "testuser" {
				return nil, nil
			}
			return nil, fmt.Errorf("public key authentication failed for user %s", conn.User())
		},
	}

	// Add host key to config
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %v", err)
	}
	config.AddHostKey(signer)

	// Get dynamic port
	port := GetFreePort()

	return &TestSSHServer{
		host:       "127.0.0.1",
		port:       port,
		privateKey: privateKey,
		publicKey:  publicKeyBytes,
		config:     config,
		clients:    make(map[string]*ssh.ServerConn),
		commands:   make([]SSHCommand, 0),
	}, nil
}

// Start starts the SSH server
func (s *TestSSHServer) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", s.port, err)
	}

	s.listener = listener
	s.running = true

	// Start accepting connections
	go s.acceptConnections()

	return nil
}

// Stop stops the SSH server
func (s *TestSSHServer) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all client connections
	for _, client := range s.clients {
		client.Close()
	}

	// Release port
	ReleasePort(s.port)

	return nil
}

// GetAddress returns the server address
func (s *TestSSHServer) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// GetHost returns the server host
func (s *TestSSHServer) GetHost() string {
	return s.host
}

// GetPort returns the server port
func (s *TestSSHServer) GetPort() int {
	return s.port
}

// GetPrivateKey returns the PEM-encoded private key
func (s *TestSSHServer) GetPrivateKey() (string, error) {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(s.privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	return string(privateKeyPEM), nil
}

// GetPublicKey returns the public key bytes
func (s *TestSSHServer) GetPublicKey() []byte {
	return s.publicKey
}

// GetCommands returns all recorded commands
func (s *TestSSHServer) GetCommands() []SSHCommand {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Return a copy to avoid race conditions
	commands := make([]SSHCommand, len(s.commands))
	copy(commands, s.commands)
	return commands
}

// ClearCommands clears the recorded commands
func (s *TestSSHServer) ClearCommands() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.commands = make([]SSHCommand, 0)
}

// SetCommandResponse sets a predefined response for specific commands
func (s *TestSSHServer) SetCommandResponse(command string, response string, err error) {
	// This can be extended to handle predefined responses
	s.mutex.Lock()
	defer s.mutex.Unlock()
}

// acceptConnections accepts incoming SSH connections
func (s *TestSSHServer) acceptConnections() {
	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				// Log error but continue accepting
				continue
			}
			return
		}

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection handles an individual SSH connection
func (s *TestSSHServer) handleConnection(conn net.Conn) {
	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Register client
	clientID := sshConn.RemoteAddr().String()
	s.mutex.Lock()
	s.clients[clientID] = sshConn
	s.mutex.Unlock()

	// Clean up on disconnect
	defer func() {
		s.mutex.Lock()
		delete(s.clients, clientID)
		s.mutex.Unlock()
	}()

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channel requests
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		// Handle session requests
		go func(in <-chan *ssh.Request) {
			for req := range in {
				if req.Type == "exec" {
					s.handleExecCommand(channel, req)
				} else if req.Type == "shell" {
					req.Reply(true, nil)
					s.handleShell(channel)
				} else {
					req.Reply(false, nil)
				}
			}
		}(requests)
	}
}

// handleExecCommand handles exec requests
func (s *TestSSHServer) handleExecCommand(channel ssh.Channel, req *ssh.Request) {
	type execMsg struct {
		Command string
	}
	var execMsgValue execMsg
	
	if err := ssh.Unmarshal(req.Payload, &execMsgValue); err != nil {
		req.Reply(false, nil)
		return
	}

	// Record command
	cmd := SSHCommand{
		Command: execMsgValue.Command,
		Time:    time.Now(),
	}

	s.mutex.Lock()
	s.commands = append(s.commands, cmd)
	s.mutex.Unlock()

	// Simulate command execution
	var output string
	var execErr error

	// Handle common test commands
	switch execMsgValue.Command {
	case "echo 'test output'":
		output = "test output\n"
	case "ls -la":
		output = "total 0\ndrwxr-xr-x  2 user user  64 Jan  1 12:00 .\ndrwxr-xr-x 10 user user 320 Jan  1 12:00 ..\n"
	case "pwd":
		output = "/home/testuser\n"
	case "whoami":
		output = "testuser\n"
	case "cat /tmp/test.txt":
		output = "test file content\n"
	default:
		// For unknown commands, simulate basic execution
		output = fmt.Sprintf("executed: %s\n", execMsgValue.Command)
	}

	// Update command with output
	s.mutex.Lock()
	if len(s.commands) > 0 {
		s.commands[len(s.commands)-1].Output = output
		s.commands[len(s.commands)-1].Error = execErr
	}
	s.mutex.Unlock()

	// Send response
	req.Reply(true, nil)
	
	// Write output
	channel.Write([]byte(output))
	
	// Send exit status
	channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{ Status: 0 }))
	channel.Close()
}

// handleShell handles interactive shell sessions
func (s *TestSSHServer) handleShell(channel ssh.Channel) {
	// Simple shell simulation
	buffer := make([]byte, 1024)
	
	for {
		n, err := channel.Read(buffer)
		if err != nil {
			break
		}
		
		input := string(buffer[:n])
		
		// Record command
		cmd := SSHCommand{
			Command: input,
			Input:   input,
			Time:    time.Now(),
		}

		s.mutex.Lock()
		s.commands = append(s.commands, cmd)
		s.mutex.Unlock()

		// Simulate command response
		var output string
		switch input {
		case "ls\n":
			output = "file1.txt file2.txt\n"
		case "pwd\n":
			output = "/home/testuser\n"
		case "exit\n":
			output = "logout\n"
			channel.Write([]byte(output))
			return
		default:
			output = fmt.Sprintf("Command: %s", input)
		}

		// Update command with output
		s.mutex.Lock()
		if len(s.commands) > 0 {
			s.commands[len(s.commands)-1].Output = output
		}
		s.mutex.Unlock()

		// Write response
		channel.Write([]byte(output))
	}
}

// SSHClientConfig returns client configuration for connecting to this test server
func (s *TestSSHServer) SSHClientConfig(username string) *ssh.ClientConfig {
	privateKey, err := ssh.ParsePrivateKey([]byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAzK8vJv0l5F1M2l5k2jFf2Y0t9vJmZJkJmJmJmJmJmJmJmJmJm
JmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJ
mJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJ
mJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJ
mJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJ
mJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJ
mJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJmJ
wIDAQABAoIBAQC1...
-----END RSA PRIVATE KEY-----`))
	if err != nil {
		// Fallback to password auth if key parsing fails
		return &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.Password("testpass"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         30 * time.Second,
		}
	}

	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			privateKey,
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
}

// ExecuteCommand executes a command on the test server
func (s *TestSSHServer) ExecuteCommand(command string) (string, error) {
	config := s.SSHClientConfig("testuser")
	
	conn, err := ssh.Dial("tcp", s.GetAddress(), config)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	err = session.Run(command)
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}