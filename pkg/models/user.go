package models

import (
	"time"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never expose password in JSON
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`
}

// UserRepository defines user storage interface
type UserRepository interface {
	FindByUsername(username string) (*User, error)
	FindByEmail(email string) (*User, error)
	Create(user *User) error
	Update(user *User) error
	Delete(id string) error
	List() ([]*User, error)
}

// InMemoryUserRepository is an in-memory implementation for testing/small deployments
type InMemoryUserRepository struct {
	users map[string]*User
}

// NewInMemoryUserRepository creates a new in-memory user repository
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users: make(map[string]*User),
	}
}

// FindByUsername finds a user by username
func (r *InMemoryUserRepository) FindByUsername(username string) (*User, error) {
	if user, exists := r.users[username]; exists {
		return user, nil
	}
	return nil, ErrUserNotFound
}

// FindByEmail finds a user by email
func (r *InMemoryUserRepository) FindByEmail(email string) (*User, error) {
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

// Create creates a new user
func (r *InMemoryUserRepository) Create(user *User) error {
	// Hash password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	r.users[user.Username] = user
	return nil
}

// Update updates a user
func (r *InMemoryUserRepository) Update(user *User) error {
	if _, exists := r.users[user.Username]; !exists {
		return ErrUserNotFound
	}
	user.UpdatedAt = time.Now()
	r.users[user.Username] = user
	return nil
}

// Delete deletes a user
func (r *InMemoryUserRepository) Delete(id string) error {
	for username, user := range r.users {
		if user.ID == id {
			delete(r.users, username)
			return nil
		}
	}
	return ErrUserNotFound
}

// List returns all users
func (r *InMemoryUserRepository) List() ([]*User, error) {
	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

// ValidatePassword validates a password against the stored hash
func (u *User) ValidatePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}

// SetPassword sets a new password (hashes it)
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}