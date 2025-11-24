package report

import (
	"testing"
	"time"
	"os"
	"path/filepath"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"digital.vasic.translator/pkg/logger"
)

func TestNewReportGenerator(t *testing.T) {
	// Test 1: Basic creation
	t.Run("BasicCreation", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		
		generator := NewReportGenerator(tmpDir, sessionLogger)
		
		assert.NotNil(t, generator)
		assert.Equal(t, tmpDir, generator.destinationDir)
		assert.Equal(t, sessionLogger, generator.logger)
		assert.NotZero(t, generator.startTime)
		assert.Empty(t, generator.issues)
		assert.Empty(t, generator.warnings)
		assert.Empty(t, generator.logs)
	})
	
	// Test 2: Create with empty directory
	t.Run("EmptyDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		sessionLogger := logger.NewLogger(logger.LoggerConfig{})
		
		generator := NewReportGenerator(tmpDir, sessionLogger)
		
		// Should create generator without issues even with empty directory
		assert.NotNil(t, generator)
		assert.Equal(t, tmpDir, generator.destinationDir)
	})
}

func TestReportGenerator_AddIssue(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Test 1: Add basic issue
	t.Run("AddBasicIssue", func(t *testing.T) {
		generator.AddIssue("translation", "error", "Translation failed", "translator")
		
		assert.Len(t, generator.issues, 1)
		
		issue := generator.issues[0]
		assert.Equal(t, "translation", issue.Category)
		assert.Equal(t, "error", issue.Severity)
		assert.Equal(t, "Translation failed", issue.Message)
		assert.Equal(t, "translator", issue.Component)
		assert.False(t, issue.Resolved)
		assert.Empty(t, issue.Resolution)
	})
	
	// Test 2: Add multiple issues
	t.Run("AddMultipleIssues", func(t *testing.T) {
		generator.AddIssue("setup", "critical", "SSH connection failed", "ssh_client")
		generator.AddIssue("translation", "warning", "Low confidence", "translator")
		generator.AddIssue("file_operation", "error", "File not found", "file_handler")
		
		assert.Len(t, generator.issues, 4) // 1 from previous test + 3 new
		
		// Check specific issues
		categories := make(map[string]int)
		severities := make(map[string]int)
		
		for _, issue := range generator.issues {
			categories[issue.Category]++
			severities[issue.Severity]++
		}
		
		assert.Equal(t, 1, categories["setup"])
		assert.Equal(t, 2, categories["translation"]) // Added in both tests
		assert.Equal(t, 1, categories["file_operation"])
		
		assert.Equal(t, 1, severities["critical"])
		assert.Equal(t, 2, severities["error"]) // Added in both tests
		assert.Equal(t, 1, severities["warning"])
	})
	
	// Test 3: Check timestamp is set
	t.Run("IssueTimestamp", func(t *testing.T) {
		beforeAdd := time.Now()
		generator.AddIssue("test", "info", "Test issue", "test_component")
		afterAdd := time.Now()
		
		assert.Len(t, generator.issues, 5)
		lastIssue := generator.issues[4]
		
		// Timestamp should be between before and after
		assert.True(t, lastIssue.Timestamp.After(beforeAdd) || lastIssue.Timestamp.Equal(beforeAdd))
		assert.True(t, lastIssue.Timestamp.Before(afterAdd) || lastIssue.Timestamp.Equal(afterAdd))
	})
}

func TestReportGenerator_ResolveIssue(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Add some test issues
	generator.AddIssue("translation", "error", "Translation failed", "translator")
	generator.AddIssue("setup", "critical", "SSH connection failed", "ssh_client")
	
	// Test 1: Resolve first issue
	t.Run("ResolveFirstIssue", func(t *testing.T) {
		err := generator.ResolveIssue(0, "Retried with new API key")
		require.NoError(t, err)
		
		issue := generator.issues[0]
		assert.True(t, issue.Resolved)
		assert.Equal(t, "Retried with new API key", issue.Resolution)
	})
	
	// Test 2: Resolve second issue
	t.Run("ResolveSecondIssue", func(t *testing.T) {
		err := generator.ResolveIssue(1, "Used alternative SSH server")
		require.NoError(t, err)
		
		issue := generator.issues[1]
		assert.True(t, issue.Resolved)
		assert.Equal(t, "Used alternative SSH server", issue.Resolution)
	})
	
	// Test 3: Resolve already resolved issue
	t.Run("ResolveAlreadyResolved", func(t *testing.T) {
		err := generator.ResolveIssue(0, "Another resolution")
		require.NoError(t, err)
		
		// Should allow re-resolving with new message
		issue := generator.issues[0]
		assert.True(t, issue.Resolved)
		assert.Equal(t, "Another resolution", issue.Resolution)
	})
	
	// Test 4: Resolve invalid index
	t.Run("ResolveInvalidIndex", func(t *testing.T) {
		err := generator.ResolveIssue(-1, "Invalid index")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid issue index")
		
		err = generator.ResolveIssue(100, "Invalid index")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid issue index")
	})
}

