package internal

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SMSBackup struct {
	XMLName  xml.Name    `xml:"smses"`
	Count    int         `xml:"count,attr"`
	Messages []SMSEntry  `xml:"sms"`
	MMS      []MMSEntry  `xml:"mms"`
	Calls    []CallEntry `xml:"call"`
}

type SMSEntry struct {
	Address       string `xml:"address,attr"`
	Date          string `xml:"date,attr"`
	Type          string `xml:"type,attr"`
	Body          string `xml:"body,attr"`
	Read          string `xml:"read,attr"`
	ThreadID      string `xml:"thread_id,attr"`
	Subject       string `xml:"subject,attr"`
	Protocol      string `xml:"protocol,attr"`
	TOA           string `xml:"toa,attr"`
	SCTOA         string `xml:"sc_toa,attr"`
	ServiceCenter string `xml:"service_center,attr"`
	Status        string `xml:"status,attr"`
	SubID         string `xml:"sub_id,attr"`
	ReadableDate  string `xml:"readable_date,attr"`
	ContactName   string `xml:"contact_name,attr"`
}

type MMSEntry struct {
	Address      string    `xml:"address,attr"`
	Date         string    `xml:"date,attr"`
	Type         string    `xml:"msg_box,attr"`
	Read         string    `xml:"read,attr"`
	ThreadID     string    `xml:"thread_id,attr"`
	Subject      string    `xml:"sub,attr"`
	TrID         string    `xml:"tr_id,attr"`
	ContentType  string    `xml:"ct_t,attr"`
	ReadReport   string    `xml:"rr,attr"`
	ReadStatus   string    `xml:"read_status,attr"`
	MessageID    string    `xml:"m_id,attr"`
	MessageSize  string    `xml:"m_size,attr"`
	MessageType  string    `xml:"m_type,attr"`
	SimSlot      string    `xml:"sim_slot,attr"`
	ReadableDate string    `xml:"readable_date,attr"`
	ContactName  string    `xml:"contact_name,attr"`
	Parts        []MMSPart `xml:"parts>part"`
	Addrs        []MMSAddr `xml:"addrs>addr"`
	Body         string    `xml:"body,attr"`
}

type MMSPart struct {
	Seq         string `xml:"seq,attr"`
	ContentType string `xml:"ct,attr"`
	Name        string `xml:"name,attr"`
	Charset     string `xml:"chset,attr"`
	CL          string `xml:"cl,attr"`
	Text        string `xml:"text,attr"`
	Data        string `xml:"data,attr"`
}

type MMSAddr struct {
	Address string `xml:"address,attr"`
	Type    string `xml:"type,attr"`
	Charset string `xml:"charset,attr"`
}

type CallEntry struct {
	Number         string `xml:"number,attr"`
	Duration       string `xml:"duration,attr"`
	Date           string `xml:"date,attr"`
	Type           string `xml:"type,attr"`
	Presentation   string `xml:"presentation,attr"`
	SubscriptionID string `xml:"subscription_id,attr"`
	ReadableDate   string `xml:"readable_date,attr"`
	ContactName    string `xml:"contact_name,attr"`
}

type ParseResult struct {
	Messages []Message
	Calls    []CallLog
}

func ParseSMSBackup(r io.Reader) (ParseResult, error) {
	var backup SMSBackup
	decoder := xml.NewDecoder(r)
	err := decoder.Decode(&backup)
	if err != nil {
		return ParseResult{}, err
	}

	var result ParseResult

	// Parse SMS messages
	for _, sms := range backup.Messages {
		msg, err := convertSMSEntry(sms)
		if err != nil {
			slog.Error("Error parsing SMS", "error", err)
			continue
		}
		result.Messages = append(result.Messages, msg)
	}

	// Parse MMS messages
	for _, mms := range backup.MMS {
		msg, err := convertMMSEntry(mms)
		if err != nil {
			slog.Error("Error parsing MMS", "error", err)
			continue
		}
		result.Messages = append(result.Messages, msg)
	}

	// Parse call logs
	for _, call := range backup.Calls {
		callLog, err := convertCallEntry(call)
		if err != nil {
			slog.Error("Error parsing call log", "error", err)
			continue
		}
		result.Calls = append(result.Calls, callLog)
	}

	return result, nil
}

