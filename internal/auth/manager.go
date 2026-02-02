package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"biblio-audiobook-builder-tts/internal/storage"
)

// AuthMode represents the authentication mode
type AuthMode string

const (
	AuthModeInternal   AuthMode = "internal"
	AuthModeBiblioAuth AuthMode = "biblio-auth"
)

// Manager handles authentication for internal and biblio-auth modes
type Manager struct {
	mode         AuthMode
	internalAuth *Auth             // Used in internal mode
	biblioAuth   *BiblioAuthClient // Used in biblio-auth mode
}

// NewManager creates a new authentication manager
func NewManager(mode string, database storage.AuthDB, biblioAuthURL string) (*Manager, error) {
	m := &Manager{
		mode: AuthMode(mode),
	}

	// Initialize internal auth for internal mode
	if m.mode == AuthModeInternal && database != nil {
		m.internalAuth = New(database)
	}

	// Initialize biblio-auth client if in biblio-auth mode
	if m.mode == AuthModeBiblioAuth && biblioAuthURL != "" {
		m.biblioAuth = NewBiblioAuthClient(biblioAuthURL)
	}

	return m, nil
}

// GetMode returns the current authentication mode
func (m *Manager) GetMode() AuthMode {
	return m.mode
}

// IsInternalMode returns true if using internal authentication
func (m *Manager) IsInternalMode() bool {
	return m.mode == AuthModeInternal
}

// IsBiblioAuthMode returns true if using Biblio Auth
func (m *Manager) IsBiblioAuthMode() bool {
	return m.mode == AuthModeBiblioAuth
}

// GetInternalAuth returns the internal auth provider
func (m *Manager) GetInternalAuth() *Auth {
	return m.internalAuth
}

// GetBiblioAuth returns the Biblio Auth client
func (m *Manager) GetBiblioAuth() *BiblioAuthClient {
	return m.biblioAuth
}

