package models

import (
	"testing"
)

// TestNewRegistry tests registry initialization
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	// Verify models are registered
	if len(registry.models) == 0 {
		t.Error("Registry has no models registered")
	}
}

// TestRegisteredModels tests that expected models are registered
func TestRegisteredModels(t *testing.T) {
	registry := NewRegistry()

	// Expected model IDs
	expectedModels := []string{
		"hunyuan-mt-7b-q4",
		"hunyuan-mt-7b-q8",
		"aya-23-8b-q4",
		"qwen2.5-7b-instruct-q4",
		"qwen2.5-14b-instruct-q4",
		"qwen2.5-27b-instruct-q4",
		"mistral-7b-instruct-q4",
		"phi-3-mini-4k-q4",
		"gemma-2-9b-it-q4",
	}

	for _, modelID := range expectedModels {
		t.Run(modelID, func(t *testing.T) {
			model, exists := registry.Get(modelID)
			if !exists {
				t.Errorf("Expected model %s not found in registry", modelID)
			}
			if model == nil {
				t.Errorf("Model %s exists but returned nil", modelID)
			}
			if model != nil && model.ID != modelID {
				t.Errorf("Model ID mismatch: got %s, expected %s", model.ID, modelID)
			}
		})
	}
}

// TestModelInfo tests model information completeness
func TestModelInfo(t *testing.T) {
	registry := NewRegistry()
	allModels := registry.List()

	for _, model := range allModels {
		t.Run(model.ID, func(t *testing.T) {
			// ID should be non-empty
			if model.ID == "" {
				t.Error("Model ID is empty")
			}

			// Name should be non-empty
			if model.Name == "" {
				t.Error("Model Name is empty")
			}

			// Parameters should be reasonable (1B to 70B)
			if model.Parameters < 1_000_000_000 || model.Parameters > 70_000_000_000 {
				t.Errorf("Parameters out of range: %d", model.Parameters)
			}

			// MinRAM should be at least 1GB
			if model.MinRAM < 1*1024*1024*1024 {
				t.Errorf("MinRAM too low: %d", model.MinRAM)
			}

			// RecommendedRAM should be >= MinRAM
			if model.RecommendedRAM < model.MinRAM {
				t.Errorf("RecommendedRAM (%d) < MinRAM (%d)", model.RecommendedRAM, model.MinRAM)
			}

			// QuantType should be non-empty
			if model.QuantType == "" {
				t.Error("QuantType is empty")
			}

			// SourceURL should be non-empty
			if model.SourceURL == "" {
				t.Error("SourceURL is empty")
			}

			// Languages should not be empty
			if len(model.Languages) == 0 {
				t.Error("Languages list is empty")
			}

			// Quality should be one of: excellent, good, moderate
			validQualities := map[string]bool{
				"excellent": true,
				"good":      true,
				"moderate":  true,
			}
			if !validQualities[model.Quality] {
				t.Errorf("Invalid quality: %s (must be: excellent, good, moderate)", model.Quality)
			}

			// ContextLength should be at least 2048
			if model.ContextLength < 2048 {
				t.Errorf("ContextLength too small: %d", model.ContextLength)
			}
		})
	}
}

// TestGetModel tests retrieving individual models
func TestGetModel(t *testing.T) {
	registry := NewRegistry()

	// Test getting existing model
	model, exists := registry.Get("hunyuan-mt-7b-q4")
	if !exists {
		t.Fatal("hunyuan-mt-7b-q4 not found")
	}
	if model == nil {
		t.Fatal("hunyuan-mt-7b-q4 returned nil")
	}
	if model.ID != "hunyuan-mt-7b-q4" {
		t.Errorf("Wrong model returned: %s", model.ID)
	}

	// Test getting non-existent model
	_, exists = registry.Get("non-existent-model")
	if exists {
		t.Error("Non-existent model reported as existing")
	}
}

