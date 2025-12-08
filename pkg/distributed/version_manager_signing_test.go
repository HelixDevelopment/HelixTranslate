package distributed

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestVersionManager_GenerateSigningKeys tests the key generation functionality
func TestVersionManager_GenerateSigningKeys(t *testing.T) {
	t.Run("GenerateSigningKeys_Success", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create temporary directory for keys
		keyDir, err := os.MkdirTemp("", "signing-keys-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(keyDir)
		
		// Generate keys
		privateKeyPath, publicKeyPath, err := vm.generateSigningKeys(keyDir)
		if err != nil {
			t.Fatalf("Failed to generate signing keys: %v", err)
		}
		
		// Check that files exist
		if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
			t.Error("Private key file was not created")
		}
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			t.Error("Public key file was not created")
		}
		
		// Validate private key
		privateKeyData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			t.Fatalf("Failed to read private key: %v", err)
		}
		
		block, _ := pem.Decode(privateKeyData)
		if block == nil {
			t.Error("Failed to decode private key PEM")
		}
		
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			t.Fatalf("Failed to parse private key: %v", err)
		}
		
		if privateKey.N.BitLen() != 2048 {
			t.Errorf("Expected 2048-bit key, got %d bits", privateKey.N.BitLen())
		}
		
		// Validate public key
		publicKeyData, err := os.ReadFile(publicKeyPath)
		if err != nil {
			t.Fatalf("Failed to read public key: %v", err)
		}
		
		block, _ = pem.Decode(publicKeyData)
		if block == nil {
			t.Error("Failed to decode public key PEM")
		}
		
		publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			t.Fatalf("Failed to parse public key: %v", err)
		}
		
		if publicKey.N.Cmp(privateKey.N) != 0 {
			t.Error("Public and private keys don't match")
		}
	})
	
	t.Run("GenerateSigningKeys_InvalidDir", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Try to use an invalid directory path
		invalidDir := "/invalid/path/that/does/not/exist"
		
		// Should fail to create directory
		_, _, err := vm.generateSigningKeys(invalidDir)
		if err == nil {
			t.Error("Expected error when generating keys in invalid directory")
		}
	})
}

// TestVersionManager_VerifyUpdatePackage tests the signature verification functionality
func TestVersionManager_VerifyUpdatePackage(t *testing.T) {
	t.Run("VerifyUpdatePackage_ValidSignature", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "verify-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Generate test package
		packagePath := filepath.Join(tempDir, "test-package.tar.gz")
		testData := []byte("test package content")
		if err := os.WriteFile(packagePath, testData, 0644); err != nil {
			t.Fatalf("Failed to create test package: %v", err)
		}
		
		// Generate signing keys
		keyDir := filepath.Join(tempDir, "keys")
		privateKeyPath, publicKeyPath, err := vm.generateSigningKeys(keyDir)
		if err != nil {
			t.Fatalf("Failed to generate keys: %v", err)
		}
		
		// Sign the package
		signaturePath, err := vm.signUpdatePackage(packagePath, privateKeyPath)
		if err != nil {
			t.Fatalf("Failed to sign package: %v", err)
		}
		
		// Verify the signature
		err = vm.verifyUpdatePackage(packagePath, signaturePath, publicKeyPath)
		if err != nil {
			t.Errorf("Failed to verify valid signature: %v", err)
		}
		
		// Clean up signature file
		os.Remove(signaturePath)
	})
	
	t.Run("VerifyUpdatePackage_InvalidSignature", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "verify-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Generate test package
		packagePath := filepath.Join(tempDir, "test-package.tar.gz")
		testData := []byte("test package content")
		if err := os.WriteFile(packagePath, testData, 0644); err != nil {
			t.Fatalf("Failed to create test package: %v", err)
		}
		
		// Generate signing keys
		keyDir := filepath.Join(tempDir, "keys")
		_, publicKeyPath, err := vm.generateSigningKeys(keyDir)
		if err != nil {
			t.Fatalf("Failed to generate keys: %v", err)
		}
		
		// Create invalid signature
		invalidSignature := []byte("invalid signature data")
		signaturePath := filepath.Join(tempDir, "invalid.sig")
		if err := os.WriteFile(signaturePath, invalidSignature, 0644); err != nil {
			t.Fatalf("Failed to write invalid signature: %v", err)
		}
		
		// Try to verify with invalid signature
		err = vm.verifyUpdatePackage(packagePath, signaturePath, publicKeyPath)
		if err == nil {
			t.Error("Expected verification to fail with invalid signature")
		}
	})
	
	t.Run("VerifyUpdatePackage_MissingFiles", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Try to verify with non-existent files
		err := vm.verifyUpdatePackage(
			"/non/existent/package.tar.gz",
			"/non/existent/signature.sig",
			"/non/existent/public.pub",
		)
		if err == nil {
			t.Error("Expected error when verifying non-existent files")
		}
	})
}

