package distributed

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

// SignedUpdatePackage represents a signed update package
type SignedUpdatePackage struct {
	PackagePath   string
	SignaturePath string
	PublicKeyPath string
	Version       string
	Timestamp     time.Time
}

// VersionMetrics represents version management metrics
type VersionMetrics struct {
	// Update metrics
	TotalUpdates      int64
	SuccessfulUpdates int64
	FailedUpdates     int64
	UpdateDuration    time.Duration
	LastUpdateTime    time.Time

	// Rollback metrics
	TotalRollbacks      int64
	SuccessfulRollbacks int64
	FailedRollbacks     int64
	RollbackDuration    time.Duration
	LastRollbackTime    time.Time

	// Version drift metrics
	WorkersChecked   int64
	WorkersUpToDate  int64
	WorkersOutdated  int64
	WorkersUnhealthy int64
	LastDriftCheck   time.Time
	MaxDriftDuration time.Duration

	// Security metrics
	SignatureVerifications int64
	SignatureSuccesses     int64
	SignatureFailures      int64
	KeyGenerations         int64

	// Backup metrics
	BackupsCreated     int64
	BackupsActive      int64
	BackupsExpired     int64
	BackupStorageBytes int64
}

// DriftAlert represents a version drift alert
type DriftAlert struct {
	WorkerID        string
	CurrentVersion  VersionInfo
	ExpectedVersion VersionInfo
	DriftDuration   time.Duration
	Severity        string // "low", "medium", "high", "critical"
	Timestamp       time.Time
	Message         string
	AlertID         string
	Acknowledged    bool
	AcknowledgedAt  *time.Time
	AcknowledgedBy  string
}

// AlertChannel represents an alert notification channel
type AlertChannel interface {
	SendAlert(alert *DriftAlert) error
	Name() string
}

// EmailAlertChannel sends alerts via email
type EmailAlertChannel struct {
	SMTPHost    string
	SMTPPort    int
	Username    string
	Password    string
	FromAddress string
	ToAddresses []string
}

// WebhookAlertChannel sends alerts via HTTP webhook
type WebhookAlertChannel struct {
	URL        string
	Method     string
	Headers    map[string]string
	HTTPClient *http.Client
}

// SlackAlertChannel sends alerts to Slack
type SlackAlertChannel struct {
	WebhookURL string
	Channel    string
	Username   string
	HTTPClient *http.Client
}

// AlertManager manages alert notifications
type AlertManager struct {
	channels     []AlertChannel
	alertHistory []*DriftAlert
	maxHistory   int
	mu           sync.RWMutex
}

// NewAlertManager creates a new alert manager
func NewAlertManager(maxHistory int) *AlertManager {
	if maxHistory <= 0 {
		maxHistory = 1000
	}

	return &AlertManager{
		channels:     make([]AlertChannel, 0),
		alertHistory: make([]*DriftAlert, 0),
		maxHistory:   maxHistory,
	}
}

// AddChannel adds an alert channel
func (am *AlertManager) AddChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.channels = append(am.channels, channel)
}

// SendAlert sends an alert through all configured channels
func (am *AlertManager) SendAlert(alert *DriftAlert) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate alert ID if not set
	if alert.AlertID == "" {
		alert.AlertID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}

	// Add to history
	am.alertHistory = append(am.alertHistory, alert)

	// Trim history if needed
	if len(am.alertHistory) > am.maxHistory {
		am.alertHistory = am.alertHistory[len(am.alertHistory)-am.maxHistory:]
	}

	// Send through all channels
	var lastErr error
	for _, channel := range am.channels {
		if err := channel.SendAlert(alert); err != nil {
			lastErr = err
			// Log error but continue with other channels
		}
	}

	return lastErr
}

// GetAlertHistory returns alert history
func (am *AlertManager) GetAlertHistory(limit int) []*DriftAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	// Return most recent alerts first
	result := make([]*DriftAlert, limit)
	copy(result, am.alertHistory[len(am.alertHistory)-limit:])
	return result
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(alertID, acknowledgedBy string) bool {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, alert := range am.alertHistory {
		if alert.AlertID == alertID && !alert.Acknowledged {
			now := time.Now()
			alert.Acknowledged = true
			alert.AcknowledgedAt = &now
			alert.AcknowledgedBy = acknowledgedBy
			return true
		}
	}
	return false
}

