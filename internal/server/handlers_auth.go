package server

import (
	"encoding/json"
	"net/http"

	"biblio-audiobook-builder-tts/internal/auth"
	"biblio-audiobook-builder-tts/internal/storage"
)

// AuthInfoResponse represents the response for auth info endpoint
type AuthInfoResponse struct {
	Mode          string `json:"mode"`
	Authenticated bool   `json:"authenticated"`
	User          *struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user,omitempty"`
	LoginURL  string `json:"login_url,omitempty"`
	LogoutURL string `json:"logout_url,omitempty"`
}

// handleAuthInfo returns authentication info and current user
func (s *Server) handleAuthInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := AuthInfoResponse{
		Mode:          string(s.authManager.GetMode()),
		Authenticated: false,
	}

	// Check if user is authenticated
	if s.authManager.CheckAuth(w, r) {
		response.Authenticated = true
		user := auth.GetUserFromContext(r.Context())
		if user != nil {
			response.User = &struct {
				ID       int64  `json:"id"`
				Username string `json:"username"`
				Role     string `json:"role"`
			}{
				ID:       user.ID,
				Username: user.Username,
				Role:     user.Role,
			}
		}
	}

	// Add login/logout URLs for biblio-auth mode
	if s.authManager.IsBiblioAuthMode() {
		response.LoginURL = s.authManager.GetLoginURL("")
		response.LogoutURL = s.authManager.GetLogoutURL()
	}

	s.jsonResponse(w, http.StatusOK, response)
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	User    *struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user,omitempty"`
}

// handleAuthLogin handles login requests (internal mode only)
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// In biblio-auth mode, redirect to Biblio Auth login
	if s.authManager.IsBiblioAuthMode() {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"redirect": s.authManager.GetLoginURL(""),
		})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonResponse(w, http.StatusBadRequest, LoginResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Authenticate user
	user, err := s.authManager.Authenticate(req.Username, req.Password)
	if err != nil {
		s.jsonResponse(w, http.StatusUnauthorized, LoginResponse{
			Success: false,
			Error:   "Invalid username or password",
		})
		return
	}

	// Create session
	session, err := s.authManager.CreateSession(user.ID)
	if err != nil {
		s.jsonResponse(w, http.StatusInternalServerError, LoginResponse{
			Success: false,
			Error:   "Failed to create session",
		})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	})

	s.jsonResponse(w, http.StatusOK, LoginResponse{
		Success: true,
		User: &struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Role     string `json:"role"`
		}{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	})
}

// handleAuthLogout handles logout requests
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// In biblio-auth mode, redirect to Biblio Auth logout
	if s.authManager.IsBiblioAuthMode() {
		http.Redirect(w, r, s.authManager.GetLogoutURL(), http.StatusFound)
		return
	}

	// Internal mode: delete session
	cookie, err := r.Cookie("session")
	if err == nil {
		s.authManager.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	s.jsonResponse(w, http.StatusOK, map[string]bool{"success": true})
}

// SetupCheckResponse represents the response for setup check
type SetupCheckResponse struct {
	SetupRequired bool   `json:"setup_required"`
	Mode          string `json:"mode"`
}

// handleSetupCheck checks if initial setup is required (internal mode only)
func (s *Server) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := SetupCheckResponse{
		Mode:          string(s.authManager.GetMode()),
		SetupRequired: false,
	}

	// Setup is only required in internal mode when no users exist
	if s.authManager.IsInternalMode() {
		hasUsers, err := s.authManager.HasUsers()
		if err == nil && !hasUsers {
			response.SetupRequired = true
		}
	}

	s.jsonResponse(w, http.StatusOK, response)
}

// SetupRequest represents a setup request
type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleSetup handles initial admin setup (internal mode only)
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Setup only available in internal mode
	if !s.authManager.IsInternalMode() {
		s.jsonError(w, http.StatusBadRequest, "Setup not available in biblio-auth mode")
		return
	}

	// Check if users already exist
	hasUsers, err := s.authManager.HasUsers()
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to check users")
		return
	}
	if hasUsers {
		s.jsonError(w, http.StatusBadRequest, "Setup already completed")
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		s.jsonError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Create admin user
	user, err := s.authManager.CreateUser(req.Username, req.Password, storage.RoleAdmin)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to create admin user")
		return
	}

	// Create session
	session, err := s.authManager.CreateSession(user.ID)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	})

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
