package distributed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestVersionManager_UploadUpdatePackage tests upload functionality
func TestVersionManager_UploadUpdatePackage(t *testing.T) {
	t.Run("UploadUpdatePackage_ValidServer", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server that accepts uploads
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}

			if r.Header.Get("Content-Type") != "application/octet-stream" {
				t.Errorf("Expected Content-Type application/octet-stream, got %s", r.Header.Get("Content-Type"))
			}

			if r.Header.Get("X-Update-Version") == "" {
				t.Error("Expected X-Update-Version header")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create temporary package file
		tempDir, err := os.MkdirTemp("", "upload-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		packagePath := filepath.Join(tempDir, "test-package.tar.gz")
		testData := []byte("test package content")
		if err := os.WriteFile(packagePath, testData, 0644); err != nil {
			t.Fatalf("Failed to create test package: %v", err)
		}

		// Create service with server URL
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "paired",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Upload package
		ctx := context.Background()
		err = vm.uploadUpdatePackage(ctx, service, packagePath)
		if err != nil {
			t.Errorf("Failed to upload package: %v", err)
		}
	})

	t.Run("UploadUpdatePackage_FailedServer", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal server error"))
		}))
		defer server.Close()

		// Create temporary package file
		tempDir, err := os.MkdirTemp("", "upload-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		packagePath := filepath.Join(tempDir, "test-package.tar.gz")
		testData := []byte("test package content")
		if err := os.WriteFile(packagePath, testData, 0644); err != nil {
			t.Fatalf("Failed to create test package: %v", err)
		}

		// Create service with server URL
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "paired",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Try to upload package
		ctx := context.Background()
		err = vm.uploadUpdatePackage(ctx, service, packagePath)
		if err == nil {
			t.Error("Expected error when uploading to failing server")
		}
	})
}

// TestVersionManager_TriggerWorkerUpdate tests update trigger
func TestVersionManager_TriggerWorkerUpdate(t *testing.T) {
	t.Run("TriggerWorkerUpdate_ValidServer", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server that accepts triggers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}

			if r.Header.Get("X-Update-Version") == "" {
				t.Error("Expected X-Update-Version header")
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create service
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "updating",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Trigger update
		ctx := context.Background()
		err := vm.triggerWorkerUpdate(ctx, service)
		if err != nil {
			t.Errorf("Failed to trigger update: %v", err)
		}
	})

	t.Run("TriggerWorkerUpdate_ServerError", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid request"))
		}))
		defer server.Close()

		// Create service
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "updating",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Try to trigger update
		ctx := context.Background()
		err := vm.triggerWorkerUpdate(ctx, service)
		if err == nil {
			t.Error("Expected error when triggering update on failing server")
		}
	})
}

// TestVersionManager_RollbackWorkerUpdate tests rollback functionality
func TestVersionManager_RollbackWorkerUpdate(t *testing.T) {
	t.Run("RollbackWorkerUpdate_NoBackup", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create service without backup
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "updating",
			Version: VersionInfo{
				CodebaseVersion: "1.0.0",
				LastUpdated:     time.Now(),
			},
		}

		// Try rollback without backup
		ctx := context.Background()
		err := vm.rollbackWorkerUpdate(ctx, service)
		if err == nil {
			t.Error("Expected error when rolling back without backup")
		}
	})

	t.Run("RollbackWorkerUpdate_ValidServer", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server that accepts rollback
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/update/rollback":
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("X-Backup-ID") == "" {
					t.Error("Expected X-Backup-ID header")
				}
				w.WriteHeader(http.StatusOK)
			case "/api/v1/version":
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				version := VersionInfo{
					CodebaseVersion: "1.0.0",
					BuildTime:       time.Now().Format(time.RFC3339),
					GitCommit:       "test-commit",
					GoVersion:       "go1.19",
					Components: map[string]string{
						"translator": "1.0.0",
					},
					LastUpdated: time.Now(),
				}
				json.NewEncoder(w).Encode(version)
			default:
				t.Errorf("Unexpected request to %s", r.URL.Path)
			}
		}))
		defer server.Close()

		// Create service with backup
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "updating",
			Version: VersionInfo{
				CodebaseVersion: "1.0.0",
				LastUpdated:     time.Now(),
			},
		}

		// Create backup manually for testing
		backup := &UpdateBackup{
			WorkerID:        service.WorkerID,
			BackupID:        "test-backup-123",
			Timestamp:       time.Now(),
			OriginalVersion: service.Version,
			Status:          "active",
		}
		vm.backups[service.WorkerID] = backup

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Trigger rollback
		ctx := context.Background()
		err := vm.rollbackWorkerUpdate(ctx, service)
		if err != nil {
			t.Errorf("Failed to trigger rollback: %v", err)
		}

		// Check that backup status is updated
		if backup.Status != "rolled_back" {
			t.Errorf("Expected backup status 'rolled_back', got '%s'", backup.Status)
		}
	})
}

