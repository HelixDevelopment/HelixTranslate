package deployment

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHDeployer handles SSH-based deployment of Docker containers
type SSHDeployer struct {
	logger *log.Logger
}

// NewSSHDeployer creates a new SSH deployer
func NewSSHDeployer(logger *log.Logger) *SSHDeployer {
	return &SSHDeployer{
		logger: logger,
	}
}

// DeployInstance deploys a Docker container instance via SSH
func (sd *SSHDeployer) DeployInstance(ctx context.Context, config *DeploymentConfig) (string, error) {
	sd.logger.Printf("Deploying instance %s to %s@%s", config.ContainerName, config.User, config.Host)

	// Establish SSH connection
	client, err := sd.connectSSH(config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH: %w", err)
	}
	defer client.Close()

	// Ensure Docker is installed and running
	if err := sd.ensureDocker(ctx, client); err != nil {
		return "", fmt.Errorf("failed to ensure Docker: %w", err)
	}

	// Pull Docker image
	if err := sd.pullImage(ctx, client, config.DockerImage); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Stop and remove existing container if it exists
	if err := sd.cleanupExistingContainer(ctx, client, config.ContainerName); err != nil {
		sd.logger.Printf("Warning: failed to cleanup existing container: %v", err)
	}

	// Create and start container
	containerID, err := sd.createAndStartContainer(ctx, client, config)
	if err != nil {
		return "", fmt.Errorf("failed to create and start container: %w", err)
	}

	sd.logger.Printf("Successfully deployed container %s", containerID[:12])
	return containerID, nil
}

