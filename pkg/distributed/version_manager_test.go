package distributed

import (
	"testing"
	"time"
)

func TestNewAlertManager(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		manager := NewAlertManager(100)
		
		if manager == nil {
			t.Error("Expected non-nil alert manager")
		}
		
		if manager.maxHistory != 100 {
			t.Errorf("Expected max history to be 100, got %d", manager.maxHistory)
		}
		
		if len(manager.channels) != 0 {
			t.Errorf("Expected no channels initially, got %d", len(manager.channels))
		}
		
		if len(manager.alertHistory) != 0 {
			t.Errorf("Expected no alerts initially, got %d", len(manager.alertHistory))
		}
	})
	
	t.Run("ConstructorWithZeroMaxHistory", func(t *testing.T) {
		manager := NewAlertManager(0)
		
		if manager.maxHistory != 1000 {
			t.Errorf("Expected default max history to be 1000, got %d", manager.maxHistory)
		}
	})
}

func TestAlertManager_AddChannel(t *testing.T) {
	manager := NewAlertManager(100)
	
	// Create mock alert channel
	mockChannel := &MockAlertChannel{}
	
	// Add channel
	manager.AddChannel(mockChannel)
	
	if len(manager.channels) != 1 {
		t.Errorf("Expected 1 channel after adding, got %d", len(manager.channels))
	}
	
	// Add another channel
	mockChannel2 := &MockAlertChannel{}
	manager.AddChannel(mockChannel2)
	
	if len(manager.channels) != 2 {
		t.Errorf("Expected 2 channels after adding second, got %d", len(manager.channels))
	}
}

func TestAlertManager_SendAlert(t *testing.T) {
	t.Run("SendWithNoChannels", func(t *testing.T) {
		manager := NewAlertManager(100)
		
		alert := &DriftAlert{
			WorkerID:       "worker1",
			Severity:       "warning",
			DriftDuration:  time.Hour,
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			CurrentVersion: VersionInfo{CodebaseVersion: "1.0.1"},
		}
		
		err := manager.SendAlert(alert)
		if err != nil {
			t.Errorf("Expected no error when sending to no channels, got %v", err)
		}
		
		if alert.AlertID == "" {
			t.Error("Expected alert ID to be generated")
		}
		
		if len(manager.alertHistory) != 1 {
			t.Errorf("Expected 1 alert in history, got %d", len(manager.alertHistory))
		}
	})
	
	t.Run("SendWithChannels", func(t *testing.T) {
		manager := NewAlertManager(100)
		
		mockChannel := &MockAlertChannel{}
		manager.AddChannel(mockChannel)
		
		alert := &DriftAlert{
			WorkerID:       "worker1",
			Severity:       "warning",
			DriftDuration:  time.Hour,
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			CurrentVersion: VersionInfo{CodebaseVersion: "1.0.1"},
		}
		
		err := manager.SendAlert(alert)
		if err != nil {
			t.Errorf("Expected no error when sending to channels, got %v", err)
		}
		
		if mockChannel.sentAlert == nil {
			t.Error("Expected mock channel to receive alert")
		}
		
		if mockChannel.sentAlert.WorkerID != alert.WorkerID {
			t.Errorf("Expected alert worker ID '%s', got '%s'", alert.WorkerID, mockChannel.sentAlert.WorkerID)
		}
	})
}

func TestAlertManager_GetAlertHistory(t *testing.T) {
	manager := NewAlertManager(5)
	
	// Add some alerts
	for i := 0; i < 3; i++ {
		alert := &DriftAlert{
			WorkerID:       "worker1",
			Severity:       "warning",
			DriftDuration:  time.Hour,
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			CurrentVersion: VersionInfo{CodebaseVersion: "1.0.1"},
		}
		manager.SendAlert(alert)
	}
	
	t.Run("GetAllHistory", func(t *testing.T) {
		history := manager.GetAlertHistory(0)
		if len(history) != 3 {
			t.Errorf("Expected 3 alerts in history, got %d", len(history))
		}
	})
	
	t.Run("GetLimitedHistory", func(t *testing.T) {
		history := manager.GetAlertHistory(2)
		if len(history) != 2 {
			t.Errorf("Expected 2 alerts in limited history, got %d", len(history))
		}
	})
	
	t.Run("GetExcessiveLimit", func(t *testing.T) {
		history := manager.GetAlertHistory(100)
		if len(history) != 3 {
			t.Errorf("Expected 3 alerts when limit exceeds history, got %d", len(history))
		}
	})
}

