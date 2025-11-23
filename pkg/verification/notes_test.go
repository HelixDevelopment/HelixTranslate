package verification

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

func TestTranslationNotes_AddNote(t *testing.T) {
	notes := NewTranslationNotes()

	tests := []struct {
		name     string
		noteType NoteType
		content  string
		metadata map[string]interface{}
		expectError bool
	}{
		{
			name:     "valid note",
			noteType: NoteTypeStyle,
			content:  "The translation should be more formal",
			metadata: map[string]interface{}{
				"severity": "medium",
			},
			expectError: false,
		},
		{
			name:     "empty content",
			noteType: NoteTypeStyle,
			content:  "",
			metadata: map[string]interface{}{},
			expectError: true,
		},
		{
			name:     "invalid note type",
			noteType: NoteType("invalid"),
			content:  "Test note",
			metadata: map[string]interface{}{},
			expectError: true,
		},
		{
			name:     "nil metadata",
			noteType: NoteTypeStyle,
			content:  "Test note with nil metadata",
			metadata: nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noteID, err := notes.AddNote(tt.noteType, tt.content, tt.metadata)
			if (err != nil) != tt.expectError {
				t.Errorf("AddNote() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if noteID == "" {
					t.Error("Expected non-empty note ID")
				}

				// Verify note was added
				retrievedNote, exists := notes.GetNote(noteID)
				if !exists {
					t.Error("Note was not added")
				} else {
					if retrievedNote.Type != tt.noteType {
						t.Errorf("Expected note type %v, got %v", tt.noteType, retrievedNote.Type)
					}
					if retrievedNote.Content != tt.content {
						t.Errorf("Expected content %s, got %s", tt.content, retrievedNote.Content)
					}
				}
			}
		})
	}
}

func TestTranslationNotes_GetNotesByType(t *testing.T) {
	notes := NewTranslationNotes()

	// Add test notes
	styleNoteID, _ := notes.AddNote(NoteTypeStyle, "Style issue", map[string]interface{}{"severity": "low"})
	grammarNoteID, _ := notes.AddNote(NoteTypeGrammar, "Grammar issue", map[string]interface{}{"severity": "high"})
	terminologyNoteID, _ := notes.AddNote(NoteTypeTerminology, "Terminology issue", map[string]interface{}{"severity": "medium"})
	anotherStyleNoteID, _ := notes.AddNote(NoteTypeStyle, "Another style issue", map[string]interface{}{"severity": "medium"})

	tests := []struct {
		name         string
		noteType     NoteType
		expectedCount int
		expectedIDs  []string
	}{
		{
			name:         "style notes",
			noteType:     NoteTypeStyle,
			expectedCount: 2,
			expectedIDs:  []string{styleNoteID, anotherStyleNoteID},
		},
		{
			name:         "grammar notes",
			noteType:     NoteTypeGrammar,
			expectedCount: 1,
			expectedIDs:  []string{grammarNoteID},
		},
		{
			name:         "terminology notes",
			noteType:     NoteTypeTerminology,
			expectedCount: 1,
			expectedIDs:  []string{terminologyNoteID},
		},
		{
			name:         "missing type",
			noteType:     NoteType("missing"),
			expectedCount: 0,
			expectedIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundNotes := notes.GetNotesByType(tt.noteType)
			
			if len(foundNotes) != tt.expectedCount {
				t.Errorf("Expected %d notes, got %d", tt.expectedCount, len(foundNotes))
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, note := range foundNotes {
					if note.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected note ID %s not found in results", expectedID)
				}
			}
		})
	}
}