func convertSMSEntry(sms SMSEntry) (Message, error) {
	dateMs, err := strconv.ParseInt(sms.Date, 10, 64)
	if err != nil {
		return Message{}, err
	}

	msgType, _ := strconv.Atoi(sms.Type)
	read := sms.Read == "1"
	threadID, _ := strconv.Atoi(sms.ThreadID)
	protocol, _ := strconv.Atoi(sms.Protocol)
	status, _ := strconv.Atoi(sms.Status)
	subID, _ := strconv.Atoi(sms.SubID)

	// Normalize the phone number to remove formatting differences
	normalizedAddress := normalizePhoneNumber(sms.Address)

	// For SMS, the address is the single phone number
	addresses := []string{}
	if normalizedAddress != "" {
		addresses = append(addresses, normalizedAddress)
	}

	// For received SMS messages, the sender is the address
	var sender string
	if msgType == 1 && normalizedAddress != "" {
		sender = normalizedAddress
	}

	return Message{
		Address:       normalizedAddress,
		Body:          sms.Body,
		Type:          msgType,
		Date:          time.Unix(dateMs/1000, 0),
		Read:          read,
		ThreadID:      threadID,
		Subject:       normalizeNullString(sms.Subject),
		Protocol:      protocol,
		Status:        status,
		ServiceCenter: sms.ServiceCenter,
		SubID:         subID,
		ContactName:   sms.ContactName,
		Sender:        sender,
		Addresses:     addresses,
	}, nil
}

func convertMMSEntry(mms MMSEntry) (Message, error) {
	dateMs, err := strconv.ParseInt(mms.Date, 10, 64)
	if err != nil {
		return Message{}, err
	}

	msgType, _ := strconv.Atoi(mms.Type)
	read := mms.Read == "1"
	threadID, _ := strconv.Atoi(mms.ThreadID)
	readReport, _ := strconv.Atoi(mms.ReadReport)
	readStatus, _ := strconv.Atoi(mms.ReadStatus)
	messageSize, _ := strconv.Atoi(mms.MessageSize)
	messageType, _ := strconv.Atoi(mms.MessageType)
	simSlot, _ := strconv.Atoi(mms.SimSlot)

	// Normalize the phone number to remove formatting differences
	normalizedAddress := normalizePhoneNumber(mms.Address)

	// Extract all addresses from MMS and find the sender (type 137 = FROM)
	// Include ALL addresses to keep group conversations consistent
	addressMap := make(map[string]bool)
	var senderAddress string
	var firstAddress string

	for _, addr := range mms.Addrs {
		if addr.Address != "" {
			// Normalize each address to prevent duplicates due to formatting
			normalizedAddr := normalizePhoneNumber(addr.Address)
			if normalizedAddr != "" {
				addressMap[normalizedAddr] = true

				// Remember the first address we encounter
				if firstAddress == "" {
					firstAddress = normalizedAddr
				}

				// Type 137 (0x89) = FROM (sender in Android MMS)
				// For received messages, this tells us who sent it
				addrType, _ := strconv.Atoi(addr.Type)
				if addrType == 137 {
					senderAddress = normalizedAddr
				}
			}
		}
	}

	// If no type 137 sender was found for a received message, use the first address
	// or the single address for 1-on-1 conversations
	if msgType == 1 && senderAddress == "" {
		if len(addressMap) == 1 && firstAddress != "" {
			// 1-on-1 conversation: the single address is definitely the sender
			senderAddress = firstAddress
		} else if len(addressMap) > 1 && firstAddress != "" {
			// Group conversation without explicit sender: use first address as best guess
			senderAddress = firstAddress
		}
	}

	// Convert map to sorted, deduplicated slice
	addresses := make([]string, 0, len(addressMap))
	for addr := range addressMap {
		addresses = append(addresses, addr)
	}

	// Sort addresses for consistency
	sort.Strings(addresses)

	// Determine the primary address field for conversation grouping
	var primaryAddress string
	if len(addresses) >= 3 {
		// Group MMS (3+ participants) - join all normalized addresses to create a consistent group identifier
		primaryAddress = strings.Join(addresses, ",")
	} else if len(addresses) > 0 {
		// MMS with 1-2 addresses - use the normalized address
		primaryAddress = normalizedAddress
	} else {
		// Fallback to normalized mms.Address if no addresses found in mms.Addrs
		primaryAddress = normalizedAddress
	}

	// For received messages, store the sender in the Sender field
	// This allows us to display who sent each message in the UI
	var sender string
	if msgType == 1 && senderAddress != "" {
		// Received message - store the sender address
		sender = senderAddress
	}

	msg := Message{
		Address:     primaryAddress,
		Type:        msgType,
		Date:        time.Unix(dateMs/1000, 0),
		Read:        read,
		ThreadID:    threadID,
		Subject:     normalizeNullString(mms.Subject),
		ContentType: mms.ContentType,
		ReadReport:  readReport,
		ReadStatus:  readStatus,
		MessageID:   mms.MessageID,
		MessageSize: messageSize,
		MessageType: messageType,
		SimSlot:     simSlot,
		ContactName: mms.ContactName,
		Sender:      sender,
		Addresses:   addresses,
	}

	// Extract body text and media from parts
	var bodyText string
	for _, part := range mms.Parts {
		// Skip SMIL content - it's presentation metadata, not actual message content
		if isSMILContentType(part.ContentType) {
			continue
		}

		// Check for VCF (vCard) files - these are text/* but should be treated as media attachments
		if isVCardContentType(part.ContentType) && part.Data != "" {
			if msg.MediaType == "" { // Only store first media item
				data, err := base64.StdEncoding.DecodeString(part.Data)
				if err == nil {
					msg.MediaType = part.ContentType
					msg.MediaData = data
				}
			}
			continue
		}

		// Check for media - media parts often have text="null" which should be ignored
		if part.ContentType != "" && part.Data != "" && !isTextContentType(part.ContentType) {
			// This is media content (image, video, audio, etc.)
			if msg.MediaType == "" { // Only store first media item
				data, err := base64.StdEncoding.DecodeString(part.Data)
				if err == nil {
					// Store all media as-is (including HEIC images in original format)
					msg.MediaType = part.ContentType
					msg.MediaData = data
				}
			}
		} else if part.Text != "" && normalizeNullString(part.Text) != "" {
			// This is actual text content (not "null")
			bodyText += part.Text + " "
		}
	}

	if bodyText != "" {
		msg.Body = strings.TrimSpace(bodyText)
	}

	// Extract group name from RCS proto: tr_id if available
	// Use it as the subject if the current subject is empty or starts with "proto:"
	if mms.TrID != "" && strings.HasPrefix(mms.TrID, "proto:") {
		groupName := extractGroupNameFromTrID(mms.TrID)
		if groupName != "" {
			// Only use the extracted name if subject is empty or also starts with "proto:"
			if msg.Subject == "" || strings.HasPrefix(mms.Subject, "proto:") {
				msg.Subject = groupName
			}
		}
	}

	return msg, nil
}

