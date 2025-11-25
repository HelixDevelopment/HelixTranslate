package llm

import (
	"context"
	"digital.vasic.translator/pkg/hardware"
	"digital.vasic.translator/pkg/models"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
)

// TestFindLlamaCppExecutable tests locating llama-cli
func TestFindLlamaCppExecutable(t *testing.T) {
	// Store original PATH and HOME
	originalPath := os.Getenv("PATH")
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("PATH", originalPath)
		os.Setenv("HOME", originalHome)
	}()

	t.Run("executable_found_in_system", func(t *testing.T) {
		path, err := findLlamaCppExecutable()

		if err != nil {
			// llama.cpp not installed - skip integration tests
			t.Skip("llama.cpp not installed - install with: brew install llama.cpp")
			return
		}

		if path == "" {
			t.Error("findLlamaCppExecutable() returned empty path")
		}

		// Verify executable exists and is executable
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Executable not found at %s: %v", path, err)
		}

		// Check if it's executable (Unix-like systems)
		if info.Mode()&0111 == 0 {
			t.Errorf("File at %s is not executable", path)
		}

		t.Logf("Found llama-cli at: %s", path)
	})

	t.Run("executable_not_found_with_empty_path", func(t *testing.T) {
		// Set PATH to empty to ensure llama-cli is not found
		os.Setenv("PATH", "")
		os.Setenv("HOME", "/nonexistent/home")
		
		path, err := findLlamaCppExecutable()
		assert.Error(t, err)
		assert.Empty(t, path)
		assert.Contains(t, err.Error(), "llama-cli not found")
	})
}

// TestNewLlamaCppClient tests client initialization
func TestNewLlamaCppClient(t *testing.T) {
	// Check if llama.cpp is available
	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	// Note: This test will attempt to auto-download a model
	// We use a minimal config to test initialization
	config := TranslationConfig{
		Provider: "llamacpp",
		// Don't specify a model - let it auto-select
	}

	t.Run("Auto-select model", func(t *testing.T) {
		client, err := NewLlamaCppClient(config)

		if err != nil {
			// If error is about model download/selection, that's acceptable
			if strings.Contains(err.Error(), "download") || strings.Contains(err.Error(), "not found") {
				t.Skipf("Skipping test - no suitable model available: %v", err)
			}
			t.Fatalf("NewLlamaCppClient() failed: %v", err)
		}

		if client == nil {
			t.Fatal("NewLlamaCppClient() returned nil client")
		}

		// Verify fields are initialized
		if client.modelPath == "" {
			t.Error("Model path not set")
		}

		if client.modelInfo == nil {
			t.Error("Model info not set")
		}

		if client.hardwareCaps == nil {
			t.Error("Hardware capabilities not detected")
		}

		if client.threads < 1 {
			t.Errorf("Invalid thread count: %d", client.threads)
		}

		if client.contextSize < 1 {
			t.Errorf("Invalid context size: %d", client.contextSize)
		}

		t.Logf("Client initialized:")
		t.Logf("  Model: %s", client.modelInfo.Name)
		t.Logf("  Model Path: %s", client.modelPath)
		t.Logf("  Threads: %d", client.threads)
		t.Logf("  Context Size: %d", client.contextSize)
		t.Logf("  GPU: %v (%s)", client.hardwareCaps.HasGPU, client.hardwareCaps.GPUType)
	})

	t.Run("Specific model selection", func(t *testing.T) {
		// Try to use Hunyuan-MT-7B Q4 (recommended for translation)
		configWithModel := TranslationConfig{
			Provider: "llamacpp",
			Model:    "hunyuan-mt-7b-q4",
		}

		client, err := NewLlamaCppClient(configWithModel)

		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "download") {
				t.Skip("Hunyuan-MT-7B Q4 not available - test requires model to be downloaded")
			}
			if strings.Contains(err.Error(), "insufficient resources") {
				t.Skip("Insufficient resources for model")
			}
			t.Fatalf("NewLlamaCppClient() with specific model failed: %v", err)
		}

		if client.modelInfo.ID != "hunyuan-mt-7b-q4" {
			t.Errorf("Wrong model selected: got %s, expected hunyuan-mt-7b-q4", client.modelInfo.ID)
		}
	})
}

