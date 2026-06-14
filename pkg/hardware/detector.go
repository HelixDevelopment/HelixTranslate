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
	Architecture string // arm64, amd64, etc.
	TotalRAM     uint64 // in bytes
	AvailableRAM uint64 // in bytes
	CPUModel     string // e.g., "Apple M3 Pro"
	CPUCores     int    // physical cores
	HasGPU       bool   // GPU acceleration available
	GPUType      string // metal, cuda, rocm, vulkan, or empty
	MaxModelSize uint64 // estimated max model size in parameters (7B, 13B, etc.)
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
	return parseSysctlBytes(string(output))
}

// parseSysctlBytes parses the raw output of `sysctl -n hw.memsize` (a bare
// byte count, optionally surrounded by whitespace, e.g. "17179869184\n") into
// a uint64 byte count.
func parseSysctlBytes(output string) (uint64, error) {
	ramBytes, err := strconv.ParseUint(strings.TrimSpace(output), 10, 64)
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
	return parseLinuxMeminfoKB(string(output))
}

// parseLinuxMeminfoKB parses a single /proc/meminfo line of the form
// "MemTotal:       16384000 kB" (or "MemAvailable: ...") — field [1] is the
// kB count — and returns the value converted to bytes.
func parseLinuxMeminfoKB(output string) (uint64, error) {
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected meminfo format")
	}

	ramKB, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, err
	}

	// Guard the kB->bytes conversion against uint64 overflow: a corrupt or
	// hostile /proc/meminfo could report a kB value above 2^64/1024, which
	// would silently wrap to an absurd small byte count (err==nil) and mis-size
	// the model downstream. Reject it instead of returning a corrupt value.
	const maxKB = ^uint64(0) / 1024
	if ramKB > maxKB {
		return 0, fmt.Errorf("meminfo kB value %d overflows uint64 byte count", ramKB)
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
	return parseWmicMem(string(output))
}

// parseWmicMem parses the raw output of `wmic computersystem get
// totalphysicalmemory`, whose first line is the column header
// "TotalPhysicalMemory" and whose second line is the byte count, e.g.:
//
//	TotalPhysicalMemory
//	17179869184
func parseWmicMem(output string) (uint64, error) {
	lines := strings.Split(output, "\n")
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
		return parseVMStatAvailableBytes(string(output)), nil

	case "linux":
		cmd := exec.Command("grep", "MemAvailable", "/proc/meminfo")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		return parseLinuxMeminfoKB(string(output))

	case "windows":
		// Use PowerShell to get available memory (more reliable than wmic)
		cmd := exec.Command("powershell", "-Command",
			"(Get-CimInstance -ClassName Win32_OperatingSystem).FreePhysicalMemory * 1024")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		return parsePowerShellBytes(string(output))

	case "freebsd", "openbsd", "netbsd", "dragonfly":
		// Use sysctl for BSD systems
		cmd := exec.Command("sysctl", "hw.usermem")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		return parseSysctlLabeledAvailableBytes(string(output))

	default:
		return 0, fmt.Errorf("not implemented for %s", runtime.GOOS)
	}
}

// parseVMStatAvailableBytes parses macOS `vm_stat` output, summing the free +
// inactive + speculative page counts and multiplying by the reported page
// size. The header line declares the page size ("...page size of 16384
// bytes)"); per-class lines look like "Pages free:    123456." (trailing
// period stripped). Page size defaults to 16384 (Apple Silicon) when the
// header is absent — matching the original inline behavior exactly.
func parseVMStatAvailableBytes(output string) uint64 {
	lines := strings.Split(output, "\n")
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
					// Only override the default when the header carries a
					// valid, non-zero page size. A malformed/empty value must
					// NOT zero out the page size — that would report all the
					// parsed pages as 0 bytes available RAM (worse than an
					// absent header, which keeps the 16384 default).
					if parsed, err := strconv.ParseUint(parts[i+1], 10, 64); err == nil && parsed > 0 {
						pageSize = parsed
					}
					break
				}
			}
		}
	}

	// Available RAM = free + inactive + speculative pages
	totalAvailablePages := freePages + inactivePages + speculativePages
	return totalAvailablePages * pageSize
}

// parsePowerShellBytes parses the raw output of a PowerShell expression that
// emits a bare byte count (e.g. FreePhysicalMemory * 1024) — a single integer
// optionally surrounded by whitespace — into a uint64 byte count.
func parsePowerShellBytes(output string) (uint64, error) {
	availBytes, err := strconv.ParseUint(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return 0, err
	}
	return availBytes, nil
}

// parseSysctlLabeledAvailableBytes parses labeled BSD `sysctl hw.usermem`
// output (format: "hw.usermem: 12345678"), then estimates available memory as
// ~70% of the reported total — matching the original inline behavior exactly.
func parseSysctlLabeledAvailableBytes(output string) (uint64, error) {
	totalMem, err := parseSysctlLabeledUint(output)
	if err != nil {
		return 0, err
	}
	// Estimate available memory (roughly 70% of total)
	return uint64(float64(totalMem) * 0.7), nil
}

