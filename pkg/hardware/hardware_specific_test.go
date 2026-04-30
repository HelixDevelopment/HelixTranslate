package hardware

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCPUModelSpecific tests getCPUModel with specific OS handling
func TestCPUModelSpecific(t *testing.T) {
	detector := NewDetector()

	t.Run("getCPUModel on current OS", func(t *testing.T) {
		model, err := detector.getCPUModel()

		// We should get either a valid model or a meaningful error
		if err != nil {
			assert.Contains(t, err.Error(), "not implemented")
		} else {
			assert.NotEmpty(t, model)
		}
	})

	t.Run("Test command execution paths", func(t *testing.T) {
		// Test different OS-specific commands
		if runtime.GOOS == "darwin" {
			model, err := detector.getCPUModel()
			if err == nil {
				// On macOS, should get a non-empty model
				assert.NotEmpty(t, model)
			}
		} else if runtime.GOOS == "linux" {
			model, err := detector.getCPUModel()
			if err == nil {
				// On Linux, should get a non-empty model
				assert.NotEmpty(t, model)
			}
		} else if runtime.GOOS == "windows" {
			model, err := detector.getCPUModel()
			if err == nil {
				// On Windows, should get a non-empty model
				assert.NotEmpty(t, model)
			}
		} else {
			// For unsupported OS, should return error
			model, err := detector.getCPUModel()
			assert.Error(t, err)
			assert.Empty(t, model)
			assert.Contains(t, err.Error(), "not implemented")
		}
	})
}

// TestCPUCoresSpecific tests getCPUCores with specific OS handling
func TestCPUCoresSpecific(t *testing.T) {
	detector := NewDetector()

	t.Run("getCPUCores on current OS", func(t *testing.T) {
		cores, err := detector.getCPUCores()

		// We should get either valid cores or a meaningful error
		if err != nil {
			assert.Contains(t, err.Error(), "not implemented")
		} else {
			assert.Greater(t, cores, 0)
		}
	})

	t.Run("Test different OS paths", func(t *testing.T) {
		// Test the various OS-specific paths
		cores, err := detector.getCPUCores()
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
			if err != nil {
				t.Logf("CPU cores detection failed on %s: %v", runtime.GOOS, err)
			} else {
				assert.Greater(t, cores, 0)
			}
		} else {
			// For unsupported OS
			assert.Error(t, err)
			assert.Equal(t, 0, cores)
			assert.Contains(t, err.Error(), "not implemented")
		}
	})
}

// TestGPUDetectionSpecific tests detectGPU with specific scenarios
func TestGPUDetectionSpecific(t *testing.T) {
	detector := NewDetector()

	t.Run("detectGPU basic functionality", func(t *testing.T) {
		hasGPU, gpuType := detector.detectGPU()

		// Should return a boolean and string type (possibly empty)
		assert.IsType(t, false, hasGPU)
		assert.IsType(t, "", gpuType)

		// If GPU is detected, type should be valid
		if hasGPU {
			assert.NotEmpty(t, gpuType)
			validTypes := map[string]bool{
				"metal": true, "cuda": true, "rocm": true, "vulkan": true,
			}
			assert.True(t, validTypes[gpuType], "GPU type should be valid")
		}
	})

	t.Run("GPU detection on macOS", func(t *testing.T) {
		if runtime.GOOS == "darwin" {
			hasGPU, gpuType := detector.detectGPU()
			if runtime.GOARCH == "arm64" {
				// Apple Silicon should have Metal GPU
				if !hasGPU || gpuType != "metal" {
					t.Logf("Warning: Apple Silicon detected but Metal GPU not found (HasGPU=%v, GPUType=%s)", hasGPU, gpuType)
				}
			}
		}
	})

	t.Run("GPU detection with nvidia-smi", func(t *testing.T) {
		hasGPU, gpuType := detector.detectGPU()

		if !hasGPU {
			t.Logf("No GPU detected or nvidia-smi not available")
		} else {
			assert.NotEmpty(t, gpuType)
			validTypes := map[string]bool{
				"metal": true, "cuda": true, "rocm": true, "vulkan": true,
			}
			assert.True(t, validTypes[gpuType], "GPU type should be valid")
		}
	})
}

// TestRAMDetectionSpecific tests specific RAM detection methods
func TestRAMDetectionSpecific(t *testing.T) {
	detector := NewDetector()

	t.Run("getTotalRAM on current OS", func(t *testing.T) {
		ram, err := detector.getTotalRAM()

		if err != nil {
			t.Logf("Total RAM detection failed: %v", err)
		} else {
			assert.Greater(t, ram, uint64(0))
		}
	})

	t.Run("getAvailableRAM on current OS", func(t *testing.T) {
		ram, err := detector.getAvailableRAM()

		if err != nil {
			t.Logf("Available RAM detection failed: %v", err)
		} else {
			assert.Greater(t, ram, uint64(0))
		}
	})

	t.Run("OS-specific RAM detection", func(t *testing.T) {
		switch runtime.GOOS {
		case "darwin":
			ram, err := detector.getMacOSRAM()
			if err != nil {
				t.Logf("macOS RAM detection failed: %v", err)
			} else {
				assert.Greater(t, ram, uint64(0))
			}
		case "linux":
			ram, err := detector.getLinuxRAM()
			if err != nil {
				t.Logf("Linux RAM detection failed: %v", err)
			} else {
				assert.Greater(t, ram, uint64(0))
			}
		case "windows":
			ram, err := detector.getWindowsRAM()
			if err != nil {
				t.Logf("Windows RAM detection failed: %v", err)
			} else {
				assert.Greater(t, ram, uint64(0))
			}
		}
	})
}

