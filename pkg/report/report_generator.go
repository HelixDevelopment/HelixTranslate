package report

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/pkg/logger"
)

// ReportGenerator creates comprehensive reports for translation sessions
type ReportGenerator struct {
	destinationDir string
	logger         logger.Logger
	startTime      time.Time
	issues         []Issue
	warnings       []Warning
	logs           []LogEntry
}

// Issue represents a problem discovered during translation
type Issue struct {
	Timestamp time.Time
	Category  string // "setup", "connection", "translation", "conversion", "file_operation"
	Severity  string // "critical", "error", "warning"
	Message   string
	Component string // which component had the issue
	Resolved  bool
	Resolution string
}

// Warning represents a warning discovered during translation
type Warning struct {
	Timestamp time.Time
	Category  string
	Message   string
	Component string
	Details   map[string]interface{}
}

// LogEntry represents a log entry from the session
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Component string
	Details   map[string]interface{}
}

// TranslationSession represents a complete translation session for reporting
type TranslationSession struct {
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	InputFile       string
	OutputFile      string
	SSHHost         string
	SSHUser         string
	TotalSteps      int
	CompletedSteps  int
	FilesCreated    []string
	FilesDownloaded []string
	HashMatch       bool
	CodeUpdated     bool
	Success         bool
	ErrorMessage    string
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(destinationDir string, sessionLogger logger.Logger) *ReportGenerator {
	return &ReportGenerator{
		destinationDir: destinationDir,
		logger:         sessionLogger,
		startTime:      time.Now(),
		issues:         make([]Issue, 0),
		warnings:       make([]Warning, 0),
		logs:           make([]LogEntry, 0),
	}
}

// AddIssue records an issue discovered during translation
func (r *ReportGenerator) AddIssue(category, severity, message, component string) {
	issue := Issue{
		Timestamp: time.Now(),
		Category:  category,
		Severity:  severity,
		Message:   message,
		Component: component,
		Resolved:  false,
	}
	r.issues = append(r.issues, issue)
	
	r.logger.Error("Issue recorded", map[string]interface{}{
		"category":  category,
		"severity":  severity,
		"message":   message,
		"component": component,
	})
}

// ResolveIssue marks an issue as resolved
func (r *ReportGenerator) ResolveIssue(index int, resolution string) error {
	if index < 0 || index >= len(r.issues) {
		return fmt.Errorf("invalid issue index: %d", index)
	}
	
	r.issues[index].Resolved = true
	r.issues[index].Resolution = resolution
	
	r.logger.Info("Issue resolved", map[string]interface{}{
		"issue_index": index,
		"resolution":  resolution,
	})
	
	return nil
}

// AddWarning records a warning discovered during translation
func (r *ReportGenerator) AddWarning(category, message, component string, details map[string]interface{}) {
	warning := Warning{
		Timestamp: time.Now(),
		Category:  category,
		Message:   message,
		Component: component,
		Details:   details,
	}
	r.warnings = append(r.warnings, warning)
	
	r.logger.Warn("Warning recorded", map[string]interface{}{
		"category":  category,
		"message":   message,
		"component": component,
		"details":   details,
	})
}

// AddLogEntry records a log entry from the session
func (r *ReportGenerator) AddLogEntry(level, message, component string, details map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Component: component,
		Details:   details,
	}
	r.logs = append(r.logs, entry)
}

// CopyLogFiles copies relevant log files to the destination directory
func (r *ReportGenerator) CopyLogFiles(ctx context.Context) error {
	logFiles := []string{
		"translator.log",
		"ssh_worker.log",
		"markdown_workflow.log",
		"llama_cpp.log",
		"error.log",
		"system.log",
	}

	for _, logFile := range logFiles {
		// Try to copy from common log locations
		locations := []string{
			filepath.Join("/var/log", logFile),
			filepath.Join("/tmp", logFile),
			filepath.Join(".", logFile),
			filepath.Join("logs", logFile),
		}

		copied := false
		for _, location := range locations {
			if _, err := os.Stat(location); err == nil {
				if err := r.copyLogFile(location); err == nil {
					copied = true
					break
				}
			}
		}

		if !copied {
			r.AddWarning("logging", fmt.Sprintf("Log file not found: %s", logFile), "report_generator", 
				map[string]interface{}{"file": logFile})
		}
	}

	return nil
}

// copyLogFile copies a single log file to the destination directory
func (r *ReportGenerator) copyLogFile(sourcePath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read log file %s: %w", sourcePath, err)
	}

	destinationPath := filepath.Join(r.destinationDir, filepath.Base(sourcePath))
	if err := os.WriteFile(destinationPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write log file %s: %w", destinationPath, err)
	}

	r.logger.Info("Log file copied", map[string]interface{}{
		"source": sourcePath,
		"destination": destinationPath,
	})

	return nil
}

