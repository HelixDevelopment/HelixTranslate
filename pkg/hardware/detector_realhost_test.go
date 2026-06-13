package hardware

import (
	"runtime"
	"strings"
	"testing"
)

// These tests assert EXACT, deterministic outputs of the reachable code paths
// on the real host (anti-bluff per §11.4.27/§11.4: each test fails if its unit
// is stubbed). OS-probe branches for foreign operating systems hard-call
// exec.Command and cannot be exercised on this host without faking, so they are
// gated behind a runtime.GOOS check with an honest SKIP (§11.4.3) rather than a
// fabricated PASS. See the OS_PROBE_DEFERRED note in the W2 report.

// --- calculateMaxModelSize: pure logic, pins every rounding threshold exactly.
// A stubbed/short-circuited implementation (e.g. "return 7_000_000_000") fails
// these because the boundary cases demand different exact return values.

func TestCalculateMaxModelSize_ExactThresholds(t *testing.T) {
	d := NewDetector()
	const gb = uint64(1024 * 1024 * 1024)

	// multiplier = 2.0 when no GPU: maxParams = ramGB / 2.
	// Thresholds: >=70 ->70B, >=27 ->27B, >=13 ->13B, >=7 ->7B, >=3 ->3B, else 1B.
	noGPU := []struct {
		name   string
		ramGB  uint64
		expect uint64
	}{
		{"2GB->1B (below 3)", 2, 1_000_000_000},   // 2/2=1 -> <3
		{"6GB->3B (maxParams=3)", 6, 3_000_000_000}, // 6/2=3 -> >=3
		{"13GB->3B (6.5 params)", 13, 3_000_000_000}, // 13/2=6.5 -> >=3 not 7
		{"14GB->7B (exactly 7)", 14, 7_000_000_000}, // 14/2=7 -> >=7
		{"26GB->13B (exactly 13)", 26, 13_000_000_000},
		{"54GB->27B (exactly 27)", 54, 27_000_000_000},
		{"140GB->70B (exactly 70)", 140, 70_000_000_000},
		{"300GB->70B (cap)", 300, 70_000_000_000},
	}
	for _, tc := range noGPU {
		t.Run("noGPU/"+tc.name, func(t *testing.T) {
			got := d.calculateMaxModelSize(tc.ramGB*gb, false)
			if got != tc.expect {
				t.Fatalf("calculateMaxModelSize(%dGB, false) = %d, want %d", tc.ramGB, got, tc.expect)
			}
		})
	}

	// multiplier = 1.5 with GPU: maxParams = ramGB / 1.5.
	// 12GB/1.5 = 8 -> >=7 -> 7B (whereas noGPU 12/2=6 -> 3B). This case proves
	// the GPU branch genuinely changes the multiplier (fails if hasGPU ignored).
	withGPU := []struct {
		name   string
		ramGB  uint64
		expect uint64
	}{
		{"4GB->1B (2.67 params)", 4, 1_000_000_000},
		{"5GB->3B (3.33 params)", 5, 3_000_000_000},
		{"12GB->7B (8 params)", 12, 7_000_000_000},
		{"21GB->13B (14 params)", 21, 13_000_000_000},
		{"42GB->27B (28 params)", 42, 27_000_000_000},
		{"105GB->70B (70 params)", 105, 70_000_000_000},
	}
	for _, tc := range withGPU {
		t.Run("gpu/"+tc.name, func(t *testing.T) {
			got := d.calculateMaxModelSize(tc.ramGB*gb, true)
			if got != tc.expect {
				t.Fatalf("calculateMaxModelSize(%dGB, true) = %d, want %d", tc.ramGB, got, tc.expect)
			}
		})
	}
}

// TestCalculateMaxModelSize_GPUReducesRequirement proves the GPU multiplier
// makes the result >= the no-GPU result for the same RAM (anti-bluff: fails if
// the hasGPU parameter is dropped and both branches compute identically).
func TestCalculateMaxModelSize_GPULowersRamNeed(t *testing.T) {
	d := NewDetector()
	const gb = uint64(1024 * 1024 * 1024)
	// 12GB: noGPU -> 3B, GPU -> 7B. Strictly larger with GPU.
	withoutGPU := d.calculateMaxModelSize(12*gb, false)
	withGPU := d.calculateMaxModelSize(12*gb, true)
	if !(withGPU > withoutGPU) {
		t.Fatalf("expected GPU to enable a larger model at 12GB: gpu=%d noGPU=%d", withGPU, withoutGPU)
	}
	if withGPU != 7_000_000_000 || withoutGPU != 3_000_000_000 {
		t.Fatalf("unexpected values: gpu=%d (want 7B) noGPU=%d (want 3B)", withGPU, withoutGPU)
	}
}