// TestVersionManager_ValidateWorkerForWork tests worker validation
func TestVersionManager_ValidateWorkerForWork(t *testing.T) {
	t.Run("ValidateWorkerForWork_ValidWorker", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server that responds to health and version checks
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/version" {
				// Return version info
				version := VersionInfo{
					CodebaseVersion: vm.localVersion.CodebaseVersion,
					BuildTime:       time.Now().Format(time.RFC3339),
					GitCommit:       "test-commit",
					GoVersion:       "go1.19",
					Components: map[string]string{
						"translator":  vm.localVersion.CodebaseVersion,
						"api":         "1.0.0",
						"distributed": "1.0.0",
					},
					LastUpdated: time.Now(),
				}
				json.NewEncoder(w).Encode(version)
				return
			}

			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("healthy"))
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		// Create service
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "paired",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Validate worker
		ctx := context.Background()
		err := vm.ValidateWorkerForWork(ctx, service)
		if err != nil {
			t.Errorf("Failed to validate worker: %v", err)
		}
	})

	t.Run("ValidateWorkerForWork_OutdatedVersion", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server with outdated version
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/version" {
				// Return outdated version
				version := VersionInfo{
					CodebaseVersion: "old-version",
					BuildTime:       time.Now().Format(time.RFC3339),
					GitCommit:       "old-commit",
					GoVersion:       "go1.18",
					Components: map[string]string{
						"translator":  "old-version",
						"api":         "0.9.0",
						"distributed": "0.9.0",
					},
					LastUpdated: time.Now(),
				}
				json.NewEncoder(w).Encode(version)
				return
			}

			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("healthy"))
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		// Create service
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "paired",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Try to validate worker
		ctx := context.Background()
		err := vm.ValidateWorkerForWork(ctx, service)
		if err == nil {
			t.Error("Expected error for outdated worker")
		}
	})

	t.Run("ValidateWorkerForWork_Unhealthy", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Create test server with correct version but unhealthy
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/version" {
				// Return correct version
				version := VersionInfo{
					CodebaseVersion: vm.localVersion.CodebaseVersion,
					BuildTime:       time.Now().Format(time.RFC3339),
					GitCommit:       "test-commit",
					GoVersion:       "go1.19",
					Components: map[string]string{
						"translator":  vm.localVersion.CodebaseVersion,
						"api":         "1.0.0",
						"distributed": "1.0.0",
					},
					LastUpdated: time.Now(),
				}
				json.NewEncoder(w).Encode(version)
				return
			}

			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("unhealthy"))
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		// Create service
		service := &RemoteService{
			WorkerID: "test-worker",
			Host:     "localhost",
			Port:     8080,
			Protocol: "http",
			Status:   "paired",
		}

		// Set base URL to test server
		vm.SetBaseURL(server.URL)

		// Try to validate worker
		ctx := context.Background()
		err := vm.ValidateWorkerForWork(ctx, service)
		if err == nil {
			t.Error("Expected error for unhealthy worker")
		}
	})
}

// TestVersionManager_InstallWorkerExtended tests worker installation
func TestVersionManager_InstallWorkerExtended(t *testing.T) {
	t.Run("InstallWorker_Success", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Install worker with sufficient time
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := vm.InstallWorker(ctx, "test-worker", "localhost", 22)
		if err != nil {
			t.Errorf("Failed to install worker: %v", err)
		}
	})

	t.Run("InstallWorker_Cancelled", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Cancel context immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := vm.InstallWorker(ctx, "test-worker", "localhost", 22)
		if err == nil {
			t.Error("Expected error when context is cancelled")
		}
	})
}

