package deployment

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// APICommunicationLogger logs all REST API communications between distributed nodes
type APICommunicationLogger struct {
	logFile *os.File
	logger  *log.Logger
	mu      sync.Mutex
}

// NewAPICommunicationLogger creates a new API communication logger
func NewAPICommunicationLogger(logPath string) (*APICommunicationLogger, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(file, "", 0) // No prefix, we'll format ourselves

	return &APICommunicationLogger{
		logFile: file,
		logger:  logger,
	}, nil
}

// LogCommunication logs an API communication event
func (acl *APICommunicationLogger) LogCommunication(logEntry *APICommunicationLog) error {
	acl.mu.Lock()
	defer acl.mu.Unlock()

	// Format like Retrofit library (Android) for impeccable readability
	var logLine string

	if logEntry.StatusCode == 0 {
		// Outgoing request
		timestamp := logEntry.Timestamp.Format("2006/01/02 15:04:05.000")
		sizeInfo := ""
		if logEntry.RequestSize > 0 {
			sizeInfo = fmt.Sprintf(" (%d-byte body)", logEntry.RequestSize)
		}
		logLine = fmt.Sprintf("%s --> %s %s://%s:%d%s%s",
			timestamp,
			logEntry.Method,
			acl.getProtocol(logEntry.TargetPort),
			logEntry.TargetHost,
			logEntry.TargetPort,
			logEntry.URL,
			sizeInfo)
	} else {
		// Incoming response
		timestamp := logEntry.Timestamp.Format("2006/01/02 15:04:05.000")
		duration := acl.formatDuration(logEntry.Duration)
		sizeInfo := ""
		if logEntry.ResponseSize > 0 {
			sizeInfo = fmt.Sprintf(", %d-byte body", logEntry.ResponseSize)
		}
		statusText := acl.getStatusText(logEntry.StatusCode)

		logLine = fmt.Sprintf("%s <-- %d %s %s://%s:%d%s (%s%s)",
			timestamp,
			logEntry.StatusCode,
			statusText,
			acl.getProtocol(logEntry.TargetPort),
			logEntry.TargetHost,
			logEntry.TargetPort,
			logEntry.URL,
			duration,
			sizeInfo)

		// Add error information if present
		if logEntry.Error != "" {
			logLine += fmt.Sprintf("\n%s <-- HTTP FAILED: %s", timestamp, logEntry.Error)
		}
	}

	acl.logger.Println(logLine)
	return nil
}

// LogRequest logs an outgoing API request
func (acl *APICommunicationLogger) LogRequest(sourceHost string, sourcePort int, targetHost string, targetPort int, method, url string, requestSize int64) *APICommunicationLog {
	entry := &APICommunicationLog{
		Timestamp:   time.Now(),
		SourceHost:  sourceHost,
		SourcePort:  sourcePort,
		TargetHost:  targetHost,
		TargetPort:  targetPort,
		Method:      method,
		URL:         url,
		RequestSize: requestSize,
	}

	// Log asynchronously to avoid blocking
	go func() {
		if err := acl.LogCommunication(entry); err != nil {
			log.Printf("Failed to log API request: %v", err)
		}
	}()

	return entry
}

// LogResponse logs the response for a previously logged request
func (acl *APICommunicationLogger) LogResponse(entry *APICommunicationLog, statusCode int, responseSize int64, duration time.Duration, err error) {
	entry.StatusCode = statusCode
	entry.ResponseSize = responseSize
	entry.Duration = duration
	entry.Timestamp = time.Now() // Update timestamp for response logging

	if err != nil {
		entry.Error = err.Error()
	}

	// Update the existing log entry
	go func() {
		if logErr := acl.LogCommunication(entry); logErr != nil {
			log.Printf("Failed to log API response: %v", logErr)
		}
	}()
}

// GetLogs retrieves recent log entries
func (acl *APICommunicationLogger) GetLogs(limit int) ([]*APICommunicationLog, error) {
	acl.mu.Lock()
	defer acl.mu.Unlock()

	// This is a simplified implementation
	// In a real system, you'd want to read from the log file and parse recent entries
	return []*APICommunicationLog{}, nil
}

// Close closes the logger
func (acl *APICommunicationLogger) Close() error {
	acl.mu.Lock()
	defer acl.mu.Unlock()

	if acl.logFile != nil {
		return acl.logFile.Close()
	}
	return nil
}

// GetStats returns communication statistics
func (acl *APICommunicationLogger) GetStats() map[string]interface{} {
	// This would parse the log file to generate statistics
	// For now, return empty stats
	return map[string]interface{}{
		"total_requests":  0,
		"total_responses": 0,
		"error_count":     0,
		"avg_duration":    "0s",
	}
}

// Helper methods for Retrofit-style formatting

// getProtocol returns the protocol (http/https) based on port
func (acl *APICommunicationLogger) getProtocol(port int) string {
	if port == 443 || port == 8443 {
		return "https"
	}
	return "http"
}

// formatDuration formats duration in Retrofit style (e.g., "150ms", "2.5s")
func (acl *APICommunicationLogger) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// getStatusText returns the HTTP status text for common status codes
func (acl *APICommunicationLogger) getStatusText(statusCode int) string {
	switch statusCode {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 409:
		return "Conflict"
	case 422:
		return "Unprocessable Entity"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	case 504:
		return "Gateway Timeout"
	default:
		return ""
	}
}
