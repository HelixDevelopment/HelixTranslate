package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthService handles authentication
type AuthService struct {
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(jwtSecret string, tokenTTL time.Duration) *AuthService {
	// Validate secret key
	if len(jwtSecret) < 16 {
		panic("JWT secret key must be at least 16 characters long")
	}
	return &AuthService{
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

// GenerateToken generates a JWT token
func (as *AuthService) GenerateToken(userID, username string, roles []string) (string, error) {
	// Validate inputs
	if userID == "" {
		return "", errors.New("userID cannot be empty")
	}
	if username == "" {
		return "", errors.New("username cannot be empty")
	}
	if as.tokenTTL <= 0 {
		return "", errors.New("token TTL must be positive")
	}

	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(as.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(as.jwtSecret)
}

// ValidateToken validates a JWT token
func (as *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	// Validate input
	if tokenString == "" {
		return nil, errors.New("token cannot be empty")
	}

	start := time.Now()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return as.jwtSecret, nil
	})

	// Add small artificial delay for invalid tokens to prevent brute force
	if err != nil {
		// Sleep at least 10 microseconds for security
		elapsed := time.Since(start)
		if elapsed < 10*time.Microsecond {
			time.Sleep(10*time.Microsecond - elapsed)
		}
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshToken generates a new token with extended expiration.
//
// A refresh MUST NOT resurrect a dead session: it re-checks that the supplied
// claims are still live before minting a new token. Without this, a caller (or
// attacker) holding the claims of an EXPIRED session could obtain a perpetually
// fresh token, extending a terminated session indefinitely — a "refresh accepts
// expired token without re-validation" auth bypass (see
// TestAdv_JWT_RefreshRejectsExpiredClaims). Claims with no expiry at all are
// likewise not a bounded, live session and are rejected.
func (as *AuthService) RefreshToken(claims *Claims) (string, error) {
	if claims == nil {
		return "", errors.New("claims cannot be nil")
	}
	if claims.ExpiresAt == nil {
		return "", errors.New("cannot refresh claims without an expiry")
	}
	if !claims.ExpiresAt.After(time.Now()) {
		return "", errors.New("cannot refresh an expired token")
	}

	newClaims := Claims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Roles:    claims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(as.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return token.SignedString(as.jwtSecret)
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// APIKeyStore manages API keys.
//
// It is safe for concurrent use: the underlying map is guarded by an RWMutex
// (mirroring RateLimiter in this package). Without this lock, concurrent
// AddKey/RevokeKey (writers) and ValidateKey (readers) race the map and the Go
// runtime fatally panics ("concurrent map read and map write"), taking down a
// security-critical surface — see TestAdv_APIKeyStore_ConcurrentAccessNoRace.
type APIKeyStore struct {
	mu   sync.RWMutex
	keys map[string]APIKeyInfo
}

// APIKeyInfo contains API key metadata
type APIKeyInfo struct {
	Key       string
	UserID    string
	Name      string
	CreatedAt time.Time
	ExpiresAt *time.Time
	Active    bool
}

// NewAPIKeyStore creates a new API key store
func NewAPIKeyStore() *APIKeyStore {
	return &APIKeyStore{
		keys: make(map[string]APIKeyInfo),
	}
}

// AddKey adds an API key
func (aks *APIKeyStore) AddKey(key string, info APIKeyInfo) {
	aks.mu.Lock()
	defer aks.mu.Unlock()
	aks.keys[key] = info
}

// ValidateKey validates an API key
func (aks *APIKeyStore) ValidateKey(key string) (*APIKeyInfo, bool) {
	aks.mu.RLock()
	defer aks.mu.RUnlock()
	info, ok := aks.keys[key]
	if !ok {
		return nil, false
	}

	if !info.Active {
		return nil, false
	}

	if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
		return nil, false
	}

	return &info, true
}

// RevokeKey revokes an API key
func (aks *APIKeyStore) RevokeKey(key string) {
	aks.mu.Lock()
	defer aks.mu.Unlock()
	if info, ok := aks.keys[key]; ok {
		info.Active = false
		aks.keys[key] = info
	}
}
