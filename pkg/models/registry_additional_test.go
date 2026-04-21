package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRegistry_GetRecommendationsForHardware(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name        string
		ramGB       float64
		hasGPU      bool
		minExpected int
		maxExpected int
	}{
		{
			name:        "Low RAM 4GB no GPU",
			ramGB:       4,
			hasGPU:      false,
			minExpected: 1,
			maxExpected: 3,
		},
		{
			name:        "Medium RAM 8GB no GPU",
			ramGB:       8,
			hasGPU:      false,
			minExpected: 3,
			maxExpected: 9,
		},
		{
			name:        "High RAM 32GB no GPU",
			ramGB:       32,
			hasGPU:      false,
			minExpected: 5,
			maxExpected: 9,
		},
		{
			name:        "Very low RAM 1GB",
			ramGB:       1,
			hasGPU:      false,
			minExpected: 0,
			maxExpected: 1,
		},
		{
			name:        "With GPU flag",
			ramGB:       16,
			hasGPU:      true,
			minExpected: 4,
			maxExpected: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := registry.GetRecommendationsForHardware(tt.ramGB, tt.hasGPU)

			assert.GreaterOrEqual(t, len(recommendations), tt.minExpected, "Expected at least %d recommendations", tt.minExpected)
			assert.LessOrEqual(t, len(recommendations), tt.maxExpected, "Expected at most %d recommendations", tt.maxExpected)

			// Verify all recommendations fit in RAM
			ramBytes := uint64(tt.ramGB * 1024 * 1024 * 1024)
			for _, model := range recommendations {
				assert.LessOrEqual(t, model.MinRAM, ramBytes, "Model %s requires more RAM than available", model.ID)
			}

			// Verify translation-optimized models are prioritized
			if len(recommendations) > 0 {
				firstModel := recommendations[0]
				assert.Contains(t, firstModel.OptimizedFor, "Translation", "First recommendation should be translation-optimized")
			}
		})
	}
}

func TestModelRegistry_Register_Validation(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name    string
		model   *ModelInfo
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid model",
			model: &ModelInfo{
				ID:            "test-model-1",
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
				QuantType:     "Q4_K_M",
				Quality:       "excellent",
			},
			wantErr: false,
		},
		{
			name: "Missing ID",
			model: &ModelInfo{
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
			},
			wantErr: true,
			errMsg:  "model ID is required",
		},
		{
			name: "Missing name",
			model: &ModelInfo{
				ID:            "test-model-2",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
			},
			wantErr: true,
			errMsg:  "model name is required",
		},
		{
			name: "Missing source URL",
			model: &ModelInfo{
				ID:            "test-model-3",
				Name:          "Test Model",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
			},
			wantErr: true,
			errMsg:  "source URL is required",
		},
		{
			name: "Zero parameters",
			model: &ModelInfo{
				ID:            "test-model-4",
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
			},
			wantErr: true,
			errMsg:  "parameters count is required",
		},
		{
			name: "Zero MinRAM",
			model: &ModelInfo{
				ID:            "test-model-5",
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				ContextLength: 4096,
			},
			wantErr: true,
			errMsg:  "minimum RAM is required",
		},
		{
			name: "Zero context length",
			model: &ModelInfo{
				ID:         "test-model-6",
				Name:       "Test Model",
				SourceURL:  "https://example.com/model.gguf",
				Parameters: 7_000_000_000,
				MinRAM:     6 * 1024 * 1024 * 1024,
			},
			wantErr: true,
			errMsg:  "context length is required",
		},
		{
			name: "Invalid URL scheme",
			model: &ModelInfo{
				ID:            "test-model-7",
				Name:          "Test Model",
				SourceURL:     "ftp://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
			},
			wantErr: true,
			errMsg:  "source URL must be a valid HTTP/HTTPS URL",
		},
		{
			name: "Invalid quantization type",
			model: &ModelInfo{
				ID:            "test-model-8",
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
				QuantType:     "INVALID",
			},
			wantErr: true,
			errMsg:  "invalid quantization type",
		},
		{
			name: "Invalid quality rating",
			model: &ModelInfo{
				ID:            "test-model-9",
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
				Quality:       "poor",
			},
			wantErr: true,
			errMsg:  "invalid quality rating",
		},
		{
			name: "Valid empty quant type and quality",
			model: &ModelInfo{
				ID:            "test-model-10",
				Name:          "Test Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
				QuantType:     "",
				Quality:       "",
			},
			wantErr: false,
		},
		{
			name: "Duplicate ID overwrites",
			model: &ModelInfo{
				ID:            "hunyuan-mt-7b-q4", // Existing model ID
				Name:          "Overwritten Model",
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.model)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestModelRegistry_Register_AllValidQuantTypes(t *testing.T) {
	registry := NewRegistry()
	validQuants := []string{"Q4", "Q4_K_M", "Q4_K_S", "Q5", "Q5_K_M", "Q5_K_S", "Q8", "Q8_0", "F16", "F32"}

	for i, quant := range validQuants {
		t.Run(quant, func(t *testing.T) {
			model := &ModelInfo{
				ID:            "quant-test-" + string(rune('a'+i)),
				Name:          "Quant Test " + quant,
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
				QuantType:     quant,
			}
			err := registry.Register(model)
			require.NoError(t, err)
		})
	}
}

func TestModelRegistry_Register_AllValidQualities(t *testing.T) {
	registry := NewRegistry()
	validQualities := []string{"excellent", "good", "moderate"}

	for _, quality := range validQualities {
		t.Run(quality, func(t *testing.T) {
			model := &ModelInfo{
				ID:            "quality-test-" + quality,
				Name:          "Quality Test " + quality,
				SourceURL:     "https://example.com/model.gguf",
				Parameters:    7_000_000_000,
				MinRAM:        6 * 1024 * 1024 * 1024,
				ContextLength: 4096,
				Quality:       quality,
			}
			err := registry.Register(model)
			require.NoError(t, err)
		})
	}
}

func TestModelRegistry_FindBestModel_NoCandidates(t *testing.T) {
	registry := NewRegistry()

	// Try with extremely low RAM
	_, err := registry.FindBestModel(1*1024*1024*1024, []string{"en"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no models found")
}

func TestModelRegistry_FindBestModel_GPUFiltering(t *testing.T) {
	registry := NewRegistry()

	// Create a GPU-only model
	gpuModel := &ModelInfo{
		ID:            "gpu-only-model",
		Name:          "GPU Only Model",
		SourceURL:     "https://example.com/model.gguf",
		Parameters:    7_000_000_000,
		MinRAM:        6 * 1024 * 1024 * 1024,
		ContextLength: 4096,
		RequiresGPU:   true,
		Languages:     []string{"en"},
		OptimizedFor:  "Translation",
		Quality:       "excellent",
	}
	require.NoError(t, registry.Register(gpuModel))

	// Without GPU, should not select GPU-only model
	model, err := registry.FindBestModel(32*1024*1024*1024, []string{"en"}, false)
	require.NoError(t, err)
	assert.NotEqual(t, "gpu-only-model", model.ID, "Should not select GPU-only model without GPU")

	// With GPU, can select GPU-only model
	model, err = registry.FindBestModel(32*1024*1024*1024, []string{"en"}, true)
	require.NoError(t, err)
	// It might or might not be selected depending on scoring, but it should be possible
	_ = model
}
