package internal


import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// CustomCORSMiddleware creates a custom CORS middleware that properly handles credentials
func CustomCORSMiddleware() echo.MiddlewareFunc {
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
		"http://localhost:3000": true,
		"http://localhost:8081": true,
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")

			// Check if origin is allowed
			if allowedOrigins[origin] {
				c.Response().Header().Set("Access-Control-Allow-Origin", origin)
				c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if c.Request().Method == http.MethodOptions {
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
				c.Response().Header().Set("Access-Control-Max-Age", "3600")
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}
