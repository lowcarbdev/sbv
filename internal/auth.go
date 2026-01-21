package internal

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var authDB *sql.DB

// InitAuthDB initializes the authentication database
func InitAuthDB(filepath string) error {
	var err error
	authDB, err = sql.Open("sqlite3", filepath)
	if err != nil {
		return err
	}

	if err = authDB.Ping(); err != nil {
		return err
	}

	// Set busy timeout for better concurrent access
	_, err = authDB.Exec("PRAGMA busy_timeout=5000;")
	if err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable WAL mode if requested (better for concurrent reads during writes)
	if UseWALMode {
		_, err = authDB.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			return fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS settings (
		user_id TEXT PRIMARY KEY,
		settings_json TEXT NOT NULL DEFAULT '{}',
		updated_at INTEGER NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`

	_, err = authDB.Exec(createTableSQL)
	return err
}

// CreateUser creates a new user with hashed password
func CreateUser(username, password string) (*User, error) {
	// Generate UUID for user
	userID := uuid.New().String()

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	createdAt := time.Now().Unix()

	_, err = authDB.Exec(
		"INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)",
		userID, username, string(hashedPassword), createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &User{
		ID:           userID,
		Username:     username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Unix(createdAt, 0),
	}, nil
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(username string) (*User, error) {
	var user User
	var createdAt int64

	err := authDB.QueryRow(
		"SELECT id, username, password_hash, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &createdAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	return &user, nil
}

// VerifyPassword checks if the provided password matches the user's password hash
func VerifyPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// GetUsernameByID retrieves username by user ID
func GetUsernameByID(userID string) (string, error) {
	var username string
	err := authDB.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found")
		}
		return "", err
	}
	return username, nil
}

// GenerateSessionID generates a random session ID
func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session for a user
func CreateSession(userID string, username string) (*Session, error) {
	sessionID, err := GenerateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	createdAt := time.Now()
	expiresAt := createdAt.Add(30 * 24 * time.Hour) // 30 days

	_, err = authDB.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		sessionID, userID, createdAt.Unix(), expiresAt.Unix(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &Session{
		ID:        sessionID,
		UserID:    userID,
		Username:  username,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

// GetSession retrieves a session by ID
func GetSession(sessionID string) (*Session, error) {
	var session Session
	var createdAt, expiresAt int64

	err := authDB.QueryRow(
		`SELECT s.id, s.user_id, u.username, s.created_at, s.expires_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = ?`,
		sessionID,
	).Scan(&session.ID, &session.UserID, &session.Username, &createdAt, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}

	session.CreatedAt = time.Unix(createdAt, 0)
	session.ExpiresAt = time.Unix(expiresAt, 0)

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		DeleteSession(sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// DeleteSession deletes a session by ID
func DeleteSession(sessionID string) error {
	_, err := authDB.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

// CleanExpiredSessions removes all expired sessions
func CleanExpiredSessions() error {
	_, err := authDB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().Unix())
	return err
}

// UpdatePassword updates a user's password
func UpdatePassword(userID string, newPassword string) error {
	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = authDB.Exec(
		"UPDATE users SET password_hash = ? WHERE id = ?",
		string(hashedPassword), userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// ListUsers returns all users in the database
func ListUsers() ([]User, error) {
	rows, err := authDB.Query("SELECT id, username, password_hash, created_at FROM users ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var createdAt int64
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.CreatedAt = time.Unix(createdAt, 0)
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}
