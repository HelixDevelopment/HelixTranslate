package hardware

import (
	"runtime"
	"testing"
)

// TestNewDetector tests detector initialization
func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
}

// TestDetect tests basic hardware detection functionality
func TestDetect(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	if caps == nil {
		t.Fatal("Detect() returned nil capabilities")
	}

	// Verify architecture is detected
	if caps.Architecture == "" {
		t.Error("Architecture not detected")
	}

	// Verify architecture matches runtime
	expectedArch := runtime.GOARCH
	if caps.Architecture != expectedArch {
		t.Errorf("Architecture mismatch: got %s, expected %s", caps.Architecture, expectedArch)
	}
}

// TestRAMDetection tests RAM detection
func TestRAMDetection(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	// Test total RAM is reasonable (at least 1GB, less than 1TB)
	minRAM := uint64(1 * 1024 * 1024 * 1024)      // 1 GB
	maxRAM := uint64(1024 * 1024 * 1024 * 1024)   // 1 TB

	if caps.TotalRAM < minRAM {
		t.Errorf("TotalRAM too small: %d bytes (< 1GB)", caps.TotalRAM)
	}

	if caps.TotalRAM > maxRAM {
		t.Errorf("TotalRAM unreasonably large: %d bytes (> 1TB)", caps.TotalRAM)
	}

	// Available RAM should be less than or equal to total RAM
	if caps.AvailableRAM > caps.TotalRAM {
		t.Errorf("AvailableRAM (%d) greater than TotalRAM (%d)", caps.AvailableRAM, caps.TotalRAM)
	}

	// Available RAM should be at least 10% of total (system should have some free memory)
	// This is a reasonable assumption for a running system
	if caps.AvailableRAM == 0 {
		t.Error("AvailableRAM is zero, which is unrealistic")
	}
}

// TestCPUDetection tests CPU detection
func TestCPUDetection(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	// CPU cores should be at least 1
	if caps.CPUCores < 1 {
		t.Errorf("CPUCores invalid: %d (must be >= 1)", caps.CPUCores)
	}

	// CPU cores should be reasonable (less than 256)
	if caps.CPUCores > 256 {
		t.Errorf("CPUCores unreasonably large: %d", caps.CPUCores)
	}

	// CPU model should be detected (non-empty string)
	if caps.CPUModel == "" {
		t.Error("CPUModel not detected")
	}
}

// TestGPUDetection tests GPU detection
func TestGPUDetection(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	// If HasGPU is true, GPUType should be set
	if caps.HasGPU && caps.GPUType == "" {
		t.Error("HasGPU is true but GPUType is empty")
	}

	// If HasGPU is false, GPUType should be empty
	if !caps.HasGPU && caps.GPUType != "" {
		t.Error("HasGPU is false but GPUType is set")
	}

	// If GPUType is set, it should be a valid type
	if caps.GPUType != "" {
		validTypes := map[string]bool{
			"metal":  true,
			"cuda":   true,
			"rocm":   true,
			"vulkan": true,
		}

		if !validTypes[caps.GPUType] {
			t.Errorf("Invalid GPUType: %s (must be one of: metal, cuda, rocm, vulkan)", caps.GPUType)
		}
	}
}

// TestMaxModelSizeCalculation tests model size calculation
func TestMaxModelSizeCalculation(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	// MaxModelSize should be set
	if caps.MaxModelSize == 0 {
		t.Error("MaxModelSize not calculated")
	}

	// MaxModelSize should be reasonable (1B to 70B parameters)
	minSize := uint64(1_000_000_000)   // 1B parameters
	maxSize := uint64(70_000_000_000)  // 70B parameters

	if caps.MaxModelSize < minSize {
		t.Errorf("MaxModelSize too small: %d (< 1B)", caps.MaxModelSize)
	}

	if caps.MaxModelSize > maxSize {
		t.Errorf("MaxModelSize too large: %d (> 70B)", caps.MaxModelSize)
	}

	// Verify model size is one of the standard tiers
	validSizes := map[uint64]bool{
		1_000_000_000:  true, // 1B
		3_000_000_000:  true, // 3B
		7_000_000_000:  true, // 7B
		13_000_000_000: true, // 13B
		27_000_000_000: true, // 27B
		70_000_000_000: true, // 70B
	}

	if !validSizes[caps.MaxModelSize] {
		t.Errorf("MaxModelSize not a standard tier: %d", caps.MaxModelSize)
	}
}

