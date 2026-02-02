package storage

import (
	"database/sql"
	"time"
)

// User roles
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User represents a user in the database
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthDB defines the interface for authentication database operations
type AuthDB interface {
	// User operations
	CreateUser(user *User) (int64, error)
	GetUserByID(id int64) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUsers() ([]User, error)
	UpdateUserPassword(id int64, passwordHash string) error
	UpdateUserRole(id int64, role string) error
	DeleteUser(id int64) error
	CountUsers() (int, error)

	// Session operations
	CreateSession(session *Session) error
	GetSession(id string) (*Session, error)
	DeleteSession(id string) error
	DeleteExpiredSessions() error
	DeleteUserSessions(userID int64) error
}

// MigrateAuth creates the authentication tables
func (db *DB) MigrateAuth() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// CreateUser creates a new user in the database
func (db *DB) CreateUser(user *User) (int64, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	result, err := db.conn.Exec(`
		INSERT INTO users (username, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, user.Username, user.PasswordHash, user.Role, time.Now(), time.Now())
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(id int64) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user := &User{}
	err := db.conn.QueryRow(`
		SELECT id, username, password_hash, role, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user := &User{}
	err := db.conn.QueryRow(`
		SELECT id, username, password_hash, role, created_at, updated_at
		FROM users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUsers retrieves all users
func (db *DB) GetUsers() ([]User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rows, err := db.conn.Query(`
		SELECT id, username, password_hash, role, created_at, updated_at
		FROM users ORDER BY username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateUserPassword updates a user's password hash
func (db *DB) UpdateUserPassword(id int64, passwordHash string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?
	`, passwordHash, time.Now(), id)
	return err
}

// UpdateUserRole updates a user's role
func (db *DB) UpdateUserRole(id int64, role string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		UPDATE users SET role = ?, updated_at = ? WHERE id = ?
	`, role, time.Now(), id)
	return err
}

// DeleteUser deletes a user by ID
func (db *DB) DeleteUser(id int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// CountUsers returns the number of users
func (db *DB) CountUsers() (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// CreateSession creates a new session
func (db *DB) CreateSession(session *Session) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec(`
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, session.ID, session.UserID, session.ExpiresAt, time.Now())
	return err
}

// GetSession retrieves a session by ID
func (db *DB) GetSession(id string) (*Session, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	session := &Session{}
	err := db.conn.QueryRow(`
		SELECT id, user_id, expires_at, created_at
		FROM sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession deletes a session by ID
func (db *DB) DeleteSession(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// DeleteExpiredSessions removes all expired sessions
func (db *DB) DeleteExpiredSessions() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (db *DB) DeleteUserSessions(userID int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}
