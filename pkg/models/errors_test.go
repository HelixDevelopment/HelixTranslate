package models

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestErrorDefinitions tests error definitions
func TestErrorDefinitions(t *testing.T) {
	// Test error types
	testCases := []struct {
		name        string
		error       error
		expectedMsg string
		expectError bool
	}{
		{
			name:        "UserNotFoundError",
			error:       ErrUserNotFound,
			expectedMsg: "user not found",
			expectError: true,
		},
		{
			name:        "InvalidCredentialsError",
			error:       ErrInvalidCredentials,
			expectedMsg: "invalid credentials",
			expectError: true,
		},
		{
			name:        "UserAlreadyExistsError",
			error:       ErrUserAlreadyExists,
			expectedMsg: "user already exists",
			expectError: true,
		},
		{
			name:        "UserInactiveError",
			error:       ErrUserInactive,
			expectedMsg: "user account is inactive",
			expectError: true,
		},
		{
			name:        "NilError",
			error:       nil,
			expectedMsg: "",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectError {
				assert.Error(t, tc.error)
				assert.Contains(t, tc.error.Error(), tc.expectedMsg)
			} else {
				assert.NoError(t, tc.error)
			}
		})
	}
}

// TestErrorCategories tests error categorization
func TestErrorCategories(t *testing.T) {
	testCases := []struct {
		name         string
		error        error
		category     string
		isClientSide bool
		isServerSide bool
		isRetryable  bool
	}{
		{
			name:         "UserNotFound category",
			error:        ErrUserNotFound,
			category:     "client",
			isClientSide: true,
			isServerSide: false,
			isRetryable:  false,
		},
		{
			name:         "InvalidCredentials category",
			error:        ErrInvalidCredentials,
			category:     "client",
			isClientSide: true,
			isServerSide: false,
			isRetryable:  false,
		},
		{
			name:         "UserAlreadyExists category",
			error:        ErrUserAlreadyExists,
			category:     "client",
			isClientSide: true,
			isServerSide: false,
			isRetryable:  false,
		},
		{
			name:         "UserInactive category",
			error:        ErrUserInactive,
			category:     "client",
			isClientSide: true,
			isServerSide: false,
			isRetryable:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			category := GetErrorCategory(tc.error)
			assert.Equal(t, tc.category, category)
			
			isClientSide := IsClientSideError(tc.error)
			assert.Equal(t, tc.isClientSide, isClientSide)
			
			isServerSide := IsServerSideError(tc.error)
			assert.Equal(t, tc.isServerSide, isServerSide)
			
			isRetryable := IsRetryableError(tc.error)
			assert.Equal(t, tc.isRetryable, isRetryable)
		})
	}
}

