package version

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

// RED: exclusion must match path COMPONENTS / glob patterns, not arbitrary
// substrings. A directory whose name merely CONTAINS an exclude token (e.g.
// "vendored" contains "vendor") must NOT be skipped. The pre-fix code used
// strings.Contains(path, "vendor") and silently dropped the directory's source
// files from the codebase hash, corrupting distributed worker version-sync
// (two different codebases can hash identical / identical codebases can differ).
func TestExclude_SubstringDirNotOverExcluded(t *testing.T) {
	tempDir := t.TempDir()
	legitDir := filepath.Join(tempDir, "src", "vendored") // contains "vendor"
	if err := os.MkdirAll(legitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legitDir, "real.go"), []byte("package vendored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewCodebaseHasher()
	withFile := sha256.New()
	if err := h.processDirectory(withFile, filepath.Join(tempDir, "src")); err != nil {
		t.Fatalf("processDirectory: %v", err)
	}

	emptyDir := filepath.Join(tempDir, "src_empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	empty := sha256.New()
	if err := h.processDirectory(empty, emptyDir); err != nil {
		t.Fatal(err)
	}

	if string(withFile.Sum(nil)) == string(empty.Sum(nil)) {
		t.Errorf("file in 'vendored' dir was wrongly excluded (substring match on 'vendor')")
	}
}

// RED: a real directory named exactly "vendor" (a path component) MUST still be
// excluded — the fix must not regress legitimate exclusions.
func TestExclude_RealVendorDirStillExcluded(t *testing.T) {
	tempDir := t.TempDir()
	vendorDir := filepath.Join(tempDir, "src", "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "dep.go"), []byte("package dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewCodebaseHasher()
	withVendor := sha256.New()
	if err := h.processDirectory(withVendor, filepath.Join(tempDir, "src")); err != nil {
		t.Fatalf("processDirectory: %v", err)
	}

	emptyDir := filepath.Join(tempDir, "src_empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	empty := sha256.New()
	if err := h.processDirectory(empty, emptyDir); err != nil {
		t.Fatal(err)
	}

	if string(withVendor.Sum(nil)) != string(empty.Sum(nil)) {
		t.Errorf("real 'vendor' dir was NOT excluded (regression of legitimate exclusion)")
	}
}

// RED: a relevant-extension file whose NAME contains an exclude token as a
// substring (e.g. "prod.env.json" contains ".env") must NOT be excluded.
func TestExclude_RelevantFileWithSubstringTokenIncluded(t *testing.T) {
	h := NewCodebaseHasher()
	if !h.shouldIncludeFile("config/prod.env.json") {
		t.Errorf("prod.env.json wrongly excluded via .env substring match")
	}
}

// RED: a real ".env" file (full base name) MUST still be excluded.
func TestExclude_RealEnvFileStillExcluded(t *testing.T) {
	h := NewCodebaseHasher()
	// .env has no relevant extension anyway, but make the intent explicit with a
	// configuration that would otherwise include it.
	h.RelevantExtensions = []string{".env"}
	if h.shouldIncludeFile("config/.env") {
		t.Errorf("real .env file was NOT excluded")
	}
}

// RED: glob exclude patterns must actually work. With strings.Contains the '*'
// was literal so "*.md" never matched "notes.md". After the fix, a glob exclude
// against the base name must filter the file out.
func TestExclude_GlobPatternLive(t *testing.T) {
	h := NewCodebaseHasher()
	h.RelevantExtensions = []string{".md"}
	h.ExcludePatterns = []string{"*.md"}
	if h.shouldIncludeFile("docs/notes.md") {
		t.Errorf("glob exclude '*.md' did not match notes.md (dead glob pattern)")
	}
}

// RED: the default coverage glob must exclude a coverage artifact. "coverage*.out"
// should match "coverage123.out" via glob (impossible with literal Contains).
func TestExclude_CoverageGlobLive(t *testing.T) {
	h := NewCodebaseHasher()
	h.RelevantExtensions = []string{".out"}
	// default ExcludePatterns contains "coverage*.out"
	if h.shouldIncludeFile("coverage123.out") {
		t.Errorf("coverage*.out glob did not exclude coverage123.out")
	}
}