// EmailAlertChannel implementation
func (e *EmailAlertChannel) Name() string {
	return "email"
}

func (e *EmailAlertChannel) SendAlert(alert *DriftAlert) error {
	subject := fmt.Sprintf("[%s] Version Drift Alert: %s", strings.ToUpper(alert.Severity), alert.WorkerID)

	body := fmt.Sprintf(`Version Drift Alert

Worker ID: %s
Severity: %s
Drift Duration: %v

Current Version: %s
Expected Version: %s

Message: %s

Timestamp: %s
Alert ID: %s

This is an automated alert from the version management system.
`,
		alert.WorkerID,
		alert.Severity,
		alert.DriftDuration,
		alert.CurrentVersion.CodebaseVersion,
		alert.ExpectedVersion.CodebaseVersion,
		alert.Message,
		alert.Timestamp.Format(time.RFC3339),
		alert.AlertID,
	)

	// Create email message
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		e.FromAddress,
		strings.Join(e.ToAddresses, ","),
		subject,
		body,
	)

	// Send email
	auth := smtp.PlainAuth("", e.Username, e.Password, e.SMTPHost)
	addr := fmt.Sprintf("%s:%d", e.SMTPHost, e.SMTPPort)

	return smtp.SendMail(addr, auth, e.FromAddress, e.ToAddresses, []byte(message))
}

// WebhookAlertChannel implementation
func (w *WebhookAlertChannel) Name() string {
	return "webhook"
}

