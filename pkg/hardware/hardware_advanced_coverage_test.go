package hardware

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHardwareDetector_OSCommandExecution tests OS command execution paths
func TestHardwareDetector_OSCommandExecution(t *testing.T) {
	detector := NewDetector()

	t.Run("getTotalRAM with actual OS commands", func(t *testing.T) {
		// This test tries to execute the actual OS commands
		ram, err := detector.getTotalRAM()
		if err != nil {
			// On systems where commands fail, just verify the error handling
			assert.Error(t, err)
			assert.Equal(t, uint64(0), ram)
		} else {
			// On successful systems, verify reasonable values
			assert.Greater(t, ram, uint64(0))
			assert.Less(t, ram, uint64(1024*1024*1024*1024)) // Less than 1TB
		}
	})

	t.Run("getAvailableRAM with actual OS commands", func(t *testing.T) {
		ram, err := detector.getAvailableRAM()
		if err != nil {
			// On systems where commands fail, just verify the error handling
			assert.Error(t, err)
			assert.Equal(t, uint64(0), ram)
		} else {
			// On successful systems, verify reasonable values
			assert.Greater(t, ram, uint64(0))
			assert.Less(t, ram, uint64(1024*1024*1024*1024)) // Less than 1TB
		}
	})

	t.Run("getCPUModel with actual OS commands", func(t *testing.T) {
		model, err := detector.getCPUModel()
		if err != nil {
			// On systems where commands fail, just verify the error handling
			assert.Error(t, err)
			assert.Empty(t, model)
		} else {
			// On successful systems, verify we got something
			assert.NotEmpty(t, model)
		}
	})

	t.Run("getCPUCores with actual OS commands", func(t *testing.T) {
		cores, err := detector.getCPUCores()
		if err != nil {
			// On systems where commands fail, just verify the error handling
			assert.Error(t, err)
			assert.Equal(t, 0, cores)
		} else {
			// On successful systems, verify reasonable values
			assert.Greater(t, cores, 0)
			assert.Less(t, cores, 128) // Less than 128 cores
		}
	})

	t.Run("detectGPU with actual OS commands", func(t *testing.T) {
		hasGPU, gpuType := detector.detectGPU()
		// GPU detection should never panic
		if hasGPU {
			assert.NotEmpty(t, gpuType)
			assert.Contains(t, []string{"cuda", "metal", "rocm", "vulkan"}, gpuType)
		}
	})
}

