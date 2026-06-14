package version

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot walks up from the package directory to the module root (where the
// VERSION file and go.mod live).
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate module root (go.mod) from %s", dir)
	return ""
}

func authoritativeVersion(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	return strings.TrimSpace(string(b))
}

// TestAppVersionMatchesVERSIONFile is the mechanical binding: the in-code
// single-source constant MUST equal the authoritative VERSION file.
func TestAppVersionMatchesVERSIONFile(t *testing.T) {
	want := authoritativeVersion(t)
	if AppVersion != want {
		t.Fatalf("version.AppVersion = %q, want VERSION file value %q", AppVersion, want)
	}
}

// TestNoBinaryDeclaresDivergentVersionLiteral asserts that no cmd/*/main.go
// declares its own hardcoded version literal (appVersion / version = "x.y.z").
// Every binary MUST report version.AppVersion so all binaries agree with the
// authoritative VERSION file. This is RED while any binary hardcodes a literal
// (e.g. 3.0.0 / 2.1.0 / 2.0.0 / 1.0.0 != 2.3.0) and GREEN once single-sourced.
func TestNoBinaryDeclaresDivergentVersionLiteral(t *testing.T) {
	root := repoRoot(t)
	cmdDir := filepath.Join(root, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}

	// Matches a Go declaration assigning a semver-looking string literal to an
	// identifier named (app)version, e.g. `appVersion = "3.0.0"` or
	// `const version = "2.0.0"`. Does NOT match XML/EPUB attrs like
	// `version="1.0"` (no surrounding ` = ` Go-assignment spacing + ident).
	re := regexp.MustCompile(`(?m)\b(?:app[Vv]ersion|version)\s*=\s*"\d+\.\d+\.\d+"`)

	var offenders []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mainPath := filepath.Join(cmdDir, e.Name(), "main.go")
		b, err := os.ReadFile(mainPath)
		if err != nil {
			continue // not every cmd dir has main.go
		}
		for _, m := range re.FindAllString(string(b), -1) {
			offenders = append(offenders, e.Name()+"/main.go: "+strings.TrimSpace(m))
		}
	}

	if len(offenders) > 0 {
		t.Fatalf("binaries declare divergent hardcoded version literals (must use version.AppVersion):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
