package internal


import (
	"os"
	"strings"
	"testing"
	"time"
)

const sampleXML = `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<?xml-stylesheet type="text/xsl" href="sms.xsl"?>
<smses count="2">
  <sms protocol="0" address="332" date="1285799668193" type="2" subject="null" body="Sample Message Sent from the phone" toa="null" sc_toa="null" service_center="null" read="1" status="-1" locked="0" readable_date="Sep 30, 2010 8:34:28 AM" contact_name="(Unknown)" />
  <sms protocol="0" address="4433221123" date="1289643415810" type="1" subject="null" body="Sample Message received by the phone" toa="null" sc_toa="null" service_center="null" read="0" status="-1" locked="0" readable_date="Nov 13, 2010 9:16:55 PM" contact_name="(Unknown)" />
</smses>`

func TestSampleXMLParsing(t *testing.T) {
	// Parse the XML
	reader := strings.NewReader(sampleXML)
	result, err := ParseSMSBackup(reader)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Verify we got 2 messages
	if len(result.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(result.Messages))
	}

	// Verify first message (sent)
	msg1 := result.Messages[0]
	if msg1.Address != "332" {
		t.Errorf("Expected address '332', got '%s'", msg1.Address)
	}
	if msg1.Type != 2 {
		t.Errorf("Expected type 2 (sent), got %d", msg1.Type)
	}
	if msg1.Body != "Sample Message Sent from the phone" {
		t.Errorf("Expected body 'Sample Message Sent from the phone', got '%s'", msg1.Body)
	}
	if msg1.Protocol != 0 {
		t.Errorf("Expected protocol 0, got %d", msg1.Protocol)
	}
	if !msg1.Read {
		t.Errorf("Expected message to be read (read=1)")
	}
	if msg1.Status != -1 {
		t.Errorf("Expected status -1, got %d", msg1.Status)
	}
	// Check date: 1285799668193 milliseconds = Sep 30, 2010 8:34:28 AM
	expectedDate1 := time.Unix(1285799668, 0)
	if !msg1.Date.Equal(expectedDate1) {
		t.Errorf("Expected date %v, got %v", expectedDate1, msg1.Date)
	}

	// Verify second message (received)
	msg2 := result.Messages[1]
	// Phone number normalization adds +1 to 10-digit US numbers
	if msg2.Address != "+14433221123" {
		t.Errorf("Expected address '+14433221123', got '%s'", msg2.Address)
	}
	if msg2.Type != 1 {
		t.Errorf("Expected type 1 (received), got %d", msg2.Type)
	}
	if msg2.Body != "Sample Message received by the phone" {
		t.Errorf("Expected body 'Sample Message received by the phone', got '%s'", msg2.Body)
	}
	if msg2.Read {
		t.Errorf("Expected message to be unread (read=0)")
	}
	// Check date: 1289643415810 milliseconds = Nov 13, 2010 9:16:55 PM
	expectedDate2 := time.Unix(1289643415, 0)
	if !msg2.Date.Equal(expectedDate2) {
		t.Errorf("Expected date %v, got %v", expectedDate2, msg2.Date)
	}

	// Verify no call logs in this sample
	if len(result.Calls) != 0 {
		t.Errorf("Expected 0 call logs, got %d", len(result.Calls))
	}
}

