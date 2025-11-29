package distributed

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

func TestVersionManager_signUpdatePackage(t *testing.T) {
	t.Run("signUpdatePackage_Success", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()
		
		// Generate test RSA key
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate test key: %v", err)
		}
		
		// Write private key to file
		keyPath := filepath.Join(tmpDir, "test_key.pem")
		keyData := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
		if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
			t.Fatalf("Failed to write test key: %v", err)
		}
		
		// Create test package
		packagePath := filepath.Join(tmpDir, "test_package.tar.gz")
		packageContent := []byte("test package content")
		if err := os.WriteFile(packagePath, packageContent, 0644); err != nil {
			t.Fatalf("Failed to write test package: %v", err)
		}
		
		// Test signUpdatePackage
		vm := &VersionManager{}
		sigPath, err := vm.signUpdatePackage(packagePath, keyPath)
		if err != nil {
			t.Errorf("Expected no error signing package, got %v", err)
		}
		
		// Verify signature file was created
		if sigPath != packagePath+".sig" {
			t.Errorf("Expected signature path '%s', got '%s'", packagePath+".sig", sigPath)
		}
		
		// Check signature file exists
		if _, err := os.Stat(sigPath); os.IsNotExist(err) {
			t.Error("Signature file was not created")
		}
	})
	
	t.Run("signUpdatePackage_InvalidPrivateKey", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create invalid private key file
		keyPath := filepath.Join(tmpDir, "invalid_key.pem")
		if err := os.WriteFile(keyPath, []byte("invalid key data"), 0600); err != nil {
			t.Fatalf("Failed to write invalid key: %v", err)
		}
		
		// Create test package
		packagePath := filepath.Join(tmpDir, "test_package.tar.gz")
		if err := os.WriteFile(packagePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to write test package: %v", err)
		}
		
		// Test signUpdatePackage with invalid key
		vm := &VersionManager{}
		_, err := vm.signUpdatePackage(packagePath, keyPath)
		if err == nil {
			t.Error("Expected error for invalid private key")
		}
	})
	
	t.Run("signUpdatePackage_NonExistentFiles", func(t *testing.T) {
		vm := &VersionManager{}
		
		// Test with non-existent private key
		_, err := vm.signUpdatePackage("nonexistent_package.tar.gz", "nonexistent_key.pem")
		if err == nil {
			t.Error("Expected error for non-existent private key")
		}
	})
}

func TestVersionManager_CheckVersionDrift(t *testing.T) {
	t.Run("CheckVersionDrift_WithServices", func(t *testing.T) {
		// Create VersionManager with basic setup
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Set local version for testing
		vm.localVersion = VersionInfo{CodebaseVersion: "v1.0.0"}
		
		// Create test services
		now := time.Now()
		services := []*RemoteService{
			{
				WorkerID: "worker1",
				Version: VersionInfo{
					CodebaseVersion: "v1.0.0", // Same as local
					LastUpdated:     now,
				},
				LastSeen: now,
			},
			{
				WorkerID: "worker2",
				Version: VersionInfo{
					CodebaseVersion: "v0.9.0", // Older version
					LastUpdated:     now.Add(-2 * time.Hour), // 2 hours ago
				},
				LastSeen: now,
			},
		}
		
		// Test CheckVersionDrift - this will create alerts for unreachable workers
		// Since we're not setting up mock HTTP servers, both workers will be "unreachable"
		alerts := vm.CheckVersionDrift(context.Background(), services)
		
		// Should have 2 alerts for unreachable workers
		if len(alerts) != 2 {
			t.Errorf("Expected 2 alerts for unreachable workers, got %d", len(alerts))
		}
		
		// Check that alerts have high severity for unreachable workers
		for _, alert := range alerts {
			if alert.Severity != "high" {
				t.Errorf("Expected high severity for unreachable worker, got %s", alert.Severity)
			}
			if !strings.Contains(alert.Message, "unreachable") {
				t.Errorf("Expected 'unreachable' in message, got: %s", alert.Message)
			}
		}
		
		// Check metrics - both should be unhealthy since they're unreachable
		if vm.metrics.WorkersUnhealthy != 2 {
			t.Errorf("Expected 2 workers unhealthy, got %d", vm.metrics.WorkersUnhealthy)
		}
	})
	
	t.Run("CheckVersionDrift_EmptyServices", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Test with empty services list
		alerts := vm.CheckVersionDrift(context.Background(), []*RemoteService{})
		
		// Should have no alerts
		if len(alerts) != 0 {
			t.Errorf("Expected no alerts for empty services list, got %d", len(alerts))
		}
		
		// Check metrics
		if vm.metrics.WorkersChecked != 0 {
			t.Errorf("Expected 0 workers checked, got %d", vm.metrics.WorkersChecked)
		}
	})
}