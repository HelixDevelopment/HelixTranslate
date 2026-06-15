package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateTLSCertificates_KeyFileNotWorldReadable asserts the generated
// private key file is NOT readable/writable by group or other (i.e. mode 0600).
// A TLS private key written world-readable is a credential-leak class defect:
// any local user could read the server's private key. generateTLSCertificates
// writes into a fixed "certs/" directory, so the test runs inside a temp working
// directory and restores cwd afterwards.
func TestGenerateTLSCertificates_KeyFileNotWorldReadable(t *testing.T) {
	tmp := t.TempDir()

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir to temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	if err := generateTLSCertificates(); err != nil {
		t.Fatalf("generateTLSCertificates: %v", err)
	}

	keyPath := filepath.Join(tmp, "certs", "server.key")
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}

	mode := info.Mode().Perm()
	// Group/other bits must all be clear on a private key.
	if mode&0o077 != 0 {
		t.Fatalf("private key %s has insecure permissions %o; group/other must have no access (expect 0600)", keyPath, mode)
	}
}
