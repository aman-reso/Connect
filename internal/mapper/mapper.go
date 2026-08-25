package mapper

import (
	"Connect/internal/domain"
	"Connect/internal/dto"
)

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

func (m *Mapper) ToAuthResponse(user *domain.User, token string, isNew bool, wallet *domain.Wallet) *dto.AuthResponse {
	msg := "Login successful"
	if isNew && user.Role == domain.RoleUser {
		msg = "Welcome! ₹50 bonus incentive added to your wallet!"
	}

	return &dto.AuthResponse{
		User:      user,
		Token:     token,
		IsNewUser: isNew,
		Wallet:    wallet,
		Message:   msg,
	}
}

func (m *Mapper) ToModelListResponse(models []*domain.User) *dto.ModelListResponse {
	return &dto.ModelListResponse{
		Count:  len(models),
		Models: models,
	}
}

func (m *Mapper) ToWalletResponse(wallet *domain.Wallet, txs []*domain.Transaction) *dto.WalletResponse {
	return &dto.WalletResponse{
		Wallet:       wallet,
		Transactions: txs,
	}
}

func (m *Mapper) ToRoomListResponse(rooms []*domain.GroupRoom) *dto.RoomListResponse {
	return &dto.RoomListResponse{
		Count: len(rooms),
		Rooms: rooms,
	}
}

func (m *Mapper) ToCallHistoryResponse(calls []*domain.CallRecord) *dto.CallHistoryResponse {
	return &dto.CallHistoryResponse{
		Count:         len(calls),
		Calls:         calls,
		PrivacyNotice: "Calls are End-to-End Encrypted (DTLS-SRTP). No audio recordings or voice files are stored on any servers.",
	}
}

func (m *Mapper) ToPaymentOrderResponse(order *domain.PaymentOrder, msg string) *dto.PaymentOrderResponse {
	return &dto.PaymentOrderResponse{
		Order:   order,
		Message: msg,
	}
}

func (m *Mapper) ToPaymentTimelineResponse(order *domain.PaymentOrder, logs []*domain.PaymentAuditLog) *dto.PaymentTimelineResponse {
	return &dto.PaymentTimelineResponse{
		PaymentID: order.ID,
		Order:     order,
		Logs:      logs,
		Count:     len(logs),
	}
}

func (m *Mapper) ToModelOnboardingResponse(profile *domain.ModelProfile, msg string) *dto.ModelOnboardingResponse {
	return &dto.ModelOnboardingResponse{
		Profile: profile,
		Message: msg,
	}
}

func (m *Mapper) ToReportResponse(report *domain.ModelReport, msg string) *dto.ReportResponse {
	return &dto.ReportResponse{
		Report:  report,
		Message: msg,
	}
}

func (m *Mapper) ToListReportsResponse(reports []*domain.ModelReport) *dto.ListReportsResponse {
	return &dto.ListReportsResponse{
		Count:   len(reports),
		Reports: reports,
	}
}

func (m *Mapper) ToFavoriteModelsResponse(models []*domain.User) *dto.FavoriteModelsResponse {
	return &dto.FavoriteModelsResponse{
		Count:  len(models),
		Models: models,
	}
}

