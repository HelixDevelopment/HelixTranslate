package sshworker

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/version"
	"golang.org/x/crypto/ssh"
)

// SSHWorker manages remote SSH connections and execution
type SSHWorker struct {
	host       string
	username   string
	password   string
	privateKey string
	port       int
	client     *ssh.Client
	logger     logger.Logger
	config     SSHWorkerConfig
}

// SSHWorkerConfig holds configuration for SSH worker
type SSHWorkerConfig struct {
	Host              string
	Username          string
	Password          string
	PrivateKeyPath    string
	Port              int
	RemoteDir         string
	ConnectionTimeout time.Duration
	CommandTimeout    time.Duration
}

// NewSSHWorker creates a new SSH worker
func NewSSHWorker(config SSHWorkerConfig, logger logger.Logger) (*SSHWorker, error) {
	worker := &SSHWorker{
		host:     config.Host,
		username: config.Username,
		password: config.Password,
		port:     config.Port,
		logger:   logger,
		config:   config,
	}
	
	return worker, nil
}

// Connect establishes SSH connection to remote worker
func (w *SSHWorker) Connect(ctx context.Context) error {
	var authMethods []ssh.AuthMethod

	// Try private key first
	if keyPath := os.Getenv("SSH_PRIVATE_KEY_PATH"); keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	// Fallback to password
	if len(authMethods) == 0 && w.password != "" {
		authMethods = append(authMethods, ssh.Password(w.password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication method available")
	}

	config := &ssh.ClientConfig{
		User:            w.username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", w.host, w.port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	w.client = client
	return nil
}

// Disconnect closes the SSH connection
func (w *SSHWorker) Disconnect() error {
	if w.client != nil {
		return w.client.Close()
	}
	return nil
}

// ExecuteCommand runs a command on remote worker
func (w *SSHWorker) ExecuteCommand(ctx context.Context, command string) (*CommandResult, error) {
	if w.client == nil {
		return nil, fmt.Errorf("SSH client not connected")
	}

	session, err := w.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Set up pipes for stdout and stderr
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Run command with context
	result := make(chan error, 1)
	go func() {
		result <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return nil, fmt.Errorf("command execution cancelled: %w", ctx.Err())
	case err := <-result:
		if err != nil {
			return &CommandResult{
				ExitCode: 1,
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				Error:     err,
			}, nil
		}

		return &CommandResult{
			ExitCode: 0,
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			Error:     nil,
		}, nil
	}
}

// UploadFile uploads a file to remote worker
func (w *SSHWorker) UploadFile(ctx context.Context, localPath, remotePath string) error {
	if w.client == nil {
		return fmt.Errorf("SSH client not connected")
	}

	session, err := w.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Use SCP to upload file
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Create remote directory if needed
	dir := filepath.Dir(remotePath)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", dir)
	if _, err := w.ExecuteCommand(ctx, mkdirCmd); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Write file content remotely using base64 encoding for binary files (chunked)
	contentBase64 := base64.StdEncoding.EncodeToString(content)
	contentSize := len(contentBase64)
	chunkSize := 50000 // Split into chunks to avoid command line limits
	
	w.logger.Debug("Uploading file in chunks", map[string]interface{}{
		"total_size": contentSize,
		"chunk_size": chunkSize,
		"chunks": (contentSize + chunkSize - 1) / chunkSize,
	})
	
	for i := 0; i < contentSize; i += chunkSize {
		end := i + chunkSize
		if end > contentSize {
			end = contentSize
		}
		
		chunk := contentBase64[i:end]
		var writeCmd string
		
		if i == 0 {
			// First chunk - create file with first part
			writeCmd = fmt.Sprintf("echo '%s' | base64 -d > %s", chunk, remotePath)
		} else {
			// Subsequent chunks - append to file
			writeCmd = fmt.Sprintf("echo '%s' | base64 -d >> %s", chunk, remotePath)
		}
		
		result, err := w.ExecuteCommand(ctx, writeCmd)
		if err != nil {
			return fmt.Errorf("failed to upload file chunk %d: %w", i/chunkSize, err)
		}
		if result.ExitCode != 0 {
			w.logger.Debug("Chunk upload failed", map[string]interface{}{
				"chunk": i / chunkSize,
				"command": writeCmd,
				"stderr": result.Stderr,
				"exit_code": result.ExitCode,
			})
			return fmt.Errorf("upload failed at chunk %d: %s", i/chunkSize, result.Stderr)
		}
	}

	return nil
}

// DownloadFile downloads a file from remote worker
func (w *SSHWorker) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	if w.client == nil {
		return fmt.Errorf("SSH client not connected")
	}

	// Read remote file
	cmd := fmt.Sprintf("cat %s", remotePath)
	result, err := w.ExecuteCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("download failed: %s", result.Stderr)
	}

	// Write to local file
	if err := os.WriteFile(localPath, []byte(result.Stdout), 0644); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

// ensureConnection ensures SSH connection is established
func (w *SSHWorker) ensureConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.ConnectionTimeout)
	defer cancel()
	
	return w.Connect(ctx)
}

