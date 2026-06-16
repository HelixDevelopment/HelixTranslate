package storage

import (
	"path/filepath"
	"testing"
)

// TestSQLite_DefaultsToSingleWriterConnection is the DETERMINISTIC §11.4.135
// regression guard for the SQLITE_BUSY concurrency fix. The probabilistic
// TestStress_SQLiteCache_ConcurrentPutGet catches the *symptom* (writes failing
// under contention) but only intermittently on a fast disk; this guard asserts
// the *mechanism* so a revert FAILs deterministically: NewSQLiteStorage with no
// explicit MaxOpenConns MUST cap the pool at a single connection, so SQLite's
// single-file writers serialize cleanly instead of colliding (SQLITE_BUSY).
//
// Revert the `else { db.SetMaxOpenConns(1) }` default → MaxOpenConnections is 0
// (unlimited) → this test FAILs. RED-on-broken, GREEN-on-fixed.
func TestSQLite_DefaultsToSingleWriterConnection(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "guard.db")
	st, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	defer st.Close()

	if got := st.db.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("default MaxOpenConnections = %d, want 1 — SQLite must serialize "+
			"writers (§11.4.85 SQLITE_BUSY fix); was the SetMaxOpenConns(1) default reverted?", got)
	}
}

// TestSQLite_RespectsExplicitMaxOpenConns guards the opt-out: an explicit
// config value MUST still win over the single-connection default (so callers
// that want a larger pool, relying on _busy_timeout for contention, can opt in).
func TestSQLite_RespectsExplicitMaxOpenConns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "guard_explicit.db")
	st, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath, MaxOpenConns: 4})
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	defer st.Close()

	if got := st.db.Stats().MaxOpenConnections; got != 4 {
		t.Fatalf("explicit MaxOpenConns=4 → MaxOpenConnections = %d, want 4 (default must not override an explicit value)", got)
	}
}

// TestSQLite_PlainDatabaseStringIsValidPath is a light guard that the DSN-build
// logic (which now appends `_busy_timeout` and merges query params) still leaves
// a usable storage handle for a plain path — schema init must succeed, proving
// the busy_timeout/encryption param composition did not produce a malformed DSN.
func TestSQLite_PlainDatabaseStringIsValidPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "guard_plain.db")
	st, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStorage with plain path failed (malformed DSN after _busy_timeout merge?): %v", err)
	}
	defer st.Close()
	// initSchema ran inside NewSQLiteStorage; a Ping confirms the handle is live.
	if err := st.db.Ping(); err != nil {
		t.Fatalf("db.Ping after open failed: %v", err)
	}
}
