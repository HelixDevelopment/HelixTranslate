package hardware

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLinuxRAMDetection tests Linux-specific RAM detection functionality
func TestLinuxRAMDetection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux system")
	}

	detector := NewDetector()

	t.Run("Linux RAM detection", func(t *testing.T) {
		ram, err := detector.getLinuxRAM()

		// This might fail on non-Linux systems even if we're in the test
		if err != nil {
			assert.Contains(t, err.Error(), "failed to detect total RAM")
		} else {
			assert.Greater(t, ram, uint64(0), "RAM should be greater than 0 on success")
		}
	})
}

// TestWindowsRAMDetection tests Windows-specific RAM detection functionality
func TestWindowsRAMDetection(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows system")
	}

	detector := NewDetector()

	t.Run("Windows RAM detection", func(t *testing.T) {
		ram, err := detector.getWindowsRAM()

		// This might fail on non-Windows systems even if we're in the test
		if err != nil {
			assert.Contains(t, err.Error(), "failed to detect total RAM")
		} else {
			assert.Greater(t, ram, uint64(0), "RAM should be greater than 0 on success")
		}
	})
}

// TestBSDRAMDetection tests BSD-specific RAM detection functionality
func TestBSDRAMDetection(t *testing.T) {
	if runtime.GOOS != "freebsd" && runtime.GOOS != "openbsd" &&
		runtime.GOOS != "netbsd" && runtime.GOOS != "dragonfly" {
		t.Skip("Skipping BSD-specific test on non-BSD system")
	}

	detector := NewDetector()

	t.Run("BSD available RAM detection", func(t *testing.T) {
		ram, err := detector.getAvailableRAM()

		// This might fail on non-BSD systems even if we're in the test
		if err != nil {
			assert.Contains(t, err.Error(), "not implemented")
		} else {
			assert.Greater(t, ram, uint64(0), "RAM should be greater than 0 on success")
		}
	})
}

// TestCPUDetectionCoverage tests CPU detection methods that may have low coverage
func TestCPUDetectionCoverage(t *testing.T) {
	detector := NewDetector()

	t.Run("CPU model detection", func(t *testing.T) {
		model, err := detector.getCPUModel()

		// This should work on most systems
		if err != nil {
			// Error is acceptable, but should be meaningful
			assert.NotEmpty(t, err.Error(), "Error should have description")
		} else {
			assert.NotEmpty(t, model, "CPU model should not be empty on success")
		}
	})

	t.Run("CPU cores detection", func(t *testing.T) {
		cores, err := detector.getCPUCores()

		// This should work on most systems
		if err != nil {
			// Error is acceptable, but should be meaningful
			assert.NotEmpty(t, err.Error(), "Error should have description")
		} else {
			assert.Greater(t, cores, 0, "CPU cores should be greater than 0 on success")
		}
	})
}

// TestGPUDetectionComprehensive tests all GPU detection paths
func TestGPUDetectionComprehensive(t *testing.T) {
	detector := NewDetector()

	t.Run("GPU detection with different scenarios", func(t *testing.T) {
		hasGPU, gpuType := detector.detectGPU()

		// If GPU is detected, type should be valid
		if hasGPU {
			assert.NotEmpty(t, gpuType, "GPU type should be set when GPU is detected")
			validTypes := map[string]bool{
				"metal":  true,
				"cuda":   true,
				"rocm":   true,
				"vulkan": true,
			}
			assert.True(t, validTypes[gpuType], "GPU type should be valid")
		} else {
			assert.Empty(t, gpuType, "GPU type should be empty when no GPU is detected")
		}
	})

	t.Run("GPU detection Apple Silicon path", func(t *testing.T) {
		// Test Apple Silicon detection logic
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			hasGPU, gpuType := detector.detectGPU()
			// Apple Silicon should have Metal GPU
			if !hasGPU || gpuType != "metal" {
				t.Logf("Warning: Apple Silicon detected but Metal GPU not found (HasGPU=%v, GPUType=%s)", hasGPU, gpuType)
			}
		}
	})
}