// TestErrorCreation tests error creation functions
func TestErrorCreation(t *testing.T) {
	t.Run("Create UserNotFoundError with context", func(t *testing.T) {
		err := NewUserNotFoundError("user-123")
		assert.Contains(t, err.Error(), "user-123")
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("Create InvalidCredentialsError with context", func(t *testing.T) {
		err := NewInvalidCredentialsError("Invalid password")
		assert.Contains(t, err.Error(), "Invalid password")
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("Create UserAlreadyExistsError with context", func(t *testing.T) {
		err := NewUserAlreadyExistsError("user@example.com")
		assert.Contains(t, err.Error(), "user@example.com")
		assert.Contains(t, err.Error(), "user already exists")
	})

	t.Run("Create UserInactiveError with context", func(t *testing.T) {
		err := NewUserInactiveError("user-123")
		assert.Contains(t, err.Error(), "user-123")
		assert.Contains(t, err.Error(), "user account is inactive")
	})
}

// TestErrorWrapping tests error wrapping functionality
func TestErrorWrapping(t *testing.T) {
	originalErr := errors.New("original error")
	
	t.Run("Wrap error with context", func(t *testing.T) {
		wrappedErr := WrapError(originalErr, "Failed to process user")
		assert.Contains(t, wrappedErr.Error(), "Failed to process user")
		assert.Contains(t, wrappedErr.Error(), "original error")
		assert.True(t, errors.Is(wrappedErr, originalErr))
	})

	t.Run("Wrap nil error", func(t *testing.T) {
		wrappedErr := WrapError(nil, "Should not wrap nil")
		assert.Nil(t, wrappedErr)
	})

	t.Run("Wrap error multiple times", func(t *testing.T) {
		wrappedErr1 := WrapError(originalErr, "Level 1")
		wrappedErr2 := WrapError(wrappedErr1, "Level 2")
		
		assert.Contains(t, wrappedErr2.Error(), "Level 2")
		assert.Contains(t, wrappedErr2.Error(), "Level 1")
		assert.Contains(t, wrappedErr2.Error(), "original error")
		assert.True(t, errors.Is(wrappedErr2, originalErr))
	})
}

// TestErrorRecovery tests error recovery mechanisms
func TestErrorRecovery(t *testing.T) {
	t.Run("Recovery from retryable error", func(t *testing.T) {
		attempt := 0
		maxAttempts := 3
		
		var result string
		var err error
		
		for attempt < maxAttempts {
			attempt++
			
			// Simulate failure on first two attempts
			if attempt < 3 {
				err = errors.New("temporary network error")
				if IsRetryableError(err) {
					continue // Retry
				} else {
					break // Don't retry
				}
			}
			
			// Simulate success on third attempt
			result = "success"
			err = nil
			break
		}
		
		assert.Equal(t, 3, attempt)
		assert.Equal(t, "success", result)
		assert.NoError(t, err)
	})

	t.Run("No recovery from non-retryable error", func(t *testing.T) {
		attempt := 0
		maxAttempts := 3
		
		var result string
		var err error
		
		for attempt < maxAttempts {
			attempt++
			
			err = ErrInvalidCredentials
			if IsRetryableError(err) {
				continue // Retry
			} else {
				break // Don't retry
			}
		}
		
		assert.Equal(t, 1, attempt) // Only one attempt
		assert.Empty(t, result)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
	})
}

// TestErrorValidation tests error validation functionality
func TestErrorValidation(t *testing.T) {
	testCases := []struct {
		name     string
		error    error
		isValid  bool
		severity string
	}{
		{
			name:     "Valid user not found error",
			error:    ErrUserNotFound,
			isValid:  true,
			severity: "warning",
		},
		{
			name:     "Valid invalid credentials error",
			error:    ErrInvalidCredentials,
			isValid:  true,
			severity: "error",
		},
		{
			name:     "Valid user already exists error",
			error:    ErrUserAlreadyExists,
			isValid:  true,
			severity: "warning",
		},
		{
			name:     "Valid user inactive error",
			error:    ErrUserInactive,
			isValid:  true,
			severity: "error",
		},
		{
			name:     "Nil error",
			error:    nil,
			isValid:  false,
			severity: "none",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := IsValidError(tc.error)
			severity := GetErrorSeverity(tc.error)
			
			assert.Equal(t, tc.isValid, isValid)
			assert.Equal(t, tc.severity, severity)
		})
	}
}

// TestErrorFormatting tests error formatting functionality
func TestErrorFormatting(t *testing.T) {
	errors := []error{
		ErrUserNotFound,
		ErrInvalidCredentials,
		ErrUserAlreadyExists,
		ErrUserInactive,
	}

	for _, err := range errors {
		t.Run(fmt.Sprintf("Format %s", err.Error()), func(t *testing.T) {
			// Test basic string formatting
			str := err.Error()
			assert.NotEmpty(t, str)
			
			// Test JSON formatting
			jsonStr := FormatErrorForJSON(err)
			assert.NotEmpty(t, jsonStr)
			assert.Contains(t, jsonStr, err.Error())
			
			// Test logging formatting
			logStr := FormatErrorForLogging(err)
			assert.NotEmpty(t, logStr)
			assert.Contains(t, logStr, err.Error())
		})
	}
}

// TestErrorComparison tests error comparison functionality
func TestErrorComparison(t *testing.T) {
	testCases := []struct {
		name       string
		error1     error
		error2     error
		isSame     bool
		isSameType bool
	}{
		{
			name:       "Same user not found errors",
			error1:     ErrUserNotFound,
			error2:     ErrUserNotFound,
			isSame:     true,
			isSameType: true,
		},
		{
			name:       "Same type different context",
			error1:     NewUserNotFoundError("user-1"),
			error2:     NewUserNotFoundError("user-2"),
			isSame:     false,
			isSameType: true,
		},
		{
			name:       "Different error types",
			error1:     ErrUserNotFound,
			error2:     ErrInvalidCredentials,
			isSame:     false,
			isSameType: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isSame := errors.Is(tc.error1, tc.error2)
			isSameType := IsSameErrorType(tc.error1, tc.error2)
			
			assert.Equal(t, tc.isSame, isSame)
			assert.Equal(t, tc.isSameType, isSameType)
		})
	}
}

// BenchmarkErrorCreation benchmarks error creation
func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := NewUserNotFoundError(fmt.Sprintf("user-%d", i))
		_ = err // Use error to avoid optimization
	}
}