// normalizeNullString converts the string "null" to an empty string
func normalizeNullString(s string) string {
	if strings.TrimSpace(strings.ToLower(s)) == "null" {
		return ""
	}
	return s
}

// isTextContentType checks if a content type is text-based
func isTextContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(ct, "text/") ||
		ct == "application/xml" ||
		ct == "application/json"
}

// isSMILContentType checks if a content type is SMIL markup
func isSMILContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	return ct == "application/smil" ||
		strings.HasPrefix(ct, "application/smil+") ||
		strings.Contains(ct, "smil")
}

// isSMILMarkup checks if the body text is SMIL (Synchronized Multimedia Integration Language) markup
// which is MMS presentation metadata and should not be displayed to users
func isSMILMarkup(body string) bool {
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "<smil") || strings.HasPrefix(trimmed, "<?xml")
}

// isVCardContentType checks if a content type is vCard format
func isVCardContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	return ct == "text/vcard" || ct == "text/x-vcard" || ct == "text/directory"
}

// extractGroupNameFromTrID extracts the group conversation name from RCS proto: tr_id field
func extractGroupNameFromTrID(trID string) string {
	return ""
	/*
		// Check if tr_id starts with "proto:"
		if !strings.HasPrefix(trID, "proto:") {
			return ""
		}

		// Remove the "proto:" prefix
		protoData := strings.TrimPrefix(trID, "proto:")

		// Base64 decode the remaining bytes
		decoded, err := base64.StdEncoding.DecodeString(protoData)
		if err != nil {
			slog.Error("Failed to base64 decode tr_id", "error", err)
			return ""
		}

		// Check if we have enough bytes (need at least 84 bytes: offset 83 + 1 for length)
		if len(decoded) < 84 {
			slog.Debug("Decoded tr_id too short", "bytes", len(decoded), "required", 84)
			return ""
		}

		// Read the length byte at offset 83
		nameLength := int(decoded[83])

		// Check if we have enough bytes for the name
		if len(decoded) < 84+nameLength {
			slog.Debug("Not enough bytes for group name", "have", len(decoded), "need", 84+nameLength)
			return ""
		}

		// Extract the group name string
		groupName := string(decoded[84 : 84+nameLength])

		slog.Debug("Extracted group name from tr_id", "group_name", groupName)
		return groupName
	*/
}

