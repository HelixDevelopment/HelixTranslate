package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAuthService tests auth service creation
func TestNewAuthService(t *testing.T) {
	// Test valid creation
	jwtSecret := "this-is-a-valid-secret-key-for-testing"
	tokenTTL := 24 * time.Hour
	
	auth := NewAuthService(jwtSecret, tokenTTL)
	require.NotNil(t, auth)
	assert.Equal(t, []byte(jwtSecret), auth.jwtSecret)
	assert.Equal(t, tokenTTL, auth.tokenTTL)

	// Test panic with short secret
	assert.Panics(t, func() {
		NewAuthService("short", tokenTTL)
	})
}

// TestAuthService_GenerateToken tests token generation
func TestAuthService_GenerateToken(t *testing.T) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	// Test successful token generation
	token, err := auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Test with empty userID
	_, err = auth.GenerateToken("", "testuser", []string{"user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "userID cannot be empty")

	// Test with empty username
	_, err = auth.GenerateToken("user123", "", []string{"user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username cannot be empty")

	// Test with zero TTL
	authZeroTTL := NewAuthService("test-secret-key-16-chars", 0)
	_, err = authZeroTTL.GenerateToken("user123", "testuser", []string{"user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token TTL must be positive")

	// Test with negative TTL
	authNegTTL := NewAuthService("test-secret-key-16-chars", -time.Hour)
	_, err = authNegTTL.GenerateToken("user123", "testuser", []string{"user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token TTL must be positive")
}

// TestAuthService_ValidateToken tests token validation
func TestAuthService_ValidateToken(t *testing.T) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	// Generate a valid token
	token, err := auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	require.NoError(t, err)

	// Test valid token validation
	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, []string{"user", "admin"}, claims.Roles)

	// Test with empty token
	_, err = auth.ValidateToken("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token cannot be empty")

	// Test with invalid token
	_, err = auth.ValidateToken("invalid.token.here")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature is invalid")

	// Test with token signed with different secret
	otherAuth := NewAuthService("different-secret-key-16", time.Hour)
	token2, err := otherAuth.GenerateToken("user456", "otheruser", []string{"user"})
	require.NoError(t, err)
	
	_, err = auth.ValidateToken(token2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature is invalid")

	// Test expired token
	authExpired := NewAuthService("test-secret-key-16-chars", 1*time.Millisecond)
	expiredToken, err := authExpired.GenerateToken("user123", "testuser", []string{"user"})
	require.NoError(t, err)
	
	time.Sleep(10 * time.Millisecond) // Wait for token to expire
	_, err = authExpired.ValidateToken(expiredToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

// TestAuthService_RefreshToken tests token refresh
func TestAuthService_RefreshToken(t *testing.T) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	// Generate original token
	originalToken, err := auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	require.NoError(t, err)
	
	// Validate original token to get claims
	originalClaims, err := auth.ValidateToken(originalToken)
	require.NoError(t, err)
	
	// Test successful refresh
	newToken, err := auth.RefreshToken(originalClaims)
	require.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, originalToken, newToken) // Should be different token
	
	// Validate new token
	newClaims, err := auth.ValidateToken(newToken)
	require.NoError(t, err)
	assert.Equal(t, originalClaims.UserID, newClaims.UserID)
	assert.Equal(t, originalClaims.Username, newClaims.Username)
	assert.Equal(t, originalClaims.Roles, newClaims.Roles)
	
	// New token should have later expiration time
	assert.True(t, newClaims.ExpiresAt.After(originalClaims.ExpiresAt.Time))

	// Test with nil claims
	_, err = auth.RefreshToken(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claims cannot be nil")
}

// TestGenerateAPIKey tests API key generation
func TestGenerateAPIKey(t *testing.T) {
	// Test successful generation
	apiKey, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, apiKey)
	assert.Equal(t, 44, len(apiKey)) // Base64 encoding of 32 bytes

	// Test multiple keys are unique
	key1, _ := GenerateAPIKey()
	key2, _ := GenerateAPIKey()
	assert.NotEqual(t, key1, key2)
}

// TestAPIKeyStore tests API key store operations
func TestAPIKeyStore(t *testing.T) {
	store := NewAPIKeyStore()
	require.NotNil(t, store)

	// Test adding and validating key
	apiKey := "test-api-key-123"
	info := APIKeyInfo{
		Key:       apiKey,
		UserID:    "user123",
		Name:      "Test Key",
		CreatedAt: time.Now(),
		Active:    true,
	}

	store.AddKey(apiKey, info)

	// Test valid key validation
	retrievedInfo, valid := store.ValidateKey(apiKey)
	require.True(t, valid)
	require.NotNil(t, retrievedInfo)
	assert.Equal(t, info.UserID, retrievedInfo.UserID)
	assert.Equal(t, info.Name, retrievedInfo.Name)
	assert.True(t, retrievedInfo.Active)

	// Test invalid key
	_, valid = store.ValidateKey("non-existent-key")
	assert.False(t, valid)

	// Test revoked key
	store.RevokeKey(apiKey)
	_, valid = store.ValidateKey(apiKey)
	assert.False(t, valid)

	// Verify key info is updated
	retrievedInfo, _ = store.ValidateKey(apiKey)
	if retrievedInfo != nil {
		assert.False(t, retrievedInfo.Active)
	}
}

// TestAPIKeyStore_Expiration tests key expiration
func TestAPIKeyStore_Expiration(t *testing.T) {
	store := NewAPIKeyStore()
	
	apiKey := "test-expiring-key"
	expiredTime := time.Now().Add(-1 * time.Hour) // Already expired
	info := APIKeyInfo{
		Key:       apiKey,
		UserID:    "user123",
		Name:      "Expired Key",
		CreatedAt:  time.Now(),
		ExpiresAt: &expiredTime,
		Active:    true,
	}

	store.AddKey(apiKey, info)

	// Test expired key
	_, valid := store.ValidateKey(apiKey)
	assert.False(t, valid)

	// Test non-expiring key (nil ExpiresAt)
	nonExpiringKey := "test-non-expiring-key"
	nonExpiringInfo := APIKeyInfo{
		Key:       nonExpiringKey,
		UserID:    "user456",
		Name:      "Non-Expiring Key",
		CreatedAt:  time.Now(),
		ExpiresAt: nil, // No expiration
		Active:    true,
	}

	store.AddKey(nonExpiringKey, nonExpiringInfo)
	_, valid = store.ValidateKey(nonExpiringKey)
	assert.True(t, valid)
}

// TestAPIKeyStore_InactiveKeys tests inactive key validation
func TestAPIKeyStore_InactiveKeys(t *testing.T) {
	store := NewAPIKeyStore()
	
	apiKey := "test-inactive-key"
	info := APIKeyInfo{
		Key:       apiKey,
		UserID:    "user123",
		Name:      "Inactive Key",
		CreatedAt:  time.Now(),
		Active:    false, // Inactive from creation
	}

	store.AddKey(apiKey, info)

	// Test inactive key
	_, valid := store.ValidateKey(apiKey)
	assert.False(t, valid)

	// Test activating key
	info.Active = true
	store.AddKey(apiKey, info)
	
	retrievedInfo, valid := store.ValidateKey(apiKey)
	assert.True(t, valid)
	assert.True(t, retrievedInfo.Active)
}

// TestClaims tests JWT claims structure
func TestClaims(t *testing.T) {
	now := time.Now()
	claims := Claims{
		UserID:   "user123",
		Username: "testuser",
		Roles:    []string{"user", "admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, []string{"user", "admin"}, claims.Roles)
	assert.Equal(t, now.Add(time.Hour).Unix(), claims.ExpiresAt.Unix())
	assert.Equal(t, now.Unix(), claims.IssuedAt.Unix())
	assert.Equal(t, now.Unix(), claims.NotBefore.Unix())
}

// TestAPIKeyInfo tests API key info structure
func TestAPIKeyInfo(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	info := APIKeyInfo{
		Key:       "test-api-key-123",
		UserID:    "user123",
		Name:      "Test API Key",
		CreatedAt: now,
		ExpiresAt: &future,
		Active:    true,
	}

	assert.Equal(t, "test-api-key-123", info.Key)
	assert.Equal(t, "user123", info.UserID)
	assert.Equal(t, "Test API Key", info.Name)
	assert.Equal(t, now, info.CreatedAt)
	assert.Equal(t, &future, info.ExpiresAt)
	assert.True(t, info.Active)
}

// Benchmark tests for performance
func BenchmarkAuthService_GenerateToken(b *testing.B) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	}
}

func BenchmarkAuthService_ValidateToken(b *testing.B) {
	auth := NewAuthService("test-secret-key-16-chars", time.Hour)
	token, _ := auth.GenerateToken("user123", "testuser", []string{"user", "admin"})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.ValidateToken(token)
	}
}

func BenchmarkGenerateAPIKey(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateAPIKey()
	}
}

func BenchmarkAPIKeyStore_ValidateKey(b *testing.B) {
	store := NewAPIKeyStore()
	apiKey, _ := GenerateAPIKey()
	info := APIKeyInfo{
		Key:       apiKey,
		UserID:    "user123",
		Name:      "Benchmark Key",
		CreatedAt:  time.Now(),
		Active:    true,
	}
	store.AddKey(apiKey, info)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ValidateKey(apiKey)
	}
}