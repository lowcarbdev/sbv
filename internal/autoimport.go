package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AutoImportService manages automatic file imports for all users
type AutoImportService struct {
	dataDir        string
	checkInterval  time.Duration
	cancelFunc     context.CancelFunc
	ctx            context.Context
}

// NewAutoImportService creates a new auto-import service
func NewAutoImportService(dataDir string) *AutoImportService {
	ctx, cancel := context.WithCancel(context.Background())
	return &AutoImportService{
		dataDir:       dataDir,
		checkInterval: 1 * time.Minute,
		cancelFunc:    cancel,
		ctx:           ctx,
	}
}

// Start begins the auto-import background job
func (s *AutoImportService) Start() {
	slog.Info("Starting auto-import service", "checkInterval", s.checkInterval)

	go func() {
		// Run immediately on start
		s.scanAllUsers()

		// Then run on interval
		ticker := time.NewTicker(s.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.scanAllUsers()
			case <-s.ctx.Done():
				slog.Info("Auto-import service stopped")
				return
			}
		}
	}()
}

// Stop gracefully stops the auto-import service
func (s *AutoImportService) Stop() {
	slog.Info("Stopping auto-import service")
	s.cancelFunc()
}

// scanAllUsers scans all user directories for files to import
func (s *AutoImportService) scanAllUsers() {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		slog.Error("Failed to read data directory", "error", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		userID := entry.Name()
		s.scanUserDirectory(userID)
	}
}

// scanUserDirectory scans a single user's ingest directory
func (s *AutoImportService) scanUserDirectory(userID string) {
	ingestDir := filepath.Join(s.dataDir, userID, "ingest")

	// Check if ingest directory exists
	if _, err := os.Stat(ingestDir); os.IsNotExist(err) {
		// Create ingest directory if it doesn't exist
		if err := os.MkdirAll(ingestDir, 0755); err != nil {
			slog.Error("Failed to create ingest directory", "userID", userID, "error", err)
		}
		return
	}

	entries, err := os.ReadDir(ingestDir)
	if err != nil {
		slog.Error("Failed to read ingest directory", "userID", userID, "error", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Skip hidden files (starting with .)
		if strings.HasPrefix(filename, ".") {
			continue
		}

		// Skip log files
		if strings.HasSuffix(filename, ".log") {
			continue
		}

		filePath := filepath.Join(ingestDir, filename)
		s.processFile(userID, filePath, filename)
	}
}

// processFile processes a single file for import
func (s *AutoImportService) processFile(userID, filePath, filename string) {
	// Check if file is stable (not being written to)
	if !s.isFileStable(filePath) {
		slog.Debug("File not stable yet, skipping", "userID", userID, "file", filename)
		return
	}

	slog.Info("Processing file for import", "userID", userID, "file", filename)

	// Create log file for this import
	logPath := filePath + ".log"
	logFile, err := os.Create(logPath)
	if err != nil {
		slog.Error("Failed to create log file", "userID", userID, "file", filename, "error", err)
		return
	}
	defer logFile.Close()

	logWriter := &importLogger{
		file:   logFile,
		userID: userID,
		filename: filename,
	}

	logWriter.log("Starting import of %s", filename)
	startTime := time.Now()

	// Get username from auth database
	username, err := GetUsernameByID(userID)
	if err != nil {
		logWriter.log("ERROR: Failed to get username: %v", err)
		slog.Error("Failed to get username", "userID", userID, "error", err)
		return
	}

	// Get user database
	userDB, err := GetUserDB(userID, username)
	if err != nil {
		logWriter.log("ERROR: Failed to get user database: %v", err)
		slog.Error("Failed to get user database", "userID", userID, "error", err)
		return
	}

	// Determine file type and parse
	var parseErr error
	if strings.HasSuffix(strings.ToLower(filename), ".xml") {
		logWriter.log("Detected XML backup file")
		parseErr = s.parseXMLBackup(userDB, filePath, logWriter)
	} else {
		logWriter.log("ERROR: Unsupported file type")
		slog.Warn("Unsupported file type", "userID", userID, "file", filename)
		return
	}

	// Move file to complete directory
	completeDir := filepath.Join(s.dataDir, userID, "complete")
	if err := os.MkdirAll(completeDir, 0755); err != nil {
		logWriter.log("ERROR: Failed to create complete directory: %v", err)
		slog.Error("Failed to create complete directory", "userID", userID, "error", err)
		return
	}

	// Generate unique filename if file already exists in complete dir
	completePath := filepath.Join(completeDir, filename)
	if _, err := os.Stat(completePath); err == nil {
		// File exists, add timestamp
		timestamp := time.Now().Format("20060102_150405")
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		filename = fmt.Sprintf("%s_%s%s", base, timestamp, ext)
		completePath = filepath.Join(completeDir, filename)
	}

	duration := time.Since(startTime)

	if parseErr != nil {
		logWriter.log("ERROR: Import failed: %v", parseErr)
		logWriter.log("File will remain in ingest directory for manual review")
		logWriter.log("Import duration: %s", duration)
		slog.Error("Import failed", "userID", userID, "file", filename, "error", parseErr, "duration", duration)
	} else {
		// Move file to complete directory
		if err := os.Rename(filePath, completePath); err != nil {
			logWriter.log("ERROR: Failed to move file to complete directory: %v", err)
			slog.Error("Failed to move file", "userID", userID, "error", err)
			return
		}

		// Move log file too
		logDestPath := completePath + ".log"
		logFile.Close() // Close before moving
		if err := os.Rename(logPath, logDestPath); err != nil {
			slog.Warn("Failed to move log file", "userID", userID, "error", err)
		}

		logWriter.log("Import completed successfully in %s", duration)
		logWriter.log("File moved to: %s", completePath)
		slog.Info("Import completed", "userID", userID, "file", filename, "duration", duration)
	}
}

// isFileStable checks if a file has finished being written
// Returns true if file size hasn't changed in the last 5 seconds
func (s *AutoImportService) isFileStable(filePath string) bool {
	info1, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	size1 := info1.Size()
	mod1 := info1.ModTime()

	// Wait 5 seconds
	time.Sleep(5 * time.Second)

	info2, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	size2 := info2.Size()
	mod2 := info2.ModTime()

	// File is stable if size and modification time haven't changed
	return size1 == size2 && mod1.Equal(mod2)
}

// parseXMLBackup parses an XML backup file
func (s *AutoImportService) parseXMLBackup(userDB *sql.DB, filePath string, logger *importLogger) error {
	logger.log("Parsing XML backup file")

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for progress tracking
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()
	logger.log("File size: %d bytes", fileSize)

	// Parse the XML backup using streaming parser
	totalProcessed, totalSkipped, err := ParseSMSBackupStreaming(userDB, file, 100)
	if err != nil {
		return fmt.Errorf("failed to parse backup: %w", err)
	}

	logger.log("Import statistics:")
	logger.log("  Total processed: %d", totalProcessed)
	logger.log("  Total skipped (duplicates): %d", totalSkipped)

	return nil
}

// importLogger writes log messages to a file
type importLogger struct {
	file     *os.File
	userID   string
	filename string
}

func (l *importLogger) log(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	l.file.WriteString(logLine)
	l.file.Sync() // Ensure it's written to disk

	slog.Info("Auto-import", "userID", l.userID, "file", l.filename, "message", message)
}
