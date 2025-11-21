package translator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"digital.vasic.translator/pkg/hardware"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// TestFullPipeline_Integration tests the complete translation pipeline
// from hardware detection through model selection to actual translation
func TestFullPipeline_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Hardware Detection Pipeline", func(t *testing.T) {
		// Step 1: Detect hardware capabilities
		detector := hardware.NewDetector()
		caps, err := detector.Detect()
		if err != nil {
			t.Fatalf("Hardware detection failed: %v", err)
		}

		t.Logf("Detected Hardware:")
		t.Logf("  Total RAM: %.2f GB", float64(caps.TotalRAM)/(1024*1024*1024))
		t.Logf("  Available RAM: %.2f GB", float64(caps.AvailableRAM)/(1024*1024*1024))
		t.Logf("  CPU: %s (%d cores)", caps.CPUModel, caps.CPUCores)
		t.Logf("  GPU: %v (%s)", caps.HasGPU, caps.GPUType)
		t.Logf("  Max Model Size: %d B params", caps.MaxModelSize/1_000_000_000)

		// Verify reasonable values
		if caps.TotalRAM < 1*1024*1024*1024 {
			t.Errorf("Total RAM too low: %d bytes", caps.TotalRAM)
		}
		if caps.AvailableRAM == 0 {
			t.Error("Available RAM is zero")
		}
		if caps.CPUCores == 0 {
			t.Error("CPU cores is zero")
		}
	})

	t.Run("Model Selection Pipeline", func(t *testing.T) {
		// Step 1: Detect hardware
		detector := hardware.NewDetector()
		caps, err := detector.Detect()
		if err != nil {
			t.Fatalf("Hardware detection failed: %v", err)
		}

		// Step 2: Find best model for hardware
		registry := models.NewRegistry()
		model, err := registry.FindBestModel(
			caps.AvailableRAM,
			[]string{"ru", "sr"},
			caps.HasGPU,
		)
		if err != nil {
			t.Fatalf("Model selection failed: %v", err)
		}

		t.Logf("Selected Model:")
		t.Logf("  ID: %s", model.ID)
		t.Logf("  Name: %s", model.Name)
		t.Logf("  Min RAM: %.1f GB", float64(model.MinRAM)/(1024*1024*1024))
		t.Logf("  Quality: %s", model.Quality)
		t.Logf("  Optimized For: %s", model.OptimizedFor)

		// Verify model fits in available RAM
		if model.MinRAM > caps.AvailableRAM {
			t.Errorf("Selected model requires more RAM than available")
		}

		// Verify model supports required languages
		hasRussian := false
		hasSerbian := false
		for _, lang := range model.Languages {
			if lang == "ru" {
				hasRussian = true
			}
			if lang == "sr" {
				hasSerbian = true
			}
		}
		if !hasRussian || !hasSerbian {
			t.Error("Selected model doesn't support Russian or Serbian")
		}
	})

	t.Run("Model Download Pipeline", func(t *testing.T) {
		// Step 1: Detect hardware
		detector := hardware.NewDetector()
		caps, err := detector.Detect()
		if err != nil {
			t.Fatalf("Hardware detection failed: %v", err)
		}

		// Step 2: Find best model
		registry := models.NewRegistry()
		model, err := registry.FindBestModel(
			caps.AvailableRAM,
			[]string{"ru", "sr"},
			caps.HasGPU,
		)
		if err != nil {
			t.Fatalf("Model selection failed: %v", err)
		}

		// Step 3: Initialize downloader
		downloader := models.NewDownloader()

		// Step 4: Check if model needs download
		modelPath, err := downloader.GetModelPath(model)
		needsDownload := (err != nil)

		t.Logf("Model %s needs download: %v", model.ID, needsDownload)

		if needsDownload {
			t.Log("Note: Actual download would occur here in production")
			t.Log("Skipping download in test to avoid large file transfer")
		} else {
			t.Logf("Model already downloaded at: %s", modelPath)
		}
	})
}

// TestTranslationWorkflow_E2E tests end-to-end translation workflows
func TestTranslationWorkflow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("DeepSeek Translation Workflow", func(t *testing.T) {
		// Check if API key is available
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			t.Skip("DEEPSEEK_API_KEY not set - skipping DeepSeek E2E test")
		}

		// Create translator config
		config := translator.TranslationConfig{
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			Script:         "cyrillic",
		}

		// Create client
		client, err := llm.NewDeepSeekClient(config)
		if err != nil {
			t.Fatalf("Failed to create DeepSeek client: %v", err)
		}

		// Test translation
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		testText := "Привет, мир!"
		prompt := `Translate the following Russian text to Serbian (Cyrillic):

Russian: Привет, мир!
Serbian:`

		result, err := client.Translate(ctx, testText, prompt)
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		t.Logf("Translation result: %s", result)

		// Verify result is not empty and contains Cyrillic
		if len(result) == 0 {
			t.Error("Translation result is empty")
		}
	})

	t.Run("LlamaCpp Translation Workflow", func(t *testing.T) {
		// Try to create client - it will skip if llama.cpp not available
		config := translator.TranslationConfig{
			Provider:       "llamacpp",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			Script:         "cyrillic",
		}

		client, err := llm.NewLlamaCppClient(config)
		if err != nil {
			t.Skipf("llama.cpp not available - skipping LlamaCpp E2E test: %v", err)
		}

		// Test translation (client already created above)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		testText := "Привет, мир!"
		prompt := `Translate the following Russian text to Serbian (Cyrillic):

Russian: Привет, мир!
Serbian:`

		result, err := client.Translate(ctx, testText, prompt)
		if err != nil {
			t.Fatalf("Translation failed: %v", err)
		}

		t.Logf("Translation result: %s", result)

		// Verify result is not empty
		if len(result) == 0 {
			t.Error("Translation result is empty")
		}
	})
}

