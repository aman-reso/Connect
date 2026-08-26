package usecase

import (
	"errors"
	"fmt"
	"time"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository"
	"github.com/google/uuid"
)

// 1. Auth UseCase
type AuthUseCase struct {
	userRepo   repository.UserRepository
	walletRepo repository.WalletRepository
	mapper     *mapper.Mapper
}

func NewAuthUseCase(uRepo repository.UserRepository, wRepo repository.WalletRepository, m *mapper.Mapper) *AuthUseCase {
	return &AuthUseCase{userRepo: uRepo, walletRepo: wRepo, mapper: m}
}

func (uc *AuthUseCase) RegisterOrLogin(req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	if req.Phone == "" {
		return nil, errors.New("phone number is required")
	}
	if req.Name == "" {
		req.Name = "User_" + req.Phone[len(req.Phone)-4:]
	}
	if req.Role == "" {
		req.Role = domain.RoleUser
	}

	user, token, isNew, err := uc.userRepo.CreateOrLogin(req.Phone, req.Name, req.Role)
	if err != nil {
		return nil, err
	}

	wallet, _ := uc.walletRepo.GetWallet(user.ID)
	return uc.mapper.ToAuthResponse(user, token, isNew, wallet), nil
}

func (uc *AuthUseCase) ValidateToken(token string) (*domain.User, error) {
	return uc.userRepo.GetByToken(token)
}

func (uc *AuthUseCase) ListModels() (*dto.ModelListResponse, error) {
	models, err := uc.userRepo.ListModels()
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToModelListResponse(models), nil
}

func (uc *AuthUseCase) ListModelsAdvanced(filter *dto.ModelFilterQuery) (*dto.ModelListResponse, error) {
	if filter == nil {
		filter = &dto.ModelFilterQuery{Filter: "all", Page: 1, Limit: 20}
	}
	domainFilter := &domain.ModelFilterParams{
		Filter:        filter.Filter,
		Latitude:      filter.Lat,
		Longitude:     filter.Lng,
		MaxDistanceKM: filter.MaxDistanceKM,
		City:          filter.City,
		State:         filter.State,
		MinAge:        filter.MinAge,
		MaxAge:        filter.MaxAge,
		Gender:        filter.Gender,
		Language:      filter.Language,
		Interest:      filter.Interest,
		MinRate:       filter.MinRate,
		MaxRate:       filter.MaxRate,
		IsOnline:      filter.IsOnline,
		SortBy:        filter.SortBy,
		Page:          filter.Page,
		Limit:         filter.Limit,
	}

	items, totalCount, err := uc.userRepo.ListModelsAdvanced(domainFilter)
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToPaginatedModelListResponse(items, totalCount, filter), nil
}

// 2. Wallet UseCase
type WalletUseCase struct {
	walletRepo repository.WalletRepository
	mapper     *mapper.Mapper
}

func NewWalletUseCase(wRepo repository.WalletRepository, m *mapper.Mapper) *WalletUseCase {
	return &WalletUseCase{walletRepo: wRepo, mapper: m}
}

func (uc *WalletUseCase) GetWallet(userID string) (*dto.WalletResponse, error) {
	wallet, err := uc.walletRepo.GetWallet(userID)
	if err != nil {
		return nil, err
	}
	txs, _ := uc.walletRepo.GetTransactions(userID)
	return uc.mapper.ToWalletResponse(wallet, txs), nil
}

func (uc *WalletUseCase) Recharge(userID string, amount float64) (*dto.WalletResponse, error) {
	if amount <= 0 {
		return nil, errors.New("invalid recharge amount")
	}
	wallet, err := uc.walletRepo.Recharge(userID, amount)
	if err != nil {
		return nil, err
	}
	txs, _ := uc.walletRepo.GetTransactions(userID)
	return uc.mapper.ToWalletResponse(wallet, txs), nil
}

// 3. Call UseCase
type CallUseCase struct {
	callRepo   repository.CallRepository
	userRepo   repository.UserRepository
	walletRepo repository.WalletRepository
	mapper     *mapper.Mapper
}

func NewCallUseCase(cRepo repository.CallRepository, uRepo repository.UserRepository, wRepo repository.WalletRepository, m *mapper.Mapper) *CallUseCase {
	return &CallUseCase{callRepo: cRepo, userRepo: uRepo, walletRepo: wRepo, mapper: m}
}

