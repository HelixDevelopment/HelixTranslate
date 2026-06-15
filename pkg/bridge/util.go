package bridge

import (
	"os"
	"path/filepath"
)

// osGetenv is the default environment source for Open. Indirected so tests can
// inject a fake without touching the process environment (§11.4.10).
func osGetenv(k string) string { return os.Getenv(k) }

// ensureDir creates the parent directory of dbPath if missing.
func ensureDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