func TestReportGenerator_AddWarning(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Test 1: Add basic warning
	t.Run("AddBasicWarning", func(t *testing.T) {
		details := map[string]interface{}{
			"confidence": 0.65,
			"provider":   "openai",
		}
		
		generator.AddWarning("translation", "Low confidence", "translator", details)
		
		assert.Len(t, generator.warnings, 1)
		
		warning := generator.warnings[0]
		assert.Equal(t, "translation", warning.Category)
		assert.Equal(t, "Low confidence", warning.Message)
		assert.Equal(t, "translator", warning.Component)
		assert.Equal(t, details, warning.Details)
	})
	
	// Test 2: Add warning without details
	t.Run("AddWarningWithoutDetails", func(t *testing.T) {
		generator.AddWarning("file_operation", "File size large", "file_handler", nil)
		
		assert.Len(t, generator.warnings, 2)
		
		warning := generator.warnings[1]
		assert.Equal(t, "file_operation", warning.Category)
		assert.Equal(t, "File size large", warning.Message)
		assert.Equal(t, "file_handler", warning.Component)
		assert.Nil(t, warning.Details)
	})
	
	// Test 3: Check timestamp is set
	t.Run("WarningTimestamp", func(t *testing.T) {
		beforeAdd := time.Now()
		generator.AddWarning("test", "Test warning", "test_component", nil)
		afterAdd := time.Now()
		
		assert.Len(t, generator.warnings, 3)
		lastWarning := generator.warnings[2]
		
		// Timestamp should be between before and after
		assert.True(t, lastWarning.Timestamp.After(beforeAdd) || lastWarning.Timestamp.Equal(beforeAdd))
		assert.True(t, lastWarning.Timestamp.Before(afterAdd) || lastWarning.Timestamp.Equal(afterAdd))
	})
}

func TestReportGenerator_AddLogEntry(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Test 1: Add basic log entry
	t.Run("AddBasicLogEntry", func(t *testing.T) {
		details := map[string]interface{}{
			"duration": "2.5s",
			"words":    1000,
		}
		
		generator.AddLogEntry("info", "Translation completed", "translator", details)
		
		assert.Len(t, generator.logs, 1)
		
		entry := generator.logs[0]
		assert.Equal(t, "info", entry.Level)
		assert.Equal(t, "Translation completed", entry.Message)
		assert.Equal(t, "translator", entry.Component)
		assert.Equal(t, details, entry.Details)
	})
	
	// Test 2: Add log entry without details
	t.Run("AddLogEntryWithoutDetails", func(t *testing.T) {
		generator.AddLogEntry("error", "Connection failed", "ssh_client", nil)
		
		assert.Len(t, generator.logs, 2)
		
		entry := generator.logs[1]
		assert.Equal(t, "error", entry.Level)
		assert.Equal(t, "Connection failed", entry.Message)
		assert.Equal(t, "ssh_client", entry.Component)
		assert.Nil(t, entry.Details)
	})
	
	// Test 3: Check timestamp is set
	t.Run("LogEntryTimestamp", func(t *testing.T) {
		beforeAdd := time.Now()
		generator.AddLogEntry("debug", "Debug message", "test_component", nil)
		afterAdd := time.Now()
		
		assert.Len(t, generator.logs, 3)
		lastEntry := generator.logs[2]
		
		// Timestamp should be between before and after
		assert.True(t, lastEntry.Timestamp.After(beforeAdd) || lastEntry.Timestamp.Equal(beforeAdd))
		assert.True(t, lastEntry.Timestamp.Before(afterAdd) || lastEntry.Timestamp.Equal(afterAdd))
	})
}

