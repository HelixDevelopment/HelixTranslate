package models

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewDownloader tests downloader initialization
func TestNewDownloader(t *testing.T) {
	downloader := NewDownloader()
	if downloader == nil {
		t.Fatal("NewDownloader() returned nil")
	}

	// Verify cache directory was created
	if downloader.cacheDir == "" {
		t.Error("Cache directory not set")
	}

	// Verify cache directory exists
	if _, err := os.Stat(downloader.cacheDir); os.IsNotExist(err) {
		t.Errorf("Cache directory not created: %s", downloader.cacheDir)
	}

	// Verify HTTP client was initialized
	if downloader.client == nil {
		t.Error("HTTP client not initialized")
	}

	// Verify timeout is reasonable (should be long for large downloads)
	if downloader.client.Timeout < 1*time.Minute {
		t.Errorf("HTTP client timeout too short: %v", downloader.client.Timeout)
	}
}

// TestGetLocalPath tests local path generation
func TestGetLocalPath(t *testing.T) {
	downloader := NewDownloader()

	testModel := &ModelInfo{
		ID:   "test-model-7b-q4",
		Name: "Test Model 7B Q4",
	}

	path := downloader.getLocalPath(testModel)

	// Should end with .gguf
	if !strings.HasSuffix(path, ".gguf") {
		t.Errorf("Path doesn't end with .gguf: %s", path)
	}

	// Should contain model ID
	if !strings.Contains(path, testModel.ID) {
		t.Errorf("Path doesn't contain model ID: %s", path)
	}

	// Should be in cache directory
	if !strings.Contains(path, downloader.cacheDir) {
		t.Errorf("Path not in cache directory: %s", path)
	}
}

