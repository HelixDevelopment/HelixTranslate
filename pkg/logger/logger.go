package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Log levels
const (
	DEBUG = "debug"
	INFO  = "info"
	WARN  = "warn"
	ERROR = "error"
	FATAL = "fatal"
)

// Log formats
const (
	FORMAT_TEXT = "text"
	FORMAT_JSON = "json"
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Level      string
	Format     string
	OutputFile string
}

// Logger interface for logging operations
type Logger interface {
	Debug(message string, fields map[string]interface{})
	Info(message string, fields map[string]interface{})
	Warn(message string, fields map[string]interface{})
	Error(message string, fields map[string]interface{})
	Fatal(message string, fields map[string]interface{})
}

// StandardLogger implements the Logger interface
type StandardLogger struct {
	level  string
	format string
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(config LoggerConfig) Logger {
	// Set default level if not specified
	if config.Level == "" {
		config.Level = INFO
	}

	// Set default format if not specified
	if config.Format == "" {
		config.Format = FORMAT_TEXT
	}

	// Determine output
	var output *os.File
	if config.OutputFile != "" {
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file %s: %v, using stdout", config.OutputFile, err)
			output = os.Stdout
		} else {
			output = file
		}
	} else {
		output = os.Stdout
	}

	return &StandardLogger{
		level:  strings.ToLower(config.Level),
		format: strings.ToLower(config.Format),
		logger: log.New(output, "", 0), // We'll handle our own formatting
	}
}

// levelRank maps a known level name to its numeric severity (lower = more
// verbose). The second return value reports whether the level was recognized.
func levelRank(level string) (int, bool) {
	switch level {
	case DEBUG:
		return 0, true
	case INFO:
		return 1, true
	case WARN:
		return 2, true
	case ERROR:
		return 3, true
	case FATAL:
		return 4, true
	default:
		return 0, false
	}
}

// shouldLog determines if a message should be logged based on log level.
//
// Both the configured level and the message level are validated. An
// unrecognized configured level (e.g. the common typo "warning") MUST NOT
// fail open and log everything — it falls back to the INFO default. An
// unrecognized message level MUST NOT be silently dropped — it is treated as
// at least as severe as the configured level so it is never lost.
func (l *StandardLogger) shouldLog(messageLevel string) bool {
	currentLevelValue, ok := levelRank(l.level)
	if !ok {
		// Misconfigured level: fall back to the documented default (INFO)
		// rather than failing open to DEBUG.
		currentLevelValue, _ = levelRank(INFO)
	}

	messageLevelValue, ok := levelRank(messageLevel)
	if !ok {
		// Unknown message severity: do not silently drop it.
		return true
	}

	return messageLevelValue >= currentLevelValue
}

// formatMessage formats the log message based on the configured format
func (l *StandardLogger) formatMessage(level, message string, fields map[string]interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	switch l.format {
	case FORMAT_JSON:
		return l.formatJSON(level, message, fields, timestamp)
	default:
		return l.formatText(level, message, fields, timestamp)
	}
}

// formatText formats the message in plain text format
func (l *StandardLogger) formatText(level, message string, fields map[string]interface{}, timestamp string) string {
	var sb strings.Builder

	// Basic format: [timestamp] LEVEL: message
	sb.WriteString(fmt.Sprintf("[%s] %s: %s", timestamp, strings.ToUpper(level), message))

	// Add fields if present
	if len(fields) > 0 {
		sb.WriteString(" |")
		for key, value := range fields {
			sb.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
	}

	return sb.String()
}

// formatJSON formats the message as JSON
func (l *StandardLogger) formatJSON(level, message string, fields map[string]interface{}, timestamp string) string {
	logData := make(map[string]interface{}, len(fields)+3)

	// Add user fields first. Any field whose key collides with a reserved log
	// key (timestamp/level/message) is re-homed under a "fields." prefix so the
	// authoritative log metadata set below is never clobbered AND the user value
	// is never silently dropped (a dropped severity would corrupt downstream
	// level filtering/alerting).
	for key, value := range fields {
		switch key {
		case "timestamp", "level", "message":
			logData["fields."+key] = value
		default:
			logData[key] = value
		}
	}

	// Reserved metadata is authoritative — set last so it always wins.
	logData["timestamp"] = timestamp
	logData["level"] = level
	logData["message"] = message

	// Use json.Marshal for proper JSON formatting
	jsonBytes, err := json.Marshal(logData)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal log","message":"%s","timestamp":"%s","level":"%s"}`, message, timestamp, level)
	}
	return string(jsonBytes)
}

// log is the internal logging method
func (l *StandardLogger) log(level, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	formatted := l.formatMessage(level, message, fields)
	l.logger.Println(formatted)
}

// Debug logs a debug message
func (l *StandardLogger) Debug(message string, fields map[string]interface{}) {
	l.log(DEBUG, message, fields)
}

// Info logs an info message
func (l *StandardLogger) Info(message string, fields map[string]interface{}) {
	l.log(INFO, message, fields)
}

// Warn logs a warning message
func (l *StandardLogger) Warn(message string, fields map[string]interface{}) {
	l.log(WARN, message, fields)
}

// Error logs an error message
func (l *StandardLogger) Error(message string, fields map[string]interface{}) {
	l.log(ERROR, message, fields)
}

// Fatal logs a fatal message and exits the program
func (l *StandardLogger) Fatal(message string, fields map[string]interface{}) {
	l.log(FATAL, message, fields)
	os.Exit(1)
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

// NewNoOpLogger creates a no-op logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}

// Debug logs a debug message (no-op)
func (l *NoOpLogger) Debug(message string, fields map[string]interface{}) {
	// No-op
}

// Info logs an info message (no-op)
func (l *NoOpLogger) Info(message string, fields map[string]interface{}) {
	// No-op
}

// Warn logs a warning message (no-op)
func (l *NoOpLogger) Warn(message string, fields map[string]interface{}) {
	// No-op
}

// Error logs an error message (no-op)
func (l *NoOpLogger) Error(message string, fields map[string]interface{}) {
	// No-op
}

// Fatal logs a fatal message and exits the program (no-op)
func (l *NoOpLogger) Fatal(message string, fields map[string]interface{}) {
	// No-op
}
