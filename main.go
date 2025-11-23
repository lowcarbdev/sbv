package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lowcarbdev/sbv/internal"
)

var logger *slog.Logger

func main() {
	// Initialize slog logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Initialize authentication database
	dbPathPrefix := os.Getenv("DB_PATH_PREFIX")
	if dbPathPrefix == "" {
		dbPathPrefix = "."
	}
	authDBPath := dbPathPrefix + "/sbv.db"

	err := internal.InitAuthDB(authDBPath)
	if err != nil {
		logger.Error("Failed to initialize authentication database", "error", err)
		os.Exit(1)
	}
	logger.Info("Authentication database initialized", "path", authDBPath)

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Use custom CORS middleware that properly handles credentials
	e.Use(internal.CustomCORSMiddleware())

	// Configure timeouts for large file uploads
	e.Server.ReadTimeout = 30 * time.Minute
	e.Server.WriteTimeout = 30 * time.Minute
	e.Server.ReadHeaderTimeout = 1 * time.Minute
	e.Server.IdleTimeout = 2 * time.Minute
	e.Server.MaxHeaderBytes = 1 << 20 // 1 MB max header size

	// Public routes (no authentication required)
	// Apply NoCacheMiddleware to prevent browser caching of auth responses
	e.POST("/api/auth/register", internal.HandleRegister, internal.NoCacheMiddleware)
	e.POST("/api/auth/login", internal.HandleLogin, internal.NoCacheMiddleware)
	e.POST("/api/auth/logout", internal.HandleLogout, internal.NoCacheMiddleware)

	// Protected routes (authentication required)
	protected := e.Group("/api")
	protected.Use(internal.AuthMiddleware)
	protected.Use(internal.NoCacheMiddleware) // Prevent browser caching of API responses

	protected.GET("/auth/me", internal.HandleMe)
	protected.POST("/auth/change-password", internal.HandleChangePassword)
	protected.POST("/upload", internal.HandleUpload)
	protected.GET("/conversations", internal.HandleConversations)
	protected.GET("/messages", internal.HandleMessages)
	protected.GET("/activity", internal.HandleActivity)
	protected.GET("/calls", internal.HandleCalls)
	protected.GET("/daterange", internal.HandleDateRange)
	protected.GET("/progress", internal.HandleProgress)
	protected.GET("/media", internal.HandleMedia)
	protected.GET("/search", internal.HandleSearch)

	// Health check
	e.GET("/api/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// Version endpoint (public, no authentication required)
	e.GET("/api/version", internal.HandleVersion)

	// Serve static files from frontend/dist if it exists (for production/Docker)
	if _, err := os.Stat("./frontend/dist"); err == nil {
		// Serve static assets (JS, CSS, images, etc.)
		e.Static("/assets", "./frontend/dist/assets")
		e.File("/favicon.ico", "./frontend/dist/favicon.ico")
		e.File("/favicon.svg", "./frontend/dist/favicon.svg")
		e.File("/apple-touch-icon.png", "./frontend/dist/apple-touch-icon.png")
		e.File("/favicon-96x96.png", "./frontend/dist/favicon-96x96.png")
		e.File("/web-app-manifest-192x192.png", "./frontend/dist/web-app-manifest-192x192.png")
		e.File("/web-app-manifest-512x512.png", "./frontend/dist/web-app-manifest-512x512.png")
		e.File("/site.webmanifest", "./frontend/dist/site.webmanifest")

		// SPA fallback - serve index.html for all non-API routes
		// This must be last so it doesn't interfere with API routes
		e.GET("/*", func(c echo.Context) error {
			return c.File("./frontend/dist/index.html")
		})

		logger.Info("Serving static files from ./frontend/dist with SPA routing support")
	}

	// Start pprof server in a separate goroutine for profiling
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8081"
		}
		pprofPort := "6060"
		logger.Info("Memory profiling available", "url", "http://localhost:"+pprofPort+"/debug/pprof/")
		if err := http.ListenAndServe(":"+pprofPort, nil); err != nil {
			logger.Error("pprof server failed", "error", err)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Create HTTP server with longer timeouts for large file uploads
	server := &http.Server{
		Addr:              ":" + port,
		ReadTimeout:       30 * time.Minute, // Allow 30 minutes for reading large uploads
		WriteTimeout:      30 * time.Minute, // Allow 30 minutes for writing responses
		ReadHeaderTimeout: 1 * time.Minute,  // Header read timeout
		IdleTimeout:       2 * time.Minute,  // Idle connection timeout
		MaxHeaderBytes:    1 << 20,          // 1 MB max header size
	}

	logger.Info("Server starting", "port", port)
	logger.Info("Upload timeout set to 30 minutes for large backup files")

	e.Server = server
	// Start server
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
