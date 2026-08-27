package memory

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/repository"
	"github.com/google/uuid"
)

type MemoryStore struct {
	Users      repository.UserRepository
	Wallets    repository.WalletRepository
	Calls      repository.CallRepository
	Rooms      repository.RoomRepository
	Messages   repository.MessageRepository
	Payments   repository.PaymentRepository
	Onboarding repository.ModelOnboardingRepository
	Reports    repository.ReportRepository
	Favorites  repository.FavoriteRepository
}

func NewMemoryStore() *MemoryStore {
	shared := &memoryData{
		users:         make(map[string]*domain.User),
		tokens:        make(map[string]string),
		wallets:       make(map[string]*domain.Wallet),
		transactions:  make(map[string][]*domain.Transaction),
		callRecords:   make(map[string]*domain.CallRecord),
		userCalls:     make(map[string][]string),
		rooms:         make(map[string]*domain.GroupRoom),
		messages:      make(map[string][]*domain.EphemeralMessage),
		paymentOrders: make(map[string]*domain.PaymentOrder),
		userOrders:    make(map[string][]string),
		auditLogs:     make(map[string][]*domain.PaymentAuditLog),
		profiles:      make(map[string]*domain.ModelProfile),
		reports:       make(map[string]*domain.ModelReport),
		modelReports:  make(map[string][]string),
		userFavorites: make(map[string]map[string]time.Time),
	}
	shared.seedDefaultModels()

	wRepo := &memWalletRepo{data: shared}
	store := &MemoryStore{
		Users:      &memUserRepo{data: shared},
		Wallets:    wRepo,
		Calls:      &memCallRepo{data: shared, wallets: wRepo},
		Rooms:      &memRoomRepo{data: shared, wallets: wRepo},
		Messages:   &memMessageRepo{data: shared},
		Payments:   &memPaymentRepo{data: shared},
		Onboarding: &memOnboardingRepo{data: shared},
		Reports:    &memReportRepo{data: shared},
		Favorites:  &memFavoriteRepo{data: shared},
	}
	go shared.startEphemeralCleaner()
	return store
}

type memoryData struct {
	mu            sync.RWMutex
	users         map[string]*domain.User
	tokens        map[string]string
	wallets       map[string]*domain.Wallet
	transactions  map[string][]*domain.Transaction
	callRecords   map[string]*domain.CallRecord
	userCalls     map[string][]string
	rooms         map[string]*domain.GroupRoom
	messages      map[string][]*domain.EphemeralMessage
	paymentOrders map[string]*domain.PaymentOrder
	userOrders    map[string][]string
	auditLogs     map[string][]*domain.PaymentAuditLog
	profiles      map[string]*domain.ModelProfile
	reports       map[string]*domain.ModelReport
	modelReports  map[string][]string
	userFavorites map[string]map[string]time.Time
}