func (w *WebhookAlertChannel) SendAlert(alert *DriftAlert) error {
	if w.HTTPClient == nil {
		w.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	if w.Method == "" {
		w.Method = "POST"
	}

	payload := map[string]interface{}{
		"alert_id":         alert.AlertID,
		"worker_id":        alert.WorkerID,
		"severity":         alert.Severity,
		"drift_duration":   alert.DriftDuration.String(),
		"current_version":  alert.CurrentVersion.CodebaseVersion,
		"expected_version": alert.ExpectedVersion.CodebaseVersion,
		"message":          alert.Message,
		"timestamp":        alert.Timestamp.Format(time.RFC3339),
		"acknowledged":     alert.Acknowledged,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal alert payload: %w", err)
	}

	req, err := http.NewRequest(w.Method, w.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range w.Headers {
		req.Header.Set(key, value)
	}

	resp, err := w.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SlackAlertChannel implementation
func (s *SlackAlertChannel) Name() string {
	return "slack"
}

func (s *SlackAlertChannel) SendAlert(alert *DriftAlert) error {
	if s.HTTPClient == nil {
		s.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	if s.Username == "" {
		s.Username = "Version Monitor"
	}

	color := "good"
	switch alert.Severity {
	case "low":
		color = "good"
	case "medium":
		color = "warning"
	case "high":
		color = "danger"
	case "critical":
		color = "#FF0000"
	}

	payload := map[string]interface{}{
		"channel":  s.Channel,
		"username": s.Username,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": fmt.Sprintf("Version Drift Alert - %s", strings.ToUpper(alert.Severity)),
				"fields": []map[string]interface{}{
					{
						"title": "Worker ID",
						"value": alert.WorkerID,
						"short": true,
					},
					{
						"title": "Drift Duration",
						"value": alert.DriftDuration.String(),
						"short": true,
					},
					{
						"title": "Current Version",
						"value": alert.CurrentVersion.CodebaseVersion,
						"short": true,
					},
					{
						"title": "Expected Version",
						"value": alert.ExpectedVersion.CodebaseVersion,
						"short": true,
					},
				},
				"text":   alert.Message,
				"footer": fmt.Sprintf("Alert ID: %s", alert.AlertID),
				"ts":     alert.Timestamp.Unix(),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// VersionCacheEntry represents a cached version check result
type VersionCacheEntry struct {
	VersionInfo VersionInfo
	Timestamp   time.Time
	TTL         time.Duration
}

// VersionManager handles version checking, updates, and validation for remote workers
type VersionManager struct {
	localVersion VersionInfo
	httpClient   *http.Client
	eventBus     *events.EventBus
	updateDir    string
	backupDir    string
	backups      map[string]*UpdateBackup // workerID -> backup
	metrics      *VersionMetrics
	alerts       []*DriftAlert
	alertManager *AlertManager
	versionCache map[string]*VersionCacheEntry // workerID -> cached version info
	cacheTTL     time.Duration
	baseURL      string // For testing: override the URL construction
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
		metrics:      &VersionMetrics{},
		alerts:       make([]*DriftAlert, 0),
		alertManager: NewAlertManager(1000),
		versionCache: make(map[string]*VersionCacheEntry),
		cacheTTL:     5 * time.Minute, // Cache version checks for 5 minutes
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
	// Check cache first
	if cached, exists := vm.versionCache[service.WorkerID]; exists && time.Since(cached.Timestamp) < cached.TTL {
		// Use cached version
		service.Version = cached.VersionInfo
		isUpToDate := vm.compareVersions(vm.localVersion, cached.VersionInfo)

		// Emit cached event
		event := events.Event{
			Type:      "worker_version_checked_cached",
			SessionID: "system",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"worker_id":      service.WorkerID,
				"local_version":  vm.localVersion.CodebaseVersion,
				"worker_version": cached.VersionInfo.CodebaseVersion,
				"up_to_date":     isUpToDate,
				"cached":         true,
			},
		}
		vm.eventBus.Publish(event)

		return isUpToDate, nil
	}

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

	// Update cache
	vm.versionCache[service.WorkerID] = &VersionCacheEntry{
		VersionInfo: workerVersion,
		Timestamp:   time.Now(),
		TTL:         vm.cacheTTL,
	}

	// Update service with version info
	service.Version = workerVersion

	// Compare versions
	isUpToDate := vm.compareVersions(vm.localVersion, workerVersion)

	// Record metrics
	if isUpToDate {
		// This will be counted in drift detection
	} else {
		// Could add per-worker metrics here if needed
	}

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
			"cached":         false,
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
	return vm.UpdateWorkerWithSigning(ctx, service, "", "")
}

// UpdateWorkerWithSigning updates a worker with optional signature verification
func (vm *VersionManager) UpdateWorkerWithSigning(ctx context.Context, service *RemoteService, privateKeyPath, expectedPublicKeyPath string) error {
	service.Status = "updating"

	// Emit update started event
	event := events.Event{
		Type:      "worker_update_started",
		SessionID: "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"worker_id":         service.WorkerID,
			"target_version":    vm.localVersion.CodebaseVersion,
			"current_version":   service.Version.CodebaseVersion,
			"signature_enabled": privateKeyPath != "",
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

	var updatePackage string
	var signedPackage *SignedUpdatePackage

	// Create update package (signed or unsigned)
	if privateKeyPath != "" {
		signedPackage, err = vm.createSignedUpdatePackage(privateKeyPath)
		if err != nil {
			vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
			return fmt.Errorf("failed to create signed update package: %w", err)
		}
		updatePackage = signedPackage.PackagePath
		backup.UpdatePackage = updatePackage
	} else {
		updatePackage, err = vm.createUpdatePackage()
		if err != nil {
			vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
			return fmt.Errorf("failed to create update package: %w", err)
		}
		backup.UpdatePackage = updatePackage
	}

	// Upload update package to worker
	if err := vm.uploadUpdatePackage(ctx, service, updatePackage); err != nil {
		vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
		return fmt.Errorf("failed to upload update package: %w", err)
	}

	// Upload signature and public key if signed
	if signedPackage != nil {
		if err := vm.uploadSignatureFiles(ctx, service, signedPackage); err != nil {
			vm.rollbackWorkerUpdate(ctx, service) // Rollback on failure
			return fmt.Errorf("failed to upload signature files: %w", err)
		}
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
			"signed":    signedPackage != nil,
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
	// Simplified worker installation process
	// In a real implementation, this would:
	// 1. Test connectivity to worker
	// 2. Transfer binaries via SCP/SFTP
	// 3. Install dependencies
	// 4. Configure service
	// 5. Verify installation

	// For now, simulate the installation process
	installSteps := []string{
		"checking_connectivity",
		"transferring_binaries",
		"installing_dependencies",
		"configuring_service",
		"verifying_installation",
	}

	for _, step := range installSteps {
		// Simulate step execution time
		select {
		case <-time.After(1 * time.Second):
			// Continue with next step
		case <-ctx.Done():
			return fmt.Errorf("installation cancelled during step: %s", step)
		}
	}

	// Record installation in metrics (simplified - just increment a counter)
	// Note: In real implementation, this would update proper metrics

	return nil
}

// GetMetrics returns current version management metrics
func (vm *VersionManager) GetMetrics() *VersionMetrics {
	return vm.metrics
}

// GetAlerts returns current version drift alerts
func (vm *VersionManager) GetAlerts() []*DriftAlert {
	return vm.alerts
}

// AddAlertChannel adds an alert notification channel
func (vm *VersionManager) AddAlertChannel(channel AlertChannel) {
	vm.alertManager.AddChannel(channel)
}

// BatchUpdateWorkers performs concurrent updates on multiple workers
func (vm *VersionManager) BatchUpdateWorkers(ctx context.Context, services []*RemoteService, maxConcurrency int) *BatchUpdateResult {
	if maxConcurrency <= 0 {
		maxConcurrency = 3 // Default concurrency
	}

	result := &BatchUpdateResult{
		TotalWorkers: len(services),
		Successful:   make([]string, 0),
		Failed:       make([]BatchUpdateError, 0),
		Skipped:      make([]string, 0),
		StartTime:    time.Now(),
	}

	// Use semaphore to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, service := range services {
		wg.Add(1)
		go func(svc *RemoteService) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if already up to date first
			upToDate, err := vm.CheckWorkerVersion(ctx, svc)
			if err != nil {
				mu.Lock()
				result.Failed = append(result.Failed, BatchUpdateError{
					WorkerID: svc.WorkerID,
					Error:    fmt.Sprintf("version check failed: %v", err),
				})
				mu.Unlock()
				return
			}

			if upToDate {
				mu.Lock()
				result.Skipped = append(result.Skipped, svc.WorkerID)
				mu.Unlock()
				return
			}

			// Perform update
			if err := vm.UpdateWorker(ctx, svc); err != nil {
				mu.Lock()
				result.Failed = append(result.Failed, BatchUpdateError{
					WorkerID: svc.WorkerID,
					Error:    fmt.Sprintf("update failed: %v", err),
				})
				mu.Unlock()
				return
			}

			// Success
			mu.Lock()
			result.Successful = append(result.Successful, svc.WorkerID)
			mu.Unlock()
		}(service)
	}

	wg.Wait()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// BatchUpdateResult contains the results of a batch update operation
type BatchUpdateResult struct {
	TotalWorkers int
	Successful   []string
	Failed       []BatchUpdateError
	Skipped      []string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
}

// BatchUpdateError represents an error that occurred during batch update
type BatchUpdateError struct {
	WorkerID string
	Error    string
}

// GetSuccessRate returns the success rate as a percentage
func (r *BatchUpdateResult) GetSuccessRate() float64 {
	if r.TotalWorkers == 0 {
		return 100.0
	}
	return float64(len(r.Successful)) / float64(r.TotalWorkers) * 100.0
}

// GetSummary returns a summary string of the batch update results
func (r *BatchUpdateResult) GetSummary() string {
	return fmt.Sprintf("Batch update completed: %d/%d successful (%.1f%%), %d failed, %d skipped in %v",
		len(r.Successful), r.TotalWorkers, r.GetSuccessRate(), len(r.Failed), len(r.Skipped), r.Duration)
}

// ClearCache clears the version check cache
func (vm *VersionManager) ClearCache() {
	vm.versionCache = make(map[string]*VersionCacheEntry)
}

// SetCacheTTL sets the cache TTL for version checks
func (vm *VersionManager) SetCacheTTL(ttl time.Duration) {
	if ttl > 0 {
		vm.cacheTTL = ttl
	}
}

// GetCacheStats returns cache statistics
func (vm *VersionManager) GetCacheStats() map[string]interface{} {
	totalEntries := len(vm.versionCache)
	now := time.Now()
	validEntries := 0
	expiredEntries := 0

	for _, entry := range vm.versionCache {
		if now.Sub(entry.Timestamp) < entry.TTL {
			validEntries++
		} else {
			expiredEntries++
		}
	}

	return map[string]interface{}{
		"total_entries":   totalEntries,
		"valid_entries":   validEntries,
		"expired_entries": expiredEntries,
		"cache_ttl":       vm.cacheTTL.String(),
		"hit_rate":        "N/A", // Would need hit/miss counters to calculate
	}
}

// GetAlertHistory returns alert history with optional limit
func (vm *VersionManager) GetAlertHistory(limit int) []*DriftAlert {
	return vm.alertManager.GetAlertHistory(limit)
}

// AcknowledgeAlert marks an alert as acknowledged
func (vm *VersionManager) AcknowledgeAlert(alertID, acknowledgedBy string) bool {
	return vm.alertManager.AcknowledgeAlert(alertID, acknowledgedBy)
}

// CheckVersionDrift performs comprehensive version drift detection across all workers
func (vm *VersionManager) CheckVersionDrift(ctx context.Context, services []*RemoteService) []*DriftAlert {
	alerts := make([]*DriftAlert, 0)
	now := time.Now()

	vm.metrics.LastDriftCheck = now
	vm.metrics.WorkersChecked = int64(len(services))

	upToDateCount := int64(0)
	outdatedCount := int64(0)
	unhealthyCount := int64(0)

	for _, service := range services {
		// Check version
		isUpToDate, err := vm.CheckWorkerVersion(ctx, service)
		if err != nil {
			unhealthyCount++
			alert := &DriftAlert{
				WorkerID:        service.WorkerID,
				CurrentVersion:  service.Version,
				ExpectedVersion: vm.localVersion,
				DriftDuration:   time.Since(service.LastSeen),
				Severity:        "high",
				Timestamp:       now,
				Message:         fmt.Sprintf("Worker %s is unreachable: %v", service.WorkerID, err),
			}
			alerts = append(alerts, alert)
			continue
		}

		if isUpToDate {
			upToDateCount++
		} else {
			outdatedCount++

			// Calculate drift duration
			driftDuration := now.Sub(service.Version.LastUpdated)
			if driftDuration > vm.metrics.MaxDriftDuration {
				vm.metrics.MaxDriftDuration = driftDuration
			}

			// Determine severity based on drift duration
			severity := vm.calculateDriftSeverity(driftDuration)

			alert := &DriftAlert{
				WorkerID:        service.WorkerID,
				CurrentVersion:  service.Version,
				ExpectedVersion: vm.localVersion,
				DriftDuration:   driftDuration,
				Severity:        severity,
				Timestamp:       now,
				Message: fmt.Sprintf("Worker %s is running version %s, expected %s (drift: %v)",
					service.WorkerID, service.Version.CodebaseVersion,
					vm.localVersion.CodebaseVersion, driftDuration),
			}
			alerts = append(alerts, alert)

			// Send alert through alert manager
			if err := vm.alertManager.SendAlert(alert); err != nil {
				// Log error but don't fail the drift check
				fmt.Printf("Failed to send alert for worker %s: %v\n", service.WorkerID, err)
			}
		}
	}

	// Update metrics
	vm.metrics.WorkersUpToDate = upToDateCount
	vm.metrics.WorkersOutdated = outdatedCount
	vm.metrics.WorkersUnhealthy = unhealthyCount

	// Store alerts
	vm.alerts = alerts

	// Emit drift check event
	event := events.Event{
		Type:      "version_drift_check_completed",
		SessionID: "system",
		Timestamp: now,
		Data: map[string]interface{}{
			"workers_checked":    vm.metrics.WorkersChecked,
			"workers_up_to_date": vm.metrics.WorkersUpToDate,
			"workers_outdated":   vm.metrics.WorkersOutdated,
			"workers_unhealthy":  vm.metrics.WorkersUnhealthy,
			"alerts_generated":   len(alerts),
		},
	}
	vm.eventBus.Publish(event)

	return alerts
}

// calculateDriftSeverity determines alert severity based on drift duration
func (vm *VersionManager) calculateDriftSeverity(driftDuration time.Duration) string {
	switch {
	case driftDuration > 24*time.Hour:
		return "critical"
	case driftDuration > 12*time.Hour:
		return "high"
	case driftDuration > 6*time.Hour:
		return "medium"
	default:
		return "low"
	}
}

// RecordUpdateMetrics records metrics for a completed update operation
func (vm *VersionManager) RecordUpdateMetrics(success bool, duration time.Duration) {
	vm.metrics.TotalUpdates++
	vm.metrics.LastUpdateTime = time.Now()

	if success {
		vm.metrics.SuccessfulUpdates++
	} else {
		vm.metrics.FailedUpdates++
	}

	// Update average duration (simple moving average)
	if vm.metrics.UpdateDuration == 0 {
		vm.metrics.UpdateDuration = duration
	} else {
		// Weighted average favoring recent measurements
		vm.metrics.UpdateDuration = (vm.metrics.UpdateDuration + duration) / 2
	}
}

// RecordRollbackMetrics records metrics for a completed rollback operation
func (vm *VersionManager) RecordRollbackMetrics(success bool, duration time.Duration) {
	vm.metrics.TotalRollbacks++
	vm.metrics.LastRollbackTime = time.Now()

	if success {
		vm.metrics.SuccessfulRollbacks++
	} else {
		vm.metrics.FailedRollbacks++
	}

	// Update average duration
	if vm.metrics.RollbackDuration == 0 {
		vm.metrics.RollbackDuration = duration
	} else {
		vm.metrics.RollbackDuration = (vm.metrics.RollbackDuration + duration) / 2
	}
}

// RecordSignatureMetrics records metrics for signature operations
func (vm *VersionManager) RecordSignatureMetrics(success bool) {
	vm.metrics.SignatureVerifications++

	if success {
		vm.metrics.SignatureSuccesses++
	} else {
		vm.metrics.SignatureFailures++
	}
}

// RecordBackupMetrics records metrics for backup operations
func (vm *VersionManager) RecordBackupMetrics() {
	vm.metrics.BackupsCreated++

	// Count active backups
	activeCount := int64(0)
	for _, backup := range vm.backups {
		if backup.Status == "active" {
			activeCount++
		}
	}
	vm.metrics.BackupsActive = activeCount
}

// GetHealthStatus returns overall health status of version management
func (vm *VersionManager) GetHealthStatus() map[string]interface{} {
	now := time.Now()
	driftCheckAge := now.Sub(vm.metrics.LastDriftCheck)

	// Calculate health score (0-100)
	healthScore := 100.0

	// Penalize for outdated workers
	if vm.metrics.WorkersChecked > 0 {
		outdatedRatio := float64(vm.metrics.WorkersOutdated) / float64(vm.metrics.WorkersChecked)
		healthScore -= outdatedRatio * 50
	}

	// Penalize for unhealthy workers
	if vm.metrics.WorkersChecked > 0 {
		unhealthyRatio := float64(vm.metrics.WorkersUnhealthy) / float64(vm.metrics.WorkersChecked)
		healthScore -= unhealthyRatio * 30
	}

	// Penalize for old drift checks
	if driftCheckAge > time.Hour {
		agePenalty := float64(driftCheckAge/time.Hour) * 5
		if agePenalty > 20 {
			agePenalty = 20
		}
		healthScore -= agePenalty
	}

	// Penalize for update failures
	if vm.metrics.TotalUpdates > 0 {
		failureRatio := float64(vm.metrics.FailedUpdates) / float64(vm.metrics.TotalUpdates)
		healthScore -= failureRatio * 10
	}

	if healthScore < 0 {
		healthScore = 0
	}

	status := "healthy"
	if healthScore < 70 {
		status = "warning"
	}
	if healthScore < 40 {
		status = "critical"
	}

	return map[string]interface{}{
		"status":                status,
		"health_score":          healthScore,
		"last_drift_check":      vm.metrics.LastDriftCheck,
		"drift_check_age":       driftCheckAge,
		"workers_checked":       vm.metrics.WorkersChecked,
		"workers_up_to_date":    vm.metrics.WorkersUpToDate,
		"workers_outdated":      vm.metrics.WorkersOutdated,
		"workers_unhealthy":     vm.metrics.WorkersUnhealthy,
		"active_alerts":         len(vm.alerts),
		"update_success_rate":   vm.calculateSuccessRate(vm.metrics.SuccessfulUpdates, vm.metrics.TotalUpdates),
		"rollback_success_rate": vm.calculateSuccessRate(vm.metrics.SuccessfulRollbacks, vm.metrics.TotalRollbacks),
	}
}

// calculateSuccessRate calculates success rate as percentage
func (vm *VersionManager) calculateSuccessRate(successes, total int64) float64 {
	if total == 0 {
		return 100.0
	}
	return float64(successes) / float64(total) * 100.0
}

// signUpdatePackage creates a digital signature for an update package
func (vm *VersionManager) signUpdatePackage(packagePath, privateKeyPath string) (string, error) {
	// Read the private key
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse the private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Read the package file
	packageData, err := os.ReadFile(packagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read package file: %w", err)
	}

	// Create hash of the package
	hash := sha256.Sum256(packageData)

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign package: %w", err)
	}

	// Create signature file path
	sigPath := packagePath + ".sig"

	// Write signature to file
	if err := os.WriteFile(sigPath, signature, 0644); err != nil {
		return "", fmt.Errorf("failed to write signature file: %w", err)
	}

	return sigPath, nil
}

// verifyUpdatePackage verifies the digital signature of an update package
func (vm *VersionManager) verifyUpdatePackage(packagePath, signaturePath, publicKeyPath string) error {
	// Read the public key
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse the public key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Read the package file
	packageData, err := os.ReadFile(packagePath)
	if err != nil {
		return fmt.Errorf("failed to read package file: %w", err)
	}

	// Read the signature
	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("failed to read signature file: %w", err)
	}

	// Create hash of the package
	hash := sha256.Sum256(packageData)

	// Verify the signature
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// generateSigningKeys generates a new RSA key pair for signing
func (vm *VersionManager) generateSigningKeys(keyDir string) (privateKeyPath, publicKeyPath string, err error) {
	// Ensure key directory exists
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create key directory: %w", err)
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyPath = filepath.Join(keyDir, "translator-signing-key.pem")
	privateFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateFile.Close()

	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		return "", "", fmt.Errorf("failed to encode private key: %w", err)
	}

	// Encode public key to PEM
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	}

	publicKeyPath = filepath.Join(keyDir, "translator-signing-key.pub")
	publicFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicFile.Close()

	if err := pem.Encode(publicFile, publicKeyPEM); err != nil {
		return "", "", fmt.Errorf("failed to encode public key: %w", err)
	}

	return privateKeyPath, publicKeyPath, nil
}

// createSignedUpdatePackage creates and signs an update package
func (vm *VersionManager) createSignedUpdatePackage(privateKeyPath string) (*SignedUpdatePackage, error) {
	// Create the update package
	packagePath, err := vm.createUpdatePackage()
	if err != nil {
		return nil, fmt.Errorf("failed to create update package: %w", err)
	}

	// Sign the package
	signaturePath, err := vm.signUpdatePackage(packagePath, privateKeyPath)
	if err != nil {
		os.Remove(packagePath) // Clean up on failure
		return nil, fmt.Errorf("failed to sign update package: %w", err)
	}

	// Get public key path (assume it's alongside private key)
	publicKeyPath := strings.TrimSuffix(privateKeyPath, ".pem") + ".pub"

	signedPackage := &SignedUpdatePackage{
		PackagePath:   packagePath,
		SignaturePath: signaturePath,
		PublicKeyPath: publicKeyPath,
		Version:       vm.localVersion.CodebaseVersion,
		Timestamp:     time.Now(),
	}

	return signedPackage, nil
}

// uploadSignatureFiles uploads signature and public key files to the worker
func (vm *VersionManager) uploadSignatureFiles(ctx context.Context, service *RemoteService, signedPackage *SignedUpdatePackage) error {
	// Upload signature file
	if err := vm.uploadFileToWorker(ctx, service, signedPackage.SignaturePath, "signature"); err != nil {
		return fmt.Errorf("failed to upload signature file: %w", err)
	}

	// Upload public key file
	if err := vm.uploadFileToWorker(ctx, service, signedPackage.PublicKeyPath, "public_key"); err != nil {
		return fmt.Errorf("failed to upload public key file: %w", err)
	}

	return nil
}

// uploadFileToWorker uploads a file to the worker with a specific type
func (vm *VersionManager) uploadFileToWorker(ctx context.Context, service *RemoteService, filePath, fileType string) error {
	var uploadURL string
	if vm.baseURL != "" {
		uploadURL = fmt.Sprintf("%s/api/v1/update/upload/%s", vm.baseURL, fileType)
	} else {
		uploadURL = fmt.Sprintf("%s://%s:%d/api/v1/update/upload/%s", service.Protocol, service.Host, service.Port, fileType)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-File-Type", fileType)
	req.Header.Set("X-Update-Version", vm.localVersion.CodebaseVersion)

	resp, err := vm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("file upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
