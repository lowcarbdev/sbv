package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/labstack/echo/v4"
)

// fakeOIDCProvider is a minimal OIDC identity provider for tests: it serves
// a discovery document, a JWKS endpoint, and a token endpoint that returns
// an RSA-signed ID token for the configured subject.
type fakeOIDCProvider struct {
	server   *httptest.Server
	key      *rsa.PrivateKey
	username string
	nonce    string
}

func newFakeOIDCProvider(t *testing.T) *fakeOIDCProvider {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	p := &fakeOIDCProvider{key: key}

	mux := http.NewServeMux()
	p.server = httptest.NewServer(mux)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                                p.server.URL,
			"authorization_endpoint":                p.server.URL + "/authorize",
			"token_endpoint":                        p.server.URL + "/token",
			"jwks_uri":                              p.server.URL + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test-key", Algorithm: "RS256", Use: "sig"}},
		})
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		idToken := p.signIDToken(t, map[string]interface{}{
			"iss":                p.server.URL,
			"aud":                "test-client",
			"sub":                "subject-123",
			"preferred_username": p.username,
			"nonce":              p.nonce,
			"exp":                time.Now().Add(time.Hour).Unix(),
			"iat":                time.Now().Unix(),
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"id_token":     idToken,
		})
	})

	t.Cleanup(p.server.Close)
	return p
}

func (p *fakeOIDCProvider) signIDToken(t *testing.T, claims map[string]interface{}) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("Failed to marshal claims: %v", err)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: p.key},
		(&jose.SignerOptions{}).WithHeader("kid", "test-key"),
	)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}
	jws, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}
	token, err := jws.CompactSerialize()
	if err != nil {
		t.Fatalf("Failed to serialize token: %v", err)
	}
	return token
}

// setupOIDCTest configures the environment and a fresh auth database
func setupOIDCTest(t *testing.T, provider *fakeOIDCProvider) {
	t.Helper()

	tmpAuthDB := "test_oidc_auth.db"
	os.Remove(tmpAuthDB)
	if err := InitAuthDB(tmpAuthDB); err != nil {
		t.Fatalf("Failed to initialize auth database: %v", err)
	}

	t.Setenv("OIDC_ISSUER_URL", provider.server.URL)
	t.Setenv("OIDC_CLIENT_ID", "test-client")
	t.Setenv("OIDC_CLIENT_SECRET", "test-secret")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:8085/api/auth/oidc/callback")

	// Reset the cached provider so each test rediscovers against its own server
	oidcProvider = nil
	oidcProviderErr = nil
	oidcProviderOnce = sync.Once{}

	t.Cleanup(func() {
		os.Remove(tmpAuthDB)
		oidcProvider = nil
		oidcProviderErr = nil
		oidcProviderOnce = sync.Once{}
	})
}

func TestOIDCLoginRedirect(t *testing.T) {
	provider := newFakeOIDCProvider(t)
	setupOIDCTest(t, provider)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := HandleOIDCLogin(c); err != nil {
		t.Fatalf("HandleOIDCLogin failed: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected 302 redirect, got %d", rec.Code)
	}

	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Invalid redirect location: %v", err)
	}
	if !strings.HasPrefix(location.String(), provider.server.URL+"/authorize") {
		t.Errorf("Expected redirect to provider authorize endpoint, got %s", location)
	}
	if location.Query().Get("client_id") != "test-client" {
		t.Errorf("Expected client_id=test-client, got %s", location.Query().Get("client_id"))
	}
	if location.Query().Get("state") == "" || location.Query().Get("nonce") == "" {
		t.Error("Expected state and nonce parameters in authorize URL")
	}

	cookies := rec.Result().Cookies()
	var hasState, hasNonce bool
	for _, cookie := range cookies {
		if cookie.Name == "oidc_state" && cookie.Value == location.Query().Get("state") {
			hasState = true
		}
		if cookie.Name == "oidc_nonce" && cookie.Value == location.Query().Get("nonce") {
			hasNonce = true
		}
	}
	if !hasState || !hasNonce {
		t.Error("Expected oidc_state and oidc_nonce cookies matching the authorize URL")
	}
}

