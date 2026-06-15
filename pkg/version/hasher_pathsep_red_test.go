package version

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// RED: The codebase hash is documented to be "consistent across systems"
// (hasher.go: "Removed timestamp to ensure consistent hashes across systems")
// because distributed worker version-sync compares the hash of the SAME source
// tree checked out on different OSes (a macOS coordinator vs a Linux/Windows
// worker). addFileToHash records the walked path verbatim with
// fmt.Fprintf(hasher, "file:%s\n", path). filepath.Walk yields '/'-separated
// paths on unix but '\'-separated paths on Windows, so the SAME directory tree
// hashes DIFFERENTLY on Windows than on unix → spurious version mismatch →
// version_manager triggers a needless rollback/update of a worker running
// identical code.
//
// The fix records a separator-NORMALIZED path. This test asserts the bytes the
// hasher records for a path are separator-independent. Pre-fix it FAILS because
// the raw '\' bytes are written.
func TestHash_RecordedPathSeparatorNormalized(t *testing.T) {
	h := NewCodebaseHasher()

	unixBuf := &bytes.Buffer{}
	winBuf := &bytes.Buffer{}

	// recordedPathLine is the canonical seam the fix introduces: the exact
	// "file:<path>" line written into the digest for a given walked path.
	h.recordPathLine(unixBuf, "pkg/version/real.go")
	h.recordPathLine(winBuf, `pkg\version\real.go`)

	if !bytes.Equal(unixBuf.Bytes(), winBuf.Bytes()) {
		t.Errorf("recorded path line is OS-separator-dependent:\n unix=%q\n win =%q\n → identical source tree hashes differently across OSes (false version drift)",
			unixBuf.String(), winBuf.String())
	}

	// Sanity: the recorded form must be the slash form (not lost entirely).
	if !strings.Contains(unixBuf.String(), filepath.ToSlash("pkg/version/real.go")) {
		t.Errorf("recorded path no longer contains the logical path: %q", unixBuf.String())
	}
}