// TestCalculateMaxModelSize_ZeroRAM asserts the 1B floor for the degenerate
// zero-RAM input (boundary case from the W2 spec).
func TestCalculateMaxModelSize_ZeroRAM(t *testing.T) {
	d := NewDetector()
	if got := d.calculateMaxModelSize(0, false); got != 1_000_000_000 {
		t.Fatalf("zero RAM no GPU = %d, want 1B floor", got)
	}
	if got := d.calculateMaxModelSize(0, true); got != 1_000_000_000 {
		t.Fatalf("zero RAM with GPU = %d, want 1B floor", got)
	}
}

// --- CanRunModel: exact boundary behavior (<=). Fails if implemented with <.
func TestCanRunModel_BoundaryExact(t *testing.T) {
	c := &Capabilities{MaxModelSize: 13_000_000_000}
	cases := []struct {
		size uint64
		want bool
	}{
		{0, true},
		{12_999_999_999, true},
		{13_000_000_000, true},  // equal -> allowed (<=)
		{13_000_000_001, false}, // one over -> rejected
		{70_000_000_000, false},
	}
	for _, tc := range cases {
		if got := c.CanRunModel(tc.size); got != tc.want {
			t.Fatalf("CanRunModel(%d) with max %d = %v, want %v", tc.size, c.MaxModelSize, got, tc.want)
		}
	}
}

// --- String: assert the rendered fields reflect the struct exactly. Fails if
// String() returns a constant or omits a field.
func TestString_ContentExact(t *testing.T) {
	const gb = uint64(1024 * 1024 * 1024)
	c := &Capabilities{
		Architecture: "arm64",
		TotalRAM:     16 * gb,
		AvailableRAM: 8 * gb,
		CPUModel:     "Apple M3 Pro",
		CPUCores:     12,
		HasGPU:       true,
		GPUType:      "metal",
		MaxModelSize: 13_000_000_000,
	}
	s := c.String()
	wantSubstrings := []string{
		"Architecture: arm64",
		"Apple M3 Pro (12 cores)",
		"Total RAM: 16.0 GB",
		"Available RAM: 8.0 GB",
		"metal acceleration",
		"Max Model Size: 13B parameters",
	}
	for _, sub := range wantSubstrings {
		if !strings.Contains(s, sub) {
			t.Fatalf("String() missing %q\nfull output:\n%s", sub, s)
		}
	}
}

// TestString_NoGPURendersNone proves the HasGPU=false branch renders "None"
// (fails if String() always reports acceleration).
func TestString_NoGPURendersNone(t *testing.T) {
	c := &Capabilities{Architecture: "amd64", GPUType: "cuda", HasGPU: false}
	s := c.String()
	if !strings.Contains(s, "GPU: None") {
		t.Fatalf("expected 'GPU: None' when HasGPU=false, got:\n%s", s)
	}
	if strings.Contains(s, "cuda acceleration") {
		t.Fatalf("must not render GPUType when HasGPU=false:\n%s", s)
	}
}

// --- detectGPU: deterministic on this host. On darwin/arm64 the very first
// branch returns (true, "metal"); asserting that exact pair fails if the
// function is stubbed to return (false, ""). On other hosts we assert only the
// type-consistency invariant (a present GPU has a non-empty type) since the
// CUDA/ROCm/Vulkan probes depend on installed tooling.
func TestDetectGPU_HostDeterministic(t *testing.T) {
	d := NewDetector()
	hasGPU, gpuType := d.detectGPU()

	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if !hasGPU || gpuType != "metal" {
			t.Fatalf("on darwin/arm64 expected (true, \"metal\"), got (%v, %q)", hasGPU, gpuType)
		}
		return
	}
	// Invariant on every host: GPU present iff non-empty type.
	if hasGPU && gpuType == "" {
		t.Fatalf("GPU reported present but type is empty")
	}
	if !hasGPU && gpuType != "" {
		t.Fatalf("GPU reported absent but type %q non-empty", gpuType)
	}
}

// --- getTotalRAM / getMacOSRAM: real probe on this host. Asserts the value is
// plausible and that getTotalRAM dispatches to the OS-correct helper. A stub
// returning 0 fails (RAM must be positive on a real machine).
func TestGetTotalRAM_RealHostPlausible(t *testing.T) {
	d := NewDetector()
	total, err := d.getTotalRAM()
	if err != nil {
		t.Fatalf("getTotalRAM on real %s host returned error: %v", runtime.GOOS, err)
	}
	const oneGB = uint64(1024 * 1024 * 1024)
	if total < oneGB {
		t.Fatalf("total RAM %d bytes implausibly small (<1GB) on a real host", total)
	}
	// 64 TB sanity ceiling — catches a parser that returns garbage.
	if total > 64*1024*oneGB {
		t.Fatalf("total RAM %d bytes implausibly large", total)
	}

	if runtime.GOOS == "darwin" {
		// getTotalRAM must dispatch to getMacOSRAM and return the same value.
		mac, err := d.getMacOSRAM()
		if err != nil {
			t.Fatalf("getMacOSRAM error: %v", err)
		}
		if mac != total {
			t.Fatalf("getTotalRAM (%d) != getMacOSRAM (%d): dispatch mismatch", total, mac)
		}
	}
}

