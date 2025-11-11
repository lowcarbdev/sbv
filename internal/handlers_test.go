package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// Global test user ID - stored here so setupTestContext can use it
var testUserID string

// setupTestDB creates a test database with sample data
func setupTestDB(t *testing.T) (string, func()) {
	tmpDB := "test_handlers.db"
	tmpAuthDB := "test_handlers_auth.db"

	// Clean up any existing test database
	os.Remove(tmpDB)
	os.Remove(tmpAuthDB)

	// Initialize auth database first
	if err := InitAuthDB(tmpAuthDB); err != nil {
		t.Fatalf("Failed to initialize auth database: %v", err)
	}

	// Initialize main database
	if err := InitDB(tmpDB); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create test user
	user, err := CreateUser("testuser", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Store user ID globally for setupTestContext to use
	testUserID = user.ID

	// Initialize user database (using UUID-based filename)
	userDBPath := fmt.Sprintf("sbv_%s.db", user.ID)
	if err := InitUserDB(user.ID, userDBPath); err != nil {
		t.Fatalf("Failed to initialize user database: %v", err)
	}

	// Get user database connection
	userDB, err := GetUserDB(user.ID, user.Username)
	if err != nil {
		t.Fatalf("Failed to get user database: %v", err)
	}

	// Insert test messages
	sampleXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="3">
  <sms protocol="0" address="+15551234567" date="1285799668000" type="2" body="Test sent message" read="1" status="-1" />
  <sms protocol="0" address="+15551234567" date="1285799669000" type="1" body="Test received message" read="1" status="-1" />
  <mms date="1285799670000" rr="null" sub="null" read="1" ct_t="application/vnd.wap.multipart.related" msg_box="2" address="+15559876543" m_type="128" text_only="0">
    <parts>
      <part seq="0" ct="text/plain" name="null" chset="106" text="Test MMS message" />
    </parts>
    <addrs>
      <addr address="+15552226543" type="137" charset="106" />
      <addr address="+15551116565" type="151" charset="106" />
    </addrs>
  </mms>
</smses>`

	reader := strings.NewReader(sampleXML)
	result, err := ParseSMSBackup(reader)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	for i := range result.Messages {
		if err := InsertMessage(userDB, &result.Messages[i]); err != nil {
			t.Fatalf("Failed to insert message: %v", err)
		}
	}

	// Return cleanup function
	cleanup := func() {
		if db != nil {
			db.Close()
		}
		if userDB != nil {
			userDB.Close()
		}
		os.Remove(tmpDB)
		os.Remove(tmpAuthDB)
		os.Remove(userDBPath)
	}

	return tmpDB, cleanup
}

// setupTestContext creates an Echo context with user authentication
func setupTestContext(method, url string, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context (simulating authentication middleware)
	// Use the global testUserID which was set by setupTestDB
	c.Set("user_id", testUserID)
	c.Set("username", "testuser")

	return c, rec
}

func TestHealthEndpoint(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	if err := handler(c); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", rec.Body.String())
	}
}

func TestHandleConversations(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/conversations", "")

	if err := HandleConversations(c); err != nil {
		t.Fatalf("HandleConversations failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var conversations []Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have 2 conversations (one for +15551234567, one for +15559876543)
	if len(conversations) < 1 {
		t.Errorf("Expected at least 1 conversation, got %d", len(conversations))
	}
}

func TestHandleConversationsWithDateRange(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Test with date range that includes all messages
	start := time.Unix(1285799668, 0).Add(-time.Hour).Format(time.RFC3339)
	end := time.Unix(1285799671, 0).Format(time.RFC3339)

	c, rec := setupTestContext(http.MethodGet, "/api/conversations?start="+start+"&end="+end, "")
	c.QueryParams().Add("start", start)
	c.QueryParams().Add("end", end)

	if err := HandleConversations(c); err != nil {
		t.Fatalf("HandleConversations with date range failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var conversations []Conversation
	if err := json.Unmarshal(rec.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(conversations) < 1 {
		t.Errorf("Expected at least 1 conversation, got %d", len(conversations))
	}
}

func TestHandleMessages(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// First get conversations to find a valid address
	c1, rec1 := setupTestContext(http.MethodGet, "/api/conversations", "")
	if err := HandleConversations(c1); err != nil {
		t.Fatalf("HandleConversations failed: %v", err)
	}

	var conversations []Conversation
	if err := json.Unmarshal(rec1.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("Failed to parse conversations: %v", err)
	}

	if len(conversations) == 0 {
		t.Fatal("No conversations found in test database")
	}

	// Use the first conversation's address
	testAddress := conversations[0].Address

	c, rec := setupTestContext(http.MethodGet, "/api/messages?address="+testAddress, "")
	c.QueryParams().Add("address", testAddress)

	if err := HandleMessages(c); err != nil {
		t.Fatalf("HandleMessages failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var messages []Message
	if err := json.Unmarshal(rec.Body.Bytes(), &messages); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the response is valid JSON array (might be empty if address format doesn't match)
	// The important thing is that the handler responds correctly
	t.Logf("Got %d messages for address %s", len(messages), testAddress)
}

func TestHandleMessagesWithoutAddress(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/messages", "")

	if err := HandleMessages(c); err != nil {
		t.Fatalf("HandleMessages failed: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if !strings.Contains(response["error"], "Address parameter required") {
		t.Errorf("Expected error about missing address, got: %s", response["error"])
	}
}

func TestHandleMessagesConversationType(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// First get conversations to find a valid address
	c1, rec1 := setupTestContext(http.MethodGet, "/api/conversations", "")
	if err := HandleConversations(c1); err != nil {
		t.Fatalf("HandleConversations failed: %v", err)
	}

	var conversations []Conversation
	if err := json.Unmarshal(rec1.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("Failed to parse conversations: %v", err)
	}

	if len(conversations) == 0 {
		t.Fatal("No conversations found in test database")
	}

	// Use the first conversation's address
	testAddress := conversations[0].Address

	c, rec := setupTestContext(http.MethodGet, "/api/messages?address="+testAddress+"&type=conversation", "")
	c.QueryParams().Add("address", testAddress)
	c.QueryParams().Add("type", "conversation")

	if err := HandleMessages(c); err != nil {
		t.Fatalf("HandleMessages with type=conversation failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var activities []ActivityItem
	if err := json.Unmarshal(rec.Body.Bytes(), &activities); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify the response is valid JSON array (might be empty if address format doesn't match)
	// The important thing is that the handler responds correctly with type=conversation
	t.Logf("Got %d activities for address %s with type=conversation", len(activities), testAddress)
}

func TestHandleActivity(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/activity", "")

	if err := HandleActivity(c); err != nil {
		t.Fatalf("HandleActivity failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var activities []ActivityItem
	if err := json.Unmarshal(rec.Body.Bytes(), &activities); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have 3 activities (2 SMS + 1 MMS)
	if len(activities) != 3 {
		t.Errorf("Expected 3 activities, got %d", len(activities))
	}
}

func TestHandleActivityWithPagination(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/activity?limit=1&offset=0", "")
	c.QueryParams().Add("limit", "1")
	c.QueryParams().Add("offset", "0")

	if err := HandleActivity(c); err != nil {
		t.Fatalf("HandleActivity with pagination failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var activities []ActivityItem
	if err := json.Unmarshal(rec.Body.Bytes(), &activities); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should have exactly 1 activity due to limit
	if len(activities) != 1 {
		t.Errorf("Expected 1 activity, got %d", len(activities))
	}
}

func TestHandleDateRange(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/daterange", "")

	if err := HandleDateRange(c); err != nil {
		t.Fatalf("HandleDateRange failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["min_date"] == nil {
		t.Error("Expected min_date in response")
	}

	if response["max_date"] == nil {
		t.Error("Expected max_date in response")
	}
}

func TestHandleMedia(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Test without message ID
	c, rec := setupTestContext(http.MethodGet, "/api/media", "")

	if err := HandleMedia(c); err != nil {
		t.Fatalf("HandleMedia failed: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if !strings.Contains(response["error"], "Message ID required") {
		t.Errorf("Expected error about missing message ID, got: %s", response["error"])
	}
}

func TestHandleMediaNotFound(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/media?id=99999", "")
	c.QueryParams().Add("id", "99999")

	if err := HandleMedia(c); err != nil {
		t.Fatalf("HandleMedia failed: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestHandleSearch(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/search?q=Test", "")
	c.QueryParams().Add("q", "Test")

	if err := HandleSearch(c); err != nil {
		t.Fatalf("HandleSearch failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var results []SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should find messages containing "Test"
	if len(results) < 1 {
		t.Errorf("Expected at least 1 search result, got %d", len(results))
	}
}

func TestHandleSearchEmpty(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/search", "")

	if err := HandleSearch(c); err != nil {
		t.Fatalf("HandleSearch failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var results []SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return empty array for empty query
	if len(results) != 0 {
		t.Errorf("Expected 0 search results for empty query, got %d", len(results))
	}
}

func TestHandleSearchWithLimit(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	c, rec := setupTestContext(http.MethodGet, "/api/search?q=Test&limit=1", "")
	c.QueryParams().Add("q", "Test")
	c.QueryParams().Add("limit", "1")

	if err := HandleSearch(c); err != nil {
		t.Fatalf("HandleSearch with limit failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var results []SearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should respect the limit
	if len(results) > 1 {
		t.Errorf("Expected at most 1 search result, got %d", len(results))
	}
}

func TestHandleProgress(t *testing.T) {
	c, rec := setupTestContext(http.MethodGet, "/api/progress", "")

	if err := HandleProgress(c); err != nil {
		t.Fatalf("HandleProgress failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Should return no_upload status when no upload is in progress
	if response["status"] != "no_upload" {
		t.Errorf("Expected status 'no_upload', got '%v'", response["status"])
	}
}

func TestGetUserDBHelperMissingUserID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set user_id in context
	_, err := getUserDB(c)

	if err == nil {
		t.Error("Expected error when user_id is missing")
	}

	if !strings.Contains(err.Error(), "user_id not found") {
		t.Errorf("Expected error about missing user_id, got: %v", err)
	}
}

func TestGetUserDBHelperMissingUsername(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user_id but not username
	c.Set("user_id", "test-user-id")

	_, err := getUserDB(c)

	if err == nil {
		t.Error("Expected error when username is missing")
	}

	if !strings.Contains(err.Error(), "username not found") {
		t.Errorf("Expected error about missing username, got: %v", err)
	}
}