// executeCommand is a helper method for command execution
func (w *SSHWorker) executeCommand(ctx context.Context, command string) (*CommandResult, error) {
	return w.ExecuteCommand(ctx, command)
}

// UploadData uploads data content to a remote file
func (w *SSHWorker) UploadData(ctx context.Context, data []byte, remotePath string) error {
	w.logger.Info("Uploading data to remote file", map[string]interface{}{
		"remote_path": remotePath,
		"data_size":   len(data),
	})

	if err := w.ensureConnection(); err != nil {
		return fmt.Errorf("failed to establish connection: %w", err)
	}

	// Create remote directory if needed
	remoteDir := filepath.Dir(remotePath)
	mkdirCmd := fmt.Sprintf("mkdir -p '%s'", remoteDir)
	if _, err := w.executeCommand(ctx, mkdirCmd); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Write data content remotely using base64 encoding for binary safety
	encodedData := base64.StdEncoding.EncodeToString(data)
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s", encodedData, remotePath)
	result, err := w.ExecuteCommand(ctx, writeCmd)
	if err != nil {
		return fmt.Errorf("failed to upload data: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("data upload failed: %s", result.Stderr)
	}

	w.logger.Info("Data uploaded successfully", map[string]interface{}{
		"remote_path": remotePath,
		"data_size":   len(data),
	})

	return nil
}

// SyncCodebase synchronizes the codebase with remote worker
func (w *SSHWorker) SyncCodebase(ctx context.Context, localBasePath string) error {
	// Create a consistent archive using git for cross-platform compatibility
	archivePath := filepath.Join(os.TempDir(), "codebase.tar.gz")
	
	// Initialize git repository if not already initialized
	if _, err := os.Stat(filepath.Join(localBasePath, ".git")); os.IsNotExist(err) {
		initCmd := exec.Command("git", "init")
		initCmd.Dir = localBasePath
		if err := initCmd.Run(); err != nil {
			w.logger.Debug("Git init failed, falling back to tar", map[string]interface{}{
				"error": err,
			})
			return w.syncCodebaseWithTar(ctx, localBasePath, archivePath)
		}
	
	// Add all files and create a commit
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = localBasePath
	if err := addCmd.Run(); err != nil {
		w.logger.Debug("Git add failed", map[string]interface{}{
			"error": err,
		})
		return w.syncCodebaseWithTar(ctx, localBasePath, archivePath)
	}
	}
	
	// Create archive using git archive for consistency
	archiveCmd := exec.Command("git", "archive", "--format=tar.gz", "HEAD", "-o", archivePath)
	archiveCmd.Dir = localBasePath
	if err := archiveCmd.Run(); err != nil {
		w.logger.Debug("Git archive failed", map[string]interface{}{
			"error": err,
		})
		return w.syncCodebaseWithTar(ctx, localBasePath, archivePath)
	}
	
	// Verify archive was created
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("archive creation failed: %w", err)
	}

	// Upload archive to remote
	remoteArchivePath := filepath.Join(w.config.RemoteDir, "codebase.tar.gz")
	if err := w.UploadFile(ctx, archivePath, remoteArchivePath); err != nil {
		return fmt.Errorf("failed to upload codebase archive: %w", err)
	}

	// Extract archive on remote with proper directory setup
	extractCmd := fmt.Sprintf("cd %s && tar -xzf codebase.tar.gz && ls -la && pwd", w.config.RemoteDir)
	result, err := w.ExecuteCommand(ctx, extractCmd)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	if result.ExitCode != 0 {
		w.logger.Debug("Archive extraction failed", map[string]interface{}{
			"command": extractCmd,
			"stderr": result.Stderr,
			"exit_code": result.ExitCode,
		})
		return fmt.Errorf("archive extraction failed: %s", result.Stderr)
	}

	// Verify Go modules work
	testCmd := fmt.Sprintf("cd %s && go version", w.config.RemoteDir)
	testResult, testErr := w.ExecuteCommand(ctx, testCmd)
	if testErr != nil {
		w.logger.Debug("Go test failed", map[string]interface{}{
			"command": testCmd,
			"error": testErr,
		})
	} else if testResult.ExitCode != 0 {
		w.logger.Debug("Go version check failed", map[string]interface{}{
			"command": testCmd,
			"stderr": testResult.Stderr,
		})
	} else {
		w.logger.Info("Go setup verified", map[string]interface{}{
			"version": testResult.Stdout,
		})
	}

	// Clean up local archive
	os.Remove(archivePath)

	return nil
}

