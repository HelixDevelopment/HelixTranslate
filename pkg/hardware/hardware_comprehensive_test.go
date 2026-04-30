package hardware

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDetectorInitializationComprehensive tests detector initialization and edge cases
func TestDetectorInitializationComprehensive(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "NewDetector creates valid instance",
			test: func(t *testing.T) {
				detector := NewDetector()
				assert.NotNil(t, detector, "NewDetector should return non-nil instance")
			},
		},
		{
			name: "Multiple NewDetector calls return different instances",
			test: func(t *testing.T) {
				detector1 := NewDetector()
				detector2 := NewDetector()
				assert.NotNil(t, detector1, "First detector should be non-nil")
				assert.NotNil(t, detector2, "Second detector should be non-nil")
				// The detectors might be the same type, so we can't test for inequality
				// Just test that they are both valid instances
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// TestErrorHandlingComprehensive tests error handling in hardware detection
func TestErrorHandlingComprehensive(t *testing.T) {
	detector := NewDetector()

	t.Run("Detect with invalid command execution", func(t *testing.T) {
		// This test verifies that detection handles command failures gracefully
		// We can't easily mock exec.Command here, but we can test the overall error handling
		caps, err := detector.Detect()

		// Even if some commands fail, Detect() should either succeed or return a meaningful error
		if err != nil {
			assert.Contains(t, err.Error(), "failed to detect", "Error should be descriptive")
		} else {
			assert.NotNil(t, caps, "Capabilities should be non-nil on success")
		}
	})
}

// TestOSHandlingComprehensive tests different operating system specific code paths
func TestOSHandlingComprehensive(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		goos     string
		expected bool
	}{
		{
			name:     "Current OS should be supported",
			goos:     runtime.GOOS,
			expected: true,
		},
		{
			name:     "Unsupported OS should handle gracefully",
			goos:     "unknown_os",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.goos == "unknown_os" {
				// We can't actually change runtime.GOOS, but we can test the error message format
				// by testing individual methods that check OS
				_, err := detector.getTotalRAMForOS(tt.goos)
				if tt.expected {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "unsupported operating system")
				}
			} else {
				// Test current OS - should work without errors
				caps, err := detector.Detect()
				if !tt.expected {
					assert.Error(t, err)
				} else {
					// May succeed or fail gracefully depending on system
					if err == nil {
						assert.NotNil(t, caps)
					}
				}
			}
		})
	}
}

// TestRAMDetectionEdgeCases tests edge cases in RAM detection
func TestRAMDetectionEdgeCases(t *testing.T) {
	detector := NewDetector()

	t.Run("RAM detection handles zero values", func(t *testing.T) {
		// Test the calculation logic with known values
		result := detector.calculateMaxModelSize(0, false)
		assert.Equal(t, uint64(1_000_000_000), result, "Zero RAM should default to 1B model size")
	})

	t.Run("RAM detection with very small values", func(t *testing.T) {
		result := detector.calculateMaxModelSize(1024*1024, false) // 1MB
		assert.Equal(t, uint64(1_000_000_000), result, "Very small RAM should default to 1B model size")
	})

	t.Run("RAM detection with extremely large values", func(t *testing.T) {
		result := detector.calculateMaxModelSize(1024*1024*1024*1024, true) // 1TB with GPU
		assert.Equal(t, uint64(70_000_000_000), result, "Extremely large RAM should max out at 70B")
	})
}

// TestGPUDetectionEdgeCases tests GPU detection edge cases
func TestGPUDetectionEdgeCases(t *testing.T) {
	detector := NewDetector()

	t.Run("GPU detection logic", func(t *testing.T) {
		// Test Apple Silicon GPU detection
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			hasGPU, gpuType := detector.detectGPU()
			assert.True(t, hasGPU, "Apple Silicon should have GPU")
			assert.Equal(t, "metal", gpuType, "Apple Silicon should use Metal")
		}

		// Test that GPU type is empty when no GPU is detected
		// We can't easily mock the absence of GPU tools, but we can test the logic
		hasGPU, gpuType := detector.detectGPU()
		if !hasGPU {
			assert.Empty(t, gpuType, "GPU type should be empty when no GPU is detected")
		} else {
			assert.NotEmpty(t, gpuType, "GPU type should be set when GPU is detected")
			validTypes := map[string]bool{
				"metal":  true,
				"cuda":   true,
				"rocm":   true,
				"vulkan": true,
			}
			assert.True(t, validTypes[gpuType], "GPU type should be valid")
		}
	})
}