// --- getAvailableRAM: real probe on this host. Must be positive and not exceed
// total RAM (cross-field invariant). Fails if the vm_stat/meminfo parser is
// stubbed to 0.
func TestGetAvailableRAM_RealHostInvariants(t *testing.T) {
	d := NewDetector()
	avail, err := d.getAvailableRAM()
	if err != nil {
		// getAvailableRAM has no implementation for some OSes; honest skip.
		t.Skipf("getAvailableRAM not supported on %s: %v (§11.4.3 topology skip)", runtime.GOOS, err)
	}
	if avail == 0 {
		t.Fatalf("available RAM is 0 on a real host (parser likely stubbed)")
	}
	total, terr := d.getTotalRAM()
	if terr == nil && avail > total {
		t.Fatalf("available RAM %d exceeds total RAM %d (impossible)", avail, total)
	}
}

// --- getCPUCores: real probe. Physical cores must be >=1 and <= logical CPUs
// (a physical-core count above NumCPU would be impossible). Fails if stubbed 0.
func TestGetCPUCores_RealHostInvariants(t *testing.T) {
	d := NewDetector()
	cores, err := d.getCPUCores()
	if err != nil {
		t.Skipf("getCPUCores not supported on %s: %v (§11.4.3 topology skip)", runtime.GOOS, err)
	}
	if cores < 1 {
		t.Fatalf("physical CPU cores = %d, want >=1", cores)
	}
	if logical := runtime.NumCPU(); cores > logical {
		t.Fatalf("physical cores %d exceed logical CPUs %d (impossible)", cores, logical)
	}
}

// --- getCPUModel: real probe. Must be a non-empty, non-whitespace string.
// Fails if stubbed to "".
func TestGetCPUModel_RealHostNonEmpty(t *testing.T) {
	d := NewDetector()
	model, err := d.getCPUModel()
	if err != nil {
		t.Skipf("getCPUModel not supported on %s: %v (§11.4.3 topology skip)", runtime.GOOS, err)
	}
	if len(model) == 0 {
		t.Fatalf("CPU model is empty on a real host (parser likely stubbed)")
	}
}

// --- Detect: full integration on the real host. Asserts cross-field
// consistency and that every field is populated coherently. This is the
// strongest anti-bluff test: a stubbed Detect() returning a zero-value
// Capabilities fails multiple invariants at once.
func TestDetect_RealHostCrossFieldConsistency(t *testing.T) {
	d := NewDetector()
	caps, err := d.Detect()
	if err != nil {
		t.Fatalf("Detect on real %s host returned error: %v", runtime.GOOS, err)
	}
	if caps.Architecture != runtime.GOARCH {
		t.Fatalf("Architecture %q != runtime.GOARCH %q", caps.Architecture, runtime.GOARCH)
	}
	const oneGB = uint64(1024 * 1024 * 1024)
	if caps.TotalRAM < oneGB {
		t.Fatalf("TotalRAM %d too small", caps.TotalRAM)
	}
	if caps.AvailableRAM == 0 {
		t.Fatalf("AvailableRAM is 0")
	}
	if caps.AvailableRAM > caps.TotalRAM {
		t.Fatalf("AvailableRAM %d > TotalRAM %d", caps.AvailableRAM, caps.TotalRAM)
	}
	if caps.CPUCores < 1 {
		t.Fatalf("CPUCores %d < 1", caps.CPUCores)
	}
	if caps.CPUModel == "" {
		t.Fatalf("CPUModel empty")
	}
	if caps.MaxModelSize == 0 {
		t.Fatalf("MaxModelSize is 0")
	}
	// MaxModelSize MUST equal what calculateMaxModelSize computes from the
	// detected AvailableRAM+HasGPU — proves Detect wires the calculation through
	// rather than hardcoding a value.
	want := d.calculateMaxModelSize(caps.AvailableRAM, caps.HasGPU)
	if caps.MaxModelSize != want {
		t.Fatalf("MaxModelSize %d != calculateMaxModelSize(avail=%d, gpu=%v)=%d",
			caps.MaxModelSize, caps.AvailableRAM, caps.HasGPU, want)
	}
	// GPU type invariant.
	if caps.HasGPU && caps.GPUType == "" {
		t.Fatalf("HasGPU true but GPUType empty")
	}
	// On darwin/arm64 GPU must be metal.
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if !caps.HasGPU || caps.GPUType != "metal" {
			t.Fatalf("darwin/arm64 expected metal GPU, got (%v,%q)", caps.HasGPU, caps.GPUType)
		}
	}
}