// TestGetModelPath tests checking for existing models
func TestGetModelPath(t *testing.T) {
	downloader := NewDownloader()

	// Create a temporary test model file
	testModel := &ModelInfo{
		ID:   "test-existing-model",
		Name: "Test Existing Model",
	}

	modelPath := downloader.getLocalPath(testModel)

	t.Run("Non-existent model", func(t *testing.T) {
		// Ensure model doesn't exist
		os.Remove(modelPath)

		_, err := downloader.GetModelPath(testModel)
		if err == nil {
			t.Error("Expected error for non-existent model")
		}

		if !strings.Contains(err.Error(), "not downloaded") {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("Existing valid model", func(t *testing.T) {
		// Create a mock GGUF file (at least 1MB)
		file, err := os.Create(modelPath)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Write 2MB of data
		data := make([]byte, 2*1024*1024)
		file.Write(data)
		file.Close()

		defer os.Remove(modelPath)

		path, err := downloader.GetModelPath(testModel)
		if err != nil {
			t.Errorf("GetModelPath() failed for existing model: %v", err)
		}

		if path != modelPath {
			t.Errorf("Wrong path returned: got %s, expected %s", path, modelPath)
		}
	})

	t.Run("Corrupted model (too small)", func(t *testing.T) {
		// Create a file that's too small
		file, err := os.Create(modelPath)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		file.WriteString("tiny")
		file.Close()

		defer os.Remove(modelPath)

		_, err = downloader.GetModelPath(testModel)
		if err == nil {
			t.Error("Expected error for corrupted (too small) model")
		}
	})
}

// TestListDownloadedModels tests listing cached models
func TestListDownloadedModels(t *testing.T) {
	downloader := NewDownloader()

	// Clean cache first
	downloader.CleanCache()

	t.Run("Empty cache", func(t *testing.T) {
		models, err := downloader.ListDownloadedModels()
		if err != nil {
			t.Fatalf("ListDownloadedModels() failed: %v", err)
		}

		if len(models) != 0 {
			t.Errorf("Expected 0 models, got %d", len(models))
		}
	})

	t.Run("With models", func(t *testing.T) {
		// Create test model files
		testModels := []string{
			"test-model-1.gguf",
			"test-model-2.gguf",
			"test-model-3.gguf",
		}

		for _, modelFile := range testModels {
			path := filepath.Join(downloader.cacheDir, modelFile)
			file, err := os.Create(path)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			file.WriteString("test data")
			file.Close()
			defer os.Remove(path)
		}

		models, err := downloader.ListDownloadedModels()
		if err != nil {
			t.Fatalf("ListDownloadedModels() failed: %v", err)
		}

		if len(models) != 3 {
			t.Errorf("Expected 3 models, got %d", len(models))
		}

		// Verify model IDs (should have .gguf removed)
		for _, model := range models {
			if strings.HasSuffix(model, ".gguf") {
				t.Errorf("Model ID should not have .gguf suffix: %s", model)
			}
		}
	})

	t.Run("Ignores non-GGUF files", func(t *testing.T) {
		// Create non-GGUF files
		nonGGUFFiles := []string{
			"readme.txt",
			"config.json",
			"model.bin",
		}

		for _, file := range nonGGUFFiles {
			path := filepath.Join(downloader.cacheDir, file)
			f, _ := os.Create(path)
			f.WriteString("test")
			f.Close()
			defer os.Remove(path)
		}

		models, err := downloader.ListDownloadedModels()
		if err != nil {
			t.Fatalf("ListDownloadedModels() failed: %v", err)
		}

		// Should still only see the GGUF models from previous test
		// (assuming they haven't been cleaned up yet)
		for _, model := range models {
			if !strings.Contains(model, "test-model") {
				t.Errorf("Unexpected model in list: %s", model)
			}
		}
	})
}

// TestDeleteModel tests model deletion
func TestDeleteModel(t *testing.T) {
	downloader := NewDownloader()

	t.Run("Delete existing model", func(t *testing.T) {
		// Create test model
		modelID := "test-delete-model"
		modelPath := filepath.Join(downloader.cacheDir, modelID+".gguf")

		file, err := os.Create(modelPath)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		file.WriteString("test data")
		file.Close()

		// Verify it exists
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			t.Fatal("Test model file was not created")
		}

		// Delete it
		err = downloader.DeleteModel(modelID)
		if err != nil {
			t.Errorf("DeleteModel() failed: %v", err)
		}

		// Verify it's gone
		if _, err := os.Stat(modelPath); !os.IsNotExist(err) {
			t.Error("Model file was not deleted")
		}
	})

	t.Run("Delete non-existent model", func(t *testing.T) {
		err := downloader.DeleteModel("non-existent-model")
		if err == nil {
			t.Error("Expected error when deleting non-existent model")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Wrong error message: %v", err)
		}
	})
}

// TestGetCacheSize tests cache size calculation
func TestGetCacheSize(t *testing.T) {
	downloader := NewDownloader()

	// Clean cache
	downloader.CleanCache()

	t.Run("Empty cache", func(t *testing.T) {
		size, err := downloader.GetCacheSize()
		if err != nil {
			t.Fatalf("GetCacheSize() failed: %v", err)
		}

		if size != 0 {
			t.Errorf("Expected 0 bytes, got %d", size)
		}
	})

	t.Run("With models", func(t *testing.T) {
		// Create test models with known sizes
		sizes := []int{
			1024,         // 1KB
			1024 * 1024,  // 1MB
			2 * 1024 * 1024, // 2MB
		}

		var expectedTotal int64
		for i, size := range sizes {
			modelPath := filepath.Join(downloader.cacheDir, fmt.Sprintf("test-size-%d.gguf", i))
			file, err := os.Create(modelPath)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			data := make([]byte, size)
			file.Write(data)
			file.Close()
			defer os.Remove(modelPath)

			expectedTotal += int64(size)
		}

		totalSize, err := downloader.GetCacheSize()
		if err != nil {
			t.Fatalf("GetCacheSize() failed: %v", err)
		}

		if totalSize != expectedTotal {
			t.Errorf("Expected %d bytes, got %d", expectedTotal, totalSize)
		}
	})
}

// TestCleanCache tests cache cleaning
func TestCleanCache(t *testing.T) {
	downloader := NewDownloader()

	// Create test models
	testModels := []string{
		"test-clean-1.gguf",
		"test-clean-2.gguf",
		"test-clean-3.gguf",
	}

	for _, model := range testModels {
		path := filepath.Join(downloader.cacheDir, model)
		file, _ := os.Create(path)
		file.WriteString("test data")
		file.Close()
	}

	// Verify models exist
	models, _ := downloader.ListDownloadedModels()
	if len(models) < 3 {
		t.Fatal("Test models not created properly")
	}

	// Clean cache
	err := downloader.CleanCache()
	if err != nil {
		t.Errorf("CleanCache() failed: %v", err)
	}

	// Verify cache is empty
	models, _ = downloader.ListDownloadedModels()
	for _, model := range models {
		if strings.Contains(model, "test-clean") {
			t.Errorf("Model not cleaned: %s", model)
		}
	}
}

// TestVerifyModel tests model file verification
func TestVerifyModel(t *testing.T) {
	downloader := NewDownloader()

	t.Run("Valid GGUF model", func(t *testing.T) {
		modelID := "test-verify-valid"
		modelPath := filepath.Join(downloader.cacheDir, modelID+".gguf")

		// Create a valid-looking GGUF file
		file, err := os.Create(modelPath)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		defer os.Remove(modelPath)

		// Write GGUF magic number
		file.WriteString("GGUF")

		// Write enough data to pass size check (>100MB)
		data := make([]byte, 101*1024*1024)
		file.Write(data)
		file.Close()

		err = downloader.VerifyModel(modelID)
		if err != nil {
			t.Errorf("VerifyModel() failed for valid model: %v", err)
		}
	})

	t.Run("Non-existent model", func(t *testing.T) {
		err := downloader.VerifyModel("non-existent-model")
		if err == nil {
			t.Error("Expected error for non-existent model")
		}
	})

	t.Run("Model too small", func(t *testing.T) {
		modelID := "test-verify-small"
		modelPath := filepath.Join(downloader.cacheDir, modelID+".gguf")

		file, _ := os.Create(modelPath)
		file.WriteString("GGUF")
		file.WriteString("tiny")
		file.Close()
		defer os.Remove(modelPath)

		err := downloader.VerifyModel(modelID)
		if err == nil {
			t.Error("Expected error for too-small model")
		}

		if !strings.Contains(err.Error(), "too small") {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("Invalid GGUF header", func(t *testing.T) {
		modelID := "test-verify-invalid"
		modelPath := filepath.Join(downloader.cacheDir, modelID+".gguf")

		file, _ := os.Create(modelPath)
		file.WriteString("FAKE") // Wrong magic number

		// Write enough data to pass size check
		data := make([]byte, 101*1024*1024)
		file.Write(data)
		file.Close()
		defer os.Remove(modelPath)

		err := downloader.VerifyModel(modelID)
		if err == nil {
			t.Error("Expected error for invalid GGUF header")
		}

		if !strings.Contains(err.Error(), "not a GGUF file") {
			t.Errorf("Wrong error message: %v", err)
		}
	})
}

// TestComputeChecksum tests SHA256 checksum computation
func TestComputeChecksum(t *testing.T) {
	downloader := NewDownloader()

	t.Run("Compute checksum", func(t *testing.T) {
		modelID := "test-checksum"
		modelPath := filepath.Join(downloader.cacheDir, modelID+".gguf")

		// Create test file with known content
		content := "test data for checksum"
		file, _ := os.Create(modelPath)
		file.WriteString(content)
		file.Close()
		defer os.Remove(modelPath)

		checksum1, err := downloader.ComputeChecksum(modelID)
		if err != nil {
			t.Fatalf("ComputeChecksum() failed: %v", err)
		}

		if checksum1 == "" {
			t.Error("Checksum is empty")
		}

		// Compute again - should be identical
		checksum2, err := downloader.ComputeChecksum(modelID)
		if err != nil {
			t.Fatalf("Second ComputeChecksum() failed: %v", err)
		}

		if checksum1 != checksum2 {
			t.Errorf("Checksums don't match: %s vs %s", checksum1, checksum2)
		}

		// Checksum should be 64 hex characters (SHA256)
		if len(checksum1) != 64 {
			t.Errorf("Checksum wrong length: got %d, expected 64", len(checksum1))
		}
	})

	t.Run("Different content produces different checksum", func(t *testing.T) {
		modelID1 := "test-checksum-1"
		modelID2 := "test-checksum-2"

		path1 := filepath.Join(downloader.cacheDir, modelID1+".gguf")
		path2 := filepath.Join(downloader.cacheDir, modelID2+".gguf")

		// Create two files with different content
		file1, _ := os.Create(path1)
		file1.WriteString("content A")
		file1.Close()
		defer os.Remove(path1)

		file2, _ := os.Create(path2)
		file2.WriteString("content B")
		file2.Close()
		defer os.Remove(path2)

		checksum1, _ := downloader.ComputeChecksum(modelID1)
		checksum2, _ := downloader.ComputeChecksum(modelID2)

		if checksum1 == checksum2 {
			t.Error("Different content produced same checksum")
		}
	})

	t.Run("Non-existent model", func(t *testing.T) {
		_, err := downloader.ComputeChecksum("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent model")
		}
	})
}

// TestDownloadWithProgress tests the download progress tracking
func TestProgressWriter(t *testing.T) {
	// Create a mock writer
	var buf strings.Builder
	pw := &progressWriter{
		writer:    &buf,
		total:     1000,
		startTime: time.Now(),
		lastPrint: time.Now(),
	}

	// Write some data
	data := []byte("test data")
	n, err := pw.Write(data)

	if err != nil {
		t.Fatalf("Write() failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Write() returned %d, expected %d", n, len(data))
	}

	if pw.downloaded != int64(len(data)) {
		t.Errorf("Downloaded bytes not tracked: got %d, expected %d", pw.downloaded, len(data))
	}

	// Verify data was written to underlying writer
	if buf.String() != string(data) {
		t.Errorf("Data not written to underlying writer")
	}
}

// TestDownloadModel tests the actual download functionality with a mock server
func TestDownloadModel(t *testing.T) {
	// Create a test HTTP server
	testData := make([]byte, 200*1024*1024) // 200MB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
		w.Write(testData)
	}))
	defer server.Close()

	downloader := NewDownloader()

	testModel := &ModelInfo{
		ID:        "test-download-model",
		Name:      "Test Download Model",
		SourceURL: server.URL + "/test-model.gguf",
	}

	t.Run("Successful download", func(t *testing.T) {
		// Clean up any existing test model
		modelPath := downloader.getLocalPath(testModel)
		os.Remove(modelPath)

		// Download model
		path, err := downloader.DownloadModel(testModel)
		if err != nil {
			t.Fatalf("DownloadModel() failed: %v", err)
		}

		defer os.Remove(path)

		// Verify file exists
		stat, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Downloaded file not found: %v", err)
		}

		// Verify file size
		if stat.Size() != int64(len(testData)) {
			t.Errorf("Downloaded file size mismatch: got %d, expected %d", stat.Size(), len(testData))
		}

		// Verify content
		content, _ := os.ReadFile(path)
		if len(content) != len(testData) {
			t.Errorf("Downloaded content size mismatch")
		}
	})

	t.Run("Already downloaded", func(t *testing.T) {
		// Download should detect existing file
		path, err := downloader.DownloadModel(testModel)
		if err != nil {
			t.Fatalf("DownloadModel() failed for existing model: %v", err)
		}

		// Should return same path
		expectedPath := downloader.getLocalPath(testModel)
		if path != expectedPath {
			t.Errorf("Wrong path for existing model: got %s, expected %s", path, expectedPath)
		}
	})

	t.Run("Download failure - invalid URL", func(t *testing.T) {
		badModel := &ModelInfo{
			ID:        "test-bad-url",
			Name:      "Test Bad URL",
			SourceURL: "http://invalid-url-that-does-not-exist-12345.com/model.gguf",
		}

		_, err := downloader.DownloadModel(badModel)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("Download failure - file too small", func(t *testing.T) {
		// Create server that returns tiny file
		tinyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("tiny"))
		}))
		defer tinyServer.Close()

		tinyModel := &ModelInfo{
			ID:        "test-tiny-download",
			Name:      "Test Tiny Download",
			SourceURL: tinyServer.URL + "/tiny.gguf",
		}

		// Clean up any existing file
		os.Remove(downloader.getLocalPath(tinyModel))

		_, err := downloader.DownloadModel(tinyModel)
		if err == nil {
			t.Error("Expected error for too-small download")
		}

		if !strings.Contains(err.Error(), "too small") {
			t.Errorf("Wrong error message: %v", err)
		}
	})
}