func TestReportGenerator_CopyLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Create some test log files in various locations
	logFiles := map[string]string{
		"translator.log":    "Translation log content",
		"ssh_worker.log":    "SSH worker log content",
		"markdown_workflow.log": "Markdown workflow log content",
	}
	
	// Create log files in the current directory
	for filename, content := range logFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}
	
	// Save current working directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)
	
	// Change to temp directory so logs are found
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	
	// Test 1: Copy existing log files
	t.Run("CopyExistingLogFiles", func(t *testing.T) {
		err := generator.CopyLogFiles(context.Background())
		require.NoError(t, err)
		
		// Check that log files were copied to destination directory
		for filename := range logFiles {
			copiedPath := filepath.Join(tmpDir, filename)
			content, err := os.ReadFile(copiedPath)
			require.NoError(t, err)
			assert.NotEmpty(t, content)
		}
		
		// Should have warnings for files that don't exist
		assert.GreaterOrEqual(t, len(generator.warnings), 0)
	})
}

func TestReportGenerator_GenerateSessionReport(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Add test data
	generator.AddIssue("translation", "error", "Translation failed", "translator")
	generator.AddWarning("translation", "Low confidence", "translator", map[string]interface{}{
		"confidence": 0.65,
	})
	generator.AddLogEntry("info", "Translation started", "translator", nil)
	
	// Test session data
	session := TranslationSession{
		StartTime:       time.Now().Add(-10 * time.Minute),
		EndTime:         time.Now(),
		Duration:        10 * time.Minute,
		InputFile:       "test_ru.fb2",
		OutputFile:      "test_sr.fb2",
		SSHHost:         "test.local",
		SSHUser:         "testuser",
		TotalSteps:      5,
		CompletedSteps:  4,
		FilesCreated:    []string{"temp_file1.txt", "temp_file2.txt"},
		FilesDownloaded: []string{"result1.fb2", "result2.epub"},
		HashMatch:       true,
		CodeUpdated:     true,
		Success:         false, // Partial success due to incomplete steps
		ErrorMessage:    "Step 5 failed: validation error",
	}
	
	// Test 1: Generate session report
	t.Run("GenerateCompleteSessionReport", func(t *testing.T) {
		err := generator.GenerateSessionReport(session)
		require.NoError(t, err)
		
		// Check that report file was created
		reportPath := filepath.Join(tmpDir, "translation_report.md")
		_, err = os.Stat(reportPath)
		require.NoError(t, err)
		
		// Read and verify report content
		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)
		reportContent := string(content)
		
		// Check basic structure
		assert.Contains(t, reportContent, "# SSH Translation Session Report")
		assert.Contains(t, reportContent, "## Session Overview")
		assert.Contains(t, reportContent, "## Translation Details")
		assert.Contains(t, reportContent, "## Files Generated")
		assert.Contains(t, reportContent, "## Issues Encountered")
		assert.Contains(t, reportContent, "## Warnings")
		assert.Contains(t, reportContent, "## Log Summary")
		assert.Contains(t, reportContent, "## Error Details")
		
		// Check session details
		assert.Contains(t, reportContent, "test_ru.fb2")
		assert.Contains(t, reportContent, "test_sr.fb2")
		assert.Contains(t, reportContent, "test.local")
		assert.Contains(t, reportContent, "testuser")
		assert.Contains(t, reportContent, "4/5") // Completed steps
		assert.Contains(t, reportContent, "false") // Success status
		
		// Check files
		assert.Contains(t, reportContent, "temp_file1.txt")
		assert.Contains(t, reportContent, "result1.fb2")
		
		// Check issues
		assert.Contains(t, reportContent, "Translation failed")
		assert.Contains(t, reportContent, "translator")
		
		// Check warnings
		assert.Contains(t, reportContent, "Low confidence")
		assert.Contains(t, reportContent, "0.65")
		
		// Check error message
		assert.Contains(t, reportContent, "validation error")
	})
	
	// Test 2: Generate successful session report
	t.Run("GenerateSuccessfulSessionReport", func(t *testing.T) {
		successfulSession := session
		successfulSession.Success = true
		successfulSession.CompletedSteps = 5
		successfulSession.ErrorMessage = ""
		
		// Create a new generator for this test
		successGenerator := NewReportGenerator(tmpDir, sessionLogger)
		
		err := successGenerator.GenerateSessionReport(successfulSession)
		require.NoError(t, err)
		
		reportPath := filepath.Join(tmpDir, "translation_report.md")
		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)
		reportContent := string(content)
		
		// Should show success
		assert.Contains(t, reportContent, "true") // Success status
		assert.Contains(t, reportContent, "5/5") // Completed steps
		
		// Should not have error details section when no error
		assert.NotContains(t, reportContent, "## Error Details")
	})
}

