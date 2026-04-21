package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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

	// Encode public key to PEM format
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Create SSH server config
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			// Accept any password for testing
			if conn.User() == "testuser" && string(password) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", conn.User())
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// Accept any public key for testing
			if conn.User() == "testuser" {
				return nil, nil
			}
			return nil, fmt.Errorf("public key rejected for %q", conn.User())
		},
	}

	// Add host key
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %v", err)
	}
	config.AddHostKey(signer)

	return &TestSSHServer{
		host:       "localhost",
		port:       GetFreePort(),
		privateKey: privateKey,
		publicKey:  publicKeyPEM,
		config:     config,
		clients:    make(map[string]*ssh.ServerConn),
		commands:   make([]SSHCommand, 0),
	}, nil
}

// Start starts the SSH test server
func (s *TestSSHServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.host, s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on %s:%d: %v", s.host, s.port, err)
	}

	s.listener = listener
	s.running = true

	// Start accepting connections
	go func() {
		for s.running {
			conn, err := listener.Accept()
			if err != nil {
				if s.running {
					fmt.Printf("Error accepting connection: %v\n", err)
				}
				continue
			}

			// Handle connection in goroutine
			go s.handleConnection(conn)
		}
	}()

	return nil
}

// Stop stops the SSH test server
func (s *TestSSHServer) Stop() {
	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}
	ReleasePort(s.port)
}

// handleConnection handles an incoming SSH connection
func (s *TestSSHServer) handleConnection(conn net.Conn) {
	// Convert to SSH connection
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Store client
	clientID := fmt.Sprintf("%s:%s", sshConn.RemoteAddr().Network(), sshConn.RemoteAddr().String())
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

		// Handle session requests
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "exec":
					s.handleExec(channel, req)
				case "shell":
					s.handleShell(channel)
					req.Reply(true, nil)
				default:
					req.Reply(false, nil)
				}
			}
		}(requests)
	}
}

// handleExec handles exec requests
func (s *TestSSHServer) handleExec(channel ssh.Channel, req *ssh.Request) {
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
	channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{Status: 0}))
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

		// Echo back input with prefix
		response := fmt.Sprintf("shell: %s", string(buffer[:n]))
		channel.Write([]byte(response))
	}
}

// SSHClientConfig returns client configuration for connecting to this test server
func (s *TestSSHServer) SSHClientConfig(username string) *ssh.ClientConfig {
	// Use the generated private key from the server
	signer, err := ssh.NewSignerFromKey(s.privateKey)
	if err != nil {
		// Fallback to password auth if key signing fails
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
			ssh.PublicKeys(signer),
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

// GetAddress returns the server address
func (s *TestSSHServer) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// GetPort returns the server port
func (s *TestSSHServer) GetPort() int {
	return s.port
}

// GetCommands returns the list of executed commands
func (s *TestSSHServer) GetCommands() []SSHCommand {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	commands := make([]SSHCommand, len(s.commands))
	copy(commands, s.commands)
	return commands
}

// ClearCommands clears the command history
func (s *TestSSHServer) ClearCommands() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.commands = make([]SSHCommand, 0)
}