func (d *memoryData) seedDefaultModels() {
	seedModels := []*domain.User{
		{
			ID:                 "model-1",
			Phone:              "9876543210",
			Name:               "Aanya Sharma",
			Role:               domain.RoleModel,
			AvatarURL:          "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=400&auto=format&fit=crop&q=80",
			Bio:                "Love deep late-night conversations, music & psychology 🌙",
			Age:                22,
			Gender:             "female",
			City:               "New Delhi",
			State:              "Delhi",
			Country:            "India",
			Latitude:           28.6139,
			Longitude:          77.2090,
			Rating:             4.95,
			ReviewCount:        142,
			TotalCallsCount:    380,
			TotalMinutesSpoken: 2600,
			VoiceRatePerMin:    10.0,
			VideoRatePerMin:    20.0,
			GroupRatePerMin:    5.0,
			ChatRatePerMsg:     1.0,
			IsOnline:           true,
			IsBusy:             false,
			CreatedAt:          time.Now().Add(-15 * 24 * time.Hour),
		},
		{
			ID:                 "model-2",
			Phone:              "9876543211",
			Name:               "Riya Sen",
			Role:               domain.RoleModel,
			AvatarURL:          "https://images.unsplash.com/photo-1517841905240-472988babdf9?w=400&auto=format&fit=crop&q=80",
			Bio:                "Artist & traveler. Let's talk about dreams & coffee ☕✨",
			Age:                24,
			Gender:             "female",
			City:               "Mumbai",
			State:              "Maharashtra",
			Country:            "India",
			Latitude:           19.0760,
			Longitude:          72.8777,
			Rating:             4.88,
			ReviewCount:        96,
			TotalCallsCount:    210,
			TotalMinutesSpoken: 1450,
			VoiceRatePerMin:    15.0,
			VideoRatePerMin:    25.0,
			GroupRatePerMin:    7.0,
			ChatRatePerMsg:     2.0,
			IsOnline:           true,
			IsBusy:             false,
			CreatedAt:          time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:                 "model-3",
			Phone:              "9876543212",
			Name:               "Pooja Verma",
			Role:               domain.RoleModel,
			AvatarURL:          "https://images.unsplash.com/photo-1524504388940-b1c1722653e1?w=400&auto=format&fit=crop&q=80",
			Bio:                "Friendly listener & anime enthusiast. Always here to cheer you up!",
			Age:                21,
			Gender:             "female",
			City:               "Bangalore",
			State:              "Karnataka",
			Country:            "India",
			Latitude:           12.9716,
			Longitude:          77.5946,
			Rating:             4.92,
			ReviewCount:        88,
			TotalCallsCount:    190,
			TotalMinutesSpoken: 1300,
			VoiceRatePerMin:    20.0,
			VideoRatePerMin:    35.0,
			GroupRatePerMin:    8.0,
			ChatRatePerMsg:     2.5,
			IsOnline:           true,
			IsBusy:             false,
			CreatedAt:          time.Now().Add(-10 * 24 * time.Hour),
		},
		{
			ID:                 "model-4",
			Phone:              "9876543213",
			Name:               "Kavya Patel",
			Role:               domain.RoleModel,
			AvatarURL:          "https://images.unsplash.com/photo-1529626455594-4ff0802cfb7e?w=400&auto=format&fit=crop&q=80",
			Bio:                "Tech nerd & astrologer. Get your birth chart read with me 🔮",
			Age:                23,
			Gender:             "female",
			City:               "Ahmedabad",
			State:              "Gujarat",
			Country:            "India",
			Latitude:           23.0225,
			Longitude:          72.5714,
			Rating:             4.85,
			ReviewCount:        64,
			TotalCallsCount:    140,
			TotalMinutesSpoken: 950,
			VoiceRatePerMin:    12.0,
			VideoRatePerMin:    22.0,
			GroupRatePerMin:    6.0,
			ChatRatePerMsg:     1.5,
			IsOnline:           true,
			IsBusy:             false,
			CreatedAt:          time.Now().Add(-2 * 24 * time.Hour), // Brand New
		},
	}

	for _, mod := range seedModels {
		d.users[mod.ID] = mod
		tok := fmt.Sprintf("token_%s_seed", mod.ID)
		d.tokens[tok] = mod.ID
		mod.ActiveToken = tok
		d.wallets[mod.ID] = &domain.Wallet{
			UserID:      mod.ID,
			Balance:     0,
			BonusGiven:  0,
			TotalSpent:  0,
			TotalEarned: 0,
			UpdatedAt:   time.Now(),
		}
		d.profiles[mod.ID] = &domain.ModelProfile{
			ID:              "prof_" + mod.ID,
			UserID:          mod.ID,
			DisplayName:     mod.Name,
			Bio:             mod.Bio,
			AvatarURL:       mod.AvatarURL,
			Age:             mod.Age,
			Gender:          mod.Gender,
			City:            mod.City,
			State:           mod.State,
			Country:         mod.Country,
			Latitude:        mod.Latitude,
			Longitude:       mod.Longitude,
			Languages:       "English, Hindi",
			Interests:       "Conversations, Music, Life",
			VoiceRatePerMin: mod.VoiceRatePerMin,
			VideoRatePerMin: mod.VideoRatePerMin,
			GroupRatePerMin: mod.GroupRatePerMin,
			ChatRatePerMsg:  mod.ChatRatePerMsg,
			Status:          domain.OnboardingStatusApproved,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
	}
}

func (d *memoryData) startEphemeralCleaner() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for k, msgs := range d.messages {
			var valid []*domain.EphemeralMessage
			for _, msg := range msgs {
				if msg.ExpiresAt.After(now) {
					valid = append(valid, msg)
				}
			}
			d.messages[k] = valid
		}
		d.mu.Unlock()
	}
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	rad := math.Pi / 180.0
	dlat := (lat2 - lat1) * rad
	dlon := (lon2 - lon1) * rad

	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return 6371 * c
}

// ----------------- USER REPOSITORY -----------------
type memUserRepo struct{ data *memoryData }

