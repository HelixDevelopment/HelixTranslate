package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const fixtureIssues = `# Tracker — Open

### §1. [ATM-100] First open item title here
**Status:** Queued
**Type:** Bug
- some detail

### §2. [ATM-101] Second open item with a longer descriptive title
**Status:** Operator-blocked
**Type:** Task
- detail
`

const fixtureFixed = `# Tracker — Fixed

### §1. [ATM-001] A closed bug that was fixed
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Evidence: commit abc1234

### §2. [ATM-002] A closed task that was completed
**Status:** Completed (→ Fixed.md)
**Type:** Task
- Evidence: commit def5678
`

func writeFixture(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
	return p
}

// TestSyncMdToDB_PopulatesRealDB writes tiny Issues/Fixed fixtures, runs the
// real md-to-db sync, opens the produced SQLite DB and asserts concrete row
// contents. Mutation-proof: corrupt the parser or the writer and the exact-row
// assertions below fail.
func TestSyncMdToDB_PopulatesRealDB(t *testing.T) {
	dir := t.TempDir()
	issues := filepath.Join(dir, "Issues.md")
	fixed := filepath.Join(dir, "Fixed.md")
	dbPath := filepath.Join(dir, "wi.db")
	if err := os.WriteFile(issues, []byte(fixtureIssues), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixed, []byte(fixtureFixed), 0o600); err != nil {
		t.Fatal(err)
	}

	if rc := cmdSync([]string{"md-to-db", "-issues", issues, "-fixed", fixed, "-db", dbPath}); rc != 0 {
		t.Fatalf("cmdSync returned %d, want 0", rc)
	}

	// Open the real DB and verify it is genuinely queryable.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var total int
	if err := db.QueryRow("SELECT count(*) FROM items").Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != 4 {
		t.Fatalf("row count = %d, want 4", total)
	}

	var open, fixedN int
	if err := db.QueryRow("SELECT count(*) FROM items WHERE location='open'").Scan(&open); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM items WHERE location='fixed'").Scan(&fixedN); err != nil {
		t.Fatal(err)
	}
	if open != 2 || fixedN != 2 {
		t.Fatalf("open=%d fixed=%d, want 2/2", open, fixedN)
	}

	// Assert a specific row's full contents (mutation-proof on parse + write).
	var typ, status, loc, title string
	row := db.QueryRow("SELECT type, status, location, title FROM items WHERE atm_id='ATM-101'")
	if err := row.Scan(&typ, &status, &loc, &title); err != nil {
		t.Fatalf("scan ATM-101: %v", err)
	}
	if typ != "Task" {
		t.Errorf("ATM-101 type = %q, want Task", typ)
	}
	if status != "Operator-blocked" {
		t.Errorf("ATM-101 status = %q, want Operator-blocked", status)
	}
	if loc != "open" {
		t.Errorf("ATM-101 location = %q, want open", loc)
	}
	if title != "Second open item with a longer descriptive title" {
		t.Errorf("ATM-101 title = %q", title)
	}

	// A fixed item keeps its closure-vocabulary status verbatim.
	var fStatus string
	if err := db.QueryRow("SELECT status FROM items WHERE atm_id='ATM-002'").Scan(&fStatus); err != nil {
		t.Fatalf("scan ATM-002: %v", err)
	}
	if fStatus != "Completed (→ Fixed.md)" {
		t.Errorf("ATM-002 status = %q, want Completed (→ Fixed.md)", fStatus)
	}
}

// TestValidate_DetectsDrift proves validate exits non-zero when the DB no longer
// matches the markdown — the anti-bluff guard that makes a stale DB a failure.
func TestValidate_DetectsDrift(t *testing.T) {
	dir := t.TempDir()
	issues := filepath.Join(dir, "Issues.md")
	fixed := filepath.Join(dir, "Fixed.md")
	dbPath := filepath.Join(dir, "wi.db")
	mustWrite(t, issues, fixtureIssues)
	mustWrite(t, fixed, fixtureFixed)

	if rc := cmdSync([]string{"md-to-db", "-issues", issues, "-fixed", fixed, "-db", dbPath}); rc != 0 {
		t.Fatalf("sync rc=%d", rc)
	}
	// In sync → validate passes.
	if rc := cmdValidate([]string{"-issues", issues, "-fixed", fixed, "-db", dbPath}); rc != 0 {
		t.Fatalf("validate (in-sync) rc=%d, want 0", rc)
	}

	// Mutate the markdown (add an item) without re-syncing → drift.
	mustWrite(t, issues, fixtureIssues+
		"\n### §3. [ATM-102] A new item not yet in the DB\n**Status:** Queued\n**Type:** Task\n")
	if rc := cmdValidate([]string{"-issues", issues, "-fixed", fixed, "-db", dbPath}); rc == 0 {
		t.Fatal("validate (drift) rc=0, want non-zero — drift not detected")
	}
}

// TestParseAll_RejectsDuplicateID guards the cross-file integrity invariant.
func TestParseAll_RejectsDuplicateID(t *testing.T) {
	dir := t.TempDir()
	issues := filepath.Join(dir, "Issues.md")
	fixed := filepath.Join(dir, "Fixed.md")
	mustWrite(t, issues, "### [ATM-500] x\n**Status:** Queued\n**Type:** Bug\n")
	mustWrite(t, fixed, "### [ATM-500] x\n**Status:** Fixed (→ Fixed.md)\n**Type:** Bug\n")

	if _, err := parseAll(issues, fixed); err == nil {
		t.Fatal("parseAll accepted a duplicate ATM id, want error")
	}
}

// TestParseFile_RequiresStatusAndType guards §11.4.148 D1.
func TestParseFile_RequiresStatusAndType(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Issues.md")
	mustWrite(t, p, "### [ATM-600] no metadata here\n- just a detail\n")
	if _, err := parseFile(p, locationOpen); err == nil {
		t.Fatal("parseFile accepted an item with no Status/Type, want error")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