// isHEICContentType checks if a content type is HEIC/HEIF format
func isHEICContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	return strings.Contains(ct, "heic") || strings.Contains(ct, "heif")
}

// needsVideoConversion checks if a video format needs conversion for browser compatibility
func needsVideoConversion(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	unsupportedFormats := []string{
		"3gpp", "3gp", "3g2", "3gpp2",
		"video/3gpp", "video/3gp", "video/3gpp2", "video/3g2",
		"video/x-matroska", // MKV container (may have various codecs)
	}

	for _, format := range unsupportedFormats {
		if strings.Contains(ct, format) {
			return true
		}
	}
	return false
}

// convertHEICtoJPEG is implemented in heic_enabled.go (with -tags heic) or heic_disabled.go (default)
// When HEIC support is enabled, it converts HEIC image data to JPEG format
// When HEIC support is disabled, it returns a placeholder image

// convertVideoToMP4 converts unsupported video formats (like 3GP) to MP4 using ffmpeg
// Returns the converted MP4 data or an error if conversion fails
func convertVideoToMP4(videoData []byte) ([]byte, error) {
	// Create temporary files for input and output
	tmpInputFile, err := os.CreateTemp("", "video-input-*.3gp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}
	defer os.Remove(tmpInputFile.Name())
	defer tmpInputFile.Close()

	tmpOutputFile, err := os.CreateTemp("", "video-output-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %w", err)
	}
	defer os.Remove(tmpOutputFile.Name())
	tmpOutputFile.Close()

	// Write input video data to temp file
	_, err = tmpInputFile.Write(videoData)
	if err != nil {
		return nil, fmt.Errorf("failed to write input video: %w", err)
	}
	tmpInputFile.Close()

	// Run ffmpeg to convert video to MP4 with H.264 codec
	// -i: input file
	// -c:v libx264: use H.264 video codec
	// -c:a aac: use AAC audio codec
	// -movflags +faststart: optimize for streaming
	// -preset fast: balance between speed and quality
	// -crf 23: constant rate factor (quality, lower is better, 23 is good default)
	cmd := exec.Command("ffmpeg",
		"-i", tmpInputFile.Name(),
		"-c:v", "libx264",
		"-c:a", "aac",
		"-movflags", "+faststart",
		"-preset", "fast",
		"-crf", "23",
		"-y", // overwrite output file
		tmpOutputFile.Name(),
	)

	// Capture stderr for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w, stderr: %s", err, stderr.String())
	}

	// Read converted video data
	convertedData, err := os.ReadFile(tmpOutputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read converted video: %w", err)
	}

	return convertedData, nil
}

func convertCallEntry(call CallEntry) (CallLog, error) {
	dateMs, err := strconv.ParseInt(call.Date, 10, 64)
	if err != nil {
		return CallLog{}, err
	}

	duration, _ := strconv.Atoi(call.Duration)
	callType, _ := strconv.Atoi(call.Type)
	presentation, _ := strconv.Atoi(call.Presentation)

	// Normalize the phone number to remove formatting differences
	normalizedNumber := normalizePhoneNumber(call.Number)

	return CallLog{
		Number:         normalizedNumber,
		Duration:       duration,
		Date:           time.Unix(dateMs/1000, 0),
		Type:           callType,
		Presentation:   presentation,
		SubscriptionID: call.SubscriptionID,
		ContactName:    call.ContactName,
	}, nil
}

// UploadProgress tracks the progress of an ongoing upload
type UploadProgress struct {
	TotalMessages     int       `json:"total_messages"`
	ProcessedMessages int       `json:"processed_messages"`
	TotalCalls        int       `json:"total_calls"`
	ProcessedCalls    int       `json:"processed_calls"`
	Status            string    `json:"status"` // "parsing", "importing", "completed", "error"
	ErrorMessage      string    `json:"error_message,omitempty"`
	StartTime         time.Time `json:"start_time"`
	mu                sync.RWMutex
}

