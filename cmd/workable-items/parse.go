package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// nowUTC is overridable in tests so the timestamped header is deterministic.
var nowUTC = func() string { return time.Now().UTC().Format("2006-01-02T15:04:05Z") }

// Item is one workable item parsed from a markdown tracker.
type Item struct {
	ATMID    string // e.g. "ATM-065"
	Type     string // Bug | Feature | Task (from **Type:**)
	Status   string // raw status string (from **Status:**)
	Location string // open | fixed
	Title    string // heading text after the [ATM-NNN] token
}

// headingRE matches an ATM heading line, e.g.
//
//	### §1. [ATM-065] Decide the single authoritative version number
//
// capturing the ATM id and the trailing title.
var headingRE = regexp.MustCompile(`^#+\s+.*?\[(ATM-\d+)\]\s*(.*?)\s*$`)

// fieldRE matches a bold metadata line, e.g. `**Status:** Operator-blocked`
// or `**Type:** Bug`, capturing the field name and value.
var fieldRE = regexp.MustCompile(`^\*\*(Status|Type):\*\*\s*(.*?)\s*$`)

// parseFile parses one markdown tracker file, attaching the given location to
// every item. Status/Type are read from the bold metadata lines that follow
// each heading (within the heading's section, before the next heading).
func parseFile(path, location string) ([]Item, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		items []Item
		cur   *Item
	)
	flush := func() {
		if cur != nil {
			items = append(items, *cur)
			cur = nil
		}
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if m := headingRE.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Item{ATMID: m[1], Title: m[2], Location: location}
			continue
		}
		if cur == nil {
			continue
		}
		if m := fieldRE.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "Status":
				if cur.Status == "" {
					cur.Status = m[2]
				}
			case "Type":
				if cur.Type == "" {
					cur.Type = m[2]
				}
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}

	// Integrity: every parsed item must carry status + type (§11.4.148 D1).
	for _, it := range items {
		if it.Status == "" {
			return nil, fmt.Errorf("%s missing **Status:**", it.ATMID)
		}
		if it.Type == "" {
			return nil, fmt.Errorf("%s missing **Type:**", it.ATMID)
		}
	}
	return items, nil
}

// diff returns a sorted list of human-readable drift descriptions between the
// markdown-derived set and the DB-derived set. Empty result == in sync.
func diff(md, db []Item) []string {
	mdByID := index(md)
	dbByID := index(db)

	var drift []string
	for id, m := range mdByID {
		d, ok := dbByID[id]
		if !ok {
			drift = append(drift, fmt.Sprintf("%s present in markdown, missing from DB", id))
			continue
		}
		var mismatches []string
		if m.Type != d.Type {
			mismatches = append(mismatches, fmt.Sprintf("type %q!=%q", m.Type, d.Type))
		}
		if m.Status != d.Status {
			mismatches = append(mismatches, fmt.Sprintf("status %q!=%q", m.Status, d.Status))
		}
		if m.Location != d.Location {
			mismatches = append(mismatches, fmt.Sprintf("location %q!=%q", m.Location, d.Location))
		}
		if m.Title != d.Title {
			mismatches = append(mismatches, fmt.Sprintf("title %q!=%q", m.Title, d.Title))
		}
		if len(mismatches) > 0 {
			drift = append(drift, fmt.Sprintf("%s: %s", id, strings.Join(mismatches, ", ")))
		}
	}
	for id := range dbByID {
		if _, ok := mdByID[id]; !ok {
			drift = append(drift, fmt.Sprintf("%s present in DB, missing from markdown", id))
		}
	}
	sort.Strings(drift)
	return drift
}

func index(items []Item) map[string]Item {
	m := make(map[string]Item, len(items))
	for _, it := range items {
		m[it.ATMID] = it
	}
	return m
}

// issuesLevel derives the Issues_Summary "Level" column purely from Status,
// matching scripts/testing/generate_issues_summary.sh exactly:
// operator-blocked/blocked → High, design → Medium, otherwise Normal.
func issuesLevel(status string) string {
	s := strings.ToLower(status)
	switch {
	case strings.Contains(s, "operator-blocked"), strings.Contains(s, "blocked"):
		return "High"
	case strings.Contains(s, "design"):
		return "Medium"
	default:
		return "Normal"
	}
}

