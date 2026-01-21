package internal

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// userDBs stores per-user database connections (keyed by user ID)
var userDBs = make(map[string]*sql.DB)
var userDBsMutex sync.RWMutex

// UseWALMode controls whether WAL journal mode is enabled for databases
var UseWALMode bool

// truncateString truncates a string to maxLen characters for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SanitizeUsername converts a username to a safe filesystem name
func SanitizeUsername(username string) string {
	// Convert to lowercase
	safe := strings.ToLower(username)

	// Replace spaces and special characters with underscores
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	safe = reg.ReplaceAllString(safe, "_")

	// Remove leading/trailing underscores
	safe = strings.Trim(safe, "_")

	// Ensure it's not empty
	if safe == "" {
		safe = "user"
	}

	return safe
}

func InitDB(filepath string) error {
	var err error
	db, err = sql.Open("sqlite3", filepath)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	// Set busy timeout for better concurrent access
	_, err = db.Exec("PRAGMA busy_timeout=5000;")
	if err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable WAL mode if requested (better for concurrent reads during writes)
	if UseWALMode {
		_, err = db.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			return fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	createTableSQL := `
	-- Unified table for SMS messages, MMS messages, and call logs
	-- record_type: 1 = SMS, 2 = MMS, 3 = call
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		record_type INTEGER NOT NULL DEFAULT 1,
		address TEXT NOT NULL,
		body TEXT,
		type INTEGER NOT NULL,
		date INTEGER NOT NULL,
		read INTEGER DEFAULT 0,
		thread_id INTEGER,
		subject TEXT,
		media_type TEXT,
		media_data BLOB,
		protocol INTEGER,
		status INTEGER,
		service_center TEXT,
		sub_id INTEGER,
		contact_name TEXT,
		sender TEXT,
		content_type TEXT,
		read_report INTEGER,
		read_status INTEGER,
		message_id TEXT,
		message_size INTEGER,
		message_type INTEGER,
		sim_slot INTEGER,
		addresses TEXT,
		duration INTEGER,
		presentation INTEGER,
		subscription_id TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_address ON messages(address);
	CREATE INDEX IF NOT EXISTS idx_date ON messages(date);
	CREATE INDEX IF NOT EXISTS idx_thread ON messages(thread_id);
	CREATE INDEX IF NOT EXISTS idx_record_type ON messages(record_type);
	CREATE INDEX IF NOT EXISTS idx_record_type_date ON messages(record_type, date);

	-- Create unique constraints for idempotent imports
	-- record_type differentiates SMS (1), MMS (2), and calls (3)
	CREATE UNIQUE INDEX IF NOT EXISTS idx_message_unique ON messages(record_type, address, date, type, COALESCE(body, ''), COALESCE(content_type, ''), COALESCE(message_id, ''), COALESCE(duration, 0));

	-- Create FTS5 virtual table for full-text search of messages
	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		message_id UNINDEXED,
		address UNINDEXED,
		body,
		contact_name UNINDEXED,
		date UNINDEXED,
		content='messages',
		content_rowid='id'
	);

	-- Create triggers to keep FTS table in sync
	CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, message_id, address, body, contact_name, date)
		VALUES (new.id, new.id, new.address, new.body, new.contact_name, new.date);
	END;

	CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, message_id, address, body, contact_name, date)
		VALUES('delete', old.id, old.id, old.address, old.body, old.contact_name, old.date);
	END;

	CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, message_id, address, body, contact_name, date)
		VALUES('delete', old.id, old.id, old.address, old.body, old.contact_name, old.date);
		INSERT INTO messages_fts(rowid, message_id, address, body, contact_name, date)
		VALUES (new.id, new.id, new.address, new.body, new.contact_name, new.date);
	END;
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	slog.Info("Database initialized successfully")
	return nil
}

// InitUserDB initializes a database for a specific user
func InitUserDB(userID string, filepath string) error {
	userDB, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return err
	}

	if err = userDB.Ping(); err != nil {
		return err
	}

	// Set busy timeout for better concurrent access
	_, err = userDB.Exec("PRAGMA busy_timeout=5000;")
	if err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable WAL mode if requested (better for concurrent reads during writes)
	if UseWALMode {
		_, err = userDB.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			return fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	createTableSQL := `
	-- Unified table for SMS messages, MMS messages, and call logs
	-- record_type: 1 = SMS, 2 = MMS, 3 = call
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		record_type INTEGER NOT NULL DEFAULT 1,
		address TEXT NOT NULL,
		body TEXT,
		type INTEGER NOT NULL,
		date INTEGER NOT NULL,
		read INTEGER DEFAULT 0,
		thread_id INTEGER,
		subject TEXT,
		media_type TEXT,
		media_data BLOB,
		protocol INTEGER,
		status INTEGER,
		service_center TEXT,
		sub_id INTEGER,
		contact_name TEXT,
		sender TEXT,
		content_type TEXT,
		read_report INTEGER,
		read_status INTEGER,
		message_id TEXT,
		message_size INTEGER,
		message_type INTEGER,
		sim_slot INTEGER,
		addresses TEXT,
		duration INTEGER,
		presentation INTEGER,
		subscription_id TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_address ON messages(address);
	CREATE INDEX IF NOT EXISTS idx_date ON messages(date);
	CREATE INDEX IF NOT EXISTS idx_thread ON messages(thread_id);
	CREATE INDEX IF NOT EXISTS idx_record_type ON messages(record_type);
	CREATE INDEX IF NOT EXISTS idx_record_type_date ON messages(record_type, date);

	-- record_type differentiates SMS (1), MMS (2), and calls (3)
	CREATE UNIQUE INDEX IF NOT EXISTS idx_message_unique ON messages(record_type, address, date, type, COALESCE(body, ''), COALESCE(content_type, ''), COALESCE(message_id, ''), COALESCE(duration, 0));

	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		message_id UNINDEXED,
		address UNINDEXED,
		body,
		contact_name UNINDEXED,
		date UNINDEXED,
		content='messages',
		content_rowid='id'
	);

	CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, message_id, address, body, contact_name, date)
		VALUES (new.id, new.id, new.address, new.body, new.contact_name, new.date);
	END;

	CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, message_id, address, body, contact_name, date)
		VALUES('delete', old.id, old.id, old.address, old.body, old.contact_name, old.date);
	END;

	CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, message_id, address, body, contact_name, date)
		VALUES('delete', old.id, old.id, old.address, old.body, old.contact_name, old.date);
		INSERT INTO messages_fts(rowid, message_id, address, body, contact_name, date)
		VALUES (new.id, new.id, new.address, new.body, new.contact_name, new.date);
	END;
	`

	_, err = userDB.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// Store in map
	userDBsMutex.Lock()
	userDBs[userID] = userDB
	userDBsMutex.Unlock()

	slog.Info("User database initialized", "user_id", userID, "path", filepath)
	return nil
}

// GetUserDB retrieves the database connection for a specific user, creating it if it doesn't exist
func GetUserDB(userID string, username string) (*sql.DB, error) {
	userDBsMutex.RLock()
	userDB, exists := userDBs[userID]
	userDBsMutex.RUnlock()

	if !exists {
		// Database not in cache, try to open or create it
		dbPathPrefix := os.Getenv("DB_PATH_PREFIX")
		if dbPathPrefix == "" {
			dbPathPrefix = "."
		}
		// Use UUID as database filename instead of sanitized username
		filepath := fmt.Sprintf("%s/sbv_%s.db", dbPathPrefix, userID)

		// InitUserDB will create the database if it doesn't exist
		if err := InitUserDB(userID, filepath); err != nil {
			return nil, fmt.Errorf("failed to initialize user database: %w", err)
		}

		userDBsMutex.RLock()
		userDB = userDBs[userID]
		userDBsMutex.RUnlock()
	}

	return userDB, nil
}

func InsertMessage(userDB *sql.DB, msg *Message) error {
	// Convert addresses slice to JSON string
	var addressesJSON string
	if len(msg.Addresses) > 0 {
		addresses := strings.Join(msg.Addresses, ",")
		addressesJSON = addresses
	}

	// Determine record type: 1 = SMS, 2 = MMS
	// MMS messages have ContentType set (e.g., 'application/vnd.wap.multipart.related')
	// SMS messages do not have ContentType
	recordType := 1 // Default to SMS
	if msg.ContentType != "" {
		recordType = 2 // MMS
	}

	query := `
		INSERT INTO messages (
			record_type, address, body, type, date, read, thread_id, subject, media_type, media_data,
			protocol, status, service_center, sub_id, contact_name, sender,
			content_type, read_report, read_status, message_id, message_size, message_type, sim_slot, addresses
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING
	`
	result, err := userDB.Exec(query,
		recordType, // record_type: 1 = SMS, 2 = MMS
		msg.Address,
		msg.Body,
		msg.Type,
		msg.Date.Unix(),
		msg.Read,
		msg.ThreadID,
		msg.Subject,
		msg.MediaType,
		msg.MediaData,
		msg.Protocol,
		msg.Status,
		msg.ServiceCenter,
		msg.SubID,
		msg.ContactName,
		msg.Sender,
		msg.ContentType,
		msg.ReadReport,
		msg.ReadStatus,
		msg.MessageID,
		msg.MessageSize,
		msg.MessageType,
		msg.SimSlot,
		addressesJSON,
	)
	if err != nil {
		slog.Debug("InsertMessage: Error inserting message", "error", err)
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	msg.ID = id

	return nil
}

func InsertCallLog(userDB *sql.DB, call *CallLog) error {
	query := `
		INSERT INTO messages (record_type, address, type, date, duration, presentation, subscription_id, contact_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING
	`
	result, err := userDB.Exec(query,
		3, // record_type: 3 = call
		call.Number,
		call.Type,
		call.Date.Unix(),
		call.Duration,
		call.Presentation,
		call.SubscriptionID,
		call.ContactName,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	call.ID = id
	return nil
}

// InsertCallLogBatch inserts multiple call logs in a single transaction for better performance
func InsertCallLogBatch(userDB *sql.DB, calls []CallLog) error {
	if len(calls) == 0 {
		return nil
	}

	tx, err := userDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO messages (record_type, address, type, date, duration, presentation, subscription_id, contact_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range calls {
		_, err := stmt.Exec(
			3, // record_type: 3 = call
			calls[i].Number,
			calls[i].Type,
			calls[i].Date.Unix(),
			calls[i].Duration,
			calls[i].Presentation,
			calls[i].SubscriptionID,
			calls[i].ContactName,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func GetConversations(userDB *sql.DB, startDate, endDate *time.Time) ([]Conversation, error) {
	// Build a query that groups all activity (messages and calls) by address
	query := `
		SELECT
			address,
			MAX(COALESCE(contact_name, '')) as contact_name,
			(
				SELECT COALESCE(subject, '')
				FROM messages m2
				WHERE m2.address = messages.address
					AND m2.subject IS NOT NULL
					AND m2.subject != ''
				ORDER BY date DESC
				LIMIT 1
			) as subject,
			(
				SELECT
					CASE
						WHEN record_type = 1 THEN body  -- SMS
						WHEN record_type = 2 THEN body  -- MMS
						WHEN record_type = 3 AND type = 1 THEN 'Incoming call'
						WHEN record_type = 3 AND type = 2 THEN 'Outgoing call'
						WHEN record_type = 3 AND type = 3 THEN 'Missed call'
						WHEN record_type = 3 AND type = 4 THEN 'Voicemail'
						WHEN record_type = 3 AND type = 5 THEN 'Rejected call'
						WHEN record_type = 3 AND type = 6 THEN 'Refused call'
						ELSE 'Call'
					END
				FROM messages m3
				WHERE m3.address = messages.address
				ORDER BY date DESC
				LIMIT 1
			) as last_message,
			MAX(date) as last_date,
			COUNT(*) as activity_count
		FROM messages
		WHERE 1=1
	`

	args := []interface{}{}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, startDate.Unix())
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, endDate.Unix())
	}

	query += `
		GROUP BY address
		ORDER BY last_date DESC
	`

	rows, err := userDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []Conversation{}
	for rows.Next() {
		var c Conversation
		var lastDateUnix int64
		var subject sql.NullString
		err := rows.Scan(&c.Address, &c.ContactName, &subject, &c.LastMessage, &lastDateUnix, &c.MessageCount)
		if err != nil {
			return nil, err
		}
		c.LastDate = time.Unix(lastDateUnix, 0)
		c.Subject = subject.String
		c.Type = "conversation" // Changed from "message" or "call" to indicate it's a merged conversation
		conversations = append(conversations, c)
	}

	return conversations, nil
}

func formatCallType(callType int) string {
	switch callType {
	case 1:
		return "Incoming call"
	case 2:
		return "Outgoing call"
	case 3:
		return "Missed call"
	case 4:
		return "Voicemail"
	case 5:
		return "Rejected call"
	case 6:
		return "Refused call"
	default:
		return "Call"
	}
}

func GetMessages(userDB *sql.DB, address string, startDate, endDate *time.Time) ([]Message, error) {
	query := `
		SELECT id, address, body, type, date, read, thread_id,
		       COALESCE(subject, ''), COALESCE(media_type, ''), COALESCE(media_data, ''),
		       COALESCE(protocol, 0), COALESCE(status, 0), COALESCE(service_center, ''),
		       COALESCE(sub_id, 0), COALESCE(contact_name, ''), COALESCE(sender, ''),
		       COALESCE(content_type, ''), COALESCE(read_report, 0), COALESCE(read_status, 0),
		       COALESCE(message_id, ''), COALESCE(message_size, 0), COALESCE(message_type, 0),
		       COALESCE(sim_slot, 0), COALESCE(addresses, '')
		FROM messages
		WHERE record_type IN (1, 2) AND address = ?  -- 1 = SMS, 2 = MMS
	`

	args := []interface{}{address}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, startDate.Unix())
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, endDate.Unix())
	}

	query += " ORDER BY date ASC"

	slog.Debug("GetMessages: executing query", "address", address)
	slog.Debug("GetMessages: SQL query", "query", query)
	slog.Debug("GetMessages: query arguments", "args", args)

	rows, err := userDB.Query(query, args...)
	if err != nil {
		slog.Debug("GetMessages: Query error", "error", err)
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		var dateUnix int64
		var readInt int
		var addressesStr string
		err := rows.Scan(&m.ID, &m.Address, &m.Body, &m.Type, &dateUnix,
			&readInt, &m.ThreadID, &m.Subject, &m.MediaType, &m.MediaData,
			&m.Protocol, &m.Status, &m.ServiceCenter, &m.SubID, &m.ContactName, &m.Sender,
			&m.ContentType, &m.ReadReport, &m.ReadStatus, &m.MessageID,
			&m.MessageSize, &m.MessageType, &m.SimSlot, &addressesStr)
		if err != nil {
			return nil, err
		}
		m.Date = time.Unix(dateUnix, 0)
		m.Read = readInt == 1

		// Parse addresses from comma-separated string
		if addressesStr != "" {
			m.Addresses = strings.Split(addressesStr, ",")
		}

		// Don't load media data - it will be fetched on demand via /api/media
		// Clear MediaData to save memory in response
		m.MediaData = nil

		slog.Debug("GetMessages: Message", "id", m.ID, "address", m.Address, "media_type", m.MediaType, "body", truncateString(m.Body, 50))

		messages = append(messages, m)
	}

	slog.Debug("GetMessages: Returning messages", "count", len(messages), "address", address)
	return messages, nil
}

func GetCallLogs(userDB *sql.DB, number string, startDate, endDate *time.Time) ([]CallLog, error) {
	query := `
		SELECT id, address, duration, date, type,
		       COALESCE(presentation, 0), COALESCE(subscription_id, ''), COALESCE(contact_name, '')
		FROM messages
		WHERE record_type = 3 AND address = ?  -- 3 = call
	`

	args := []interface{}{number}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, startDate.Unix())
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, endDate.Unix())
	}

	query += " ORDER BY date ASC"

	rows, err := userDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	calls := []CallLog{}
	for rows.Next() {
		var c CallLog
		var dateUnix int64
		err := rows.Scan(&c.ID, &c.Number, &c.Duration, &dateUnix, &c.Type,
			&c.Presentation, &c.SubscriptionID, &c.ContactName)
		if err != nil {
			return nil, err
		}
		c.Date = time.Unix(dateUnix, 0)
		calls = append(calls, c)
	}

	return calls, nil
}

func GetAllCalls(userDB *sql.DB, startDate, endDate *time.Time, limit, offset int) ([]CallLog, error) {
	query := `
		SELECT id, address, duration, date, type,
		       COALESCE(presentation, 0), COALESCE(subscription_id, ''), COALESCE(contact_name, '')
		FROM messages
		WHERE record_type = 3  -- 3 = call
	`

	args := []interface{}{}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, startDate.Unix())
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, endDate.Unix())
	}

	query += " ORDER BY date ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := userDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	calls := []CallLog{}
	for rows.Next() {
		var c CallLog
		var dateUnix int64
		err := rows.Scan(&c.ID, &c.Number, &c.Duration, &dateUnix, &c.Type,
			&c.Presentation, &c.SubscriptionID, &c.ContactName)
		if err != nil {
			return nil, err
		}
		c.Date = time.Unix(dateUnix, 0)
		calls = append(calls, c)
	}

	return calls, nil
}

func GetActivity(userDB *sql.DB, startDate, endDate *time.Time, limit, offset int) ([]ActivityItem, error) {
	return GetActivityByAddress(userDB, "", startDate, endDate, limit, offset)
}

func GetActivityByAddress(userDB *sql.DB, address string, startDate, endDate *time.Time, limit, offset int) ([]ActivityItem, error) {
	var activities []ActivityItem

	// Query from unified table
	query := `
		SELECT record_type, date, address, COALESCE(contact_name, '') as contact_name,
		       id, body, type, read, thread_id, COALESCE(subject, ''),
		       COALESCE(media_type, ''), COALESCE(media_data, ''),
		       COALESCE(protocol, 0), COALESCE(status, 0), COALESCE(service_center, ''),
		       COALESCE(sub_id, 0), COALESCE(content_type, ''), COALESCE(read_report, 0),
		       COALESCE(read_status, 0), COALESCE(message_id, ''), COALESCE(message_size, 0),
		       COALESCE(message_type, 0), COALESCE(sim_slot, 0), COALESCE(addresses, ''),
		       COALESCE(duration, 0), COALESCE(presentation, 0), COALESCE(subscription_id, ''),
		       COALESCE(sender, '')
		FROM messages
		WHERE 1=1
	`

	args := []interface{}{}
	if address != "" {
		query += " AND address = ?"
		args = append(args, address)
	}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, startDate.Unix())
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, endDate.Unix())
	}

	query += " ORDER BY date ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	slog.Debug("GetActivityByAddress: executing query", "address", address, "limit", limit, "offset", offset)
	slog.Debug("GetActivityByAddress: SQL query", "query", query)
	slog.Debug("GetActivityByAddress: query arguments", "args", args)

	rows, err := userDB.Query(query, args...)
	if err != nil {
		slog.Debug("GetActivityByAddress: Query error", "error", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var recordType int64
		var dateUnix int64
		var address, contactName string

		// Shared fields
		var id sql.NullInt64
		var itemType sql.NullInt64 // type field - used for both message type and call type

		// Message fields
		var body, subject, mediaType, serviceCenter, contentType, messageID, subscriptionID, addressesStr, sender sql.NullString
		var readInt, threadID, protocol, status, subID, readReport, readStatus, messageSize, messageTypeField, simSlot sql.NullInt64
		var mediaData []byte

		// Call fields
		var duration, presentation sql.NullInt64

		err := rows.Scan(&recordType, &dateUnix, &address, &contactName,
			&id, &body, &itemType, &readInt, &threadID, &subject,
			&mediaType, &mediaData,
			&protocol, &status, &serviceCenter,
			&subID, &contentType, &readReport,
			&readStatus, &messageID, &messageSize,
			&messageTypeField, &simSlot, &addressesStr,
			&duration, &presentation, &subscriptionID, &sender)
		if err != nil {
			return nil, err
		}

		var activityTypeStr string
		if recordType == 1 || recordType == 2 {
			// 1 = SMS, 2 = MMS
			activityTypeStr = "message"
		} else if recordType == 3 {
			// 3 = call
			activityTypeStr = "call"
		}

		activity := ActivityItem{
			Type:        activityTypeStr,
			Date:        time.Unix(dateUnix, 0),
			Address:     address,
			ContactName: contactName,
		}

		if (recordType == 1 || recordType == 2) && id.Valid {
			// Handle SMS (1) and MMS (2)
			msg := &Message{
				ID:            id.Int64,
				Address:       address,
				Body:          body.String,
				Date:          time.Unix(dateUnix, 0),
				ThreadID:      int(threadID.Int64),
				Subject:       subject.String,
				MediaType:     mediaType.String,
				MediaData:     mediaData,
				Protocol:      int(protocol.Int64),
				Status:        int(status.Int64),
				ServiceCenter: serviceCenter.String,
				SubID:         int(subID.Int64),
				ContactName:   contactName,
				ContentType:   contentType.String,
				ReadReport:    int(readReport.Int64),
				ReadStatus:    int(readStatus.Int64),
				MessageID:     messageID.String,
				MessageSize:   int(messageSize.Int64),
				MessageType:   int(messageTypeField.Int64),
				SimSlot:       int(simSlot.Int64),
				Sender:        sender.String,
			}
			if itemType.Valid {
				msg.Type = int(itemType.Int64)
			}
			if readInt.Valid {
				msg.Read = readInt.Int64 == 1
			}

			// Parse addresses from comma-separated string
			slog.Debug("GetActivityByAddress: addressesStr raw", "id", id.Int64, "valid", addressesStr.Valid, "value", addressesStr.String)
			if addressesStr.Valid && addressesStr.String != "" {
				msg.Addresses = strings.Split(addressesStr.String, ",")
				slog.Debug("GetActivityByAddress: addresses split result", "id", id.Int64, "count", len(msg.Addresses), "values", msg.Addresses)
			} else if strings.Contains(address, ",") {
				// Fallback: If addresses field is empty but address contains commas,
				// this is a group conversation - parse the address field
				msg.Addresses = strings.Split(address, ",")
				slog.Debug("GetActivityByAddress: addresses from address field", "id", id.Int64, "count", len(msg.Addresses), "values", msg.Addresses)
			}

			// Don't load media data - it will be fetched on demand via /api/media
			// Clear MediaData to save memory in response
			msg.MediaData = nil

			slog.Debug("GetActivityByAddress: Message", "id", msg.ID, "address", msg.Address, "type", msg.Type, "sender", msg.Sender, "addresses", msg.Addresses, "media_type", msg.MediaType, "body", truncateString(msg.Body, 50))

			activity.Message = msg
		} else if recordType == 3 && id.Valid {
			// Handle calls (3)
			call := &CallLog{
				ID:             id.Int64,
				Number:         address,
				Duration:       int(duration.Int64),
				Date:           time.Unix(dateUnix, 0),
				Type:           int(itemType.Int64),
				Presentation:   int(presentation.Int64),
				SubscriptionID: subscriptionID.String,
				ContactName:    contactName,
			}
			slog.Debug("GetActivityByAddress: Call", "id", call.ID, "number", call.Number, "type", call.Type, "duration", call.Duration)
			activity.Call = call
		}

		activities = append(activities, activity)
	}

	slog.Debug("GetActivityByAddress: Returning activities", "count", len(activities), "address", address)
	return activities, nil
}

// GetMediaByAddress fetches only media items (images/videos) for a specific address
func GetMediaByAddress(userDB *sql.DB, address string, startDate, endDate *time.Time) ([]Message, error) {
	query := `
		SELECT id, address, COALESCE(body, '') as body, date,
		       COALESCE(contact_name, '') as contact_name, COALESCE(media_type, '') as media_type,
		       read, thread_id
		FROM messages
		WHERE record_type IN (1, 2)
		AND media_type IS NOT NULL
		AND media_type != ''
		AND (media_type LIKE 'image/%' OR media_type LIKE 'video/%')
	`

	args := []interface{}{}
	if address != "" {
		query += " AND address = ?"
		args = append(args, address)
	}
	if startDate != nil {
		query += " AND date >= ?"
		args = append(args, startDate.Unix())
	}
	if endDate != nil {
		query += " AND date <= ?"
		args = append(args, endDate.Unix())
	}

	query += " ORDER BY date DESC"

	rows, err := userDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mediaItems []Message
	for rows.Next() {
		var m Message
		var dateUnix int64
		var readInt int64

		err := rows.Scan(&m.ID, &m.Address, &m.Body, &dateUnix, &m.ContactName, &m.MediaType, &readInt, &m.ThreadID)
		if err != nil {
			return nil, err
		}

		m.Date = time.Unix(dateUnix, 0)
		m.Read = readInt == 1

		mediaItems = append(mediaItems, m)
	}

	return mediaItems, nil
}

func GetMessageMedia(userDB *sql.DB, messageID string) ([]byte, string, error) {
	query := `
		SELECT COALESCE(media_data, ''), COALESCE(media_type, '')
		FROM messages
		WHERE id = ? AND record_type IN (1, 2)  -- 1 = SMS, 2 = MMS
	`

	slog.Debug("GetMessageMedia: Fetching media", "message_id", messageID)
	slog.Debug("GetMessageMedia: SQL query", "query", query)

	var mediaData []byte
	var mediaType string

	err := userDB.QueryRow(query, messageID).Scan(&mediaData, &mediaType)
	if err != nil {
		slog.Debug("GetMessageMedia: Error scanning row", "message_id", messageID, "error", err)
		return nil, "", err
	}

	slog.Debug("GetMessageMedia: Found media", "media_type", mediaType, "data_length", len(mediaData), "message_id", messageID)

	if len(mediaData) == 0 || mediaType == "" {
		slog.Debug("GetMessageMedia: No media found", "message_id", messageID)
		return nil, "", fmt.Errorf("no media found")
	}

	// Convert HEIC to JPEG if needed
	if isHEICContentType(mediaType) {
		convertedData, err := convertHEICtoJPEG(mediaData)
		if err != nil {
			slog.Error("Failed to convert HEIC to JPEG", "message_id", messageID, "error", err)
			// Return original if conversion fails
			return mediaData, mediaType, nil
		}
		return convertedData, "image/jpeg", nil
	}

	// Convert unsupported video formats (3GP, etc.) to MP4 if needed
	if needsVideoConversion(mediaType) {
		slog.Info("Converting video to MP4", "from_type", mediaType, "message_id", messageID)
		convertedData, err := convertVideoToMP4(mediaData)
		if err != nil {
			slog.Error("Failed to convert video to MP4", "message_id", messageID, "error", err)
			// Return original if conversion fails
			return mediaData, mediaType, nil
		}
		slog.Info("Successfully converted video to MP4", "message_id", messageID)
		return convertedData, "video/mp4", nil
	}

	return mediaData, mediaType, nil
}

func GetDateRange(userDB *sql.DB) (time.Time, time.Time, error) {
	var minDate, maxDate int64

	// Get min/max from unified messages table
	query := "SELECT MIN(date), MAX(date) FROM messages"
	var min, max sql.NullInt64
	err := userDB.QueryRow(query).Scan(&min, &max)
	if err != nil && err != sql.ErrNoRows {
		return time.Time{}, time.Time{}, err
	}

	if !min.Valid || !max.Valid {
		return time.Time{}, time.Time{}, fmt.Errorf("no data available")
	}

	minDate = min.Int64
	maxDate = max.Int64

	return time.Unix(minDate, 0), time.Unix(maxDate, 0), nil
}

// SearchResult represents a message search result
type SearchResult struct {
	MessageID   int64     `json:"message_id"`
	Address     string    `json:"address"`
	ContactName string    `json:"contact_name"`
	Body        string    `json:"body"`
	Date        time.Time `json:"date"`
	Snippet     string    `json:"snippet"`
}

// SearchMessages performs full-text search on message contents
func SearchMessages(userDB *sql.DB, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	sqlQuery := `
		SELECT
			m.id,
			m.address,
			COALESCE(m.contact_name, ''),
			m.body,
			m.date,
			snippet(messages_fts, 2, '<mark>', '</mark>', '...', 50) as snippet
		FROM messages_fts
		JOIN messages m ON messages_fts.rowid = m.id
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := userDB.Query(sqlQuery, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		var dateUnix int64
		err := rows.Scan(&r.MessageID, &r.Address, &r.ContactName, &r.Body, &dateUnix, &r.Snippet)
		if err != nil {
			return nil, err
		}
		r.Date = time.Unix(dateUnix, 0)
		results = append(results, r)
	}

	return results, nil
}
