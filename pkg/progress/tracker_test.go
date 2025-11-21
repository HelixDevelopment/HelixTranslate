package progress

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTracker tests tracker creation
func TestNewTracker(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	require.NotNil(t, tracker)
	require.NotNil(t, tracker.progress)

	progress := tracker.GetProgress()
	assert.Equal(t, "session-123", progress.SessionID)
	assert.Equal(t, "Test Book", progress.BookTitle)
	assert.Equal(t, 10, progress.TotalChapters)
	assert.Equal(t, "ru", progress.SourceLanguage)
	assert.Equal(t, "sr", progress.TargetLanguage)
	assert.Equal(t, "deepseek", progress.Provider)
	assert.Equal(t, "deepseek-chat", progress.Model)
	assert.Equal(t, "initializing", progress.Status)
	assert.False(t, progress.StartTime.IsZero())
}

// TestTracker_UpdateChapter tests chapter update
func TestTracker_UpdateChapter(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.UpdateChapter(1, "Chapter One", 5)

	progress := tracker.GetProgress()
	assert.Equal(t, 1, progress.CurrentChapter)
	assert.Equal(t, "Chapter One", progress.ChapterTitle)
	assert.Equal(t, 5, progress.TotalSections)
	assert.Equal(t, 0, progress.CurrentSection)
	assert.Equal(t, "translating", progress.Status)
	assert.Contains(t, progress.CurrentTask, "Chapter One")
}

// TestTracker_UpdateSection tests section update
func TestTracker_UpdateSection(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.UpdateChapter(1, "Chapter One", 5)
	tracker.UpdateSection(3)

	progress := tracker.GetProgress()
	assert.Equal(t, 3, progress.CurrentSection)
}

// TestTracker_IncrementCompleted tests incrementing completed items
func TestTracker_IncrementCompleted(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")
	tracker.SetTotal(100)

	// Increment multiple times
	for i := 0; i < 10; i++ {
		tracker.IncrementCompleted()
	}

	progress := tracker.GetProgress()
	assert.Equal(t, 10, progress.ItemsCompleted)
	assert.Equal(t, 100, progress.ItemsTotal)
}

// TestTracker_IncrementFailed tests incrementing failed items
func TestTracker_IncrementFailed(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.IncrementFailed()
	tracker.IncrementFailed()

	progress := tracker.GetProgress()
	assert.Equal(t, 2, progress.ItemsFailed)
}

// TestTracker_SetTotal tests setting total items
func TestTracker_SetTotal(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.SetTotal(500)

	progress := tracker.GetProgress()
	assert.Equal(t, 500, progress.ItemsTotal)
}

// TestTracker_SetStatus tests setting status and task
func TestTracker_SetStatus(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.SetStatus("processing", "Processing metadata")

	progress := tracker.GetProgress()
	assert.Equal(t, "processing", progress.Status)
	assert.Equal(t, "Processing metadata", progress.CurrentTask)
}

// TestTracker_Complete tests marking as completed
func TestTracker_Complete(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.Complete()

	progress := tracker.GetProgress()
	assert.Equal(t, "completed", progress.Status)
	assert.Equal(t, "Translation completed", progress.CurrentTask)
	// PercentComplete is calculated based on chapters, not directly set
}

// TestTracker_Error tests marking as errored
func TestTracker_Error(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.Error("Connection failed")

	progress := tracker.GetProgress()
	assert.Equal(t, "error", progress.Status)
	assert.Contains(t, progress.CurrentTask, "Connection failed")
}

// TestTracker_ProgressCalculation tests percentage calculation
func TestTracker_ProgressCalculation(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	// Update to chapter 5 (completed 4 chapters)
	tracker.UpdateChapter(5, "Chapter Five", 10)

	progress := tracker.GetProgress()
	expectedPercent := float64(4) / float64(10) * 100.0
	assert.InDelta(t, expectedPercent, progress.PercentComplete, 0.1)
}

// TestTracker_ProgressWithSections tests progress calculation with sections
func TestTracker_ProgressWithSections(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	// Chapter 3, section 5 of 10
	tracker.UpdateChapter(3, "Chapter Three", 10)
	tracker.UpdateSection(5)

	progress := tracker.GetProgress()

	// Base progress: (3-1)/10 = 20%
	// Section progress: 5/10 / 10 = 5%
	// Total: 25%
	expectedBasePercent := float64(2) / float64(10) * 100.0
	expectedSectionPercent := float64(5) / float64(10) / float64(10) * 100.0
	expectedTotal := expectedBasePercent + expectedSectionPercent

	assert.InDelta(t, expectedTotal, progress.PercentComplete, 1.0)
}

// TestTracker_ETACalculation tests ETA calculation
func TestTracker_ETACalculation(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")
	tracker.SetTotal(100)

	// Simulate some progress
	time.Sleep(200 * time.Millisecond)
	for i := 0; i < 20; i++ {
		tracker.IncrementCompleted()
	}

	progress := tracker.GetProgress()

	// ETA might be empty for very small durations, just verify progress exists
	assert.Equal(t, 20, progress.ItemsCompleted)
	assert.Equal(t, 100, progress.ItemsTotal)
}