func (r *memUserRepo) CreateOrLogin(phone, name string, role domain.UserRole) (*domain.User, string, bool, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	for _, u := range r.data.users {
		if u.Phone == phone {
			for tok, uid := range r.data.tokens {
				if uid == u.ID {
					delete(r.data.tokens, tok)
				}
			}
			newToken := fmt.Sprintf("token_%s_%s", u.ID, uuid.New().String()[:8])
			r.data.tokens[newToken] = u.ID
			u.ActiveToken = newToken
			u.IsOnline = true
			return u, newToken, false, nil
		}
	}

	id := "user-" + uuid.New().String()[:8]
	if role == domain.RoleModel {
		id = "model-" + uuid.New().String()[:8]
	}

	avatar := "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=400"
	if role == domain.RoleModel {
		avatar = "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=400"
	}

	token := fmt.Sprintf("token_%s_%s", id, uuid.New().String()[:8])
	newUser := &domain.User{
		ID:                 id,
		Phone:              phone,
		Name:               name,
		Role:               role,
		AvatarURL:          avatar,
		Bio:                "Hey there! Connecting on Connect.",
		Age:                21,
		Gender:             "female",
		City:               "New Delhi",
		State:              "Delhi",
		Country:            "India",
		Latitude:           28.6139,
		Longitude:          77.2090,
		Rating:             4.90,
		ReviewCount:        0,
		TotalCallsCount:    0,
		TotalMinutesSpoken: 0,
		VoiceRatePerMin:    12.0,
		VideoRatePerMin:    20.0,
		GroupRatePerMin:    6.0,
		ChatRatePerMsg:     1.5,
		IsOnline:           true,
		IsBusy:             false,
		ActiveToken:        token,
		CreatedAt:          time.Now(),
	}
	r.data.users[id] = newUser
	r.data.tokens[token] = id

	bonus := 0.0
	if role == domain.RoleUser {
		bonus = 50.0 // ₹50 Signup Welcome Incentive
	}

	r.data.wallets[id] = &domain.Wallet{
		UserID:      id,
		Balance:     bonus,
		BonusGiven:  bonus,
		TotalSpent:  0,
		TotalEarned: 0,
		UpdatedAt:   time.Now(),
	}

	if bonus > 0 {
		r.data.transactions[id] = append(r.data.transactions[id], &domain.Transaction{
			ID:          uuid.New().String(),
			UserID:      id,
			Amount:      bonus,
			Type:        domain.TxTypeWelcomeBonus,
			Description: "Welcome Bonus Incentive credited: ₹50.00",
			CreatedAt:   time.Now(),
		})
	}

	return newUser, token, true, nil
}

func (r *memUserRepo) GetByToken(token string) (*domain.User, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	uid, ok := r.data.tokens[token]
	if !ok {
		return nil, errors.New("invalid or expired token")
	}
	u, ok := r.data.users[uid]
	if !ok || u.ActiveToken != token {
		return nil, errors.New("session expired: logged in on another device")
	}
	return u, nil
}

