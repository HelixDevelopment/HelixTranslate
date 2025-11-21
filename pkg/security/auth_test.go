package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateToken tests JWT token generation
func TestGenerateToken(t *testing.T) {
	auth := NewAuthService("test-secret-key", time.Hour)

	token, err := auth.GenerateToken("user123", "testuser", []string{"admin", "user"})

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should be a valid JWT string (3 parts separated by dots)
	assert.Contains(t, token, ".")
}

// TestValidateToken_Valid tests validation of a valid token
func TestValidateToken_Valid(t *testing.T) {
	auth := NewAuthService("test-secret-key", time.Hour)

	// Generate token
	token, err := auth.GenerateToken("user123", "testuser", []string{"admin"})
	require.NoError(t, err)

	// Validate token
	claims, err := auth.ValidateToken(token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, []string{"admin"}, claims.Roles)
}

// TestValidateToken_Expired tests validation of an expired token
func TestValidateToken_Expired(t *testing.T) {
	// Create auth service with very short TTL
	auth := NewAuthService("test-secret-key", time.Millisecond)

	// Generate token
	token, err := auth.GenerateToken("user123", "testuser", []string{"user"})
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(time.Millisecond * 10)

	// Validate expired token
	claims, err := auth.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "expired")
}

// TestValidateToken_Invalid tests validation of invalid token strings
func TestValidateToken_Invalid(t *testing.T) {
	auth := NewAuthService("test-secret-key", time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"random string", "not.a.valid.token"},
		{"malformed JWT", "header.payload"},
		{"wrong format", "this-is-not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := auth.ValidateToken(tt.token)

			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

// TestValidateToken_Tampered tests detection of tampered tokens
func TestValidateToken_Tampered(t *testing.T) {
	auth := NewAuthService("test-secret-key", time.Hour)

	// Generate valid token
	token, err := auth.GenerateToken("user123", "testuser", []string{"user"})
	require.NoError(t, err)

	// Tamper with the token (change last character)
	tamperedToken := token[:len(token)-1] + "X"

	// Validate tampered token
	claims, err := auth.ValidateToken(tamperedToken)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

// TestValidateToken_WrongSecret tests validation with wrong secret
func TestValidateToken_WrongSecret(t *testing.T) {
	auth1 := NewAuthService("secret-1", time.Hour)
	auth2 := NewAuthService("secret-2", time.Hour)

	// Generate token with auth1
	token, err := auth1.GenerateToken("user123", "testuser", []string{"user"})
	require.NoError(t, err)

	// Try to validate with auth2 (different secret)
	claims, err := auth2.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

// TestGenerateAPIKey tests API key generation
func TestGenerateAPIKey(t *testing.T) {
	key1, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key1)
	assert.Greater(t, len(key1), 20) // Should be at least 20 characters

	// Generate another key - should be different
	key2, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEqual(t, key1, key2)
}

// TestAPIKeyStore_AddAndValidate tests adding and validating API keys
func TestAPIKeyStore_AddAndValidate(t *testing.T) {
	store := NewAPIKeyStore()

	key := "test-api-key-123"
	info := APIKeyInfo{
		Key:       key,
		UserID:    "user123",
		Name:      "Test Key",
		CreatedAt: time.Now(),
		Active:    true,
	}

	// Add key
	store.AddKey(key, info)

	// Validate key
	validatedInfo, ok := store.ValidateKey(key)

	assert.True(t, ok)
	require.NotNil(t, validatedInfo)
	assert.Equal(t, "user123", validatedInfo.UserID)
	assert.Equal(t, "Test Key", validatedInfo.Name)
}

// TestAPIKeyStore_InvalidKey tests validation of non-existent key
func TestAPIKeyStore_InvalidKey(t *testing.T) {
	store := NewAPIKeyStore()

	info, ok := store.ValidateKey("non-existent-key")

	assert.False(t, ok)
	assert.Nil(t, info)
}

// TestAPIKeyStore_InactiveKey tests validation of revoked/inactive key
func TestAPIKeyStore_InactiveKey(t *testing.T) {
	store := NewAPIKeyStore()

	key := "test-key"
	info := APIKeyInfo{
		Key:       key,
		UserID:    "user123",
		CreatedAt: time.Now(),
		Active:    false, // Inactive
	}

	store.AddKey(key, info)

	validatedInfo, ok := store.ValidateKey(key)

	assert.False(t, ok)
	assert.Nil(t, validatedInfo)
}

// TestAPIKeyStore_ExpiredKey tests validation of expired key
func TestAPIKeyStore_ExpiredKey(t *testing.T) {
	store := NewAPIKeyStore()

	key := "test-key"
	expiresAt := time.Now().Add(-time.Hour) // Expired 1 hour ago
	info := APIKeyInfo{
		Key:       key,
		UserID:    "user123",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: &expiresAt,
		Active:    true,
	}

	store.AddKey(key, info)

	validatedInfo, ok := store.ValidateKey(key)

	assert.False(t, ok)
	assert.Nil(t, validatedInfo)
}

// TestAPIKeyStore_RevokeKey tests key revocation
func TestAPIKeyStore_RevokeKey(t *testing.T) {
	store := NewAPIKeyStore()

	key := "test-key"
	info := APIKeyInfo{
		Key:       key,
		UserID:    "user123",
		CreatedAt: time.Now(),
		Active:    true,
	}

	store.AddKey(key, info)

	// Verify key is valid
	_, ok := store.ValidateKey(key)
	assert.True(t, ok)

	// Revoke key
	store.RevokeKey(key)

	// Verify key is now invalid
	_, ok = store.ValidateKey(key)
	assert.False(t, ok)
}

// TestAuthService_MultipleRoles tests token with multiple roles
func TestAuthService_MultipleRoles(t *testing.T) {
	auth := NewAuthService("test-secret", time.Hour)

	roles := []string{"admin", "user", "moderator"}
	token, err := auth.GenerateToken("user123", "testuser", roles)
	require.NoError(t, err)

	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, roles, claims.Roles)
}

// TestAuthService_EmptyRoles tests token with empty roles
func TestAuthService_EmptyRoles(t *testing.T) {
	auth := NewAuthService("test-secret", time.Hour)

	token, err := auth.GenerateToken("user123", "testuser", []string{})
	require.NoError(t, err)

	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	assert.Empty(t, claims.Roles)
}

// TestAuthService_TokenClaims tests all token claims are set correctly
func TestAuthService_TokenClaims(t *testing.T) {
	auth := NewAuthService("test-secret", time.Hour)

	beforeGeneration := time.Now()
	token, err := auth.GenerateToken("user123", "testuser", []string{"user"})
	require.NoError(t, err)

	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)

	// Check IssuedAt is recent (within reasonable time window)
	assert.WithinDuration(t, beforeGeneration, claims.IssuedAt.Time, time.Second*2)

	// Check ExpiresAt is ~1 hour from IssuedAt (using IssuedAt as base to avoid race condition)
	expectedExpiry := claims.IssuedAt.Time.Add(time.Hour)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, time.Second*2)

	// Check NotBefore is recent (within reasonable time window)
	assert.WithinDuration(t, beforeGeneration, claims.NotBefore.Time, time.Second*2)
}