// callbackContext builds a callback request carrying the state/nonce cookies
func callbackContext(state, nonce string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/auth/oidc/callback?code=test-code&state=%s", state), nil)
	req.AddCookie(&http.Cookie{Name: "oidc_state", Value: state})
	req.AddCookie(&http.Cookie{Name: "oidc_nonce", Value: nonce})
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestOIDCCallbackProvisionsUser(t *testing.T) {
	provider := newFakeOIDCProvider(t)
	setupOIDCTest(t, provider)
	provider.username = "oidcuser"
	provider.nonce = "test-nonce"

	c, rec := callbackContext("test-state", "test-nonce")

	if err := HandleOIDCCallback(c); err != nil {
		t.Fatalf("HandleOIDCCallback failed: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("Expected 302 redirect, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Location") != "/" {
		t.Errorf("Expected redirect to /, got %s", rec.Header().Get("Location"))
	}

	// User should have been provisioned without a usable password
	user, err := GetUserByUsername("oidcuser")
	if err != nil {
		t.Fatalf("Expected user to be provisioned: %v", err)
	}
	if VerifyPassword(user, oidcPasswordMarker) {
		t.Error("Password login must not succeed for OIDC-provisioned accounts")
	}

	// A session cookie should have been set and be valid
	var sessionID string
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "session_id" && cookie.Value != "" {
			sessionID = cookie.Value
		}
	}
	if sessionID == "" {
		t.Fatal("Expected session_id cookie")
	}
	session, err := GetSession(sessionID)
	if err != nil {
		t.Fatalf("Expected valid session: %v", err)
	}
	if session.Username != "oidcuser" {
		t.Errorf("Expected session for oidcuser, got %s", session.Username)
	}

	// Clean up the user database file created during provisioning
	os.Remove(fmt.Sprintf("sbv_%s.db", user.ID))
}

func TestOIDCCallbackRejectsBadState(t *testing.T) {
	provider := newFakeOIDCProvider(t)
	setupOIDCTest(t, provider)

	// URL carries a different state than the oidc_state cookie
	c, rec := callbackContext("attacker-state", "test-nonce")
	c.Request().URL.RawQuery = "code=test-code&state=different-state"

	if err := HandleOIDCCallback(c); err != nil {
		t.Fatalf("HandleOIDCCallback returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for state mismatch, got %d", rec.Code)
	}
}

func TestOIDCCallbackRejectsBadNonce(t *testing.T) {
	provider := newFakeOIDCProvider(t)
	setupOIDCTest(t, provider)
	provider.username = "oidcuser2"
	provider.nonce = "provider-nonce" // ID token will carry this nonce

	// Browser presents a different nonce cookie than the token contains
	c, rec := callbackContext("test-state", "other-nonce")

	if err := HandleOIDCCallback(c); err != nil {
		t.Fatalf("HandleOIDCCallback returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for nonce mismatch, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := GetUserByUsername("oidcuser2"); err == nil {
		t.Error("User must not be provisioned when nonce verification fails")
	}
}

func TestOIDCCallbackHonorsDisabledRegistration(t *testing.T) {
	provider := newFakeOIDCProvider(t)
	setupOIDCTest(t, provider)
	provider.username = "unknownuser"
	provider.nonce = "test-nonce"
	t.Setenv("DISABLE_REGISTRATION", "true")

	c, rec := callbackContext("test-state", "test-nonce")

	if err := HandleOIDCCallback(c); err != nil {
		t.Fatalf("HandleOIDCCallback returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 when registration is disabled, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := GetUserByUsername("unknownuser"); err == nil {
		t.Error("User must not be provisioned when registration is disabled")
	}
}
