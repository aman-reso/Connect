package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"Connect/pkg/models"
	"github.com/google/uuid"
)

const WelcomeBonusAmount = 50.0 // ₹50 Signup Welcome Incentive

type Store struct {
	mu           sync.RWMutex
	users        map[string]*models.User
	tokens       map[string]string // token -> userID
	wallets      map[string]*models.Wallet
	transactions map[string][]*models.Transaction // userID -> transactions
	callRecords  map[string]*models.CallRecord    // callID -> CallRecord
	userCalls    map[string][]string              // userID -> list of callIDs
	rooms        map[string]*models.GroupRoom     // roomID -> GroupRoom
	messages     map[string][]*models.EphemeralMessage // conversationKey -> messages
}

func NewStore() *Store {
	s := &Store{
		users:        make(map[string]*models.User),
		tokens:       make(map[string]string),
		wallets:      make(map[string]*models.Wallet),
		transactions: make(map[string][]*models.Transaction),
		callRecords:  make(map[string]*models.CallRecord),
		userCalls:    make(map[string][]string),
		rooms:        make(map[string]*models.GroupRoom),
		messages:     make(map[string][]*models.EphemeralMessage),
	}
	s.seedDefaultModels()
	go s.startEphemeralCleaner()
	return s
}

// Seed initial model profiles with rates
func (s *Store) seedDefaultModels() {
	seedModels := []*models.User{
		{
			ID:              "model-1",
			Phone:           "9876543210",
			Name:            "Aanya Sharma",
			Role:            models.RoleModel,
			AvatarURL:       "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=400&auto=format&fit=crop&q=80",
			Bio:             "Love deep late-night conversations, music & psychology 🌙",
			VoiceRatePerMin: 10.0, // 1-on-1: ₹10/min
			GroupRatePerMin: 5.0,  // Group Room: ₹5/min
			ChatRatePerMsg:  1.0,  // ₹1/msg
			IsOnline:        true,
			IsBusy:          false,
			CreatedAt:       time.Now(),
		},
		{
			ID:              "model-2",
			Phone:           "9876543211",
			Name:            "Riya Sen",
			Role:            models.RoleModel,
			AvatarURL:       "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=400&auto=format&fit=crop&q=80",
			Bio:             "Artist & traveler. Let's talk about dreams & coffee ☕✨",
			VoiceRatePerMin: 15.0, // 1-on-1: ₹15/min
			GroupRatePerMin: 7.0,  // Group Room: ₹7/min
			ChatRatePerMsg:  2.0,  // ₹2/msg
			IsOnline:        true,
			IsBusy:          false,
			CreatedAt:       time.Now(),
		},
		{
			ID:              "model-3",
			Phone:           "9876543212",
			Name:            "Pooja Verma",
			Role:            models.RoleModel,
			AvatarURL:       "https://images.unsplash.com/photo-1524504388940-b1c1722653e1?w=400&auto=format&fit=crop&q=80",
			Bio:             "Friendly listener & anime enthusiast. Always here to cheer you up!",
			VoiceRatePerMin: 20.0, // 1-on-1: ₹20/min
			GroupRatePerMin: 8.0,  // Group Room: ₹8/min
			ChatRatePerMsg:  2.5,  // ₹2.5/msg
			IsOnline:        true,
			IsBusy:          false,
			CreatedAt:       time.Now(),
		},
	}

	for _, m := range seedModels {
		s.users[m.ID] = m
		s.tokens["token_"+m.ID] = m.ID
		s.wallets[m.ID] = &models.Wallet{
			UserID:      m.ID,
			Balance:     0,
			BonusGiven:  0,
			TotalSpent:  0,
			TotalEarned: 0,
			UpdatedAt:   time.Now(),
		}
	}
}

