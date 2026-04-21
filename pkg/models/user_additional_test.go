package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_ValidatePassword(t *testing.T) {
	t.Run("Valid password", func(t *testing.T) {
		user := &User{
			Username: "testuser",
			Password: "plaintext",
		}
		// Hash the password first
		err := user.SetPassword("correctpassword")
		require.NoError(t, err)

		// Validate with correct password
		err = user.ValidatePassword("correctpassword")
		assert.NoError(t, err)
	})

	t.Run("Invalid password", func(t *testing.T) {
		user := &User{
			Username: "testuser",
			Password: "plaintext",
		}
		err := user.SetPassword("correctpassword")
		require.NoError(t, err)

		// Validate with wrong password
		err = user.ValidatePassword("wrongpassword")
		assert.Error(t, err)
	})

	t.Run("Empty password", func(t *testing.T) {
		user := &User{
			Username: "testuser",
			Password: "",
		}
		err := user.SetPassword("correctpassword")
		require.NoError(t, err)

		// Validate with empty password
		err = user.ValidatePassword("")
		assert.Error(t, err)
	})
}

func TestUser_SetPassword(t *testing.T) {
	t.Run("Set password hashes correctly", func(t *testing.T) {
		user := &User{
			Username: "testuser",
		}

		originalPassword := "mypassword123"
		err := user.SetPassword(originalPassword)
		require.NoError(t, err)

		// Password should be hashed (not plaintext)
		assert.NotEqual(t, originalPassword, user.Password)
		assert.NotEmpty(t, user.Password)

		// Should be able to validate
		err = user.ValidatePassword(originalPassword)
		assert.NoError(t, err)
	})

	t.Run("Set password twice overwrites", func(t *testing.T) {
		user := &User{
			Username: "testuser",
		}

		err := user.SetPassword("firstpassword")
		require.NoError(t, err)
		firstHash := user.Password

		err = user.SetPassword("secondpassword")
		require.NoError(t, err)
		secondHash := user.Password

		// Hashes should be different
		assert.NotEqual(t, firstHash, secondHash)

		// Old password should not work
		err = user.ValidatePassword("firstpassword")
		assert.Error(t, err)

		// New password should work
		err = user.ValidatePassword("secondpassword")
		assert.NoError(t, err)
	})

	t.Run("Set empty password", func(t *testing.T) {
		user := &User{
			Username: "testuser",
		}

		err := user.SetPassword("")
		require.NoError(t, err)

		// Empty password validation should work
		err = user.ValidatePassword("")
		assert.NoError(t, err)
	})
}

func TestInMemoryUserRepository_Create_PasswordHashing(t *testing.T) {
	repo := NewInMemoryUserRepository()

	user := &User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Password: "plaintext123",
		IsActive: true,
	}

	err := repo.Create(user)
	require.NoError(t, err)

	// Password should be hashed after creation
	assert.NotEqual(t, "plaintext123", user.Password)

	// Should be able to validate
	validateErr := user.ValidatePassword("plaintext123")
	assert.NoError(t, validateErr)
}

func TestInMemoryUserRepository_Update_UsernameChange(t *testing.T) {
	repo := NewInMemoryUserRepository()

	user := &User{
		ID:       "user-1",
		Username: "oldname",
		Email:    "test@example.com",
		Password: "password",
		IsActive: true,
	}

	err := repo.Create(user)
	require.NoError(t, err)

	// Update username
	user.Username = "newname"
	err = repo.Update(user)
	require.NoError(t, err)

	// Old username should not be found
	_, err = repo.FindByUsername("oldname")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)

	// New username should be found
	found, err := repo.FindByUsername("newname")
	require.NoError(t, err)
	assert.Equal(t, "newname", found.Username)
}

func TestInMemoryUserRepository_Delete_NotFound(t *testing.T) {
	repo := NewInMemoryUserRepository()

	err := repo.Delete("non-existent-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestInMemoryUserRepository_FindByEmail_NotFound(t *testing.T) {
	repo := NewInMemoryUserRepository()

	_, err := repo.FindByEmail("nonexistent@example.com")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestInMemoryUserRepository_List_Empty(t *testing.T) {
	repo := NewInMemoryUserRepository()

	users, err := repo.List()
	require.NoError(t, err)
	assert.Empty(t, users)
}
