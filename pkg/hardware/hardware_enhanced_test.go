package hardware

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetector_NewDetector tests the constructor for the Detector
func TestDetector_NewDetector(t *testing.T) {
	detector := NewDetector()
	assert.NotNil(t, detector)
}

// TestDetector_Detect_comprehensive tests the Detect method with comprehensive checks
func TestDetector_Detect_comprehensive(t *testing.T) {
	detector := NewDetector()
	capabilities, err := detector.Detect()
	
	require.NoError(t, err)
	require.NotNil(t, capabilities)
	
	// Test that all fields have reasonable values
	assert.Equal(t, runtime.GOARCH, capabilities.Architecture)
	assert.Greater(t, capabilities.TotalRAM, uint64(0))
	assert.Greater(t, capabilities.AvailableRAM, uint64(0))
	assert.GreaterOrEqual(t, capabilities.CPUCores, 1)
	
	// Check that available RAM is not greater than total RAM
	assert.LessOrEqual(t, capabilities.AvailableRAM, capabilities.TotalRAM)
	
	// GPU detection might be false or true depending on system
	// CPU model might be empty on some systems, that's OK
	_ = capabilities.HasGPU
	_ = capabilities.GPUType
	_ = capabilities.CPUModel
	
	// Max model size should be set based on RAM
	assert.Greater(t, capabilities.MaxModelSize, uint64(0))
}

// TestDetector_Detect_platformSpecific tests platform-specific detection logic
func TestDetector_Detect_platformSpecific(t *testing.T) {
	detector := NewDetector()
	
	// Test getTotalRAM
	totalRAM, err := detector.getTotalRAM()
	if err == nil {
		assert.Greater(t, totalRAM, uint64(0))
	} else {
		// Error is acceptable for some systems
		assert.Contains(t, err.Error(), "failed to detect")
	}
	
	// Test getAvailableRAM
	availableRAM, err := detector.getAvailableRAM()
	if err == nil {
		assert.Greater(t, availableRAM, uint64(0))
	} else {
		// Error is acceptable for some systems
		assert.Contains(t, err.Error(), "failed to detect")
	}
	
	// Test getCPUModel and getCPUCores
	cpuModel, err := detector.getCPUModel()
	if err != nil {
		assert.Contains(t, err.Error(), "failed to detect")
	} else {
		assert.NotEmpty(t, cpuModel)
	}
	
	cpuCores, err := detector.getCPUCores()
	if err == nil {
		assert.GreaterOrEqual(t, cpuCores, 1)
	} else {
		// Error is acceptable for some systems
		assert.Contains(t, err.Error(), "failed to detect")
	}
	
	// Test detectGPU
	hasGPU, gpuType := detector.detectGPU()
	_ = hasGPU
	_ = gpuType
	
	// Test calculateMaxModelSize
	maxModelSize := detector.calculateMaxModelSize(8*1024*1024*1024, false) // 8GB RAM, no GPU
	assert.Greater(t, maxModelSize, uint64(0))
	assert.LessOrEqual(t, maxModelSize, uint64(70)) // Should be reasonable for 8GB
}

// TestDetector_calculateMaxModelSize tests max model size calculation with different RAM sizes
func TestDetector_calculateMaxModelSize(t *testing.T) {
	detector := NewDetector()
	
	tests := []struct {
		name           string
		ramBytes       uint64
		expectedMinSize uint64
		expectedMaxSize uint64
	}{
		{
			name:           "Low RAM (4GB)",
			ramBytes:       4 * 1024 * 1024 * 1024,
			expectedMinSize: 1,  // At least 1B model
			expectedMaxSize: 3,  // Should be small for 4GB
		},
		{
			name:           "Medium RAM (8GB)",
			ramBytes:       8 * 1024 * 1024 * 1024,
			expectedMinSize: 1,
			expectedMaxSize: 7,  // Should be around 7B for 8GB
		},
		{
			name:           "High RAM (16GB)",
			ramBytes:       16 * 1024 * 1024 * 1024,
			expectedMinSize: 3,
			expectedMaxSize: 13, // Should be around 13B for 16GB
		},
		{
			name:           "Very High RAM (32GB)",
			ramBytes:       32 * 1024 * 1024 * 1024,
			expectedMinSize: 7,
			expectedMaxSize: 30, // Should support larger models
		},
		{
			name:           "Extremely High RAM (64GB)",
			ramBytes:       64 * 1024 * 1024 * 1024,
			expectedMinSize: 13,
			expectedMaxSize: 70, // Maximum supported size
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelSize := detector.calculateMaxModelSize(tt.ramBytes, false) // Assume no GPU
			assert.GreaterOrEqual(t, modelSize, tt.expectedMinSize)
			assert.LessOrEqual(t, modelSize, tt.expectedMaxSize)
		})
	}
}

// TestDetector_edgeCases tests edge cases and error conditions
func TestDetector_edgeCases(t *testing.T) {
	detector := NewDetector()
	
	t.Run("Zero RAM calculation", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(0, false)
		assert.Equal(t, uint64(1), modelSize) // Should default to 1B
	})
	
	t.Run("Very small RAM", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(1024*1024, false) // 1MB
		assert.Equal(t, uint64(1), modelSize) // Should default to 1B
	})
	
	t.Run("Very large RAM", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(1024*1024*1024*1024, false) // 1TB
		assert.Equal(t, uint64(70), modelSize) // Should cap at 70B
	})
}

// TestCapabilities_toString tests string representation of capabilities
func TestCapabilities_toString(t *testing.T) {
	caps := &Capabilities{
		Architecture: "arm64",
		TotalRAM:     8 * 1024 * 1024 * 1024, // 8GB
		AvailableRAM: 6 * 1024 * 1024 * 1024, // 6GB
		CPUModel:     "Apple M2",
		CPUCores:     8,
		HasGPU:       true,
		GPUType:      "metal",
		MaxModelSize: 7,
	}
	
	str := caps.String()
	assert.Contains(t, str, "arm64")
	assert.Contains(t, str, "Apple M2")
	assert.Contains(t, str, "7B")
	assert.Contains(t, str, "metal")
}

// TestGPUType_detection tests specific GPU type detection
func TestGPUType_detection(t *testing.T) {
	detector := NewDetector()
	
	// This test will pass on systems with GPU and fail on systems without
	// The important thing is to test detection logic
	hasGPU, gpuType := detector.detectGPU()
	
	if hasGPU {
		validTypes := []string{"metal", "cuda", "rocm", "vulkan", "opencl"}
		valid := false
		for _, vt := range validTypes {
			if gpuType == vt {
				valid = true
				break
			}
		}
		assert.True(t, valid, "GPU type should be one of the valid types")
	} else {
		assert.Empty(t, gpuType, "GPU type should be empty when no GPU is detected")
	}
}

// BenchmarkDetector_Detect benchmarks the hardware detection
func BenchmarkDetector_Detect(b *testing.B) {
	detector := NewDetector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect()
	}
}

// BenchmarkDetector_getTotalRAM benchmarks RAM detection
func BenchmarkDetector_getTotalRAM(b *testing.B) {
	detector := NewDetector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.getTotalRAM()
	}
}

// BenchmarkDetector_calculateMaxModelSize benchmarks max model size calculation
func BenchmarkDetector_calculateMaxModelSize(b *testing.B) {
	detector := NewDetector()
	ramSize := uint64(16 * 1024 * 1024 * 1024) // 16GB
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.calculateMaxModelSize(ramSize, false)
	}
}