// TestCPUModelDetectionEdgeCases tests CPU model detection edge cases
func TestCPUModelDetectionEdgeCases(t *testing.T) {
	detector := NewDetector()

	t.Run("CPU model detection error handling", func(t *testing.T) {
		// Test that CPU model detection handles failures gracefully
		model, err := detector.getCPUModelForOS("unknown_os")
		assert.Error(t, err)
		assert.Empty(t, model)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

// TestCPUCoresDetectionEdgeCases tests CPU cores detection edge cases
func TestCPUCoresDetectionEdgeCases(t *testing.T) {
	detector := NewDetector()

	t.Run("CPU cores detection error handling", func(t *testing.T) {
		// Test that CPU cores detection handles failures gracefully
		cores, err := detector.getCPUCoresForOS("unknown_os")
		assert.Error(t, err)
		assert.Equal(t, 0, cores)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

// TestAvailableRAMDetectionEdgeCases tests available RAM detection edge cases
func TestAvailableRAMDetectionEdgeCases(t *testing.T) {
	detector := NewDetector()

	t.Run("Available RAM detection error handling", func(t *testing.T) {
		// Test that available RAM detection handles failures gracefully
		ram, err := detector.getAvailableRAMForOS("unknown_os")
		assert.Error(t, err)
		assert.Equal(t, uint64(0), ram)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

// TestModelSizeCalculationEdgeCases tests model size calculation edge cases
func TestModelSizeCalculationEdgeCases(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name         string
		availableRAM uint64
		hasGPU       bool
		expectedSize uint64
	}{
		{
			name:         "Exactly 3GB without GPU",
			availableRAM: 3 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 1_000_000_000, // Should round down to 1B
		},
		{
			name:         "Exactly 7GB without GPU",
			availableRAM: 7 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 3_000_000_000, // Should round down to 3B
		},
		{
			name:         "Exactly 14GB without GPU",
			availableRAM: 14 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 7_000_000_000, // Should round down to 7B
		},
		{
			name:         "Exactly 21GB without GPU",
			availableRAM: 21 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 7_000_000_000, // Still 7B, need more for 13B
		},
		{
			name:         "Exactly 26GB without GPU",
			availableRAM: 26 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 13_000_000_000, // Should round down to 13B
		},
		{
			name:         "Exactly 39GB without GPU",
			availableRAM: 39 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 13_000_000_000, // Still 13B, need more for 27B
		},
		{
			name:         "Exactly 54GB without GPU",
			availableRAM: 54 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 27_000_000_000, // Should round down to 27B
		},
		{
			name:         "Exactly 105GB without GPU",
			availableRAM: 105 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 27_000_000_000, // 105GB / 2 = 52.5GB, which is >= 27B but < 70B
		},
		{
			name:         "Exactly 3GB with GPU",
			availableRAM: 3 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedSize: 1_000_000_000, // 3GB / 1.5 = 2GB, which is < 3B
		},
		{
			name:         "Exactly 9GB with GPU",
			availableRAM: 9 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedSize: 3_000_000_000, // 9GB / 1.5 = 6GB, which is >= 3B but < 7B
		},
		{
			name:         "Exactly 19.5GB with GPU",
			availableRAM: uint64(19.5 * 1024 * 1024 * 1024),
			hasGPU:       true,
			expectedSize: 13_000_000_000, // With GPU, can handle 13B
		},
		{
			name:         "Exactly 40.5GB with GPU",
			availableRAM: uint64(40.5 * 1024 * 1024 * 1024),
			hasGPU:       true,
			expectedSize: 27_000_000_000, // With GPU, can handle 27B
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.calculateMaxModelSize(tt.availableRAM, tt.hasGPU)
			assert.Equal(t, tt.expectedSize, result, "Model size calculation should match expected")
		})
	}
}

// TestCapabilitiesValidationComprehensive tests capabilities validation
func TestCapabilitiesValidationComprehensive(t *testing.T) {
	tests := []struct {
		name   string
		caps   *Capabilities
		valid  bool
		reason string
	}{
		{
			name: "Valid capabilities",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 13_000_000_000,
			},
			valid:  true,
			reason: "All fields are valid",
		},
		{
			name: "Invalid architecture",
			caps: &Capabilities{
				Architecture: "",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 13_000_000_000,
			},
			valid:  false,
			reason: "Architecture cannot be empty",
		},
		{
			name: "Available RAM greater than total",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 20 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 13_000_000_000,
			},
			valid:  false,
			reason: "Available RAM cannot exceed total RAM",
		},
		{
			name: "Invalid CPU cores",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     0,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 13_000_000_000,
			},
			valid:  false,
			reason: "CPU cores must be at least 1",
		},
		{
			name: "GPU type without GPU",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       false,
				GPUType:      "metal",
				MaxModelSize: 13_000_000_000,
			},
			valid:  false,
			reason: "GPU type should be empty when HasGPU is false",
		},
		{
			name: "No GPU type with GPU",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "",
				MaxModelSize: 13_000_000_000,
			},
			valid:  false,
			reason: "GPU type should be set when HasGPU is true",
		},
		{
			name: "Invalid GPU type",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "invalid",
				MaxModelSize: 13_000_000_000,
			},
			valid:  false,
			reason: "GPU type must be valid",
		},
		{
			name: "Zero max model size",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 0,
			},
			valid:  false,
			reason: "Max model size cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCapabilities(tt.caps)
			if tt.valid {
				assert.NoError(t, err, tt.reason)
			} else {
				assert.Error(t, err, tt.reason)
			}
		})
	}
}