// TestGetProviderName tests provider name
func TestGetProviderName(t *testing.T) {
	client := &LlamaCppClient{}
	name := client.GetProviderName()

	if name != "llamacpp" {
		t.Errorf("GetProviderName() = %s, expected llamacpp", name)
	}
}

// TestValidate tests client validation
func TestValidate(t *testing.T) {
	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	// Create a client
	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client for validation test: %v", err)
	}

	t.Run("Valid client", func(t *testing.T) {
		err := client.Validate()
		if err != nil {
			t.Errorf("Validate() failed for valid client: %v", err)
		}
	})

	t.Run("Invalid model path", func(t *testing.T) {
		// Create client with invalid model path
		invalidClient := &LlamaCppClient{
			modelPath:    "/nonexistent/model.gguf",
			modelInfo:    &models.ModelInfo{MinRAM: 1024 * 1024 * 1024},
			hardwareCaps: &hardware.Capabilities{AvailableRAM: 16 * 1024 * 1024 * 1024},
			executable:   client.executable,
		}

		err := invalidClient.Validate()
		if err == nil {
			t.Error("Expected error for invalid model path")
		}
	})

	t.Run("Insufficient RAM", func(t *testing.T) {
		// Create client with insufficient RAM
		insufficientClient := &LlamaCppClient{
			modelPath:    client.modelPath,
			modelInfo:    &models.ModelInfo{MinRAM: 100 * 1024 * 1024 * 1024}, // 100GB
			hardwareCaps: &hardware.Capabilities{AvailableRAM: 1 * 1024 * 1024 * 1024}, // 1GB
			executable:   client.executable,
		}

		err := insufficientClient.Validate()
		if err == nil {
			t.Error("Expected error for insufficient RAM")
		}

		if !strings.Contains(err.Error(), "insufficient RAM") {
			t.Errorf("Wrong error message: %v", err)
		}
	})
}

// INTEGRATION TEST - Requires real llama.cpp and model
func TestTranslate_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if llama.cpp is available
	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed - install with: brew install llama.cpp")
	}

	// Create client
	config := TranslationConfig{
		Provider: "llamacpp",
		// Let it auto-select the best available model
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client: %v", err)
	}

	// Validate client
	if err := client.Validate(); err != nil {
		t.Skipf("Client validation failed: %v", err)
	}

	t.Run("Simple translation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Test text (Russian)
		testText := "Привет, мир!"

		// Create translation prompt
		prompt := `Translate the following Russian text to Serbian (Cyrillic):

Russian: Привет, мир!
Serbian:`

		t.Logf("Translating: %s", testText)
		t.Logf("Using model: %s", client.modelInfo.Name)

		startTime := time.Now()
		result, err := client.Translate(ctx, testText, prompt)
		duration := time.Since(startTime)

		if err != nil {
			t.Fatalf("Translate() failed: %v", err)
		}

		if result == "" {
			t.Error("Translation returned empty result")
		}

		t.Logf("Translation result: %s", result)
		t.Logf("Translation took: %v", duration)

		// Basic validation - result should not be the same as input
		if result == testText {
			t.Error("Translation returned input unchanged")
		}

		// Result should contain Cyrillic characters
		hasCyrillic := false
		for _, r := range result {
			if r >= 0x0400 && r <= 0x04FF {
				hasCyrillic = true
				break
			}
		}
		if !hasCyrillic {
			t.Logf("Warning: Translation doesn't contain Cyrillic characters: %s", result)
		}
	})

	t.Run("Empty text", func(t *testing.T) {
		ctx := context.Background()
		result, err := client.Translate(ctx, "", "test prompt")

		if err != nil {
			t.Errorf("Translate() failed for empty text: %v", err)
		}

		if result != "" {
			t.Errorf("Expected empty result for empty input, got: %s", result)
		}
	})

	t.Run("Whitespace only", func(t *testing.T) {
		ctx := context.Background()
		result, err := client.Translate(ctx, "   ", "test prompt")

		if err != nil {
			t.Errorf("Translate() failed for whitespace: %v", err)
		}

		if result != "   " {
			t.Logf("Whitespace input returned: %s", result)
		}
	})
}

