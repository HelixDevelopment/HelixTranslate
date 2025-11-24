package logger

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	config := LoggerConfig{
		Level:  "info",
		Format: "text",
	}
	logger := NewLogger(config)

	// Test all log levels
	logger.Debug("debug message", map[string]interface{}{"key": "value"}) // Should not log
	logger.Info("info message", map[string]interface{}{"key": "value"})   // Should log
	logger.Warn("warn message", map[string]interface{}{"key": "value"})   // Should log
	logger.Error("error message", map[string]interface{}{"key": "value"})  // Should log
}

func TestLogger_Formats(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"text", "["},
		{"json", "{"},
	}

	for _, test := range tests {
		config := LoggerConfig{
			Level:  "info",
			Format: test.format,
		}
		logger := NewLogger(config)
		
		// Capture output (would need to redirect stdout in real implementation)
		logger.Info("test message", map[string]interface{}{"key": "value"})
	}
}

func TestLogger_JSONFormatting(t *testing.T) {
	config := LoggerConfig{
		Level:  "debug",
		Format: "json",
	}
	logger := NewLogger(config)

	// Test the JSON formatting directly
	stdLogger := logger.(*StandardLogger)
	fields := map[string]interface{}{"test": "value"}
	
	result := stdLogger.formatJSON("info", "test message", fields, "2023-01-01 12:00:00")
	
	// Verify it's valid JSON
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(result), &parsed)
	if err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
	
	// Check required fields
	if parsed["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", parsed["message"])
	}
	if parsed["level"] != "info" {
		t.Errorf("Expected level 'info', got %v", parsed["level"])
	}
	if parsed["test"] != "value" {
		t.Errorf("Expected test field 'value', got %v", parsed["test"])
	}
}

func TestLogger_TextFormatting(t *testing.T) {
	config := LoggerConfig{
		Level:  "debug",
		Format: "text",
	}
	logger := NewLogger(config)

	// Test the text formatting directly
	stdLogger := logger.(*StandardLogger)
	fields := map[string]interface{}{"test": "value"}
	
	result := stdLogger.formatText("info", "test message", fields, "2023-01-01 12:00:00")
	
	// Check format contains expected elements
	if !strings.Contains(result, "[2023-01-01 12:00:00]") {
		t.Errorf("Expected timestamp in output: %s", result)
	}
	if !strings.Contains(result, "INFO: test message") {
		t.Errorf("Expected level and message in output: %s", result)
	}
	if !strings.Contains(result, "test=value") {
		t.Errorf("Expected fields in output: %s", result)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	config := LoggerConfig{
		Level:  "warn",
		Format: "text",
	}
	logger := NewLogger(config)

	// Test the shouldLog method directly
	stdLogger := logger.(*StandardLogger)
	
	if !stdLogger.shouldLog("error") {
		t.Error("Error level should be logged")
	}
	if !stdLogger.shouldLog("warn") {
		t.Error("Warn level should be logged")
	}
	if stdLogger.shouldLog("info") {
		t.Error("Info level should not be logged at warn level")
	}
	if stdLogger.shouldLog("debug") {
		t.Error("Debug level should not be logged at warn level")
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := NewNoOpLogger()
	
	// These should all be no-ops and not panic
	logger.Debug("test", nil)
	logger.Info("test", nil)
	logger.Warn("test", nil)
	logger.Error("test", nil)
	// Don't call Fatal as it exits the program
}

func TestLogger_Defaults(t *testing.T) {
	// Test empty config defaults
	config := LoggerConfig{}
	logger := NewLogger(config)
	
	stdLogger := logger.(*StandardLogger)
	if stdLogger.level != "info" {
		t.Errorf("Default level should be info, got %s", stdLogger.level)
	}
	if stdLogger.format != "text" {
		t.Errorf("Default format should be text, got %s", stdLogger.format)
	}
}