package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"biblio-audiobook-builder-tts/internal/storage"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrSessionExpired     = errors.New("session expired")
	ErrUnauthorized       = errors.New("unauthorized")
)

const (
	SessionDuration = 24 * time.Hour * 30 // 30 days
	BcryptCost      = 12
)

// Auth handles internal authentication with local database
type Auth struct {
	db storage.AuthDB
}

// New creates a new Auth instance
func New(database storage.AuthDB) *Auth {
	return &Auth{db: database}
}

// HashPassword hashes a password using bcrypt
func (a *Auth) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a hash
func (a *Auth) CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser creates a new user
func (a *Auth) CreateUser(username, password, role string) (*storage.User, error) {
	existing, _ := a.db.GetUserByUsername(username)
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := a.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &storage.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}

	id, err := a.db.CreateUser(user)
	if err != nil {
		return nil, err
	}

	user.ID = id
	return user, nil
}

// Authenticate validates username and password
func (a *Auth) Authenticate(username, password string) (*storage.User, error) {
	user, err := a.db.GetUserByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !a.CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// CreateSession creates a new session for a user
func (a *Auth) CreateSession(userID int64) (*storage.Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &storage.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	if err := a.db.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession validates a session ID and returns the user
func (a *Auth) ValidateSession(sessionID string) (*storage.User, error) {
	session, err := a.db.GetSession(sessionID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if time.Now().After(session.ExpiresAt) {
		a.db.DeleteSession(sessionID)
		return nil, ErrSessionExpired
	}

	user, err := a.db.GetUserByID(session.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return user, nil
}

// DeleteSession deletes a session
func (a *Auth) DeleteSession(sessionID string) error {
	return a.db.DeleteSession(sessionID)
}

// DeleteExpiredSessions removes all expired sessions
func (a *Auth) DeleteExpiredSessions() error {
	return a.db.DeleteExpiredSessions()
}

// HasUsers checks if any users exist
func (a *Auth) HasUsers() (bool, error) {
	count, err := a.db.CountUsers()
	return count > 0, err
}

// GetUser returns a user by ID
func (a *Auth) GetUser(id int64) (*storage.User, error) {
	return a.db.GetUserByID(id)
}

// GetUsers returns all users
func (a *Auth) GetUsers() ([]storage.User, error) {
	return a.db.GetUsers()
}

// UpdateUserPassword updates a user's password
func (a *Auth) UpdateUserPassword(userID int64, newPassword string) error {
	hash, err := a.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return a.db.UpdateUserPassword(userID, hash)
}

// UpdateUserRole updates a user's role
func (a *Auth) UpdateUserRole(userID int64, role string) error {
	return a.db.UpdateUserRole(userID, role)
}

// DeleteUser deletes a user
func (a *Auth) DeleteUser(userID int64) error {
	return a.db.DeleteUser(userID)
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