func (uc *CallUseCase) InitiateCall(caller *domain.User, receiverID string, callTypeOpt ...string) (*domain.CallRecord, error) {
	receiver, err := uc.userRepo.GetByID(receiverID)
	if err != nil {
		return nil, fmt.Errorf("host not found")
	}

	callType := "voice"
	if len(callTypeOpt) > 0 && callTypeOpt[0] != "" {
		callType = callTypeOpt[0]
	}

	rate := receiver.VoiceRatePerMin
	// Balance Check
	wallet, err := uc.walletRepo.GetWallet(caller.ID)
	if err != nil || wallet.Balance < rate {
		return nil, fmt.Errorf("insufficient balance: minimum ₹%.2f required for 1 min call", rate)
	}

	if receiver.IsBusy {
		return nil, fmt.Errorf("%s is currently busy on another call", receiver.Name)
	}

	callID := "call_" + uuid.New().String()[:8]
	record := &domain.CallRecord{
		ID:           callID,
		CallerID:     caller.ID,
		CallerName:   caller.Name,
		ReceiverID:   receiver.ID,
		ReceiverName: receiver.Name,
		CallType:     callType,
		Status:       domain.CallStatusRinging,
		RatePerMin:   rate,
		CreatedAt:    time.Now(),
	}

	if err := uc.callRepo.Create(record); err != nil {
		return nil, err
	}

	_ = uc.userRepo.SetPresence(receiver.ID, true, true)
	return record, nil
}

func (uc *CallUseCase) AcceptCall(callID string) (*domain.CallRecord, error) {
	record, err := uc.callRepo.GetByID(callID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	record.Status = domain.CallStatusActive
	record.StartedAt = &now
	record.LastHeartbeat = &now

	if err := uc.callRepo.Update(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (uc *CallUseCase) RejectCall(callID string) error {
	record, err := uc.callRepo.GetByID(callID)
	if err != nil {
		return err
	}

	record.Status = domain.CallStatusRejected
	record.EndReason = "Model declined the call"
	_ = uc.callRepo.Update(record)
	_ = uc.userRepo.SetPresence(record.ReceiverID, true, false)
	_ = uc.userRepo.SetPresence(record.CallerID, true, false)
	return nil
}

func (uc *CallUseCase) EndCall(callID string, reason string) (float64, int, error) {
	record, err := uc.callRepo.GetByID(callID)
	if err != nil {
		return 0, 0, err
	}

	now := time.Now()
	record.EndedAt = &now
	durationSec := 0
	if record.StartedAt != nil {
		durationSec = int(now.Sub(*record.StartedAt).Seconds())
	}
	record.DurationSeconds = durationSec
	record.EndReason = reason

	if reason == "Balance exhausted" {
		record.Status = domain.CallStatusExhausted
	} else if record.Status == domain.CallStatusActive {
		record.Status = domain.CallStatusCompleted
	}

	cost, err := uc.walletRepo.ProcessCallSettlement(record.CallerID, record.ReceiverID, callID, durationSec, record.RatePerMin, reason)
	record.TotalCost = cost

	_ = uc.callRepo.Update(record)
	_ = uc.userRepo.SetPresence(record.ReceiverID, true, false)
	_ = uc.userRepo.SetPresence(record.CallerID, true, false)
	return cost, durationSec, err
}

func (uc *CallUseCase) GetHistory(userID string) (*dto.CallHistoryResponse, error) {
	calls, err := uc.callRepo.GetUserHistory(userID)
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToCallHistoryResponse(calls), nil
}

// 4. Room UseCase (Group Audio Lounge)
type RoomUseCase struct {
	roomRepo repository.RoomRepository
	mapper   *mapper.Mapper
}

func NewRoomUseCase(rRepo repository.RoomRepository, m *mapper.Mapper) *RoomUseCase {
	return &RoomUseCase{roomRepo: rRepo, mapper: m}
}

func (uc *RoomUseCase) CreateRoom(host *domain.User, title string, ratePerMin float64) (*domain.GroupRoom, error) {
	if host.Role != domain.RoleModel {
		return nil, errors.New("only hosts/models can start a group audio lounge")
	}
	return uc.roomRepo.Create(host, title, ratePerMin)
}

func (uc *RoomUseCase) ListRooms() (*dto.RoomListResponse, error) {
	rooms, err := uc.roomRepo.ListActive()
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToRoomListResponse(rooms), nil
}

func (uc *RoomUseCase) JoinRoom(roomID string, user *domain.User) (*domain.RoomParticipant, error) {
	return uc.roomRepo.AddParticipant(roomID, user)
}

func (uc *RoomUseCase) LeaveRoom(roomID, userID, reason string) (float64, int, error) {
	return uc.roomRepo.RemoveParticipant(roomID, userID, reason)
}