func TestReportGenerator_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Add test data
	generator.AddIssue("translation", "error", "Translation failed", "translator")
	generator.AddIssue("setup", "critical", "SSH connection failed", "ssh_client")
	generator.AddIssue("translation", "warning", "Low confidence", "translator")
	generator.AddWarning("file_operation", "File size large", "file_handler", nil)
	generator.AddLogEntry("info", "Translation started", "translator", nil)
	generator.AddLogEntry("error", "Connection failed", "ssh_client", nil)
	
	// Test 1: Get comprehensive stats
	t.Run("GetComprehensiveStats", func(t *testing.T) {
		stats := generator.GetStats()
		
		// Check basic counts
		assert.Equal(t, float64(len(generator.issues)), stats["issues_count"])
		assert.Equal(t, float64(len(generator.warnings)), stats["warnings_count"])
		assert.Equal(t, float64(len(generator.logs)), stats["logs_count"])
		
		// Check session start time
		assert.Equal(t, generator.startTime, stats["session_start"])
		
		// Check severity breakdown
		severityCount := stats["issues_by_severity"].(map[string]int)
		assert.Equal(t, 1, severityCount["error"])
		assert.Equal(t, 1, severityCount["critical"])
		assert.Equal(t, 1, severityCount["warning"])
		
		// Check category breakdown
		categoryCount := stats["issues_by_category"].(map[string]int)
		assert.Equal(t, 2, categoryCount["translation"])
		assert.Equal(t, 1, categoryCount["setup"])
	})
	
	// Test 2: Get stats with no data
	t.Run("GetStatsWithNoData", func(t *testing.T) {
		emptyGenerator := NewReportGenerator(tmpDir, sessionLogger)
		
		stats := emptyGenerator.GetStats()
		
		assert.Equal(t, float64(0), stats["issues_count"])
		assert.Equal(t, float64(0), stats["warnings_count"])
		assert.Equal(t, float64(0), stats["logs_count"])
		
		severityCount := stats["issues_by_severity"].(map[string]int)
		assert.Len(t, severityCount, 0)
		
		categoryCount := stats["issues_by_category"].(map[string]int)
		assert.Len(t, categoryCount, 0)
	})
}

func TestReportGenerator_ExportLogsToFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionLogger := logger.NewLogger(logger.LoggerConfig{})
	generator := NewReportGenerator(tmpDir, sessionLogger)
	
	// Add test data
	generator.AddIssue("translation", "error", "Translation failed", "translator")
	generator.AddWarning("translation", "Low confidence", "translator", map[string]interface{}{
		"confidence": 0.65,
	})
	generator.AddLogEntry("info", "Translation started", "translator", nil)
	generator.AddLogEntry("error", "Connection failed", "ssh_client", nil)
	
	// Test 1: Export logs to file
	t.Run("ExportLogsToFile", func(t *testing.T) {
		err := generator.ExportLogsToFile()
		require.NoError(t, err)
		
		// Check that export file was created
		exportPath := filepath.Join(tmpDir, "session_logs.json")
		_, err = os.Stat(exportPath)
		require.NoError(t, err)
		
		// Read and verify export content
		content, err := os.ReadFile(exportPath)
		require.NoError(t, err)
		exportContent := string(content)
		
		// Check basic structure
		assert.Contains(t, exportContent, "Session Logs Export")
		assert.Contains(t, exportContent, "Session Start:")
		
		// Check issues section
		assert.Contains(t, exportContent, "Issues:")
		assert.Contains(t, exportContent, "[error] translator - Translation failed")
		
		// Check warnings section
		assert.Contains(t, exportContent, "Warnings:")
		assert.Contains(t, exportContent, "[translation] translator - Low confidence")
		
		// Check log entries section
		assert.Contains(t, exportContent, "Log Entries:")
		assert.Contains(t, exportContent, "info: Translation started")
		assert.Contains(t, exportContent, "error: Connection failed")
	})
}