// syncCodebaseWithTar falls back to tar-based synchronization
func (w *SSHWorker) syncCodebaseWithTar(ctx context.Context, localBasePath string, archivePath string) error {
	// Create archive locally
	cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(cancelCtx, "tar", "-czf", archivePath, "--format=ustar",
		"--exclude=.git", "--exclude=node_modules", "--exclude=__pycache__",
		"--exclude=*.log", "--exclude=vendor",
		"cmd", "pkg", "internal", "scripts", "docs",
		"Makefile", "Dockerfile", "go.mod", "go.sum")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = localBasePath
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create archive: %v (stderr: %s)", err, stderr.String())
	}
	
	// Verify archive was created
	if _, err := os.Stat(archivePath); err != nil {
		return fmt.Errorf("archive creation failed: %w", err)
	}

	// Upload archive to remote
	remoteArchivePath := filepath.Join(w.config.RemoteDir, "codebase.tar.gz")
	if err := w.UploadFile(ctx, archivePath, remoteArchivePath); err != nil {
		return fmt.Errorf("failed to upload codebase archive: %w", err)
	}

	// Extract archive on remote with proper directory setup
	extractCmd := fmt.Sprintf("cd %s && tar -xzf codebase.tar.gz && ls -la && pwd", w.config.RemoteDir)
	result, err := w.ExecuteCommand(ctx, extractCmd)
	if err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}
	if result.ExitCode != 0 {
		w.logger.Debug("Archive extraction failed", map[string]interface{}{
			"command": extractCmd,
			"stderr": result.Stderr,
			"exit_code": result.ExitCode,
		})
		return fmt.Errorf("archive extraction failed: %s", result.Stderr)
	}

	// Verify Go modules work
	testCmd := fmt.Sprintf("cd %s && go version", w.config.RemoteDir)
	testResult, testErr := w.ExecuteCommand(ctx, testCmd)
	if testErr != nil {
		w.logger.Debug("Go test failed", map[string]interface{}{
			"command": testCmd,
			"error": testErr,
		})
	} else if testResult.ExitCode != 0 {
		w.logger.Debug("Go version check failed", map[string]interface{}{
			"command": testCmd,
			"stderr": testResult.Stderr,
		})
	} else {
		w.logger.Info("Go setup verified", map[string]interface{}{
			"version": testResult.Stdout,
		})
	}

	// Clean up local archive
	os.Remove(archivePath)

	return nil
}

