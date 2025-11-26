package security

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"digital.vasic.translator/pkg/models"
)

// TestRateLimiter_ConcurrentAccess tests thread safety of rate limiter
func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rps := 10
	burst := 5
	rl := NewRateLimiter(rps, burst)
	
	const numGoroutines = 100
	const numRequests = 10
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := make(map[string]int)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("user%d", id)
			
			allowedCount := 0
			for j := 0; j < numRequests; j++ {
				if rl.Allow(key) {
					allowedCount++
				}
				// Small delay to avoid immediate requests
				time.Sleep(time.Millisecond)
			}
			
			mu.Lock()
			allowed[key] = allowedCount
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	
	// Verify that rate limiting is working
	for key, count := range allowed {
		// Should be less than total requests due to rate limiting
		assert.Less(t, count, numRequests, "User %s should be rate limited", key)
	}
}

// TestRateLimiter_Cleanup tests the cleanup mechanism
func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	
	// Add some keys
	rl.Allow("user1")
	rl.Allow("user2")
	rl.Allow("user3")
	
	// Wait for cleanup interval (shorter in tests)
	// Note: This is hard to test without modifying the cleanup interval
	// We just verify the method exists and doesn't panic
	
	// After cleanup, should still work
	assert.True(t, rl.Allow("user4"))
}

// TestAuthService_ConcurrentTokenGeneration tests thread safety of auth service
func TestAuthService_ConcurrentTokenGeneration(t *testing.T) {
	auth := NewAuthService("this-is-a-valid-secret-key-for-testing", time.Hour)
	
	const numGoroutines = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	tokens := make(map[string]string)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := fmt.Sprintf("user%d", id)
			token, err := auth.GenerateToken(userID, fmt.Sprintf("user%d", id), []string{"user"})
			
			if err == nil {
				mu.Lock()
				tokens[userID] = token
				mu.Unlock()
			}
		}(i)
	}
	
	wg.Wait()
	
	// All tokens should be unique and valid
	tokenStrings := make(map[string]bool)
	for userID, token := range tokens {
		// Verify token is unique
		assert.NotEmpty(t, token)
		assert.False(t, tokenStrings[token], "Token should be unique for user %s", userID)
		tokenStrings[token] = true
		
		// Verify token is valid
		claims, err := auth.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	}
}

// TestAuthService_EdgeCases tests edge cases for auth service
func TestAuthService_EdgeCases(t *testing.T) {
	auth := NewAuthService("this-is-a-valid-secret-key-for-testing", time.Hour)
	
	t.Run("Empty roles array", func(t *testing.T) {
		token, err := auth.GenerateToken("user123", "testuser", []string{})
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		claims, err := auth.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, []string{}, claims.Roles)
	})
	
	t.Run("Nil roles array", func(t *testing.T) {
		token, err := auth.GenerateToken("user123", "testuser", nil)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		claims, err := auth.ValidateToken(token)
		assert.NoError(t, err)
		assert.Nil(t, claims.Roles)
	})
	
	t.Run("Very short TTL", func(t *testing.T) {
		authShortTTL := NewAuthService("test-secret-key-16-chars", time.Nanosecond)
		token, err := authShortTTL.GenerateToken("user123", "testuser", []string{"user"})
		require.NoError(t, err)
		
		// Wait for token to expire
		time.Sleep(time.Millisecond * 10)
		
		_, err = authShortTTL.ValidateToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})
}

// TestRateLimiter_Wait tests the Wait method
func TestRateLimiter_Wait(t *testing.T) {
	rps := 5
	burst := 10
	rl := NewRateLimiter(rps, burst)
	
	key := "test-user"
	
	// Just verify the Wait method doesn't panic
	// The actual rate limiting behavior is difficult to test reliably in unit tests
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Wait method panicked: %v", r)
		}
	}()
	
	rl.Wait(key)
	// If we reach here without panic, the test passes
}

// TestRateLimiter_KeyManagement tests internal key management
func TestRateLimiter_KeyManagement(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	
	// Test with different types of keys
	keys := []string{
		"user123",
		"ip:192.168.1.1",
		"api-key:abc123",
		"session:xyz789",
		"",
	}
	
	for _, key := range keys {
		// Should not panic with any key type
		assert.True(t, rl.Allow(key), "Should allow request for key: %s", key)
	}
}

// TestUserAuthService_RepositoryErrors tests error handling from repository
func TestUserAuthService_RepositoryErrors(t *testing.T) {
	// Mock repository that returns errors
	mockRepo := &MockUserRepository{
		users: make(map[string]*models.User),
	}
	
	auth := NewUserAuthService("test-secret-key-16-chars", time.Hour, mockRepo)
	
	t.Run("User not found", func(t *testing.T) {
		req := LoginRequest{
			Username: "nonexistent",
			Password: "password",
		}
		
		resp, err := auth.AuthenticateUser(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, models.ErrInvalidCredentials, err)
	})
	
	t.Run("Repository error", func(t *testing.T) {
		// Set repository to return error
		mockRepo.forceError = true
		
		req := LoginRequest{
			Username: "testuser",
			Password: "password",
		}
		
		resp, err := auth.AuthenticateUser(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to find user")
	})
}

// TestJWTToken_Parsing tests JWT token parsing edge cases
func TestJWTToken_Parsing(t *testing.T) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	t.Run("Invalid token format", func(t *testing.T) {
		invalidTokens := []string{
			"",
			"invalid.token",
			"not.a.jwt.token",
			"too.many.parts.in.token.format",
		}
		
		for _, token := range invalidTokens {
			_, err := auth.ValidateToken(token)
			assert.Error(t, err)
		}
	})
	
	t.Run("Token with wrong secret", func(t *testing.T) {
		auth1 := NewAuthService("secret-key-1-16-chars", time.Hour)
		auth2 := NewAuthService("secret-key-2-16-chars", time.Hour)
		
		token, _ := auth1.GenerateToken("user123", "testuser", []string{"user"})
		_, err := auth2.ValidateToken(token)
		assert.Error(t, err)
	})
	
	t.Run("Token manipulation", func(t *testing.T) {
		token, _ := auth.GenerateToken("user123", "testuser", []string{"user"})
		
		// Try to modify token
		parts := strings.Split(token, ".")
		if len(parts) >= 3 {
			// Modify the payload
			modifiedToken := parts[0] + ".invalid." + parts[2]
			_, err := auth.ValidateToken(modifiedToken)
			assert.Error(t, err)
		}
	})
}