var (
	uploadProgress     *UploadProgress
	uploadProgressLock sync.RWMutex
)

// GetUploadProgress returns the current upload progress
func GetUploadProgress() *UploadProgress {
	uploadProgressLock.RLock()
	defer uploadProgressLock.RUnlock()

	if uploadProgress == nil {
		return nil
	}

	uploadProgress.mu.RLock()
	defer uploadProgress.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &UploadProgress{
		TotalMessages:     uploadProgress.TotalMessages,
		ProcessedMessages: uploadProgress.ProcessedMessages,
		TotalCalls:        uploadProgress.TotalCalls,
		ProcessedCalls:    uploadProgress.ProcessedCalls,
		Status:            uploadProgress.Status,
		ErrorMessage:      uploadProgress.ErrorMessage,
		StartTime:         uploadProgress.StartTime,
	}
}

// SetUploadProgress initializes or updates the upload progress
func SetUploadProgress(total, processed int, status string) {
	uploadProgressLock.Lock()
	defer uploadProgressLock.Unlock()

	if uploadProgress == nil {
		uploadProgress = &UploadProgress{
			StartTime: time.Now(),
		}
	}

	uploadProgress.mu.Lock()
	defer uploadProgress.mu.Unlock()

	uploadProgress.TotalMessages = total
	uploadProgress.ProcessedMessages = processed
	uploadProgress.Status = status
}

// UpdateMessageProgress updates the progress for messages
func UpdateMessageProgress(processed int) {
	uploadProgressLock.RLock()
	defer uploadProgressLock.RUnlock()

	if uploadProgress == nil {
		return
	}

	uploadProgress.mu.Lock()
	defer uploadProgress.mu.Unlock()

	uploadProgress.ProcessedMessages = processed
}

// UpdateCallProgress updates the progress for calls
func UpdateCallProgress(processed int) {
	uploadProgressLock.RLock()
	defer uploadProgressLock.RUnlock()

	if uploadProgress == nil {
		return
	}

	uploadProgress.mu.Lock()
	defer uploadProgress.mu.Unlock()

	uploadProgress.ProcessedCalls = processed
}

// ClearUploadProgress clears the upload progress
func ClearUploadProgress() {
	uploadProgressLock.Lock()
	defer uploadProgressLock.Unlock()
	uploadProgress = nil
}

// SaveUploadedFile saves the uploaded file to a temporary location
func SaveUploadedFile(file io.Reader, filename string) (string, error) {
	// Create temp directory if it doesn't exist
	tempDir := os.TempDir()
	uploadDir := filepath.Join(tempDir, "sbv-uploads")
	err := os.MkdirAll(uploadDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create upload directory: %v", err)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp(uploadDir, "backup-*.xml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	// Copy uploaded file to temp file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	return tempFile.Name(), nil
}

// ProcessUploadedFile processes the uploaded file in the background
func ProcessUploadedFile(userID string, username string, filePath string) {
	defer func() {
		// Always clean up the temp file when done
		slog.Info("Removing temporary file", "path", filePath)
		if err := os.Remove(filePath); err != nil {
			slog.Warn("Failed to remove temp file", "path", filePath, "error", err)
		}
	}()

	slog.Info("Starting background processing", "path", filePath, "user", username)

	// Get user database
	userDB, err := GetUserDB(userID, username)
	if err != nil {
		slog.Error("Error getting user database", "error", err)
		SetUploadProgress(0, 0, "error")
		uploadProgressLock.Lock()
		if uploadProgress != nil {
			uploadProgress.mu.Lock()
			uploadProgress.ErrorMessage = fmt.Sprintf("Failed to get user database: %v", err)
			uploadProgress.mu.Unlock()
		}
		uploadProgressLock.Unlock()
		return
	}

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("Error opening file", "error", err)
		SetUploadProgress(0, 0, "error")
		uploadProgressLock.Lock()
		if uploadProgress != nil {
			uploadProgress.mu.Lock()
			uploadProgress.ErrorMessage = fmt.Sprintf("Failed to open file: %v", err)
			uploadProgress.mu.Unlock()
		}
		uploadProgressLock.Unlock()
		return
	}
	defer file.Close()

	// Process with streaming parser (batch size 1 for minimal memory)
	messageCount, callCount, err := ParseSMSBackupStreaming(userDB, file, 1) // Insert immediately, no batching
	if err != nil {
		slog.Error("Error processing file", "error", err)
		SetUploadProgress(0, 0, "error")
		uploadProgressLock.Lock()
		if uploadProgress != nil {
			uploadProgress.mu.Lock()
			uploadProgress.ErrorMessage = fmt.Sprintf("Failed to process file: %v", err)
			uploadProgress.mu.Unlock()
		}
		uploadProgressLock.Unlock()
		return
	}

	slog.Info("Completed processing", "messages", messageCount, "calls", callCount)
}

