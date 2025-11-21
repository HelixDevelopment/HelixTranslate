package verification

import (
	"strings"
	"testing"
	"time"
)

// TestNoteCollection tests note collection functionality
func TestNoteCollection(t *testing.T) {
	nc := NewNoteCollection()

	note1 := &LiteraryNote{
		ID:         "note1",
		PassNumber: 1,
		SectionID:  "chapter1",
		NoteType:   NoteTypeCharacter,
		Importance: ImportanceCritical,
		Title:      "Main character development",
		Content:    "Character shows growth",
		CreatedAt:  time.Now(),
	}

	note2 := &LiteraryNote{
		ID:         "note2",
		PassNumber: 1,
		SectionID:  "chapter1",
		NoteType:   NoteTypeTone,
		Importance: ImportanceHigh,
		Title:      "Dark atmosphere",
		Content:    "Maintains ominous tone",
		CreatedAt:  time.Now(),
	}

	nc.Add(note1)
	nc.Add(note2)

	if len(nc.All) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(nc.All))
	}

	sectionNotes := nc.GetForSection("chapter1")
	if len(sectionNotes) != 2 {
		t.Errorf("Expected 2 notes for section, got %d", len(sectionNotes))
	}

	characterNotes := nc.GetByType(NoteTypeCharacter)
	if len(characterNotes) != 1 {
		t.Errorf("Expected 1 character note, got %d", len(characterNotes))
	}

	critical := nc.GetCritical()
	if len(critical) != 1 {
		t.Errorf("Expected 1 critical note, got %d", len(critical))
	}
}

// TestFilterNotesByImportance tests note filtering
func TestFilterNotesByImportance(t *testing.T) {
	notes := []*LiteraryNote{
		{Importance: ImportanceLow, Title: "Low"},
		{Importance: ImportanceMedium, Title: "Medium"},
		{Importance: ImportanceHigh, Title: "High"},
		{Importance: ImportanceCritical, Title: "Critical"},
	}

	// Filter for high+
	filtered := FilterNotesByImportance(notes, ImportanceHigh)
	if len(filtered) != 2 { // High + Critical
		t.Errorf("Expected 2 notes, got %d", len(filtered))
	}

	// Filter for medium+
	filtered = FilterNotesByImportance(notes, ImportanceMedium)
	if len(filtered) != 3 { // Medium + High + Critical
		t.Errorf("Expected 3 notes, got %d", len(filtered))
	}
}

// TestMergeNotes tests note merging
func TestMergeNotes(t *testing.T) {
	notes := []*LiteraryNote{
		{
			SectionID:  "chapter1",
			NoteType:   NoteTypeCharacter,
			Title:      "Main Character",
			Content:    "First observation",
			Importance: ImportanceMedium,
			Examples:   []string{"Example 1"},
		},
		{
			SectionID:  "chapter1",
			NoteType:   NoteTypeCharacter,
			Title:      "Main Character", // Duplicate title
			Content:    "Second observation",
			Importance: ImportanceHigh, // Higher importance
			Examples:   []string{"Example 2"},
		},
		{
			SectionID: "chapter2",
			NoteType:  NoteTypeTone,
			Title:     "Dark Mood",
			Content:   "Different note",
			Importance: ImportanceLow,
		},
	}

	merged := MergeNotes(notes)

	// Should merge first two, keep third
	if len(merged) != 2 {
		t.Errorf("Expected 2 merged notes, got %d", len(merged))
	}

	// Check merged note has higher importance
	var mainCharNote *LiteraryNote
	for _, note := range merged {
		if note.Title == "Main Character" {
			mainCharNote = note
			break
		}
	}

	if mainCharNote == nil {
		t.Error("Main Character note not found after merge")
	} else {
		if mainCharNote.Importance != ImportanceHigh {
			t.Errorf("Expected importance High, got %s", mainCharNote.Importance)
		}
		if len(mainCharNote.Examples) != 2 {
			t.Errorf("Expected 2 examples after merge, got %d", len(mainCharNote.Examples))
		}
	}
}

// TestFormatNotesForContext tests note formatting
func TestFormatNotesForContext(t *testing.T) {
	notes := []*LiteraryNote{
		{
			NoteType:     NoteTypeCharacter,
			Title:        "Protagonist",
			Content:      "Strong character arc",
			Importance:   ImportanceCritical,
			Examples:     []string{"Chapter 1", "Chapter 5"},
			Implications: "Must preserve character voice",
		},
	}

	formatted := FormatNotesForContext(notes)

	if formatted == "" {
		t.Error("Formatted notes should not be empty")
	}

	if !strings.Contains(formatted, "Protagonist") {
		t.Error("Formatted notes should contain title")
	}

	if !strings.Contains(formatted, "Strong character arc") {
		t.Error("Formatted notes should contain content")
	}
}

