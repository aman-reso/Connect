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

type ChatUseCase struct {
	msgRepo    repository.MessageRepository
	userRepo   repository.UserRepository
	walletRepo repository.WalletRepository
	mapper     *mapper.Mapper
}

func NewChatUseCase(
	mRepo repository.MessageRepository,
	uRepo repository.UserRepository,
	wRepo repository.WalletRepository,
	m *mapper.Mapper,
) *ChatUseCase {
	return &ChatUseCase{
		msgRepo:    mRepo,
		userRepo:   uRepo,
		walletRepo: wRepo,
		mapper:     m,
	}
}

func (uc *ChatUseCase) SendMessage(senderID, receiverID, content string) (*domain.EphemeralMessage, error) {
	if senderID == "" || receiverID == "" || content == "" {
		return nil, errors.New("sender, receiver, and content are required")
	}

	sender, err := uc.userRepo.GetByID(senderID)
	if err != nil {
		return nil, fmt.Errorf("sender not found: %w", err)
	}

	cost := 0.0
	if sender.Role == domain.RoleUser {
		cost = 1.0 // Standard chat rate per message
		w, wErr := uc.walletRepo.GetWallet(senderID)
		if wErr == nil && w.Balance >= cost {
			_ = uc.walletRepo.DeductChatFee(senderID, receiverID, cost)
		}
	}

	now := time.Now()
	msg := &domain.EphemeralMessage{
		ID:         fmt.Sprintf("msg_%d_%s", now.Unix(), uuid.New().String()[:6]),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		Cost:       cost,
		ExpiresAt:  now.Add(24 * time.Hour), // 24-Hour Strict Ephemeral Expiration
		IsRead:     false,
		CreatedAt:  now,
	}

	if err := uc.msgRepo.Save(msg); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return msg, nil
}

func (uc *ChatUseCase) GetMessages(userA, userB string) ([]*domain.EphemeralMessage, error) {
	return uc.msgRepo.GetActive(userA, userB)
}

func (uc *ChatUseCase) GetConversations(userID string) ([]*dto.ConversationDTO, error) {
	return uc.msgRepo.GetConversations(userID)
}

func (uc *ChatUseCase) PurgeExpired() (int64, error) {
	err := uc.msgRepo.PurgeExpired()
	return 0, err
}
