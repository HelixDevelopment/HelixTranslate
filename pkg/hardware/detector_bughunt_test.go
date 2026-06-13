package hardware

import "testing"

// Bug-hunt wave (pkg/hardware). Each test REPRODUCES a real defect on the
// current parser code (RED), and pins the corrected behavior (GREEN). Per
// §11.4.115 the test reproduces the defect first; the fix flips it green.
// Mutation-proof: reverting the source fix makes the corresponding test FAIL.

// Bug A — parseVMStatAvailableBytes: a PRESENT-but-malformed "page size of"
// header silently zeroes the page size (strconv.ParseUint error discarded,
// assigning 0), overwriting the documented default of 16384. Result: real
// free/inactive/speculative pages are reported as 0 bytes available RAM.
//
// FACT (captured on pre-fix code via /tmp/repro_pagesize.go):
//
//	malformed-header => 0   (350000 pages parsed; default 16384 -> 5734400000)
//
// The function's own doc states the page size "defaults to 16384 ... when the
// header is absent" — a malformed header must not be WORSE than an absent one.
func TestParseVMStatAvailableBytes_MalformedPageSizeKeepsDefault(t *testing.T) {
	// free=100000 + inactive=200000 + speculative=50000 = 350000 pages.
	in := "Mach Virtual Memory Statistics: (page size of unknown bytes)\n" +
		"Pages free:                              100000.\n" +
		"Pages inactive:                          200000.\n" +
		"Pages speculative:                        50000.\n"

	const wantDefault = uint64(350000) * 16384 // 5_734_400_000

	got := parseVMStatAvailableBytes(in)
	if got == 0 {
		t.Fatalf("malformed page-size header zeroed available RAM: got 0, "+
			"want default-page-size result %d (350000 pages * 16384)", wantDefault)
	}
	if got != wantDefault {
		t.Fatalf("parseVMStatAvailableBytes(malformed header)=%d, want %d "+
			"(must fall back to the documented 16384 default)", got, wantDefault)
	}
}

// Companion: an empty page-size value (header present, "of" then non-numeric/
// missing) must also retain the default rather than zeroing it.
func TestParseVMStatAvailableBytes_EmptyPageSizeValueKeepsDefault(t *testing.T) {
	in := "Mach Virtual Memory Statistics: (page size of  bytes)\n" +
		"Pages free:    10.\nPages inactive:    20.\nPages speculative:    30.\n"
	const want = uint64(60) * 16384
	if got := parseVMStatAvailableBytes(in); got != want {
		t.Fatalf("empty page-size value: got %d, want default-based %d", got, want)
	}
}

// Bug B — parseLinuxMeminfoKB: the kB->bytes conversion `ramKB * 1024` can
// silently overflow uint64 and wrap to an absurd small value with err==nil.
// A corrupt/garbage /proc/meminfo (or a malicious one over SSH) reporting a kB
// value above 2^64/1024 wraps to a tiny byte count, which downstream sizes the
// LLM far too small — a silent-corruption defect.
//
// FACT (captured on pre-fix code via /tmp/repro_overflow.go):
//
//	huge kB => bytes=1024 err=<nil>   (18014398509481985 kB wrapped to 1024)
//
// Correct behavior: detect the overflow and return an error rather than a
// plausible-looking wrapped value.
func TestParseLinuxMeminfoKB_OverflowReturnsError(t *testing.T) {
	// 2^64/1024 = 18014398509481984; one above it overflows on *1024.
	in := "MemTotal: 18014398509481985 kB"
	got, err := parseLinuxMeminfoKB(in)
	if err == nil {
		t.Fatalf("overflowing kB value silently wrapped to %d with no error; "+
			"want an error instead of a corrupt value", got)
	}
	// And valid values still convert correctly (no regression).
	if v, err := parseLinuxMeminfoKB("MemTotal: 16384000 kB"); err != nil || v != 16384000*1024 {
		t.Fatalf("valid conversion regressed: got %d,%v want %d,nil", v, err, uint64(16384000)*1024)
	}
}
