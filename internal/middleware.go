package internal


import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AuthMiddleware checks for a valid session cookie
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get session ID from cookie
		cookie, err := c.Cookie("session_id")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Unauthorized: No session found",
			})
		}

		// Validate session
		session, err := GetSession(cookie.Value)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Unauthorized: Invalid or expired session",
			})
		}

		// Store session in context for use by handlers
		c.Set("session", session)
		c.Set("user_id", session.UserID)
		c.Set("username", session.Username)

		return next(c)
	}
}

// NoCacheMiddleware adds cache control headers to prevent browser caching
// This ensures that dynamic API responses are always fetched fresh from the server
func NoCacheMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Set headers to prevent caching
		c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, private")
		c.Response().Header().Set("Pragma", "no-cache")
		c.Response().Header().Set("Expires", "0")

		return next(c)
	}
}