// GetRemoteCodebaseHash retrieves the codebase hash from remote worker
func (w *SSHWorker) GetRemoteCodebaseHash(ctx context.Context) (string, error) {
	// Ensure we have a connection
	if err := w.ensureConnection(); err != nil {
		return "", fmt.Errorf("failed to establish connection: %w", err)
	}

	// Change to the remote directory and verify the binary exists
	checkDirCmd := fmt.Sprintf("cd %s && pwd && ls -la translator", w.config.RemoteDir)
	checkResult, checkErr := w.ExecuteCommand(ctx, checkDirCmd)
	if checkErr != nil {
		w.logger.Debug("Directory check failed", map[string]interface{}{
			"command": checkDirCmd,
			"error": checkErr,
		})
	} else {
		w.logger.Debug("Directory and binary check", map[string]interface{}{
			"command": checkDirCmd,
			"stdout": checkResult.Stdout,
			"stderr": checkResult.Stderr,
		})
	}

	// For minimal setup, check if essential files exist and get a simple hash
	checkFilesCmd := fmt.Sprintf("cd %s && ls -la translator python_translation.sh go.mod 2>/dev/null | sha256sum | cut -d' ' -f1", w.config.RemoteDir)
	result, err := w.ExecuteCommand(ctx, checkFilesCmd)
	if err != nil {
		return "", fmt.Errorf("failed to check essential files: %w", err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("essential files not found on remote")
	}

	hash := strings.TrimSpace(result.Stdout)
	if hash == "" {
		return "", fmt.Errorf("empty hash received from remote")
	}

	return hash, nil
}

// VerifyCodebaseVersion checks if remote worker has the same codebase version
func (w *SSHWorker) VerifyCodebaseVersion(ctx context.Context) (bool, string, string, error) {
	// Calculate local hash
	localHasher := version.NewCodebaseHasher()
	localHash, err := localHasher.CalculateHash()
	if err != nil {
		return false, "", "", fmt.Errorf("failed to calculate local hash: %w", err)
	}

	// Get remote hash
	remoteHash, err := w.GetRemoteCodebaseHash(ctx)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get remote hash: %w", err)
	}

	// Compare versions
	isEqual := localHasher.CompareVersions(localHash, remoteHash)
	
	return isEqual, localHash, remoteHash, nil
}