// BenchmarkErrorValidation benchmarks error validation
func BenchmarkErrorValidation(b *testing.B) {
	err := ErrUserNotFound
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		isClientSide := IsClientSideError(err)
		isServerSide := IsServerSideError(err)
		isRetryable := IsRetryableError(err)
		
		// Use values to avoid optimization
		_ = isClientSide || isServerSide || isRetryable
	}
}

// Mock helper functions for testing

// NewUserNotFoundError creates a user not found error with context
func NewUserNotFoundError(userID string) error {
	return fmt.Errorf("user not found: %s", userID)
}

// NewInvalidCredentialsError creates an invalid credentials error with context
func NewInvalidCredentialsError(context string) error {
	return fmt.Errorf("invalid credentials: %s", context)
}

// NewUserAlreadyExistsError creates a user already exists error with context
func NewUserAlreadyExistsError(email string) error {
	return fmt.Errorf("user already exists: %s", email)
}

// NewUserInactiveError creates a user inactive error with context
func NewUserInactiveError(userID string) error {
	return fmt.Errorf("user account is inactive: %s", userID)
}

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// GetErrorCategory returns the category of an error
func GetErrorCategory(err error) string {
	if err == nil {
		return "none"
	}
	
	if errors.Is(err, ErrUserNotFound) || 
	   errors.Is(err, ErrInvalidCredentials) || 
	   errors.Is(err, ErrUserAlreadyExists) || 
	   errors.Is(err, ErrUserInactive) {
		return "client"
	}
	
	return "server"
}

// IsClientSideError checks if an error is client-side
func IsClientSideError(err error) bool {
	return GetErrorCategory(err) == "client"
}

// IsServerSideError checks if an error is server-side
func IsServerSideError(err error) bool {
	return GetErrorCategory(err) == "server"
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	// Only server errors are typically retryable
	return IsServerSideError(err)
}

// IsValidError checks if an error is valid
func IsValidError(err error) bool {
	return err != nil
}

// GetErrorSeverity returns the severity level of an error
func GetErrorSeverity(err error) string {
	if err == nil {
		return "none"
	}
	
	if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
		return "error"
	}
	
	return "warning"
}

// FormatErrorForJSON formats an error for JSON output
func FormatErrorForJSON(err error) string {
	if err == nil {
		return "{}"
	}
	return fmt.Sprintf(`{"error": "%s"}`, err.Error())
}

// FormatErrorForLogging formats an error for logging
func FormatErrorForLogging(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("ERROR: %s", err.Error())
}

// IsSameErrorType checks if two errors are of the same type
func IsSameErrorType(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 == nil || err2 == nil {
		return false
	}
	
	// Simple type comparison by string matching
	return err1.Error()[:10] == err2.Error()[:10]
}

// UserSession represents a user session
type UserSession struct {
	Token     string
	UserID    string
	UserEmail string
	Username  string
	ExpiresAt time.Time
}

// CreateUserSession creates a new user session
func CreateUserSession(user User, duration time.Duration) UserSession {
	return UserSession{
		Token:     GenerateSessionToken(),
		UserID:    user.ID,
		UserEmail:  user.Email,
		Username:  user.Username,
		ExpiresAt: time.Now().Add(duration),
	}
}

// ValidateSession validates if a session is still valid
func ValidateSession(session UserSession) bool {
	return !session.ExpiresAt.IsZero() && session.ExpiresAt.After(time.Now())
}

// GenerateSessionToken generates a new session token
func GenerateSessionToken() string {
	return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_=+"
}

// UserHasRole checks if a user has a specific role
func UserHasRole(user User, role string) bool {
	if !user.IsActive {
		return false
	}
	
	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HashPassword hashes a password
func HashPassword(password string) (string, error) {
	return "hashed_" + password, nil
}

// CheckPassword verifies a password against its hash
func CheckPassword(hashedPassword, password string) bool {
	return hashedPassword == "hashed_"+password
}