// GenerateSessionReport generates a markdown report for the translation session
func (r *ReportGenerator) GenerateSessionReport(session TranslationSession) error {
	var report bytes.Buffer

	// Header
	report.WriteString("# SSH Translation Session Report\n\n")
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// Session Overview
	report.WriteString("## Session Overview\n\n")
	report.WriteString(fmt.Sprintf("- **Start Time:** %s\n", session.StartTime.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("- **End Time:** %s\n", session.EndTime.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("- **Duration:** %v\n", session.Duration))
	report.WriteString(fmt.Sprintf("- **Success:** %t\n\n", session.Success))

	// Translation Details
	report.WriteString("## Translation Details\n\n")
	report.WriteString(fmt.Sprintf("- **Input File:** `%s`\n", session.InputFile))
	report.WriteString(fmt.Sprintf("- **Output File:** `%s`\n", session.OutputFile))
	report.WriteString(fmt.Sprintf("- **SSH Host:** `%s`\n", session.SSHHost))
	report.WriteString(fmt.Sprintf("- **SSH User:** `%s`\n", session.SSHUser))
	report.WriteString(fmt.Sprintf("- **Steps Completed:** %d/%d\n", session.CompletedSteps, session.TotalSteps))
	report.WriteString(fmt.Sprintf("- **Hash Match:** %t\n", session.HashMatch))
	report.WriteString(fmt.Sprintf("- **Code Updated:** %t\n\n", session.CodeUpdated))

	// Files Created/Downloaded
	report.WriteString("## Files Generated\n\n")
	if len(session.FilesCreated) > 0 {
		report.WriteString("### Remote Files Created\n\n")
		for i, file := range session.FilesCreated {
			report.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, file))
		}
		report.WriteString("\n")
	}

	if len(session.FilesDownloaded) > 0 {
		report.WriteString("### Local Files Downloaded\n\n")
		for i, file := range session.FilesDownloaded {
			report.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, file))
		}
		report.WriteString("\n")
	}

	// Issues Section
	if len(r.issues) > 0 {
		report.WriteString("## Issues Encountered\n\n")
		
		criticalIssues := 0
		errorIssues := 0
		warningIssues := 0
		
		for _, issue := range r.issues {
			switch issue.Severity {
			case "critical":
				criticalIssues++
			case "error":
				errorIssues++
			case "warning":
				warningIssues++
			}
		}
		
		report.WriteString(fmt.Sprintf("- **Critical Issues:** %d\n", criticalIssues))
		report.WriteString(fmt.Sprintf("- **Error Issues:** %d\n", errorIssues))
		report.WriteString(fmt.Sprintf("- **Warning Issues:** %d\n\n", warningIssues))
		
		for i, issue := range r.issues {
			status := "❌ Open"
			if issue.Resolved {
				status = "✅ Resolved"
			}
			
			report.WriteString(fmt.Sprintf("### Issue #%d - %s\n\n", i+1, status))
			report.WriteString(fmt.Sprintf("- **Category:** %s\n", issue.Category))
			report.WriteString(fmt.Sprintf("- **Severity:** %s\n", issue.Severity))
			report.WriteString(fmt.Sprintf("- **Component:** %s\n", issue.Component))
			report.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", issue.Timestamp.Format("2006-01-02 15:04:05")))
			report.WriteString(fmt.Sprintf("- **Message:** %s\n", issue.Message))
			
			if issue.Resolved {
				report.WriteString(fmt.Sprintf("- **Resolution:** %s\n", issue.Resolution))
			}
			
			report.WriteString("\n")
		}
	}

	// Warnings Section
	if len(r.warnings) > 0 {
		report.WriteString("## Warnings\n\n")
		
		for i, warning := range r.warnings {
			report.WriteString(fmt.Sprintf("### Warning #%d\n\n", i+1))
			report.WriteString(fmt.Sprintf("- **Category:** %s\n", warning.Category))
			report.WriteString(fmt.Sprintf("- **Component:** %s\n", warning.Component))
			report.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", warning.Timestamp.Format("2006-01-02 15:04:05")))
			report.WriteString(fmt.Sprintf("- **Message:** %s\n", warning.Message))
			
			if len(warning.Details) > 0 {
				report.WriteString("- **Details:**\n")
				for key, value := range warning.Details {
					report.WriteString(fmt.Sprintf("  - %s: %v\n", key, value))
				}
			}
			
			report.WriteString("\n")
		}
	}

	// Log Summary Section
	if len(r.logs) > 0 {
		report.WriteString("## Log Summary\n\n")
		
		logLevels := make(map[string]int)
		components := make(map[string]int)
		
		for _, log := range r.logs {
			logLevels[log.Level]++
			components[log.Component]++
		}
		
		report.WriteString("### Log Levels\n\n")
		for level, count := range logLevels {
			report.WriteString(fmt.Sprintf("- **%s:** %d entries\n", strings.ToUpper(level), count))
		}
		
		report.WriteString("\n### Components\n\n")
		for component, count := range components {
			report.WriteString(fmt.Sprintf("- **%s:** %d entries\n", component, count))
		}
		
		report.WriteString(fmt.Sprintf("\n**Total Log Entries:** %d\n\n", len(r.logs)))
	}

	// Recent Log Entries (last 20)
	if len(r.logs) > 0 {
		report.WriteString("## Recent Log Entries (Last 20)\n\n")
		
		start := len(r.logs) - 20
		if start < 0 {
			start = 0
		}
		
		for i := start; i < len(r.logs); i++ {
			log := r.logs[i]
			report.WriteString(fmt.Sprintf("**[%s]** `%s` - %s\n", 
				log.Timestamp.Format("15:04:05"), 
				strings.ToUpper(log.Level), 
				log.Message))
			
			if log.Component != "" {
				report.WriteString(fmt.Sprintf("Component: `%s`\n", log.Component))
			}
			
			if len(log.Details) > 0 {
				for key, value := range log.Details {
					report.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
				}
			}
			
			report.WriteString("\n")
		}
	}

	// Error Message Section
	if session.ErrorMessage != "" {
		report.WriteString("## Error Details\n\n")
		report.WriteString(fmt.Sprintf("```\n%s\n```\n\n", session.ErrorMessage))
	}

	// Footer
	report.WriteString("---\n")
	report.WriteString(fmt.Sprintf("*Report generated by SSH Translation System at %s*\n", 
		time.Now().Format("2006-01-02 15:04:05")))

	// Write report to file
	reportPath := filepath.Join(r.destinationDir, "translation_report.md")
	if err := os.WriteFile(reportPath, report.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	r.logger.Info("Translation report generated", map[string]interface{}{
		"report_path": reportPath,
		"issues_count": len(r.issues),
		"warnings_count": len(r.warnings),
		"logs_count": len(r.logs),
	})

	return nil
}