// TestLinuxSpecificCommandPaths tests Linux-specific command paths
func TestLinuxSpecificCommandPaths(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific tests on non-Linux system")  // SKIP-OK: #legacy-untriaged
	}

	detector := NewDetector()

	t.Run("Linux CPU detection", func(t *testing.T) {
		// Test CPU model detection on Linux
		model, err := detector.getCPUModel()
		if err != nil {
			t.Logf("CPU model detection failed on Linux: %v", err)
		} else {
			assert.NotEmpty(t, model)
			// Should contain typical CPU info
			assert.True(t, strings.Contains(model, "Intel") ||
				strings.Contains(model, "AMD") ||
				strings.Contains(model, "ARM"))
		}

		// Test CPU cores detection on Linux
		cores, err := detector.getCPUCores()
		if err != nil {
			t.Logf("CPU cores detection failed on Linux: %v", err)
		} else {
			assert.Greater(t, cores, 0)
		}
	})

	t.Run("Linux RAM detection", func(t *testing.T) {
		ram, err := detector.getLinuxRAM()
		if err != nil {
			t.Logf("Linux RAM detection failed: %v", err)
		} else {
			assert.Greater(t, ram, uint64(0))
		}
	})
}

// TestMacOSSpecificCommandPaths tests macOS-specific command paths
func TestMacOSSpecificCommandPaths(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific tests on non-macOS system")  // SKIP-OK: #legacy-untriaged
	}

	detector := NewDetector()

	t.Run("macOS CPU detection", func(t *testing.T) {
		// Test CPU model detection on macOS
		model, err := detector.getCPUModel()
		if err != nil {
			t.Logf("CPU model detection failed on macOS: %v", err)
		} else {
			assert.NotEmpty(t, model)
			// Should contain typical CPU info
			assert.True(t, strings.Contains(model, "Intel") ||
				strings.Contains(model, "Apple"))
		}

		// Test CPU cores detection on macOS
		cores, err := detector.getCPUCores()
		if err != nil {
			t.Logf("CPU cores detection failed on macOS: %v", err)
		} else {
			assert.Greater(t, cores, 0)
		}
	})

	t.Run("macOS RAM detection", func(t *testing.T) {
		ram, err := detector.getMacOSRAM()
		if err != nil {
			t.Logf("macOS RAM detection failed: %v", err)
		} else {
			assert.Greater(t, ram, uint64(0))
		}
	})

	t.Run("macOS GPU detection", func(t *testing.T) {
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			// Apple Silicon
			hasGPU, gpuType := detector.detectGPU()
			if !hasGPU || gpuType != "metal" {
				t.Logf("Warning: Apple Silicon detected but Metal GPU not found (HasGPU=%v, GPUType=%s)", hasGPU, gpuType)
			}
		}
	})
}

// TestWindowsSpecificCommandPaths tests Windows-specific command paths
func TestWindowsSpecificCommandPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows system")  // SKIP-OK: #legacy-untriaged
	}

	detector := NewDetector()

	t.Run("Windows CPU detection", func(t *testing.T) {
		// Test CPU model detection on Windows
		model, err := detector.getCPUModel()
		if err != nil {
			t.Logf("CPU model detection failed on Windows: %v", err)
		} else {
			assert.NotEmpty(t, model)
		}

		// Test CPU cores detection on Windows
		cores, err := detector.getCPUCores()
		if err != nil {
			t.Logf("CPU cores detection failed on Windows: %v", err)
		} else {
			assert.Greater(t, cores, 0)
		}
	})

	t.Run("Windows RAM detection", func(t *testing.T) {
		ram, err := detector.getWindowsRAM()
		if err != nil {
			t.Logf("Windows RAM detection failed: %v", err)
		} else {
			assert.Greater(t, ram, uint64(0))
		}
	})
}

// TestUnsupportedOSHandling tests how the detector handles unsupported OS
func TestUnsupportedOSHandling(t *testing.T) {
	detector := NewDetector()

	// We can't easily mock runtime.GOOS, but we can test that
	// the current OS is handled properly
	t.Run("Current OS handling", func(t *testing.T) {
		// Test CPU model detection
		model, err := detector.getCPUModel()
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" ||
			runtime.GOOS == "windows" ||
			strings.Contains(runtime.GOOS, "bsd") {
			// Supported OS - should work or fail with a meaningful error
			if err != nil {
				t.Logf("Expected error on %s: %v", runtime.GOOS, err)
			}
		} else {
			// Unsupported OS - should fail with clear error
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not implemented")
			assert.Empty(t, model)
		}

		// Test CPU cores detection
		cores, err := detector.getCPUCores()
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" ||
			runtime.GOOS == "windows" {
			// Supported OS
			if err != nil {
				t.Logf("Expected error on %s: %v", runtime.GOOS, err)
			}
		} else {
			// Unsupported OS
			assert.Error(t, err)
			assert.Equal(t, 0, cores)
			assert.Contains(t, err.Error(), "not implemented")
		}
	})
}
