package verification

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestPolishingDatabase(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_polishing_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test database creation
	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test schema initialization
	if err := db.initSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
}

func TestPolishingDatabase_SessionLifecycle(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_session_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	session := &PolishingSession{
		SessionID:   "test-session-1",
		BookID:      "book-123",
		BookTitle:   "Test Book",
		StartedAt:   time.Now(),
		ConfigJSON:  `{"test": "config"}`,
		Status:      "running",
	}

	// Test session creation
	err = db.CreateSession(session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test session retrieval
	retrieved, err := db.GetSession(session.SessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if retrieved.SessionID != session.SessionID {
		t.Errorf("Expected session ID %s, got %s", session.SessionID, retrieved.SessionID)
	}

	if retrieved.BookID != session.BookID {
		t.Errorf("Expected book ID %s, got %s", session.BookID, retrieved.BookID)
	}

	if retrieved.Status != session.Status {
		t.Errorf("Expected status %s, got %s", session.Status, retrieved.Status)
	}

	// Test session update
	completedAt := time.Now()
	err = db.UpdateSession(session.SessionID, "completed", completedAt, 3)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	updated, err := db.GetSession(session.SessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated session: %v", err)
	}

	if updated.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", updated.Status)
	}

	if updated.TotalPasses != 3 {
		t.Errorf("Expected total passes 3, got %d", updated.TotalPasses)
	}
}

func TestPolishingDatabase_PassLifecycle(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_pass_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session first
	session := &PolishingSession{
		SessionID:   "test-session-pass",
		BookID:      "book-456",
		BookTitle:   "Pass Test Book",
		StartedAt:   time.Now(),
		Status:      "running",
	}
	err = db.CreateSession(session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	pass := &PassRecord{
		PassID:     "test-pass-1",
		SessionID:  session.SessionID,
		PassNumber: 1,
		Providers:  `["openai", "zhipu"]`,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	// Test pass creation
	err = db.CreatePass(pass)
	if err != nil {
		t.Fatalf("Failed to create pass: %v", err)
	}

	// Test pass retrieval
	retrieved, err := db.GetPass(pass.PassID)
	if err != nil {
		t.Fatalf("Failed to retrieve pass: %v", err)
	}

	if retrieved.PassID != pass.PassID {
		t.Errorf("Expected pass ID %s, got %s", pass.PassID, retrieved.PassID)
	}

	if retrieved.SessionID != pass.SessionID {
		t.Errorf("Expected session ID %s, got %s", pass.SessionID, retrieved.SessionID)
	}

	if retrieved.PassNumber != pass.PassNumber {
		t.Errorf("Expected pass number %d, got %d", pass.PassNumber, retrieved.PassNumber)
	}

	// Test pass update
	completedAt := time.Now()
	err = db.UpdatePass(pass.PassID, "completed", completedAt)
	if err != nil {
		t.Fatalf("Failed to update pass: %v", err)
	}

	updated, err := db.GetPass(pass.PassID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated pass: %v", err)
	}

	if updated.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", updated.Status)
	}
}

func TestPolishingDatabase_Notes(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_notes_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session and pass
	session := &PolishingSession{
		SessionID:   "test-session-notes",
		BookID:      "book-notes",
		BookTitle:   "Notes Test Book",
		StartedAt:   time.Now(),
		Status:      "running",
	}
	db.CreateSession(session)

	pass := &PassRecord{
		PassID:     "test-pass-notes",
		SessionID:  session.SessionID,
		PassNumber: 1,
		StartedAt:  time.Now(),
		Status:     "running",
	}
	db.CreatePass(pass)

	note := &LiteraryNote{
		ID:          "note-1",
		SectionID:   "section-123",
		Location:    "Chapter 1, Page 5",
		Provider:    "openai",
		NoteType:    NoteTypeStyle,
		Importance:  ImportanceCritical,
		Title:       "Style Issue",
		Content:     "The style needs improvement",
		Examples:    []string{"Example 1", "Example 2"},
		Implications: "This affects readability",
		CreatedAt:   time.Now(),
	}

	// Test note saving
	err = db.SaveNote(note, pass.PassID)
	if err != nil {
		t.Fatalf("Failed to save note: %v", err)
	}

	// Test retrieving notes for section
	notes, err := db.GetNotesForSection(note.SectionID)
	if err != nil {
		t.Fatalf("Failed to retrieve notes for section: %v", err)
	}

	if len(notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(notes))
	}

	retrieved := notes[0]
	if retrieved.ID != note.ID {
		t.Errorf("Expected note ID %s, got %s", note.ID, retrieved.ID)
	}

	if retrieved.Provider != note.Provider {
		t.Errorf("Expected provider %s, got %s", note.Provider, retrieved.Provider)
	}

	if retrieved.NoteType != note.NoteType {
		t.Errorf("Expected note type %s, got %s", note.NoteType, retrieved.NoteType)
	}

	// Test retrieving notes for pass
	passNotes, err := db.GetNotesForPass(pass.PassID)
	if err != nil {
		t.Fatalf("Failed to retrieve notes for pass: %v", err)
	}

	if len(passNotes) != 1 {
		t.Fatalf("Expected 1 note for pass, got %d", len(passNotes))
	}
}

func TestPolishingDatabase_Results(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_results_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session and pass
	session := &PolishingSession{
		SessionID:   "test-session-results",
		BookID:      "book-results",
		BookTitle:   "Results Test Book",
		StartedAt:   time.Now(),
		Status:      "running",
	}
	db.CreateSession(session)

	pass := &PassRecord{
		PassID:     "test-pass-results",
		SessionID:  session.SessionID,
		PassNumber: 1,
		StartedAt:  time.Now(),
		Status:     "running",
	}
	db.CreatePass(pass)

	result := &PolishingResult{
		SectionID:       "section-results",
		Location:        "Chapter 2",
		OriginalText:    "Original text",
		TranslatedText:  "Translated text",
		PolishedText:    "Polished text",
		SpiritScore:     8.5,
		LanguageScore:   7.8,
		ContextScore:    9.0,
		VocabularyScore: 8.2,
		OverallScore:    8.4,
		Consensus:       1,
		Confidence:      0.85,
	}

	// Test result saving
	err = db.SaveResult(result, pass.PassID)
	if err != nil {
		t.Fatalf("Failed to save result: %v", err)
	}

	// Test changes saving
	changes := []Change{
		{
			Location:   "Chapter 2, Line 5",
			Original:   "Original text",
			Polished:   "Polished text",
			Reason:     "Grammar correction",
			Agreement:  2,
			Confidence: 0.9,
		},
	}

	err = db.SaveChanges(changes, pass.PassID, result.SectionID)
	if err != nil {
		t.Fatalf("Failed to save changes: %v", err)
	}

	// Test retrieving results for pass
	results, err := db.GetResultsForPass(pass.PassID)
	if err != nil {
		t.Fatalf("Failed to retrieve results for pass: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	retrieved := results[0]
	if retrieved.SectionID != result.SectionID {
		t.Errorf("Expected section ID %s, got %s", result.SectionID, retrieved.SectionID)
	}

	if retrieved.OverallScore != result.OverallScore {
		t.Errorf("Expected overall score %f, got %f", result.OverallScore, retrieved.OverallScore)
	}
}

func TestPolishingDatabase_SessionStats(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_stats_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session and pass
	session := &PolishingSession{
		SessionID:   "test-session-stats",
		BookID:      "book-stats",
		BookTitle:   "Stats Test Book",
		StartedAt:   time.Now(),
		Status:      "completed",
	}
	db.CreateSession(session)

	pass := &PassRecord{
		PassID:     "test-pass-stats",
		SessionID:  session.SessionID,
		PassNumber: 1,
		StartedAt:  time.Now(),
		Status:     "completed",
	}
	db.CreatePass(pass)

	// Add some test data
	note := &LiteraryNote{
		ID:          "note-stats",
		SectionID:   "section-stats",
		Provider:    "openai",
		NoteType:    NoteTypeStyle,
		Importance:  ImportanceMedium,
		Title:       "Test Note",
		Content:     "Test content",
		CreatedAt:   time.Now(),
	}
	db.SaveNote(note, pass.PassID)

	result := &PolishingResult{
		SectionID:       "section-stats",
		OverallScore:    8.5,
		Consensus:       1,
		Confidence:      0.85,
	}
	db.SaveResult(result, pass.PassID)

	// Test session stats
	stats, err := db.GetSessionStats(session.SessionID)
	if err != nil {
		t.Fatalf("Failed to retrieve session stats: %v", err)
	}

	if totalPasses, ok := stats["total_passes"].(int); !ok || totalPasses != 1 {
		t.Errorf("Expected total_passes=1, got %v", stats["total_passes"])
	}

	if totalNotes, ok := stats["total_notes"].(int); !ok || totalNotes != 1 {
		t.Errorf("Expected total_notes=1, got %v", stats["total_notes"])
	}

	if avgScore, ok := stats["avg_overall_score"].(float64); !ok || avgScore != 8.5 {
		t.Errorf("Expected avg_overall_score=8.5, got %v", stats["avg_overall_score"])
	}
}

func TestPolishingDatabase_ExportSession(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_export_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session and pass
	session := &PolishingSession{
		SessionID:   "test-session-export",
		BookID:      "book-export",
		BookTitle:   "Export Test Book",
		StartedAt:   time.Now(),
		ConfigJSON:  `{"test": "export"}`,
		Status:      "completed",
	}
	db.CreateSession(session)

	pass := &PassRecord{
		PassID:     "test-pass-export",
		SessionID:  session.SessionID,
		PassNumber: 1,
		Providers:  `["openai"]`,
		StartedAt:  time.Now(),
		Status:     "completed",
	}
	db.CreatePass(pass)

	// Test session export
	export, err := db.ExportSession(session.SessionID)
	if err != nil {
		t.Fatalf("Failed to export session: %v", err)
	}

	// Verify export structure
	if export["session"] == nil {
		t.Error("Session data missing from export")
	}

	if export["passes"] == nil {
		t.Error("Passes data missing from export")
	}

	if export["stats"] == nil {
		t.Error("Stats data missing from export")
	}

	// Verify session data
	sessionData := export["session"].(*PolishingSession)
	if sessionData.SessionID != session.SessionID {
		t.Errorf("Expected session ID %s, got %s", session.SessionID, sessionData.SessionID)
	}
}

func TestPolishingDatabase_ErrorHandling(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_errors_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test retrieving non-existent session
	_, err = db.GetSession("non-existent")
	if err == nil {
		t.Error("Expected error when retrieving non-existent session")
	}

	// Test retrieving non-existent pass
	_, err = db.GetPass("non-existent")
	if err == nil {
		t.Error("Expected error when retrieving non-existent pass")
	}

	// Test retrieving notes for non-existent section
	notes, err := db.GetNotesForSection("non-existent")
	if err != nil {
		t.Errorf("Unexpected error retrieving notes for non-existent section: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("Expected 0 notes for non-existent section, got %d", len(notes))
	}
}

// Benchmark tests
func BenchmarkPolishingDatabase_CreateSession(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "bench_session_*.db")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		session := &PolishingSession{
			SessionID:   fmt.Sprintf("session-%d", i),
			BookID:      "book-bench",
			BookTitle:   "Benchmark Book",
			StartedAt:   time.Now(),
			Status:      "running",
		}
		db.CreateSession(session)
	}
}

func BenchmarkPolishingDatabase_SaveResult(b *testing.B) {
	tmpFile, err := os.CreateTemp("", "bench_result_*.db")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewPolishingDatabase(tmpFile.Name())
	if err != nil {
		b.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create session and pass
	session := &PolishingSession{
		SessionID:   "bench-session",
		BookID:      "bench-book",
		BookTitle:   "Benchmark Book",
		StartedAt:   time.Now(),
		Status:      "running",
	}
	db.CreateSession(session)

	pass := &PassRecord{
		PassID:     "bench-pass",
		SessionID:  session.SessionID,
		PassNumber: 1,
		StartedAt:  time.Now(),
		Status:     "running",
	}
	db.CreatePass(pass)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := &PolishingResult{
			SectionID:       fmt.Sprintf("section-%d", i),
			OverallScore:    8.5,
			Consensus:       1,
			Confidence:      0.85,
		}
		db.SaveResult(result, pass.PassID)
	}
}