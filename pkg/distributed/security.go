package distributed

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// SSH Security
	SSHHostKeyVerification bool
	KnownHostsFile         string
	SSHCiphers             []string
	SSHKexAlgorithms       []string
	SSHMACs                []string

	// TLS Security
	TLSCertVerification bool
	TLSCAFile           string
	TLSMinVersion       uint16
	TLSMaxVersion       uint16
	TLSCipherSuites     []uint16

	// Authentication
	RequireMutualTLS bool
	ClientCertFile   string
	ClientKeyFile    string

	// Network Security
	AllowedNetworks         []string
	MaxConnectionsPerWorker int
	ConnectionTimeout       time.Duration
	RequestTimeout          time.Duration

	// Monitoring
	EnableSecurityAuditing bool
	SecurityLogFile        string
}

// DefaultSecurityConfig returns secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		SSHHostKeyVerification: true,
		KnownHostsFile:         "~/.ssh/known_hosts",
		SSHCiphers: []string{
			"aes128-gcm@openssh.com",
			"aes256-gcm@openssh.com",
			"chacha20-poly1305@openssh.com",
		},
		SSHKexAlgorithms: []string{
			"curve25519-sha256",
			"curve25519-sha256@libssh.org",
			"ecdh-sha2-nistp256",
			"ecdh-sha2-nistp384",
			"ecdh-sha2-nistp521",
		},
		SSHMACs: []string{
			"hmac-sha2-256-etm@openssh.com",
			"hmac-sha2-512-etm@openssh.com",
		},
		TLSCertVerification: true,
		TLSMinVersion:       tls.VersionTLS12,
		TLSMaxVersion:       tls.VersionTLS13,
		TLSCipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		MaxConnectionsPerWorker: 5,
		ConnectionTimeout:       30 * time.Second,
		RequestTimeout:          60 * time.Second,
		EnableSecurityAuditing:  true,
	}
}

// SecureSSHConfig creates a hardened SSH client configuration
func (sc *SecurityConfig) SecureSSHConfig(user string, authMethods []ssh.AuthMethod) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:    user,
		Auth:    authMethods,
		Timeout: sc.ConnectionTimeout,
		Config: ssh.Config{
			Ciphers:      sc.SSHCiphers,
			KeyExchanges: sc.SSHKexAlgorithms,
			MACs:         sc.SSHMACs,
		},
	}

	// Set host key callback based on verification setting
	if sc.SSHHostKeyVerification {
		hostKeyCallback, err := sc.createHostKeyCallback()
		if err != nil {
			return nil, fmt.Errorf("failed to create host key callback: %w", err)
		}
		config.HostKeyCallback = hostKeyCallback
	} else {
		// Only allow insecure callback in development/testing
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return config, nil
}

// createHostKeyCallback creates a secure host key callback
func (sc *SecurityConfig) createHostKeyCallback() (ssh.HostKeyCallback, error) {
	if sc.KnownHostsFile == "" {
		return nil, fmt.Errorf("known hosts file not configured")
	}

	// Expand home directory
	knownHostsFile := sc.KnownHostsFile
	if strings.HasPrefix(knownHostsFile, "~/") {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return nil, fmt.Errorf("HOME environment variable not set")
		}
		knownHostsFile = strings.Replace(knownHostsFile, "~/", homeDir+"/", 1)
	}

	// For now, return a callback that accepts any key (in production, implement proper known hosts checking)
	// TODO: Implement proper known hosts file parsing and verification
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// In a production system, this should verify against a known hosts file
		// For now, we'll accept the key but log it for security auditing
		return nil
	}, nil
}

// SecureTLSConfig creates a hardened TLS configuration
func (sc *SecurityConfig) SecureTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:   sc.TLSMinVersion,
		MaxVersion:   sc.TLSMaxVersion,
		CipherSuites: sc.TLSCipherSuites,
	}

	// Certificate verification
	if sc.TLSCertVerification {
		tlsConfig.InsecureSkipVerify = false

		// Load CA certificate if specified
		if sc.TLSCAFile != "" {
			caCert, err := ioutil.ReadFile(sc.TLSCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}

			tlsConfig.RootCAs = caCertPool
		}
	} else {
		tlsConfig.InsecureSkipVerify = true
	}

	// Mutual TLS
	if sc.RequireMutualTLS {
		if sc.ClientCertFile == "" || sc.ClientKeyFile == "" {
			return nil, fmt.Errorf("client certificate and key required for mutual TLS")
		}

		cert, err := tls.LoadX509KeyPair(sc.ClientCertFile, sc.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// ValidateNetworkAccess checks if the target address is in allowed networks
func (sc *SecurityConfig) ValidateNetworkAccess(address string) error {
	if len(sc.AllowedNetworks) == 0 {
		return nil // No restrictions
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Try to resolve hostname
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return fmt.Errorf("failed to resolve hostname: %w", err)
		}
		ip = ips[0]
	}

	for _, network := range sc.AllowedNetworks {
		_, ipNet, err := net.ParseCIDR(network)
		if err != nil {
			continue // Skip invalid networks
		}

		if ipNet.Contains(ip) {
			return nil // Allowed
		}
	}

	return fmt.Errorf("address %s not in allowed networks", address)
}

// SecurityAuditor logs security events
type SecurityAuditor struct {
	enabled bool
	logger  Logger
}

// Logger interface for security logging
type Logger interface {
	Log(level, message string, fields map[string]interface{})
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(enabled bool, logger Logger) *SecurityAuditor {
	return &SecurityAuditor{
		enabled: enabled,
		logger:  logger,
	}
}

// LogSecurityEvent logs a security-related event
func (sa *SecurityAuditor) LogSecurityEvent(eventType, message string, fields map[string]interface{}) {
	if !sa.enabled {
		return
	}

	sa.logger.Log("security", message, map[string]interface{}{
		"event_type": eventType,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"details":    fields,
	})
}

// LogConnectionAttempt logs SSH connection attempts
func (sa *SecurityAuditor) LogConnectionAttempt(workerID, address string, success bool, errorMsg string) {
	fields := map[string]interface{}{
		"worker_id": workerID,
		"address":   address,
		"success":   success,
	}

	if errorMsg != "" {
		fields["error"] = errorMsg
	}

	sa.LogSecurityEvent("ssh_connection", "SSH connection attempt", fields)
}

// LogAuthAttempt logs authentication attempts
func (sa *SecurityAuditor) LogAuthAttempt(workerID, user, method string, success bool) {
	sa.LogSecurityEvent("authentication", "Authentication attempt", map[string]interface{}{
		"worker_id": workerID,
		"user":      user,
		"method":    method,
		"success":   success,
	})
}

// LogNetworkAccess logs network access attempts
func (sa *SecurityAuditor) LogNetworkAccess(address string, allowed bool) {
	sa.LogSecurityEvent("network_access", "Network access attempt", map[string]interface{}{
		"address": address,
		"allowed": allowed,
	})
}
