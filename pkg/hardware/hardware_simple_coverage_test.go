package hardware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHardwareDetector_CalculateMaxModelSizeSimple tests calculateMaxModelSize with simple inputs
func TestHardwareDetector_CalculateMaxModelSizeSimple(t *testing.T) {
	detector := NewDetector()

	// Test various RAM and GPU configurations
	testCases := []struct {
		name        string
		totalRAM    uint64
		hasGPU      bool
		expectZero   bool
	}{
		{"Zero RAM", 0, false, true},
		{"Small RAM", 1024 * 1024 * 1024, false, true}, // 1GB
		{"Medium RAM", 8 * 1024 * 1024 * 1024, false, false}, // 8GB
		{"Large RAM with GPU", 16 * 1024 * 1024 * 1024, true, false}, // 16GB with GPU
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modelSize := detector.calculateMaxModelSize(tc.totalRAM, tc.hasGPU)
			if tc.expectZero {
				assert.Equal(t, uint64(0), modelSize)
			} else {
				assert.Greater(t, modelSize, uint64(0))
			}
		})
	}
}

// TestHardwareDetector_CanRunModelSimple tests CanRunModel with simple inputs
func TestHardwareDetector_CanRunModelSimple(t *testing.T) {
	capabilities := &Capabilities{
		TotalRAM:     8 * 1024 * 1024 * 1024, // 8GB
		AvailableRAM:  4 * 1024 * 1024 * 1024, // 4GB
		MaxModelSize: 100 * 1024 * 1024,      // 100MB
		CPUModel:      "Test CPU",
		CPUCores:      4,
		HasGPU:        false,
	}

	testCases := []struct {
		name        string
		modelSize   uint64
		expectCan   bool
	}{
		{"Zero size model", 0, true},
		{"Small model", 1024, true}, // 1KB
		{"Exact fit model", 100 * 1024 * 1024, true}, // 100MB
		{"Too large model", 200 * 1024 * 1024, false}, // 200MB
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canRun := capabilities.CanRunModel(tc.modelSize)
			assert.Equal(t, tc.expectCan, canRun)
		})
	}
}

// TestCapabilities_StringSimple tests String method with different GPU configurations
func TestCapabilities_StringSimple(t *testing.T) {
	testCases := []struct {
		name        string
		capabilities *Capabilities
		expectGPU    bool
	}{
		{
			"No GPU",
			&Capabilities{
				TotalRAM:     8 * 1024 * 1024 * 1024,
				AvailableRAM:  4 * 1024 * 1024 * 1024,
				MaxModelSize: 100 * 1024 * 1024,
				CPUModel:      "Test CPU",
				CPUCores:      4,
				HasGPU:        false,
				GPUType:       "",
			},
			false,
		},
		{
			"Metal GPU",
			&Capabilities{
				TotalRAM:     8 * 1024 * 1024 * 1024,
				AvailableRAM:  4 * 1024 * 1024 * 1024,
				MaxModelSize: 100 * 1024 * 1024,
				CPUModel:      "Test CPU",
				CPUCores:      4,
				HasGPU:        true,
				GPUType:       "metal",
			},
			true,
		},
		{
			"CUDA GPU",
			&Capabilities{
				TotalRAM:     8 * 1024 * 1024 * 1024,
				AvailableRAM:  4 * 1024 * 1024 * 1024,
				MaxModelSize: 100 * 1024 * 1024,
				CPUModel:      "Test CPU",
				CPUCores:      4,
				HasGPU:        true,
				GPUType:       "cuda",
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.capabilities.String()
			assert.NotEmpty(t, str)
			
			// Check if GPU type is in the string based on expectation
			if tc.expectGPU {
				assert.Contains(t, str, tc.capabilities.GPUType)
			} else {
				assert.Contains(t, str, "No GPU")
			}
		})
	}
}

// TestHardwareDetector_EdgeCases tests edge cases
func TestHardwareDetector_EdgeCases(t *testing.T) {
	detector := NewDetector()

	t.Run("Zero RAM calculation", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(0, false)
		assert.Equal(t, uint64(0), modelSize)
	})

	t.Run("Max RAM calculation without GPU", func(t *testing.T) {
		// Test with maximum reasonable RAM value
		modelSize := detector.calculateMaxModelSize(1024*1024*1024*1024, false) // 1TB
		assert.Greater(t, modelSize, uint64(0))
	})

	t.Run("Max RAM calculation with GPU", func(t *testing.T) {
		// Test with maximum reasonable RAM value and GPU
		modelSize := detector.calculateMaxModelSize(1024*1024*1024*1024, true) // 1TB with GPU
		assert.Greater(t, modelSize, uint64(0))
	})
}