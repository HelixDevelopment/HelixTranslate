package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
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
	return &AuthService{
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

// GenerateToken generates a JWT token
func (as *AuthService) GenerateToken(userID, username string, roles []string) (string, error) {
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
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return as.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// APIKeyStore manages API keys
type APIKeyStore struct {
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
	aks.keys[key] = info
}

// ValidateKey validates an API key
func (aks *APIKeyStore) ValidateKey(key string) (*APIKeyInfo, bool) {
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
	if info, ok := aks.keys[key]; ok {
		info.Active = false
		aks.keys[key] = info
	}
}