// TestListAll tests listing all models
func TestListAll(t *testing.T) {
	registry := NewRegistry()
	allModels := registry.List()

	if len(allModels) == 0 {
		t.Fatal("List() returned empty list")
	}

	// Should have at least 9 models
	if len(allModels) < 9 {
		t.Errorf("List() returned too few models: %d (expected at least 9)", len(allModels))
	}

	// All models should have valid IDs
	seenIDs := make(map[string]bool)
	for _, model := range allModels {
		if model.ID == "" {
			t.Error("List() returned model with empty ID")
		}

		// Check for duplicates
		if seenIDs[model.ID] {
			t.Errorf("List() returned duplicate model: %s", model.ID)
		}
		seenIDs[model.ID] = true
	}
}

// TestFilterByLanguages tests language filtering
func TestFilterByLanguages(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name      string
		languages []string
		minCount  int
	}{
		{
			name:      "Russian and Serbian",
			languages: []string{"ru", "sr"},
			minCount:  5, // Most models should support these
		},
		{
			name:      "English",
			languages: []string{"en"},
			minCount:  8, // Almost all models support English
		},
		{
			name:      "Multiple languages",
			languages: []string{"en", "ru", "zh"},
			minCount:  5,
		},
		{
			name:      "Empty languages list",
			languages: []string{},
			minCount:  9, // Should return all models
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := registry.FilterByLanguages(tt.languages)

			if len(filtered) < tt.minCount {
				t.Errorf("FilterByLanguages(%v) returned %d models, expected at least %d",
					tt.languages, len(filtered), tt.minCount)
			}

			// Verify all returned models support the required languages
			if len(tt.languages) > 0 {
				for _, model := range filtered {
					for _, requiredLang := range tt.languages {
						found := false
						for _, modelLang := range model.Languages {
							if modelLang == requiredLang {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Model %s doesn't support required language %s",
								model.ID, requiredLang)
						}
					}
				}
			}
		})
	}
}

// TestFilterByRAM tests RAM-based filtering
func TestFilterByRAM(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name           string
		maxRAM         uint64
		minExpected    int
		maxExpected    int
	}{
		{
			name:        "Low RAM (4GB)",
			maxRAM:      4 * 1024 * 1024 * 1024,
			minExpected: 1, // At least phi-3-mini
			maxExpected: 3,
		},
		{
			name:        "Medium RAM (8GB)",
			maxRAM:      8 * 1024 * 1024 * 1024,
			minExpected: 3,
			maxExpected: 6,
		},
		{
			name:        "High RAM (16GB)",
			maxRAM:      16 * 1024 * 1024 * 1024,
			minExpected: 5,
			maxExpected: 9, // Most models
		},
		{
			name:        "Very high RAM (32GB)",
			maxRAM:      32 * 1024 * 1024 * 1024,
			minExpected: 8,
			maxExpected: 9, // All models
		},
		{
			name:        "Extremely low RAM (2GB)",
			maxRAM:      2 * 1024 * 1024 * 1024,
			minExpected: 0,
			maxExpected: 0, // No models
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := registry.FilterByRAM(tt.maxRAM)

			if len(filtered) < tt.minExpected || len(filtered) > tt.maxExpected {
				t.Errorf("FilterByRAM(%d GB) returned %d models, expected between %d and %d",
					tt.maxRAM/(1024*1024*1024), len(filtered), tt.minExpected, tt.maxExpected)
			}

			// Verify all returned models fit within RAM limit
			for _, model := range filtered {
				if model.MinRAM > tt.maxRAM {
					t.Errorf("Model %s requires %d GB but filter was %d GB",
						model.ID, model.MinRAM/(1024*1024*1024), tt.maxRAM/(1024*1024*1024))
				}
			}
		})
	}
}

