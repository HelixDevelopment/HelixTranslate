package progress

import (
	"strconv"
	"sync"
	"time"
)

// TranslationProgress tracks detailed translation progress
type TranslationProgress struct {
	// Book information
	BookTitle      string `json:"book_title"`
	TotalChapters  int    `json:"total_chapters"`
	CurrentChapter int    `json:"current_chapter"`
	ChapterTitle   string `json:"chapter_title"`
	CurrentSection int    `json:"current_section"`
	TotalSections  int    `json:"total_sections"`

	// Progress metrics
	PercentComplete float64 `json:"percent_complete"`
	ItemsTotal      int     `json:"items_total"`
	ItemsCompleted  int     `json:"items_completed"`
	ItemsFailed     int     `json:"items_failed"`

	// Time tracking
	StartTime    time.Time `json:"start_time"`
	ElapsedTime  string    `json:"elapsed_time"`
	EstimatedETA string    `json:"estimated_eta"`

	// Translation details
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`

	// Status
	Status      string `json:"status"` // "initializing", "translating", "completed", "error"
	CurrentTask string `json:"current_task"`
	SessionID   string `json:"session_id"`
}

// Tracker manages translation progress
type Tracker struct {
	mu       sync.RWMutex
	progress *TranslationProgress
}

// NewTracker creates a new progress tracker
func NewTracker(sessionID, bookTitle string, totalChapters int, sourceLanguage, targetLanguage, provider, model string) *Tracker {
	return &Tracker{
		progress: &TranslationProgress{
			SessionID:      sessionID,
			BookTitle:      bookTitle,
			TotalChapters:  totalChapters,
			SourceLanguage: sourceLanguage,
			TargetLanguage: targetLanguage,
			Provider:       provider,
			Model:          model,
			StartTime:      time.Now(),
			Status:         "initializing",
		},
	}
}

// UpdateChapter updates the current chapter being translated
func (t *Tracker) UpdateChapter(chapterNum int, chapterTitle string, totalSections int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.CurrentChapter = chapterNum
	t.progress.ChapterTitle = chapterTitle
	t.progress.TotalSections = totalSections
	t.progress.CurrentSection = 0
	t.progress.Status = "translating"
	t.progress.CurrentTask = "Translating chapter " + chapterTitle

	t.updateProgress()
}

// UpdateSection updates the current section being translated
func (t *Tracker) UpdateSection(sectionNum int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.CurrentSection = sectionNum
	t.updateProgress()
}

// IncrementCompleted increments the completed items counter
func (t *Tracker) IncrementCompleted() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.ItemsCompleted++
	t.updateProgress()
}

// IncrementFailed increments the failed items counter
func (t *Tracker) IncrementFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.ItemsFailed++
	t.updateProgress()
}

// SetTotal sets the total number of items to translate
func (t *Tracker) SetTotal(total int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.ItemsTotal = total
}

// SetStatus updates the status
func (t *Tracker) SetStatus(status, task string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Status = status
	t.progress.CurrentTask = task
}

// Complete marks the translation as completed
func (t *Tracker) Complete() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Status = "completed"
	t.progress.CurrentTask = "Translation completed"
	t.updateProgress()
	// Force 100% AFTER updateProgress: a Complete() before any chapter update
	// leaves CurrentChapter=0, so the chapter-based recompute would otherwise
	// reset PercentComplete to 0 and clobber the completion state.
	t.progress.PercentComplete = 100.0
	t.progress.EstimatedETA = "Completed"
}

// Error marks the translation as errored
func (t *Tracker) Error(errorMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Status = "error"
	t.progress.CurrentTask = "Error: " + errorMsg
}

// GetProgress returns a copy of the current progress
func (t *Tracker) GetProgress() TranslationProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Take a snapshot copy first; derived time fields are computed on the COPY,
	// never on the shared struct. Writing t.progress here would be a data race
	// because RLock permits multiple concurrent readers.
	snapshot := *t.progress

	elapsed := time.Since(snapshot.StartTime)
	snapshot.ElapsedTime = formatDuration(elapsed)

	// Only refine the ETA from item counts while the run is genuinely IN
	// PROGRESS. Once it is complete (100% / status "completed"), updateProgress
	// already set EstimatedETA="Completed"; recomputing the items projection
	// here would clobber that with a stale/empty remaining-time value, so a
	// finished translation would report a non-"Completed" ETA to the dashboard.
	if snapshot.ItemsCompleted > 0 && snapshot.ItemsTotal > 0 &&
		snapshot.PercentComplete < 100.0 && snapshot.Status != "completed" {
		avgTimePerItem := elapsed / time.Duration(snapshot.ItemsCompleted)
		remainingItems := snapshot.ItemsTotal - snapshot.ItemsCompleted
		if remainingItems < 0 {
			remainingItems = 0
		}
		estimatedRemaining := avgTimePerItem * time.Duration(remainingItems)
		snapshot.EstimatedETA = formatDuration(estimatedRemaining)
	}

	return snapshot
}

// updateProgress calculates percentage and updates progress (must be called with lock held)
func (t *Tracker) updateProgress() {
	if t.progress.TotalChapters > 0 {
		// Completed chapters = CurrentChapter-1, but never negative (CurrentChapter
		// is 0 before any chapter update, which would otherwise yield a negative
		// percentage).
		completedChapters := t.progress.CurrentChapter - 1
		if completedChapters < 0 {
			completedChapters = 0
		}
		t.progress.PercentComplete = float64(completedChapters) / float64(t.progress.TotalChapters) * 100.0

		// Add section progress within current chapter
		if t.progress.TotalSections > 0 {
			sectionPercent := float64(t.progress.CurrentSection) / float64(t.progress.TotalSections) / float64(t.progress.TotalChapters) * 100.0
			t.progress.PercentComplete += sectionPercent
		}

		// Clamp to [0, 100].
		if t.progress.PercentComplete > 100.0 {
			t.progress.PercentComplete = 100.0
		}
		if t.progress.PercentComplete < 0.0 {
			t.progress.PercentComplete = 0.0
		}
	}

	// Update elapsed time
	elapsed := time.Since(t.progress.StartTime)
	t.progress.ElapsedTime = formatDuration(elapsed)

	// Calculate ETA. Use float arithmetic for the projection: converting a
	// fractional percentage directly to time.Duration truncates it (e.g.
	// time.Duration(2.5)==2, time.Duration(0.5)==0 which would divide-by-zero),
	// producing wrong ETAs or a panic.
	if t.progress.PercentComplete > 0 && t.progress.PercentComplete < 100 {
		totalEstimated := time.Duration(float64(elapsed) * (100.0 / t.progress.PercentComplete))
		remaining := totalEstimated - elapsed
		t.progress.EstimatedETA = formatDuration(remaining)
	} else if t.progress.PercentComplete >= 100 {
		t.progress.EstimatedETA = "Completed"
	} else {
		t.progress.EstimatedETA = "Calculating..."
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	// Join only the non-empty unit parts. formatTime returns "" for a zero
	// value, so an exact 5h (minutes==0) or exact 2m (seconds==0) must NOT
	// render with a trailing space ("5 hours ") — these strings are surfaced
	// verbatim to the dashboard as EstimatedETA / ElapsedTime.
	if hours > 0 {
		return joinNonEmpty(formatTime(hours, "hour"), formatTime(minutes, "minute"))
	} else if minutes > 0 {
		return joinNonEmpty(formatTime(minutes, "minute"), formatTime(seconds, "second"))
	} else {
		return formatTime(seconds, "second")
	}
}

// joinNonEmpty joins the larger- and smaller-unit fragments with a single
// space, dropping any empty fragment so no leading/trailing/double space is
// produced.
func joinNonEmpty(major, minor string) string {
	if major == "" {
		return minor
	}
	if minor == "" {
		return major
	}
	return major + " " + minor
}

// formatTime formats a time value with proper singular/plural
func formatTime(value int, unit string) string {
	if value == 0 {
		return ""
	}
	if value == 1 {
		return "1 " + unit
	}
	return formatInt(value) + " " + unit + "s"
}

// formatInt formats an integer
func formatInt(n int) string {
	return strconv.Itoa(n)
}
