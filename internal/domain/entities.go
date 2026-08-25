package domain

import (
	"time"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleModel UserRole = "model"
)

type CallStatus string

const (
	CallStatusInitiated  CallStatus = "initiated"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusActive     CallStatus = "active"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusRejected   CallStatus = "rejected"
	CallStatusBusy       CallStatus = "busy"
	CallStatusExhausted  CallStatus = "balance_exhausted"
	CallStatusMissed     CallStatus = "missed"
	CallStatusCancelled  CallStatus = "cancelled"
)

// User Entity
type User struct {
	ID              string    `json:"id"`
	Phone           string    `json:"phone"`
	Name            string    `json:"name"`
	Role            UserRole  `json:"role"` // "user" or "model"
	AvatarURL       string    `json:"avatar_url"`
	Bio             string    `json:"bio,omitempty"`
	VoiceRatePerMin float64   `json:"voice_rate_per_min"`
	GroupRatePerMin float64   `json:"group_rate_per_min"`
	ChatRatePerMsg  float64   `json:"chat_rate_per_msg"`
	IsOnline        bool      `json:"is_online"`
	IsBusy          bool      `json:"is_busy"`
	ActiveToken     string    `json:"active_token,omitempty"` // Single-Device Session Enforcement
	DeviceID        string    `json:"device_id,omitempty"`
	ActiveRoomID    string    `json:"active_room_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// Wallet Entity
type Wallet struct {
	UserID      string    `json:"user_id"`
	Balance     float64   `json:"balance"` // In Rupees
	BonusGiven  float64   `json:"bonus_given"`
	TotalSpent  float64   `json:"total_spent"`
	TotalEarned float64   `json:"total_earned"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TransactionType string

const (
	TxTypeWelcomeBonus TransactionType = "welcome_bonus"
	TxTypeRecharge     TransactionType = "recharge"
	TxTypeCallDebit    TransactionType = "call_debit"
	TxTypeCallCredit   TransactionType = "call_credit"
	TxTypeGroupDebit   TransactionType = "group_call_debit"
	TxTypeGroupCredit  TransactionType = "group_call_credit"
	TxTypeChatDebit    TransactionType = "chat_debit"
	TxTypeChatCredit   TransactionType = "chat_credit"
)

// Transaction Entity (Financial Ledger)
type Transaction struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Amount      float64         `json:"amount"`
	Type        TransactionType `json:"type"`
	Description string          `json:"description"`
	CallID      string          `json:"call_id,omitempty"`
	RoomID      string          `json:"room_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// CallRecord Entity (METADATA ONLY - ZERO AUDIO RECORDED/STORED)
type CallRecord struct {
	ID              string     `json:"id"`
	CallerID        string     `json:"caller_id"`
	CallerName      string     `json:"caller_name"`
	ReceiverID      string     `json:"receiver_id"`
	ReceiverName    string     `json:"receiver_name"`
	CallType        string     `json:"call_type"` // "voice", "group_voice", "video"
	Status          CallStatus `json:"status"`
	RatePerMin      float64    `json:"rate_per_min"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	LastHeartbeat   *time.Time `json:"last_heartbeat,omitempty"`
	DurationSeconds int        `json:"duration_seconds"`
	TotalCost       float64    `json:"total_cost"`
	EndReason       string     `json:"end_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

// GroupRoom Entity (Multi-User Audio Lounge)
type GroupRoom struct {
	ID              string                      `json:"id"`
	HostID          string                      `json:"host_id"`
	HostName        string                      `json:"host_name"`
	HostAvatar      string                      `json:"host_avatar"`
	Title           string                      `json:"title"`
	RatePerMin      float64                     `json:"rate_per_min"`
	MaxParticipants int                         `json:"max_participants"`
	IsActive        bool                        `json:"is_active"`
	Participants    map[string]*RoomParticipant `json:"participants"`
	CreatedAt       time.Time                   `json:"created_at"`
}

// RoomParticipant Entity
type RoomParticipant struct {
	UserID          string     `json:"user_id"`
	Name            string     `json:"name"`
	AvatarURL       string     `json:"avatar_url"`
	JoinedAt        time.Time  `json:"joined_at"`
	LeftAt          *time.Time `json:"left_at,omitempty"`
	DurationSeconds int        `json:"duration_seconds"`
	TotalCost       float64    `json:"total_cost"`
	IsMuted         bool       `json:"is_muted"`
	IsHost          bool       `json:"is_host"`
}

// EphemeralMessage Entity (Self-destructing text)
type EphemeralMessage struct {
	ID         string    `json:"id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id,omitempty"`
	RoomID     string    `json:"room_id,omitempty"`
	Content    string    `json:"content"`
	Cost       float64   `json:"cost"`
	ExpiresAt  time.Time `json:"expires_at"`
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}

// Payment States
type PaymentStatus string

const (
	PaymentStatusInitiated        PaymentStatus = "initiated"
	PaymentStatusInProgress       PaymentStatus = "inprogress"
	PaymentStatusSuccessful       PaymentStatus = "successful"
	PaymentStatusFailed           PaymentStatus = "failed"
	PaymentStatusRefundInProgress PaymentStatus = "refund_inprogress"
	PaymentStatusRefundDone       PaymentStatus = "refund_done"
)

// PaymentOrder Entity (Stateful Order with retry parent linkage)
type PaymentOrder struct {
	ID                 string        `json:"id"`
	OriginalPaymentID  string        `json:"original_payment_id,omitempty"` // Parent order ID if retried
	UserID             string        `json:"user_id"`
	Amount             float64       `json:"amount"` // In Rupees
	Currency           string        `json:"currency"`
	Status             PaymentStatus `json:"status"`
	GatewayName        string        `json:"gateway_name"` // "razorpay", "stripe", "upi"
	GatewayOrderID     string        `json:"gateway_order_id,omitempty"`
	GatewayPaymentID   string        `json:"gateway_payment_id,omitempty"`
	GatewaySignature   string        `json:"gateway_signature,omitempty"`
	FailureReason      string        `json:"failure_reason,omitempty"`
	RefundReason       string        `json:"refund_reason,omitempty"`
	RefundID           string        `json:"refund_id,omitempty"`
	RetryCount         int           `json:"retry_count"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	CompletedAt        *time.Time    `json:"completed_at,omitempty"`
}