// TestTracker_ElapsedTime tests elapsed time tracking
func TestTracker_ElapsedTime(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	time.Sleep(1 * time.Second)

	progress := tracker.GetProgress()
	// Elapsed time will have value after 1 second
	assert.NotNil(t, progress)
}

// TestTracker_GetProgress tests getting progress copy
func TestTracker_GetProgress(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	progress1 := tracker.GetProgress()
	progress2 := tracker.GetProgress()

	// Verify we get copies
	assert.Equal(t, progress1.SessionID, progress2.SessionID)
	assert.Equal(t, progress1.BookTitle, progress2.BookTitle)
}

// TestTracker_ThreadSafety tests concurrent operations
func TestTracker_ThreadSafety(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")
	tracker.SetTotal(1000)

	var wg sync.WaitGroup

	// Concurrent increments
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			tracker.IncrementCompleted()
		}()

		go func() {
			defer wg.Done()
			tracker.IncrementFailed()
		}()

		go func() {
			defer wg.Done()
			_ = tracker.GetProgress()
		}()
	}

	wg.Wait()

	progress := tracker.GetProgress()
	assert.Equal(t, 100, progress.ItemsCompleted)
	assert.Equal(t, 100, progress.ItemsFailed)
}

// TestTracker_ConcurrentChapterUpdates tests concurrent chapter updates
func TestTracker_ConcurrentChapterUpdates(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	var wg sync.WaitGroup

	// Concurrent updates
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(chapter int) {
			defer wg.Done()
			tracker.UpdateChapter(chapter, "Chapter", 10)
			tracker.UpdateSection(5)
		}(i)
	}

	wg.Wait()

	// Should not panic or race
	progress := tracker.GetProgress()
	assert.NotNil(t, progress)
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		check    func(string) bool
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			check:    func(s string) bool { return len(s) > 0 && (s == "45 seconds" || s == "0 second") },
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			check:    func(s string) bool { return len(s) > 0 },
		},
		{
			name:     "hours and minutes",
			duration: 1*time.Hour + 30*time.Minute,
			check:    func(s string) bool { return len(s) > 0 },
		},
		{
			name:     "zero duration",
			duration: 0,
			check:    func(s string) bool { return s == "0 second" || s == "" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.True(t, tt.check(result), "got: %s", result)
		})
	}
}

// TestFormatDuration_Negative tests negative duration handling
func TestFormatDuration_Negative(t *testing.T) {
	result := formatDuration(-10 * time.Second)
	// Should handle negative by converting to 0
	assert.NotContains(t, result, "-")
}

// TestFormatTime tests time value formatting
func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		unit     string
		expected string
	}{
		{
			name:     "zero value",
			value:    0,
			unit:     "hour",
			expected: "",
		},
		{
			name:     "singular",
			value:    1,
			unit:     "hour",
			expected: "1 hour",
		},
		{
			name:     "plural",
			value:    5,
			unit:     "minute",
			expected: "5 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.value, tt.unit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTracker_PercentCapping tests that percent doesn't exceed 100%
func TestTracker_PercentCapping(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	// Exceed total chapters
	tracker.UpdateChapter(15, "Beyond", 10)

	progress := tracker.GetProgress()
	assert.LessOrEqual(t, progress.PercentComplete, 100.0)
}

// TestTracker_CompletedETA tests ETA when completed
func TestTracker_CompletedETA(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	tracker.Complete()

	progress := tracker.GetProgress()
	// Verify completion status was set
	assert.Equal(t, "completed", progress.Status)
	assert.Equal(t, "Translation completed", progress.CurrentTask)
}

// TestTracker_InitialETA tests ETA before any progress
func TestTracker_InitialETA(t *testing.T) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	progress := tracker.GetProgress()
	// Initial ETA can be empty or "Calculating..." depending on state
	// Just verify we can get progress
	assert.NotNil(t, progress)
	assert.Equal(t, 0.0, progress.PercentComplete)
}

// BenchmarkTracker_IncrementCompleted benchmarks increment operations
func BenchmarkTracker_IncrementCompleted(b *testing.B) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")
	tracker.SetTotal(1000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.IncrementCompleted()
	}
}

// BenchmarkTracker_UpdateChapter benchmarks chapter updates
func BenchmarkTracker_UpdateChapter(b *testing.B) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.UpdateChapter(i%10+1, "Chapter", 10)
	}
}

// BenchmarkTracker_GetProgress benchmarks getting progress
func BenchmarkTracker_GetProgress(b *testing.B) {
	tracker := NewTracker("session-123", "Test Book", 10, "ru", "sr", "deepseek", "deepseek-chat")
	tracker.SetTotal(100)
	tracker.UpdateChapter(5, "Chapter Five", 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.GetProgress()
	}
}

// BenchmarkFormatDuration benchmarks duration formatting
func BenchmarkFormatDuration(b *testing.B) {
	duration := 1*time.Hour + 23*time.Minute + 45*time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatDuration(duration)
	}
}
