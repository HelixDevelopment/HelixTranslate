package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"digital.vasic.translator/pkg/models"
)

// dummyPasswordHash is a fixed, valid bcrypt hash (of a random throwaway value)
// used ONLY to spend a comparable amount of CPU on the user-not-found path so
// that login response time does not reveal whether a username exists (CWE-208
// timing-based username enumeration). It is generated once at package init at
// the same DefaultCost the repository uses to store real password hashes, so the
// not-found path's bcrypt cost matches the wrong-password path's. No real
// password ever hashes to this value, and it is never compared as a credential
// of any account, so a successful compare against it cannot authenticate anyone.
var dummyPasswordHash []byte

func init() {
	// Hash a random value (not a constant) so the dummy hash is unguessable and
	// not shared across builds. DefaultCost matches models.InMemoryUserRepository.
	seed := make([]byte, 32)
	_, _ = rand.Read(seed)
	h, err := bcrypt.GenerateFromPassword(seed, bcrypt.DefaultCost)
	if err != nil {
		// bcrypt.GenerateFromPassword only errors on an out-of-range cost, which
		// DefaultCost is not; fall back to a known-valid precomputed cost-10 hash
		// so the timing mitigation is never silently disabled.
		h = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")
	}
	dummyPasswordHash = h
}

// UserAuthService extends AuthService with user validation
type UserAuthService struct {
	*AuthService
	userRepo models.UserRepository
}

// NewUserAuthService creates a new user authentication service
func NewUserAuthService(jwtSecret string, tokenTTL time.Duration, userRepo models.UserRepository) *UserAuthService {
	return &UserAuthService{
		AuthService: NewAuthService(jwtSecret, tokenTTL),
		userRepo:    userRepo,
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
			// Spend a comparable bcrypt computation against a dummy hash before
			// returning, so the user-not-found path's response time matches the
			// wrong-password path (which runs bcrypt over a real hash). Without
			// this, an unauthenticated attacker who times the login response can
			// distinguish existing usernames from non-existent ones even though
			// both return the identical ErrInvalidCredentials value (CWE-208
			// username enumeration via timing). The compare always fails (no
			// password equals the random dummy hash) — its only purpose is to
			// consume time, never to authenticate.
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(req.Password))
			return nil, models.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Validate password FIRST — before any account-status check. Reporting
	// IsActive before the password is verified leaks whether a username exists
	// and is inactive to an unauthenticated attacker (CWE-204 observable
	// response discrepancy → username enumeration): a wrong password on an
	// inactive account would otherwise return ErrUserInactive while a wrong
	// password on an active/unknown account returns ErrInvalidCredentials, and
	// the API layer maps those to distinct HTTP 403/401 responses. Validating
	// the password first makes the wrong-password response indistinguishable
	// regardless of account status.
	if err := user.ValidatePassword(req.Password); err != nil {
		return nil, models.ErrInvalidCredentials
	}

	// Only after the caller has proven they hold the correct password may we
	// disclose that the account is inactive.
	if !user.IsActive {
		return nil, models.ErrUserInactive
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