// connectSSH establishes SSH connection
func (sd *SSHDeployer) connectSSH(config *DeploymentConfig) (*ssh.Client, error) {
	// Read private key
	key, err := os.ReadFile(config.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// SSH client config
	sshConfig := &ssh.ClientConfig{
		User: config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	// Connect
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", config.Host), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return client, nil
}

// ensureDocker ensures Docker is installed and running
func (sd *SSHDeployer) ensureDocker(ctx context.Context, client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Check if Docker is installed
	if err := session.Run("which docker"); err != nil {
		return fmt.Errorf("Docker is not installed on remote host")
	}

	// Start Docker service if not running
	session, err = client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Check Docker daemon status
	if err := session.Run("systemctl is-active --quiet docker"); err != nil {
		// Try to start Docker
		session, err = client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		if err := session.Run("sudo systemctl start docker"); err != nil {
			return fmt.Errorf("failed to start Docker service: %w", err)
		}
	}

	return nil
}

// pullImage pulls the Docker image
func (sd *SSHDeployer) pullImage(ctx context.Context, client *ssh.Client, image string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("docker pull %s", image)
	return session.Run(cmd)
}

// cleanupExistingContainer stops and removes existing container
func (sd *SSHDeployer) cleanupExistingContainer(ctx context.Context, client *ssh.Client, containerName string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Stop container if running
	cmd := fmt.Sprintf("docker stop %s 2>/dev/null || true", containerName)
	if err := session.Run(cmd); err != nil {
		sd.logger.Printf("Warning: failed to stop container %s: %v", containerName, err)
	}

	session, err = client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Remove container
	cmd = fmt.Sprintf("docker rm %s 2>/dev/null || true", containerName)
	if err := session.Run(cmd); err != nil {
		sd.logger.Printf("Warning: failed to remove container %s: %v", containerName, err)
	}

	return nil
}

// createAndStartContainer creates and starts the Docker container
func (sd *SSHDeployer) createAndStartContainer(ctx context.Context, client *ssh.Client, config *DeploymentConfig) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// Build Docker run command
	cmd := sd.buildDockerRunCommand(config)

	sd.logger.Printf("Running Docker command: %s", cmd)

	// Execute command and capture output
	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("docker run failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Extract container ID from output
	containerID := strings.TrimSpace(stdout.String())
	if containerID == "" {
		return "", fmt.Errorf("no container ID returned")
	}

	return containerID, nil
}

// buildDockerRunCommand builds the Docker run command
func (sd *SSHDeployer) buildDockerRunCommand(config *DeploymentConfig) string {
	var cmd strings.Builder
	cmd.WriteString("docker run -d --name ")
	cmd.WriteString(config.ContainerName)

	// Add restart policy
	if config.RestartPolicy != "" {
		cmd.WriteString(" --restart ")
		cmd.WriteString(config.RestartPolicy)
	} else {
		cmd.WriteString(" --restart unless-stopped")
	}

	// Add networks
	for _, network := range config.Networks {
		cmd.WriteString(" --network ")
		cmd.WriteString(network)
	}

	// Add port mappings
	for _, port := range config.Ports {
		cmd.WriteString(fmt.Sprintf(" -p %d:%d", port.HostPort, port.ContainerPort))
		if port.Protocol != "" && port.Protocol != "tcp" {
			cmd.WriteString("/")
			cmd.WriteString(port.Protocol)
		}
	}

	// Add volume mappings
	for _, volume := range config.Volumes {
		cmd.WriteString(" -v ")
		cmd.WriteString(volume.HostPath)
		cmd.WriteString(":")
		cmd.WriteString(volume.ContainerPath)
		if volume.ReadOnly {
			cmd.WriteString(":ro")
		}
	}

	// Add environment variables
	for key, value := range config.Environment {
		cmd.WriteString(" -e ")
		cmd.WriteString(key)
		cmd.WriteString("=\"")
		cmd.WriteString(value)
		cmd.WriteString("\"")
	}

	// Add health check if configured
	if config.HealthCheck != nil {
		cmd.WriteString(" --health-cmd=\"")
		cmd.WriteString(strings.Join(config.HealthCheck.Test, " "))
		cmd.WriteString("\"")

		if config.HealthCheck.Interval > 0 {
			cmd.WriteString(fmt.Sprintf(" --health-interval=%s", config.HealthCheck.Interval))
		}
		if config.HealthCheck.Timeout > 0 {
			cmd.WriteString(fmt.Sprintf(" --health-timeout=%s", config.HealthCheck.Timeout))
		}
		if config.HealthCheck.Retries > 0 {
			cmd.WriteString(fmt.Sprintf(" --health-retries=%d", config.HealthCheck.Retries))
		}
		if config.HealthCheck.StartPeriod > 0 {
			cmd.WriteString(fmt.Sprintf(" --health-start-period=%s", config.HealthCheck.StartPeriod))
		}
	}

	// Add image name
	cmd.WriteString(" ")
	cmd.WriteString(config.DockerImage)

	return cmd.String()
}

// CheckInstanceHealth checks the health of a deployed instance
func (sd *SSHDeployer) CheckInstanceHealth(ctx context.Context, host string, port int) (bool, error) {
	url := fmt.Sprintf("http://%s:%d/health", host, port)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// GetContainerLogs retrieves logs from a container
func (sd *SSHDeployer) GetContainerLogs(ctx context.Context, config *DeploymentConfig, lines int) (string, error) {
	client, err := sd.connectSSH(config)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	cmd := fmt.Sprintf("docker logs --tail %d %s", lines, config.ContainerName)

	var output strings.Builder
	session.Stdout = &output
	session.Stderr = &output

	if err := session.Run(cmd); err != nil {
		return "", err
	}

	return output.String(), nil
}

// StopInstance stops a deployed instance
func (sd *SSHDeployer) StopInstance(ctx context.Context, config *DeploymentConfig) error {
	client, err := sd.connectSSH(config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("docker stop %s", config.ContainerName)
	return session.Run(cmd)
}

// RemoveInstance removes a deployed instance
func (sd *SSHDeployer) RemoveInstance(ctx context.Context, config *DeploymentConfig) error {
	client, err := sd.connectSSH(config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("docker rm %s", config.ContainerName)
	return session.Run(cmd)
}

// UpdateInstance updates a deployed instance to a new image version
func (sd *SSHDeployer) UpdateInstance(ctx context.Context, config *DeploymentConfig) (string, error) {
	sd.logger.Printf("Updating instance %s to image %s...", config.ContainerName, config.DockerImage)

	client, err := sd.connectSSH(config)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// Stop the existing container
	stopCmd := fmt.Sprintf("docker stop %s", config.ContainerName)
	if err := session.Run(stopCmd); err != nil {
		return "", fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the existing container
	removeCmd := fmt.Sprintf("docker rm %s", config.ContainerName)
	if err := session.Run(removeCmd); err != nil {
		return "", fmt.Errorf("failed to remove container: %w", err)
	}

	// Pull the new image
	pullCmd := fmt.Sprintf("docker pull %s", config.DockerImage)
	session2, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session2.Close()

	if err := session2.Run(pullCmd); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	// Start the new container
	runCmd := sd.buildDockerRunCommand(config)
	session3, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session3.Close()

	var output strings.Builder
	session3.Stdout = &output

	if err := session3.Run(runCmd); err != nil {
		return "", fmt.Errorf("failed to start updated container: %w", err)
	}

	containerID := strings.TrimSpace(output.String())
	sd.logger.Printf("Instance %s updated successfully: %s", config.ContainerName, containerID[:12])

	return containerID, nil
}

// RestartInstance restarts a deployed instance
func (sd *SSHDeployer) RestartInstance(ctx context.Context, config *DeploymentConfig) error {
	sd.logger.Printf("Restarting instance %s...", config.ContainerName)

	client, err := sd.connectSSH(config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("docker restart %s", config.ContainerName)
	return session.Run(cmd)
}

// Close closes the SSH deployer
func (sd *SSHDeployer) Close() error {
	// No persistent connections to close
	return nil
}