func (r *memUserRepo) GetByID(id string) (*domain.User, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	u, ok := r.data.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (r *memUserRepo) ListModels() ([]*domain.User, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var list []*domain.User
	for _, u := range r.data.users {
		if u.Role == domain.RoleModel {
			list = append(list, u)
		}
	}
	return list, nil
}

func (r *memUserRepo) ListModelsAdvanced(filter *domain.ModelFilterParams) ([]*domain.ModelItem, int, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var matched []*domain.ModelItem
	hasGeo := filter.Latitude != 0 && filter.Longitude != 0

	for _, u := range r.data.users {
		if u.Role != domain.RoleModel {
			continue
		}

		prof := r.data.profiles[u.ID]

		// Filters
		if filter.IsOnline != nil && *filter.IsOnline && !u.IsOnline {
			continue
		}
		if filter.MinAge > 0 && u.Age < filter.MinAge {
			continue
		}
		if filter.MaxAge > 0 && u.Age > filter.MaxAge {
			continue
		}
		if filter.Gender != "" && filter.Gender != "all" && !strings.EqualFold(u.Gender, filter.Gender) {
			continue
		}
		if filter.City != "" && !strings.Contains(strings.ToLower(u.City), strings.ToLower(filter.City)) {
			continue
		}
		if filter.MinRate > 0 && u.VoiceRatePerMin < filter.MinRate {
			continue
		}
		if filter.MaxRate > 0 && u.VoiceRatePerMin > filter.MaxRate {
			continue
		}

		it := &domain.ModelItem{
			User:            *u,
			DisplayName:     u.Name,
			Languages:       []string{"English", "Hindi"},
			Interests:       []string{"Conversations", "Music"},
			ProfileVerified: true,
		}

		if prof != nil {
			if prof.DisplayName != "" {
				it.DisplayName = prof.DisplayName
			}
			if prof.AudioIntroURL != "" {
				it.AudioIntroURL = prof.AudioIntroURL
			}
			if len(prof.GalleryURLs) > 0 {
				it.GalleryURLs = prof.GalleryURLs
			}
			if prof.Languages != "" {
				it.Languages = nil
				for _, part := range strings.Split(prof.Languages, ",") {
					if p := strings.TrimSpace(part); p != "" {
						it.Languages = append(it.Languages, p)
					}
				}
			}
			if prof.Interests != "" {
				it.Interests = nil
				for _, part := range strings.Split(prof.Interests, ",") {
					if p := strings.TrimSpace(part); p != "" {
						it.Interests = append(it.Interests, p)
					}
				}
			}
		}

		// Language filter
		if filter.Language != "" {
			langMatch := false
			for _, l := range it.Languages {
				if strings.Contains(strings.ToLower(l), strings.ToLower(filter.Language)) {
					langMatch = true
					break
				}
			}
			if !langMatch {
				continue
			}
		}

		// Interest filter
		if filter.Interest != "" {
			intMatch := false
			for _, in := range it.Interests {
				if strings.Contains(strings.ToLower(in), strings.ToLower(filter.Interest)) {
					intMatch = true
					break
				}
			}
			if !intMatch {
				continue
			}
		}

		// Calculate Distance
		if hasGeo && u.Latitude != 0 && u.Longitude != 0 {
			dist := haversineDistance(filter.Latitude, filter.Longitude, u.Latitude, u.Longitude)
			it.DistanceKM = &dist
		}

		// Badges
		if time.Since(u.CreatedAt) < 14*24*time.Hour {
			it.IsNew = true
			it.Badges = append(it.Badges, "New Creator")
		}
		if u.Rating >= 4.90 {
			it.Badges = append(it.Badges, "Top Rated")
		}
		if u.TotalCallsCount >= 100 {
			it.Badges = append(it.Badges, "Popular")
		}
		it.Badges = append(it.Badges, "Verified")
		if it.DistanceKM != nil && *it.DistanceKM <= 25.0 {
			it.Badges = append(it.Badges, "Nearby")
		}

		matched = append(matched, it)
	}

	// Sorting
	switch filter.Filter {
	case "nearby":
		sort.Slice(matched, func(i, j int) bool {
			if matched[i].DistanceKM == nil {
				return false
			}
			if matched[j].DistanceKM == nil {
				return true
			}
			return *matched[i].DistanceKM < *matched[j].DistanceKM
		})
	case "new":
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].CreatedAt.After(matched[j].CreatedAt)
		})
	case "top":
		sort.Slice(matched, func(i, j int) bool {
			if matched[i].Rating != matched[j].Rating {
				return matched[i].Rating > matched[j].Rating
			}
			return matched[i].TotalCallsCount > matched[j].TotalCallsCount
		})
	default:
		switch filter.SortBy {
		case "distance":
			sort.Slice(matched, func(i, j int) bool {
				if matched[i].DistanceKM == nil {
					return false
				}
				if matched[j].DistanceKM == nil {
					return true
				}
				return *matched[i].DistanceKM < *matched[j].DistanceKM
			})
		case "rating":
			sort.Slice(matched, func(i, j int) bool {
				return matched[i].Rating > matched[j].Rating
			})
		case "newest":
			sort.Slice(matched, func(i, j int) bool {
				return matched[i].CreatedAt.After(matched[j].CreatedAt)
			})
		case "calls", "popularity":
			sort.Slice(matched, func(i, j int) bool {
				return matched[i].TotalCallsCount > matched[j].TotalCallsCount
			})
		case "price_low":
			sort.Slice(matched, func(i, j int) bool {
				return matched[i].VoiceRatePerMin < matched[j].VoiceRatePerMin
			})
		case "price_high":
			sort.Slice(matched, func(i, j int) bool {
				return matched[i].VoiceRatePerMin > matched[j].VoiceRatePerMin
			})
		default:
			sort.Slice(matched, func(i, j int) bool {
				if matched[i].IsOnline != matched[j].IsOnline {
					return matched[i].IsOnline
				}
				return matched[i].Rating > matched[j].Rating
			})
		}
	}

	totalCount := len(matched)
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= totalCount {
		return []*domain.ModelItem{}, totalCount, nil
	}
	end := start + limit
	if end > totalCount {
		end = totalCount
	}

	return matched[start:end], totalCount, nil
}

func (r *memUserRepo) UpdateUserOnboarding(userID string, p *domain.ModelProfile) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	u, ok := r.data.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	if p.DisplayName != "" {
		u.Name = p.DisplayName
	}
	if p.Bio != "" {
		u.Bio = p.Bio
	}
	if p.AvatarURL != "" {
		u.AvatarURL = p.AvatarURL
	}
	if p.Age > 0 {
		u.Age = p.Age
	}
	if p.Gender != "" {
		u.Gender = p.Gender
	}
	if p.City != "" {
		u.City = p.City
	}
	if p.State != "" {
		u.State = p.State
	}
	if p.Country != "" {
		u.Country = p.Country
	}
	if p.Latitude != 0 {
		u.Latitude = p.Latitude
	}
	if p.Longitude != 0 {
		u.Longitude = p.Longitude
	}
	if p.VoiceRatePerMin > 0 {
		u.VoiceRatePerMin = p.VoiceRatePerMin
	}
	if p.VideoRatePerMin > 0 {
		u.VideoRatePerMin = p.VideoRatePerMin
	}
	if p.GroupRatePerMin > 0 {
		u.GroupRatePerMin = p.GroupRatePerMin
	}
	if p.ChatRatePerMsg > 0 {
		u.ChatRatePerMsg = p.ChatRatePerMsg
	}
	return nil
}