// ParseSMSBackupStreaming parses SMS backup file with streaming to reduce memory usage
// Each message is inserted immediately and memory is freed aggressively
func ParseSMSBackupStreaming(userDB *sql.DB, r io.Reader, batchSize int) (int, int, error) {
	// Initialize progress tracking
	uploadProgressLock.Lock()
	uploadProgress = &UploadProgress{
		Status:    "parsing",
		StartTime: time.Now(),
	}
	uploadProgressLock.Unlock()

	decoder := xml.NewDecoder(r)

	var messageCount, callCount int

	// Track total count from root element if available
	var totalCount int

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			SetUploadProgress(0, 0, "error")
			return messageCount, callCount, err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Get total count from root element
			if elem.Name.Local == "smses" {
				for _, attr := range elem.Attr {
					if attr.Name.Local == "count" {
						totalCount, _ = strconv.Atoi(attr.Value)
						uploadProgressLock.Lock()
						uploadProgress.mu.Lock()
						uploadProgress.TotalMessages = totalCount
						uploadProgress.mu.Unlock()
						uploadProgressLock.Unlock()
					}
				}
			}

			// Process SMS messages
			if elem.Name.Local == "sms" {
				var sms SMSEntry
				err := decoder.DecodeElement(&sms, &elem)
				if err != nil {
					slog.Error("Error decoding SMS", "error", err)
					continue
				}

				msg, err := convertSMSEntry(sms)
				if err != nil {
					slog.Error("Error converting SMS", "error", err)
					continue
				}

				// Insert immediately - no batching
				err = InsertMessage(userDB, &msg)
				if err != nil {
					slog.Error("Error inserting message", "error", err)
				} else {
					messageCount++
					UpdateMessageProgress(messageCount)
				}

				// Force garbage collection every 1000 messages to keep memory low
				if messageCount%1000 == 0 {
					runtime.GC()
				}
			}

			// Process MMS messages
			if elem.Name.Local == "mms" {
				var mms MMSEntry
				err := decoder.DecodeElement(&mms, &elem)
				if err != nil {
					slog.Error("Error decoding MMS", "error", err)
					continue
				}

				msg, err := convertMMSEntry(mms)

				// Clear the MMS struct immediately after conversion to free base64 strings
				mms.Parts = nil
				mms = MMSEntry{}

				if err != nil {
					slog.Error("Error converting MMS", "error", err)
					continue
				}

				// Insert immediately - no batching
				err = InsertMessage(userDB, &msg)
				if err != nil {
					slog.Error("Error inserting message", "error", err)
				} else {
					messageCount++
					UpdateMessageProgress(messageCount)
				}

				// Clear the message data immediately after insert
				msg.MediaData = nil
				msg = Message{}

				// Force garbage collection every 100 MMS messages (they're larger)
				if messageCount%100 == 0 {
					runtime.GC()
				}
			}

			// Process call logs
			if elem.Name.Local == "call" {
				var call CallEntry
				err := decoder.DecodeElement(&call, &elem)
				if err != nil {
					slog.Error("Error decoding call", "error", err)
					continue
				}

				callLog, err := convertCallEntry(call)
				if err != nil {
					slog.Error("Error converting call", "error", err)
					continue
				}

				// Insert immediately - no batching
				err = InsertCallLog(userDB, &callLog)
				if err != nil {
					slog.Error("Error inserting call log", "error", err)
				} else {
					callCount++
					uploadProgressLock.Lock()
					uploadProgress.mu.Lock()
					uploadProgress.TotalCalls++
					uploadProgress.ProcessedCalls = callCount
					uploadProgress.mu.Unlock()
					uploadProgressLock.Unlock()
				}
			}
		}
	}

	// Final garbage collection
	runtime.GC()

	// Mark as completed
	SetUploadProgress(messageCount, messageCount, "completed")

	return messageCount, callCount, nil
}