// User & Auth methods
func (s *Store) CreateOrLoginUser(phone, name string, role models.UserRole) (*models.User, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.users {
		if u.Phone == phone {
			// Invalidate all previous session tokens for this user
			for tok, uid := range s.tokens {
				if uid == u.ID {
					delete(s.tokens, tok)
				}
			}
			newToken := fmt.Sprintf("token_%s_%s", u.ID, uuid.New().String()[:8])
			s.tokens[newToken] = u.ID
			u.ActiveToken = newToken
			return u, newToken, false
		}
	}

	id := "user-" + uuid.New().String()[:8]
	if role == models.RoleModel {
		id = "model-" + uuid.New().String()[:8]
	}

	avatar := "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=400&auto=format&fit=crop&q=80"
	if role == models.RoleModel {
		avatar = "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=400&auto=format&fit=crop&q=80"
	}

	token := fmt.Sprintf("token_%s_%s", id, uuid.New().String()[:8])
	user := &models.User{
		ID:              id,
		Phone:           phone,
		Name:            name,
		Role:            role,
		AvatarURL:       avatar,
		Bio:             "Hey there! Connecting on the app.",
		VoiceRatePerMin: 12.0,
		GroupRatePerMin: 6.0,
		ChatRatePerMsg:  1.5,
		IsOnline:        true,
		IsBusy:          false,
		ActiveToken:     token,
		CreatedAt:       time.Now(),
	}
	s.users[id] = user
	s.tokens[token] = id

	bonus := 0.0
	if role == models.RoleUser {
		bonus = WelcomeBonusAmount
	}

	wallet := &models.Wallet{
		UserID:      id,
		Balance:     bonus,
		BonusGiven:  bonus,
		TotalSpent:  0,
		TotalEarned: 0,
		UpdatedAt:   time.Now(),
	}
	s.wallets[id] = wallet

	if bonus > 0 {
		tx := &models.Transaction{
			ID:          uuid.New().String(),
			UserID:      id,
			Amount:      bonus,
			Type:        models.TxTypeWelcomeBonus,
			Description: fmt.Sprintf("Welcome Bonus Incentive credited: ₹%.2f", bonus),
			CreatedAt:   time.Now(),
		}
		s.transactions[id] = append(s.transactions[id], tx)
	}

	return user, token, true
}

func (s *Store) GetUserByToken(token string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uid, ok := s.tokens[token]
	if !ok {
		return nil, errors.New("invalid or expired token")
	}
	u, ok := s.users[uid]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (s *Store) GetUserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (s *Store) ListModels() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*models.User
	for _, u := range s.users {
		if u.Role == models.RoleModel {
			list = append(list, u)
		}
	}
	return list
}

func (s *Store) SetUserPresence(id string, isOnline, isBusy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, ok := s.users[id]; ok {
		u.IsOnline = isOnline
		u.IsBusy = isBusy
	}
}

// Group Room Management
func (s *Store) CreateGroupRoom(host *models.User, title string, ratePerMin float64) (*models.GroupRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ratePerMin <= 0 {
		ratePerMin = host.GroupRatePerMin
		if ratePerMin <= 0 {
			ratePerMin = 5.0
		}
	}

	roomID := "room_" + uuid.New().String()[:8]
	room := &models.GroupRoom{
		ID:              roomID,
		HostID:          host.ID,
		HostName:        host.Name,
		HostAvatar:      host.AvatarURL,
		Title:           title,
		RatePerMin:      ratePerMin,
		MaxParticipants: 10,
		IsActive:        true,
		Participants:    make(map[string]*models.RoomParticipant),
		CreatedAt:       time.Now(),
	}

	// Host joins as participant (free of charge)
	room.Participants[host.ID] = &models.RoomParticipant{
		UserID:    host.ID,
		Name:      host.Name,
		AvatarURL: host.AvatarURL,
		JoinedAt:  time.Now(),
		IsMuted:   false,
		IsHost:    true,
	}

	s.rooms[roomID] = room
	host.ActiveRoomID = roomID
	host.IsBusy = true

	return room, nil
}

func (s *Store) ListActiveRooms() []*models.GroupRoom {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*models.GroupRoom
	for _, r := range s.rooms {
		if r.IsActive {
			list = append(list, r)
		}
	}
	return list
}

func (s *Store) GetGroupRoom(roomID string) (*models.GroupRoom, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.rooms[roomID]
	if !ok || !r.IsActive {
		return nil, errors.New("group room not found or closed")
	}
	return r, nil
}

func (s *Store) JoinGroupRoom(roomID string, user *models.User) (*models.RoomParticipant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok || !room.IsActive {
		return nil, errors.New("room not active")
	}

	if len(room.Participants) >= room.MaxParticipants {
		return nil, errors.New("room is full")
	}

	// Check if user has at least 1 min balance (if not host)
	if user.ID != room.HostID {
		w, ok := s.wallets[user.ID]
		if !ok || w.Balance < room.RatePerMin {
			return nil, fmt.Errorf("insufficient balance: ₹%.2f required for 1 min in group call", room.RatePerMin)
		}
	}

	participant := &models.RoomParticipant{
		UserID:    user.ID,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		JoinedAt:  time.Now(),
		IsMuted:   false,
		IsHost:    user.ID == room.HostID,
	}
	room.Participants[user.ID] = participant
	user.ActiveRoomID = roomID

	return participant, nil
}

