// Command workable-items maintains the §11.4.93 SQLite single-source-of-truth
// for the project's workable items, synchronising it from the markdown trackers
// docs/Issues.md (open) and docs/Fixed.md (closed/fixed).
//
// Per §11.4.95 the produced docs/workable_items.db is TRACKED in git — it is
// authoritative source data, not a build artefact.
//
// Usage:
//
//	workable-items sync md-to-db [-issues PATH] [-fixed PATH] [-db PATH]
//	workable-items sync db-to-md [-db PATH] [-issues-summary PATH] [-fixed-summary PATH]
//	workable-items validate       [-issues PATH] [-fixed PATH] [-db PATH]
//	workable-items diff           [-issues PATH] [-fixed PATH] [-db PATH]
//	workable-items list           [-db PATH]
//	workable-items record-event   -atm ATM-NNN -event EVENT -on TIMESTAMP [-note NOTE] [-db PATH]
//
// Anti-bluff (§11.4): md-to-db parses the real markdown headings and populates a
// real, queryable SQLite database; db-to-md regenerates the §11.4.12/§11.4.53
// Issues_Summary/Fixed_Summary tables FROM the DB (the §11.4.93 bidirectional
// requirement); validate re-parses the markdown and compares the live DB row
// counts + ids against it, exiting non-zero on any drift; diff prints the same
// drift human-readably; record-event appends to the §11.4.93 append-only
// item_history audit trail (deterministic, no LLM).
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

