package security

import (
	"testing"
	"time"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Test data structures
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	IP       string `json:"ip,omitempty"`
}

type AuthConfig struct {
	JWTSecret     string        `json:"jwt_secret"`
	TokenExpiry   time.Duration `json:"token_expiry"`
	PasswordCost  int           `json:"password_cost"`
	RateLimitRPS  int           `json:"rate_limit_rps"`
	RateLimitBurst int          `json:"rate_limit_burst"`
}

type UserAuth struct {
	config      AuthConfig
	userStore   map[string]*User
	rateLimiter map[string]*RateLimitState
}

type RateLimitState struct {
	Tokens    int
	LastSeen  time.Time
}

type TokenClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	UserID   string `json:"user_id"`
}

func TestUserAuth_AuthenticateUser(t *testing.T) {
	auth := NewUserAuth(AuthConfig{
		JWTSecret:    "test-secret-key-16",
		TokenExpiry:  time.Hour,
		PasswordCost: 12,
	})
	
	// Create test user
	testUser := &User{
		ID:       "user123",
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	// Add user to store (AddUserToStore will hash the password)
	auth.AddUserToStore(&User{
		ID:       testUser.ID,
		Username: testUser.Username,
		Password: testUser.Password, // Plain text password, will be hashed by AddUserToStore
		Email:    testUser.Email,
		Role:     testUser.Role,
	})
	
	// Test 1: Valid authentication
	t.Run("ValidAuthentication", func(t *testing.T) {
		authenticatedUser, err := auth.AuthenticateUser("testuser", "password123")
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, authenticatedUser.ID)
		assert.Equal(t, testUser.Username, authenticatedUser.Username)
		assert.Equal(t, testUser.Email, authenticatedUser.Email)
		assert.Equal(t, testUser.Role, authenticatedUser.Role)
		// Password should not be included in response
		assert.Empty(t, authenticatedUser.Password)
	})
	
	// Test 2: Invalid password
	t.Run("InvalidPassword", func(t *testing.T) {
		_, err := auth.AuthenticateUser("testuser", "wrongpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})
	
	// Test 3: Non-existent user
	t.Run("NonExistentUser", func(t *testing.T) {
		_, err := auth.AuthenticateUser("nonexistent", "password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
	
	// Test 4: Empty username
	t.Run("EmptyUsername", func(t *testing.T) {
		_, err := auth.AuthenticateUser("", "password123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")
	})
	
	// Test 5: Empty password
	t.Run("EmptyPassword", func(t *testing.T) {
		_, err := auth.AuthenticateUser("testuser", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password cannot be empty")
	})
}

func TestUserAuth_GenerateToken(t *testing.T) {
	auth := NewUserAuth(AuthConfig{
		JWTSecret:   "test-secret-key",
		TokenExpiry: time.Hour,
	})
	
	user := &User{
		ID:       "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	// Test 1: Generate token
	t.Run("GenerateToken", func(t *testing.T) {
		token, err := auth.GenerateToken(user)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		
		// Validate token structure
		assert.Greater(t, len(token), 50) // JWT tokens are long
		assert.Contains(t, token, ".") // JWT has three parts separated by dots
	})
	
	// Test 2: Validate token
	t.Run("ValidateToken", func(t *testing.T) {
		token, err := auth.GenerateToken(user)
		require.NoError(t, err)
		
		claims, err := auth.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Username, claims.Username)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	})
	
	// Test 3: Expired token
	t.Run("ExpiredToken", func(t *testing.T) {
		expiredAuth := NewUserAuth(AuthConfig{
			JWTSecret:   "test-secret-key",
			TokenExpiry: -time.Hour, // Already expired
		})
		
		expiredToken, err := expiredAuth.GenerateToken(user)
		require.NoError(t, err)
		
		_, err = auth.ValidateToken(expiredToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is expired")
	})
	
	// Test 4: Invalid token format
	t.Run("InvalidTokenFormat", func(t *testing.T) {
		invalidTokens := []string{
			"invalid-token",
			"only.two.parts",
			"",
			"not.a.jwt.token.at.all",
		}
		
		for _, token := range invalidTokens {
			_, err := auth.ValidateToken(token)
			assert.Error(t, err, "Token '%s' should be invalid", token)
		}
	})
	
	// Test 5: Token with wrong signature
	t.Run("WrongSignature", func(t *testing.T) {
		token, err := auth.GenerateToken(user)
		require.NoError(t, err)
		
		// Modify token to make signature invalid
		parts := []rune(token)
		if len(parts) > 5 {
			parts[len(parts)-3] = 'X' // Modify signature
			modifiedToken := string(parts)
			
			_, err = auth.ValidateToken(modifiedToken)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "signature is invalid")
		}
	})
}

func TestUserAuth_HashPassword(t *testing.T) {
	auth := NewUserAuth(AuthConfig{
		PasswordCost: 12,
	})
	
	password := "test-password-123"
	
	// Test 1: Hash password
	t.Run("HashPassword", func(t *testing.T) {
		hashedPassword, err := auth.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hashedPassword)
		assert.NotEqual(t, password, hashedPassword)
		assert.Greater(t, len(hashedPassword), 50) // bcrypt hashes are long
		assert.Contains(t, hashedPassword, "$") // bcrypt format includes $
	})
	
	// Test 2: Verify password
	t.Run("VerifyPassword", func(t *testing.T) {
		hashedPassword, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		err = auth.VerifyPassword(hashedPassword, password)
		require.NoError(t, err)
	})
	
	// Test 3: Invalid password
	t.Run("InvalidPassword", func(t *testing.T) {
		hashedPassword, err := auth.HashPassword(password)
		require.NoError(t, err)
		
		err = auth.VerifyPassword(hashedPassword, "wrong-password")
		assert.Error(t, err)
		assert.Equal(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})
	
	// Test 4: Invalid hash format
	t.Run("InvalidHashFormat", func(t *testing.T) {
		invalidHashes := []string{
			"invalid-hash",
			"$2a$12", // incomplete
			"$invalid$format$here",
		}
		
		for _, hash := range invalidHashes {
			err := auth.VerifyPassword(hash, password)
			assert.Error(t, err, "Hash '%s' should be invalid", hash)
		}
	})
	
	// Test 5: Different password costs
	t.Run("DifferentPasswordCosts", func(t *testing.T) {
		costs := []int{10, 12, 14}
		
		for _, cost := range costs {
			auth := NewUserAuth(AuthConfig{
				PasswordCost: cost,
			})
			
			hashedPassword, err := auth.HashPassword(password)
			require.NoError(t, err)
			
			err = auth.VerifyPassword(hashedPassword, password)
			require.NoError(t, err)
			
			// Check that cost is correctly embedded in hash
			assert.Contains(t, hashedPassword, fmt.Sprintf("$2a$%02d$", cost))
		}
	})
}

func TestUserAuth_RateLimiting(t *testing.T) {
	auth := NewUserAuth(AuthConfig{
		JWTSecret:      "test-secret-key-16",
		TokenExpiry:    time.Hour,
		RateLimitRPS:   5,
		RateLimitBurst: 10,
	})
	
	user := &User{
		ID:       "user123",
		Username: "testuser",
		IP:       "192.168.1.1",
	}
	
	// Test 1: Within rate limit
	t.Run("WithinRateLimit", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			allowed := auth.CheckRateLimit(user)
			assert.True(t, allowed, "Request %d should be allowed", i)
			time.Sleep(10 * time.Millisecond) // Small delay between requests
		}
	})
	
	// Test 2: Exceeds rate limit
	t.Run("ExceedsRateLimit", func(t *testing.T) {
		// Reset rate limiter for this test
		auth.rateLimiter = make(map[string]*RateLimitState)
		
		// Exceed burst limit quickly
		exceeded := false
		for i := 0; i < 15; i++ {
			allowed := auth.CheckRateLimit(user)
			if !allowed {
				exceeded = true
				break
			}
		}
		assert.True(t, exceeded, "Should have been rate limited after burst")
	})
	
	// Test 3: Rate limit recovery
	t.Run("RateLimitRecovery", func(t *testing.T) {
		// Reset rate limiter
		auth.rateLimiter = make(map[string]*RateLimitState)
		
		// Exhaust burst limit
		for i := 0; i < 10; i++ {
			auth.CheckRateLimit(user)
		}
		
		// Should be rate limited
		allowed := auth.CheckRateLimit(user)
		assert.False(t, allowed)
		
		// Wait for token refill (actually wait for time to pass)
		time.Sleep(time.Second)
		
		// Should be allowed again after token refill
		allowed = auth.CheckRateLimit(user)
		assert.True(t, allowed)
	})
	
	// Test 4: Different users have separate limits
	t.Run("SeparateUserLimits", func(t *testing.T) {
		user1 := &User{Username: "user1", IP: "192.168.1.1"}
		user2 := &User{Username: "user2", IP: "192.168.1.2"}
		
		// Reset rate limiter
		auth.rateLimiter = make(map[string]*RateLimitState)
		
		// Exhaust user1's limit
		for i := 0; i < 10; i++ {
			auth.CheckRateLimit(user1)
		}
		
		// user1 should be rate limited
		allowed1 := auth.CheckRateLimit(user1)
		assert.False(t, allowed1)
		
		// user2 should still be allowed
		allowed2 := auth.CheckRateLimit(user2)
		assert.True(t, allowed2)
	})
}

func TestUserAuth_UserManagement(t *testing.T) {
	auth := NewUserAuth(AuthConfig{
		JWTSecret:   "test-secret-key",
		TokenExpiry: time.Hour,
	})
	
	// Test 1: Add user to store
	t.Run("AddUser", func(t *testing.T) {
		user := &User{
			ID:       "user123",
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
			Role:     "user",
		}
		
		err := auth.AddUserToStore(user)
		require.NoError(t, err)
		
		// Verify user exists
		storedUser, exists := auth.userStore[user.Username]
		assert.True(t, exists)
		assert.Equal(t, user.ID, storedUser.ID)
		assert.Equal(t, user.Username, storedUser.Username)
		assert.Equal(t, user.Email, storedUser.Email)
	})
	
	// Test 2: Add duplicate user
	t.Run("AddDuplicateUser", func(t *testing.T) {
		user := &User{
			Username: "testuser", // Same username as above
			Email:    "different@example.com",
		}
		
		err := auth.AddUserToStore(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user already exists")
	})
	
	// Test 3: Get user from store
	t.Run("GetUser", func(t *testing.T) {
		user, err := auth.GetUserFromStore("testuser")
		require.NoError(t, err)
		assert.Equal(t, "user123", user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
	})
	
	// Test 4: Get non-existent user
	t.Run("GetNonExistentUser", func(t *testing.T) {
		_, err := auth.GetUserFromStore("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
	
	// Test 5: Update user
	t.Run("UpdateUser", func(t *testing.T) {
		updatedUser := &User{
			ID:       "user123",
			Username: "testuser",
			Email:    "updated@example.com",
			Role:     "admin",
		}
		
		err := auth.UpdateUserInStore(updatedUser)
		require.NoError(t, err)
		
		user, err := auth.GetUserFromStore("testuser")
		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", user.Email)
		assert.Equal(t, "admin", user.Role)
	})
	
	// Test 6: Delete user
	t.Run("DeleteUser", func(t *testing.T) {
		err := auth.DeleteUserFromStore("testuser")
		require.NoError(t, err)
		
		_, err = auth.GetUserFromStore("testuser")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserAuth_SecurityEdgeCases(t *testing.T) {
	auth := NewUserAuth(AuthConfig{
		JWTSecret:    "test-secret-key-16",
		TokenExpiry:  time.Hour,
		PasswordCost: 12,
	})
	
	// Test 1: SQL injection in username
	t.Run("SQLInjectionUsername", func(t *testing.T) {
		maliciousUsername := "user'; DROP TABLE users; --"
		
		_, err := auth.AuthenticateUser(maliciousUsername, "password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
	
	// Test 2: XSS in email
	t.Run("XSSInEmail", func(t *testing.T) {
		user := &User{
			ID:       "user123",
			Username: "testuser",
			Email:    "<script>alert('xss')</script>@example.com",
			Password: "password123",
		}
		
		// This should be handled safely
		err := auth.AddUserToStore(user)
		require.NoError(t, err)
		
		retrievedUser, err := auth.GetUserFromStore("testuser")
		require.NoError(t, err)
		assert.Equal(t, user.Email, retrievedUser.Email)
	})
	
	// Test 3: Very long username
	t.Run("VeryLongUsername", func(t *testing.T) {
		longUsername := string(make([]byte, 1000)) // 1000 character username
		
		user := &User{
			Username: longUsername,
			Email:    "test@example.com",
		}
		
		err := auth.AddUserToStore(user)
		// Should either succeed or fail gracefully without panicking
		assert.True(t, err == nil || len(err.Error()) > 0)
	})
	
	// Test 4: Unicode characters
	t.Run("UnicodeCharacters", func(t *testing.T) {
		user := &User{
			ID:       "用户123",
			Username: "用户名",
			Email:    "用户@example.com",
			Password: "密码123",
		}
		
		err := auth.AddUserToStore(user)
		require.NoError(t, err)
		
		authenticatedUser, err := auth.AuthenticateUser("用户名", "密码123")
		require.NoError(t, err)
		assert.Equal(t, user.ID, authenticatedUser.ID)
	})
	
	// Test 5: Token with invalid algorithm
	t.Run("InvalidAlgorithm", func(t *testing.T) {
		// Create a token with "none" algorithm
		token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
			"username": "testuser",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})
		
		noneToken, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		
		_, err = auth.ValidateToken(noneToken)
		assert.Error(t, err)
	})
}

// Mock implementations
func NewUserAuth(config AuthConfig) *UserAuth {
	return &UserAuth{
		config:      config,
		userStore:   make(map[string]*User),
		rateLimiter: make(map[string]*RateLimitState),
	}
}

func (ua *UserAuth) AuthenticateUser(username, password string) (*User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}
	
	user, exists := ua.userStore[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Return user without password
	return &User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}, nil
}

func (ua *UserAuth) GenerateToken(user *User) (string, error) {
	claims := TokenClaims{
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		UserID:   user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ua.config.TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "translator-system",
			Subject:   user.ID,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(ua.config.JWTSecret))
}

func (ua *UserAuth) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(ua.config.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, fmt.Errorf("invalid token")
}

func (ua *UserAuth) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), ua.config.PasswordCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (ua *UserAuth) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (ua *UserAuth) CheckRateLimit(user *User) bool {
	key := user.Username
	if user.IP != "" {
		key = user.IP
	}
	
	state, exists := ua.rateLimiter[key]
	if !exists {
		state = &RateLimitState{
			Tokens:   ua.config.RateLimitBurst,
			LastSeen: time.Now(),
		}
		ua.rateLimiter[key] = state
	}
	
	// Calculate time elapsed
	now := time.Now()
	elapsed := now.Sub(state.LastSeen)
	state.LastSeen = now
	
	// Add tokens based on elapsed time
	tokensToAdd := float64(elapsed) * float64(ua.config.RateLimitRPS) / float64(time.Second)
	state.Tokens += int(tokensToAdd)
	
	if state.Tokens > ua.config.RateLimitBurst {
		state.Tokens = ua.config.RateLimitBurst
	}
	
	if state.Tokens > 0 {
		state.Tokens--
		return true
	}
	
	return false
}

func (ua *UserAuth) AddUserToStore(user *User) error {
	if _, exists := ua.userStore[user.Username]; exists {
		return fmt.Errorf("user already exists")
	}
	
	hashedPassword, err := ua.HashPassword(user.Password)
	if err != nil {
		return err
	}
	
	user.Password = hashedPassword
	ua.userStore[user.Username] = user
	return nil
}

func (ua *UserAuth) GetUserFromStore(username string) (*User, error) {
	user, exists := ua.userStore[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	return user, nil
}

func (ua *UserAuth) UpdateUserInStore(user *User) error {
	if _, exists := ua.userStore[user.Username]; !exists {
		return fmt.Errorf("user not found")
	}
	
	ua.userStore[user.Username] = user
	return nil
}

func (ua *UserAuth) DeleteUserFromStore(username string) error {
	if _, exists := ua.userStore[username]; !exists {
		return fmt.Errorf("user not found")
	}
	
	delete(ua.userStore, username)
	return nil
}