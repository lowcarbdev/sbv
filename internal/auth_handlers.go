package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// setSessionCookie sets the session cookie, marking it Secure when
// SECURE_COOKIES=true or the request arrived over HTTPS (c.Scheme()
// honors X-Forwarded-Proto for reverse proxy deployments)
func setSessionCookie(c echo.Context, value string, expires time.Time) {
	c.SetCookie(&http.Cookie{
		Name:     "session_id",
		Value:    value,
		Expires:  expires,
		HttpOnly: true,
		Secure:   os.Getenv("SECURE_COOKIES") == "true" || c.Scheme() == "https",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}

// RegistrationEnabled reports whether new user sign-ups are allowed
func RegistrationEnabled() bool {
	return os.Getenv("DISABLE_REGISTRATION") != "true"
}

// HandleConfig returns public configuration for the frontend
func HandleConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]bool{
		"registration_enabled": RegistrationEnabled(),
	})
}

func HandleRegister(c echo.Context) error {
	if !RegistrationEnabled() {
		return c.JSON(http.StatusForbidden, AuthResponse{
			Success: false,
			Error:   "Registration is disabled",
		})
	}

	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate input
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username is required",
		})
	}
	if len(req.Username) < 3 {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username must be at least 3 characters",
		})
	}
	if len(req.Password) < 6 {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Password must be at least 6 characters",
		})
	}

	// Create user
	user, err := CreateUser(req.Username, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return c.JSON(http.StatusConflict, AuthResponse{
				Success: false,
				Error:   "Username already exists",
			})
		}
		slog.Error("Error creating user", "error", err)
		return c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create user",
		})
	}

	// Create session
	session, err := CreateSession(user.ID, user.Username)
	if err != nil {
		slog.Error("Error creating session", "error", err)
		return c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create session",
		})
	}

	// Set session cookie
	setSessionCookie(c, session.ID, session.ExpiresAt)

	// Initialize user's database (using UUID as filename)
	dbPathPrefix := os.Getenv("DB_PATH_PREFIX")
	if dbPathPrefix == "" {
		dbPathPrefix = "."
	}
	userDBPath := fmt.Sprintf("%s/sbv_%s.db", dbPathPrefix, user.ID)
	if err := InitUserDB(user.ID, userDBPath); err != nil {
		slog.Error("Error initializing user database", "error", err)
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		User:    user,
		Session: session,
	})
}

func HandleLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate input
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Username and password are required",
		})
	}

	// Get user
	user, err := GetUserByUsername(req.Username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid username or password",
		})
	}

	// Verify password
	if !VerifyPassword(user, req.Password) {
		return c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid username or password",
		})
	}

	// Create session
	session, err := CreateSession(user.ID, user.Username)
	if err != nil {
		slog.Error("Error creating session", "error", err)
		return c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create session",
		})
	}

	// Set session cookie
	setSessionCookie(c, session.ID, session.ExpiresAt)

	return c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		User:    user,
		Session: session,
	})
}

func HandleLogout(c echo.Context) error {
	// Get session ID from cookie
	cookie, err := c.Cookie("session_id")
	if err == nil {
		// Delete session from database
		DeleteSession(cookie.Value)
	}

	// Clear cookie
	setSessionCookie(c, "", time.Now().Add(-1*time.Hour))

	return c.JSON(http.StatusOK, map[string]bool{
		"success": true,
	})
}

func HandleMe(c echo.Context) error {
	// Get session from context (set by AuthMiddleware)
	session, ok := c.Get("session").(*Session)
	if !ok {
		return c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Unauthorized",
		})
	}

	// Get user
	user, err := GetUserByUsername(session.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to get user info",
		})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Success: true,
		User:    user,
		Session: session,
	})
}

func HandleChangePassword(c echo.Context) error {
	// Get session from context (set by AuthMiddleware)
	session, ok := c.Get("session").(*Session)
	if !ok {
		return c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Unauthorized",
		})
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Validate input
	if req.OldPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "All fields are required",
		})
	}

	if req.NewPassword != req.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "New passwords do not match",
		})
	}

	if len(req.NewPassword) < 6 {
		return c.JSON(http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "New password must be at least 6 characters",
		})
	}

	// Get user
	user, err := GetUserByUsername(session.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to get user info",
		})
	}

	// Verify old password
	if !VerifyPassword(user, req.OldPassword) {
		return c.JSON(http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Current password is incorrect",
		})
	}

	// Update password
	if err := UpdatePassword(user.ID, req.NewPassword); err != nil {
		slog.Error("Error updating password", "error", err)
		return c.JSON(http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to update password",
		})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Success: true,
	})
}