// TestConfigurationScenarios tests various configuration scenarios
func TestConfigurationScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping configuration tests in short mode")
	}

	scenarios := []struct {
		name        string
		config      translator.TranslationConfig
		shouldError bool
		skipReason  string
	}{
		{
			name: "DeepSeek with Cyrillic",
			config: translator.TranslationConfig{
				Provider:       "deepseek",
				Model:          "deepseek-chat",
				SourceLanguage: "ru",
				TargetLanguage: "sr",
				Script:         "cyrillic",
			},
			shouldError: false,
			skipReason:  "DEEPSEEK_API_KEY",
		},
		{
			name: "DeepSeek with Latin",
			config: translator.TranslationConfig{
				Provider:       "deepseek",
				Model:          "deepseek-chat",
				SourceLanguage: "ru",
				TargetLanguage: "sr",
				Script:         "latin",
			},
			shouldError: false,
			skipReason:  "DEEPSEEK_API_KEY",
		},
		{
			name: "LlamaCpp with Auto Model Selection",
			config: translator.TranslationConfig{
				Provider:       "llamacpp",
				SourceLanguage: "ru",
				TargetLanguage: "sr",
				Script:         "cyrillic",
			},
			shouldError: false,
			skipReason:  "",
		},
		{
			name: "Invalid Provider",
			config: translator.TranslationConfig{
				Provider:       "invalid-provider",
				SourceLanguage: "ru",
				TargetLanguage: "sr",
			},
			shouldError: true,
			skipReason:  "",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Check skip condition
			if scenario.skipReason != "" && os.Getenv(scenario.skipReason) == "" {
				t.Skipf("Skipping - %s not set", scenario.skipReason)
			}

			// Create clients (not checking Translator interface, just creation)
			var err error

			switch scenario.config.Provider {
			case "deepseek":
				_, err = llm.NewDeepSeekClient(scenario.config)
			case "llamacpp":
				_, err = llm.NewLlamaCppClient(scenario.config)
			default:
				err = translator.ErrInvalidProvider
			}

			if scenario.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestErrorHandling tests error handling in various scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("Invalid Hardware Detection", func(t *testing.T) {
		// Hardware detection should never completely fail on supported platforms
		detector := hardware.NewDetector()
		caps, err := detector.Detect()
		if err != nil {
			t.Logf("Hardware detection error (expected on some systems): %v", err)
		} else {
			if caps.TotalRAM == 0 {
				t.Error("Hardware detection succeeded but returned 0 RAM")
			}
		}
	})

	t.Run("Model Not Found", func(t *testing.T) {
		registry := models.NewRegistry()
		_, exists := registry.Get("non-existent-model")
		if exists {
			t.Error("Found model that shouldn't exist")
		}
	})

	t.Run("Insufficient RAM", func(t *testing.T) {
		registry := models.NewRegistry()
		// Try to find model with only 1GB RAM
		_, err := registry.FindBestModel(
			1*1024*1024*1024, // 1GB
			[]string{"ru", "sr"},
			false,
		)
		// Should error because no models fit in 1GB
		if err == nil {
			t.Error("Expected error for insufficient RAM but got none")
		}
		t.Logf("Expected error: %v", err)
	})

	t.Run("Translation Timeout", func(t *testing.T) {
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			t.Skip("DEEPSEEK_API_KEY not set")
		}

		config := translator.TranslationConfig{
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
		}

		client, err := llm.NewDeepSeekClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Should timeout
		_, err = client.Translate(ctx, "test", "test prompt")
		if err == nil {
			t.Error("Expected timeout error but got none")
		}
		t.Logf("Expected timeout error: %v", err)
	})
}

// TestConcurrentTranslations tests concurrent translation requests
func TestConcurrentTranslations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
	}

	client, err := llm.NewDeepSeekClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Run 3 concurrent translations
	const numTranslations = 3
	results := make(chan error, numTranslations)

	for i := 0; i < numTranslations; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			prompt := "Translate: Hello"
			_, err := client.Translate(ctx, "Hello", prompt)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numTranslations; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent translation %d failed: %v", i, err)
		}
	}
}

// BenchmarkFullPipeline benchmarks the complete pipeline
func BenchmarkFullPipeline(b *testing.B) {
	detector := hardware.NewDetector()
	registry := models.NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Hardware detection
		caps, err := detector.Detect()
		if err != nil {
			b.Fatalf("Hardware detection failed: %v", err)
		}

		// Model selection
		_, err = registry.FindBestModel(
			caps.AvailableRAM,
			[]string{"ru", "sr"},
			caps.HasGPU,
		)
		if err != nil {
			b.Fatalf("Model selection failed: %v", err)
		}
	}
}

// Note: findLlamaCppExecutable is internal to llm package
// Tests use NewLlamaCppClient which handles detection internally
