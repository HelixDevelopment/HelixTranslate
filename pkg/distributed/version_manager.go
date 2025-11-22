package distributed

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/pkg/events"
)

// UpdateBackup represents a backup of a worker's state before an update
type UpdateBackup struct {
	WorkerID        string
	BackupID        string
	Timestamp       time.Time
	OriginalVersion VersionInfo
	BackupPath      string
	UpdatePackage   string
	Status          string // "created", "active", "rolled_back", "expired"
}

// VersionManager handles version checking, updates, and validation for remote workers
type VersionManager struct {
	localVersion VersionInfo
	httpClient   *http.Client
	eventBus     *events.EventBus
	updateDir    string
	backupDir    string
	backups      map[string]*UpdateBackup // workerID -> backup
	baseURL      string                   // For testing: override the URL construction
}

// NewVersionManager creates a new version manager
func NewVersionManager(eventBus *events.EventBus) *VersionManager {
	// Get local version information
	localVersion := getLocalVersionInfo()

	// Create HTTP client for version checks and downloads
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &VersionManager{
		localVersion: localVersion,
		httpClient:   httpClient,
		eventBus:     eventBus,
		updateDir:    "/tmp/translator-updates",
		backupDir:    "/tmp/translator-backups",
		backups:      make(map[string]*UpdateBackup),
	}
}

// getLocalVersionInfo retrieves version information for the local codebase
func getLocalVersionInfo() VersionInfo {
	version := VersionInfo{
		CodebaseVersion: getCodebaseVersion(),
		BuildTime:       getBuildTime(),
		GitCommit:       getGitCommit(),
		GoVersion:       getGoVersion(),
		Components:      make(map[string]string),
		LastUpdated:     time.Now(),
	}

	// Add component versions
	version.Components["translator"] = version.CodebaseVersion
	version.Components["api"] = "1.0.0"
	version.Components["distributed"] = "1.0.0"
	version.Components["deployment"] = "1.0.0"

	return version
}

// getCodebaseVersion returns the current codebase version
func getCodebaseVersion() string {
	// Try to read from version file first
	if version, err := readVersionFile("VERSION"); err == nil {
		return strings.TrimSpace(version)
	}

	// Try git describe
	if version, err := runCommand("git", "describe", "--tags", "--abbrev=0"); err == nil {
		return strings.TrimSpace(version)
	}

	// Try git rev-parse
	if commit, err := runCommand("git", "rev-parse", "--short", "HEAD"); err == nil {
		return fmt.Sprintf("dev-%s", strings.TrimSpace(commit))
	}

	return "unknown"
}

// getBuildTime returns the build timestamp
func getBuildTime() string {
	if buildTime, err := runCommand("date", "-u", "+%Y-%m-%dT%H:%M:%SZ"); err == nil {
		return strings.TrimSpace(buildTime)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

// getGitCommit returns the current git commit hash
func getGitCommit() string {
	if commit, err := runCommand("git", "rev-parse", "HEAD"); err == nil {
		return strings.TrimSpace(commit)
	}
	return "unknown"
}

// getGoVersion returns the Go version used to build
func getGoVersion() string {
	if version, err := runCommand("go", "version"); err == nil {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			return parts[2]
		}
	}
	return "unknown"
}

// readVersionFile reads version from a file
func readVersionFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// runCommand executes a shell command and returns its output
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// SetBaseURL sets the base URL for testing purposes
func (vm *VersionManager) SetBaseURL(baseURL string) {
	vm.baseURL = baseURL
}

// CheckWorkerVersion checks if a worker's version matches the local version
func (vm *VersionManager) CheckWorkerVersion(ctx context.Context, service *RemoteService) (bool, error) {
	// Query worker for its version
	var versionURL string
	if vm.baseURL != "" {
		versionURL = vm.baseURL + "/api/v1/version"
	} else {
		versionURL = fmt.Sprintf("%s://%s:%d/api/v1/version", service.Protocol, service.Host, service.Port)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", versionURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create version request: %w", err)
	}

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to query worker version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("worker version endpoint returned status %d", resp.StatusCode)
	}

	var workerVersion VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&workerVersion); err != nil {
		return false, fmt.Errorf("failed to decode worker version: %w", err)
	}

	// Update service with version info
	service.Version = workerVersion

	// Compare versions
	isUpToDate := vm.compareVersions(vm.localVersion, workerVersion)

	// Emit event
	event := events.Event{
		Type:      "worker_version_checked",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id":      service.WorkerID,
			"local_version":  vm.localVersion.CodebaseVersion,
			"worker_version": workerVersion.CodebaseVersion,
			"up_to_date":     isUpToDate,
		},
	}
	vm.eventBus.Publish(event)

	return isUpToDate, nil
}