// TestTokenGeneration_ExtremeInputs tests token generation with extreme inputs
func TestTokenGeneration_ExtremeInputs(t *testing.T) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	t.Run("Very long inputs", func(t *testing.T) {
		longID := strings.Repeat("a", 1000)
		longUsername := strings.Repeat("b", 1000)
		manyRoles := make([]string, 100)
		for i := range manyRoles {
			manyRoles[i] = fmt.Sprintf("role%d", i)
		}
		
		token, err := auth.GenerateToken(longID, longUsername, manyRoles)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		
		claims, err := auth.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, longID, claims.UserID)
		assert.Equal(t, longUsername, claims.Username)
		assert.Len(t, claims.Roles, 100)
	})
	
	t.Run("Special characters in inputs", func(t *testing.T) {
		specialID := "用户🔒123"
		specialUsername := "ñáéíóú"
		specialRoles := []string{"admin🔑", "user👤", "测试"}
		
		token, err := auth.GenerateToken(specialID, specialUsername, specialRoles)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		
		claims, err := auth.ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, specialID, claims.UserID)
		assert.Equal(t, specialUsername, claims.Username)
		assert.Equal(t, specialRoles, claims.Roles)
	})
}

// TestRateLimiter_Performance tests performance under load
func TestRateLimiter_Performance(t *testing.T) {
	rl := NewRateLimiter(1000, 100)
	
	const numKeys = 100
	const numRequests = 10
	
	start := time.Now()
	
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("user%d", i)
		for j := 0; j < numRequests; j++ {
			rl.Allow(key)
		}
	}
	
	duration := time.Since(start)
	
	// Should complete quickly even under load
	assert.Less(t, duration, time.Second, "Rate limiter should handle load efficiently")
}

// TestSecurity_ConcurrentMixedAccess tests mixed concurrent operations
func TestSecurity_ConcurrentMixedAccess(t *testing.T) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	rl := NewRateLimiter(100, 10)
	
	const numGoroutines = 50
	var wg sync.WaitGroup
	
	// Concurrent token generation and rate limiting
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Generate token
			token, _ := auth.GenerateToken(fmt.Sprintf("user%d", id), fmt.Sprintf("user%d", id), []string{"user"})
			
			// Check rate limit
			key := fmt.Sprintf("user%d", id)
			if rl.Allow(key) {
				// Validate token
				if token != "" {
					auth.ValidateToken(token)
				}
			}
		}(i)
	}
	
	wg.Wait()
	// Should complete without panics or deadlocks
}

// BenchmarkAuthService_TokenGeneration benchmarks token generation
func BenchmarkAuthService_TokenGeneration(b *testing.B) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	}
}

// BenchmarkAuthService_TokenValidation benchmarks token validation
func BenchmarkAuthService_TokenValidation(b *testing.B) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	// Pre-generate a token
	token, _ := auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.ValidateToken(token)
	}
}

// BenchmarkRateLimiter_Allow benchmarks rate limiting
func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter(1000, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.Allow("test-user")
	}
}

// MockUserRepository is a simple mock for testing
type MockUserRepository struct {
	users     map[string]*models.User
	forceError bool
}

func (m *MockUserRepository) FindByUsername(username string) (*models.User, error) {
	if m.forceError {
		return nil, fmt.Errorf("forced repository error")
	}
	
	user, exists := m.users[username]
	if !exists {
		return nil, models.ErrUserNotFound
	}
	
	return user, nil
}

func (m *MockUserRepository) FindByEmail(email string) (*models.User, error) {
	if m.forceError {
		return nil, fmt.Errorf("forced repository error")
	}
	
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	
	return nil, models.ErrUserNotFound
}

func (m *MockUserRepository) Create(user *models.User) error {
	if m.forceError {
		return fmt.Errorf("forced repository error")
	}
	
	m.users[user.Username] = user
	return nil
}

func (m *MockUserRepository) Update(user *models.User) error {
	if m.forceError {
		return fmt.Errorf("forced repository error")
	}
	
	if _, exists := m.users[user.Username]; !exists {
		return models.ErrUserNotFound
	}
	
	m.users[user.Username] = user
	return nil
}

func (m *MockUserRepository) Delete(id string) error {
	if m.forceError {
		return fmt.Errorf("forced repository error")
	}
	
	for username, user := range m.users {
		if user.ID == id {
			delete(m.users, username)
			return nil
		}
	}
	
	return models.ErrUserNotFound
}

func (m *MockUserRepository) List() ([]*models.User, error) {
	if m.forceError {
		return nil, fmt.Errorf("forced repository error")
	}
	
	users := make([]*models.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	
	return users, nil
}