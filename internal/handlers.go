package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// getUserDB is a helper function to get the user's database connection from the context
func getUserDB(c echo.Context) (*sql.DB, error) {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return nil, fmt.Errorf("user_id not found in context")
	}
	username, ok := c.Get("username").(string)
	if !ok {
		return nil, fmt.Errorf("username not found in context")
	}
	return GetUserDB(userID, username)
}

func HandleUpload(c echo.Context) error {
	// Use a smaller memory limit for the form parsing itself (32 MB)
	// Large files will be streamed directly to disk
	err := c.Request().ParseMultipartForm(32 << 20) // 32 MB max in memory
	if err != nil {
		slog.Error("Error parsing form", "error", err)
		return c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Error:   "Failed to parse form data. File may be too large or corrupted.",
		})
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		slog.Error("Error getting file", "error", err)
		return c.JSON(http.StatusBadRequest, UploadResponse{
			Success: false,
			Error:   "Failed to get file from form",
		})
	}
	defer file.Close()

	slog.Info("Receiving file", "filename", header.Filename, "size", header.Size)

	// Save uploaded file to temporary location first
	tempFilePath, err := SaveUploadedFile(file, header.Filename)
	if err != nil {
		slog.Error("Error saving file", "error", err)
		return c.JSON(http.StatusInternalServerError, UploadResponse{
			Success: false,
			Error:   "Failed to save uploaded file: " + err.Error(),
		})
	}

	slog.Info("File saved", "path", tempFilePath)

	// Get user ID from context
	userID, ok := c.Get("user_id").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, UploadResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	// Get username from context
	username, ok := c.Get("username").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, UploadResponse{
			Success: false,
			Error:   "User not authenticated",
		})
	}

	// Start background processing with user context
	go ProcessUploadedFile(userID, username, tempFilePath)

	// Return immediately - client will poll /api/progress for status
	return c.JSON(http.StatusOK, UploadResponse{
		Success:      true,
		MessageCount: 0,
		CallLogCount: 0,
		Processing:   true,
	})
}

func HandleConversations(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	var startDate, endDate *time.Time

	if startStr := c.QueryParam("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err == nil {
			startDate = &t
		}
	}

	if endStr := c.QueryParam("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err == nil {
			endDate = &t
		}
	}

	conversations, err := GetConversations(userDB, startDate, endDate)
	if err != nil {
		slog.Error("Error getting conversations", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get conversations",
		})
	}

	return c.JSON(http.StatusOK, conversations)
}

func HandleMessages(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	address := c.QueryParam("address")
	convType := c.QueryParam("type")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Address parameter required",
		})
	}

	var startDate, endDate *time.Time

	if startStr := c.QueryParam("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err == nil {
			startDate = &t
		}
	}

	if endStr := c.QueryParam("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err == nil {
			endDate = &t
		}
	}

	// If type is "call", return call logs instead of messages
	if convType == "call" {
		calls, err := GetCallLogs(userDB, address, startDate, endDate)
		if err != nil {
			slog.Error("Error getting call logs", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get call logs",
			})
		}
		return c.JSON(http.StatusOK, calls)
	}

	// If type is "conversation", return combined messages and calls
	if convType == "conversation" {
		// Parse limit and offset parameters
		limit := 100000 // Default to 100k (effectively unlimited for most users)
		offset := 0

		if limitStr := c.QueryParam("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		if offsetStr := c.QueryParam("offset"); offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}

		// Get user ID from context to fetch settings
		userID, ok := c.Get("user_id").(string)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "User not authenticated",
			})
		}

		// Fetch user settings to check if calls should be shown
		settings, err := GetUserSettings(userID)
		if err != nil {
			slog.Error("Error getting user settings", "error", err)
			// If we can't get settings, default to showing calls
			settings = GetDefaultSettings()
		}

		activities, err := GetActivityByAddress(userDB, address, startDate, endDate, limit, offset)
		if err != nil {
			slog.Error("Error getting activity", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get activity",
			})
		}

		// Filter out calls if show_calls setting is false
		if !settings.Conversations.ShowCalls {
			filteredActivities := []ActivityItem{}
			for _, activity := range activities {
				if activity.Type != "call" {
					filteredActivities = append(filteredActivities, activity)
				}
			}
			activities = filteredActivities
		}

		return c.JSON(http.StatusOK, activities)
	}

	messages, err := GetMessages(userDB, address, startDate, endDate)
	if err != nil {
		slog.Error("Error getting messages", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get messages",
		})
	}

	return c.JSON(http.StatusOK, messages)
}

func HandleActivity(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	var startDate, endDate *time.Time

	if startStr := c.QueryParam("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err == nil {
			startDate = &t
		}
	}

	if endStr := c.QueryParam("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err == nil {
			endDate = &t
		}
	}

	// Parse pagination parameters
	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
	}

	activities, err := GetActivity(userDB, startDate, endDate, limit, offset)
	if err != nil {
		slog.Error("Error getting activity", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get activity",
		})
	}

	return c.JSON(http.StatusOK, activities)
}

func HandleCalls(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	var startDate, endDate *time.Time

	if startStr := c.QueryParam("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err == nil {
			startDate = &t
		}
	}

	if endStr := c.QueryParam("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err == nil {
			endDate = &t
		}
	}

	// Parse pagination parameters
	limit := 50 // default limit
	offset := 0 // default offset

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
	}

	calls, err := GetAllCalls(userDB, startDate, endDate, limit, offset)
	if err != nil {
		slog.Error("Error getting calls", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get calls",
		})
	}

	return c.JSON(http.StatusOK, calls)
}

func HandleDateRange(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	minDate, maxDate, err := GetDateRange(userDB)
	if err != nil {
		slog.Error("Error getting date range", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get date range",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"min_date": minDate,
		"max_date": maxDate,
	})
}

func HandleProgress(c echo.Context) error {
	progress := GetUploadProgress()
	if progress == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "no_upload",
		})
	}

	return c.JSON(http.StatusOK, progress)
}

func HandleMedia(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	// Get message ID from query parameter
	messageID := c.QueryParam("id")
	if messageID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Message ID required",
		})
	}

	// Fetch media from database
	media, contentType, err := GetMessageMedia(userDB, messageID)
	if err != nil {
		slog.Error("Error getting media", "error", err)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Media not found",
		})
	}

	if len(media) == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "No media for this message",
		})
	}

	// Set appropriate headers
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", len(media)))

	// Write binary data with proper content type
	return c.Blob(http.StatusOK, contentType, media)
}

func HandleSearch(c echo.Context) error {
	userDB, err := getUserDB(c)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user database",
		})
	}

	// Get search query from query parameter
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusOK, []SearchResult{})
	}

	// Get limit from query parameter, default to 100
	limit := 100
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// Perform search
	results, err := SearchMessages(userDB, query, limit)
	if err != nil {
		slog.Error("Error searching messages", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Search failed: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, results)
}

// HandleVersion returns the application version
func HandleVersion(c echo.Context) error {
	// Try to read version from version.json file first (Docker builds)
	versionFile := "/app/version.json"
	if data, err := os.ReadFile(versionFile); err == nil {
		var versionData map[string]string
		if err := json.Unmarshal(data, &versionData); err == nil {
			return c.JSON(http.StatusOK, versionData)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"version": "dev",
	})
}