// compareVersions compares two version infos
func (vm *VersionManager) compareVersions(local, remote VersionInfo) bool {
	// Compare codebase versions
	if local.CodebaseVersion != remote.CodebaseVersion {
		return false
	}

	// Compare critical components
	criticalComponents := []string{"translator", "api", "distributed"}
	for _, component := range criticalComponents {
		if local.Components[component] != remote.Components[component] {
			return false
		}
	}

	return true
}

// UpdateWorker updates a worker to the latest version
func (vm *VersionManager) UpdateWorker(ctx context.Context, service *RemoteService) error {
	service.Status = "updating"

	// Emit update started event
	event := events.Event{
		Type:      "worker_update_started",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id":       service.WorkerID,
			"target_version":  vm.localVersion.CodebaseVersion,
			"current_version": service.Version.CodebaseVersion,
		},
	}
	vm.eventBus.Publish(event)

	// Create backup before starting update
	backup, err := vm.createWorkerBackup(ctx, service)
	if err != nil {
		service.Status = "outdated"
		return fmt.Errorf("failed to create backup: %w", err)
	}
	backup.Status = "active"

	// Create update package
	updatePackage, err := vm.createUpdatePackage()
	if err != nil {
		vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
		return fmt.Errorf("failed to create update package: %w", err)
	}
	backup.UpdatePackage = updatePackage

	// Upload update package to worker
	if err := vm.uploadUpdatePackage(ctx, service, updatePackage); err != nil {
		vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
		return fmt.Errorf("failed to upload update package: %w", err)
	}

	// Trigger update on worker
	if err := vm.triggerWorkerUpdate(ctx, service); err != nil {
		vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
		return fmt.Errorf("failed to trigger worker update: %w", err)
	}

	// Wait for update completion
	if err := vm.waitForUpdateCompletion(ctx, service); err != nil {
		vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
		return fmt.Errorf("update failed to complete: %w", err)
	}

	// Verify update
	if upToDate, err := vm.CheckWorkerVersion(ctx, service); err != nil || !upToDate {
		vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
		return fmt.Errorf("update verification failed")
	}

	service.Status = "paired"

	// Mark backup as completed (no longer active)
	backup.Status = "completed"

	// Emit update completed event
	event = events.Event{
		Type:      "worker_update_completed",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id": service.WorkerID,
			"version":   vm.localVersion.CodebaseVersion,
		},
	}
	vm.eventBus.Publish(event)

	return nil
}

// createUpdatePackage creates a compressed package of the current codebase
func (vm *VersionManager) createUpdatePackage() (string, error) {
	// Ensure update directory exists
	if err := os.MkdirAll(vm.updateDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create update directory: %w", err)
	}

	// Create package filename
	packageName := fmt.Sprintf("translator-update-%s-%d.tar.gz",
		vm.localVersion.CodebaseVersion, time.Now().Unix())

	packagePath := filepath.Join(vm.updateDir, packageName)

	// Create tar.gz archive of current directory (excluding .git, build, etc.)
	cmd := exec.Command("tar", "-czf", packagePath, "--exclude=.git", "--exclude=build",
		"--exclude=node_modules", "--exclude=.DS_Store", ".")
	cmd.Dir = "." // Current directory

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create update package: %w", err)
	}

	return packagePath, nil
}

