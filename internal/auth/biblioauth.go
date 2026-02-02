package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BiblioAuthClient handles authentication with Biblio Auth service
type BiblioAuthClient struct {
	baseURL string
	client  *http.Client
}

// UserInfo represents user information from Biblio Auth
type UserInfo struct {
	ID       int      `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Groups   []string `json:"groups"`
}

// ValidateResponse represents the response from Biblio Auth validate endpoint
type ValidateResponse struct {
	Valid bool     `json:"valid"`
	User  UserInfo `json:"user"`
	Error string   `json:"error"`
}

// LoginResponse represents the response from Biblio Auth login endpoint
type LoginResponse struct {
	Success bool     `json:"success"`
	User    UserInfo `json:"user"`
	Error   string   `json:"error"`
}

// NewBiblioAuthClient creates a new Biblio Auth client
func NewBiblioAuthClient(baseURL string) *BiblioAuthClient {
	return &BiblioAuthClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateSession validates a session token with Biblio Auth
func (c *BiblioAuthClient) ValidateSession(token string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/validate", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add the auth token as a cookie
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: token,
	})

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validation failed with status: %d", resp.StatusCode)
	}

	var result ValidateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Valid {
		return nil, fmt.Errorf("invalid session: %s", result.Error)
	}

	return &result.User, nil
}

// ValidateBasicAuth validates username/password credentials with Biblio Auth
func (c *BiblioAuthClient) ValidateBasicAuth(username, password string) (*UserInfo, error) {
	loginReq := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/login", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid username or password")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("authentication failed: %s", result.Error)
	}

	return &result.User, nil
}

// GetLoginURL returns the URL to redirect users to for login
func (c *BiblioAuthClient) GetLoginURL(returnURL string) string {
	return fmt.Sprintf("%s/login?returnUrl=%s", c.baseURL, returnURL)
}

// GetLogoutURL returns the URL to redirect users to for logout
func (c *BiblioAuthClient) GetLogoutURL() string {
	return fmt.Sprintf("%s/api/logout", c.baseURL)
}

// IsAdmin checks if the user has admin privileges
func (c *BiblioAuthClient) IsAdmin(user *UserInfo) bool {
	for _, group := range user.Groups {
		if group == "admin" {
			return true
		}
	}
	return false
}