// TestVersionManager_CreateSignedUpdatePackage tests the signed package creation
func TestVersionManager_CreateSignedUpdatePackage(t *testing.T) {
	t.Run("CreateSignedUpdatePackage_Success", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create temporary directory for keys
		keyDir, err := os.MkdirTemp("", "signed-package-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(keyDir)
		
		// Generate signing keys
		privateKeyPath, _, err := vm.generateSigningKeys(keyDir)
		if err != nil {
			t.Fatalf("Failed to generate keys: %v", err)
		}
		
		// Create signed package
		signedPackage, err := vm.createSignedUpdatePackage(privateKeyPath)
		if err != nil {
			t.Fatalf("Failed to create signed update package: %v", err)
		}
		
		// Verify package structure
		if signedPackage.PackagePath == "" {
			t.Error("Package path is empty")
		}
		if signedPackage.SignaturePath == "" {
			t.Error("Signature path is empty")
		}
		if signedPackage.PublicKeyPath == "" {
			t.Error("Public key path is empty")
		}
		if signedPackage.Version == "" {
			t.Error("Version is empty")
		}
		if signedPackage.Timestamp.IsZero() {
			t.Error("Timestamp is zero")
		}
		
		// Check that files exist
		if _, err := os.Stat(signedPackage.PackagePath); os.IsNotExist(err) {
			t.Error("Package file was not created")
		}
		if _, err := os.Stat(signedPackage.SignaturePath); os.IsNotExist(err) {
			t.Error("Signature file was not created")
		}
		
		// Clean up
		os.Remove(signedPackage.PackagePath)
		os.Remove(signedPackage.SignaturePath)
	})
	
	t.Run("CreateSignedUpdatePackage_InvalidKey", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Try with non-existent private key
		invalidKeyPath := "/non/existent/private.key"
		
		// Should fail
		_, err := vm.createSignedUpdatePackage(invalidKeyPath)
		if err == nil {
			t.Error("Expected error with invalid private key path")
		}
	})
}

// TestVersionManager_UpdateWorkerWithSigning tests the update worker with signing
func TestVersionManager_UpdateWorkerWithSigning(t *testing.T) {
	t.Run("UpdateWorkerWithSigning_InvalidKey", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create mock service
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "paired",
			Version: VersionInfo{
				CodebaseVersion: "v1.0.0",
				LastUpdated:     time.Now(),
			},
		}
		
		// Try to update with invalid signing key
		ctx := context.Background()
		err := vm.UpdateWorkerWithSigning(ctx, service, "/invalid/key.pem", "")
		
		// Should fail due to invalid key
		if err == nil {
			t.Error("Expected error with invalid signing key")
		}
		
		// Status should be reverted to outdated
		if service.Status != "outdated" {
			t.Errorf("Expected status 'outdated', got '%s'", service.Status)
		}
	})
}

// TestVersionManager_HealthStatus tests the health status reporting
func TestVersionManager_HealthStatus(t *testing.T) {
	t.Run("GetHealthStatus_Default", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Get health status
		status := vm.GetHealthStatus()
		
		// Check required fields
		if status["status"] == nil {
			t.Error("Status field is missing")
		}
		if status["health_score"] == nil {
			t.Error("Health score field is missing")
		}
		if status["last_drift_check"] == nil {
			t.Error("Last drift check field is missing")
		}
		if status["workers_checked"] == nil {
			t.Error("Workers checked field is missing")
		}
		if status["active_alerts"] == nil {
			t.Error("Active alerts field is missing")
		}
		
		// Health score should be a valid percentage
		score, ok := status["health_score"].(float64)
		if !ok || score < 0 || score > 100 {
			t.Errorf("Invalid health score: %v", status["health_score"])
		}
		
		// Status should be one of the expected values
		statusStr, ok := status["status"].(string)
		if !ok || (statusStr != "healthy" && statusStr != "warning" && statusStr != "critical") {
			t.Errorf("Invalid status value: %v", status["status"])
		}
	})
}