// INTEGRATION TEST - Test GPU acceleration
func TestGPUAcceleration_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	// Detect hardware
	detector := hardware.NewDetector()
	caps, err := detector.Detect()
	if err != nil {
		t.Fatalf("Hardware detection failed: %v", err)
	}

	if !caps.HasGPU {
		t.Skip("No GPU detected - skipping GPU acceleration test")
	}

	t.Logf("GPU detected: %s", caps.GPUType)

	// Create client
	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client: %v", err)
	}

	// Verify GPU settings are enabled
	if !client.hardwareCaps.HasGPU {
		t.Error("GPU not enabled in client despite detection")
	}

	if client.hardwareCaps.GPUType == "" {
		t.Error("GPU type not set in client")
	}

	t.Logf("GPU acceleration enabled: %s", client.hardwareCaps.GPUType)

	// Test a small translation to verify GPU is working
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	prompt := `Translate: Hello -> Serbian:`
	result, err := client.Translate(ctx, "Hello", prompt)

	if err != nil {
		t.Errorf("Translation with GPU failed: %v", err)
	}

	t.Logf("GPU-accelerated translation result: %s", result)
}

// INTEGRATION TEST - Test performance metrics
func TestPerformanceMetrics_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test with a longer text to measure tokens/second
	testText := `This is a longer test text to measure performance.
It contains multiple sentences and should generate enough tokens
to provide a reasonable performance metric.`

	prompt := `Translate to Serbian: ` + testText

	startTime := time.Now()
	result, err := client.Translate(ctx, testText, prompt)
	duration := time.Since(startTime)

	if err != nil {
		t.Skipf("Translation failed: %v", err)
	}

	// Calculate approximate tokens/second
	// Rough estimate: 1 token ≈ 4 characters
	estimatedTokens := len(result) / 4
	tokensPerSecond := float64(estimatedTokens) / duration.Seconds()

	t.Logf("Performance Metrics:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Output length: %d characters", len(result))
	t.Logf("  Estimated tokens: %d", estimatedTokens)
	t.Logf("  Tokens/second: %.2f", tokensPerSecond)
	t.Logf("  Model: %s", client.modelInfo.Name)
	t.Logf("  Threads: %d", client.threads)
	t.Logf("  GPU: %v (%s)", client.hardwareCaps.HasGPU, client.hardwareCaps.GPUType)

	// Sanity check - should process at least 1 token/second
	if tokensPerSecond < 1.0 {
		t.Errorf("Performance too slow: %.2f tokens/second", tokensPerSecond)
	}
}

// INTEGRATION TEST - Test context length handling
func TestContextLength_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client: %v", err)
	}

	t.Logf("Model context length: %d", client.contextSize)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Test with moderate-length text
	testText := strings.Repeat("This is a test sentence. ", 50) // ~150 words

	prompt := `Translate the following text to Serbian:

` + testText

	result, err := client.Translate(ctx, testText, prompt)

	if err != nil {
		if strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "length") {
			t.Logf("Context length limit reached (expected for very long texts): %v", err)
		} else {
			t.Errorf("Translation failed: %v", err)
		}
		return
	}

	if result == "" {
		t.Error("Translation returned empty result for long text")
	}

	t.Logf("Successfully translated %d characters to %d characters", len(testText), len(result))
}

// INTEGRATION TEST - Test cancellation
func TestTranslateCancellation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client: %v", err)
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try to translate - should be cancelled
	longText := strings.Repeat("Test text for cancellation. ", 100)
	prompt := `Translate: ` + longText

	_, err = client.Translate(ctx, longText, prompt)

	// Should get context cancelled error (or may complete if very fast)
	if err != nil {
		if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "killed") {
			t.Logf("Got error (may be cancellation): %v", err)
		}
	} else {
		t.Log("Translation completed before timeout (system is very fast)")
	}
}