func (s *Store) LeaveGroupRoom(roomID, userID string, reason string) (float64, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return 0, 0, errors.New("room not found")
	}

	p, ok := room.Participants[userID]
	if !ok {
		return 0, 0, errors.New("user not in room")
	}

	now := time.Now()
	p.LeftAt = &now
	durationSec := int(now.Sub(p.JoinedAt).Seconds())
	p.DurationSeconds = durationSec

	cost := 0.0

	// If regular user (not host), bill for time spent
	if !p.IsHost && durationSec > 0 {
		ratePerSec := room.RatePerMin / 60.0
		cost = float64(durationSec) * ratePerSec

		if callerWallet, ok := s.wallets[userID]; ok {
			if cost > callerWallet.Balance {
				cost = callerWallet.Balance
			}
			callerWallet.Balance -= cost
			callerWallet.TotalSpent += cost
			callerWallet.UpdatedAt = time.Now()

			// Credit host (80%)
			hostShare := cost * 0.8
			if hostWallet, ok := s.wallets[room.HostID]; ok {
				hostWallet.Balance += hostShare
				hostWallet.TotalEarned += hostShare
				hostWallet.UpdatedAt = time.Now()
			}

			// Ledger transactions
			s.transactions[userID] = append(s.transactions[userID], &models.Transaction{
				ID:          uuid.New().String(),
				UserID:      userID,
				Amount:      -cost,
				Type:        models.TxTypeGroupDebit,
				Description: fmt.Sprintf("Group Call with %s (%ds @ ₹%.1f/min) - %s", room.HostName, durationSec, room.RatePerMin, reason),
				RoomID:      roomID,
				CreatedAt:   time.Now(),
			})

			s.transactions[room.HostID] = append(s.transactions[room.HostID], &models.Transaction{
				ID:          uuid.New().String(),
				UserID:      room.HostID,
				Amount:      hostShare,
				Type:        models.TxTypeGroupCredit,
				Description: fmt.Sprintf("Group Call earnings from %s (%ds)", p.Name, durationSec),
				RoomID:      roomID,
				CreatedAt:   time.Now(),
			})
		}

		p.TotalCost = cost

		// Record in Call History
		recordID := "group_rec_" + uuid.New().String()[:8]
		rec := &models.CallRecord{
			ID:              recordID,
			CallerID:        userID,
			CallerName:      p.Name,
			ReceiverID:      room.HostID,
			ReceiverName:    room.HostName + " (Group Room)",
			CallType:        "group_voice",
			Status:          models.CallStatusCompleted,
			RatePerMin:      room.RatePerMin,
			StartedAt:       &p.JoinedAt,
			EndedAt:         &now,
			DurationSeconds: durationSec,
			TotalCost:       cost,
			EndReason:       reason,
			CreatedAt:       now,
		}
		s.callRecords[recordID] = rec
		s.userCalls[userID] = append(s.userCalls[userID], recordID)
	}

	delete(room.Participants, userID)
	if u, ok := s.users[userID]; ok {
		u.ActiveRoomID = ""
	}

	// If host leaves, close room for everyone
	if p.IsHost {
		room.IsActive = false
		if u, ok := s.users[room.HostID]; ok {
			u.IsBusy = false
			u.ActiveRoomID = ""
		}
	}

	return cost, durationSec, nil
}

// Wallet & Financial Ledger
func (s *Store) GetWallet(userID string) (*models.Wallet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w, ok := s.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}
	return w, nil
}

func (s *Store) GetTransactions(userID string) []*models.Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txs := s.transactions[userID]
	res := make([]*models.Transaction, len(txs))
	for i, tx := range txs {
		res[len(txs)-1-i] = tx
	}
	return res
}

func (s *Store) RechargeWallet(userID string, amount float64) (*models.Wallet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w, ok := s.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}
	w.Balance += amount
	w.UpdatedAt = time.Now()

	tx := &models.Transaction{
		ID:          uuid.New().String(),
		UserID:      userID,
		Amount:      amount,
		Type:        models.TxTypeRecharge,
		Description: fmt.Sprintf("Wallet Recharge of ₹%.2f", amount),
		CreatedAt:   time.Now(),
	}
	s.transactions[userID] = append(s.transactions[userID], tx)

	return w, nil
}