// GenerateLogArchive creates an archive of all logs in the destination directory
func (r *ReportGenerator) GenerateLogArchive() error {
	// This would create an archive of all log files
	// For now, we'll just copy individual log files
	return r.CopyLogFiles(context.Background())
}

// GetStats returns statistics about the session
func (r *ReportGenerator) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"session_start":  r.startTime,
		"issues_count":  float64(len(r.issues)),
		"warnings_count": float64(len(r.warnings)),
		"logs_count":     float64(len(r.logs)),
	}

	// Count by severity
	severityCount := make(map[string]int)
	for _, issue := range r.issues {
		severityCount[issue.Severity]++
	}
	stats["issues_by_severity"] = severityCount

	// Count by category
	categoryCount := make(map[string]int)
	for _, issue := range r.issues {
		categoryCount[issue.Category]++
	}
	stats["issues_by_category"] = categoryCount

	return stats
}

// ExportLogsToFile exports all collected logs to a structured file
func (r *ReportGenerator) ExportLogsToFile() error {
	logPath := filepath.Join(r.destinationDir, "session_logs.json")
	
	// Create structured log data
	logData := map[string]interface{}{
		"session_start": r.startTime,
		"issues":        r.issues,
		"warnings":      r.warnings,
		"logs":          r.logs,
	}

	// Convert to JSON (in a real implementation, use json.Marshal)
	// For now, we'll create a simple text file
	var content strings.Builder
	content.WriteString("Session Logs Export\n")
	content.WriteString("===================\n\n")
	content.WriteString(fmt.Sprintf("Session Start: %s\n\n", r.startTime.Format("2006-01-02 15:04:05")))

	if len(r.issues) > 0 {
		content.WriteString("Issues:\n")
		for i, issue := range r.issues {
			content.WriteString(fmt.Sprintf("  %d. [%s] %s - %s\n", i+1, issue.Severity, issue.Component, issue.Message))
		}
		content.WriteString("\n")
	}

	if len(r.warnings) > 0 {
		content.WriteString("Warnings:\n")
		for i, warning := range r.warnings {
			content.WriteString(fmt.Sprintf("  %d. [%s] %s - %s\n", i+1, warning.Category, warning.Component, warning.Message))
		}
		content.WriteString("\n")
	}

	// Write log entries
	content.WriteString("Log Entries:\n")
	for _, log := range r.logs {
		content.WriteString(fmt.Sprintf("  [%s] %s: %s\n", log.Timestamp.Format("15:04:05"), log.Level, log.Message))
	}

	_ = logData // Avoid unused variable warning

	return os.WriteFile(logPath, []byte(content.String()), 0644)
}