// TestVersionManager_AlertChannels tests the alert channel functionality
func TestVersionManager_AlertChannels(t *testing.T) {
	t.Run("AddAlertChannel_Webhook", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create webhook alert channel
		webhookChannel := &WebhookAlertChannel{
			URL:    "https://example.com/webhook",
			Method: "POST",
			Headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
		}
		
		// Add channel
		vm.AddAlertChannel(webhookChannel)
		
		// Should not panic - channel is added
	})
	
	t.Run("AddAlertChannel_Email", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create email alert channel
		emailChannel := &EmailAlertChannel{
			SMTPHost:    "smtp.example.com",
			SMTPPort:    587,
			Username:    "test@example.com",
			Password:    "password",
			FromAddress: "noreply@example.com",
			ToAddresses: []string{"admin@example.com"},
		}
		
		// Add channel
		vm.AddAlertChannel(emailChannel)
		
		// Should not panic - channel is added
	})
	
	t.Run("AddAlertChannel_Slack", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)
		
		// Create Slack alert channel
		slackChannel := &SlackAlertChannel{
			WebhookURL: "https://hooks.slack.com/services/test",
			Channel:    "#alerts",
			Username:   "VersionManager",
		}
		
		// Add channel
		vm.AddAlertChannel(slackChannel)
		
		// Should not panic - channel is added
	})
}

// TestVersionManager_AlertManager_AcknowledgeAlert tests alert acknowledgment
func TestVersionManager_AlertManager_AcknowledgeAlert(t *testing.T) {
	t.Run("AcknowledgeAlert_ValidID", func(t *testing.T) {
		// Create alert manager
		am := NewAlertManager(100)
		
		// Create test alert
		alert := &DriftAlert{
			AlertID: "test-alert-123",
			WorkerID: "test-worker",
			Severity: "medium",
			Timestamp: time.Now(),
			Message: "Test alert message",
			Acknowledged: false,
		}
		
		// Send alert (which adds to history)
		err := am.SendAlert(alert)
		if err != nil {
			t.Fatalf("Failed to send alert: %v", err)
		}
		
		// Acknowledge alert
		success := am.AcknowledgeAlert("test-alert-123", "test-user")
		
		if !success {
			t.Error("Failed to acknowledge valid alert")
		}
		
		// Get alert history and check acknowledgment
		history := am.GetAlertHistory(10)
		if len(history) == 0 {
			t.Fatal("Alert history is empty")
		}
		
		lastAlert := history[len(history)-1]
		if !lastAlert.Acknowledged {
			t.Error("Alert was not marked as acknowledged")
		}
		if lastAlert.AcknowledgedBy != "test-user" {
			t.Errorf("Expected acknowledged by 'test-user', got '%s'", lastAlert.AcknowledgedBy)
		}
		if lastAlert.AcknowledgedAt == nil {
			t.Error("AcknowledgedAt timestamp is nil")
		}
	})
	
	t.Run("AcknowledgeAlert_InvalidID", func(t *testing.T) {
		// Create alert manager
		am := NewAlertManager(100)
		
		// Try to acknowledge non-existent alert
		success := am.AcknowledgeAlert("non-existent-id", "test-user")
		
		if success {
			t.Error("Expected failure when acknowledging non-existent alert")
		}
	})
	
	t.Run("AcknowledgeAlert_AlreadyAcknowledged", func(t *testing.T) {
		// Create alert manager
		am := NewAlertManager(100)
		
		// Create test alert
		alert := &DriftAlert{
			AlertID: "test-alert-456",
			WorkerID: "test-worker",
			Severity: "low",
			Timestamp: time.Now(),
			Message: "Another test alert",
			Acknowledged: true, // Already acknowledged
			AcknowledgedBy: "previous-user",
		}
		
		// Send alert (which adds to history)
		err := am.SendAlert(alert)
		if err != nil {
			t.Fatalf("Failed to send alert: %v", err)
		}
		
		// Try to acknowledge already acknowledged alert
		success := am.AcknowledgeAlert("test-alert-456", "new-user")
		
		if success {
			t.Error("Expected failure when acknowledging already acknowledged alert")
		}
	})
}