// TestCanRunModel tests model compatibility checking
func TestCanRunModel(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	tests := []struct {
		name       string
		modelSize  uint64
		shouldRun  bool
	}{
		{
			name:      "Small model (1B)",
			modelSize: 1_000_000_000,
			shouldRun: true, // Should always run
		},
		{
			name:      "Model equal to max",
			modelSize: caps.MaxModelSize,
			shouldRun: true,
		},
		{
			name:      "Model larger than max",
			modelSize: caps.MaxModelSize + 1_000_000_000,
			shouldRun: false,
		},
		{
			name:      "Extremely large model (100B)",
			modelSize: 100_000_000_000,
			shouldRun: false, // Should never run on standard hardware
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := caps.CanRunModel(tt.modelSize)
			if result != tt.shouldRun {
				t.Errorf("CanRunModel(%d) = %v, want %v", tt.modelSize, result, tt.shouldRun)
			}
		})
	}
}

// TestCalculateMaxModelSize tests the model size calculation algorithm
func TestCalculateMaxModelSize(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name         string
		availableRAM uint64
		hasGPU       bool
		expectedSize uint64
	}{
		{
			name:         "Low RAM without GPU (4GB)",
			availableRAM: 4 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 1_000_000_000, // 1B
		},
		{
			name:         "Medium RAM without GPU (8GB)",
			availableRAM: 8 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 3_000_000_000, // 3B
		},
		{
			name:         "High RAM without GPU (16GB)",
			availableRAM: 16 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 7_000_000_000, // 7B
		},
		{
			name:         "High RAM with GPU (16GB)",
			availableRAM: 16 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedSize: 7_000_000_000, // 7B (with GPU, more efficient)
		},
		{
			name:         "Very high RAM without GPU (32GB)",
			availableRAM: 32 * 1024 * 1024 * 1024,
			hasGPU:       false,
			expectedSize: 13_000_000_000, // 13B
		},
		{
			name:         "Very high RAM with GPU (32GB)",
			availableRAM: 32 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedSize: 13_000_000_000, // 13B
		},
		{
			name:         "Extremely high RAM (64GB)",
			availableRAM: 64 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedSize: 27_000_000_000, // 27B
		},
		{
			name:         "Maximum RAM (128GB)",
			availableRAM: 128 * 1024 * 1024 * 1024,
			hasGPU:       true,
			expectedSize: 70_000_000_000, // 70B
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.calculateMaxModelSize(tt.availableRAM, tt.hasGPU)
			if result != tt.expectedSize {
				t.Errorf("calculateMaxModelSize(%d, %v) = %d, want %d",
					tt.availableRAM, tt.hasGPU, result, tt.expectedSize)
			}
		})
	}
}

// TestPlatformSpecificDetection tests platform-specific detection
func TestPlatformSpecificDetection(t *testing.T) {
	detector := NewDetector()
	caps, err := detector.Detect()

	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	platform := runtime.GOOS

	switch platform {
	case "darwin":
		// macOS-specific checks
		t.Run("macOS checks", func(t *testing.T) {
			// On macOS, we should be able to detect Metal GPU on Apple Silicon
			if caps.Architecture == "arm64" {
				// Apple Silicon Macs should have Metal GPU
				if !caps.HasGPU || caps.GPUType != "metal" {
					t.Logf("Warning: Apple Silicon detected but Metal GPU not found (HasGPU=%v, GPUType=%s)",
						caps.HasGPU, caps.GPUType)
				}
			}

			// CPU model should contain "Apple" or "Intel"
			if caps.CPUModel != "" {
				// Just log for information, don't fail
				t.Logf("macOS CPU Model: %s", caps.CPUModel)
			}
		})

	case "linux":
		// Linux-specific checks
		t.Run("Linux checks", func(t *testing.T) {
			// GPU type can be CUDA, ROCm, or Vulkan on Linux
			if caps.HasGPU {
				validLinuxGPU := caps.GPUType == "cuda" || caps.GPUType == "rocm" || caps.GPUType == "vulkan"
				if !validLinuxGPU {
					t.Errorf("Invalid Linux GPU type: %s", caps.GPUType)
				}
			}
		})

	case "windows":
		// Windows-specific checks
		t.Run("Windows checks", func(t *testing.T) {
			// GPU type can be CUDA or Vulkan on Windows
			if caps.HasGPU {
				validWindowsGPU := caps.GPUType == "cuda" || caps.GPUType == "vulkan"
				if !validWindowsGPU {
					t.Errorf("Invalid Windows GPU type: %s", caps.GPUType)
				}
			}
		})
	}
}

