package internal

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestDBBlobStoreWrite(t *testing.T) {
	s := &DBBlobStore{}
	path, err := s.Write([]byte("data"), "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func TestDBBlobStoreRead(t *testing.T) {
	s := &DBBlobStore{}
	data, mediaType, err := s.Read("anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data")
	}
	if mediaType != "" {
		t.Errorf("expected empty mediaType, got %q", mediaType)
	}
}

func TestMediaTypeToExt(t *testing.T) {
	cases := []struct {
		mime string
		ext  string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/heic", ".heic"},
		{"image/heif", ".heif"},
		{"video/mp4", ".mp4"},
		{"video/3gpp", ".3gp"},
		{"video/quicktime", ".mov"},
		{"audio/mpeg", ".mp3"},
		{"audio/amr", ".amr"},
		{"audio/ogg", ".ogg"},
		{"audio/aac", ".aac"},
		{"text/vcard", ".vcf"},
		{"text/x-vcard", ".vcf"},
		{"image/jpeg; charset=utf-8", ".jpg"},
		{"image/x-custom", ".x-custom"},
		{"application/octet-stream", ".octet-stream"},
		{"", ".bin"},
	}
	for _, c := range cases {
		got := mediaTypeToExt(c.mime)
		if got != c.ext {
			t.Errorf("mediaTypeToExt(%q) = %q, want %q", c.mime, got, c.ext)
		}
	}
}

func TestExtToMediaType(t *testing.T) {
	cases := []struct {
		ext      string
		expected string
	}{
		{".jpg", "image/jpeg"},
		{".png", "image/png"},
		{".mp4", "video/mp4"},
		{".mp3", "audio/mpeg"},
		{".vcf", "text/vcard"},
		{".unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, c := range cases {
		got := extToMediaType(c.ext)
		if got != c.expected {
			t.Errorf("extToMediaType(%q) = %q, want %q", c.ext, got, c.expected)
		}
	}
}

func TestDiskBlobStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := NewDiskBlobStore(dir)

	data := []byte("hello media")
	mediaType := "image/jpeg"

	path, err := s.Write(data, mediaType)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if !strings.HasSuffix(path, ".jpg") {
		t.Errorf("expected .jpg extension, got %q", path)
	}

	got, gotType, err := s.Read(path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("data mismatch: got %q, want %q", got, data)
	}
	if gotType != mediaType {
		t.Errorf("mediaType mismatch: got %q, want %q", gotType, mediaType)
	}
}

func TestDiskBlobStoreDedup(t *testing.T) {
	dir := t.TempDir()
	s := NewDiskBlobStore(dir)

	data := []byte("duplicate content")
	path1, err := s.Write(data, "image/png")
	if err != nil {
		t.Fatalf("first Write failed: %v", err)
	}
	path2, err := s.Write(data, "image/png")
	if err != nil {
		t.Fatalf("second Write failed: %v", err)
	}
	if path1 != path2 {
		t.Errorf("expected same path for duplicate, got %q and %q", path1, path2)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file on disk, got %d", len(entries))
	}
}


func TestDiskBlobStoreReadMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewDiskBlobStore(dir)
	_, _, err := s.Read("nonexistent.jpg")
	if err == nil {
		t.Error("expected error reading nonexistent file")
	}
}

func TestGetUserBlobStore(t *testing.T) {
	dir := t.TempDir()
	s := NewDiskBlobStore(dir)
	SetGlobalBlobStore(s)
	defer func() { SetGlobalBlobStore(&DBBlobStore{}) }()

	store := GetUserBlobStore("user-abc")
	if store != s {
		t.Error("expected GlobalBlobStore to be returned")
	}
}

