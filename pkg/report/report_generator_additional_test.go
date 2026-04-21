package report

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"digital.vasic.translator/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportGenerator_GenerateLogArchive(t *testing.T) {
	t.Run("GenerateLogArchive calls CopyLogFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		// Should not panic and should return nil
		err := generator.GenerateLogArchive()
		require.NoError(t, err)
	})
}

func TestReportGenerator_copyLogFile(t *testing.T) {
	t.Run("Copy existing log file", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		// Create a source log file
		sourceDir := t.TempDir()
		sourcePath := filepath.Join(sourceDir, "test.log")
		content := []byte("test log content")
		err := os.WriteFile(sourcePath, content, 0644)
		require.NoError(t, err)

		// Copy the file
		err = generator.copyLogFile(sourcePath)
		require.NoError(t, err)

		// Verify the file was copied
		destPath := filepath.Join(tmpDir, "test.log")
		copiedContent, err := os.ReadFile(destPath)
		require.NoError(t, err)
		assert.Equal(t, content, copiedContent)
	})

	t.Run("Copy non-existent file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		err := generator.copyLogFile("/nonexistent/path/file.log")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read log file")
	})

	t.Run("Copy file to read-only destination returns error", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := t.TempDir()
		err := os.Chmod(readOnlyDir, 0555)
		require.NoError(t, err)
		defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup

		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(readOnlyDir, sessionLogger)

		sourcePath := filepath.Join(t.TempDir(), "test.log")
		err = os.WriteFile(sourcePath, []byte("content"), 0644)
		require.NoError(t, err)

		err = generator.copyLogFile(sourcePath)
		assert.Error(t, err)
	})
}

func TestReportGenerator_CopyLogFiles_EdgeCases(t *testing.T) {
	t.Run("No log files found adds warnings", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		// Change to empty temp dir so no log files are found
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		err = generator.CopyLogFiles(context.Background())
		require.NoError(t, err)

		// Should have warnings for missing log files
		assert.GreaterOrEqual(t, len(generator.warnings), 1)
	})

	t.Run("Some log files copied, some missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		// Create one log file
		err := os.WriteFile(filepath.Join(tmpDir, "translator.log"), []byte("translator log"), 0644)
		require.NoError(t, err)

		// Change to temp dir
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		err = generator.CopyLogFiles(context.Background())
		require.NoError(t, err)

		// Should have the copied file
		_, err = os.Stat(filepath.Join(tmpDir, "translator.log"))
		require.NoError(t, err)
	})
}

func TestReportGenerator_GenerateSessionReport_EdgeCases(t *testing.T) {
	t.Run("Empty session with no issues/warnings/logs", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		session := TranslationSession{
			InputFile:  "test.fb2",
			OutputFile: "output.fb2",
			Success:    true,
		}

		err := generator.GenerateSessionReport(session)
		require.NoError(t, err)

		reportPath := filepath.Join(tmpDir, "translation_report.md")
		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)

		// Should not contain issues/warnings/log sections when empty
		reportStr := string(content)
		assert.NotContains(t, reportStr, "## Issues Encountered")
		assert.NotContains(t, reportStr, "## Warnings")
		assert.NotContains(t, reportStr, "## Log Summary")
	})

	t.Run("Session with many log entries tests recent entries limit", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		// Add 30 log entries
		for i := 0; i < 30; i++ {
			generator.AddLogEntry("info", "Log entry", "test", nil)
		}

		session := TranslationSession{
			InputFile:  "test.fb2",
			OutputFile: "output.fb2",
			Success:    true,
		}

		err := generator.GenerateSessionReport(session)
		require.NoError(t, err)

		reportPath := filepath.Join(tmpDir, "translation_report.md")
		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "Recent Log Entries (Last 20)")
	})

	t.Run("Session with resolved and unresolved issues", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		generator.AddIssue("translation", "error", "Error 1", "translator")
		generator.AddIssue("setup", "critical", "Error 2", "ssh")
		generator.ResolveIssue(0, "Fixed")

		session := TranslationSession{
			InputFile:  "test.fb2",
			OutputFile: "output.fb2",
			Success:    false,
		}

		err := generator.GenerateSessionReport(session)
		require.NoError(t, err)

		reportPath := filepath.Join(tmpDir, "translation_report.md")
		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)

		reportStr := string(content)
		assert.Contains(t, reportStr, "✅ Resolved")
		assert.Contains(t, reportStr, "❌ Open")
	})
}

func TestReportGenerator_GetStats_EdgeCases(t *testing.T) {
	t.Run("Multiple issues same severity and category", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		generator.AddIssue("translation", "error", "Error 1", "t1")
		generator.AddIssue("translation", "error", "Error 2", "t2")
		generator.AddIssue("translation", "error", "Error 3", "t3")

		stats := generator.GetStats()
		severityCount := stats["issues_by_severity"].(map[string]int)
		categoryCount := stats["issues_by_category"].(map[string]int)

		assert.Equal(t, 3, severityCount["error"])
		assert.Equal(t, 3, categoryCount["translation"])
	})

	t.Run("Issues with unknown severity", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		generator := NewReportGenerator(tmpDir, sessionLogger)

		generator.AddIssue("test", "info", "Info issue", "test")

		stats := generator.GetStats()
		severityCount := stats["issues_by_severity"].(map[string]int)

		assert.Equal(t, 1, severityCount["info"])
	})
}