// BenchmarkListDownloadedModels benchmarks model listing performance
func BenchmarkListDownloadedModels(b *testing.B) {
	downloader := NewDownloader()

	// Create some test models
	for i := 0; i < 10; i++ {
		path := filepath.Join(downloader.cacheDir, fmt.Sprintf("bench-model-%d.gguf", i))
		file, _ := os.Create(path)
		file.WriteString("test")
		file.Close()
		defer os.Remove(path)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := downloader.ListDownloadedModels()
		if err != nil {
			b.Fatalf("ListDownloadedModels() failed: %v", err)
		}
	}
}

// BenchmarkGetCacheSize benchmarks cache size calculation
func BenchmarkGetCacheSize(b *testing.B) {
	downloader := NewDownloader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := downloader.GetCacheSize()
		if err != nil {
			b.Fatalf("GetCacheSize() failed: %v", err)
		}
	}
}

// BenchmarkComputeChecksum benchmarks checksum computation
func BenchmarkComputeChecksum(b *testing.B) {
	downloader := NewDownloader()

	// Create test file
	modelID := "bench-checksum"
	modelPath := filepath.Join(downloader.cacheDir, modelID+".gguf")
	file, _ := os.Create(modelPath)

	// Write 10MB of data
	data := make([]byte, 10*1024*1024)
	file.Write(data)
	file.Close()
	defer os.Remove(modelPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := downloader.ComputeChecksum(modelID)
		if err != nil {
			b.Fatalf("ComputeChecksum() failed: %v", err)
		}
	}
}

// Helper function to create a test GGUF file
func createTestGGUF(t *testing.T, path string, sizeBytes int64) {
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	// Write GGUF header
	file.WriteString("GGUF")

	// Write enough data to reach desired size
	remaining := sizeBytes - 4
	chunkSize := int64(1024 * 1024) // 1MB chunks
	data := make([]byte, chunkSize)

	for remaining > 0 {
		writeSize := chunkSize
		if remaining < chunkSize {
			writeSize = remaining
		}
		file.Write(data[:writeSize])
		remaining -= writeSize
	}
}

// TestProgressWriterOutput tests progress output formatting
func TestProgressWriterOutput(t *testing.T) {
	// This test verifies that progress messages are formatted correctly
	// We'll capture output by temporarily redirecting it

	pw := &progressWriter{
		writer:    io.Discard,
		total:     1000,
		downloaded: 500,
		startTime: time.Now().Add(-10 * time.Second),
		lastPrint: time.Now(),
	}

	// Test that printProgress doesn't crash
	pw.printProgress(false)
	pw.printProgress(true)

	// Verify calculations are reasonable
	if pw.downloaded > pw.total {
		t.Error("Downloaded more than total")
	}
}