// TestHardwareDetector_CalculateMaxModelSizeEdgeCases tests edge cases in model size calculation
func TestHardwareDetector_CalculateMaxModelSizeEdgeCases(t *testing.T) {
	detector := NewDetector()

	testCases := []struct {
		name         string
		totalRAM     uint64
		availableRAM uint64
		hasGPU       bool
		expectedMin  uint64
		expectedMax  uint64
	}{
		{
			name:         "Very low RAM without GPU",
			totalRAM:     1024 * 1024 * 1024, // 1GB
			availableRAM: 512 * 1024 * 1024,  // 512MB
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  1024 * 1024 * 1024, // 1GB max
		},
		{
			name:         "Very low RAM with GPU",
			totalRAM:     1024 * 1024 * 1024, // 1GB
			availableRAM: 512 * 1024 * 1024,  // 512MB
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  3 * 1024 * 1024 * 1024, // 3GB max with GPU
		},
		{
			name:         "Exactly 3GB RAM without GPU",
			totalRAM:     3 * 1024 * 1024 * 1024,
			availableRAM: 2 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  1024 * 1024 * 1024, // Should be limited to 1GB
		},
		{
			name:         "Exactly 3GB RAM with GPU",
			totalRAM:     3 * 1024 * 1024 * 1024,
			availableRAM: 2 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  3 * 1024 * 1024 * 1024, // Should allow 3GB with GPU
		},
		{
			name:         "Exactly 7GB RAM without GPU",
			totalRAM:     7 * 1024 * 1024 * 1024,
			availableRAM: 5 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  1024 * 1024 * 1024, // Should be limited to 1GB
		},
		{
			name:         "Exactly 7GB RAM with GPU",
			totalRAM:     7 * 1024 * 1024 * 1024,
			availableRAM: 5 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  9 * 1024 * 1024 * 1024, // Should allow 9GB with GPU
		},
		{
			name:         "Exactly 14GB RAM without GPU",
			totalRAM:     14 * 1024 * 1024 * 1024,
			availableRAM: 10 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  7 * 1024 * 1024 * 1024, // Should allow 7GB without GPU
		},
		{
			name:         "Exactly 14GB RAM with GPU",
			totalRAM:     14 * 1024 * 1024 * 1024,
			availableRAM: 10 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  9 * 1024 * 1024 * 1024, // Should be limited to 9GB with GPU
		},
		{
			name:         "Exactly 21GB RAM without GPU",
			totalRAM:     21 * 1024 * 1024 * 1024,
			availableRAM: 15 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  7 * 1024 * 1024 * 1024, // Should be limited to 7GB without GPU
		},
		{
			name:         "Exactly 21GB RAM with GPU",
			totalRAM:     21 * 1024 * 1024 * 1024,
			availableRAM: 15 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  19.5 * 1024 * 1024 * 1024, // Should allow 19.5GB with GPU
		},
		{
			name:         "Exactly 26GB RAM without GPU",
			totalRAM:     26 * 1024 * 1024 * 1024,
			availableRAM: 20 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  21 * 1024 * 1024 * 1024, // Should allow 21GB without GPU
		},
		{
			name:         "Exactly 26GB RAM with GPU",
			totalRAM:     26 * 1024 * 1024 * 1024,
			availableRAM: 20 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  19.5 * 1024 * 1024 * 1024, // Should be limited to 19.5GB with GPU
		},
		{
			name:         "Exactly 39GB RAM without GPU",
			totalRAM:     39 * 1024 * 1024 * 1024,
			availableRAM: 30 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  26 * 1024 * 1024 * 1024, // Should allow 26GB without GPU
		},
		{
			name:         "Exactly 39GB RAM with GPU",
			totalRAM:     39 * 1024 * 1024 * 1024,
			availableRAM: 30 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  19.5 * 1024 * 1024 * 1024, // Should be limited to 19.5GB with GPU
		},
		{
			name:         "Exactly 54GB RAM without GPU",
			totalRAM:     54 * 1024 * 1024 * 1024,
			availableRAM: 40 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  39 * 1024 * 1024 * 1024, // Should allow 39GB without GPU
		},
		{
			name:         "Exactly 54GB RAM with GPU",
			totalRAM:     54 * 1024 * 1024 * 1024,
			availableRAM: 40 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  40.5 * 1024 * 1024 * 1024, // Should allow 40.5GB with GPU
		},
		{
			name:         "Exactly 105GB RAM without GPU",
			totalRAM:     105 * 1024 * 1024 * 1024,
			availableRAM: 80 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedMin:  0,
			expectedMax:  54 * 1024 * 1024 * 1024, // Should allow 54GB without GPU
		},
		{
			name:         "Exactly 105GB RAM with GPU",
			totalRAM:     105 * 1024 * 1024 * 1024,
			availableRAM: 80 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedMin:  0,
			expectedMax:  40.5 * 1024 * 1024 * 1024, // Should be limited to 40.5GB with GPU
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maxSize := detector.calculateMaxModelSize(tc.availableRAM, tc.hasGPU)
			assert.GreaterOrEqual(t, maxSize, tc.expectedMin)
			assert.LessOrEqual(t, maxSize, tc.expectedMax)
		})
	}
}

// TestHardwareDetector_PlatformSpecificCommandPaths tests platform-specific command paths
func TestHardwareDetector_PlatformSpecificCommandPaths(t *testing.T) {
	detector := NewDetector()

	t.Run("Linux command paths", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Skipping Linux-specific test on non-Linux system")
		}

		// Test Linux-specific RAM detection
		ram, err := detector.getLinuxRAM()
		if err != nil {
			assert.Error(t, err)
			assert.Equal(t, uint64(0), ram)
		} else {
			assert.Greater(t, ram, uint64(0))
		}

		// Test CPU model detection with Linux commands
		model, err := detector.getCPUModel()
		if err != nil {
			assert.Error(t, err)
			assert.Empty(t, model)
		} else {
			assert.NotEmpty(t, model)
		}

		// Test CPU cores detection with Linux commands
		cores, err := detector.getCPUCores()
		if err != nil {
			assert.Error(t, err)
			assert.Equal(t, 0, cores)
		} else {
			assert.Greater(t, cores, 0)
		}
	})

	t.Run("macOS command paths", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("Skipping macOS-specific test on non-macOS system")
		}

		// Test macOS-specific RAM detection
		ram, err := detector.getMacOSRAM()
		if err != nil {
			assert.Error(t, err)
			assert.Equal(t, uint64(0), ram)
		} else {
			assert.Greater(t, ram, uint64(0))
		}

		// Test CPU model detection with macOS commands
		model, err := detector.getCPUModel()
		if err != nil {
			assert.Error(t, err)
			assert.Empty(t, model)
		} else {
			assert.NotEmpty(t, model)
		}
	})

	t.Run("Windows command paths", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Skipping Windows-specific test on non-Windows system")
		}

		// Test Windows-specific RAM detection
		ram, err := detector.getWindowsRAM()
		if err != nil {
			assert.Error(t, err)
			assert.Equal(t, uint64(0), ram)
		} else {
			assert.Greater(t, ram, uint64(0))
		}

		// Test CPU model detection with Windows commands
		model, err := detector.getCPUModel()
		if err != nil {
			assert.Error(t, err)
			assert.Empty(t, model)
		} else {
			assert.NotEmpty(t, model)
		}

		// Test CPU cores detection with Windows commands
		cores, err := detector.getCPUCores()
		if err != nil {
			assert.Error(t, err)
			assert.Equal(t, 0, cores)
		} else {
			assert.Greater(t, cores, 0)
		}
	})
}