// TestVersionManager_EmailAlertChannel_SendAlert tests email alert channel
func TestVersionManager_EmailAlertChannel_SendAlert(t *testing.T) {
	t.Run("SendAlert_ValidAlert", func(t *testing.T) {
		// Create email channel (with invalid SMTP settings for testing)
		channel := &EmailAlertChannel{
			SMTPHost:    "invalid.smtp.server",
			SMTPPort:    587,
			Username:    "test@example.com",
			Password:    "password",
			FromAddress: "noreply@example.com",
			ToAddresses: []string{"admin@example.com"},
		}
		
		// Create test alert
		alert := &DriftAlert{
			AlertID: "test-email-alert",
			WorkerID: "test-worker",
			CurrentVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.1.0"},
			DriftDuration: time.Hour,
			Severity: "high",
			Timestamp: time.Now(),
			Message: "Version drift detected",
		}
		
		// Send alert (will fail due to invalid SMTP, but we test the structure)
		err := channel.SendAlert(alert)
		
		// Should fail due to invalid SMTP, but that's expected
		if err == nil {
			t.Error("Expected error with invalid SMTP settings")
		}
		
		// Check channel name
		if channel.Name() != "email" {
			t.Errorf("Expected channel name 'email', got '%s'", channel.Name())
		}
	})
}

// TestVersionManager_WebhookAlertChannel_SendAlert tests webhook alert channel
func TestVersionManager_WebhookAlertChannel_SendAlert(t *testing.T) {
	t.Run("SendAlert_ValidAlert", func(t *testing.T) {
		// Create webhook channel (with invalid URL for testing)
		channel := &WebhookAlertChannel{
			URL:    "http://invalid.webhook.url",
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}
		
		// Create test alert
		alert := &DriftAlert{
			AlertID: "test-webhook-alert",
			WorkerID: "test-worker",
			CurrentVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.1.0"},
			DriftDuration: time.Hour,
			Severity: "critical",
			Timestamp: time.Now(),
			Message: "Critical version drift",
		}
		
		// Send alert (will fail due to invalid URL, but we test the structure)
		err := channel.SendAlert(alert)
		
		// Should fail due to invalid URL, but that's expected
		if err == nil {
			t.Error("Expected error with invalid webhook URL")
		}
		
		// Check channel name
		if channel.Name() != "webhook" {
			t.Errorf("Expected channel name 'webhook', got '%s'", channel.Name())
		}
	})
}

// TestVersionManager_SlackAlertChannel_SendAlert tests Slack alert channel
func TestVersionManager_SlackAlertChannel_SendAlert(t *testing.T) {
	t.Run("SendAlert_ValidAlert", func(t *testing.T) {
		// Create Slack channel (with invalid webhook for testing)
		channel := &SlackAlertChannel{
			WebhookURL: "https://hooks.slack.com/invalid/webhook",
			Channel:    "#alerts",
			Username:   "VersionMonitor",
		}
		
		// Create test alert
		alert := &DriftAlert{
			AlertID: "test-slack-alert",
			WorkerID: "test-worker",
			CurrentVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.1.0"},
			DriftDuration: 30 * time.Minute,
			Severity: "medium",
			Timestamp: time.Now(),
			Message: "Medium version drift detected",
		}
		
		// Send alert (will fail due to invalid webhook, but we test the structure)
		err := channel.SendAlert(alert)
		
		// Should fail due to invalid webhook, but that's expected
		if err == nil {
			t.Error("Expected error with invalid Slack webhook")
		}
		
		// Check channel name
		if channel.Name() != "slack" {
			t.Errorf("Expected channel name 'slack', got '%s'", channel.Name())
		}
	})
}