// fixedLevel derives the Fixed_Summary "Level" column purely from Type,
// matching scripts/testing/generate_fixed_summary.sh exactly.
func fixedLevel(typ string) string {
	switch typ {
	case "Bug":
		return "Bug"
	case "Feature":
		return "Feature"
	default:
		return "Task"
	}
}

// renderSummaryTable renders ONLY the markdown summary table block (header row,
// divider, one row per item in ATM-id order) for the given location. It does NOT
// emit the file header (title / Revision / Last modified / Authority / Generated)
// because that header carries a non-deterministic `date -u` timestamp; rendering
// only the table is what makes a byte-stable round-trip honest (§11.4.6) — the
// table content is fully reconstructible from the DB, the timestamp is not.
//
// The column layout is identical to the shell generators:
//
//	| ATM ID | Level | Status | Type | One-line description |
//
// where the description IS the heading title stored in the DB.
func renderSummaryTable(items []Item, location string) string {
	rows := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Location == location {
			rows = append(rows, it)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ATMID < rows[j].ATMID })

	var b strings.Builder
	b.WriteString("| ATM ID | Level | Status | Type | One-line description |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, it := range rows {
		var level string
		if location == locationFixed {
			level = fixedLevel(it.Type)
		} else {
			level = issuesLevel(it.Status)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			it.ATMID, level, it.Status, it.Type, it.Title)
	}
	return b.String()
}

// renderIssuesSummaryDoc renders the full Issues_Summary.md document from the DB
// items, matching scripts/testing/generate_issues_summary.sh byte-for-byte
// except the non-deterministic `Last modified` timestamp.
func renderIssuesSummaryDoc(items []Item, openCount int) string {
	var b strings.Builder
	b.WriteString("# HelixTranslate — Issues Summary\n\n")
	b.WriteString("**Revision:** 1\n")
	fmt.Fprintf(&b, "**Last modified:** %s\n", nowUTC())
	b.WriteString("**Authority:** §11.4.12 (Issues_Summary sync) · §11.4.54 (ATM ID column) · " +
		"§11.4.19 (column-alignment) · §11.4.91 (clear descriptions)\n")
	b.WriteString("**Generated:** auto-generated from `docs/Issues.md` by " +
		"`scripts/testing/generate_issues_summary.sh` — do not hand-edit.\n\n")
	b.WriteString(renderSummaryTable(items, locationOpen))
	fmt.Fprintf(&b, "\nTotal open items: %d\n", openCount)
	return b.String()
}

// renderFixedSummaryDoc renders the full Fixed_Summary.md document from the DB
// items, matching scripts/testing/generate_fixed_summary.sh byte-for-byte
// except the non-deterministic `Last modified` timestamp.
func renderFixedSummaryDoc(items []Item, fixedCount int) string {
	var b strings.Builder
	b.WriteString("# HelixTranslate — Fixed Summary\n\n")
	b.WriteString("**Revision:** 1\n")
	fmt.Fprintf(&b, "**Last modified:** %s\n", nowUTC())
	b.WriteString("**Authority:** §11.4.53 (Fixed_Summary parity) · §11.4.54 (ATM ID column) · " +
		"§11.4.19 (column-alignment) · §11.4.91 (clear descriptions)\n")
	b.WriteString("**Generated:** auto-generated from `docs/Fixed.md` by " +
		"`scripts/testing/generate_fixed_summary.sh` — do not hand-edit.\n\n")
	b.WriteString(renderSummaryTable(items, locationFixed))
	fmt.Fprintf(&b, "\nTotal closed items: %d\n", fixedCount)
	return b.String()
}

// extractSummaryTable pulls the table block (from the `| ATM ID |` header row
// through the last contiguous table row) out of a rendered summary markdown file,
// so a round-trip comparison ignores the timestamped file header and trailing
// total line. Returns "" if no table header is found.
func extractSummaryTable(md string) string {
	lines := strings.Split(md, "\n")
	start := -1
	for i, ln := range lines {
		if strings.HasPrefix(ln, "| ATM ID |") {
			start = i
			break
		}
	}
	if start < 0 {
		return ""
	}
	var out []string
	for _, ln := range lines[start:] {
		if strings.HasPrefix(ln, "|") {
			out = append(out, strings.TrimRight(ln, " "))
			continue
		}
		break
	}
	return strings.Join(out, "\n") + "\n"
}
