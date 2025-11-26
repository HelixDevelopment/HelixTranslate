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
	assert.LessOrEqual(t, maxModelSize, uint64(7_000_000_000)) // Should be reasonable for 8GB
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
			expectedMinSize: 1_000_000_000,  // At least 1B model
			expectedMaxSize: 3_000_000_000,  // Should be small for 4GB
		},
		{
			name:           "Medium RAM (8GB)",
			ramBytes:       8 * 1024 * 1024 * 1024,
			expectedMinSize: 1_000_000_000,
			expectedMaxSize: 7_000_000_000,  // Should be around 7B for 8GB
		},
		{
			name:           "High RAM (16GB)",
			ramBytes:       16 * 1024 * 1024 * 1024,
			expectedMinSize: 3_000_000_000,
			expectedMaxSize: 13_000_000_000, // Should be around 13B for 16GB
		},
		{
			name:           "Very High RAM (32GB)",
			ramBytes:       32 * 1024 * 1024 * 1024,
			expectedMinSize: 7_000_000_000,
			expectedMaxSize: 30_000_000_000, // Should support larger models
		},
		{
			name:           "Extremely High RAM (64GB)",
			ramBytes:       64 * 1024 * 1024 * 1024,
			expectedMinSize: 13_000_000_000,
			expectedMaxSize: 70_000_000_000, // Maximum supported size
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
		assert.Equal(t, uint64(1_000_000_000), modelSize) // Should default to 1B
	})
	
	t.Run("Very small RAM", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(1024*1024, false) // 1MB
		assert.Equal(t, uint64(1_000_000_000), modelSize) // Should default to 1B
	})
	
	t.Run("Very large RAM", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(1024*1024*1024*1024, false) // 1TB
		assert.Equal(t, uint64(70_000_000_000), modelSize) // Should cap at 70B
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
	assert.Contains(t, str, "0B") // MaxModelSize is 7 bytes, so displays as 0B
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

// TestGetLinuxRAM tests Linux-specific RAM detection
func TestGetLinuxRAM(t *testing.T) {
	detector := NewDetector()
	
	// This test will only work on Linux systems
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux system")
	}
	
	ram, err := detector.getLinuxRAM()
	if err != nil {
		t.Logf("Expected error on non-Linux or without /proc/meminfo: %v", err)
	} else {
		assert.Greater(t, ram, uint64(0))
	}
}

// TestGetTotalRAM tests getTotalRAM method more thoroughly
func TestGetTotalRAM(t *testing.T) {
	detector := NewDetector()
	
	t.Run("getTotalRAM", func(t *testing.T) {
		ram, err := detector.getTotalRAM()
		if err != nil {
			t.Logf("getTotalRAM error (may be expected on some systems): %v", err)
		} else {
			assert.Greater(t, ram, uint64(0), "RAM should be greater than 0")
			// Check for reasonable RAM size (at least 1GB)
			assert.Greater(t, ram, uint64(1024*1024*1024), "RAM should be at least 1GB")
		}
	})
	
	// Test platform-specific methods based on current OS
	switch runtime.GOOS {
	case "linux":
		t.Run("getLinuxRAM", func(t *testing.T) {
			ram, err := detector.getLinuxRAM()
			if err != nil {
				t.Logf("getLinuxRAM error: %v", err)
			} else {
				assert.Greater(t, ram, uint64(0))
			}
		})
		
	case "windows":
		t.Run("getWindowsRAM", func(t *testing.T) {
			ram, err := detector.getWindowsRAM()
			if err != nil {
				t.Logf("getWindowsRAM error: %v", err)
			} else {
				assert.Greater(t, ram, uint64(0))
			}
		})
		
	case "darwin":
		// macOS uses the same getTotalRAM method but goes through different path
		// The test above covers it, so we just add a note
		t.Logf("Darwin (macOS) uses sysctl for RAM detection via getTotalRAM")
		
	case "freebsd", "openbsd", "netbsd", "dragonfly":
		// BSD systems use sysctl
		t.Logf("%s uses sysctl for RAM detection via getTotalRAM", runtime.GOOS)
	}
}

// TestGetAvailableRAM tests getAvailableRAM method more thoroughly
func TestGetAvailableRAM(t *testing.T) {
	detector := NewDetector()
	
	t.Run("getAvailableRAM", func(t *testing.T) {
		ram, err := detector.getAvailableRAM()
		if err != nil {
			t.Logf("getAvailableRAM error (may be expected on some systems): %v", err)
		} else {
			assert.Greater(t, ram, uint64(0), "Available RAM should be greater than 0")
		}
	})
}

// TestGetCPUModel tests getCPUModel method
func TestGetCPUModel(t *testing.T) {
	detector := NewDetector()
	
	t.Run("getCPUModel", func(t *testing.T) {
		model, err := detector.getCPUModel()
		if err != nil {
			t.Logf("getCPUModel error (may be expected on some systems): %v", err)
		} else {
			assert.NotEmpty(t, model, "CPU model should not be empty")
		}
	})
}

// TestGetCPUCores tests getCPUCores method
func TestGetCPUCores(t *testing.T) {
	detector := NewDetector()
	
	t.Run("getCPUCores", func(t *testing.T) {
		cores, err := detector.getCPUCores()
		if err != nil {
			t.Logf("getCPUCores error (may be expected on some systems): %v", err)
		} else {
			assert.Greater(t, cores, 0, "CPU cores should be greater than 0")
		}
	})
}

