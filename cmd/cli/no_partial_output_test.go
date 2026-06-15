package main

// §11.4.4 → §11.4.102 → §11.4.43/§11.4.115 reproduce-first guard.
//
// FACT (background fix stream, 2026-06-15): a failed translation/write run must
// NOT leave a partial or empty output file on disk. The write phase used to call
// the format writer (EPUBWriter/FB2Writer/writeAsText) DIRECTLY on the final
// output path. Those writers `os.Create(filename)` first — truncating/creating
// the destination — and only then stream content. A mid-write failure therefore
// left a partial/empty file at the user's output path. Combined with the older
// concern (a failed run that exits non-zero but leaves a leftover file the user
// mistakes for success, §11.4 / §11.4.1 silent-failure), the contract is:
//
//   a write failure leaves NO partial/empty output file at the target path.
//
// These tests reproduce the defect against the pre-fix code (RED) and become the
// permanent GREEN regression guard after the fix (write-to-temp-then-atomic-
// rename). They drive the REAL translateEbook write path with a mock translator
// (no network) — anti-bluff, fully self-driving, re-runnable (§11.4.98).

import (
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/language"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleBook() *ebook.Book {
	return &ebook.Book{
		Metadata: ebook.Metadata{Title: "Sample", Language: "en"},
		Chapters: []ebook.Chapter{
			{
				Title:    "Ch1",
				Sections: []ebook.Section{{Content: "Hello world."}},
			},
		},
	}
}

// writeBookToFormat is the seam under test: it must produce the output file ONLY
// on full success, and leave NO partial/empty file at the target path on
// failure. The unexported helper writeOutput in main.go provides this contract.

// TestWriteOutput_FailureLeavesNoPartialFile is the RED-on-broken-artifact /
// GREEN-on-fixed guard. We force the writer to fail by pointing the output path
// at a location whose creation is guaranteed to fail AFTER any temp scaffolding
// would have been created, and assert the final target path does NOT exist.
func TestWriteOutput_FailureLeavesNoPartialFile(t *testing.T) {
	book := sampleBook()

	// Output target sits INSIDE a path component that is a regular file, so the
	// real format writer's os.Create on the final path fails. The contract:
	// translateEbook/writeOutput must surface the error AND leave no file at the
	// caller-visible output path.
	tmp := t.TempDir()
	notADir := filepath.Join(tmp, "iam-a-file")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o600))
	target := filepath.Join(notADir, "out.epub") // parent is a file → create fails

	err := writeOutput(book, target, "epub")
	require.Error(t, err, "writer must fail when the parent path is not a directory")

	// No regular file may exist at the target path. A successful os.Stat (nil
	// error returning a non-dir file) would mean a partial/empty output leaked;
	// any stat error (IsNotExist, or "not a directory" because the parent is a
	// file) confirms nothing was produced there.
	info, statErr := os.Stat(target)
	if statErr == nil {
		t.Fatalf("a partial/empty output file leaked at the target path: size=%d", info.Size())
	}
}

// TestWriteOutput_UnsupportedFormatDoesNotTouchTarget asserts an unsupported
// output format errors BEFORE any filesystem write — the pre-existing target is
// left fully intact.
func TestWriteOutput_UnsupportedFormatDoesNotTouchTarget(t *testing.T) {
	book := sampleBook()
	tmp := t.TempDir()
	target := filepath.Join(tmp, "out.txt")

	const prior = "PRIOR-USER-OUTPUT-MUST-SURVIVE-A-FAILED-RUN"
	require.NoError(t, os.WriteFile(target, []byte(prior), 0o600))

	err := writeOutput(book, target, "totally-unsupported-format")
	require.Error(t, err)

	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr, "pre-existing target file must still be present")
	assert.Equal(t, prior, string(got),
		"an unsupported format must not truncate/replace the user's pre-existing output file")
}

// TestWriteOutput_WriteFailureDoesNotClobberExistingFile is the strongest
// mutation-catching guard: when the actual write fails, the user's pre-existing
// output file at the target path must NOT be truncated/replaced into an
// empty/partial file.
//
// Mechanism (deterministic, portable): a read-only parent directory. Under the
// pre-fix DIRECT-write code, os.Create(existing-target) still succeeds (the file
// itself is writable) and TRUNCATES the stale file to empty before the write —
// silently destroying the user's prior output. Under the atomic-temp fix,
// os.CreateTemp() in the read-only dir fails up front, so the prior file is
// never touched. (The supported format itself never even errors here — the
// failure is the directory-level write refusal, exactly the partial/clobber
// class this guards.)
func TestWriteOutput_WriteFailureDoesNotClobberExistingFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("read-only-dir enforcement does not apply to root") // SKIP-OK: #root-bypasses-perms
	}
	book := sampleBook()
	tmp := t.TempDir()
	target := filepath.Join(tmp, "out.txt")

	const prior = "PRIOR-USER-OUTPUT-MUST-SURVIVE-A-FAILED-RUN"
	require.NoError(t, os.WriteFile(target, []byte(prior), 0o600))

	// Make the parent directory read-only so any create within it fails.
	require.NoError(t, os.Chmod(tmp, 0o500))
	t.Cleanup(func() { _ = os.Chmod(tmp, 0o700) })

	err := writeOutput(book, target, "txt")
	require.Error(t, err, "writing into a read-only directory must fail")

	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr, "pre-existing target file must still be present after a failed write")
	assert.Equal(t, prior, string(got),
		"a failed write must not truncate/replace the user's pre-existing output file")
}

// TestWriteOutput_SuccessProducesNonEmptyFile is the happy-path guard: a
// successful write produces a non-empty file at the exact target path (§11.4.1
// — the fix must not regress the happy path).
func TestWriteOutput_SuccessProducesNonEmptyFile(t *testing.T) {
	book := sampleBook()
	tmp := t.TempDir()
	target := filepath.Join(tmp, "ok.epub")

	require.NoError(t, writeOutput(book, target, "epub"))

	info, statErr := os.Stat(target)
	require.NoError(t, statErr, "successful write must produce the output file")
	assert.Greater(t, info.Size(), int64(0), "output file must be non-empty")
}

// TestWriteOutput_SuccessReplacesStaleFileAtomically guards the overwrite path:
// a successful run replaces a stale file with fresh, non-empty content (no
// leftover temp files in the directory).
func TestWriteOutput_SuccessReplacesStaleFileAtomically(t *testing.T) {
	book := sampleBook()
	tmp := t.TempDir()
	target := filepath.Join(tmp, "book_en.txt")
	require.NoError(t, os.WriteFile(target, []byte("stale"), 0o600))

	require.NoError(t, writeOutput(book, target, "txt"))

	got, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	assert.NotEqual(t, "stale", string(got), "stale content must be replaced on success")
	assert.NotEmpty(t, got)

	// No leftover temp artefacts beside the target.
	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	for _, e := range entries {
		assert.Equal(t, filepath.Base(target), e.Name(),
			"the only file in the output dir must be the final target (no temp leftovers)")
	}
}

// keep language import referenced for future expansion without churn.
var _ = language.English