func TestAlertManager_AcknowledgeAlert(t *testing.T) {
	manager := NewAlertManager(100)
	
	// Add an alert
	alert := &DriftAlert{
		WorkerID:       "worker1",
		Severity:       "warning",
		DriftDuration:  time.Hour,
		ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
		CurrentVersion: VersionInfo{CodebaseVersion: "1.0.1"},
	}
	manager.SendAlert(alert)
	
	alertID := alert.AlertID
	
	t.Run("AcknowledgeExistingAlert", func(t *testing.T) {
		success := manager.AcknowledgeAlert(alertID, "testuser")
		if !success {
			t.Error("Expected successful acknowledgment")
		}
		
		if !alert.Acknowledged {
			t.Error("Expected alert to be acknowledged")
		}
		
		if alert.AcknowledgedBy != "testuser" {
			t.Errorf("Expected acknowledged by 'testuser', got '%s'", alert.AcknowledgedBy)
		}
		
		if alert.AcknowledgedAt == nil {
			t.Error("Expected acknowledged at time to be set")
		}
	})
	
	t.Run("AcknowledgeNonExistentAlert", func(t *testing.T) {
		success := manager.AcknowledgeAlert("non-existent", "testuser")
		if success {
			t.Error("Expected failure for non-existent alert")
		}
	})
	
	t.Run("AcknowledgeAlreadyAcknowledged", func(t *testing.T) {
		success := manager.AcknowledgeAlert(alertID, "anotheruser")
		if success {
			t.Error("Expected failure for already acknowledged alert")
		}
	})
}

func TestEmailAlertChannel_Name(t *testing.T) {
	channel := &EmailAlertChannel{}
	
	name := channel.Name()
	if name != "email" {
		t.Errorf("Expected channel name to be 'email', got '%s'", name)
	}
}

func TestWebhookAlertChannel_Name(t *testing.T) {
	channel := &WebhookAlertChannel{}
	
	name := channel.Name()
	if name != "webhook" {
		t.Errorf("Expected channel name to be 'webhook', got '%s'", name)
	}
}

func TestSlackAlertChannel_Name(t *testing.T) {
	channel := &SlackAlertChannel{}
	
	name := channel.Name()
	if name != "slack" {
		t.Errorf("Expected channel name to be 'slack', got '%s'", name)
	}
}

// MockAlertChannel for testing
type MockAlertChannel struct {
	sentAlert *DriftAlert
}

func (m *MockAlertChannel) SendAlert(alert *DriftAlert) error {
	m.sentAlert = alert
	return nil
}

func (m *MockAlertChannel) Name() string {
	return "mock"
}

func TestEmailAlertChannel_SendAlert(t *testing.T) {
	channel := &EmailAlertChannel{
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		Username:    "test@example.com",
		Password:    "password",
		FromAddress: "test@example.com",
		ToAddresses: []string{"admin@example.com"},
	}
	
	alert := &DriftAlert{
		WorkerID: "test-worker",
		ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
		CurrentVersion:  VersionInfo{CodebaseVersion: "1.1.0"},
		Message:    "Version drift detected",
		Timestamp:  time.Now(),
		Severity:   "warning",
	}
	
	// Send alert (will fail since we don't have a real SMTP server)
	err := channel.SendAlert(alert)
	if err == nil {
		t.Error("Expected error for sending email without real SMTP server")
	}
}

func TestWebhookAlertChannel_SendAlert(t *testing.T) {
	channel := &WebhookAlertChannel{
		URL:     "https://hooks.slack.com/test",
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	
	alert := &DriftAlert{
		WorkerID: "test-worker",
		ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
		CurrentVersion:  VersionInfo{CodebaseVersion: "1.1.0"},
		Message:    "Version drift detected",
		Timestamp:  time.Now(),
		Severity:   "warning",
	}
	
	// Send alert (may succeed or fail depending on network)
	// We're testing that the function doesn't panic and handles errors appropriately
	err := channel.SendAlert(alert)
	// We don't assert a specific error result since it depends on network conditions
	_ = err // Just to avoid unused variable error
}

func TestSlackAlertChannel_SendAlert(t *testing.T) {
	channel := &SlackAlertChannel{
		WebhookURL: "https://hooks.slack.com/test",
		Channel:    "#alerts",
	}
	
	alert := &DriftAlert{
		WorkerID: "test-worker",
		ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
		CurrentVersion:  VersionInfo{CodebaseVersion: "1.1.0"},
		Message:    "Version drift detected",
		Timestamp:  time.Now(),
		Severity:   "warning",
	}
	
	// Send alert (may succeed or fail depending on network)
	// We're testing that the function doesn't panic and handles errors appropriately
	err := channel.SendAlert(alert)
	// We don't assert a specific error result since it depends on network conditions
	_ = err // Just to avoid unused variable error
}