// TestVersionManager_MetricsRecording tests metrics recording
func TestVersionManager_MetricsRecording(t *testing.T) {
	t.Run("RecordUpdateMetrics", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Record successful update
		vm.RecordUpdateMetrics(true, 30*time.Second)

		// Check metrics
		metrics := vm.GetMetrics()
		if metrics.TotalUpdates != 1 {
			t.Errorf("Expected 1 total update, got %d", metrics.TotalUpdates)
		}
		if metrics.SuccessfulUpdates != 1 {
			t.Errorf("Expected 1 successful update, got %d", metrics.SuccessfulUpdates)
		}
		if metrics.FailedUpdates != 0 {
			t.Errorf("Expected 0 failed updates, got %d", metrics.FailedUpdates)
		}
		if metrics.UpdateDuration != 30*time.Second {
			t.Errorf("Expected 30s update duration, got %v", metrics.UpdateDuration)
		}

		// Record failed update
		vm.RecordUpdateMetrics(false, 60*time.Second)

		// Check updated metrics
		metrics = vm.GetMetrics()
		if metrics.TotalUpdates != 2 {
			t.Errorf("Expected 2 total updates, got %d", metrics.TotalUpdates)
		}
		if metrics.SuccessfulUpdates != 1 {
			t.Errorf("Expected 1 successful update, got %d", metrics.SuccessfulUpdates)
		}
		if metrics.FailedUpdates != 1 {
			t.Errorf("Expected 1 failed update, got %d", metrics.FailedUpdates)
		}
	})

	t.Run("RecordRollbackMetrics", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Record successful rollback
		vm.RecordRollbackMetrics(true, 15*time.Second)

		// Check metrics
		metrics := vm.GetMetrics()
		if metrics.TotalRollbacks != 1 {
			t.Errorf("Expected 1 total rollback, got %d", metrics.TotalRollbacks)
		}
		if metrics.SuccessfulRollbacks != 1 {
			t.Errorf("Expected 1 successful rollback, got %d", metrics.SuccessfulRollbacks)
		}
		if metrics.FailedRollbacks != 0 {
			t.Errorf("Expected 0 failed rollbacks, got %d", metrics.FailedRollbacks)
		}
		if metrics.RollbackDuration != 15*time.Second {
			t.Errorf("Expected 15s rollback duration, got %v", metrics.RollbackDuration)
		}
	})

	t.Run("RecordSignatureMetrics", func(t *testing.T) {
		eventBus := events.NewEventBus()
		vm := NewVersionManager(eventBus)

		// Record successful signature verification
		vm.RecordSignatureMetrics(true)

		// Check metrics
		metrics := vm.GetMetrics()
		if metrics.SignatureVerifications != 1 {
			t.Errorf("Expected 1 signature verification, got %d", metrics.SignatureVerifications)
		}
		if metrics.SignatureSuccesses != 1 {
			t.Errorf("Expected 1 successful signature, got %d", metrics.SignatureSuccesses)
		}
		if metrics.SignatureFailures != 0 {
			t.Errorf("Expected 0 failed signatures, got %d", metrics.SignatureFailures)
		}

		// Record failed signature verification
		vm.RecordSignatureMetrics(false)

		// Check updated metrics
		metrics = vm.GetMetrics()
		if metrics.SignatureVerifications != 2 {
			t.Errorf("Expected 2 signature verifications, got %d", metrics.SignatureVerifications)
		}
		if metrics.SignatureSuccesses != 1 {
			t.Errorf("Expected 1 successful signature, got %d", metrics.SignatureSuccesses)
		}
		if metrics.SignatureFailures != 1 {
			t.Errorf("Expected 1 failed signature, got %d", metrics.SignatureFailures)
		}
	})
}

// TestVersionManager_CalculateDriftSeverity tests drift severity calculation
func TestVersionManager_CalculateDriftSeverity(t *testing.T) {
	eventBus := events.NewEventBus()
	vm := NewVersionManager(eventBus)

	t.Run("LowSeverity", func(t *testing.T) {
		severity := vm.calculateDriftSeverity(2 * time.Hour)
		if severity != "low" {
			t.Errorf("Expected 'low' severity for 2 hours, got '%s'", severity)
		}
	})

	t.Run("MediumSeverity", func(t *testing.T) {
		severity := vm.calculateDriftSeverity(8 * time.Hour)
		if severity != "medium" {
			t.Errorf("Expected 'medium' severity for 8 hours, got '%s'", severity)
		}
	})

	t.Run("HighSeverity", func(t *testing.T) {
		severity := vm.calculateDriftSeverity(18 * time.Hour)
		if severity != "high" {
			t.Errorf("Expected 'high' severity for 18 hours, got '%s'", severity)
		}
	})

	t.Run("CriticalSeverity", func(t *testing.T) {
		severity := vm.calculateDriftSeverity(36 * time.Hour)
		if severity != "critical" {
			t.Errorf("Expected 'critical' severity for 36 hours, got '%s'", severity)
		}
	})
}