func (s *Store) DeductChatFee(callerID, modelID string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	callerWallet, ok := s.wallets[callerID]
	if !ok || callerWallet.Balance < amount {
		return errors.New("insufficient balance for chat message")
	}

	callerWallet.Balance -= amount
	callerWallet.TotalSpent += amount
	callerWallet.UpdatedAt = time.Now()

	modelCut := amount * 0.8
	if modelWallet, ok := s.wallets[modelID]; ok {
		modelWallet.Balance += modelCut
		modelWallet.TotalEarned += modelCut
		modelWallet.UpdatedAt = time.Now()
	}

	s.transactions[callerID] = append(s.transactions[callerID], &models.Transaction{
		ID:          uuid.New().String(),
		UserID:      callerID,
		Amount:      -amount,
		Type:        models.TxTypeChatDebit,
		Description: fmt.Sprintf("Chat Message to %s", modelID),
		CreatedAt:   time.Now(),
	})

	return nil
}

// Call Record History (ZERO AUDIO STORED - METADATA ONLY)
func (s *Store) CreateCallRecord(record *models.CallRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callRecords[record.ID] = record
	s.userCalls[record.CallerID] = append(s.userCalls[record.CallerID], record.ID)
	s.userCalls[record.ReceiverID] = append(s.userCalls[record.ReceiverID], record.ID)
}

func (s *Store) UpdateCallRecord(record *models.CallRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callRecords[record.ID] = record
}

func (s *Store) GetCallRecord(callID string) (*models.CallRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.callRecords[callID]
	if !ok {
		return nil, errors.New("call record not found")
	}
	return r, nil
}

func (s *Store) GetUserCallHistory(userID string) []*models.CallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	callIDs := s.userCalls[userID]
	var res []*models.CallRecord
	for i := len(callIDs) - 1; i >= 0; i-- {
		id := callIDs[i]
		if rec, ok := s.callRecords[id]; ok {
			res = append(res, rec)
		}
	}
	return res
}

func (s *Store) ProcessCallFinancials(callerID, receiverID, callID string, durationSec int, ratePerMin float64, endReason string) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if durationSec <= 0 {
		return 0, nil
	}

	minutes := float64(durationSec) / 60.0
	cost := minutes * ratePerMin

	callerWallet, ok := s.wallets[callerID]
	if !ok {
		return 0, errors.New("caller wallet not found")
	}

	if cost > callerWallet.Balance {
		cost = callerWallet.Balance
	}

	callerWallet.Balance -= cost
	if callerWallet.Balance < 0 {
		callerWallet.Balance = 0
	}
	callerWallet.TotalSpent += cost
	callerWallet.UpdatedAt = time.Now()

	modelShare := cost * 0.8
	if modelWallet, ok := s.wallets[receiverID]; ok {
		modelWallet.Balance += modelShare
		modelWallet.TotalEarned += modelShare
		modelWallet.UpdatedAt = time.Now()
	}

	s.transactions[callerID] = append(s.transactions[callerID], &models.Transaction{
		ID:          uuid.New().String(),
		UserID:      callerID,
		Amount:      -cost,
		Type:        models.TxTypeCallDebit,
		Description: fmt.Sprintf("Voice Call (%ds @ ₹%.1f/min) - %s", durationSec, ratePerMin, endReason),
		CallID:      callID,
		CreatedAt:   time.Now(),
	})

	s.transactions[receiverID] = append(s.transactions[receiverID], &models.Transaction{
		ID:          uuid.New().String(),
		UserID:      receiverID,
		Amount:      modelShare,
		Type:        models.TxTypeCallCredit,
		Description: fmt.Sprintf("Call Earnings (%ds @ ₹%.1f/min)", durationSec, ratePerMin),
		CallID:      callID,
		CreatedAt:   time.Now(),
	})

	return cost, nil
}

// Ephemeral Chat Store
func (s *Store) convKey(u1, u2 string) string {
	if u1 < u2 {
		return u1 + "_" + u2
	}
	return u2 + "_" + u1
}

func (s *Store) SaveEphemeralMessage(msg *models.EphemeralMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.convKey(msg.SenderID, msg.ReceiverID)
	if msg.RoomID != "" {
		key = "room_" + msg.RoomID
	}
	s.messages[key] = append(s.messages[key], msg)
}

func (s *Store) GetEphemeralMessages(u1, u2 string) []*models.EphemeralMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.convKey(u1, u2)
	msgs := s.messages[key]
	var active []*models.EphemeralMessage
	now := time.Now()
	for _, m := range msgs {
		if m.ExpiresAt.After(now) {
			active = append(active, m)
		}
	}
	return active
}

func (s *Store) startEphemeralCleaner() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, msgs := range s.messages {
			var valid []*models.EphemeralMessage
			for _, m := range msgs {
				if m.ExpiresAt.After(now) {
					valid = append(valid, m)
				}
			}
			s.messages[key] = valid
		}
		s.mu.Unlock()
	}
}