// TestMultiPassConfig tests configuration structure
func TestMultiPassConfig(t *testing.T) {
	config := MultiPassConfig{
		PassCount: 3,
		PassProviders: [][]string{
			{"deepseek", "anthropic"},
			{"openai", "claude"},
			{"deepseek", "openai"},
		},
		MinConsensus:      2,
		EnableNoteTaking:  true,
		MinNoteImportance: ImportanceHigh,
		CarryNotesForward: true,
		DatabasePath:      "/tmp/polish.db",
	}

	if config.PassCount != 3 {
		t.Errorf("Expected 3 passes, got %d", config.PassCount)
	}

	if len(config.PassProviders) != 3 {
		t.Errorf("Expected 3 provider sets, got %d", len(config.PassProviders))
	}

	if !config.EnableNoteTaking {
		t.Error("Expected note-taking enabled")
	}

	if !config.CarryNotesForward {
		t.Error("Expected notes to carry forward")
	}
}

// TestPolishingSession tests session structure
func TestPolishingSession(t *testing.T) {
	session := &PolishingSession{
		SessionID:   "test-session-123",
		BookID:      "book-456",
		BookTitle:   "Test Book",
		StartedAt:   time.Now(),
		ConfigJSON:  `{"pass_count":2}`,
		TotalPasses: 2,
		Status:      "running",
	}

	if session.SessionID != "test-session-123" {
		t.Errorf("Expected session ID test-session-123, got %s", session.SessionID)
	}

	if session.Status != "running" {
		t.Errorf("Expected status running, got %s", session.Status)
	}
}

// TestPassRecord tests pass record structure
func TestPassRecord(t *testing.T) {
	pass := &PassRecord{
		PassID:     "pass-1",
		SessionID:  "session-1",
		PassNumber: 1,
		Providers:  `["deepseek","anthropic"]`,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	if pass.PassNumber != 1 {
		t.Errorf("Expected pass number 1, got %d", pass.PassNumber)
	}

	if pass.Providers != `["deepseek","anthropic"]` {
		t.Errorf("Unexpected providers: %s", pass.Providers)
	}
}

// TestNoteTypes tests all note types are valid
func TestNoteTypes(t *testing.T) {
	types := []NoteType{
		NoteTypeCharacter,
		NoteTypeTone,
		NoteTypeTheme,
		NoteTypeCulture,
		NoteTypeStyle,
		NoteTypeContext,
		NoteTypeVocabulary,
		NoteTypeStructure,
	}

	if len(types) != 8 {
		t.Errorf("Expected 8 note types, got %d", len(types))
	}

	for _, noteType := range types {
		if string(noteType) == "" {
			t.Error("Note type should not be empty")
		}
	}
}

// TestImportanceLevels tests importance levels
func TestImportanceLevels(t *testing.T) {
	levels := []ImportanceLevel{
		ImportanceLow,
		ImportanceMedium,
		ImportanceHigh,
		ImportanceCritical,
	}

	// Test importance ordering
	if importanceLevel(ImportanceLow) >= importanceLevel(ImportanceMedium) {
		t.Error("Low should be less than Medium")
	}

	if importanceLevel(ImportanceMedium) >= importanceLevel(ImportanceHigh) {
		t.Error("Medium should be less than High")
	}

	if importanceLevel(ImportanceHigh) >= importanceLevel(ImportanceCritical) {
		t.Error("High should be less than Critical")
	}

	if len(levels) != 4 {
		t.Errorf("Expected 4 importance levels, got %d", len(levels))
	}
}

// Benchmark tests

func BenchmarkNoteCollectionAdd(b *testing.B) {
	nc := NewNoteCollection()
	note := &LiteraryNote{
		ID:         "bench-note",
		PassNumber: 1,
		SectionID:  "section1",
		NoteType:   NoteTypeCharacter,
		Importance: ImportanceHigh,
		Title:      "Test",
		Content:    "Test content",
		CreatedAt:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nc.Add(note)
	}
}

func BenchmarkFilterNotes(b *testing.B) {
	notes := make([]*LiteraryNote, 100)
	for i := 0; i < 100; i++ {
		importance := ImportanceLow
		if i%4 == 0 {
			importance = ImportanceCritical
		} else if i%3 == 0 {
			importance = ImportanceHigh
		} else if i%2 == 0 {
			importance = ImportanceMedium
		}

		notes[i] = &LiteraryNote{
			Importance: importance,
			Title:      "Note",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterNotesByImportance(notes, ImportanceHigh)
	}
}

func BenchmarkMergeNotes(b *testing.B) {
	notes := make([]*LiteraryNote, 50)
	for i := 0; i < 50; i++ {
		notes[i] = &LiteraryNote{
			SectionID:  "section1",
			NoteType:   NoteTypeCharacter,
			Title:      "Character Note",
			Content:    "Content",
			Importance: ImportanceMedium,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MergeNotes(notes)
	}
}