func TestAddMediaFilePathColumn(t *testing.T) {
	// Test that the column is added on a fresh DB and is idempotent on repeated calls
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	testDB, err := sql.Open("sqlite3", tmpDB)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer testDB.Close()

	// Create minimal messages table without media_file_path
	_, err = testDB.Exec(`CREATE TABLE messages (
		id INTEGER PRIMARY KEY,
		media_data BLOB,
		media_type TEXT
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// First call should add the column
	if err := addMediaFilePathColumn(testDB); err != nil {
		t.Fatalf("first addMediaFilePathColumn: %v", err)
	}

	// Second call should be a no-op (idempotent)
	if err := addMediaFilePathColumn(testDB); err != nil {
		t.Fatalf("second addMediaFilePathColumn: %v", err)
	}

	// Verify column exists by inserting a row with it
	_, err = testDB.Exec(`INSERT INTO messages (id, media_file_path) VALUES (1, 'test.jpg')`)
	if err != nil {
		t.Fatalf("insert with media_file_path: %v", err)
	}
}

// setupMessagesTable creates the messages table (full schema) and adds the
// media_file_path column plus an FTS virtual table, so tests can call InsertMessage.
func setupMessagesTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS messages (
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
	)`)
	if err != nil {
		t.Fatalf("create messages table: %v", err)
	}
	if err := addMediaFilePathColumn(db); err != nil {
		t.Fatalf("addMediaFilePathColumn: %v", err)
	}
	// FTS table needed for InsertMessage triggers
	_, _ = db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		message_id UNINDEXED, address UNINDEXED, body, contact_name UNINDEXED, date UNINDEXED,
		content='messages', content_rowid='id'
	)`)
}

func TestGetMessageMediaFallback(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	testDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer testDB.Close()
	setupMessagesTable(t, testDB)

	fakeMedia := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	msg := &Message{
		Address:     "+15551234567",
		Body:        "test",
		Type:        1,
		Date:        time.Now(),
		MediaType:   "image/jpeg",
		MediaData:   fakeMedia,
		ContentType: "application/vnd.wap.multipart.related",
	}
	if err := InsertMessage(testDB, msg); err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	// Use DBBlobStore (default) — should serve from media_data
	SetGlobalBlobStore(&DBBlobStore{})

	gotData, gotType, err := GetMessageMedia(testDB, "test-user", fmt.Sprintf("%d", msg.ID))
	if err != nil {
		t.Fatalf("GetMessageMedia: %v", err)
	}
	if string(gotData) != string(fakeMedia) {
		t.Error("data mismatch")
	}
	if gotType != "image/jpeg" {
		t.Errorf("type = %q, want image/jpeg", gotType)
	}
}

func TestGetMessageMediaNeither(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	testDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer testDB.Close()
	setupMessagesTable(t, testDB)

	// Insert message with no media
	msg := &Message{
		Address:     "+15551234567",
		Body:        "no media here",
		Type:        1,
		Date:        time.Now(),
		ContentType: "application/vnd.wap.multipart.related",
	}
	if err := InsertMessage(testDB, msg); err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	_, _, err = GetMessageMedia(testDB, "test-user", fmt.Sprintf("%d", msg.ID))
	if err == nil {
		t.Error("expected error for message with no media")
	}
}


func TestInsertMessageWithDiskBlobStore(t *testing.T) {
	// Setup: temp dir, real SQLite DB, DiskBlobStore configured
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	testDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer testDB.Close()

	setupMessagesTable(t, testDB)

	// Configure disk blob store
	blobDir := filepath.Join(dir, "media")
	var store BlobStore = NewDiskBlobStore(blobDir)
	SetGlobalBlobStore(store)
	defer func() { SetGlobalBlobStore(&DBBlobStore{}) }()

	fakeMedia := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes
	msg := &Message{
		Address:     "+15551234567",
		Body:        "test",
		Type:        1,
		Date:        time.Now(),
		MediaType:   "image/jpeg",
		MediaData:   fakeMedia,
		ContentType: "application/vnd.wap.multipart.related",
	}

	// Simulate what ParseSMSBackupStreaming does
	if msg.MediaData != nil {
		if _, isDB := store.(*DBBlobStore); !isDB {
			filePath, blobErr := store.Write(msg.MediaData, msg.MediaType)
			if blobErr != nil {
				t.Fatalf("blob write: %v", blobErr)
			}
			msg.MediaFilePath = filePath
			msg.MediaData = nil
		}
	}

	if err := InsertMessage(testDB, msg); err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	// Verify: media_data is NULL in DB
	var mediaData []byte
	var mediaFilePath string
	row := testDB.QueryRow("SELECT media_data, COALESCE(media_file_path,'') FROM messages WHERE id = ?", msg.ID)
	if err := row.Scan(&mediaData, &mediaFilePath); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if mediaData != nil {
		t.Error("expected media_data to be NULL in DB")
	}
	if mediaFilePath == "" {
		t.Error("expected media_file_path to be set")
	}

	// Verify: file exists on disk
	diskPath := filepath.Join(blobDir, mediaFilePath)
	diskData, err := os.ReadFile(diskPath)
	if err != nil {
		t.Fatalf("read disk file: %v", err)
	}
	if string(diskData) != string(fakeMedia) {
		t.Error("disk file content mismatch")
	}

	// Verify: GetMessageMedia reads it back correctly
	gotData, gotType, err := GetMessageMedia(testDB, "test-user", fmt.Sprintf("%d", msg.ID))
	if err != nil {
		t.Fatalf("GetMessageMedia: %v", err)
	}
	if string(gotData) != string(fakeMedia) {
		t.Error("GetMessageMedia data mismatch")
	}
	if gotType != "image/jpeg" {
		t.Errorf("GetMessageMedia type = %q, want image/jpeg", gotType)
	}
}