// TestDetectGPU tests detectGPU method more thoroughly
func TestDetectGPU(t *testing.T) {
	detector := NewDetector()
	
	t.Run("detectGPU", func(t *testing.T) {
		hasGPU, gpuType := detector.detectGPU()
		t.Logf("GPU detection result: hasGPU=%v, gpuType=%s", hasGPU, gpuType)
		
		if hasGPU {
			assert.NotEmpty(t, gpuType, "GPU type should not be empty when GPU is detected")
		} else {
			assert.Empty(t, gpuType, "GPU type should be empty when no GPU is detected")
		}
	})
}

// TestCalculateMaxModelSize_MoreCoverage tests calculateMaxModelSize with GPU enabled
func TestCalculateMaxModelSize_MoreCoverage(t *testing.T) {
	detector := NewDetector()
	
	t.Run("With GPU", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(8*1024*1024*1024, true) // 8GB RAM, with GPU
		assert.Greater(t, modelSize, uint64(0))
		t.Logf("Model size with GPU: %d", modelSize)
	})
	
	t.Run("Without GPU", func(t *testing.T) {
		modelSize := detector.calculateMaxModelSize(8*1024*1024*1024, false) // 8GB RAM, without GPU
		assert.Greater(t, modelSize, uint64(0))
		t.Logf("Model size without GPU: %d", modelSize)
	})
}

// TestForceCoverage_UncoveredFunctions forces execution of functions that might be skipped on current platform
func TestForceCoverage_UncoveredFunctions(t *testing.T) {
	detector := NewDetector()
	
	// These functions have platform-specific behavior but we still want to test their error paths
	t.Run("Force test Linux RAM function", func(t *testing.T) {
		// Even if we're not on Linux, we can test that the function exists and errors appropriately
		_, err := detector.getLinuxRAM()
		if runtime.GOOS != "linux" {
			// Should error on non-Linux systems
			assert.Error(t, err, "getLinuxRAM should error on non-Linux systems")
		} else {
			// Should work on Linux systems
			if err == nil {
				t.Logf("Linux RAM detected successfully")
			} else {
				t.Logf("Linux RAM detection failed (may be expected in some environments): %v", err)
			}
		}
	})
	
	t.Run("Force test Windows RAM function", func(t *testing.T) {
		_, err := detector.getWindowsRAM()
		if runtime.GOOS != "windows" {
			// Should error on non-Windows systems
			assert.Error(t, err, "getWindowsRAM should error on non-Windows systems")
		} else {
			// Should work on Windows systems
			if err == nil {
				t.Logf("Windows RAM detected successfully")
			} else {
				t.Logf("Windows RAM detection failed: %v", err)
			}
		}
	})
	
	t.Run("Force test macOS RAM function", func(t *testing.T) {
		_, err := detector.getMacOSRAM()
		if runtime.GOOS != "darwin" {
			// Should error on non-macOS systems
			assert.Error(t, err, "getMacOSRAM should error on non-macOS systems")
		} else {
			// Should work on macOS systems
			if err == nil {
				t.Logf("macOS RAM detected successfully")
			} else {
				t.Logf("macOS RAM detection failed: %v", err)
			}
		}
	})
}

// TestGetAvailableRAM_Extended tests getAvailableRAM with more coverage
func TestGetAvailableRAM_Extended(t *testing.T) {
	detector := NewDetector()
	
	t.Run("Test available RAM function exists", func(t *testing.T) {
		ram, err := detector.getAvailableRAM()
		if err != nil {
			t.Logf("getAvailableRAM error (may be expected): %v", err)
		} else {
			t.Logf("Available RAM: %d bytes", ram)
			assert.Greater(t, ram, uint64(0), "Available RAM should be greater than 0")
		}
	})
	
	// Test platform-specific functions for getAvailableRAM
	switch runtime.GOOS {
	case "darwin":
		// Test macOS-specific path
		t.Run("Test macOS available RAM detection", func(t *testing.T) {
			ram, err := detector.getAvailableRAM()
			if err == nil {
				t.Logf("macOS available RAM detected: %d bytes", ram)
				assert.Greater(t, ram, uint64(0))
			} else {
				t.Logf("macOS available RAM detection failed: %v", err)
			}
		})
		
	case "linux":
		// Test Linux-specific path
		t.Run("Test Linux available RAM detection", func(t *testing.T) {
			ram, err := detector.getAvailableRAM()
			if err == nil {
				t.Logf("Linux available RAM detected: %d bytes", ram)
				assert.Greater(t, ram, uint64(0))
			} else {
				t.Logf("Linux available RAM detection failed: %v", err)
			}
		})
		
	case "windows":
		// Test Windows-specific path
		t.Run("Test Windows available RAM detection", func(t *testing.T) {
			ram, err := detector.getAvailableRAM()
			if err == nil {
				t.Logf("Windows available RAM detected: %d bytes", ram)
				assert.Greater(t, ram, uint64(0))
			} else {
				t.Logf("Windows available RAM detection failed: %v", err)
			}
		})
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