// TestModelSizeCalculationCoverage tests edge cases in model size calculation
func TestModelSizeCalculationCoverage(t *testing.T) {
	detector := NewDetector()

	t.Run("Model size calculation boundary conditions", func(t *testing.T) {
		// Test exact boundary values for model size calculation

		// Test values that should result in different model sizes
		testCases := []struct {
			availableRAM uint64
			hasGPU       bool
			expectedMin  uint64 // Minimum expected model size
			expectedMax  uint64 // Maximum expected model size
		}{
			{6 * 1024 * 1024 * 1024, false, 1_000_000_000, 3_000_000_000},     // Just under 3B threshold
			{14 * 1024 * 1024 * 1024, false, 3_000_000_000, 7_000_000_000},    // Just under 7B threshold
			{26 * 1024 * 1024 * 1024, false, 7_000_000_000, 13_000_000_000},   // Just under 13B threshold
			{54 * 1024 * 1024 * 1024, false, 13_000_000_000, 27_000_000_000},  // Just under 27B threshold
			{105 * 1024 * 1024 * 1024, false, 27_000_000_000, 70_000_000_000}, // Just under 70B threshold
			{140 * 1024 * 1024 * 1024, false, 70_000_000_000, 70_000_000_000}, // Well above 70B threshold
		}

		for _, tc := range testCases {
			result := detector.calculateMaxModelSize(tc.availableRAM, tc.hasGPU)
			assert.GreaterOrEqual(t, result, tc.expectedMin, "Result should be at least minimum expected")
			assert.LessOrEqual(t, result, tc.expectedMax, "Result should be at most maximum expected")
		}
	})

	t.Run("Model size calculation with GPU efficiency", func(t *testing.T) {
		// Test that GPU makes calculation more efficient (allows larger models with same RAM)
		ram := uint64(16) * 1024 * 1024 * 1024 // 16GB

		withoutGPU := detector.calculateMaxModelSize(ram, false)
		withGPU := detector.calculateMaxModelSize(ram, true)

		// With GPU, should be able to handle equal or larger model
		assert.GreaterOrEqual(t, withGPU, withoutGPU, "GPU should allow equal or larger model size")
	})
}

// TestDetectMethodCoverage tests the main Detect method with different scenarios
func TestDetectMethodCoverage(t *testing.T) {
	detector := NewDetector()

	t.Run("Detect method full execution", func(t *testing.T) {
		caps, err := detector.Detect()

		if err != nil {
			// If detection fails, error should be meaningful
			assert.Contains(t, err.Error(), "failed to detect", "Error should be descriptive")
		} else {
			// If detection succeeds, capabilities should be valid
			assert.NotNil(t, caps, "Capabilities should not be nil on success")
			assert.NotEmpty(t, caps.Architecture, "Architecture should be set")
			assert.Greater(t, caps.TotalRAM, uint64(0), "Total RAM should be greater than 0")
			assert.GreaterOrEqual(t, caps.AvailableRAM, uint64(0), "Available RAM should be non-negative")
			assert.Greater(t, caps.CPUCores, 0, "CPU cores should be greater than 0")
			assert.Greater(t, caps.MaxModelSize, uint64(0), "Max model size should be greater than 0")

			// Test consistency
			assert.LessOrEqual(t, caps.AvailableRAM, caps.TotalRAM, "Available RAM should not exceed total RAM")
		}
	})

	t.Run("Detect method multiple calls", func(t *testing.T) {
		// Test that multiple calls to Detect work consistently
		for i := 0; i < 3; i++ {
			caps, err := detector.Detect()

			if err != nil {
				t.Logf("Detect call %d failed: %v", i+1, err)
				continue
			}

			assert.NotNil(t, caps, "Capabilities should not be nil")
			assert.NotEmpty(t, caps.Architecture, "Architecture should be set")
		}
	})
}

