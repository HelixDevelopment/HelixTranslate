package verification

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestMultiPassPolisher_saveNewResults_NoDuplicateWrites pins the multipass
// duplicate-write defect. report.SectionResults accumulates across chapters; the
// old polishWithNotes code re-saved the WHOLE slice every chapter, so chapter
// 0's result was re-saved on chapters 1..N-1 — producing O(n^2) duplicate
// polishing_changes rows (that table has an autoincrement PK and no unique key)
// and PK-conflicting (silently-failing) section_results inserts.
//
// This test drives the REAL production helper saveNewResults exactly as the
// chapter loop does (append one result per chapter, save after each), then
// asserts each result + change is persisted exactly once. Mutating the helper to
// save report.SectionResults (the whole slice) reproduces the original bug:
// polishing_changes then holds 1+2+...+N rows instead of N.
func TestMultiPassPolisher_saveNewResults_NoDuplicateWrites(t *testing.T) {
	tmp, err := os.CreateTemp("", "test_dupwrite_*.db")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	db, err := NewPolishingDatabase(tmp.Name())
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer db.Close()

	const sessionID = "sess-dupwrite"
	const passID = "pass-dupwrite"
	if err := db.CreateSession(&PolishingSession{
		SessionID: sessionID, BookID: "b", BookTitle: "t", StartedAt: time.Now(), Status: "running",
	}); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := db.CreatePass(&PassRecord{
		PassID: passID, SessionID: sessionID, PassNumber: 1, StartedAt: time.Now(), Status: "running",
	}); err != nil {
		t.Fatalf("CreatePass: %v", err)
	}

	mpp := &MultiPassPolisher{database: db}

	// Simulate polishWithNotes' chapter loop: the polisher appends one result per
	// chapter to report.SectionResults; saveNewResults is called after each.
	report := &PolishingReport{}
	const chapters = 5
	savedResults := 0
	for i := 0; i < chapters; i++ {
		sectionID := fmt.Sprintf("section-%d", i)
		report.SectionResults = append(report.SectionResults, &PolishingResult{
			SectionID: sectionID,
			Location:  fmt.Sprintf("Chapter %d", i+1),
			Changes: []Change{
				{Location: sectionID, Original: "o", Polished: "p", Reason: "r", Agreement: 1, Confidence: 0.9},
			},
		})
		savedResults = mpp.saveNewResults(report, savedResults, passID)
	}

	// Each section persisted exactly once.
	results, err := db.GetResultsForPass(passID)
	if err != nil {
		t.Fatalf("GetResultsForPass: %v", err)
	}
	if len(results) != chapters {
		t.Fatalf("section_results rows = %d, want %d", len(results), chapters)
	}

	// The real corruption surface: exactly one change row per chapter.
	var changeCount int
	if err := db.db.QueryRow(
		"SELECT COUNT(*) FROM polishing_changes WHERE pass_id = ?", passID,
	).Scan(&changeCount); err != nil {
		t.Fatalf("count polishing_changes: %v", err)
	}
	if changeCount != chapters {
		t.Fatalf("polishing_changes rows = %d, want %d (duplicate writes — each chapter re-saved earlier chapters)",
			changeCount, chapters)
	}
}
