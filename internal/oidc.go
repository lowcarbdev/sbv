package internal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

// OIDC single sign-on against any spec-compliant provider (Authentik,
// tinyauth, Pocket ID, Keycloak, ...). Configured entirely via environment
// variables; disabled unless OIDC_ISSUER_URL is set.
//
//	OIDC_ISSUER_URL      issuer URL used for discovery (required)
//	OIDC_CLIENT_ID       client ID registered with the provider (required)
//	OIDC_CLIENT_SECRET   client secret (required)
//	OIDC_REDIRECT_URL    callback URL; derived from the request when unset
//	OIDC_PROVIDER_NAME   label for the login button (default "SSO")
//	OIDC_SCOPES          space-separated scopes (default "openid profile email")
//	OIDC_USERNAME_CLAIM  claim used as the SBV username (default
//	                     preferred_username, falling back to email, then sub)

// OIDCEnabled reports whether OIDC login is configured
func OIDCEnabled() bool {
	return os.Getenv("OIDC_ISSUER_URL") != ""
}

// OIDCProviderName returns the display name for the login button
func OIDCProviderName() string {
	if name := os.Getenv("OIDC_PROVIDER_NAME"); name != "" {
		return name
	}
	return "SSO"
}

var (
	oidcProvider     *oidc.Provider
	oidcProviderErr  error
	oidcProviderOnce sync.Once
)

// getOIDCProvider performs issuer discovery once, lazily, so SBV still
// starts when the identity provider is temporarily unreachable
func getOIDCProvider() (*oidc.Provider, error) {
	oidcProviderOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		oidcProvider, oidcProviderErr = oidc.NewProvider(ctx, os.Getenv("OIDC_ISSUER_URL"))
		if oidcProviderErr != nil {
			// Allow a retry on the next request instead of caching the failure
			oidcProviderOnce = sync.Once{}
		}
	})
	return oidcProvider, oidcProviderErr
}

// oidcConfig builds the oauth2 config for this request. The redirect URL is
// derived from the request when OIDC_REDIRECT_URL is unset, honoring
// X-Forwarded-Proto/Host from a reverse proxy.
func oidcConfig(c echo.Context, provider *oidc.Provider) *oauth2.Config {
	redirectURL := os.Getenv("OIDC_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = fmt.Sprintf("%s://%s/api/auth/oidc/callback", c.Scheme(), c.Request().Host)
	}

	scopes := strings.Fields(os.Getenv("OIDC_SCOPES"))
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	return &oauth2.Config{
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}
}

// setOIDCFlowCookie stores a short-lived, HttpOnly value (state/nonce)
// during the redirect round-trip to the identity provider
func setOIDCFlowCookie(c echo.Context, name, value string) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   os.Getenv("SECURE_COOKIES") == "true" || c.Scheme() == "https",
		SameSite: http.SameSiteLaxMode,
		Path:     "/api/auth/oidc/",
	})
}

func clearOIDCFlowCookie(c echo.Context, name string) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/api/auth/oidc/",
	})
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HandleOIDCLogin starts the authorization code flow
func HandleOIDCLogin(c echo.Context) error {
	provider, err := getOIDCProvider()
	if err != nil {
		slog.Error("OIDC discovery failed", "error", err)
		return c.String(http.StatusBadGateway, "Identity provider is unreachable")
	}

	state, err := randomToken()
	if err != nil {
		return echo.ErrInternalServerError
	}
	nonce, err := randomToken()
	if err != nil {
		return echo.ErrInternalServerError
	}

	setOIDCFlowCookie(c, "oidc_state", state)
	setOIDCFlowCookie(c, "oidc_nonce", nonce)

	return c.Redirect(http.StatusFound, oidcConfig(c, provider).AuthCodeURL(state, oidc.Nonce(nonce)))
}

