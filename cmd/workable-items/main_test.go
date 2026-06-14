package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
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

// TestRoundTrip_MdToDbToMd is the §11.4.93 bidirectional proof: parse the
// markdown into the DB, regenerate the summary docs FROM the DB, and assert the
// regenerated summary TABLE is byte-identical to a freshly rendered table built
// directly from the same parsed items. The timestamped header is excluded
// (extractSummaryTable) because it is genuinely non-deterministic (§11.4.6) — we
// do not claim a byte-stable header, only a byte-stable table, and we prove it.
// Mutation-proof: change renderSummaryTable's column order or the Level logic
// and the exact-string assertions below fail.
func TestRoundTrip_MdToDbToMd(t *testing.T) {
	dir := t.TempDir()
	issues := filepath.Join(dir, "Issues.md")
	fixed := filepath.Join(dir, "Fixed.md")
	dbPath := filepath.Join(dir, "wi.db")
	issuesSum := filepath.Join(dir, "Issues_Summary.md")
	fixedSum := filepath.Join(dir, "Fixed_Summary.md")
	mustWrite(t, issues, fixtureIssues)
	mustWrite(t, fixed, fixtureFixed)

	// Deterministic header timestamp so the WHOLE file is reproducible too.
	saved := nowUTC
	nowUTC = func() string { return "2026-01-01T00:00:00Z" }
	defer func() { nowUTC = saved }()

	if rc := cmdSyncMdToDB([]string{"-issues", issues, "-fixed", fixed, "-db", dbPath}); rc != 0 {
		t.Fatalf("md-to-db rc=%d", rc)
	}
	if rc := cmdSyncDBToMd([]string{"-db", dbPath, "-issues-summary", issuesSum, "-fixed-summary", fixedSum}); rc != 0 {
		t.Fatalf("db-to-md rc=%d", rc)
	}

	// The DB-regenerated table must equal a table rendered straight from the
	// same source items (the round-trip preserves every column).
	srcItems, err := parseAll(issues, fixed)
	if err != nil {
		t.Fatalf("parseAll: %v", err)
	}
	wantOpen := renderSummaryTable(srcItems, locationOpen)
	gotOpenDoc, err := os.ReadFile(issuesSum)
	if err != nil {
		t.Fatal(err)
	}
	gotOpen := extractSummaryTable(string(gotOpenDoc))
	if gotOpen != wantOpen {
		t.Fatalf("Issues summary table round-trip mismatch:\n got:\n%s\nwant:\n%s", gotOpen, wantOpen)
	}

	wantFixed := renderSummaryTable(srcItems, locationFixed)
	gotFixedDoc, err := os.ReadFile(fixedSum)
	if err != nil {
		t.Fatal(err)
	}
	gotFixed := extractSummaryTable(string(gotFixedDoc))
	if gotFixed != wantFixed {
		t.Fatalf("Fixed summary table round-trip mismatch:\n got:\n%s\nwant:\n%s", gotFixed, wantFixed)
	}

	// Concrete content assertion (mutation-proof): the open fixture's ATM-101 is
	// Operator-blocked → Level High; render exactly that row.
	wantRow := "| ATM-101 | High | Operator-blocked | Task | Second open item with a longer descriptive title |"
	if !strings.Contains(gotOpen, wantRow) {
		t.Errorf("Issues table missing expected row:\n%s\nin:\n%s", wantRow, gotOpen)
	}
	// Fixed fixture ATM-002 is a Task → Level Task, keeps closure status verbatim.
	wantFixedRow := "| ATM-002 | Task | Completed (→ Fixed.md) | Task | A closed task that was completed |"
	if !strings.Contains(gotFixed, wantFixedRow) {
		t.Errorf("Fixed table missing expected row:\n%s\nin:\n%s", wantFixedRow, gotFixed)
	}
}

// TestRecordEvent_AppendOnlyHistory proves record-event genuinely INSERTs into a
// real item_history table and readHistory reads the rows back in insertion order
// — the §11.4.93 append-only audit trail. Mutation-proof: drop the INSERT in
// recordEvent and the length/content assertions fail.
func TestRecordEvent_AppendOnlyHistory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wi.db")

	if rc := cmdRecordEvent([]string{"-db", dbPath, "-atm", "ATM-100", "-event", "Opened", "-on", "2026-01-01T00:00:00Z", "-note", "first"}); rc != 0 {
		t.Fatalf("record-event #1 rc=%d", rc)
	}
	if rc := cmdRecordEvent([]string{"-db", dbPath, "-atm", "ATM-100", "-event", "Reopened", "-on", "2026-02-01T00:00:00Z", "-note", "regressed"}); rc != 0 {
		t.Fatalf("record-event #2 rc=%d", rc)
	}
	// A different item must not bleed into ATM-100's history.
	if rc := cmdRecordEvent([]string{"-db", dbPath, "-atm", "ATM-999", "-event", "Opened", "-on", "2026-03-01T00:00:00Z"}); rc != 0 {
		t.Fatalf("record-event #3 rc=%d", rc)
	}

	hist, err := readHistory(dbPath, "ATM-100")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(hist) != 2 {
		t.Fatalf("ATM-100 history len = %d, want 2", len(hist))
	}
	if hist[0].Event != "Opened" || hist[0].TS != "2026-01-01T00:00:00Z" || hist[0].Note != "first" {
		t.Errorf("event[0] = %+v", hist[0])
	}
	if hist[1].Event != "Reopened" || hist[1].Note != "regressed" {
		t.Errorf("event[1] = %+v", hist[1])
	}

	// Direct DB count confirms append-only (3 total rows across both items).
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var total int
	if err := db.QueryRow("SELECT count(*) FROM item_history").Scan(&total); err != nil {
		t.Fatalf("count item_history: %v", err)
	}
	if total != 3 {
		t.Fatalf("item_history total = %d, want 3", total)
	}
}

// TestRecordEvent_RequiresArgs guards the required-flag validation.
func TestRecordEvent_RequiresArgs(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "wi.db")
	if rc := cmdRecordEvent([]string{"-db", dbPath, "-atm", "ATM-1"}); rc == 0 {
		t.Fatal("record-event accepted missing -event/-on, want non-zero")
	}
}

// TestDiff_ReportsAndExits proves diff exits 0 when in sync and non-zero on drift.
func TestDiff_ReportsAndExits(t *testing.T) {
	dir := t.TempDir()
	issues := filepath.Join(dir, "Issues.md")
	fixed := filepath.Join(dir, "Fixed.md")
	dbPath := filepath.Join(dir, "wi.db")
	mustWrite(t, issues, fixtureIssues)
	mustWrite(t, fixed, fixtureFixed)

	if rc := cmdSyncMdToDB([]string{"-issues", issues, "-fixed", fixed, "-db", dbPath}); rc != 0 {
		t.Fatalf("sync rc=%d", rc)
	}
	if rc := cmdDiff([]string{"-issues", issues, "-fixed", fixed, "-db", dbPath}); rc != 0 {
		t.Fatalf("diff (in-sync) rc=%d, want 0", rc)
	}
	mustWrite(t, issues, fixtureIssues+
		"\n### §3. [ATM-102] A brand new open item not yet synced\n**Status:** Queued\n**Type:** Task\n")
	if rc := cmdDiff([]string{"-issues", issues, "-fixed", fixed, "-db", dbPath}); rc == 0 {
		t.Fatal("diff (drift) rc=0, want non-zero")
	}
}