// UpdateRemoteCodebase updates the remote worker codebase
func (w *SSHWorker) UpdateRemoteCodebase(ctx context.Context, localBasePath string) error {
	// Sync codebase
	if err := w.SyncCodebase(ctx, localBasePath); err != nil {
		return fmt.Errorf("failed to sync codebase: %w", err)
	}

	// Check if binary already exists in remote directory
	checkBinaryCmd := fmt.Sprintf("cd %s && test -f ./translator && echo 'binary_exists'", w.config.RemoteDir)
	checkResult, checkErr := w.ExecuteCommand(ctx, checkBinaryCmd)
	if checkErr != nil {
		return fmt.Errorf("failed to check binary: %w", checkErr)
	}
	
	binaryExists := strings.Contains(checkResult.Stdout, "binary_exists")
	if binaryExists {
		// Binary already exists, but we need to rebuild since source code changed
		w.logger.Info("Remote binary exists but will be rebuilt due to source update", nil)
	}

	// Build on remote (always rebuild after source update)
	buildCmd := fmt.Sprintf("cd %s && go mod tidy && go build -o translator ./cmd/cli", w.config.RemoteDir)
	result, err := w.ExecuteCommand(ctx, buildCmd)
	if err != nil {
		return fmt.Errorf("failed to build on remote: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("remote build failed: %s", result.Stderr)
	}

	return nil
}

// UploadEssentialFiles uploads only the binary and required scripts for faster execution
func (w *SSHWorker) UploadEssentialFiles(ctx context.Context) error {
	// Ensure remote directory exists and clean
	setupCmd := fmt.Sprintf("mkdir -p %s && rm -rf %s/*", w.config.RemoteDir, w.config.RemoteDir)
	_, err := w.ExecuteCommand(ctx, setupCmd)
	if err != nil {
		return fmt.Errorf("failed to setup remote directory: %w", err)
	}

	// Upload the pre-built binary
	binaryPath := "./build/translator-ssh"
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("binary not found at %s: %w", binaryPath, err)
	}
	
	remoteBinaryPath := filepath.Join(w.config.RemoteDir, "translator")
	if err := w.UploadFile(ctx, binaryPath, remoteBinaryPath); err != nil {
		return fmt.Errorf("failed to upload binary: %w", err)
	}

	// Make binary executable
	chmodCmd := fmt.Sprintf("chmod +x %s", remoteBinaryPath)
	_, err = w.ExecuteCommand(ctx, chmodCmd)
	if err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Upload Python translation script
	scriptPath := "./scripts/python_translation.sh"
	if _, err := os.Stat(scriptPath); err == nil {
		remoteScriptPath := filepath.Join(w.config.RemoteDir, "python_translation.sh")
		if err := w.UploadFile(ctx, scriptPath, remoteScriptPath); err != nil {
			return fmt.Errorf("failed to upload Python script: %w", err)
		}

		// Make script executable
		chmodScriptCmd := fmt.Sprintf("chmod +x %s", remoteScriptPath)
		_, err = w.ExecuteCommand(ctx, chmodScriptCmd)
		if err != nil {
			return fmt.Errorf("failed to make script executable: %w", err)
		}
	}

	// Create a minimal go.mod for the binary to work properly
	goModContent := `module digital.vasic.translator

go 1.25
`
	tempGoMod := filepath.Join(os.TempDir(), "go.mod")
	if err := os.WriteFile(tempGoMod, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to create temp go.mod: %w", err)
	}
	defer os.Remove(tempGoMod)

	remoteGoModPath := filepath.Join(w.config.RemoteDir, "go.mod")
	if err := w.UploadFile(ctx, tempGoMod, remoteGoModPath); err != nil {
		return fmt.Errorf("failed to upload go.mod: %w", err)
	}

	return nil
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    error
}

// Success returns true if command executed successfully
func (cr *CommandResult) Success() bool {
	return cr.ExitCode == 0 && cr.Error == nil
}

// Output returns combined stdout and stderr
func (cr *CommandResult) Output() string {
	return cr.Stdout + cr.Stderr
}

// GenerateSSHKey generates a new SSH key pair for authentication
func GenerateSSHKey() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode private key
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return string(privateKeyBytes), string(publicKeyBytes), nil
}

// Close closes the SSH connection and cleans up resources
func (w *SSHWorker) Close() error {
	if w.client != nil {
		return w.client.Close()
	}
	return nil
}

// TestConnection tests the SSH connection
func (w *SSHWorker) TestConnection(ctx context.Context) error {
	if err := w.Connect(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer w.Disconnect()

	// Run a simple command
	result, err := w.ExecuteCommand(ctx, "echo 'connection-test'")
	if err != nil {
		return fmt.Errorf("command execution test failed: %w", err)
	}
	if !result.Success() {
		return fmt.Errorf("command execution returned error: %s", result.Output())
	}

	if !strings.Contains(result.Stdout, "connection-test") {
		return fmt.Errorf("unexpected output: %s", result.Stdout)
	}

	return nil
}