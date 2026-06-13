package report

import (
	"sync"
	"testing"

	"digital.vasic.translator/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReportGenerator_ConcurrentAppends asserts that the report generator's
// recording methods are safe for concurrent use. ReportGenerator is exported
// public API; an SSH/distributed session can record issues, warnings, and log
// entries from multiple goroutines. Concurrent appends to the same slice with
// no synchronization is a data race that can lose entries or corrupt the slice.
func TestReportGenerator_ConcurrentAppends(t *testing.T) {
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(t.TempDir(), sessionLogger)

	const goroutines = 50
	const perGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				generator.AddIssue("translation", "error", "boom", "translator")
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				generator.AddWarning("translation", "warn", "translator", nil)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				generator.AddLogEntry("info", "log", "translator", nil)
			}
		}()
	}
	wg.Wait()

	// Every appended entry must be present — no lost writes from a racy append.
	assert.Len(t, generator.issues, goroutines*perGoroutine)
	assert.Len(t, generator.warnings, goroutines*perGoroutine)
	assert.Len(t, generator.logs, goroutines*perGoroutine)
}

// TestReportGenerator_ConcurrentReadWrite asserts the reader methods
// (GetStats, GenerateSessionReport, ExportLogsToFile) are safe to call while
// other goroutines are recording. A racy reader would either trip the race
// detector or read a torn slice; this also proves the readers do not deadlock
// against the recording mutex.
func TestReportGenerator_ConcurrentReadWrite(t *testing.T) {
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(t.TempDir(), sessionLogger)

	var wg sync.WaitGroup

	// Bounded writers run concurrently with the readers below.
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			generator.AddIssue("translation", "error", "boom", "translator")
			generator.AddLogEntry("info", "log", "translator", nil)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			generator.AddWarning("translation", "warn", "translator", nil)
		}
	}()

	// Readers run concurrently with the writers; each must observe a
	// consistent (non-torn) view and must not deadlock against the recorders.
	for i := 0; i < 30; i++ {
		stats := generator.GetStats()
		assert.NotNil(t, stats["issues_by_severity"])
		require.NoError(t, generator.ExportLogsToFile())
		require.NoError(t, generator.GenerateSessionReport(TranslationSession{InputFile: "a", OutputFile: "b"}))
	}

	wg.Wait()

	// After all writers finished, every recorded entry must be present.
	assert.Len(t, generator.issues, 200)
	assert.Len(t, generator.warnings, 200)
	assert.Len(t, generator.logs, 200)
}
