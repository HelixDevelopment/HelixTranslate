package hardware

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Capabilities represents system hardware capabilities
type Capabilities struct {
	Architecture string  // arm64, amd64, etc.
	TotalRAM     uint64  // in bytes
	AvailableRAM uint64  // in bytes
	CPUModel     string  // e.g., "Apple M3 Pro"
	CPUCores     int     // physical cores
	HasGPU       bool    // GPU acceleration available
	GPUType      string  // metal, cuda, rocm, vulkan, or empty
	MaxModelSize uint64  // estimated max model size in parameters (7B, 13B, etc.)
}

// Detector provides hardware detection functionality
type Detector struct{}

// NewDetector creates a new hardware detector
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes system hardware and returns capabilities
func (d *Detector) Detect() (*Capabilities, error) {
	caps := &Capabilities{
		Architecture: runtime.GOARCH,
	}

	var err error

	// Detect RAM
	caps.TotalRAM, err = d.getTotalRAM()
	if err != nil {
		return nil, fmt.Errorf("failed to detect total RAM: %w", err)
	}

	caps.AvailableRAM, err = d.getAvailableRAM()
	if err != nil {
		// Estimate as 70% of total if we can't get precise value
		caps.AvailableRAM = uint64(float64(caps.TotalRAM) * 0.7)
	}

	// Detect CPU
	caps.CPUModel, err = d.getCPUModel()
	if err != nil {
		caps.CPUModel = "Unknown"
	}

	caps.CPUCores, err = d.getCPUCores()
	if err != nil {
		caps.CPUCores = runtime.NumCPU()
	}

	// Detect GPU
	caps.HasGPU, caps.GPUType = d.detectGPU()

	// Calculate max model size based on available RAM
	caps.MaxModelSize = d.calculateMaxModelSize(caps.AvailableRAM, caps.HasGPU)

	return caps, nil
}

// getTotalRAM returns total system RAM in bytes
func (d *Detector) getTotalRAM() (uint64, error) {
	switch runtime.GOOS {
	case "darwin":
		return d.getMacOSRAM()
	case "linux":
		return d.getLinuxRAM()
	case "windows":
		return d.getWindowsRAM()
	default:
		return 0, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// getMacOSRAM gets RAM on macOS
func (d *Detector) getMacOSRAM() (uint64, error) {
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	ramBytes, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, err
	}

	return ramBytes, nil
}

// getLinuxRAM gets RAM on Linux
func (d *Detector) getLinuxRAM() (uint64, error) {
	cmd := exec.Command("grep", "MemTotal", "/proc/meminfo")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// MemTotal:       16384000 kB
	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected meminfo format")
	}

	ramKB, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}

	return ramKB * 1024, nil
}

// getWindowsRAM gets RAM on Windows
func (d *Detector) getWindowsRAM() (uint64, error) {
	cmd := exec.Command("wmic", "computersystem", "get", "totalphysicalmemory")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected wmic output")
	}

	ramBytes, err := strconv.ParseUint(strings.TrimSpace(lines[1]), 10, 64)
	if err != nil {
		return 0, err
	}

	return ramBytes, nil
}

// getAvailableRAM returns available RAM in bytes
func (d *Detector) getAvailableRAM() (uint64, error) {
	switch runtime.GOOS {
	case "darwin":
		// On macOS, use vm_stat to get available memory
		cmd := exec.Command("vm_stat")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}

		// Parse vm_stat output to get free + inactive + speculative pages
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

		// Available RAM = free + inactive + speculative pages
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

	default:
		return 0, fmt.Errorf("not implemented for %s", runtime.GOOS)
	}
}

// getCPUModel returns the CPU model string
func (d *Detector) getCPUModel() (string, error) {
	switch runtime.GOOS {
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

	default:
		return "", fmt.Errorf("not implemented for %s", runtime.GOOS)
	}
}

// getCPUCores returns the number of physical CPU cores
func (d *Detector) getCPUCores() (int, error) {
	switch runtime.GOOS {
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

	default:
		return 0, fmt.Errorf("not implemented for %s", runtime.GOOS)
	}
}

// detectGPU detects GPU presence and type
func (d *Detector) detectGPU() (bool, string) {
	// Check for Metal (Apple Silicon)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return true, "metal"
	}

	// Check for NVIDIA CUDA
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return true, "cuda"
	}

	// Check for AMD ROCm
	if _, err := exec.LookPath("rocm-smi"); err == nil {
		return true, "rocm"
	}

	// Check for Vulkan
	if _, err := exec.LookPath("vulkaninfo"); err == nil {
		return true, "vulkan"
	}

	return false, ""
}

// calculateMaxModelSize estimates maximum model size in parameters (e.g., 7B, 13B)
// Based on available RAM and GPU acceleration
func (d *Detector) calculateMaxModelSize(availableRAM uint64, hasGPU bool) uint64 {
	// Convert RAM to GB
	ramGB := float64(availableRAM) / (1024 * 1024 * 1024)

	// Rule of thumb: Model needs ~2x its size in parameters for inference
	// - 7B model needs ~14GB RAM (Q4 quant: ~7GB, Q8: ~10GB)
	// - 13B model needs ~26GB RAM (Q4 quant: ~13GB, Q8: ~18GB)
	// - 27B model needs ~54GB RAM (Q4 quant: ~27GB, Q8: ~36GB)

	// With GPU acceleration, we can use less RAM
	multiplier := 2.0
	if hasGPU {
		multiplier = 1.5
	}

	// Estimate max model size in billions of parameters
	maxParams := ramGB / multiplier

	// Round to standard model sizes: 7B, 13B, 27B, 70B, etc.
	if maxParams >= 70 {
		return 70_000_000_000
	} else if maxParams >= 27 {
		return 27_000_000_000
	} else if maxParams >= 13 {
		return 13_000_000_000
	} else if maxParams >= 7 {
		return 7_000_000_000
	} else if maxParams >= 3 {
		return 3_000_000_000
	}

	return 1_000_000_000 // 1B minimum
}

// String returns a human-readable summary of capabilities
func (c *Capabilities) String() string {
	ramGB := float64(c.TotalRAM) / (1024 * 1024 * 1024)
	availGB := float64(c.AvailableRAM) / (1024 * 1024 * 1024)
	maxModelB := float64(c.MaxModelSize) / 1_000_000_000

	gpuInfo := "None"
	if c.HasGPU {
		gpuInfo = fmt.Sprintf("%s acceleration", c.GPUType)
	}

	return fmt.Sprintf(
		"Hardware Capabilities:\n"+
			"  Architecture: %s\n"+
			"  CPU: %s (%d cores)\n"+
			"  Total RAM: %.1f GB\n"+
			"  Available RAM: %.1f GB\n"+
			"  GPU: %s\n"+
			"  Max Model Size: %.0fB parameters",
		c.Architecture, c.CPUModel, c.CPUCores,
		ramGB, availGB, gpuInfo, maxModelB,
	)
}

// CanRunModel checks if the system can run a model of given size
func (c *Capabilities) CanRunModel(modelSizeB uint64) bool {
	return modelSizeB <= c.MaxModelSize
}