// PaymentAuditLog Entity (Chronological step execution tracking)
type PaymentAuditLog struct {
	ID               string        `json:"id"`
	PaymentID        string        `json:"payment_id"`
	FromStatus       PaymentStatus `json:"from_status"`
	ToStatus         PaymentStatus `json:"to_status"`
	EventName        string        `json:"event_name"` // e.g. "ORDER_CREATED", "REDIRECT_GATEWAY", "WEBHOOK_SUCCESS", "REFUND_INITIATED"
	GatewayRefID     string        `json:"gateway_ref_id,omitempty"`
	GatewayCode      string        `json:"gateway_code,omitempty"`
	Message          string        `json:"message"`
	MetadataJSON     string        `json:"metadata_json,omitempty"`
	DurationMS       int64         `json:"duration_ms"`
	StartedAt        time.Time     `json:"started_at"`
	EndedAt          time.Time     `json:"ended_at"`
	CreatedAt        time.Time     `json:"created_at"`
}

// Model Onboarding Status
type OnboardingStatus string

const (
	OnboardingStatusPendingReview OnboardingStatus = "pending_review"
	OnboardingStatusApproved      OnboardingStatus = "approved"
	OnboardingStatusRejected      OnboardingStatus = "rejected"
)

// ModelProfile Entity (KYC, verification, custom rates, payout bank)
type ModelProfile struct {
	ID              string           `json:"id"`
	UserID          string           `json:"user_id"`
	DisplayName     string           `json:"display_name"`
	Bio             string           `json:"bio"`
	AvatarURL       string           `json:"avatar_url"`
	Age             int              `json:"age"` // Must be >= 18
	Gender          string           `json:"gender"`
	Languages       string           `json:"languages"` // e.g. "English, Hindi, Punjabi"
	Interests       string           `json:"interests"` // e.g. "Music, Astrology, Tech"
	VoiceRatePerMin float64          `json:"voice_rate_per_min"`
	GroupRatePerMin float64          `json:"group_rate_per_min"`
	ChatRatePerMsg  float64          `json:"chat_rate_per_msg"`
	PayoutUPI       string           `json:"payout_upi,omitempty"`
	PayoutBankAcc   string           `json:"payout_bank_acc,omitempty"`
	PayoutIFSC      string           `json:"payout_ifsc,omitempty"`
	AudioIntroURL   string           `json:"audio_intro_url,omitempty"`
	Status          OnboardingStatus `json:"status"` // "pending_review", "approved", "rejected"
	RejectionReason string           `json:"rejection_reason,omitempty"`
	ReportCount     int              `json:"report_count"`
	IsSuspended     bool             `json:"is_suspended"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// Report Category & Status
type ReportCategory string

const (
	ReportCategoryHarassment    ReportCategory = "harassment"
	ReportCategoryInappropriate ReportCategory = "inappropriate_behavior"
	ReportCategoryUnderage      ReportCategory = "underage_suspicion"
	ReportCategoryFraud         ReportCategory = "fraud"
	ReportCategoryPoorAudio     ReportCategory = "poor_audio_quality"
	ReportCategoryOther         ReportCategory = "other"
)

type ReportStatus string

const (
	ReportStatusSubmitted     ReportStatus = "submitted"
	ReportStatusInvestigating ReportStatus = "under_investigation"
	ReportStatusResolved      ReportStatus = "resolved"
	ReportStatusDismissed     ReportStatus = "dismissed"
)

// ModelReport Entity
type ModelReport struct {
	ID             string         `json:"id"`
	ReporterID     string         `json:"reporter_id"`
	ReporterName   string         `json:"reporter_name"`
	ModelID        string         `json:"model_id"`
	ModelName      string         `json:"model_name"`
	CallID         string         `json:"call_id,omitempty"`
	RoomID         string         `json:"room_id,omitempty"`
	Category       ReportCategory `json:"category"`
	Description    string         `json:"description"`
	Status         ReportStatus   `json:"status"` // "submitted", "under_investigation", "resolved", "dismissed"
	AdminAction    string         `json:"admin_action,omitempty"` // "warning", "suspension", "ban", "none"
	AdminNote      string         `json:"admin_note,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
}

// UserFavorite represents a user's bookmarked/favorited model
type UserFavorite struct {
	UserID    string    `json:"user_id"`
	ModelID   string    `json:"model_id"`
	CreatedAt time.Time `json:"created_at"`
}

