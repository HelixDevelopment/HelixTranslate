package distributed

import (
	"crypto/rand"
	"crypto/rsa"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityConfig_createHostKeyCallback(t *testing.T) {
	// Create a temporary known hosts file
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	// Generate a test SSH key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create public key: %v", err)
	}

	// Write a known hosts entry
	knownHostsContent := "example.com " + string(ssh.MarshalAuthorizedKey(publicKey))
	err = os.WriteFile(knownHostsFile, []byte(knownHostsContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write known hosts file: %v", err)
	}

	// Test with valid known hosts file
	config := &SecurityConfig{
		SSHHostKeyVerification: true,
		KnownHostsFile:         knownHostsFile,
	}

	callback, err := config.createHostKeyCallback()
	if err != nil {
		t.Fatalf("Failed to create host key callback: %v", err)
	}

	// Test valid key
	err = callback("example.com", &testAddr{}, publicKey)
	if err != nil {
		t.Errorf("Valid key should be accepted: %v", err)
	}

	// Generate a different key for testing rejection
	privateKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate second test key: %v", err)
	}

	publicKey2, err := ssh.NewPublicKey(&privateKey2.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create second public key: %v", err)
	}

	// Test invalid key
	err = callback("example.com", &testAddr{}, publicKey2)
	if err == nil {
		t.Error("Invalid key should be rejected")
	}
}

func TestSecurityConfig_loadKnownHosts(t *testing.T) {
	tmpDir := t.TempDir()
	knownHostsFile := filepath.Join(tmpDir, "known_hosts")

	// Test with non-existent file
	config := &SecurityConfig{KnownHostsFile: knownHostsFile}
	callback2, err := config.loadKnownHosts(knownHostsFile)
	if err != nil {
		t.Fatalf("Should not fail with non-existent known hosts file: %v", err)
	}

	// Should reject any key for non-existent file
	err = callback2("example.com", &testAddr{}, nil)
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Error("Non-existent known hosts file should reject all keys")
	}

	// Create empty file
	err = os.WriteFile(knownHostsFile, []byte(""), 0600)
	if err != nil {
		t.Fatalf("Failed to create empty known hosts file: %v", err)
	}

	// Test with empty file
	callback, err := config.loadKnownHosts(knownHostsFile)
	if err != nil {
		t.Fatalf("Failed to load empty known hosts file: %v", err)
	}

	// Should reject any key
	err = callback("example.com", &testAddr{}, nil)
	if err == nil || !strings.Contains(err.Error(), "no matching key found") {
		t.Error("Empty known hosts file should reject all keys")
	}
}

func TestKeysEqual(t *testing.T) {
	// Generate two identical keys
	privateKey1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate first key: %v", err)
	}

	publicKey1, err := ssh.NewPublicKey(&privateKey1.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create first public key: %v", err)
	}

	publicKey2, err := ssh.NewPublicKey(&privateKey1.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create second public key: %v", err)
	}

	// Same key should be equal
	if !keysEqual(publicKey1, publicKey2) {
		t.Error("Identical keys should be equal")
	}

	// Generate different key
	privateKey3, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate third key: %v", err)
	}

	publicKey3, err := ssh.NewPublicKey(&privateKey3.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create third public key: %v", err)
	}

	// Different keys should not be equal
	if keysEqual(publicKey1, publicKey3) {
		t.Error("Different keys should not be equal")
	}
}

// testAddr implements net.Addr for testing
type testAddr struct{}

func (a *testAddr) Network() string { return "tcp" }
func (a *testAddr) String() string  { return "127.0.0.1:22" }