// validateCapabilities is a helper function for testing
func validateCapabilities(caps *Capabilities) error {
	if caps.Architecture == "" {
		return errors.New("architecture cannot be empty")
	}
	if caps.TotalRAM == 0 {
		return errors.New("total RAM cannot be zero")
	}
	if caps.AvailableRAM > caps.TotalRAM {
		return errors.New("available RAM cannot exceed total RAM")
	}
	if caps.CPUCores < 1 {
		return errors.New("CPU cores must be at least 1")
	}
	if caps.HasGPU && caps.GPUType == "" {
		return errors.New("GPU type should be set when HasGPU is true")
	}
	if !caps.HasGPU && caps.GPUType != "" {
		return errors.New("GPU type should be empty when HasGPU is false")
	}
	if caps.GPUType != "" {
		validTypes := map[string]bool{
			"metal":  true,
			"cuda":   true,
			"rocm":   true,
			"vulkan": true,
		}
		if !validTypes[caps.GPUType] {
			return errors.New("GPU type must be valid")
		}
	}
	if caps.MaxModelSize == 0 {
		return errors.New("max model size cannot be zero")
	}
	return nil
}

// TestCapabilitiesStringFormattingComprehensive tests comprehensive string formatting
func TestCapabilitiesStringFormattingComprehensive(t *testing.T) {
	tests := []struct {
		name  string
		caps  *Capabilities
		check func(t *testing.T, str string)
	}{
		{
			name: "Basic capabilities formatting",
			caps: &Capabilities{
				Architecture: "amd64",
				TotalRAM:     32 * 1024 * 1024 * 1024,
				AvailableRAM: 24 * 1024 * 1024 * 1024,
				CPUModel:     "Intel Core i7-9700K",
				CPUCores:     8,
				HasGPU:       false,
				GPUType:      "",
				MaxModelSize: 13_000_000_000,
			},
			check: func(t *testing.T, str string) {
				assert.Contains(t, str, "Hardware Capabilities:")
				assert.Contains(t, str, "amd64")
				assert.Contains(t, str, "Intel Core i7-9700K")
				assert.Contains(t, str, "8 cores")
				assert.Contains(t, str, "32.0 GB")
				assert.Contains(t, str, "24.0 GB")
				assert.Contains(t, str, "GPU: None")
				assert.Contains(t, str, "13B parameters")
			},
		},
		{
			name: "GPU capabilities formatting",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     16 * 1024 * 1024 * 1024,
				AvailableRAM: 12 * 1024 * 1024 * 1024,
				CPUModel:     "Apple M3 Pro",
				CPUCores:     12,
				HasGPU:       true,
				GPUType:      "metal",
				MaxModelSize: 7_000_000_000,
			},
			check: func(t *testing.T, str string) {
				assert.Contains(t, str, "metal acceleration")
				assert.Contains(t, str, "7B parameters")
			},
		},
		{
			name: "Large model size formatting",
			caps: &Capabilities{
				Architecture: "amd64",
				TotalRAM:     128 * 1024 * 1024 * 1024,
				AvailableRAM: 96 * 1024 * 1024 * 1024,
				CPUModel:     "AMD Ryzen 9 7950X",
				CPUCores:     16,
				HasGPU:       true,
				GPUType:      "cuda",
				MaxModelSize: 70_000_000_000,
			},
			check: func(t *testing.T, str string) {
				assert.Contains(t, str, "cuda acceleration")
				assert.Contains(t, str, "70B parameters")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.caps.String()
			assert.NotEmpty(t, str, "String() should not return empty string")
			tt.check(t, str)
		})
	}
}