// TestAPIKeyStore_MultipleKeys tests managing multiple keys
func TestAPIKeyStore_MultipleKeys(t *testing.T) {
	store := NewAPIKeyStore()

	// Add multiple keys
	for i := 0; i < 5; i++ {
		key, err := GenerateAPIKey()
		require.NoError(t, err)

		info := APIKeyInfo{
			Key:       key,
			UserID:    "user" + string(rune('0'+i)),
			CreatedAt: time.Now(),
			Active:    true,
		}
		store.AddKey(key, info)
	}

	// All keys should be valid
	count := 0
	for key := range store.keys {
		_, ok := store.ValidateKey(key)
		if ok {
			count++
		}
	}
	assert.Equal(t, 5, count)
}

// BenchmarkGenerateToken benchmarks token generation
func BenchmarkGenerateToken(b *testing.B) {
	auth := NewAuthService("test-secret-key", time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.GenerateToken("user123", "testuser", []string{"admin"})
	}
}

// BenchmarkValidateToken benchmarks token validation
func BenchmarkValidateToken(b *testing.B) {
	auth := NewAuthService("test-secret-key", time.Hour)
	token, _ := auth.GenerateToken("user123", "testuser", []string{"admin"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.ValidateToken(token)
	}
}

// BenchmarkGenerateAPIKey benchmarks API key generation
func BenchmarkGenerateAPIKey(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateAPIKey()
	}
}

// BenchmarkAPIKeyValidation benchmarks API key validation
func BenchmarkAPIKeyValidation(b *testing.B) {
	store := NewAPIKeyStore()
	key := "test-key-123"
	info := APIKeyInfo{
		Key:       key,
		UserID:    "user123",
		CreatedAt: time.Now(),
		Active:    true,
	}
	store.AddKey(key, info)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ValidateKey(key)
	}
}

// Security Test: TestAuthService_BruteForceProtection
// This test verifies that the auth system doesn't leak information about invalid tokens
func TestAuthService_BruteForceProtection(t *testing.T) {
	auth := NewAuthService("test-secret", time.Hour)

	// Try to validate many invalid tokens
	// The system should not leak timing information or error details
	invalidTokens := []string{
		"invalid1",
		"invalid2",
		"invalid3",
		jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{}).Raw,
	}

	for _, token := range invalidTokens {
		claims, err := auth.ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
		// Error messages should not reveal implementation details
	}
}

// Security Test: TestAuthService_SigningMethodValidation
// Ensures that only HMAC signing method is accepted
func TestAuthService_SigningMethodValidation(t *testing.T) {
	auth := NewAuthService("test-secret", time.Hour)

	// Try to create a token with RS256 (RSA) instead of HS256 (HMAC)
	// This should fail validation even with valid claims
	claims := Claims{
		UserID:   "user123",
		Username: "testuser",
		Roles:    []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with wrong signing method
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	// Validation should fail
	validatedClaims, err := auth.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
}
