package models

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader manages model downloads and caching
type Downloader struct {
	cacheDir string
	client   *http.Client
}

// NewDownloader creates a new model downloader
func NewDownloader() *Downloader {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "translator", "models")
	os.MkdirAll(cacheDir, 0755)

	return &Downloader{
		cacheDir: cacheDir,
		client: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large model downloads
		},
	}
}

// GetModelPath returns the path to a model if it exists, or error if not downloaded
func (d *Downloader) GetModelPath(model *ModelInfo) (string, error) {
	modelPath := d.getLocalPath(model)

	// Check if model file exists and is valid
	if stat, err := os.Stat(modelPath); err == nil {
		// File exists, verify it's not corrupted (basic size check)
		if stat.Size() > 1024*1024 { // At least 1MB
			return modelPath, nil
		}
		// File too small, likely corrupted
		os.Remove(modelPath)
	}

	return "", fmt.Errorf("model not downloaded: %s", model.ID)
}

// DownloadModel downloads a model if not already cached
func (d *Downloader) DownloadModel(model *ModelInfo) (string, error) {
	modelPath := d.getLocalPath(model)

	// Check if already downloaded
	if _, err := os.Stat(modelPath); err == nil {
		fmt.Printf("[DOWNLOADER] Model already exists: %s\n", modelPath)
		return modelPath, nil
	}

	// Download model
	fmt.Printf("[DOWNLOADER] Downloading %s from %s\n", model.Name, model.SourceURL)
	fmt.Printf("[DOWNLOADER] This may take several minutes...\n")

	// Create temporary file
	tmpPath := modelPath + ".tmp"
	defer os.Remove(tmpPath) // Clean up on error

	// Download with progress
	err := d.downloadWithProgress(model.SourceURL, tmpPath)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	// Verify downloaded file (basic checks)
	stat, err := os.Stat(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	// Check file size is reasonable (at least 100MB for any LLM)
	if stat.Size() < 100*1024*1024 {
		return "", fmt.Errorf("downloaded file too small: %d bytes (expected > 100MB)", stat.Size())
	}

	fmt.Printf("[DOWNLOADER] Download complete: %.1f GB\n", float64(stat.Size())/(1024*1024*1024))

	// Move to final location
	err = os.Rename(tmpPath, modelPath)
	if err != nil {
		return "", fmt.Errorf("failed to move downloaded file: %w", err)
	}

	fmt.Printf("[DOWNLOADER] Model ready: %s\n", modelPath)

	return modelPath, nil
}

// downloadWithProgress downloads a file and shows progress
func (d *Downloader) downloadWithProgress(url, destPath string) error {
	// Create output file
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := d.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Get content length for progress reporting
	size := resp.ContentLength

	// Create progress writer
	writer := &progressWriter{
		writer:      out,
		total:       size,
		lastPrint:   time.Now(),
		startTime:   time.Now(),
	}

	// Copy with progress
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return err
	}

	// Final progress update
	writer.printProgress(true)
	fmt.Println() // New line after progress

	return nil
}

// progressWriter wraps a writer to show download progress
type progressWriter struct {
	writer     io.Writer
	total      int64
	downloaded int64
	lastPrint  time.Time
	startTime  time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.downloaded += int64(n)

	// Print progress every second
	if time.Since(pw.lastPrint) >= time.Second {
		pw.printProgress(false)
		pw.lastPrint = time.Now()
	}

	return n, nil
}

func (pw *progressWriter) printProgress(final bool) {
	if pw.total <= 0 {
		// Unknown size, just show downloaded amount
		fmt.Printf("\r[DOWNLOADER] Downloaded: %.1f MB", float64(pw.downloaded)/(1024*1024))
		return
	}

	// Calculate percentage and speed
	percent := float64(pw.downloaded) / float64(pw.total) * 100
	elapsed := time.Since(pw.startTime).Seconds()
	speed := float64(pw.downloaded) / elapsed / (1024 * 1024) // MB/s

	// Estimate time remaining
	if speed > 0 {
		remaining := float64(pw.total-pw.downloaded) / (speed * 1024 * 1024)
		remainingMin := int(remaining / 60)
		remainingSec := int(remaining) % 60

		if final {
			fmt.Printf("\r[DOWNLOADER] Complete: %.1f GB downloaded in %.0fs (%.1f MB/s)",
				float64(pw.total)/(1024*1024*1024), elapsed, speed)
		} else {
			fmt.Printf("\r[DOWNLOADER] Progress: %.1f%% (%.1f MB/s, ~%dm%ds remaining)",
				percent, speed, remainingMin, remainingSec)
		}
	} else {
		fmt.Printf("\r[DOWNLOADER] Progress: %.1f%%", percent)
	}
}

// getLocalPath returns the local filesystem path for a model
func (d *Downloader) getLocalPath(model *ModelInfo) string {
	// Use .gguf extension for GGUF models
	filename := model.ID + ".gguf"
	return filepath.Join(d.cacheDir, filename)
}

// ListDownloadedModels returns a list of all downloaded models
func (d *Downloader) ListDownloadedModels() ([]string, error) {
	files, err := os.ReadDir(d.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var models []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gguf") {
			// Remove .gguf extension to get model ID
			modelID := strings.TrimSuffix(file.Name(), ".gguf")
			models = append(models, modelID)
		}
	}

	return models, nil
}

// DeleteModel removes a downloaded model from cache
func (d *Downloader) DeleteModel(modelID string) error {
	modelPath := filepath.Join(d.cacheDir, modelID+".gguf")

	err := os.Remove(modelPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("model not found: %s", modelID)
		}
		return err
	}

	fmt.Printf("[DOWNLOADER] Deleted model: %s\n", modelID)
	return nil
}

// GetCacheSize returns the total size of downloaded models in bytes
func (d *Downloader) GetCacheSize() (int64, error) {
	var totalSize int64

	files, err := os.ReadDir(d.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gguf") {
			info, err := file.Info()
			if err != nil {
				continue
			}
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}

// CleanCache removes all downloaded models
func (d *Downloader) CleanCache() error {
	files, err := os.ReadDir(d.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var errors []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".gguf") {
			filePath := filepath.Join(d.cacheDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				errors = append(errors, fmt.Sprintf("failed to delete %s: %v", file.Name(), err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during cache cleaning: %s", strings.Join(errors, "; "))
	}

	fmt.Println("[DOWNLOADER] Cache cleaned")
	return nil
}

// VerifyModel verifies the integrity of a downloaded model
func (d *Downloader) VerifyModel(modelID string) error {
	modelPath := filepath.Join(d.cacheDir, modelID+".gguf")

	// Check if file exists
	stat, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("model file not found: %w", err)
	}

	// Check file size is reasonable
	if stat.Size() < 100*1024*1024 {
		return fmt.Errorf("model file too small: %d bytes (possibly corrupted)", stat.Size())
	}

	// Read first few bytes to verify it's a GGUF file
	file, err := os.Open(modelPath)
	if err != nil {
		return fmt.Errorf("failed to open model file: %w", err)
	}
	defer file.Close()

	// GGUF files start with "GGUF" magic number
	magic := make([]byte, 4)
	n, err := file.Read(magic)
	if err != nil || n != 4 {
		return fmt.Errorf("failed to read model header: %w", err)
	}

	if string(magic) != "GGUF" {
		return fmt.Errorf("invalid model file: not a GGUF file")
	}

	return nil
}

// ComputeChecksum computes SHA256 checksum of a model file
func (d *Downloader) ComputeChecksum(modelID string) (string, error) {
	modelPath := filepath.Join(d.cacheDir, modelID+".gguf")

	file, err := os.Open(modelPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
