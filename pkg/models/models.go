package models

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

// User represents a regular client or a model/host
type User struct {
	ID              string    `json:"id"`
	Phone           string    `json:"phone"`
	Name            string    `json:"name"`
	Role            UserRole  `json:"role"`           // "user" or "model"
	AvatarURL       string    `json:"avatar_url"`
	Bio             string    `json:"bio,omitempty"`
	VoiceRatePerMin float64   `json:"voice_rate_per_min"` // 1-on-1 Voice rate
	GroupRatePerMin float64   `json:"group_rate_per_min"` // Group Call rate (usually discounted e.g. ₹5/min)
	ChatRatePerMsg  float64   `json:"chat_rate_per_msg"`  // In Rupees (e.g. 2.0)
	IsOnline        bool      `json:"is_online"`
	IsBusy          bool      `json:"is_busy"`
	ActiveToken     string    `json:"active_token,omitempty"`
	DeviceID        string    `json:"device_id,omitempty"`
	ActiveRoomID    string    `json:"active_room_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// Wallet stores user balance and bonus funds
type Wallet struct {
	UserID      string    `json:"user_id"`
	Balance     float64   `json:"balance"`       // Usable balance in Rupees (includes bonus)
	BonusGiven  float64   `json:"bonus_given"`   // Signup welcome incentive (e.g. 50.0)
	TotalSpent  float64   `json:"total_spent"`
	TotalEarned float64   `json:"total_earned"`  // For models
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

// Transaction is a financial ledger record
type Transaction struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Amount      float64         `json:"amount"` // Positive for credit, negative for debit
	Type        TransactionType `json:"type"`
	Description string          `json:"description"`
	CallID      string          `json:"call_id,omitempty"`
	RoomID      string          `json:"room_id,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// CallRecord stores 1-on-1 METADATA ONLY (Zero voice/audio storage)
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
	DurationSeconds int        `json:"duration_seconds"` // Exact duration
	TotalCost       float64    `json:"total_cost"`       // Total amount billed
	EndReason       string     `json:"end_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

// GroupRoom represents a multi-participant audio lounge hosted by a model
type GroupRoom struct {
	ID              string                      `json:"id"`
	HostID          string                      `json:"host_id"`
	HostName        string                      `json:"host_name"`
	HostAvatar      string                      `json:"host_avatar"`
	Title           string                      `json:"title"`
	RatePerMin      float64                     `json:"rate_per_min"` // Per-user per-minute rate
	MaxParticipants int                         `json:"max_participants"`
	IsActive        bool                        `json:"is_active"`
	Participants    map[string]*RoomParticipant `json:"participants"` // userID -> Participant
	CreatedAt       time.Time                   `json:"created_at"`
}

// RoomParticipant tracks each individual user's join time and independent billing
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

// EphemeralMessage represents a chat message that self-destructs or is forwarded E2EE
type EphemeralMessage struct {
	ID         string    `json:"id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id,omitempty"` // For 1-on-1
	RoomID     string    `json:"room_id,omitempty"`     // For Group Room
	Content    string    `json:"content"`              // Encrypted payload or ephemeral text
	Cost       float64   `json:"cost"`
	ExpiresAt  time.Time `json:"expires_at"`           // Auto-deletion timestamp
	IsRead     bool      `json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserFavorite represents a user's favorited model
type UserFavorite struct {
	UserID    string    `json:"user_id"`
	ModelID   string    `json:"model_id"`
	CreatedAt time.Time `json:"created_at"`
}

