package dto

import (
	"time"

	"Connect/internal/domain"
)

// RegisterRequest Auth DTOs
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

// PaginationMeta Pagination Metadata
type PaginationMeta struct {
	CurrentPage int  `json:"current_page"`
	Limit       int  `json:"limit"`
	TotalCount  int  `json:"total_count"`
	TotalPages  int  `json:"total_pages"`
	HasNext     bool `json:"has_next"`
	HasPrev     bool `json:"has_prev"`
}

// Model Card DTO for search, nearby discovery & profile viewing
type ModelCardDTO struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	DisplayName        string    `json:"display_name"`
	Role               string    `json:"role"`
	AvatarURL          string    `json:"avatar_url"`
	GalleryURLs        []string  `json:"gallery_urls,omitempty"`
	Bio                string    `json:"bio"`
	Age                int       `json:"age"`
	Gender             string    `json:"gender"`
	City               string    `json:"city,omitempty"`
	State              string    `json:"state,omitempty"`
	Country            string    `json:"country,omitempty"`
	Latitude           float64   `json:"latitude,omitempty"`
	Longitude          float64   `json:"longitude,omitempty"`
	DistanceKM         *float64  `json:"distance_km,omitempty"` // Distance calculated when user lat/lng is supplied
	Languages          []string  `json:"languages"`
	Interests          []string  `json:"interests"`
	VoiceRatePerMin    float64   `json:"voice_rate_per_min"`
	VideoRatePerMin    float64   `json:"video_rate_per_min"`
	GroupRatePerMin    float64   `json:"group_rate_per_min"`
	ChatRatePerMsg     float64   `json:"chat_rate_per_msg"`
	IsOnline           bool      `json:"is_online"`
	IsBusy             bool      `json:"is_busy"`
	Rating             float64   `json:"rating"`
	ReviewCount        int       `json:"review_count"`
	TotalCallsCount    int       `json:"total_calls_count"`
	TotalMinutesSpoken int       `json:"total_minutes_spoken"`
	Badges             []string  `json:"badges"`
	IsNew              bool      `json:"is_new"`
	AudioIntroURL      string    `json:"audio_intro_url,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// Model Filter Query DTO
type ModelFilterQuery struct {
	Filter        string  `json:"filter"` // "all", "nearby", "new", "top", "online"
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	MaxDistanceKM float64 `json:"max_distance_km"`
	City          string  `json:"city"`
	State         string  `json:"state"`
	MinAge        int     `json:"min_age"`
	MaxAge        int     `json:"max_age"`
	Gender        string  `json:"gender"`
	Language      string  `json:"language"`
	Interest      string  `json:"interest"`
	MinRate       float64 `json:"min_rate"`
	MaxRate       float64 `json:"max_rate"`
	IsOnline      *bool   `json:"is_online"`
	SortBy        string  `json:"sort_by"` // "distance", "rating", "newest", "calls", "price_low", "price_high", "popularity"
	Page          int     `json:"page"`
	Limit         int     `json:"limit"`
}

// Model List Response DTO (Paginated & Filter-Aware)
type ModelListResponse struct {
	Count          int                    `json:"count"`
	Pagination     PaginationMeta         `json:"pagination"`
	FiltersApplied map[string]interface{} `json:"filters_applied"`
	Models         []*ModelCardDTO        `json:"models"`
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
	Currency    string  `json:"currency,omitempty"`     // Default "INR"
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

// Complete Model Onboarding Request DTO (Backend Specification)
type ModelOnboardingRequest struct {
	FullLegalName            string   `json:"full_legal_name,omitempty"`
	DisplayName              string   `json:"display_name"`
	Bio                      string   `json:"bio"`
	AvatarURL                string   `json:"avatar_url"`
	GalleryURLs              []string `json:"gallery_urls,omitempty"`
	DateOfBirth              string   `json:"date_of_birth,omitempty"` // YYYY-MM-DD
	Age                      int      `json:"age"`                     // Must be >= 18
	Gender                   string   `json:"gender"`
	GovtIDType               string   `json:"govt_id_type,omitempty"` // "aadhaar", "pan", "passport", "voter_id", "driving_license"
	GovtIDNumber             string   `json:"govt_id_number,omitempty"`
	GovtIDDocURL             string   `json:"govt_id_doc_url,omitempty"`
	SelfieVerificationURL    string   `json:"selfie_verification_url,omitempty"`
	City                     string   `json:"city,omitempty"`
	State                    string   `json:"state,omitempty"`
	Country                  string   `json:"country,omitempty"`
	Pincode                  string   `json:"pincode,omitempty"`
	AddressLine              string   `json:"address_line,omitempty"`
	Latitude                 float64  `json:"latitude,omitempty"`
	Longitude                float64  `json:"longitude,omitempty"`
	Languages                string   `json:"languages"` // e.g. "English, Hindi, Punjabi"
	Interests                string   `json:"interests"` // e.g. "Music, Astrology, Tech"
	VoiceRatePerMin          float64  `json:"voice_rate_per_min"`
	VideoRatePerMin          float64  `json:"video_rate_per_min"`
	GroupRatePerMin          float64  `json:"group_rate_per_min"`
	ChatRatePerMsg           float64  `json:"chat_rate_per_msg"`
	PayoutMethod             string   `json:"payout_method,omitempty"` // "upi" or "bank_transfer"
	PayoutUPI                string   `json:"payout_upi,omitempty"`
	PayoutBankAcc            string   `json:"payout_bank_acc,omitempty"`
	PayoutIFSC               string   `json:"payout_ifsc,omitempty"`
	PayoutBeneficiaryName    string   `json:"payout_beneficiary_name,omitempty"`
	PANNumber                string   `json:"pan_number,omitempty"` // For TDS compliance
	AudioIntroURL            string   `json:"audio_intro_url,omitempty"`
	AgreedToSafetyGuidelines bool     `json:"agreed_to_safety_guidelines"`
	AgreedToTerms            bool     `json:"agreed_to_terms"`
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
