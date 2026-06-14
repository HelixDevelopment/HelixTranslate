package hardware

import "testing"

// These tests exercise the PURE parsers extracted from the OS-specific
// exec.Command callers in detector.go (D9 refactor). The parsing logic for the
// Linux (/proc/meminfo, lscpu, /proc/cpuinfo), Windows (wmic, PowerShell), BSD
// (sysctl), and macOS (vm_stat) branches previously lived inline beside a
// hard-coded exec.Command, so on a non-matching host it was unreachable without
// faking (forbidden §11.4.27). Relocating it into pure functions that take the
// command's raw output lets us drive it deterministically on any host with REAL
// captured fixture outputs in the documented formats.
//
// Anti-bluff (§11.4 / §11.4.27): every case asserts an EXACT parsed value, so a
// stubbed/short-circuited parser (e.g. `return 0, nil`) FAILs. One worked
// mutation is documented in TestParseLinuxMeminfoKB_StubbedNegation.

// --- parseSysctlBytes: `sysctl -n hw.memsize` emits a bare byte count.
// Real format (macOS): a single integer with a trailing newline.
func TestParseSysctlBytes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    uint64
		wantErr bool
	}{
		// 16 GiB Apple Silicon — real `sysctl -n hw.memsize` output.
		{"16GiB", "17179869184\n", 17179869184, false},
		{"no trailing newline", "8589934592", 8589934592, false}, // 8 GiB
		{"leading/trailing spaces", "  4294967296  \n", 4294967296, false},
		{"empty", "", 0, true},
		{"non-numeric", "garbage", 0, true},
		{"negative-ish", "-1", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSysctlBytes(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseSysctlBytes(%q) err=%v wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseSysctlBytes(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseLinuxMeminfoKB: a single /proc/meminfo line "MemTotal: <kB> kB".
// Format documented in the Linux kernel `proc(5)` man page; field [1] is the
// kB count, converted to bytes (*1024).
func TestParseLinuxMeminfoKB(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    uint64
		wantErr bool
	}{
		// Real `grep MemTotal /proc/meminfo` output for a 16 GB box.
		{"MemTotal 16GB", "MemTotal:       16384000 kB\n", 16384000 * 1024, false},
		// Real `grep MemAvailable /proc/meminfo` output.
		{"MemAvailable", "MemAvailable:    8192000 kB\n", 8192000 * 1024, false},
		{"tab separated", "MemTotal:\t32768000\tkB", 32768000 * 1024, false},
		{"only key, no value", "MemTotal:", 0, true},
		{"empty", "", 0, true},
		{"non-numeric value", "MemTotal: abc kB", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLinuxMeminfoKB(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseLinuxMeminfoKB(%q) err=%v wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseLinuxMeminfoKB(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseLinuxMeminfoKB_StubbedNegation is the documented anti-bluff mutation
// (§11.4 / §11.4.27). It pins the EXACT byte conversion (kB*1024). If a future
// edit stubs parseLinuxMeminfoKB to `return 0, nil` or drops the *1024
// conversion, this assertion FAILs — proving the test exercises the real
// parsing, not a tautology.
func TestParseLinuxMeminfoKB_StubbedNegation(t *testing.T) {
	got, err := parseLinuxMeminfoKB("MemTotal:       16384000 kB")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	const want = uint64(16384000) * 1024 // = 16777216000 bytes
	if got != want {
		t.Fatalf("got %d, want %d — a stub or a missing *1024 would land here", got, want)
	}
	// Cross-check: the result must NOT equal the raw kB (would mean *1024 dropped).
	if got == 16384000 {
		t.Fatalf("parser returned raw kB %d — *1024 conversion was lost", got)
	}
}

// --- parseWmicMem: `wmic computersystem get totalphysicalmemory` emits a
// header line then the byte count on line 2. Real Windows wmic output uses
// CRLF; the second field after Split("\n") carries a trailing '\r' that
// TrimSpace removes.
func TestParseWmicMem(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    uint64
		wantErr bool
	}{
		// Real wmic output (CRLF line endings preserved as in the live command).
		{"crlf 16GB", "TotalPhysicalMemory\r\n17179869184\r\n", 17179869184, false},
		{"lf only", "TotalPhysicalMemory\n8589934592\n", 8589934592, false},
		{"header only", "TotalPhysicalMemory\n", 0, true},                  // Split -> ["...",""], line[1]="" -> ParseUint fails
		{"truly single line (no newline)", "TotalPhysicalMemory", 0, true}, // Split -> ["..."], len<2
		{"empty", "", 0, true},
		{"non-numeric body", "TotalPhysicalMemory\nNaN\n", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseWmicMem(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseWmicMem(%q) err=%v wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseWmicMem(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// --- parsePowerShellBytes: PowerShell expression emitting a bare byte count.
func TestParsePowerShellBytes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    uint64
		wantErr bool
	}{
		// (Get-CimInstance Win32_OperatingSystem).FreePhysicalMemory * 1024
		// FreePhysicalMemory is in kB; the *1024 makes it bytes. PowerShell
		// prints with a trailing CRLF.
		{"4GB free", "4194304000\r\n", 4194304000, false},
		{"lf", "2097152000\n", 2097152000, false},
		{"empty", "", 0, true},
		{"text", "Access Denied", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePowerShellBytes(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parsePowerShellBytes(%q) err=%v wantErr=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parsePowerShellBytes(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseVMStatAvailableBytes: macOS `vm_stat` output. Free + inactive +
// speculative pages, times page size declared in the header. Real vm_stat
// output (Apple Silicon, 16384-byte pages). Returns 0 on a body with no
// recognizable lines (matches original inline behavior — no error path).
func TestParseVMStatAvailableBytes(t *testing.T) {
	// Real `vm_stat` output (abridged but format-faithful). Apple Silicon
	// declares "page size of 16384 bytes". free=100000, inactive=200000,
	// speculative=50000 -> 350000 pages * 16384 = 5_734_400_000 bytes.
	real := "Mach Virtual Memory Statistics: (page size of 16384 bytes)\n" +
		"Pages free:                              100000.\n" +
		"Pages active:                            900000.\n" +
		"Pages inactive:                          200000.\n" +
		"Pages speculative:                        50000.\n" +
		"Pages wired down:                        300000.\n"
	const wantReal = uint64(350000) * 16384
	if got := parseVMStatAvailableBytes(real); got != wantReal {
		t.Errorf("parseVMStatAvailableBytes(real)=%d want %d", got, wantReal)
	}

	// Header absent -> default page size 16384 used (matches original).
	noHeader := "Pages free:    10.\nPages inactive:    20.\nPages speculative:    30.\n"
	const wantNoHeader = uint64(60) * 16384
	if got := parseVMStatAvailableBytes(noHeader); got != wantNoHeader {
		t.Errorf("parseVMStatAvailableBytes(noHeader)=%d want %d", got, wantNoHeader)
	}

	// Intel Mac declares a 4096-byte page size.
	intel := "Mach Virtual Memory Statistics: (page size of 4096 bytes)\n" +
		"Pages free:    1000.\nPages inactive:    2000.\nPages speculative:    500.\n"
	const wantIntel = uint64(3500) * 4096
	if got := parseVMStatAvailableBytes(intel); got != wantIntel {
		t.Errorf("parseVMStatAvailableBytes(intel)=%d want %d", got, wantIntel)
	}

	// Empty / unrecognized -> 0 pages * default page size = 0.
	if got := parseVMStatAvailableBytes(""); got != 0 {
		t.Errorf("parseVMStatAvailableBytes(\"\")=%d want 0", got)
	}
}

// --- parseSysctlLabeledAvailableBytes: BSD `sysctl hw.usermem` ->
// ~70% of total. Real format: "hw.usermem: 12345678".
func TestParseSysctlLabeledAvailableBytes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    uint64
		wantErr bool
	}{
		// 10_000_000_000 total -> 0.7 * = 7_000_000_000 available.
		{"10GB usermem", "hw.usermem: 10000000000", 7_000_000_000, false},
		{"with trailing newline", "hw.usermem: 1000000\n", uint64(float64(1000000) * 0.7), false},
		{"no colon", "hw.usermem 123", 0, true},
		{"non-numeric", "hw.usermem: lots", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSysctlLabeledAvailableBytes(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseSysctlLabeledAvailableBytes(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseLinuxCPUModel: `grep -m1 "model name" /proc/cpuinfo`.
// Real format: "model name      : Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz".
func TestParseLinuxCPUModel(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"intel", "model name      : Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz\n",
			"Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz", false},
		{"amd", "model name\t: AMD Ryzen 9 5950X 16-Core Processor",
			"AMD Ryzen 9 5950X 16-Core Processor", false},
		{"no colon", "model name Intel", "", true},
		{"empty", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLinuxCPUModel(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseLinuxCPUModel(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseSysctlLabeledString: BSD `sysctl hw.model`.
// Real format: "hw.model: Intel(R) Core(TM) i7-8700K".
func TestParseSysctlLabeledString(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"intel", "hw.model: Intel(R) Core(TM) i7-8700K", "Intel(R) Core(TM) i7-8700K", false},
		{"trailing newline", "hw.model: AMD EPYC 7551\n", "AMD EPYC 7551", false},
		{"no colon", "hw.model AMD", "", true},
		{"empty", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSysctlLabeledString(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseSysctlLabeledString(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseSysctlInt: bare integer (macOS hw.physicalcpu / Windows NumberOfCores).
func TestParseSysctlInt(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"12 cores", "12\n", 12, false},
		{"8 cores no newline", "8", 8, false},
		{"spaces", "  4  \n", 4, false},
		{"empty", "", 0, true},
		{"non-numeric", "many", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSysctlInt(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseSysctlInt(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseLscpuCores: scans `lscpu` for "Core(s) per socket:".
// Real `lscpu` output, field [3] of that line is the count.
func TestParseLscpuCores(t *testing.T) {
	// Real (abridged) lscpu output.
	real := "Architecture:            x86_64\n" +
		"CPU(s):                  16\n" +
		"Thread(s) per core:      2\n" +
		"Core(s) per socket:      8\n" +
		"Socket(s):               1\n"
	if got, err := parseLscpuCores(real); err != nil || got != 8 {
		t.Errorf("parseLscpuCores(real)=%d,%v want 8,nil", got, err)
	}

	// Single socket, single core.
	one := "Core(s) per socket:      1\n"
	if got, err := parseLscpuCores(one); err != nil || got != 1 {
		t.Errorf("parseLscpuCores(one)=%d,%v want 1,nil", got, err)
	}

	// No "Core(s) per socket:" line -> error (matches inline behavior).
	if _, err := parseLscpuCores("CPU(s):  16\n"); err == nil {
		t.Errorf("parseLscpuCores(no-core-line) expected error, got nil")
	}

	// Empty.
	if _, err := parseLscpuCores(""); err == nil {
		t.Errorf("parseLscpuCores(\"\") expected error, got nil")
	}
}

// TestParseLscpuCores_MultiSocket is the regression guard for the multi-socket
// physical-core undercount bug. CPUCores is documented as "physical cores"
// (detector.go), but on a dual-socket server the physical core count is
// (Core(s) per socket) * (Socket(s)). A real dual-socket Xeon reporting
// 8 cores/socket across 2 sockets has 16 physical cores; the parser must NOT
// report 8 (cores-per-socket only).
//
// Mutation proof (§1.1): reverting parseLscpuCores to return cores-per-socket
// without multiplying by Socket(s) makes every multi-socket case below FAIL.
func TestParseLscpuCores_MultiSocket(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{
			name: "dual socket 8 cores each = 16",
			in: "Architecture:            x86_64\n" +
				"CPU(s):                  64\n" +
				"Thread(s) per core:      2\n" +
				"Core(s) per socket:      8\n" +
				"Socket(s):               2\n",
			want: 16,
		},
		{
			name: "quad socket 18 cores each = 72",
			in: "Core(s) per socket:      18\n" +
				"Socket(s):               4\n",
			want: 72,
		},
		{
			name: "single socket still correct = 8",
			in: "Core(s) per socket:      8\n" +
				"Socket(s):               1\n",
			want: 8,
		},
		{
			name: "missing Socket(s) line defaults to 1 socket = 6",
			in:   "Core(s) per socket:      6\n",
			want: 6,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLscpuCores(tc.in)
			if err != nil {
				t.Fatalf("parseLscpuCores(%q) unexpected error: %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("parseLscpuCores(%s) = %d physical cores, want %d", tc.name, got, tc.want)
			}
		})
	}
}

// --- parseSysctlLabeledInt: BSD `sysctl hw.ncpu` -> "hw.ncpu: 8".
func TestParseSysctlLabeledInt(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    int
		wantErr bool
	}{
		{"8 cpus", "hw.ncpu: 8", 8, false},
		{"trailing newline", "hw.ncpu: 16\n", 16, false},
		{"no colon", "hw.ncpu 8", 0, true},
		{"non-numeric", "hw.ncpu: eight", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSysctlLabeledInt(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseSysctlLabeledInt(%q)=%d want %d", tc.in, got, tc.want)
			}
		})
	}
}

// --- parseSysctlLabeledUint: the shared labeled-uint helper.
func TestParseSysctlLabeledUint(t *testing.T) {
	if got, err := parseSysctlLabeledUint("hw.usermem: 12345678"); err != nil || got != 12345678 {
		t.Errorf("parseSysctlLabeledUint=%d,%v want 12345678,nil", got, err)
	}
	if _, err := parseSysctlLabeledUint("nocolon"); err == nil {
		t.Errorf("expected error for missing colon")
	}
}
