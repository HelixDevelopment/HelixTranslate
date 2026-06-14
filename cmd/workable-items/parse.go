package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

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
