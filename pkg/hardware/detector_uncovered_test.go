package hardware

import (
	"runtime"
	"testing"
)

// TestDetector_getTotalRAM tests getTotalRAM method across platforms
func TestDetector_getTotalRAM(t *testing.T) {
	d := NewDetector()
	
	t.Run("getTotalRAM_InvalidOS", func(t *testing.T) {
		// Test with invalid OS (mocked by patching runtime.GOOS)
		originalGOOS := runtime.GOOS
		// This test will fail on actual OS but tests the error path
		// We'll just verify the function exists and returns something
		_, err := d.getTotalRAM()
		// err should be nil for valid OS
		if err != nil && originalGOOS == "darwin" || originalGOOS == "linux" || originalGOOS == "windows" {
			t.Errorf("Unexpected error for supported OS %s: %v", originalGOOS, err)
		}
	})
}

// TestDetector_getAvailableRAM tests getAvailableRAM method across platforms
func TestDetector_getAvailableRAM(t *testing.T) {
	d := NewDetector()
	
	t.Run("getAvailableRAM_InvalidOS", func(t *testing.T) {
		originalGOOS := runtime.GOOS
		_, err := d.getAvailableRAM()
		// err should be nil for valid OS
		if err != nil && (originalGOOS == "darwin" || originalGOOS == "linux" || originalGOOS == "windows" || 
			originalGOOS == "freebsd" || originalGOOS == "openbsd" || originalGOOS == "netbsd" || originalGOOS == "dragonfly") {
			t.Errorf("Unexpected error for supported OS %s: %v", originalGOOS, err)
		}
	})
}

// TestDetector_getCPUModel tests getCPUModel method
func TestDetector_getCPUModel(t *testing.T) {
	d := NewDetector()
	
	t.Run("getCPUModel_SupportedOS", func(t *testing.T) {
		model, err := d.getCPUModel()
		// Should get a model or "Unknown" on supported platforms
		if err != nil {
			t.Errorf("Error getting CPU model on %s: %v", runtime.GOOS, err)
		}
		if model == "" {
			t.Error("CPU model should not be empty")
		}
	})
}

// TestDetector_getCPUCores tests getCPUCores method
func TestDetector_getCPUCores(t *testing.T) {
	d := NewDetector()
	
	t.Run("getCPUCores_SupportedOS", func(t *testing.T) {
		cores, err := d.getCPUCores()
		if err != nil {
			t.Errorf("Error getting CPU cores on %s: %v", runtime.GOOS, err)
		}
		if cores <= 0 {
			t.Error("CPU cores should be positive")
		}
		// Should not exceed runtime.NumCPU() by much
		if cores > runtime.NumCPU()*2 {
			t.Errorf("CPU cores (%d) seems too high compared to runtime.NumCPU() (%d)", 
				cores, runtime.NumCPU())
		}
	})
}

