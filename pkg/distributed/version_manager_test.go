package distributed

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
			WorkerID:        "worker1",
			Severity:        "warning",
			DriftDuration:   time.Hour,
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			CurrentVersion:  VersionInfo{CodebaseVersion: "1.0.1"},
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
			WorkerID:        "worker1",
			Severity:        "warning",
			DriftDuration:   time.Hour,
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			CurrentVersion:  VersionInfo{CodebaseVersion: "1.0.1"},
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
			WorkerID:        "worker1",
			Severity:        "warning",
			DriftDuration:   time.Hour,
			ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
			CurrentVersion:  VersionInfo{CodebaseVersion: "1.0.1"},
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
		WorkerID:        "worker1",
		Severity:        "warning",
		DriftDuration:   time.Hour,
		ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
		CurrentVersion:  VersionInfo{CodebaseVersion: "1.0.1"},
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

func testDriftAlert() *DriftAlert {
	return &DriftAlert{
		WorkerID:        "test-worker",
		ExpectedVersion: VersionInfo{CodebaseVersion: "1.0.0"},
		CurrentVersion:  VersionInfo{CodebaseVersion: "1.1.0"},
		Message:         "Version drift detected",
		Timestamp:       time.Now(),
		Severity:        "warning",
	}
}

// TestEmailAlertChannel_SendAlert drives the SMTP send path against a real
// local TCP listener (no external infra, no unbounded network wait per
// §11.4.3 + §11.4.27). The listener greets then closes, so SendAlert MUST
// return an error promptly — proving the bounded-dial fix prevents the
// indefinite hang that previously blew the package test budget.
func TestEmailAlertChannel_SendAlert(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("SKIP-OK: cannot bind loopback TCP listener (sandboxed network) — %v", err)
	}
	defer func() { _ = ln.Close() }()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		// Minimal SMTP greeting then immediate close — the client's
		// subsequent commands fail fast against a half-broken server.
		_, _ = conn.Write([]byte("220 local test smtp\r\n"))
		_ = conn.Close()
	}()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port := 0
	_, _ = fmt.Sscan(portStr, &port)

	channel := &EmailAlertChannel{
		SMTPHost:    host,
		SMTPPort:    port,
		Username:    "test@example.com",
		Password:    "password",
		FromAddress: "test@example.com",
		ToAddresses: []string{"admin@example.com"},
		Timeout:     5 * time.Second,
	}

	start := time.Now()
	err = channel.SendAlert(testDriftAlert())
	elapsed := time.Since(start)
	wg.Wait()

	if err == nil {
		t.Error("expected error sending email to a server that closes the connection")
	}
	// Bounded: must finish well within the configured 5s timeout, proving the
	// dial/send is deadline-guarded (the product fix).
	if elapsed > 6*time.Second {
		t.Errorf("SendAlert took %v; expected bounded by ~5s timeout — dial deadline not enforced", elapsed)
	}
}

// TestEmailAlertChannel_SendAlert_UnreachableIsBounded proves the missing-deadline
// product defect is fixed: against an unrouteable address the send returns within
// the configured timeout instead of blocking on the OS TCP timeout (minutes).
func TestEmailAlertChannel_SendAlert_UnreachableIsBounded(t *testing.T) {
	channel := &EmailAlertChannel{
		// 203.0.113.0/24 (TEST-NET-3, RFC 5737) is reserved and unrouteable.
		SMTPHost:    "203.0.113.1",
		SMTPPort:    587,
		Username:    "test@example.com",
		Password:    "password",
		FromAddress: "test@example.com",
		ToAddresses: []string{"admin@example.com"},
		Timeout:     2 * time.Second,
	}

	start := time.Now()
	err := channel.SendAlert(testDriftAlert())
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected error dialing unrouteable SMTP host")
	}
	if elapsed > 4*time.Second {
		t.Errorf("SendAlert took %v against unrouteable host; expected bounded by ~2s timeout", elapsed)
	}
}

// TestWebhookAlertChannel_SendAlert drives a real in-process HTTP server and
// asserts the webhook POST actually arrived with the alert payload — positive
// captured evidence, no external host (§11.4.3 + §11.4.27 + anti-bluff §11.4).
func TestWebhookAlertChannel_SendAlert(t *testing.T) {
	var (
		mu       sync.Mutex
		gotBody  string
		gotPath  bool
		gotCType string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		_, _ = bufio.NewReader(r.Body).WriteTo(buf)
		mu.Lock()
		gotBody = buf.String()
		gotPath = r.Method == http.MethodPost
		gotCType = r.Header.Get("Content-Type")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	channel := &WebhookAlertChannel{
		URL:     srv.URL,
		Method:  "POST",
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	if err := channel.SendAlert(testDriftAlert()); err != nil {
		t.Fatalf("SendAlert to local webhook server failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !gotPath {
		t.Error("expected webhook server to receive a POST request")
	}
	if gotCType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", gotCType)
	}
	if !strings.Contains(gotBody, "test-worker") {
		t.Errorf("expected webhook body to contain worker ID; got %q", gotBody)
	}
}

// TestSlackAlertChannel_SendAlert drives a real in-process HTTP server and
// asserts the Slack webhook POST actually arrived with a JSON payload.
func TestSlackAlertChannel_SendAlert(t *testing.T) {
	var (
		mu      sync.Mutex
		gotBody string
		gotPost bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		_, _ = bufio.NewReader(r.Body).WriteTo(buf)
		mu.Lock()
		gotBody = buf.String()
		gotPost = r.Method == http.MethodPost
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	channel := &SlackAlertChannel{
		WebhookURL: srv.URL,
		Channel:    "#alerts",
	}

	if err := channel.SendAlert(testDriftAlert()); err != nil {
		t.Fatalf("SendAlert to local Slack webhook server failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !gotPost {
		t.Error("expected Slack webhook server to receive a POST request")
	}
	if !strings.Contains(gotBody, "Version drift detected") {
		t.Errorf("expected Slack body to contain the alert message; got %q", gotBody)
	}
}