// Test GetModelInfo and GetHardwareInfo
func TestGetters(t *testing.T) {
	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not create client: %v", err)
	}

	t.Run("GetModelInfo", func(t *testing.T) {
		info := client.GetModelInfo()
		if info == nil {
			t.Error("GetModelInfo() returned nil")
		}

		if info.ID == "" {
			t.Error("Model ID is empty")
		}

		t.Logf("Model Info: %s (%s)", info.Name, info.ID)
	})

	t.Run("GetHardwareInfo", func(t *testing.T) {
		info := client.GetHardwareInfo()
		if info == nil {
			t.Error("GetHardwareInfo() returned nil")
		}

		if info.TotalRAM == 0 {
			t.Error("Hardware info has zero RAM")
		}

		t.Logf("Hardware Info: %s, %d cores, %.1f GB RAM, GPU: %v",
			info.Architecture, info.CPUCores,
			float64(info.TotalRAM)/(1024*1024*1024), info.HasGPU)
	})
}

// TestModelDownloadAndUse - End-to-end test
func TestModelDownloadAndUse_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	if _, err := findLlamaCppExecutable(); err != nil {
		t.Skip("llama.cpp not installed")
	}

	// This test verifies the entire workflow:
	// 1. Hardware detection
	// 2. Model selection
	// 3. Model download (if needed)
	// 4. Translation execution

	config := TranslationConfig{
		Provider: "llamacpp",
		// Let system auto-select best model
	}

	t.Log("Step 1: Initializing client (may download model if not cached)")
	client, err := NewLlamaCppClient(config)
	if err != nil {
		t.Skipf("Could not initialize client: %v", err)
	}

	t.Logf("Step 2: Using model: %s", client.modelInfo.Name)
	t.Logf("  Location: %s", client.modelPath)
	t.Logf("  Parameters: %dB", client.modelInfo.Parameters/1_000_000_000)
	t.Logf("  Quantization: %s", client.modelInfo.QuantType)

	t.Log("Step 3: Validating setup")
	if err := client.Validate(); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	t.Log("Step 4: Performing translation")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	testText := "Hello, world!"
	prompt := `Translate the following English text to Serbian (Cyrillic script):

English: Hello, world!
Serbian:`

	result, err := client.Translate(ctx, testText, prompt)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	t.Logf("Step 5: Translation successful")
	t.Logf("  Input: %s", testText)
	t.Logf("  Output: %s", result)

	if result == "" {
		t.Error("Translation returned empty result")
	}

	t.Log("✓ End-to-end test passed")
}

// Benchmark translation performance
func BenchmarkTranslate(b *testing.B) {
	if _, err := findLlamaCppExecutable(); err != nil {
		b.Skip("llama.cpp not installed")
	}

	config := TranslationConfig{
		Provider: "llamacpp",
	}

	client, err := NewLlamaCppClient(config)
	if err != nil {
		b.Skipf("Could not create client: %v", err)
	}

	ctx := context.Background()
	testText := "Hello, this is a test."
	prompt := `Translate: ` + testText

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Translate(ctx, testText, prompt)
		if err != nil {
			b.Fatalf("Translation failed: %v", err)
		}
	}
}

// TestExecutableSearch tests finding llama-cli in different locations
func TestExecutableSearch(t *testing.T) {
	path, err := findLlamaCppExecutable()

	if err != nil {
		t.Skipf("llama.cpp not found: %v", err)
		return
	}

	t.Logf("Found llama-cli at: %s", path)

	// Verify it's actually llama-cli by running --version
	cmd := exec.Command(path, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Warning: Could not run --version: %v", err)
		t.Logf("Output: %s", string(output))
	} else {
		t.Logf("llama-cli version info: %s", string(output))
	}
}