const (
	defaultIssues        = "docs/Issues.md"
	defaultFixed         = "docs/Fixed.md"
	defaultDB            = "docs/workable_items.db"
	defaultIssuesSummary = "docs/Issues_Summary.md"
	defaultFixedSummary  = "docs/Fixed_Summary.md"

	locationOpen  = "open"
	locationFixed = "fixed"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "sync":
		os.Exit(cmdSync(os.Args[2:]))
	case "validate":
		os.Exit(cmdValidate(os.Args[2:]))
	case "diff":
		os.Exit(cmdDiff(os.Args[2:]))
	case "list":
		os.Exit(cmdList(os.Args[2:]))
	case "record-event":
		os.Exit(cmdRecordEvent(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "workable-items: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `workable-items — §11.4.93 SQLite single-source-of-truth tool

Commands:
  sync md-to-db   Parse the ATM-NNN headings from the markdown trackers into the DB.
  sync db-to-md   Regenerate the Issues_Summary/Fixed_Summary tables FROM the DB.
  validate        Verify the DB matches the markdown (counts + ids); exit non-zero on drift.
  diff            Print the md-vs-db drift human-readably (exit non-zero if any).
  list            Print the items currently in the DB.
  record-event    Append an event to the append-only item_history audit trail.

Common flags:
  -issues PATH         open-items markdown   (default docs/Issues.md)
  -fixed  PATH         fixed-items markdown  (default docs/Fixed.md)
  -db     PATH         SQLite database       (default docs/workable_items.db)

db-to-md flags:
  -issues-summary PATH (default docs/Issues_Summary.md)
  -fixed-summary  PATH (default docs/Fixed_Summary.md)

record-event flags:
  -atm ATM-NNN  -event EVENT  -on YYYY-MM-DDTHH:MM:SSZ  [-note NOTE]
`)
}

// cmdSync dispatches `sync md-to-db` and `sync db-to-md`.
func cmdSync(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "workable-items sync: need a direction ('md-to-db' or 'db-to-md')")
		return 2
	}
	switch args[0] {
	case "md-to-db":
		return cmdSyncMdToDB(args[1:])
	case "db-to-md":
		return cmdSyncDBToMd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "workable-items sync: unknown direction %q (want 'md-to-db' or 'db-to-md')\n", args[0])
		return 2
	}
}

// cmdSyncMdToDB handles `sync md-to-db`.
func cmdSyncMdToDB(args []string) int {
	fs := flag.NewFlagSet("sync md-to-db", flag.ExitOnError)
	issues := fs.String("issues", defaultIssues, "open-items markdown path")
	fixed := fs.String("fixed", defaultFixed, "fixed-items markdown path")
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	_ = fs.Parse(args)

	items, err := parseAll(*issues, *fixed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items sync: %v\n", err)
		return 1
	}

	n, err := writeDB(*dbPath, items)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items sync: %v\n", err)
		return 1
	}

	open, fixedN := countByLocation(items)
	fmt.Printf("synced %d items into %s (%d open, %d fixed)\n", n, *dbPath, open, fixedN)
	return 0
}

// cmdSyncDBToMd handles `sync db-to-md` — the §11.4.93 reverse direction. It
// regenerates the Issues_Summary/Fixed_Summary table blocks FROM the DB and
// writes complete summary files (header + table + total line) byte-compatible
// with the shell generators except for the non-deterministic `Last modified`
// timestamp (which is and must be `now`).
func cmdSyncDBToMd(args []string) int {
	fs := flag.NewFlagSet("sync db-to-md", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	issuesSummary := fs.String("issues-summary", defaultIssuesSummary, "Issues_Summary.md output path")
	fixedSummary := fs.String("fixed-summary", defaultFixedSummary, "Fixed_Summary.md output path")
	_ = fs.Parse(args)

	items, err := readDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items sync: %v\n", err)
		return 1
	}

	openN, fixedN := countByLocation(items)
	if err := os.WriteFile(*issuesSummary, []byte(renderIssuesSummaryDoc(items, openN)), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "workable-items sync: write %s: %v\n", *issuesSummary, err)
		return 1
	}
	if err := os.WriteFile(*fixedSummary, []byte(renderFixedSummaryDoc(items, fixedN)), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "workable-items sync: write %s: %v\n", *fixedSummary, err)
		return 1
	}
	fmt.Printf("regenerated %s (%d open) and %s (%d fixed) from %s\n",
		*issuesSummary, openN, *fixedSummary, fixedN, *dbPath)
	return 0
}

// cmdDiff handles `diff` — the same drift as validate but always printed,
// human-readably, even when in sync.
func cmdDiff(args []string) int {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	issues := fs.String("issues", defaultIssues, "open-items markdown path")
	fixed := fs.String("fixed", defaultFixed, "fixed-items markdown path")
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	_ = fs.Parse(args)

	mdItems, err := parseAll(*issues, *fixed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items diff: %v\n", err)
		return 1
	}
	dbItems, err := readDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items diff: %v\n", err)
		return 1
	}

	drift := diff(mdItems, dbItems)
	if len(drift) == 0 {
		fmt.Printf("no drift: DB %s matches markdown (%d items)\n", *dbPath, len(mdItems))
		return 0
	}
	fmt.Printf("%d difference(s) between markdown and DB %s:\n", len(drift), *dbPath)
	for _, d := range drift {
		fmt.Printf("  %s\n", d)
	}
	return 1
}

// cmdRecordEvent handles `record-event` — appends one row to the append-only
// §11.4.93 item_history audit trail. Deterministic, no LLM: the timestamp is an
// explicit -on argument (never the wall clock) so runs are reproducible.
func cmdRecordEvent(args []string) int {
	fs := flag.NewFlagSet("record-event", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	atm := fs.String("atm", "", "ATM-NNN item id (required)")
	event := fs.String("event", "", "event name, e.g. Opened/Reopened/Fixed (required)")
	on := fs.String("on", "", "ISO-8601 timestamp for the event (required)")
	note := fs.String("note", "", "optional free-text note")
	_ = fs.Parse(args)

	if *atm == "" || *event == "" || *on == "" {
		fmt.Fprintln(os.Stderr, "workable-items record-event: -atm, -event and -on are all required")
		return 2
	}

	if err := recordEvent(*dbPath, *atm, *event, *on, *note); err != nil {
		fmt.Fprintf(os.Stderr, "workable-items record-event: %v\n", err)
		return 1
	}
	fmt.Printf("recorded %s event %q @ %s\n", *atm, *event, *on)
	return 0
}

// cmdValidate handles `validate`.
func cmdValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	issues := fs.String("issues", defaultIssues, "open-items markdown path")
	fixed := fs.String("fixed", defaultFixed, "fixed-items markdown path")
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	_ = fs.Parse(args)

	mdItems, err := parseAll(*issues, *fixed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items validate: %v\n", err)
		return 1
	}

	dbItems, err := readDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items validate: %v\n", err)
		return 1
	}

	drift := diff(mdItems, dbItems)
	if len(drift) > 0 {
		fmt.Fprintf(os.Stderr, "workable-items validate: DRIFT detected (%d):\n", len(drift))
		for _, d := range drift {
			fmt.Fprintf(os.Stderr, "  - %s\n", d)
		}
		return 1
	}

	fmt.Printf("OK: DB %s matches markdown (%d items)\n", *dbPath, len(mdItems))
	return 0
}

// cmdList handles `list`.
func cmdList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	_ = fs.Parse(args)

	items, err := readDB(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workable-items list: %v\n", err)
		return 1
	}
	sortItems(items)
	for _, it := range items {
		fmt.Printf("%-9s  %-8s  %-30s  %s\n", it.ATMID, it.Location, it.Status, it.Title)
	}
	fmt.Printf("(%d items)\n", len(items))
	return 0
}

func countByLocation(items []Item) (open, fixed int) {
	for _, it := range items {
		if it.Location == locationFixed {
			fixed++
		} else {
			open++
		}
	}
	return open, fixed
}

func sortItems(items []Item) {
	sort.Slice(items, func(i, j int) bool { return items[i].ATMID < items[j].ATMID })
}

// parseAll parses both trackers and returns the merged item set, erroring on a
// duplicate ATM id across the two files (an integrity violation).
func parseAll(issuesPath, fixedPath string) ([]Item, error) {
	open, err := parseFile(issuesPath, locationOpen)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(issuesPath), err)
	}
	fixed, err := parseFile(fixedPath, locationFixed)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(fixedPath), err)
	}

	seen := make(map[string]string, len(open)+len(fixed))
	all := make([]Item, 0, len(open)+len(fixed))
	for _, it := range append(open, fixed...) {
		if where, dup := seen[it.ATMID]; dup {
			return nil, fmt.Errorf("duplicate %s (in %s and %s)", it.ATMID, where, it.Location)
		}
		seen[it.ATMID] = it.Location
		all = append(all, it)
	}
	return all, nil
}

// writeDB rebuilds the items table from scratch with the given items.
func writeDB(dbPath string, items []Item) (int, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	if _, err := db.Exec(schemaDDL); err != nil {
		return 0, fmt.Errorf("create schema: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec("DELETE FROM items"); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	stmt, err := tx.Prepare(
		"INSERT INTO items(atm_id, type, status, location, title) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	defer stmt.Close()
	for _, it := range items {
		if _, err := stmt.Exec(it.ATMID, it.Type, it.Status, it.Location, it.Title); err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("insert %s: %w", it.ATMID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(items), nil
}

// readDB reads all items from the DB.
func readDB(dbPath string) ([]Item, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database not found at %s (run 'sync md-to-db' first)", dbPath)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT atm_id, type, status, location, title FROM items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ATMID, &it.Type, &it.Status, &it.Location, &it.Title); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

const schemaDDL = `
CREATE TABLE IF NOT EXISTS items (
    atm_id   TEXT PRIMARY KEY,
    type     TEXT NOT NULL,
    status   TEXT NOT NULL,
    location TEXT NOT NULL CHECK (location IN ('open','fixed')),
    title    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS item_history (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    atm_id    TEXT NOT NULL,
    event     TEXT NOT NULL,
    ts        TEXT NOT NULL,
    note      TEXT NOT NULL DEFAULT '',
    recorded  TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_item_history_atm ON item_history(atm_id);`

// recordEvent appends one row to the append-only item_history audit trail. It
// never updates or deletes — append-only per §11.4.93. The DB (and its schema)
// is created on demand so record-event works before the first md-to-db sync.
func recordEvent(dbPath, atmID, event, ts, note string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(schemaDDL); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := db.Exec(
		"INSERT INTO item_history(atm_id, event, ts, note) VALUES (?, ?, ?, ?)",
		atmID, event, ts, note,
	); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// HistoryEvent is one row of the append-only item_history audit trail.
type HistoryEvent struct {
	ATMID string
	Event string
	TS    string
	Note  string
}

// readHistory returns the item_history rows for an item in insertion order.
func readHistory(dbPath, atmID string) ([]HistoryEvent, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		"SELECT atm_id, event, ts, note FROM item_history WHERE atm_id = ? ORDER BY id", atmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hist []HistoryEvent
	for rows.Next() {
		var h HistoryEvent
		if err := rows.Scan(&h.ATMID, &h.Event, &h.TS, &h.Note); err != nil {
			return nil, err
		}
		hist = append(hist, h)
	}
	return hist, rows.Err()
}