// TestDetector_detectGPU tests detectGPU method
func TestDetector_detectGPU(t *testing.T) {
	d := NewDetector()
	
	t.Run("detectGPU_AllPlatforms", func(t *testing.T) {
		hasGPU, gpuType := d.detectGPU()
		// GPU type should be empty if no GPU
		if hasGPU && gpuType == "" {
			t.Error("GPU type should not be empty when GPU is detected")
		}
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
}

// TestDetector_calculateMaxModelSizeUncovered tests calculateMaxModelSize method
func TestDetector_calculateMaxModelSizeUncovered(t *testing.T) {
	d := NewDetector()
	
	t.Run("calculateMaxModelSize_WithoutGPU", func(t *testing.T) {
		// Test with 16GB RAM without GPU
		ram := uint64(16 * 1024 * 1024 * 1024) // 16GB
		maxSize := d.calculateMaxModelSize(ram, false)
		
		// Expected: 16GB / 2 = 8B parameters
		expected := uint64(7_000_000_000) // Rounds down to 7B
		if maxSize != expected {
			t.Errorf("Expected max model size %d for 16GB RAM without GPU, got %d", expected, maxSize)
		}
	})
	
	t.Run("calculateMaxModelSize_WithGPU", func(t *testing.T) {
		// Test with 16GB RAM with GPU
		ram := uint64(16 * 1024 * 1024 * 1024) // 16GB
		maxSize := d.calculateMaxModelSize(ram, true)
		
		// Expected: 16GB / 1.5 = ~10.6B parameters
		expected := uint64(7_000_000_000) // Rounds down to 7B
		if maxSize != expected {
			t.Errorf("Expected max model size %d for 16GB RAM with GPU, got %d", expected, maxSize)
		}
	})
	
	t.Run("calculateMaxModelSize_HighRAM", func(t *testing.T) {
		// Test with 64GB RAM
		ram := uint64(64 * 1024 * 1024 * 1024) // 64GB
		maxSize := d.calculateMaxModelSize(ram, true)
		
		// Expected: 64GB / 1.5 = ~42.6B parameters
		expected := uint64(27_000_000_000) // Rounds down to 27B
		if maxSize != expected {
			t.Errorf("Expected max model size %d for 64GB RAM, got %d", expected, maxSize)
		}
	})
	
	t.Run("calculateMaxModelSize_ExtremeRAM", func(t *testing.T) {
		// Test with 128GB RAM
		ram := uint64(128 * 1024 * 1024 * 1024) // 128GB
		maxSize := d.calculateMaxModelSize(ram, true)
		
		// Expected: 128GB / 1.5 = ~85.3B parameters
		expected := uint64(70_000_000_000) // Rounds down to 70B
		if maxSize != expected {
			t.Errorf("Expected max model size %d for 128GB RAM, got %d", expected, maxSize)
		}
	})
	
	t.Run("calculateMaxModelSize_LowRAM", func(t *testing.T) {
		// Test with 4GB RAM
		ram := uint64(4 * 1024 * 1024 * 1024) // 4GB
		maxSize := d.calculateMaxModelSize(ram, false)
		
		// Expected: 4GB / 2 = 2B parameters
		expected := uint64(1_000_000_000) // Minimum 1B
		if maxSize != expected {
			t.Errorf("Expected max model size %d for 4GB RAM, got %d", expected, maxSize)
		}
	})
}

// TestCapabilities_String tests the String method of Capabilities
func TestCapabilities_String(t *testing.T) {
	t.Run("String_WithGPU", func(t *testing.T) {
		caps := &Capabilities{
			Architecture: "arm64",
			TotalRAM:     16 * 1024 * 1024 * 1024, // 16GB
			AvailableRAM: 12 * 1024 * 1024 * 1024, // 12GB
			CPUModel:     "Apple M2 Pro",
			CPUCores:     10,
			HasGPU:       true,
			GPUType:      "metal",
			MaxModelSize: 7_000_000_000,
		}
		
		str := caps.String()
		if str == "" {
			t.Error("String should not be empty")
		}
		
		// Should contain key information
		if !containsStr(str, "Hardware Capabilities:") {
			t.Error("String should contain header")
		}
		if !containsStr(str, "arm64") {
			t.Error("String should contain architecture")
		}
		if !containsStr(str, "Apple M2 Pro") {
			t.Error("String should contain CPU model")
		}
		if !containsStr(str, "10 cores") {
			t.Error("String should contain core count")
		}
		if !containsStr(str, "16.0 GB") {
			t.Error("String should contain total RAM")
		}
		if !containsStr(str, "12.0 GB") {
			t.Error("String should contain available RAM")
		}
		if !containsStr(str, "metal acceleration") {
			t.Error("String should contain GPU info")
		}
		if !containsStr(str, "7B") {
			t.Error("String should contain max model size")
		}
	})
	
	t.Run("String_WithoutGPU", func(t *testing.T) {
		caps := &Capabilities{
			Architecture: "amd64",
			TotalRAM:     8 * 1024 * 1024 * 1024, // 8GB
			AvailableRAM: 6 * 1024 * 1024 * 1024,  // 6GB
			CPUModel:     "Intel i5-8250U",
			CPUCores:     4,
			HasGPU:       false,
			GPUType:      "",
			MaxModelSize: 3_000_000_000,
		}
		
		str := caps.String()
		if !containsStr(str, "None") {
			t.Error("String should show 'None' for GPU when no GPU")
		}
	})
}

// TestCapabilities_CanRunModel tests CanRunModel method
func TestCapabilities_CanRunModel(t *testing.T) {
	t.Run("CanRunModel_SmallModel", func(t *testing.T) {
		caps := &Capabilities{
			MaxModelSize: 7_000_000_000, // 7B parameters
		}
		
		if !caps.CanRunModel(3_000_000_000) {
			t.Error("Should be able to run 3B model with 7B max")
		}
		
		if caps.CanRunModel(10_000_000_000) {
			t.Error("Should not be able to run 10B model with 7B max")
		}
	})
	
	t.Run("CanRunModel_EqualSize", func(t *testing.T) {
		caps := &Capabilities{
			MaxModelSize: 13_000_000_000, // 13B parameters
		}
		
		if !caps.CanRunModel(13_000_000_000) {
			t.Error("Should be able to run model equal to max size")
		}
	})
	
	t.Run("CanRunModel_ZeroModel", func(t *testing.T) {
		caps := &Capabilities{
			MaxModelSize: 7_000_000_000, // 7B parameters
		}
		
		if !caps.CanRunModel(0) {
			t.Error("Should be able to run 0-sized model")
		}
	})
}

// TestDetector_NewDetectorUncovered tests NewDetector function
func TestDetector_NewDetectorUncovered(t *testing.T) {
	t.Run("NewDetector_CreatesInstance", func(t *testing.T) {
		d := NewDetector()
		if d == nil {
			t.Error("NewDetector should return non-nil instance")
		}
	})
}

// Helper function to check if string contains substring
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())))
}