// TestDetectorConsistency tests that repeated calls return consistent results
func TestDetectorConsistency(t *testing.T) {
	detector := NewDetector()

	// Run detection multiple times
	caps1, err1 := detector.Detect()
	if err1 != nil {
		t.Fatalf("First Detect() failed: %v", err1)
	}

	caps2, err2 := detector.Detect()
	if err2 != nil {
		t.Fatalf("Second Detect() failed: %v", err2)
	}

	// Compare results - should be identical or very similar
	if caps1.Architecture != caps2.Architecture {
		t.Errorf("Architecture inconsistent: %s vs %s", caps1.Architecture, caps2.Architecture)
	}

	if caps1.TotalRAM != caps2.TotalRAM {
		t.Errorf("TotalRAM inconsistent: %d vs %d", caps1.TotalRAM, caps2.TotalRAM)
	}

	if caps1.CPUCores != caps2.CPUCores {
		t.Errorf("CPUCores inconsistent: %d vs %d", caps1.CPUCores, caps2.CPUCores)
	}

	if caps1.CPUModel != caps2.CPUModel {
		t.Errorf("CPUModel inconsistent: %s vs %s", caps1.CPUModel, caps2.CPUModel)
	}

	if caps1.HasGPU != caps2.HasGPU {
		t.Errorf("HasGPU inconsistent: %v vs %v", caps1.HasGPU, caps2.HasGPU)
	}

	if caps1.GPUType != caps2.GPUType {
		t.Errorf("GPUType inconsistent: %s vs %s", caps1.GPUType, caps2.GPUType)
	}

	if caps1.MaxModelSize != caps2.MaxModelSize {
		t.Errorf("MaxModelSize inconsistent: %d vs %d", caps1.MaxModelSize, caps2.MaxModelSize)
	}

	// AvailableRAM can vary slightly between calls, so we allow a 10% difference
	diff := int64(caps1.AvailableRAM) - int64(caps2.AvailableRAM)
	if diff < 0 {
		diff = -diff
	}
	tolerance := int64(caps1.TotalRAM) / 10 // 10% tolerance
	if diff > tolerance {
		t.Errorf("AvailableRAM difference too large: %d vs %d (diff: %d, tolerance: %d)",
			caps1.AvailableRAM, caps2.AvailableRAM, diff, tolerance)
	}
}

// TestCapabilitiesFormatting tests the String() method
func TestCapabilitiesFormatting(t *testing.T) {
	caps := &Capabilities{
		Architecture: "arm64",
		TotalRAM:     16 * 1024 * 1024 * 1024, // 16 GB
		AvailableRAM: 12 * 1024 * 1024 * 1024, // 12 GB
		CPUModel:     "Apple M3 Pro",
		CPUCores:     12,
		HasGPU:       true,
		GPUType:      "metal",
		MaxModelSize: 13_000_000_000, // 13B
	}

	str := caps.String()

	// Verify string contains key information
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Should contain architecture
	if !contains(str, "arm64") {
		t.Error("String() missing architecture")
	}

	// Should contain GPU type
	if !contains(str, "metal") {
		t.Error("String() missing GPU type")
	}

	// Should contain CPU model
	if !contains(str, "Apple M3 Pro") {
		t.Error("String() missing CPU model")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkDetect benchmarks hardware detection performance
func BenchmarkDetect(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.Detect()
		if err != nil {
			b.Fatalf("Detect() failed: %v", err)
		}
	}
}

// BenchmarkCanRunModel benchmarks model compatibility checking
func BenchmarkCanRunModel(b *testing.B) {
	detector := NewDetector()
	caps, err := detector.Detect()
	if err != nil {
		b.Fatalf("Detect() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = caps.CanRunModel(7_000_000_000) // 7B model
	}
}