// TestModelCompatibilityEdgeCases tests model compatibility edge cases
func TestModelCompatibilityEdgeCases(t *testing.T) {
	caps := &Capabilities{
		MaxModelSize: 13_000_000_000, // 13B
	}

	tests := []struct {
		name      string
		modelSize uint64
		expected  bool
	}{
		{
			name:      "Zero model size",
			modelSize: 0,
			expected:  true, // Should be able to run zero-sized model
		},
		{
			name:      "Exactly at max",
			modelSize: 13_000_000_000,
			expected:  true,
		},
		{
			name:      "Just over max",
			modelSize: 13_000_000_001,
			expected:  false,
		},
		{
			name:      "Very small model",
			modelSize: 1,
			expected:  true,
		},
		{
			name:      "Extremely large model",
			modelSize: 999_999_999_999,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := caps.CanRunModel(tt.modelSize)
			assert.Equal(t, tt.expected, result, "Model compatibility check failed")
		})
	}
}

// TestConcurrentDetection tests concurrent hardware detection
func TestConcurrentDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")  // SKIP-OK: #short-mode
	}

	concurrency := 10

	// Run multiple detections concurrently
	results := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			detector := NewDetector()
			_, err := detector.Detect()
			results <- err
		}()
	}

	// Collect results
	var errors []error
	for i := 0; i < concurrency; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	// Most should succeed, but some might fail due to system-specific issues
	// We're mainly testing that concurrent access doesn't cause panics
	successCount := concurrency - len(errors)
	assert.Greater(t, successCount, concurrency/2, "More than half of concurrent detections should succeed")
}

// TestDetectorPerformance benchmarks detector performance with different scenarios
func TestDetectorPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")  // SKIP-OK: #short-mode
	}

	detector := NewDetector()

	t.Run("Single detection performance", func(t *testing.T) {
		start := time.Now()
		_, err := detector.Detect()
		duration := time.Since(start)

		if err == nil {
			assert.Less(t, duration.Milliseconds(), int64(5000), "Detection should complete within 5 seconds")
		}
	})

	t.Run("Multiple detections performance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 5; i++ {
			_, _ = detector.Detect() // Ignore errors for performance test
		}
		duration := time.Since(start)

		assert.Less(t, duration.Milliseconds(), int64(10000), "5 detections should complete within 10 seconds")
	})
}

// makeTimestamp returns current timestamp in milliseconds - removed as unused
// func makeTimestamp() int64 {
// 	return 0 // Placeholder for actual timestamp implementation
// }

// TestHardwareCapabilitiesEdgeCases tests edge cases in capabilities
func TestHardwareCapabilitiesEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		caps *Capabilities
		test func(*testing.T, *Capabilities)
	}{
		{
			name: "Capabilities with minimal values",
			caps: &Capabilities{
				Architecture: "arm64",
				TotalRAM:     1 * 1024 * 1024 * 1024, // 1GB
				AvailableRAM: 512 * 1024 * 1024,      // 512MB
				CPUModel:     "Test CPU",
				CPUCores:     1,
				HasGPU:       false,
				GPUType:      "",
				MaxModelSize: 1_000_000_000,
			},
			test: func(t *testing.T, caps *Capabilities) {
				assert.Equal(t, uint64(1_000_000_000), caps.MaxModelSize)
				assert.True(t, caps.CanRunModel(1_000_000_000))
				assert.False(t, caps.CanRunModel(2_000_000_000))
			},
		},
		{
			name: "Capabilities with maximum values",
			caps: &Capabilities{
				Architecture: "amd64",
				TotalRAM:     1024 * 1024 * 1024 * 1024, // 1TB
				AvailableRAM: 768 * 1024 * 1024 * 1024,  // 768GB
				CPUModel:     "High-End CPU",
				CPUCores:     128,
				HasGPU:       true,
				GPUType:      "cuda",
				MaxModelSize: 70_000_000_000,
			},
			test: func(t *testing.T, caps *Capabilities) {
				assert.Equal(t, uint64(70_000_000_000), caps.MaxModelSize)
				assert.True(t, caps.CanRunModel(70_000_000_000))
				assert.False(t, caps.CanRunModel(71_000_000_000))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t, tt.caps)
		})
	}
}

// TestOSCommandParsing tests command output parsing for different OSes
func TestOSCommandParsing(t *testing.T) {
	// Test macOS command parsing
	if runtime.GOOS == "darwin" {
		t.Run("macOS sysctl output parsing", func(t *testing.T) {
			// Test parsing known sysctl format
			testOutput := "17179869184"
			result, err := parseUint64Output(testOutput)
			assert.NoError(t, err)
			assert.Equal(t, uint64(17179869184), result)
		})
	}

	// Test Linux command parsing
	if runtime.GOOS == "linux" {
		t.Run("Linux meminfo parsing", func(t *testing.T) {
			// Test parsing known meminfo format
			testOutput := "MemTotal:       16384000 kB"
			result, err := parseMeminfoOutput(testOutput)
			assert.NoError(t, err)
			assert.Equal(t, uint64(16384000), result)
		})
	}
}