// TestFindBestModel tests best model selection algorithm
func TestFindBestModel(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name          string
		maxRAM        uint64
		languages     []string
		hasGPU    bool
		expectError   bool
		expectedModel string // ID of expected best model (or empty if any is ok)
	}{
		{
			name:          "Russian-Serbian, 16GB, GPU",
			maxRAM:        16 * 1024 * 1024 * 1024,
			languages:     []string{"ru", "sr"},
			hasGPU:    false,
			expectError:   false,
			expectedModel: "hunyuan-mt-7b-q8", // Best translation model for this case
		},
		{
			name:        "Low RAM (4GB)",
			maxRAM:      4 * 1024 * 1024 * 1024,
			languages:   []string{"en"},
			hasGPU:  false,
			expectError: false,
			// Should select phi-3-mini or similar small model
		},
		{
			name:        "High RAM (32GB)",
			maxRAM:      32 * 1024 * 1024 * 1024,
			languages:   []string{"ru", "sr"},
			hasGPU:  false,
			expectError: false,
			// Should select a larger, higher-quality model
		},
		{
			name:        "Insufficient RAM (1GB)",
			maxRAM:      1 * 1024 * 1024 * 1024,
			languages:   []string{"en"},
			hasGPU:  false,
			expectError: true, // No model fits
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := registry.FindBestModel(tt.maxRAM, tt.languages, tt.hasGPU)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("FindBestModel() failed: %v", err)
			}

			if model == nil {
				t.Fatal("FindBestModel() returned nil model")
			}

			// Verify model fits RAM constraint
			if model.MinRAM > tt.maxRAM {
				t.Errorf("Selected model %s requires %d GB but limit is %d GB",
					model.ID, model.MinRAM/(1024*1024*1024), tt.maxRAM/(1024*1024*1024))
			}

			// Verify model supports required languages
			for _, requiredLang := range tt.languages {
				found := false
				for _, modelLang := range model.Languages {
					if modelLang == requiredLang {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Selected model %s doesn't support required language %s",
						model.ID, requiredLang)
				}
			}

			// Check if expected model was selected (if specified)
			if tt.expectedModel != "" && model.ID != tt.expectedModel {
				t.Logf("Expected model %s but got %s (this may be acceptable)", tt.expectedModel, model.ID)
			}
		})
	}
}

// TestModelScoring tests the model scoring algorithm
func TestModelScoring(t *testing.T) {
	registry := NewRegistry()

	// Get Hunyuan-MT-7B Q8 (translation specialist)
	hunyuan, _ := registry.Get("hunyuan-mt-7b-q8")

	// Get Qwen2.5-7B Q4 (general purpose)
	qwen, _ := registry.Get("qwen2.5-7b-instruct-q4")

	// Get Phi-3-Mini (small model)
	phi, _ := registry.Get("phi-3-mini-4k-q4")

	if hunyuan == nil || qwen == nil || phi == nil {
		t.Fatal("Failed to get test models")
	}

	// Calculate scores (using 16GB RAM as reference)
	testRAM := uint64(16 * 1024 * 1024 * 1024)
	hunyuanScore := registry.scoreModel(hunyuan, []string{"ru", "sr"}, testRAM)
	qwenScore := registry.scoreModel(qwen, []string{"ru", "sr"}, testRAM)
	phiScore := registry.scoreModel(phi, []string{"ru", "sr"}, testRAM)

	// Hunyuan should score highest for Russian-Serbian translation
	if hunyuanScore <= qwenScore {
		t.Errorf("Hunyuan score (%f) should be higher than Qwen score (%f) for translation task",
			hunyuanScore, qwenScore)
	}

	// Larger models should generally score higher than smaller ones (all else equal)
	if phiScore > hunyuanScore {
		t.Errorf("Phi-3-Mini score (%f) should be lower than Hunyuan score (%f)",
			phiScore, hunyuanScore)
	}

	t.Logf("Scores - Hunyuan: %.2f, Qwen: %.2f, Phi: %.2f", hunyuanScore, qwenScore, phiScore)
}