// Authenticate authenticates a user with username/password (for internal mode)
func (m *Manager) Authenticate(username, password string) (*storage.User, error) {
	if m.internalAuth == nil {
		return nil, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.Authenticate(username, password)
}

// CreateSession creates a session for a user (for internal mode)
func (m *Manager) CreateSession(userID int64) (*storage.Session, error) {
	if m.internalAuth == nil {
		return nil, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.CreateSession(userID)
}

// ValidateSession validates a session (for internal mode)
func (m *Manager) ValidateSession(sessionID string) (*storage.User, error) {
	if m.internalAuth == nil {
		return nil, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.ValidateSession(sessionID)
}

// ValidateBiblioAuthSession validates a Biblio Auth session token
func (m *Manager) ValidateBiblioAuthSession(token string) (*UserInfo, error) {
	if m.biblioAuth == nil {
		return nil, fmt.Errorf("biblio-auth not configured")
	}
	return m.biblioAuth.ValidateSession(token)
}

// GetLoginURL returns the Biblio Auth login URL
func (m *Manager) GetLoginURL(returnURL string) string {
	if m.biblioAuth == nil {
		return ""
	}
	return m.biblioAuth.GetLoginURL(returnURL)
}

// GetLogoutURL returns the Biblio Auth logout URL
func (m *Manager) GetLogoutURL() string {
	if m.biblioAuth == nil {
		return ""
	}
	return m.biblioAuth.GetLogoutURL()
}

// DeleteSession deletes a session (internal mode only)
func (m *Manager) DeleteSession(sessionID string) error {
	if m.internalAuth == nil {
		return fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.DeleteSession(sessionID)
}

// GetUsers returns all users (internal mode only)
func (m *Manager) GetUsers() ([]storage.User, error) {
	if m.internalAuth == nil {
		return nil, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.GetUsers()
}

// GetUser returns a user by ID (internal mode only)
func (m *Manager) GetUser(id int64) (*storage.User, error) {
	if m.internalAuth == nil {
		return nil, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.GetUser(id)
}

// CreateUser creates a new user (internal mode only)
func (m *Manager) CreateUser(username, password, role string) (*storage.User, error) {
	if m.internalAuth == nil {
		return nil, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.CreateUser(username, password, role)
}

// UpdateUserPassword updates a user's password (internal mode only)
func (m *Manager) UpdateUserPassword(id int64, newPassword string) error {
	if m.internalAuth == nil {
		return fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.UpdateUserPassword(id, newPassword)
}

// UpdateUserRole updates a user's role (internal mode only)
func (m *Manager) UpdateUserRole(id int64, role string) error {
	if m.internalAuth == nil {
		return fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.UpdateUserRole(id, role)
}

// DeleteUser deletes a user (internal mode only)
func (m *Manager) DeleteUser(id int64) error {
	if m.internalAuth == nil {
		return fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.DeleteUser(id)
}

// HasUsers checks if any users exist (internal mode only)
func (m *Manager) HasUsers() (bool, error) {
	if m.internalAuth == nil {
		return false, fmt.Errorf("internal auth not configured")
	}
	return m.internalAuth.HasUsers()
}

// CheckAuth checks for valid authentication and returns true if authenticated
// In biblio-auth mode, validates against Biblio Auth service
// In internal mode, validates against internal database
func (m *Manager) CheckAuth(w http.ResponseWriter, r *http.Request) bool {
	if m.mode == AuthModeBiblioAuth {
		return m.checkBiblioAuth(w, r)
	}
	return m.checkInternalAuth(w, r)
}

// checkBiblioAuth validates auth_token cookie against Biblio Auth
func (m *Manager) checkBiblioAuth(w http.ResponseWriter, r *http.Request) bool {
	// Check for auth_token cookie (from Biblio Auth web login)
	cookie, err := r.Cookie("auth_token")
	if err == nil {
		userInfo, err := m.biblioAuth.ValidateSession(cookie.Value)
		if err == nil {
			// Convert to storage.User for context
			user := &storage.User{
				ID:       int64(userInfo.ID),
				Username: userInfo.Username,
				Role:     storage.RoleUser,
			}
			for _, group := range userInfo.Groups {
				if group == "admin" {
					user.Role = storage.RoleAdmin
					break
				}
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			*r = *r.WithContext(ctx)
			return true
		}
		log.Printf("Biblio Auth session validation failed: %v", err)
	}

	// Try Basic Auth via Biblio Auth (for API clients)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				userInfo, err := m.biblioAuth.ValidateBasicAuth(parts[0], parts[1])
				if err == nil {
					log.Printf("Biblio Auth Basic Auth successful: user=%s", userInfo.Username)
					user := &storage.User{
						ID:       int64(userInfo.ID),
						Username: userInfo.Username,
						Role:     storage.RoleUser,
					}
					for _, group := range userInfo.Groups {
						if group == "admin" {
							user.Role = storage.RoleAdmin
							break
						}
					}
					ctx := context.WithValue(r.Context(), UserContextKey, user)
					*r = *r.WithContext(ctx)
					return true
				}
				log.Printf("Biblio Auth Basic Auth failed: %v", err)
			}
		}
	}

	return false
}

// checkInternalAuth validates session cookie or Basic Auth against internal database
func (m *Manager) checkInternalAuth(w http.ResponseWriter, r *http.Request) bool {
	if m.internalAuth == nil {
		return false
	}

	// Check session cookie first
	cookie, err := r.Cookie("session")
	if err == nil {
		user, err := m.internalAuth.ValidateSession(cookie.Value)
		if err == nil {
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			*r = *r.WithContext(ctx)
			return true
		}
	}

	// Try Basic Auth
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				user, err := m.internalAuth.Authenticate(parts[0], parts[1])
				if err == nil {
					ctx := context.WithValue(r.Context(), UserContextKey, user)
					*r = *r.WithContext(ctx)
					return true
				}
			}
		}
	}

	return false
}
