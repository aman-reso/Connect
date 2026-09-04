package repository

import (
	"Connect/internal/domain"
	"Connect/internal/dto"
)

type UserRepository interface {
	CreateOrLogin(phone, name string, role domain.UserRole) (*domain.User, string, bool, error)
	GetByToken(token string) (*domain.User, error)
	GetByID(id string) (*domain.User, error)
	ListModels() ([]*domain.User, error)
	ListModelsAdvanced(filter *domain.ModelFilterParams) ([]*domain.ModelItem, int, error)
	ListOnlineUsers() ([]*domain.User, error)
	DeleteMockModels() error
	SetPresence(id string, isOnline, isBusy bool) error
	UpdateUserOnboarding(userID string, p *domain.ModelProfile) error
}

type WalletRepository interface {
	GetWallet(userID string) (*domain.Wallet, error)
	GetTransactions(userID string) ([]*domain.Transaction, error)
	Recharge(userID string, amount float64) (*domain.Wallet, error)
	ProcessCallSettlement(callerID, receiverID, callID string, durationSec int, ratePerMin float64, reason string) (float64, error)
	DeductChatFee(callerID, receiverID string, amount float64) error
	DeductLiveFee(viewerID, hostID string, amount float64, description string) error
}

type CallRepository interface {
	Create(record *domain.CallRecord) error
	Update(record *domain.CallRecord) error
	GetByID(id string) (*domain.CallRecord, error)
	GetUserHistory(userID string) ([]*domain.CallRecord, error)
	UpdateHeartbeat(callID string) error
	RecoverInterruptedCalls() error
}

type RoomRepository interface {
	Create(host *domain.User, title string, ratePerMin float64) (*domain.GroupRoom, error)
	GetByID(roomID string) (*domain.GroupRoom, error)
	ListActive() ([]*domain.GroupRoom, error)
	AddParticipant(roomID string, user *domain.User) (*domain.RoomParticipant, error)
	RemoveParticipant(roomID, userID, reason string) (float64, int, error)
}

type MessageRepository interface {
	Save(msg *domain.EphemeralMessage) error
	GetActive(u1, u2 string) ([]*domain.EphemeralMessage, error)
	GetConversations(userID string) ([]*dto.ConversationDTO, error)
	PurgeExpired() error
}

type PaymentRepository interface {
	CreateOrder(order *domain.PaymentOrder) error
	UpdateOrder(order *domain.PaymentOrder) error
	GetOrderByID(paymentID string) (*domain.PaymentOrder, error)
	GetOrdersByUserID(userID string) ([]*domain.PaymentOrder, error)
	RecordAuditLog(log *domain.PaymentAuditLog) error
	GetAuditLogs(paymentID string) ([]*domain.PaymentAuditLog, error)
}

type ModelOnboardingRepository interface {
	SaveProfile(profile *domain.ModelProfile) error
	UpdateProfile(profile *domain.ModelProfile) error
	GetProfileByUserID(userID string) (*domain.ModelProfile, error)
	ListPendingProfiles() ([]*domain.ModelProfile, error)
	IncrementReportCount(modelID string) (int, error)
	SetSuspension(modelID string, isSuspended bool) error
}

type ReportRepository interface {
	CreateReport(report *domain.ModelReport) error
	UpdateReport(report *domain.ModelReport) error
	GetReportByID(id string) (*domain.ModelReport, error)
	GetReportsForModel(modelID string) ([]*domain.ModelReport, error)
	ListRecentReports() ([]*domain.ModelReport, error)
}

type FavoriteRepository interface {
	ToggleFavorite(userID, modelID string) (bool, error) // Returns new isFavorite state
	IsFavorite(userID, modelID string) (bool, error)
	GetFavoriteModelIDs(userID string) ([]string, error)
	GetFavoriteModels(userID string) ([]*domain.User, error)
}