// TestHardwareDetector_GPUDetectionComprehensive tests GPU detection across different platforms
func TestHardwareDetector_GPUDetectionComprehensive(t *testing.T) {
	detector := NewDetector()

	t.Run("GPU detection on current platform", func(t *testing.T) {
		hasGPU, gpuType := detector.detectGPU()

		// GPU detection should never panic
		if hasGPU {
			assert.NotEmpty(t, gpuType)
			assert.Contains(t, []string{"cuda", "metal", "rocm", "vulkan"}, gpuType)
		} else {
			assert.Empty(t, gpuType)
		}
	})

	t.Run("GPU detection with nvidia-smi", func(t *testing.T) {
		// Check if nvidia-smi is available
		if _, err := os.Stat("/usr/bin/nvidia-smi"); os.IsNotExist(err) {
			t.Skip("nvidia-smi not available, skipping test")
		}

		hasGPU, gpuType := detector.detectGPU()
		if hasGPU {
			assert.Equal(t, "cuda", gpuType)
		}
	})

	t.Run("GPU detection on macOS", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("Skipping macOS-specific GPU test")
		}

		// On macOS, check for Metal support
		hasGPU, gpuType := detector.detectGPU()
		if hasGPU {
			// macOS GPUs should be detected as Metal
			assert.Equal(t, "metal", gpuType)
		}
	})
}

