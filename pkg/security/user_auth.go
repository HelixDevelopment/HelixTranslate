package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"digital.vasic.translator/pkg/models"
)

// UserAuthService extends AuthService with user validation
type UserAuthService struct {
	*AuthService
	userRepo models.UserRepository
}

// NewUserAuthService creates a new user authentication service
func NewUserAuthService(jwtSecret string, tokenTTL time.Duration, userRepo models.UserRepository) *UserAuthService {
	return &UserAuthService{
		AuthService: NewAuthService(jwtSecret, tokenTTL),
		userRepo:   userRepo,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token    string        `json:"token"`
	UserID   string        `json:"user_id"`
	Username string        `json:"username"`
	Roles    []string      `json:"roles"`
	TokenTTL time.Duration `json:"token_ttl"`
}

// AuthenticateUser authenticates a user and generates a token
func (uas *UserAuthService) AuthenticateUser(req LoginRequest) (*LoginResponse, error) {
	// Find user by username
	user, err := uas.userRepo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, models.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, models.ErrUserInactive
	}

	// Validate password
	if err := user.ValidatePassword(req.Password); err != nil {
		return nil, models.ErrInvalidCredentials
	}

	// Generate token
	token, err := uas.GenerateToken(user.ID, user.Username, user.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &LoginResponse{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		TokenTTL: uas.tokenTTL,
	}, nil
}

// ValidateUser validates a user's existence and status
func (uas *UserAuthService) ValidateUser(userID string) (*models.User, error) {
	// Find user by ID
	users, err := uas.userRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	for _, user := range users {
		if user.ID == userID {
			if !user.IsActive {
				return nil, models.ErrUserInactive
			}
			return user, nil
		}
	}

	return nil, models.ErrUserNotFound
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Roles    []string `json:"roles"`
}

// CreateUser creates a new user
func (uas *UserAuthService) CreateUser(req CreateUserRequest) (*models.User, error) {
	// Check if user already exists
	_, err := uas.userRepo.FindByUsername(req.Username)
	if err == nil {
		return nil, models.ErrUserAlreadyExists
	}
	if !errors.Is(err, models.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Check if email already exists
	_, err = uas.userRepo.FindByEmail(req.Email)
	if err == nil {
		return nil, models.ErrUserAlreadyExists
	}
	if !errors.Is(err, models.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}

	// Set default roles if none provided
	if len(req.Roles) == 0 {
		req.Roles = []string{"user"}
	}

	// Create user
	user := &models.User{
		ID:       generateUserID(),
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // Will be hashed by repository
		Roles:    req.Roles,
		IsActive: true,
	}

	if err := uas.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Clear password before returning
	user.Password = ""
	return user, nil
}

// generateUserID generates a unique user ID
func generateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}