func (r *memUserRepo) SetPresence(id string, isOnline, isBusy bool) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	if u, ok := r.data.users[id]; ok {
		u.IsOnline = isOnline
		u.IsBusy = isBusy
	}
	return nil
}

// ----------------- WALLET REPOSITORY -----------------
type memWalletRepo struct{ data *memoryData }

func (r *memWalletRepo) GetWallet(userID string) (*domain.Wallet, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	w, ok := r.data.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}
	return w, nil
}

func (r *memWalletRepo) GetTransactions(userID string) ([]*domain.Transaction, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	txs := r.data.transactions[userID]
	res := make([]*domain.Transaction, len(txs))
	for i, tx := range txs {
		res[len(txs)-1-i] = tx
	}
	return res, nil
}

func (r *memWalletRepo) Recharge(userID string, amount float64) (*domain.Wallet, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	w, ok := r.data.wallets[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}
	w.Balance += amount
	w.UpdatedAt = time.Now()

	r.data.transactions[userID] = append(r.data.transactions[userID], &domain.Transaction{
		ID:          uuid.New().String(),
		UserID:      userID,
		Amount:      amount,
		Type:        domain.TxTypeRecharge,
		Description: fmt.Sprintf("Wallet Recharge of ₹%.2f", amount),
		CreatedAt:   time.Now(),
	})
	return w, nil
}

func (r *memWalletRepo) ProcessCallSettlement(callerID, receiverID, callID string, durationSec int, ratePerMin float64, reason string) (float64, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	if durationSec <= 0 {
		return 0, nil
	}

	cost := (float64(durationSec) / 60.0) * ratePerMin
	callerWallet, ok := r.data.wallets[callerID]
	if !ok {
		return 0, errors.New("caller wallet not found")
	}

	if cost > callerWallet.Balance {
		cost = callerWallet.Balance
	}

	callerWallet.Balance -= cost
	callerWallet.TotalSpent += cost
	callerWallet.UpdatedAt = time.Now()

	modelShare := cost * 0.8
	if modelWallet, ok := r.data.wallets[receiverID]; ok {
		modelWallet.Balance += modelShare
		modelWallet.TotalEarned += modelShare
		modelWallet.UpdatedAt = time.Now()
	}

	r.data.transactions[callerID] = append(r.data.transactions[callerID], &domain.Transaction{
		ID:          uuid.New().String(),
		UserID:      callerID,
		Amount:      -cost,
		Type:        domain.TxTypeCallDebit,
		Description: fmt.Sprintf("Voice Call (%ds @ ₹%.1f/min) - %s", durationSec, ratePerMin, reason),
		CallID:      callID,
		CreatedAt:   time.Now(),
	})

	r.data.transactions[receiverID] = append(r.data.transactions[receiverID], &domain.Transaction{
		ID:          uuid.New().String(),
		UserID:      receiverID,
		Amount:      modelShare,
		Type:        domain.TxTypeCallCredit,
		Description: fmt.Sprintf("Call Earnings (%ds @ ₹%.1f/min)", durationSec, ratePerMin),
		CallID:      callID,
		CreatedAt:   time.Now(),
	})

	return cost, nil
}

func (r *memWalletRepo) DeductChatFee(callerID, receiverID string, amount float64) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	w, ok := r.data.wallets[callerID]
	if !ok || w.Balance < amount {
		return errors.New("insufficient balance for chat message")
	}

	w.Balance -= amount
	w.TotalSpent += amount
	w.UpdatedAt = time.Now()

	modelShare := amount * 0.8
	if mw, ok := r.data.wallets[receiverID]; ok {
		mw.Balance += modelShare
		mw.TotalEarned += modelShare
		mw.UpdatedAt = time.Now()
	}

	r.data.transactions[callerID] = append(r.data.transactions[callerID], &domain.Transaction{
		ID:          uuid.New().String(),
		UserID:      callerID,
		Amount:      -amount,
		Type:        domain.TxTypeChatDebit,
		Description: fmt.Sprintf("Chat Message to %s", receiverID),
		CreatedAt:   time.Now(),
	})
	return nil
}

// ----------------- CALL REPOSITORY -----------------
type memCallRepo struct {
	data    *memoryData
	wallets repository.WalletRepository
}