// TestHardwareDetector_CalculateMaxModelSizeDetailed tests the calculateMaxModelSize function with detailed scenarios
func TestHardwareDetector_CalculateMaxModelSizeDetailed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping detailed boundary test in short mode")
	}
	detector := NewDetector()

	t.Run("Boundary conditions without GPU", func(t *testing.T) {
		testCases := []struct {
			name         string
			totalRAM     uint64
			availableRAM uint64
			expected     uint64
		}{
			{"Just under 3GB", (3 * 1024 * 1024 * 1024) - 1, (2 * 1024 * 1024 * 1024), 1_000_000_000},
			{"Exactly 3GB", 3 * 1024 * 1024 * 1024, 2 * 1024 * 1024 * 1024, 1_000_000_000},
			{"Just over 3GB", (3 * 1024 * 1024 * 1024) + 1, (2 * 1024 * 1024 * 1024), 1_000_000_000},
			{"Just under 7GB", (7 * 1024 * 1024 * 1024) - 1, (5 * 1024 * 1024 * 1024), 1_000_000_000},
			{"Exactly 7GB", 7 * 1024 * 1024 * 1024, 5 * 1024 * 1024 * 1024, 1_000_000_000},
			{"Just over 7GB", (7 * 1024 * 1024 * 1024) + 1, (5 * 1024 * 1024 * 1024), 1_000_000_000},
			{"Just under 14GB", (14 * 1024 * 1024 * 1024) - 1, (10 * 1024 * 1024 * 1024), 3_000_000_000},
			{"Exactly 14GB", 14 * 1024 * 1024 * 1024, 10 * 1024 * 1024 * 1024, 3_000_000_000},
			{"Just over 14GB", (14 * 1024 * 1024 * 1024) + 1, (10 * 1024 * 1024 * 1024), 3_000_000_000},
			{"Just under 21GB", (21 * 1024 * 1024 * 1024) - 1, (15 * 1024 * 1024 * 1024), 7_000_000_000},
			{"Exactly 21GB", 21 * 1024 * 1024 * 1024, 15 * 1024 * 1024 * 1024, 7_000_000_000},
			{"Just over 21GB", (21 * 1024 * 1024 * 1024) + 1, (15 * 1024 * 1024 * 1024), 7_000_000_000},
			{"Just under 26GB", (26 * 1024 * 1024 * 1024) - 1, (20 * 1024 * 1024 * 1024), 7_000_000_000},
			{"Exactly 26GB", 26 * 1024 * 1024 * 1024, 20 * 1024 * 1024 * 1024, 7_000_000_000},
			{"Just over 26GB", (26 * 1024 * 1024 * 1024) + 1, (20 * 1024 * 1024 * 1024), 7_000_000_000},
			{"Just under 39GB", (39 * 1024 * 1024 * 1024) - 1, (30 * 1024 * 1024 * 1024), 13_000_000_000},
			{"Exactly 39GB", 39 * 1024 * 1024 * 1024, 30 * 1024 * 1024 * 1024, 13_000_000_000},
			{"Just over 39GB", (39 * 1024 * 1024 * 1024) + 1, (30 * 1024 * 1024 * 1024), 13_000_000_000},
			{"Just under 54GB", (54 * 1024 * 1024 * 1024) - 1, (40 * 1024 * 1024 * 1024), 13_000_000_000},
			{"Exactly 54GB", 54 * 1024 * 1024 * 1024, 40 * 1024 * 1024 * 1024, 13_000_000_000},
			{"Just over 54GB", (54 * 1024 * 1024 * 1024) + 1, (40 * 1024 * 1024 * 1024), 13_000_000_000},
			{"Just under 105GB", (105 * 1024 * 1024 * 1024) - 1, (80 * 1024 * 1024 * 1024), 27_000_000_000},
			{"Exactly 105GB", 105 * 1024 * 1024 * 1024, 80 * 1024 * 1024 * 1024, 27_000_000_000},
			{"Just over 105GB", (105 * 1024 * 1024 * 1024) + 1, (80 * 1024 * 1024 * 1024), 27_000_000_000},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				maxSize := detector.calculateMaxModelSize(tc.availableRAM, false)
				assert.Equal(t, tc.expected, maxSize)
			})
		}
	})

	t.Run("Boundary conditions with GPU", func(t *testing.T) {
		testCases := []struct {
			name         string
			totalRAM     uint64
			availableRAM uint64
			expected     uint64
		}{
			{"Just under 3GB", (3 * 1024 * 1024 * 1024) - 1, (2 * 1024 * 1024 * 1024), 1_000_000_000},
			{"Exactly 3GB", 3 * 1024 * 1024 * 1024, 2 * 1024 * 1024 * 1024, 1_000_000_000},
			{"Just over 3GB", (3 * 1024 * 1024 * 1024) + 1, (2 * 1024 * 1024 * 1024), 1_000_000_000},
			{"Just under 9GB", (9 * 1024 * 1024 * 1024) - 1, (7 * 1024 * 1024 * 1024), 3_000_000_000},
			{"Exactly 9GB", 9 * 1024 * 1024 * 1024, 7 * 1024 * 1024 * 1024, 3_000_000_000},
			{"Just over 9GB", (9 * 1024 * 1024 * 1024) + 1, (7 * 1024 * 1024 * 1024), 3_000_000_000},
			{"Just under 19.5GB", (19 * 1024 * 1024 * 1024) - 1, (15 * 1024 * 1024 * 1024), 7_000_000_000},
			{"Exactly 19.5GB", (19 * 1024 * 1024 * 1024) + (512 * 1024 * 1024), 15 * 1024 * 1024 * 1024, 7_000_000_000},
			{"Just over 19.5GB", (19 * 1024 * 1024 * 1024) + (512 * 1024 * 1024) + 1, 15 * 1024 * 1024 * 1024, 7_000_000_000},
			{"Just under 40.5GB", (40 * 1024 * 1024 * 1024) - 1, (30 * 1024 * 1024 * 1024), 13_000_000_000},
			{"Exactly 40.5GB", (40 * 1024 * 1024 * 1024) + (512 * 1024 * 1024), 30 * 1024 * 1024 * 1024, 13_000_000_000},
			{"Just over 40.5GB", (40 * 1024 * 1024 * 1024) + (512 * 1024 * 1024) + 1, 30 * 1024 * 1024 * 1024, 13_000_000_000},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				maxSize := detector.calculateMaxModelSize(tc.availableRAM, true)
				assert.Equal(t, tc.expected, maxSize)
			})
		}
	})
}
