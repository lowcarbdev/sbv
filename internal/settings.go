package internal

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Settings represents user settings stored as JSON
type Settings struct {
	Conversations ConversationSettings `json:"conversations"`
}

// ConversationSettings contains settings for the conversation view
type ConversationSettings struct {
	ShowCalls bool `json:"show_calls"`
}

// GetDefaultSettings returns the default settings
func GetDefaultSettings() Settings {
	return Settings{
		Conversations: ConversationSettings{
			ShowCalls: true,
		},
	}
}

// GetUserSettings retrieves settings for a user
func GetUserSettings(userID string) (Settings, error) {
	var settingsJSON string
	var updatedAt int64

	err := authDB.QueryRow(
		"SELECT settings_json, updated_at FROM settings WHERE user_id = ?",
		userID,
	).Scan(&settingsJSON, &updatedAt)

	if err == sql.ErrNoRows {
		// Return default settings if no settings exist
		return GetDefaultSettings(), nil
	}

	if err != nil {
		return Settings{}, err
	}

	var settings Settings
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return Settings{}, err
	}

	return settings, nil
}

// SaveUserSettings saves settings for a user
func SaveUserSettings(userID string, settings Settings) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	_, err = authDB.Exec(`
		INSERT INTO settings (user_id, settings_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			settings_json = excluded.settings_json,
			updated_at = excluded.updated_at
	`, userID, string(settingsJSON), now)

	return err
}

// HandleGetSettings handles GET /api/settings
func HandleGetSettings(c echo.Context) error {
	userID := c.Get("user_id").(string)

	settings, err := GetUserSettings(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get settings",
		})
	}

	return c.JSON(http.StatusOK, settings)
}

// HandleUpdateSettings handles PUT /api/settings
func HandleUpdateSettings(c echo.Context) error {
	userID := c.Get("user_id").(string)

	var settings Settings
	if err := c.Bind(&settings); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid settings data",
		})
	}

	if err := SaveUserSettings(userID, settings); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save settings",
		})
	}

	return c.JSON(http.StatusOK, settings)
}