// parseSysctlLabeledUint parses a labeled BSD sysctl line of the form
// "<key>: <value>" (e.g. "hw.usermem: 12345678", "hw.ncpu: 8") and returns the
// integer value. Used by the BSD RAM/core branches.
func parseSysctlLabeledUint(output string) (uint64, error) {
	parts := strings.Split(strings.TrimSpace(output), ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected sysctl format")
	}

	val, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, err
	}

	return val, nil
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
		return parseLinuxCPUModel(string(output))

	case "windows":
		// Use PowerShell to get CPU name
		cmd := exec.Command("powershell", "-Command",
			"(Get-CimInstance -ClassName Win32_Processor).Name")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil

	case "freebsd", "openbsd", "netbsd", "dragonfly":
		// Use sysctl for BSD systems
		cmd := exec.Command("sysctl", "hw.model")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return parseSysctlLabeledString(string(output))

	default:
		return "", fmt.Errorf("not implemented for %s", runtime.GOOS)
	}
}

// parseLinuxCPUModel parses a /proc/cpuinfo "model name" line of the form
// "model name      : Intel(R) Core(TM) i7-8700K" and returns the trimmed value
// after the first colon. Behavior-preserving: matches the original inline
// strings.Split(":")[1] semantics exactly.
func parseLinuxCPUModel(output string) (string, error) {
	parts := strings.Split(output, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected cpuinfo format")
	}
	return strings.TrimSpace(parts[1]), nil
}

// parseSysctlLabeledString parses a labeled BSD sysctl line of the form
// "<key>: <value>" (e.g. "hw.model: Intel(R) Core(TM) i7-8700K") and returns
// the trimmed string value. Behavior-preserving: matches the original inline
// strings.Split(":")[1] semantics exactly.
func parseSysctlLabeledString(output string) (string, error) {
	parts := strings.Split(strings.TrimSpace(output), ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected sysctl format")
	}
	return strings.TrimSpace(parts[1]), nil
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
		return parseSysctlInt(string(output))

	case "linux":
		cmd := exec.Command("lscpu")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		return parseLscpuCores(string(output))

	case "windows":
		// Use PowerShell to get physical cores
		cmd := exec.Command("powershell", "-Command",
			"(Get-CimInstance -ClassName Win32_Processor).NumberOfCores")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		return parseSysctlInt(string(output))

	case "freebsd", "openbsd", "netbsd", "dragonfly":
		// Use sysctl for BSD systems
		cmd := exec.Command("sysctl", "hw.ncpu")
		output, err := cmd.Output()
		if err != nil {
			return 0, err
		}
		return parseSysctlLabeledInt(string(output))

	default:
		return 0, fmt.Errorf("not implemented for %s", runtime.GOOS)
	}
}

// parseSysctlInt parses a bare integer from raw command output (a single
// number optionally surrounded by whitespace, e.g. "12\n"). Used by the macOS
// `sysctl -n hw.physicalcpu` and Windows NumberOfCores branches.
func parseSysctlInt(output string) (int, error) {
	cores, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, err
	}
	return cores, nil
}

// parseLscpuCores scans Linux `lscpu` output for the physical core count.
// CPUCores is documented as TOTAL physical cores, so on a multi-socket host
// that is (Core(s) per socket) * (Socket(s)) — e.g. a dual-socket server
// reporting "Core(s) per socket: 8" and "Socket(s): 2" has 16 physical cores,
// not 8. The "Core(s) per socket:" line is required (its absence is an error,
// matching the original inline loop); "Socket(s):" is optional and defaults to
// 1 when absent (a single-socket machine).
func parseLscpuCores(output string) (int, error) {
	coresPerSocket := 0
	foundCores := false
	sockets := 1 // default: single socket when lscpu omits the Socket(s) line

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.Contains(line, "Core(s) per socket:"):
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if v, err := strconv.Atoi(parts[3]); err == nil {
					coresPerSocket = v
					foundCores = true
				}
			}
		case strings.Contains(line, "Socket(s):"):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if v, err := strconv.Atoi(parts[len(parts)-1]); err == nil && v > 0 {
					sockets = v
				}
			}
		}
	}

	if !foundCores {
		return 0, fmt.Errorf("could not parse core count")
	}
	return coresPerSocket * sockets, nil
}

// parseSysctlLabeledInt parses a labeled BSD sysctl line of the form
// "<key>: <value>" (e.g. "hw.ncpu: 8") and returns the integer value.
// Behavior-preserving: matches the original inline strings.Split(":")[1]
// semantics exactly.
func parseSysctlLabeledInt(output string) (int, error) {
	parts := strings.Split(strings.TrimSpace(output), ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected sysctl format")
	}
	cores, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, err
	}
	return cores, nil
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

	// Check for Vulkan (cross-platform)
	if _, err := exec.LookPath("vulkaninfo"); err == nil {
		return true, "vulkan"
	}

	// Windows-specific GPU detection
	if runtime.GOOS == "windows" {
		// Check for DirectX/Vulkan capable GPUs via PowerShell
		cmd := exec.Command("powershell", "-Command",
			"Get-CimInstance -ClassName Win32_VideoController | Where-Object { $_.AdapterRAM -gt 0 } | Select-Object -First 1")
		if err := cmd.Run(); err == nil {
			// If we have a video controller, assume Vulkan support
			return true, "vulkan"
		}
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
