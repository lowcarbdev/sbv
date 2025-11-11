package internal


import "time"

type Message struct {
	ID          int64     `json:"id"`
	Address     string    `json:"address"`
	Body        string    `json:"body"`
	Type        int       `json:"type"` // 1 = received, 2 = sent, 3 = draft, 4 = outbox, 5 = failed, 6 = queued
	Date        time.Time `json:"date"`
	Read        bool      `json:"read"`
	ThreadID    int       `json:"thread_id"`
	Subject     string    `json:"subject,omitempty"`
	MediaType   string    `json:"media_type,omitempty"`
	MediaData   []byte    `json:"-"`
	MediaBase64 string    `json:"media_base64,omitempty"`
	// Additional SMS fields
	Protocol      int    `json:"protocol,omitempty"`
	Status        int    `json:"status,omitempty"` // -1 = none, 0 = complete, 32 = pending, 64 = failed
	ServiceCenter string `json:"service_center,omitempty"`
	SubID         int    `json:"sub_id,omitempty"`
	ContactName   string `json:"contact_name,omitempty"`
	Sender        string `json:"sender,omitempty"` // Sender phone number for received messages
	// Additional MMS fields
	ContentType string `json:"content_type,omitempty"` // ct_t field
	ReadReport  int    `json:"read_report,omitempty"`  // rr field
	ReadStatus  int    `json:"read_status,omitempty"`
	MessageID   string `json:"message_id,omitempty"`   // m_id field
	MessageSize int      `json:"message_size,omitempty"` // m_size field
	MessageType int      `json:"message_type,omitempty"` // m_type field
	SimSlot     int      `json:"sim_slot,omitempty"`
	Addresses   []string `json:"addresses,omitempty"` // All phone numbers in conversation (for MMS)
}

type CallLog struct {
	ID             int64     `json:"id"`
	Number         string    `json:"number"`
	Duration       int       `json:"duration"` // in seconds
	Date           time.Time `json:"date"`
	Type           int       `json:"type"`                   // 1 = incoming, 2 = outgoing, 3 = missed, 4 = voicemail, 5 = rejected, 6 = refused
	Presentation   int       `json:"presentation,omitempty"` // 1 = allowed, 2 = restricted, 3 = unknown, 4 = payphone
	SubscriptionID string    `json:"subscription_id,omitempty"`
	ContactName    string    `json:"contact_name,omitempty"`
}

type Conversation struct {
	Address      string    `json:"address"`
	ContactName  string    `json:"contact_name,omitempty"`
	Subject      string    `json:"subject,omitempty"`
	LastMessage  string    `json:"last_message"`
	LastDate     time.Time `json:"last_date"`
	MessageCount int       `json:"message_count"`
	Type         string    `json:"type"` // "sms", "mms", or "call"
}

type ActivityItem struct {
	Type        string    `json:"type"` // "message" or "call"
	Date        time.Time `json:"date"`
	Address     string    `json:"address"`
	ContactName string    `json:"contact_name,omitempty"`
	// Message-specific fields
	Message *Message `json:"message,omitempty"`
	// Call-specific fields
	Call *CallLog `json:"call,omitempty"`
}

type UploadResponse struct {
	Success      bool   `json:"success"`
	MessageCount int    `json:"message_count"`
	CallLogCount int    `json:"call_log_count"`
	Processing   bool   `json:"processing,omitempty"`
	Error        string `json:"error,omitempty"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never send password hash to client
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success  bool    `json:"success"`
	User     *User   `json:"user,omitempty"`
	Session  *Session `json:"session,omitempty"`
	Error    string  `json:"error,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}