// uploadUpdatePackage uploads the update package to the worker
func (vm *VersionManager) uploadUpdatePackage(ctx context.Context, service *RemoteService, packagePath string) error {
	var uploadURL string
	if vm.baseURL != "" {
		uploadURL = vm.baseURL + "/api/v1/update/upload"
	} else {
		uploadURL = fmt.Sprintf("%s://%s:%d/api/v1/update/upload", service.Protocol, service.Host, service.Port)
	}

	file, err := os.Open(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open update package: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Update-Version", vm.localVersion.CodebaseVersion)

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload update package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// triggerWorkerUpdate triggers the update process on the worker
func (vm *VersionManager) triggerWorkerUpdate(ctx context.Context, service *RemoteService) error {
	var updateURL string
	if vm.baseURL != "" {
		updateURL = vm.baseURL + "/api/v1/update/apply"
	} else {
		updateURL = fmt.Sprintf("%s://%s:%d/api/v1/update/apply", service.Protocol, service.Host, service.Port)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", updateURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("X-Update-Version", vm.localVersion.CodebaseVersion)

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update trigger failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// waitForUpdateCompletion waits for the worker update to complete
func (vm *VersionManager) waitForUpdateCompletion(ctx context.Context, service *RemoteService) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("update timeout")
		case <-ticker.C:
			// Check if worker is back online and updated
			if upToDate, err := vm.CheckWorkerVersion(ctx, service); err == nil && upToDate {
				return nil
			}
		}
	}
}

// ValidateWorkerForWork validates that a worker is ready for work (up to date and healthy)
func (vm *VersionManager) ValidateWorkerForWork(ctx context.Context, service *RemoteService) error {
	// Check version
	upToDate, err := vm.CheckWorkerVersion(ctx, service)
	if err != nil {
		return fmt.Errorf("version check failed: %w", err)
	}

	if !upToDate {
		return fmt.Errorf("worker %s is outdated (local: %s, worker: %s)",
			service.WorkerID, vm.localVersion.CodebaseVersion, service.Version.CodebaseVersion)
	}

	// Check health
	var healthURL string
	if vm.baseURL != "" {
		healthURL = vm.baseURL + "/health"
	} else {
		healthURL = fmt.Sprintf("%s://%s:%d/health", service.Protocol, service.Host, service.Port)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("worker health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// createWorkerBackup creates a backup of the worker's current state before update
func (vm *VersionManager) createWorkerBackup(ctx context.Context, service *RemoteService) (*UpdateBackup, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(vm.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupID := fmt.Sprintf("backup-%s-%d", service.WorkerID, time.Now().Unix())
	backupPath := filepath.Join(vm.backupDir, backupID)

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup path: %w", err)
	}

	backup := &UpdateBackup{
		WorkerID:        service.WorkerID,
		BackupID:        backupID,
		Timestamp:       time.Now(),
		OriginalVersion: service.Version,
		BackupPath:      backupPath,
		Status:          "created",
	}

	// Store backup reference
	vm.backups[service.WorkerID] = backup

	// Emit backup created event
	event := events.Event{
		Type:      "worker_backup_created",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id":        service.WorkerID,
			"backup_id":        backupID,
			"original_version": service.Version.CodebaseVersion,
		},
	}
	vm.eventBus.Publish(event)

	return backup, nil
}

// rollbackWorkerUpdate rolls back a worker to its previous state using the backup
func (vm *VersionManager) rollbackWorkerUpdate(ctx context.Context, service *RemoteService) error {
	backup, exists := vm.backups[service.WorkerID]
	if !exists {
		return fmt.Errorf("no backup found for worker %s", service.WorkerID)
	}

	if backup.Status != "active" {
		return fmt.Errorf("backup %s is not active (status: %s)", backup.BackupID, backup.Status)
	}

	// Emit rollback started event
	event := events.Event{
		Type:      "worker_rollback_started",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id":    service.WorkerID,
			"backup_id":    backup.BackupID,
			"from_version": service.Version.CodebaseVersion,
			"to_version":   backup.OriginalVersion.CodebaseVersion,
		},
	}
	vm.eventBus.Publish(event)

	// Trigger rollback on worker
	var rollbackURL string
	if vm.baseURL != "" {
		rollbackURL = vm.baseURL + "/api/v1/update/rollback"
	} else {
		rollbackURL = fmt.Sprintf("%s://%s:%d/api/v1/update/rollback", service.Protocol, service.Host, service.Port)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", rollbackURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create rollback request: %w", err)
	}

	req.Header.Set("X-Backup-ID", backup.BackupID)

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger rollback: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rollback failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Wait for rollback completion
	if err := vm.waitForRollbackCompletion(ctx, service, backup); err != nil {
		return fmt.Errorf("rollback failed to complete: %w", err)
	}

	// Restore original version info
	service.Version = backup.OriginalVersion
	service.Status = "paired"

	// Mark backup as rolled back
	backup.Status = "rolled_back"

	// Emit rollback completed event
	event = events.Event{
		Type:      "worker_rollback_completed",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id": service.WorkerID,
			"backup_id": backup.BackupID,
			"version":   backup.OriginalVersion.CodebaseVersion,
		},
	}
	vm.eventBus.Publish(event)

	return nil
}

// waitForRollbackCompletion waits for the worker rollback to complete
func (vm *VersionManager) waitForRollbackCompletion(ctx context.Context, service *RemoteService, backup *UpdateBackup) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("rollback timeout")
		case <-ticker.C:
			// Check if worker has rolled back to original version
			if _, err := vm.CheckWorkerVersion(ctx, service); err == nil {
				if service.Version.CodebaseVersion == backup.OriginalVersion.CodebaseVersion {
					return nil
				}
			}
		}
	}
}

// cleanupExpiredBackups removes old backups that are no longer needed
func (vm *VersionManager) cleanupExpiredBackups() error {
	// Remove backups older than 24 hours that are not active
	cutoff := time.Now().Add(-24 * time.Hour)

	for workerID, backup := range vm.backups {
		if backup.Timestamp.Before(cutoff) && backup.Status != "active" {
			if err := os.RemoveAll(backup.BackupPath); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Failed to remove backup %s: %v\n", backup.BackupPath, err)
			}
			delete(vm.backups, workerID)
		}
	}

	return nil
}

// GetLocalVersion returns the local version information
func (vm *VersionManager) GetLocalVersion() VersionInfo {
	return vm.localVersion
}

// InstallWorker performs initial installation on a new worker
func (vm *VersionManager) InstallWorker(ctx context.Context, workerID, host string, port int) error {
	// This would implement the full installation process
	// For now, return not implemented
	return fmt.Errorf("worker installation not yet implemented")
}