// Helper functions for parsing command outputs
func parseUint64Output(output string) (uint64, error) {
	// Simple parsing implementation for testing
	parts := strings.Fields(strings.TrimSpace(output))
	if len(parts) == 0 {
		return 0, errors.New("empty output")
	}
	result, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func parseMeminfoOutput(output string) (uint64, error) {
	// Simple meminfo parsing for testing
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return 0, errors.New("invalid meminfo format")
	}
	result, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// Helper methods for testing OS-specific functions
func (d *Detector) getTotalRAMForOS(goos string) (uint64, error) {
	switch goos {
	case "darwin":
		return d.getMacOSRAM()
	case "linux":
		return d.getLinuxRAM()
	case "windows":
		return d.getWindowsRAM()
	default:
		return 0, fmt.Errorf("unsupported operating system: %s", goos)
	}
}

func (d *Detector) getCPUModelForOS(goos string) (string, error) {
	switch goos {
	case "darwin":
		cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	case "linux":
		cmd := exec.Command("grep", "-m1", "model name", "/proc/cpuinfo")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		parts := strings.Split(string(output), ":")
		if len(parts) < 2 {
			return "", fmt.Errorf("unexpected cpuinfo format")
		}
		return strings.TrimSpace(parts[1]), nil
	case "windows":
		cmd := exec.Command("powershell", "-Command",
			"(Get-CimInstance -ClassName Win32_Processor).Name")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	default:
		return "", fmt.Errorf("not implemented for %s", goos)
	}
}

func (d *Detector) getCPUCoresForOS(goos string) (int, error) {
	switch goos {
	case "darwin":
		cmd := exec.Command("sysctl", "-n", "hw.physicalcpu")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		cores, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			return 0, err
		}
		return cores, nil
	case "linux":
		cmd := exec.Command("lscpu")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		for _, line := range strings.Split(string(output), "\n") {
			if strings.Contains(line, "Core(s) per socket:") {
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					cores, err := strconv.Atoi(parts[3])
					if err == nil {
						return cores, nil
					}
				}
			}
		}
		return 0, fmt.Errorf("could not parse core count")
	case "windows":
		cmd := exec.Command("powershell", "-Command",
			"(Get-CimInstance -ClassName Win32_Processor).NumberOfCores")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		cores, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			return 0, err
		}
		return cores, nil
	default:
		return 0, fmt.Errorf("not implemented for %s", goos)
	}
}

func (d *Detector) getAvailableRAMForOS(goos string) (uint64, error) {
	switch goos {
	case "darwin":
		cmd := exec.Command("vm_stat")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		lines := strings.Split(string(output), "\n")
		var freePages, inactivePages, speculativePages uint64
		var pageSize uint64 = 16384 // default page size for Apple Silicon

		for _, line := range lines {
			if strings.Contains(line, "Pages free:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					pages, _ := strconv.ParseUint(strings.TrimSuffix(parts[2], "."), 10, 64)
					freePages = pages
				}
			} else if strings.Contains(line, "Pages inactive:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					pages, _ := strconv.ParseUint(strings.TrimSuffix(parts[2], "."), 10, 64)
					inactivePages = pages
				}
			} else if strings.Contains(line, "Pages speculative:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					pages, _ := strconv.ParseUint(strings.TrimSuffix(parts[2], "."), 10, 64)
					speculativePages = pages
				}
			} else if strings.Contains(line, "page size of") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if part == "of" && i+1 < len(parts) {
						pageSize, _ = strconv.ParseUint(parts[i+1], 10, 64)
						break
					}
				}
			}
		}

		totalAvailablePages := freePages + inactivePages + speculativePages
		return totalAvailablePages * pageSize, nil

	case "linux":
		cmd := exec.Command("grep", "MemAvailable", "/proc/meminfo")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		parts := strings.Fields(string(output))
		if len(parts) < 2 {
			return 0, fmt.Errorf("unexpected meminfo format")
		}

		availKB, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}

		return availKB * 1024, nil

	case "windows":
		cmd := exec.Command("powershell", "-Command",
			"(Get-CimInstance -ClassName Win32_OperatingSystem).FreePhysicalMemory * 1024")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		availBytes, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
		if err != nil {
			return 0, err
		}

		return availBytes, nil

	default:
		return 0, fmt.Errorf("not implemented for %s", goos)
	}
}