func (r *memCallRepo) Create(record *domain.CallRecord) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.callRecords[record.ID] = record
	r.data.userCalls[record.CallerID] = append(r.data.userCalls[record.CallerID], record.ID)
	r.data.userCalls[record.ReceiverID] = append(r.data.userCalls[record.ReceiverID], record.ID)
	return nil
}

func (r *memCallRepo) Update(record *domain.CallRecord) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.callRecords[record.ID] = record
	return nil
}

func (r *memCallRepo) GetByID(id string) (*domain.CallRecord, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	rec, ok := r.data.callRecords[id]
	if !ok {
		return nil, errors.New("call record not found")
	}
	return rec, nil
}

func (r *memCallRepo) GetUserHistory(userID string) ([]*domain.CallRecord, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	callIDs := r.data.userCalls[userID]
	var res []*domain.CallRecord
	for i := len(callIDs) - 1; i >= 0; i-- {
		id := callIDs[i]
		if rec, ok := r.data.callRecords[id]; ok {
			res = append(res, rec)
		}
	}
	return res, nil
}

func (r *memCallRepo) UpdateHeartbeat(callID string) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	if rec, ok := r.data.callRecords[callID]; ok {
		now := time.Now()
		rec.LastHeartbeat = &now
	}
	return nil
}

func (r *memCallRepo) RecoverInterruptedCalls() error {
	return nil
}

// ----------------- ROOM REPOSITORY -----------------
type memRoomRepo struct {
	data    *memoryData
	wallets repository.WalletRepository
}

func (r *memRoomRepo) Create(host *domain.User, title string, ratePerMin float64) (*domain.GroupRoom, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	if ratePerMin <= 0 {
		ratePerMin = host.GroupRatePerMin
		if ratePerMin <= 0 {
			ratePerMin = 5.0
		}
	}

	roomID := "room_" + uuid.New().String()[:8]
	room := &domain.GroupRoom{
		ID:              roomID,
		HostID:          host.ID,
		HostName:        host.Name,
		HostAvatar:      host.AvatarURL,
		Title:           title,
		RatePerMin:      ratePerMin,
		MaxParticipants: 10,
		IsActive:        true,
		Participants:    make(map[string]*domain.RoomParticipant),
		CreatedAt:       time.Now(),
	}

	room.Participants[host.ID] = &domain.RoomParticipant{
		UserID:    host.ID,
		Name:      host.Name,
		AvatarURL: host.AvatarURL,
		JoinedAt:  time.Now(),
		IsHost:    true,
	}

	r.data.rooms[roomID] = room
	host.ActiveRoomID = roomID
	host.IsBusy = true
	return room, nil
}

func (r *memRoomRepo) GetByID(roomID string) (*domain.GroupRoom, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	room, ok := r.data.rooms[roomID]
	if !ok || !room.IsActive {
		return nil, errors.New("group room not found or closed")
	}
	return room, nil
}

func (r *memRoomRepo) ListActive() ([]*domain.GroupRoom, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var list []*domain.GroupRoom
	for _, room := range r.data.rooms {
		if room.IsActive {
			list = append(list, room)
		}
	}
	return list, nil
}

func (r *memRoomRepo) AddParticipant(roomID string, user *domain.User) (*domain.RoomParticipant, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	room, ok := r.data.rooms[roomID]
	if !ok || !room.IsActive {
		return nil, errors.New("room not active")
	}

	if len(room.Participants) >= room.MaxParticipants {
		return nil, errors.New("room is full")
	}

	if user.ID != room.HostID {
		w, ok := r.data.wallets[user.ID]
		if !ok || w.Balance < room.RatePerMin {
			return nil, fmt.Errorf("insufficient balance: ₹%.2f required for 1 min in group call", room.RatePerMin)
		}
	}

	p := &domain.RoomParticipant{
		UserID:    user.ID,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		JoinedAt:  time.Now(),
		IsHost:    user.ID == room.HostID,
	}
	room.Participants[user.ID] = p
	user.ActiveRoomID = roomID
	return p, nil
}

