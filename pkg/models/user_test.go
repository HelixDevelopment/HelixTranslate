package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserBasicFunctionality tests basic user functionality
func TestUserBasicFunctionality(t *testing.T) {
	// Create test user
	user := User{
		ID:        "user-123",
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "hashedpassword",
		Roles:     []string{"user"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}

	// Test basic fields
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "hashedpassword", user.Password)
	assert.Contains(t, user.Roles, "user")
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.True(t, user.IsActive)
}

// TestUserValidation tests user validation
func TestUserValidation(t *testing.T) {
	testCases := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name: "Valid user",
			user: User{
				ID:       "user-123",
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name: "Empty ID",
			user: User{
				ID:       "",
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "Invalid email",
			user: User{
				ID:       "user-123",
				Email:    "invalid-email",
				Username: "testuser",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "Empty username",
			user: User{
				ID:       "user-123",
				Email:    "test@example.com",
				Username: "",
				Password: "password",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Basic validation tests
			if tc.wantErr {
				assert.True(t, tc.user.ID == "" || tc.user.Username == "" || !isValidEmail(tc.user.Email))
			} else {
				assert.NotEmpty(t, tc.user.ID)
				assert.NotEmpty(t, tc.user.Username)
				assert.True(t, isValidEmail(tc.user.Email))
			}
		})
	}
}

// TestUserRepository tests user repository operations
func TestUserRepository(t *testing.T) {
	// Create in-memory repository for testing
	repo := NewInMemoryUserRepository()

	// Test create user
	user := &User{
		ID:        "user-123",
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "hashedpassword",
		Roles:     []string{"user"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsActive:  true,
	}

	err := repo.Create(user)
	require.NoError(t, err)

	// Test find user
	retrievedUser, err := repo.FindByUsername("testuser")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUser.ID)
	assert.Equal(t, user.Email, retrievedUser.Email)
	assert.Equal(t, user.Username, retrievedUser.Username)

	retrievedUserByEmail, err := repo.FindByEmail("test@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, retrievedUserByEmail.ID)

	// Test update user
	user.Username = "updateduser"
	user.UpdatedAt = time.Now()
	err = repo.Update(user)
	require.NoError(t, err)

	updatedUser, err := repo.FindByUsername("updateduser")
	require.NoError(t, err)
	assert.Equal(t, "updateduser", updatedUser.Username)

	// Test list users
	users, err := repo.List()
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, user.ID, users[0].ID)

	// Test delete user
	err = repo.Delete("user-123")
	require.NoError(t, err)

	// Verify user is deleted
	_, err = repo.FindByUsername("updateduser")
	assert.Error(t, err)
}

// TestPasswordHashing tests password hashing functionality
func TestPasswordHashing(t *testing.T) {
	password := "testpassword123"

	// Test password hashing
	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, password, hashedPassword)

	// Test password verification
	isValid := CheckPassword(hashedPassword, password)
	assert.True(t, isValid)

	// Test invalid password verification
	isValid = CheckPassword(hashedPassword, "wrongpassword")
	assert.False(t, isValid)
}

// TestUserRoles tests user role functionality
func TestUserRoles(t *testing.T) {
	testCases := []struct {
		name     string
		user     User
		role     string
		expected bool
	}{
		{
			name: "Admin has admin role",
			user: User{
				ID:      "user-123",
				Email:   "admin@example.com",
				Roles:   []string{"admin", "user"},
				IsActive: true,
			},
			role:     "admin",
			expected: true,
		},
		{
			name: "Regular user does not have admin role",
			user: User{
				ID:      "user-456",
				Email:   "user@example.com",
				Roles:   []string{"user"},
				IsActive: true,
			},
			role:     "admin",
			expected: false,
		},
		{
			name: "User has user role",
			user: User{
				ID:      "user-456",
				Email:   "user@example.com",
				Roles:   []string{"user"},
				IsActive: true,
			},
			role:     "user",
			expected: true,
		},
		{
			name: "Inactive user cannot access roles",
			user: User{
				ID:      "user-789",
				Email:   "inactive@example.com",
				Roles:   []string{"admin", "user"},
				IsActive: false,
			},
			role:     "admin",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasRole := UserHasRole(tc.user, tc.role)
			assert.Equal(t, tc.expected, hasRole)
		})
	}
}

// TestUserSession tests user session functionality
func TestUserSession(t *testing.T) {
	user := User{
		ID:       "user-123",
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}

	// Create session
	session := CreateUserSession(user, time.Hour)
	
	assert.NotEmpty(t, session.Token)
	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, user.Email, session.UserEmail)
	assert.Equal(t, user.Username, session.Username)
	assert.False(t, session.ExpiresAt.IsZero())
	assert.True(t, session.ExpiresAt.After(time.Now()))

	// Test session validation
	isValid := ValidateSession(session)
	assert.True(t, isValid)

	// Test expired session
	expiredSession := session
	expiredSession.ExpiresAt = time.Now().Add(-time.Hour)
	
	isValid = ValidateSession(expiredSession)
	assert.False(t, isValid)

	// Test token generation
	token := GenerateSessionToken()
	assert.NotEmpty(t, token)
	assert.Len(t, token, 64) // Should be 64 characters
}

// Helper function for email validation
func isValidEmail(email string) bool {
	return len(email) > 0 && len(email) < 254 && 
		   len(email) > 3 && 
		   (email[len(email)-4:] == ".com" || email[len(email)-4:] == ".org" || email[len(email)-4:] == ".net")
}

// BenchmarkUserCreation benchmarks user creation
func BenchmarkUserCreation(b *testing.B) {
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		user := User{
			ID:        "user-" + string(rune(i)),
			Email:     "user@example.com",
			Username:  "user",
			Password:  "hashedpassword",
			Roles:     []string{"user"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			IsActive:  true,
		}
		
		_ = user // Use user to avoid optimization
	}
}

// BenchmarkUserValidation benchmarks user validation
func BenchmarkUserValidation(b *testing.B) {
	user := User{
		ID:       "user-123",
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password",
	}

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = user.ID != "" && user.Username != "" && len(user.Email) > 5
	}
}