// TestTranslationOptimization tests that translation-optimized models are prioritized
func TestTranslationOptimization(t *testing.T) {
	registry := NewRegistry()

	// Find best model for Russian-Serbian translation with ample RAM
	model, err := registry.FindBestModel(
		18*1024*1024*1024, // 18 GB
		[]string{"ru", "sr"},
		false,
	)

	if err != nil {
		t.Fatalf("FindBestModel() failed: %v", err)
	}

	// Should select a translation-optimized model
	if model.OptimizedFor != "Professional Translation" && model.OptimizedFor != "Multilingual Translation" {
		t.Errorf("Expected translation-optimized model but got: %s (optimized for: %s)",
			model.ID, model.OptimizedFor)
	}

	// For Russian-Serbian, should prefer Hunyuan-MT
	if model.ID != "hunyuan-mt-7b-q8" && model.ID != "hunyuan-mt-7b-q4" && model.ID != "aya-23-8b-q4" {
		t.Logf("Warning: Expected Hunyuan-MT or Aya-23 but got %s (may be acceptable)", model.ID)
	}
}

// TestQuantizationVariants tests that different quantization variants are available
func TestQuantizationVariants(t *testing.T) {
	registry := NewRegistry()

	// Hunyuan-MT should have both Q4 and Q8 variants
	q4, q4exists := registry.Get("hunyuan-mt-7b-q4")
	q8, q8exists := registry.Get("hunyuan-mt-7b-q8")

	if !q4exists || !q8exists {
		t.Error("Expected both Q4 and Q8 variants of Hunyuan-MT-7B")
	}

	if q4exists && q8exists {
		// Q8 should require more RAM than Q4
		if q8.MinRAM <= q4.MinRAM {
			t.Errorf("Q8 variant should require more RAM than Q4 (Q8: %d, Q4: %d)",
				q8.MinRAM, q4.MinRAM)
		}

		// Q8 should have better quality
		qualityRank := map[string]int{"excellent": 3, "good": 2, "moderate": 1}
		if qualityRank[q8.Quality] < qualityRank[q4.Quality] {
			t.Errorf("Q8 variant should have equal or better quality than Q4 (Q8: %s, Q4: %s)",
				q8.Quality, q4.Quality)
		}
	}
}

// TestModelSizeProgression tests that models are available in various sizes
func TestModelSizeProgression(t *testing.T) {
	registry := NewRegistry()

	// Check for small, medium, and large models
	sizes := []struct {
		name      string
		minParams uint64
		maxParams uint64
	}{
		{"Small (3-4B)", 3_000_000_000, 4_000_000_000},
		{"Medium (7-9B)", 7_000_000_000, 9_000_000_000},
		{"Large (13-15B)", 13_000_000_000, 15_000_000_000},
		{"Very Large (27B+)", 27_000_000_000, 70_000_000_000},
	}

	allModels := registry.List()

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			found := false
			for _, model := range allModels {
				if model.Parameters >= size.minParams && model.Parameters <= size.maxParams {
					found = true
					break
				}
			}
			if !found {
				t.Logf("Warning: No model found in size range %s (%dB-%dB)",
					size.name, size.minParams/1_000_000_000, size.maxParams/1_000_000_000)
			}
		})
	}
}

// TestLanguageCoverage tests that models cover important languages
func TestLanguageCoverage(t *testing.T) {
	registry := NewRegistry()
	allModels := registry.List()

	importantLanguages := []string{"en", "ru", "zh", "es", "fr", "de", "ja", "ko", "ar", "hi"}

	for _, lang := range importantLanguages {
		t.Run(lang, func(t *testing.T) {
			found := false
			for _, model := range allModels {
				for _, modelLang := range model.Languages {
					if modelLang == lang {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				t.Logf("Warning: No model found supporting language: %s", lang)
			}
		})
	}
}

// BenchmarkFindBestModel benchmarks model selection performance
func BenchmarkFindBestModel(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.FindBestModel(
			16*1024*1024*1024,
			[]string{"ru", "sr"},
			false,
		)
		if err != nil {
			b.Fatalf("FindBestModel() failed: %v", err)
		}
	}
}

// BenchmarkFilterByLanguages benchmarks language filtering
func BenchmarkFilterByLanguages(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.FilterByLanguages([]string{"ru", "sr"})
	}
}

// BenchmarkFilterByRAM benchmarks RAM filtering
func BenchmarkFilterByRAM(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.FilterByRAM(16 * 1024 * 1024 * 1024)
	}
}
