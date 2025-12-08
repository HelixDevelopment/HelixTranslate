package hardware

import (
	"testing"
)

// TestHardware_DetectorMethods tests private methods that are difficult to reach otherwise
func TestHardware_DetectorMethods(t *testing.T) {
	d := NewDetector()
	
	t.Run("MaxModelSizeEdgeCases", func(t *testing.T) {
		// Test various RAM values and GPU configurations
		// Note: calculation is: ramGB / multiplier (2.0 for CPU, 1.5 for GPU)
		// Then round to nearest standard model size
		testCases := []struct {
			ramGB      float64
			hasGPU     bool
			expectedMax uint64
		}{
			{1, false, 1_000_000_000},    // 1/2 = 0.5 -> 1B minimum
			{2, false, 1_000_000_000},    // 2/2 = 1 -> 1B minimum
			{4, false, 1_000_000_000},    // 4/2 = 2 -> 1B (next threshold is 3B)
			{8, false, 3_000_000_000},    // 8/2 = 4 -> 3B threshold
			{16, false, 7_000_000_000},   // 16/2 = 8 -> 7B threshold
			{32, false, 13_000_000_000},   // 32/2 = 16 -> 13B threshold
			{64, false, 27_000_000_000},   // 64/2 = 32 -> 27B threshold
			{128, false, 27_000_000_000},  // 128/2 = 64 -> 27B threshold (64 < 70)
			{256, false, 70_000_000_000},  // 256/2 = 128 -> 70B max
			{8, true, 3_000_000_000},     // 8/1.5 = 5.33 -> 3B threshold
			{16, true, 7_000_000_000},    // 16/1.5 = 10.66 -> 7B threshold
			{32, true, 13_000_000_000},   // 32/1.5 = 21.33 -> 13B threshold
			{64, true, 27_000_000_000},   // 64/1.5 = 42.66 -> 27B threshold
			{128, true, 70_000_000_000},  // 128/1.5 = 85.33 -> 70B max
		}
		
		for _, tc := range testCases {
			ramBytes := uint64(tc.ramGB * 1024 * 1024 * 1024)
			maxSize := d.calculateMaxModelSize(ramBytes, tc.hasGPU)
			if maxSize != tc.expectedMax {
				t.Errorf("For %.0fGB RAM (hasGPU=%v), expected max size %d, got %d",
					tc.ramGB, tc.hasGPU, tc.expectedMax, maxSize)
			}
		}
	})
	
	t.Run("GPUDetectionAllTypes", func(t *testing.T) {
		// Test GPU detection - we can't easily mock all GPU types
		// but we can test that the function runs and returns valid output
		hasGPU, gpuType := d.detectGPU()
		
		// GPU type should be empty if no GPU
		if hasGPU && gpuType == "" {
			t.Error("GPU type should not be empty when GPU is detected")
		}
		
		// Should not have GPU type when no GPU
		if !hasGPU && gpuType != "" {
			t.Error("GPU type should be empty when no GPU is detected")
		}
		
		// GPU type should be one of known types if detected
		if hasGPU && gpuType != "" {
			validTypes := map[string]bool{
				"metal":  true,
				"cuda":   true,
				"rocm":   true,
				"vulkan": true,
			}
			if !validTypes[gpuType] {
				t.Errorf("Unknown GPU type detected: %s", gpuType)
			}
		}
	})
	
	t.Run("CapabilitiesValidation", func(t *testing.T) {
		// Create test capabilities
		caps := &Capabilities{
			Architecture: "test-arch",
			TotalRAM:     16 * 1024 * 1024 * 1024, // 16GB
			AvailableRAM: 12 * 1024 * 1024 * 1024, // 12GB
			CPUModel:     "Test CPU",
			CPUCores:     8,
			HasGPU:       true,
			GPUType:      "test-gpu",
			MaxModelSize: 7_000_000_000,
		}
		
		// Test String method
		str := caps.String()
		if str == "" {
			t.Error("String should not be empty")
		}
		
		// Test CanRunModel method
		if !caps.CanRunModel(7_000_000_000) {
			t.Error("Should be able to run model equal to max size")
		}
		
		if caps.CanRunModel(8_000_000_000) {
			t.Error("Should not be able to run model larger than max size")
		}
		
		if !caps.CanRunModel(0) {
			t.Error("Should be able to run 0-sized model")
		}
	})
}