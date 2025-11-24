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

// shouldLog determines if a message should be logged based on log level
func (l *StandardLogger) shouldLog(messageLevel string) bool {
	levels := map[int]string{
		0: DEBUG,
		1: INFO,
		2: WARN,
		3: ERROR,
		4: FATAL,
	}

	messageLevelValue := -1
	currentLevelValue := -1

	for i, level := range levels {
		if level == messageLevel {
			messageLevelValue = i
		}
		if level == l.level {
			currentLevelValue = i
		}
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
	logData := map[string]interface{}{
		"timestamp": timestamp,
		"level":     level,
		"message":   message,
	}

	// Add fields
	for key, value := range fields {
		logData[key] = value
	}

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