// HandleOIDCCallback completes the flow: verifies state and the ID token,
// finds or provisions the user, and creates a normal SBV session
func HandleOIDCCallback(c echo.Context) error {
	provider, err := getOIDCProvider()
	if err != nil {
		slog.Error("OIDC discovery failed", "error", err)
		return c.String(http.StatusBadGateway, "Identity provider is unreachable")
	}

	if errParam := c.QueryParam("error"); errParam != "" {
		slog.Warn("OIDC provider returned error", "error", errParam, "description", c.QueryParam("error_description"))
		return c.String(http.StatusUnauthorized, "Identity provider rejected the login: "+errParam)
	}

	stateCookie, err := c.Cookie("oidc_state")
	if err != nil || stateCookie.Value == "" || c.QueryParam("state") != stateCookie.Value {
		return c.String(http.StatusBadRequest, "Invalid state parameter")
	}
	nonceCookie, err := c.Cookie("oidc_nonce")
	if err != nil || nonceCookie.Value == "" {
		return c.String(http.StatusBadRequest, "Missing nonce cookie")
	}
	clearOIDCFlowCookie(c, "oidc_state")
	clearOIDCFlowCookie(c, "oidc_nonce")

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	token, err := oidcConfig(c, provider).Exchange(ctx, c.QueryParam("code"))
	if err != nil {
		slog.Error("OIDC code exchange failed", "error", err)
		return c.String(http.StatusUnauthorized, "Failed to exchange authorization code")
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return c.String(http.StatusUnauthorized, "Identity provider did not return an ID token")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: os.Getenv("OIDC_CLIENT_ID")}).Verify(ctx, rawIDToken)
	if err != nil {
		slog.Error("OIDC ID token verification failed", "error", err)
		return c.String(http.StatusUnauthorized, "Invalid ID token")
	}
	if idToken.Nonce != nonceCookie.Value {
		return c.String(http.StatusUnauthorized, "Invalid nonce")
	}

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		slog.Error("Failed to parse ID token claims", "error", err)
		return c.String(http.StatusUnauthorized, "Failed to parse ID token claims")
	}

	username := oidcUsername(claims)
	if username == "" {
		return c.String(http.StatusUnauthorized, "ID token contains no usable username claim")
	}

	user, err := findOrCreateOIDCUser(username)
	if err != nil {
		slog.Error("OIDC user provisioning failed", "username", username, "error", err)
		return c.String(http.StatusForbidden, err.Error())
	}

	session, err := CreateSession(user.ID, user.Username)
	if err != nil {
		slog.Error("Error creating session", "error", err)
		return echo.ErrInternalServerError
	}
	setSessionCookie(c, session.ID, session.ExpiresAt)

	slog.Info("OIDC login", "username", username)
	return c.Redirect(http.StatusFound, "/")
}

// oidcUsername picks the SBV username from the ID token claims
func oidcUsername(claims map[string]interface{}) string {
	claimNames := []string{"preferred_username", "email", "sub"}
	if custom := os.Getenv("OIDC_USERNAME_CLAIM"); custom != "" {
		claimNames = []string{custom}
	}
	for _, name := range claimNames {
		if value, ok := claims[name].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// findOrCreateOIDCUser looks up the user by username, provisioning a new
// account on first login unless registration is disabled. OIDC-provisioned
// accounts get a marker instead of a bcrypt hash so password login can
// never succeed for them.
func findOrCreateOIDCUser(username string) (*User, error) {
	user, err := GetUserByUsername(username)
	if err == nil {
		return user, nil
	}

	if !RegistrationEnabled() {
		return nil, fmt.Errorf("no account for '%s' and registration is disabled", username)
	}

	user, err = createOIDCUser(username)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	dbPathPrefix := os.Getenv("DB_PATH_PREFIX")
	if dbPathPrefix == "" {
		dbPathPrefix = "."
	}
	userDBPath := fmt.Sprintf("%s/sbv_%s.db", dbPathPrefix, user.ID)
	if err := InitUserDB(user.ID, userDBPath); err != nil {
		return nil, fmt.Errorf("failed to initialize user database: %w", err)
	}

	slog.Info("Provisioned new user via OIDC", "username", username)
	return user, nil
}

// oidcPasswordMarker is stored in place of a bcrypt hash for accounts
// provisioned via OIDC; bcrypt comparison against it always fails
const oidcPasswordMarker = "*oidc*"

func createOIDCUser(username string) (*User, error) {
	userID := uuid.New().String()
	createdAt := time.Now().Unix()

	_, err := authDB.Exec(
		"INSERT INTO users (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)",
		userID, username, oidcPasswordMarker, createdAt,
	)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           userID,
		Username:     username,
		PasswordHash: oidcPasswordMarker,
		CreatedAt:    time.Unix(createdAt, 0),
	}, nil
}