func (r *memRoomRepo) RemoveParticipant(roomID, userID, reason string) (float64, int, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	room, ok := r.data.rooms[roomID]
	if !ok {
		return 0, 0, errors.New("room not found")
	}

	p, ok := room.Participants[userID]
	if !ok {
		return 0, 0, errors.New("user not in room")
	}

	now := time.Now()
	durationSec := int(now.Sub(p.JoinedAt).Seconds())
	cost := 0.0

	if !p.IsHost && durationSec > 0 {
		ratePerSec := room.RatePerMin / 60.0
		cost = float64(durationSec) * ratePerSec

		if callerWallet, ok := r.data.wallets[userID]; ok {
			if cost > callerWallet.Balance {
				cost = callerWallet.Balance
			}
			callerWallet.Balance -= cost
			callerWallet.TotalSpent += cost
			callerWallet.UpdatedAt = time.Now()

			hostShare := cost * 0.8
			if hostWallet, ok := r.data.wallets[room.HostID]; ok {
				hostWallet.Balance += hostShare
				hostWallet.TotalEarned += hostShare
				hostWallet.UpdatedAt = time.Now()
			}

			r.data.transactions[userID] = append(r.data.transactions[userID], &domain.Transaction{
				ID:          uuid.New().String(),
				UserID:      userID,
				Amount:      -cost,
				Type:        domain.TxTypeGroupDebit,
				Description: fmt.Sprintf("Group Call with %s (%ds @ ₹%.1f/min) - %s", room.HostName, durationSec, room.RatePerMin, reason),
				RoomID:      roomID,
				CreatedAt:   time.Now(),
			})

			r.data.transactions[room.HostID] = append(r.data.transactions[room.HostID], &domain.Transaction{
				ID:          uuid.New().String(),
				UserID:      room.HostID,
				Amount:      hostShare,
				Type:        domain.TxTypeGroupCredit,
				Description: fmt.Sprintf("Group Call earnings from %s (%ds)", p.Name, durationSec),
				RoomID:      roomID,
				CreatedAt:   time.Now(),
			})
		}
	}

	delete(room.Participants, userID)
	if u, ok := r.data.users[userID]; ok {
		u.ActiveRoomID = ""
	}

	if p.IsHost {
		room.IsActive = false
		if u, ok := r.data.users[room.HostID]; ok {
			u.IsBusy = false
			u.ActiveRoomID = ""
		}
	}

	return cost, durationSec, nil
}

// ----------------- MESSAGE REPOSITORY -----------------
type memMessageRepo struct{ data *memoryData }

func (r *memMessageRepo) Save(msg *domain.EphemeralMessage) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	key := msg.SenderID + "_" + msg.ReceiverID
	if msg.RoomID != "" {
		key = "room_" + msg.RoomID
	}
	r.data.messages[key] = append(r.data.messages[key], msg)
	return nil
}

func (r *memMessageRepo) GetActive(u1, u2 string) ([]*domain.EphemeralMessage, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	key := u1 + "_" + u2
	msgs := r.data.messages[key]
	var active []*domain.EphemeralMessage
	now := time.Now()
	for _, msg := range msgs {
		if msg.ExpiresAt.After(now) {
			active = append(active, msg)
		}
	}
	return active, nil
}

func (r *memMessageRepo) GetConversations(userID string) ([]*dto.ConversationDTO, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var list []*dto.ConversationDTO
	seen := make(map[string]bool)
	now := time.Now()

	for _, u := range r.data.users {
		if u.ID != userID && !seen[u.ID] {
			seen[u.ID] = true
			list = append(list, &dto.ConversationDTO{
				ID:              "conv_" + u.ID,
				PartnerID:       u.ID,
				PartnerName:     u.Name,
				PartnerAvatar:   u.AvatarURL,
				LastMessage:     u.Bio,
				LastMessageTime: now.Unix() * 1000,
				UnreadCount:     0,
				IsOnline:        u.IsOnline,
			})
		}
	}
	return list, nil
}

func (r *memMessageRepo) PurgeExpired() error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	now := time.Now()
	for k, msgs := range r.data.messages {
		var valid []*domain.EphemeralMessage
		for _, msg := range msgs {
			if msg.ExpiresAt.After(now) {
				valid = append(valid, msg)
			}
		}
		r.data.messages[k] = valid
	}
	return nil
}

// ----------------- PAYMENT REPOSITORY -----------------
type memPaymentRepo struct{ data *memoryData }

func (r *memPaymentRepo) CreateOrder(order *domain.PaymentOrder) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.paymentOrders[order.ID] = order
	r.data.userOrders[order.UserID] = append(r.data.userOrders[order.UserID], order.ID)
	return nil
}

func (r *memPaymentRepo) UpdateOrder(order *domain.PaymentOrder) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	order.UpdatedAt = time.Now()
	r.data.paymentOrders[order.ID] = order
	return nil
}

func (r *memPaymentRepo) GetOrderByID(paymentID string) (*domain.PaymentOrder, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	order, ok := r.data.paymentOrders[paymentID]
	if !ok {
		return nil, errors.New("payment order not found")
	}
	return order, nil
}

func (r *memPaymentRepo) GetOrdersByUserID(userID string) ([]*domain.PaymentOrder, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	orderIDs := r.data.userOrders[userID]
	var list []*domain.PaymentOrder
	for i := len(orderIDs) - 1; i >= 0; i-- {
		id := orderIDs[i]
		if order, ok := r.data.paymentOrders[id]; ok {
			list = append(list, order)
		}
	}
	return list, nil
}