func TestTranslationNotes_UpdateNote(t *testing.T) {
	notes := NewTranslationNotes()

	noteID, err := notes.AddNote(NoteTypeStyle, "Original content", map[string]interface{}{"original": "value"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	tests := []struct {
		name        string
		noteID      string
		newContent  string
		newMetadata map[string]interface{}
		expectError bool
	}{
		{
			name:        "valid update",
			noteID:      noteID,
			newContent:  "Updated content",
			newMetadata: map[string]interface{}{"updated": "value"},
			expectError: false,
		},
		{
			name:        "non-existent note",
			noteID:      "non-existent-id",
			newContent:  "Updated content",
			newMetadata: map[string]interface{}{},
			expectError: true,
		},
		{
			name:        "empty content",
			noteID:      noteID,
			newContent:  "",
			newMetadata: map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := notes.UpdateNote(tt.noteID, tt.newContent, tt.newMetadata)
			if (err != nil) != tt.expectError {
				t.Errorf("UpdateNote() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				note, exists := notes.GetNote(tt.noteID)
				if !exists {
					t.Error("Note should exist after update")
				} else {
					if note.Content != tt.newContent {
						t.Errorf("Expected content %s, got %s", tt.newContent, note.Content)
					}
					
					if tt.newMetadata != nil {
						if val, ok := note.Metadata["updated"]; !ok || val != "value" {
							t.Error("Updated metadata not found")
						}
					}
				}
			}
		})
	}
}

func TestTranslationNotes_DeleteNote(t *testing.T) {
	notes := NewTranslationNotes()

	noteID, err := notes.AddNote(NoteTypeStyle, "Test content", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	// Verify note exists
	_, exists := notes.GetNote(noteID)
	if !exists {
		t.Fatal("Note should exist before deletion")
	}

	err = notes.DeleteNote(noteID)
	if err != nil {
		t.Errorf("DeleteNote() error = %v", err)
	}

	// Verify note no longer exists
	_, exists = notes.GetNote(noteID)
	if exists {
		t.Error("Note should not exist after deletion")
	}

	// Test deleting non-existent note
	err = notes.DeleteNote("non-existent-id")
	if err == nil {
		t.Error("Expected error when deleting non-existent note")
	}
}

func TestTranslationNotes_FilterNotes(t *testing.T) {
	notes := NewTranslationNotes()

	// Add test notes with different metadata
	_, _ = notes.AddNote(NoteTypeStyle, "Low priority", map[string]interface{}{"severity": "low", "category": "style"})
	_, _ = notes.AddNote(NoteTypeStyle, "High priority", map[string]interface{}{"severity": "high", "category": "style"})
	_, _ = notes.AddNote(NoteTypeGrammar, "Medium priority", map[string]interface{}{"severity": "medium", "category": "grammar"})
	_, _ = notes.AddNote(NoteTypeTerminology, "High priority", map[string]interface{}{"severity": "high", "category": "terminology"})

	tests := []struct {
		name           string
		filter         NoteFilter
		expectedCount  int
	}{
		{
			name: "filter by severity high",
			filter: NoteFilter{
				Severity: stringPtr("high"),
			},
			expectedCount: 2,
		},
		{
			name: "filter by category style",
			filter: NoteFilter{
				Category: stringPtr("style"),
			},
			expectedCount: 2,
		},
		{
			name: "filter by type and severity",
			filter: NoteFilter{
				Type: func() *NoteType { t := NoteTypeStyle; return &t }(),
				Severity: stringPtr("high"),
			},
			expectedCount: 1,
		},
		{
			name: "filter by multiple criteria",
			filter: NoteFilter{
				Type: func() *NoteType { t := NoteTypeStyle; return &t }(),
				Severity: stringPtr("low"),
				Category: stringPtr("style"),
			},
			expectedCount: 1,
		},
		{
			name: "no filter",
			filter: NoteFilter{},
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredNotes := notes.FilterNotes(tt.filter)
			
			if len(filteredNotes) != tt.expectedCount {
				t.Errorf("Expected %d notes, got %d", tt.expectedCount, len(filteredNotes))
			}

			// Verify filter criteria
			for _, note := range filteredNotes {
				if tt.filter.Type != nil && note.Type != *tt.filter.Type {
					t.Errorf("Note type mismatch: expected %v, got %v", *tt.filter.Type, note.Type)
				}
				if tt.filter.Severity != nil {
					if severity, ok := note.Metadata["severity"].(string); !ok || severity != *tt.filter.Severity {
						t.Errorf("Severity mismatch: expected %s, got %v", *tt.filter.Severity, note.Metadata["severity"])
					}
				}
				if tt.filter.Category != nil {
					if category, ok := note.Metadata["category"].(string); !ok || category != *tt.filter.Category {
						t.Errorf("Category mismatch: expected %s, got %v", *tt.filter.Category, note.Metadata["category"])
					}
				}
			}
		})
	}
}

func TestTranslationNotes_GetStatistics(t *testing.T) {
	notes := NewTranslationNotes()

	// Add test notes
	_, _ = notes.AddNote(NoteTypeStyle, "Style note 1", map[string]interface{}{"severity": "low"})
	_, _ = notes.AddNote(NoteTypeStyle, "Style note 2", map[string]interface{}{"severity": "high"})
	_, _ = notes.AddNote(NoteTypeGrammar, "Grammar note 1", map[string]interface{}{"severity": "medium"})
	_, _ = notes.AddNote(NoteTypeTerminology, "Terminology note 1", map[string]interface{}{"severity": "high"})

	stats := notes.GetStatistics()

	if stats.TotalNotes != 4 {
		t.Errorf("Expected total notes 4, got %d", stats.TotalNotes)
	}

	expectedCounts := map[NoteType]int{
		NoteTypeStyle:       2,
		NoteTypeGrammar:      1,
		NoteTypeTerminology: 1,
	}

	for noteType, expectedCount := range expectedCounts {
		if count := stats.NotesByType[noteType]; count != expectedCount {
			t.Errorf("Expected %d notes of type %v, got %d", expectedCount, noteType, count)
		}
	}

	expectedSeverityCounts := map[string]int{
		"low":    1,
		"medium": 1,
		"high":   2,
	}

	for severity, expectedCount := range expectedSeverityCounts {
		if count := stats.NotesBySeverity[severity]; count != expectedCount {
			t.Errorf("Expected %d notes with severity %s, got %d", expectedCount, severity, count)
		}
	}
}

func TestTranslationNotes_ExportImport(t *testing.T) {
	originalNotes := NewTranslationNotes()

	// Add test notes
	note1ID, _ := originalNotes.AddNote(NoteTypeStyle, "Style note", map[string]interface{}{"severity": "low"})
	note2ID, _ := originalNotes.AddNote(NoteTypeGrammar, "Grammar note", map[string]interface{}{"severity": "high"})

	// Export notes
	exportedData, err := originalNotes.Export()
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(exportedData) != 2 {
		t.Errorf("Expected 2 exported notes, got %d", len(exportedData))
	}

	// Import to new notes instance
	importedNotes := NewTranslationNotes()
	err = importedNotes.Import(exportedData)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	// Verify imported notes
	importedNote1, exists := importedNotes.GetNote(note1ID)
	if !exists {
		t.Error("Imported note 1 not found")
	} else {
		if importedNote1.Type != NoteTypeStyle || importedNote1.Content != "Style note" {
			t.Error("Imported note 1 content mismatch")
		}
	}

	importedNote2, exists := importedNotes.GetNote(note2ID)
	if !exists {
		t.Error("Imported note 2 not found")
	} else {
		if importedNote2.Type != NoteTypeGrammar || importedNote2.Content != "Grammar note" {
			t.Error("Imported note 2 content mismatch")
		}
	}
}

func TestTranslationNotes_ConcurrentOperations(t *testing.T) {
	notes := NewTranslationNotes()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numGoroutines := 10
	numNotesPerGoroutine := 5

	errChan := make(chan error, numGoroutines)
	noteIDChan := make(chan []string, numGoroutines)

	// Concurrent note addition
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			var noteIDs []string
			for j := 0; j < numNotesPerGoroutine; j++ {
				noteID, err := notes.AddNote(NoteTypeStyle, fmt.Sprintf("Note %d-%d", goroutineID, j), map[string]interface{}{"goroutine": goroutineID})
				if err != nil {
					errChan <- err
					return
				}
				noteIDs = append(noteIDs, noteID)
			}
			noteIDChan <- noteIDs
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errChan:
			t.Errorf("Concurrent note addition error: %v", err)
		case <-noteIDChan:
			// Successfully completed
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}

	// Verify total number of notes
	stats := notes.GetStatistics()
	expectedTotalNotes := numGoroutines * numNotesPerGoroutine
	if stats.TotalNotes != expectedTotalNotes {
		t.Errorf("Expected %d total notes, got %d", expectedTotalNotes, stats.TotalNotes)
	}
}

func TestTranslationNotes_NoteTypes(t *testing.T) {
	tests := []struct {
		name     string
		noteType NoteType
		valid    bool
	}{
		{"style", NoteTypeStyle, true},
		{"grammar", NoteTypeGrammar, true},
		{"terminology", NoteTypeTerminology, true},
		{"consistency", NoteTypeConsistency, true},
		{"accuracy", NoteTypeAccuracy, true},
		{"fluency", NoteTypeFluency, true},
		{"cultural", NoteTypeCultural, true},
		{"invalid", NoteType("invalid"), false},
		{"empty", NoteType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := NewTranslationNotes()
			
			_, err := notes.AddNote(tt.noteType, "Test content", map[string]interface{}{})
			
			if tt.valid && err != nil {
				t.Errorf("Expected no error for valid note type %v, got: %v", tt.noteType, err)
			}
			
			if !tt.valid && err == nil {
				t.Errorf("Expected error for invalid note type %v", tt.noteType)
			}
		})
	}
}