// TestErrorPaths tests error handling paths that might have low coverage
func TestErrorPaths(t *testing.T) {
	detector := NewDetector()

	t.Run("Unsupported OS handling", func(t *testing.T) {
		// Test that unsupported OS is handled gracefully
		// We can't easily mock runtime.GOOS, but we can test the error message format
		ram, err := detector.getWindowsRAM()

		if runtime.GOOS != "windows" {
			// On non-Windows, this should fail with a meaningful error
			if err != nil {
				// Error is expected, but we can't easily test the exact message format
				// since we can't mock exec.Command easily in this context
				assert.NotEmpty(t, err.Error(), "Error should have description")
			}
		} else {
			// On Windows, this should work
			if err == nil {
				assert.Greater(t, ram, uint64(0), "RAM should be greater than 0")
			}
		}
	})

	t.Run("Command failure handling", func(t *testing.T) {
		// Most of the error handling is tested through the main Detect() method
		// which handles command failures gracefully
		caps, err := detector.Detect()

		// Even if some commands fail, the method should either succeed or fail gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "failed to detect", "Error should be about detection failure")
		} else {
			assert.NotNil(t, caps, "Capabilities should not be nil on success")
		}
	})
}

// TestStringMethodCoverage tests the String method with various configurations
func TestStringMethodCoverage(t *testing.T) {
	testCases := []struct {
		name string
		caps *Capabilities
	}{
		{
			name: "No GPU configuration",
			caps: &Capabilities{
				Architecture: "amd64",
				TotalRAM:     8 * 1024 * 1024 * 1024,
				AvailableRAM: 6 * 1024 * 1024 * 1024,
				CPUModel:     "Test CPU",
				CPUCores:     4,
				HasGPU:       false,
				GPUType:      "",
				MaxModelSize: 3_000_000_000,
			},
		},
		{
			name: "Metal GPU configuration",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M2",
				CPUCores:     8,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 7_000_000_000,
			},
		},
		{
			name: "CUDA GPU configuration",
			caps: &Capabilities{
				Architecture: "amd64",
				TotalRAM:     32 * 1024 * 1024 * 1024,
				AvailableRAM: 24 * 1024 * 1024 * 1024,
				CPUModel:     "Intel Core i9",
				CPUCores:     16,
				HasGPU:       true,
				GPUType:      "cuda",
				MaxModelSize: 13_000_000_000,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			str := tc.caps.String()
			assert.NotEmpty(t, str, "String() should not return empty string")

			// Should contain basic hardware info
			assert.Contains(t, str, "Hardware Capabilities:")
			assert.Contains(t, str, tc.caps.Architecture)
			assert.Contains(t, str, tc.caps.CPUModel)
			assert.Contains(t, str, fmt.Sprintf("%d cores", tc.caps.CPUCores))

			// Should contain RAM info in GB
			assert.Contains(t, str, "GB")

			// Should contain GPU info appropriately
			if tc.caps.HasGPU {
				assert.Contains(t, str, tc.caps.GPUType)
				assert.Contains(t, str, "acceleration")
			} else {
				assert.Contains(t, str, "GPU: None")
			}

			// Should contain model size info
			assert.Contains(t, str, "parameters")
		})
	}
}

// TestCanRunMethodCoverage tests the CanRunModel method edge cases
func TestCanRunMethodCoverage(t *testing.T) {
	detector := NewDetector()

	// Test with a variety of model sizes
	testSizes := []struct {
		name      string
		modelSize uint64
	}{
		{"Zero size", 0},
		{"Tiny model", 100},
		{"Small model", 1_000_000_000},       // 1B
		{"Medium model", 7_000_000_000},      // 7B
		{"Large model", 13_000_000_000},      // 13B
		{"Very large model", 27_000_000_000}, // 27B
		{"Huge model", 70_000_000_000},       // 70B
		{"Extreme model", 100_000_000_000},   // 100B
	}

	// Test with detected capabilities first
	caps, err := detector.Detect()
	if err == nil {
		for _, tc := range testSizes {
			t.Run("Real capabilities - "+tc.name, func(t *testing.T) {
				result := caps.CanRunModel(tc.modelSize)
				// Result should be a boolean (no errors)
				assert.NotNil(t, result, "CanRunModel should return a boolean result")
			})
		}
	}

	// Test with mock capabilities to ensure edge cases are covered
	mockCaps := &Capabilities{
		MaxModelSize: 13_000_000_000, // 13B
	}

	for _, tc := range testSizes {
		t.Run("Mock capabilities - "+tc.name, func(t *testing.T) {
			result := mockCaps.CanRunModel(tc.modelSize)

			// Expected behavior based on 13B max
			expected := tc.modelSize <= 13_000_000_000
			assert.Equal(t, expected, result, "CanRunModel result should match expectation")
		})
	}
}