func TestSampleXMLDatabaseIngestion(t *testing.T) {
	// Create a temporary database file
	tmpDB := "test_messages.db"
	defer os.Remove(tmpDB) // Clean up after test

	// Initialize database
	err := InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Parse the XML
	reader := strings.NewReader(sampleXML)
	result, err := ParseSMSBackup(reader)
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Insert messages into database
	messageCount := 0
	for i := range result.Messages {
		err := InsertMessage(db, &result.Messages[i])
		if err != nil {
			t.Errorf("Failed to insert message %d: %v", i, err)
			continue
		}
		messageCount++

		// Verify the ID was set
		if result.Messages[i].ID == 0 {
			t.Errorf("Message %d: ID was not set after insert", i)
		}
	}

	// Verify we inserted 2 messages
	if messageCount != 2 {
		t.Errorf("Expected to insert 2 messages, inserted %d", messageCount)
	}

	// Retrieve messages from database and verify
	messages, err := GetMessages(db, "332", nil, nil)
	if err != nil {
		t.Fatalf("Failed to retrieve messages for address '332': %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 message for address '332', got %d", len(messages))
	} else {
		msg := messages[0]
		if msg.Body != "Sample Message Sent from the phone" {
			t.Errorf("Retrieved message has wrong body: '%s'", msg.Body)
		}
		if msg.Type != 2 {
			t.Errorf("Retrieved message has wrong type: %d", msg.Type)
		}
		if msg.Protocol != 0 {
			t.Errorf("Retrieved message has wrong protocol: %d", msg.Protocol)
		}
		if msg.Status != -1 {
			t.Errorf("Retrieved message has wrong status: %d", msg.Status)
		}
		if !msg.Read {
			t.Errorf("Retrieved message should be marked as read")
		}
	}

	// Retrieve second message
	messages2, err := GetMessages(db, "+14433221123", nil, nil)
	if err != nil {
		t.Fatalf("Failed to retrieve messages for address '+14433221123': %v", err)
	}
	if len(messages2) != 1 {
		t.Errorf("Expected 1 message for address '+14433221123', got %d", len(messages2))
	} else {
		msg := messages2[0]
		if msg.Body != "Sample Message received by the phone" {
			t.Errorf("Retrieved message has wrong body: '%s'", msg.Body)
		}
		if msg.Type != 1 {
			t.Errorf("Retrieved message has wrong type: %d", msg.Type)
		}
		if msg.Read {
			t.Errorf("Retrieved message should be marked as unread")
		}
	}

	// Test GetConversations
	conversations, err := GetConversations(db, nil, nil)
	if err != nil {
		t.Fatalf("Failed to get conversations: %v", err)
	}
	if len(conversations) != 2 {
		t.Errorf("Expected 2 conversations, got %d", len(conversations))
	}

	// Verify conversations are sorted by date (most recent first)
	// Second message (1289643415) should be first as it's more recent
	if len(conversations) == 2 {
		if conversations[0].Address != "+14433221123" {
			t.Errorf("Expected first conversation to be '+14433221123', got '%s'", conversations[0].Address)
		}
		if conversations[1].Address != "332" {
			t.Errorf("Expected second conversation to be '332', got '%s'", conversations[1].Address)
		}
		if conversations[0].MessageCount != 1 {
			t.Errorf("Expected first conversation to have 1 message, got %d", conversations[0].MessageCount)
		}
		if conversations[0].Type != "conversation" {
			t.Errorf("Expected conversation type to be 'conversation', got '%s'", conversations[0].Type)
		}
	}

	// Test date range functionality
	startDate := time.Unix(1289000000, 0) // After first message, before second
	messages3, err := GetMessages(db, "332", &startDate, nil)
	if err != nil {
		t.Fatalf("Failed to retrieve messages with date filter: %v", err)
	}
	if len(messages3) != 0 {
		t.Errorf("Expected 0 messages after start date, got %d", len(messages3))
	}

	// Get date range
	minDate, maxDate, err := GetDateRange(db)
	if err != nil {
		t.Fatalf("Failed to get date range: %v", err)
	}
	expectedMin := time.Unix(1285799668, 0)
	expectedMax := time.Unix(1289643415, 0)
	if !minDate.Equal(expectedMin) {
		t.Errorf("Expected min date %v, got %v", expectedMin, minDate)
	}
	if !maxDate.Equal(expectedMax) {
		t.Errorf("Expected max date %v, got %v", expectedMax, maxDate)
	}
}

func TestEmptyXML(t *testing.T) {
	emptyXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="0">
</smses>`

	reader := strings.NewReader(emptyXML)
	result, err := ParseSMSBackup(reader)
	if err != nil {
		t.Fatalf("Failed to parse empty XML: %v", err)
	}

	if len(result.Messages) != 0 {
		t.Errorf("Expected 0 messages, got %d", len(result.Messages))
	}
	if len(result.Calls) != 0 {
		t.Errorf("Expected 0 calls, got %d", len(result.Calls))
	}
}

func TestInvalidXML(t *testing.T) {
	invalidXML := `<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>
<smses count="1">
  <sms protocol="invalid" address="123" date="notanumber" type="2" body="Test" />
</smses>`

	reader := strings.NewReader(invalidXML)
	result, err := ParseSMSBackup(reader)

	// Should parse but skip invalid entries or use defaults
	if err != nil {
		t.Fatalf("Parser should handle invalid data gracefully: %v", err)
	}

	// The message might be parsed with default values for invalid fields
	if len(result.Messages) > 0 {
		msg := result.Messages[0]
		// Protocol "invalid" should parse as 0
		if msg.Protocol != 0 {
			t.Logf("Invalid protocol parsed as: %d", msg.Protocol)
		}
		// Date "notanumber" should result in Unix epoch
		t.Logf("Invalid date parsed as: %v", msg.Date)
	}
}
