package dto

import (
	"time"

	"Connect/internal/domain"
)

// Auth DTOs
type RegisterRequest struct {
	Phone string          `json:"phone"`
	Name  string          `json:"name"`
	Role  domain.UserRole `json:"role"`
}

type AuthResponse struct {
	User      *domain.User   `json:"user"`
	Token     string         `json:"token"`
	IsNewUser bool           `json:"is_new_user"`
	Wallet    *domain.Wallet `json:"wallet,omitempty"`
	Message   string         `json:"message"`
}

// Model List DTO
type ModelListResponse struct {
	Count  int            `json:"count"`
	Models []*domain.User `json:"models"`
}

// Wallet DTOs
type RechargeRequest struct {
	Amount float64 `json:"amount"`
}

type WalletResponse struct {
	Wallet       *domain.Wallet        `json:"wallet"`
	Transactions []*domain.Transaction `json:"transactions,omitempty"`
}

// Room DTOs
type CreateRoomRequest struct {
	Title      string  `json:"title"`
	RatePerMin float64 `json:"rate_per_min"`
}

type RoomListResponse struct {
	Count int                 `json:"count"`
	Rooms []*domain.GroupRoom `json:"rooms"`
}

// Call History DTO
type CallHistoryResponse struct {
	Count         int                  `json:"count"`
	Calls         []*domain.CallRecord `json:"calls"`
	PrivacyNotice string               `json:"privacy_notice"`
}

// Ephemeral Chat DTO
type EphemeralChatResponse struct {
	PartnerID string                     `json:"partner_id"`
	Messages  []*domain.EphemeralMessage `json:"messages"`
	Notice    string                     `json:"notice"`
}

// Live Call Tick DTO
type CallTickDTO struct {
	CallID       string    `json:"call_id,omitempty"`
	RoomID       string    `json:"room_id,omitempty"`
	DurationSec  int       `json:"duration_sec"`
	RemainingSec int       `json:"remaining_sec"`
	Cost         float64   `json:"cost"`
	Timestamp    time.Time `json:"timestamp"`
}

// Payment System DTOs
type CreatePaymentOrderRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency,omitempty"` // Default "INR"
	GatewayName string  `json:"gateway_name,omitempty"` // Default "razorpay"
}

type ProcessPaymentCallbackRequest struct {
	PaymentID        string               `json:"payment_id"`
	GatewayPaymentID string               `json:"gateway_payment_id,omitempty"`
	GatewayOrderID   string               `json:"gateway_order_id,omitempty"`
	GatewaySignature string               `json:"gateway_signature,omitempty"`
	Status           domain.PaymentStatus `json:"status"` // "successful" or "failed"
	ErrorCode        string               `json:"error_code,omitempty"`
	ErrorMessage     string               `json:"error_message,omitempty"`
}

type RetryPaymentRequest struct {
	FailedPaymentID string `json:"failed_payment_id"`
}

type InitiateRefundRequest struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

type PaymentOrderResponse struct {
	Order   *domain.PaymentOrder `json:"order"`
	Message string               `json:"message"`
}

type PaymentTimelineResponse struct {
	PaymentID string                    `json:"payment_id"`
	Order     *domain.PaymentOrder      `json:"order"`
	Logs      []*domain.PaymentAuditLog `json:"timeline"`
	Count     int                       `json:"step_count"`
}

// Model Onboarding DTOs
type ModelOnboardingRequest struct {
	DisplayName     string  `json:"display_name"`
	Bio             string  `json:"bio"`
	AvatarURL       string  `json:"avatar_url"`
	Age             int     `json:"age"` // Must be >= 18
	Gender          string  `json:"gender"`
	Languages       string  `json:"languages"`
	Interests       string  `json:"interests"`
	VoiceRatePerMin float64 `json:"voice_rate_per_min"`
	GroupRatePerMin float64 `json:"group_rate_per_min"`
	ChatRatePerMsg  float64 `json:"chat_rate_per_msg"`
	PayoutUPI       string  `json:"payout_upi,omitempty"`
	PayoutBankAcc   string  `json:"payout_bank_acc,omitempty"`
	PayoutIFSC      string  `json:"payout_ifsc,omitempty"`
	AudioIntroURL   string  `json:"audio_intro_url,omitempty"`
}

type ModelOnboardingResponse struct {
	Profile *domain.ModelProfile `json:"profile"`
	Message string               `json:"message"`
}

// Report DTOs
type CreateReportRequest struct {
	ModelID     string                `json:"model_id"`
	CallID      string                `json:"call_id,omitempty"`
	RoomID      string                `json:"room_id,omitempty"`
	Category    domain.ReportCategory `json:"category"`
	Description string                `json:"description"`
}

type ReportResponse struct {
	Report  *domain.ModelReport `json:"report"`
	Message string              `json:"message"`
}

type ResolveReportRequest struct {
	ReportID    string `json:"report_id"`
	AdminAction string `json:"admin_action"` // "warning", "suspension", "ban", "none"
	AdminNote   string `json:"admin_note"`
}

type ListReportsResponse struct {
	Count   int                   `json:"count"`
	Reports []*domain.ModelReport `json:"reports"`
}

// Favorite DTOs
type ToggleFavoriteRequest struct {
	ModelID string `json:"model_id"`
}

type ToggleFavoriteResponse struct {
	ModelID    string `json:"model_id"`
	IsFavorite bool   `json:"is_favorite"`
	Message    string `json:"message"`
}

type FavoriteModelsResponse struct {
	Count  int            `json:"count"`
	Models []*domain.User `json:"models"`
}

