package hardware

import "testing"

// RED: On a multi-socket Windows host, PowerShell
// `(Get-CimInstance Win32_Processor).NumberOfCores` emits ONE integer line per
// physical processor (e.g. "8\r\n8\r\n" for a dual-socket box, 16 physical
// cores total). The getCPUCores Windows branch fed that multi-line output to
// parseSysctlInt, a BARE-single-integer parser, which errored — so Detect()
// silently fell back to runtime.NumCPU() (LOGICAL cores, e.g. 32 with HT),
// reporting the wrong physical core count used for performance tuning. This is
// the exact multi-socket undercounting class already fixed for Linux lscpu
// (parseLscpuCores sums Socket(s)); the Windows branch had no equivalent.
//
// parseWindowsCoreCount sums the per-socket integer lines into the TOTAL
// physical core count, mirroring CPUCores' documented "total physical cores"
// contract. Host-independent (pure string parse) per §11.4.81.
func TestParseWindowsCoreCount_MultiSocketSums(t *testing.T) {
	got, err := parseWindowsCoreCount("8\r\n8\r\n")
	if err != nil {
		t.Fatalf("parseWindowsCoreCount: %v", err)
	}
	if got != 16 {
		t.Errorf("dual-socket 8+8 cores: got %d, want 16 (multi-socket undercount)", got)
	}
}

// RED: single-socket output (the common case) must still parse to that count.
func TestParseWindowsCoreCount_SingleSocket(t *testing.T) {
	got, err := parseWindowsCoreCount("12\r\n")
	if err != nil {
		t.Fatalf("parseWindowsCoreCount: %v", err)
	}
	if got != 12 {
		t.Errorf("single-socket 12 cores: got %d, want 12", got)
	}
}

// RED: genuinely unparseable output must surface an error (so Detect() can fall
// back), NOT silently return 0 cores.
func TestParseWindowsCoreCount_GarbageErrors(t *testing.T) {
	if _, err := parseWindowsCoreCount("\r\n\r\n"); err == nil {
		t.Errorf("expected error for output with no integer core lines")
	}
}