func (r *memPaymentRepo) RecordAuditLog(log *domain.PaymentAuditLog) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.auditLogs[log.PaymentID] = append(r.data.auditLogs[log.PaymentID], log)
	return nil
}

func (r *memPaymentRepo) GetAuditLogs(paymentID string) ([]*domain.PaymentAuditLog, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	return r.data.auditLogs[paymentID], nil
}

// ----------------- MODEL ONBOARDING REPOSITORY -----------------
type memOnboardingRepo struct{ data *memoryData }

func (r *memOnboardingRepo) SaveProfile(p *domain.ModelProfile) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.profiles[p.UserID] = p
	return nil
}

func (r *memOnboardingRepo) UpdateProfile(p *domain.ModelProfile) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	p.UpdatedAt = time.Now()
	r.data.profiles[p.UserID] = p
	return nil
}

func (r *memOnboardingRepo) GetProfileByUserID(userID string) (*domain.ModelProfile, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	p, ok := r.data.profiles[userID]
	if !ok {
		return nil, errors.New("model profile not found")
	}
	return p, nil
}

func (r *memOnboardingRepo) ListPendingProfiles() ([]*domain.ModelProfile, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var list []*domain.ModelProfile
	for _, p := range r.data.profiles {
		if p.Status == domain.OnboardingStatusPendingReview {
			list = append(list, p)
		}
	}
	return list, nil
}

func (r *memOnboardingRepo) IncrementReportCount(modelID string) (int, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	p, ok := r.data.profiles[modelID]
	if !ok {
		p = &domain.ModelProfile{
			ID:          "prof_" + modelID,
			UserID:      modelID,
			ReportCount: 1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		r.data.profiles[modelID] = p
		return 1, nil
	}

	p.ReportCount++
	p.UpdatedAt = time.Now()
	return p.ReportCount, nil
}

func (r *memOnboardingRepo) SetSuspension(modelID string, isSuspended bool) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	if p, ok := r.data.profiles[modelID]; ok {
		p.IsSuspended = isSuspended
		p.UpdatedAt = time.Now()
	}
	return nil
}

// ----------------- REPORT REPOSITORY -----------------
type memReportRepo struct{ data *memoryData }

func (r *memReportRepo) CreateReport(report *domain.ModelReport) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.reports[report.ID] = report
	r.data.modelReports[report.ModelID] = append(r.data.modelReports[report.ModelID], report.ID)
	return nil
}

func (r *memReportRepo) UpdateReport(report *domain.ModelReport) error {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	r.data.reports[report.ID] = report
	return nil
}

func (r *memReportRepo) GetReportByID(id string) (*domain.ModelReport, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	rep, ok := r.data.reports[id]
	if !ok {
		return nil, errors.New("report not found")
	}
	return rep, nil
}

func (r *memReportRepo) GetReportsForModel(modelID string) ([]*domain.ModelReport, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	repIDs := r.data.modelReports[modelID]
	var list []*domain.ModelReport
	for i := len(repIDs) - 1; i >= 0; i-- {
		id := repIDs[i]
		if rep, ok := r.data.reports[id]; ok {
			list = append(list, rep)
		}
	}
	return list, nil
}

func (r *memReportRepo) ListRecentReports() ([]*domain.ModelReport, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var list []*domain.ModelReport
	for _, rep := range r.data.reports {
		list = append(list, rep)
	}
	return list, nil
}

// ----------------- FAVORITE REPO (IN-MEMORY) -----------------
type memFavoriteRepo struct{ data *memoryData }

func (r *memFavoriteRepo) ToggleFavorite(userID, modelID string) (bool, error) {
	r.data.mu.Lock()
	defer r.data.mu.Unlock()

	favs, ok := r.data.userFavorites[userID]
	if !ok {
		favs = make(map[string]time.Time)
		r.data.userFavorites[userID] = favs
	}

	if _, exists := favs[modelID]; exists {
		delete(favs, modelID)
		return false, nil
	}

	favs[modelID] = time.Now()
	return true, nil
}

func (r *memFavoriteRepo) IsFavorite(userID, modelID string) (bool, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	if favs, ok := r.data.userFavorites[userID]; ok {
		_, exists := favs[modelID]
		return exists, nil
	}
	return false, nil
}

func (r *memFavoriteRepo) GetFavoriteModelIDs(userID string) ([]string, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var ids []string
	if favs, ok := r.data.userFavorites[userID]; ok {
		for id := range favs {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *memFavoriteRepo) GetFavoriteModels(userID string) ([]*domain.User, error) {
	r.data.mu.RLock()
	defer r.data.mu.RUnlock()

	var models []*domain.User
	if favs, ok := r.data.userFavorites[userID]; ok {
		for id := range favs {
			if u, exists := r.data.users[id]; exists && u.Role == domain.RoleModel {
				models = append(models, u)
			}
		}
	}
	return models, nil
}

