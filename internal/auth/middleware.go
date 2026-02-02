package auth

import (
	"context"
	"net/http"

	"biblio-audiobook-builder-tts/internal/storage"
)

type contextKey string

const UserContextKey contextKey = "user"

// GetUserFromContext retrieves user info from request context
func GetUserFromContext(ctx context.Context) *storage.User {
	user, ok := ctx.Value(UserContextKey).(*storage.User)
	if !ok {
		return nil
	}
	return user
}

// SessionMiddleware extracts user from session cookie and adds to context
func (a *Auth) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := a.ValidateSession(cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth middleware requires authentication
func (a *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin middleware requires admin role
func (a *Auth) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r.Context())
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Not authenticated. Please log in again."}`))
			return
		}
		if !user.IsAdmin() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"Admin access required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CheckSession checks for session auth and returns true if authenticated
func (a *Auth) CheckSession(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	user, err := a.ValidateSession(cookie.Value)
	if err != nil {
		return false
	}

	ctx := context.WithValue(r.Context(), UserContextKey, user)
	*r = *r.WithContext(ctx)
	return true
}

// CheckAdmin checks if the current user is an admin
func (a *Auth) CheckAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := GetUserFromContext(r.Context())
	if user == nil {
		return false
	}
	return user.IsAdmin()
}
