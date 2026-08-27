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
	var cards []*dto.ModelCardDTO
	for _, u := range models {
		age := u.Age
		if age == 0 {
			age = 21
		}
		rating := u.Rating
		if rating == 0 {
			rating = 4.90
		}
		videoRate := u.VideoRatePerMin
		if videoRate == 0 {
			videoRate = 20.0
		}
		cards = append(cards, &dto.ModelCardDTO{
			ID:                 u.ID,
			Name:               u.Name,
			DisplayName:        u.Name,
			Role:               string(u.Role),
			AvatarURL:          u.AvatarURL,
			Bio:                u.Bio,
			Age:                age,
			Gender:             u.Gender,
			City:               u.City,
			State:              u.State,
			Country:            u.Country,
			Latitude:           u.Latitude,
			Longitude:          u.Longitude,
			Languages:          []string{"English", "Hindi"},
			Interests:          []string{"Conversations", "Music"},
			VoiceRatePerMin:    u.VoiceRatePerMin,
			VideoRatePerMin:    videoRate,
			GroupRatePerMin:    u.GroupRatePerMin,
			ChatRatePerMsg:     u.ChatRatePerMsg,
			IsOnline:           u.IsOnline,
			IsBusy:             u.IsBusy,
			Rating:             rating,
			ReviewCount:        u.ReviewCount,
			TotalCallsCount:    u.TotalCallsCount,
			TotalMinutesSpoken: u.TotalMinutesSpoken,
			Badges:             []string{"Verified"},
			CreatedAt:          u.CreatedAt,
		})
	}
	return &dto.ModelListResponse{
		Count: len(cards),
		Pagination: dto.PaginationMeta{
			CurrentPage: 1,
			Limit:       len(cards),
			TotalCount:  len(cards),
			TotalPages:  1,
			HasNext:     false,
			HasPrev:     false,
		},
		FiltersApplied: map[string]interface{}{"filter": "all"},
		Models:         cards,
	}
}

func (m *Mapper) ToModelCardDTO(u *domain.User) *dto.ModelCardDTO {
	if u == nil {
		return nil
	}
	age := u.Age
	if age == 0 {
		age = 21
	}
	rating := u.Rating
	if rating == 0 {
		rating = 4.90
	}
	videoRate := u.VideoRatePerMin
	if videoRate == 0 {
		videoRate = 20.0
	}
	return &dto.ModelCardDTO{
		ID:                 u.ID,
		Name:               u.Name,
		DisplayName:        u.Name,
		Role:               string(u.Role),
		AvatarURL:          u.AvatarURL,
		Bio:                u.Bio,
		Age:                age,
		Gender:             u.Gender,
		City:               u.City,
		State:              u.State,
		Country:            u.Country,
		Latitude:           u.Latitude,
		Longitude:          u.Longitude,
		Languages:          []string{"English", "Hindi"},
		Interests:          []string{"Conversations", "Music"},
		VoiceRatePerMin:    u.VoiceRatePerMin,
		VideoRatePerMin:    videoRate,
		GroupRatePerMin:    u.GroupRatePerMin,
		ChatRatePerMsg:     u.ChatRatePerMsg,
		IsOnline:           u.IsOnline,
		IsBusy:             u.IsBusy,
		Rating:             rating,
		ReviewCount:        u.ReviewCount,
		TotalCallsCount:    u.TotalCallsCount,
		TotalMinutesSpoken: u.TotalMinutesSpoken,
		Badges:             []string{"Verified"},
		CreatedAt:          u.CreatedAt,
	}
}

func (m *Mapper) ToPaginatedModelListResponse(items []*domain.ModelItem, totalCount int, filter *dto.ModelFilterQuery) *dto.ModelListResponse {
	var cards []*dto.ModelCardDTO
	for _, it := range items {
		age := it.Age
		if age == 0 {
			age = 21
		}
		rating := it.Rating
		if rating == 0 {
			rating = 4.90
		}
		videoRate := it.VideoRatePerMin
		if videoRate == 0 {
			videoRate = 20.0
		}
		displayName := it.DisplayName
		if displayName == "" {
			displayName = it.Name
		}
		langs := it.Languages
		if len(langs) == 0 {
			langs = []string{"English", "Hindi"}
		}
		interests := it.Interests
		if len(interests) == 0 {
			interests = []string{"Conversations", "Music"}
		}

		card := &dto.ModelCardDTO{
			ID:                 it.ID,
			Name:               it.Name,
			DisplayName:        displayName,
			Role:               string(it.Role),
			AvatarURL:          it.AvatarURL,
			GalleryURLs:        it.GalleryURLs,
			Bio:                it.Bio,
			Age:                age,
			Gender:             it.Gender,
			City:               it.City,
			State:              it.State,
			Country:            it.Country,
			Latitude:           it.Latitude,
			Longitude:          it.Longitude,
			DistanceKM:         it.DistanceKM,
			Languages:          langs,
			Interests:          interests,
			VoiceRatePerMin:    it.VoiceRatePerMin,
			VideoRatePerMin:    videoRate,
			GroupRatePerMin:    it.GroupRatePerMin,
			ChatRatePerMsg:     it.ChatRatePerMsg,
			IsOnline:           it.IsOnline,
			IsBusy:             it.IsBusy,
			Rating:             rating,
			ReviewCount:        it.ReviewCount,
			TotalCallsCount:    it.TotalCallsCount,
			TotalMinutesSpoken: it.TotalMinutesSpoken,
			Badges:             it.Badges,
			IsNew:              it.IsNew,
			AudioIntroURL:      it.AudioIntroURL,
			CreatedAt:          it.CreatedAt,
		}
		cards = append(cards, card)
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	totalPages := (totalCount + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	filterName := filter.Filter
	if filterName == "" {
		filterName = "all"
	}

	filtersApplied := map[string]interface{}{
		"filter": filterName,
		"page":   page,
		"limit":  limit,
	}
	if filter.Lat != 0 && filter.Lng != 0 {
		filtersApplied["lat"] = filter.Lat
		filtersApplied["lng"] = filter.Lng
		if filter.MaxDistanceKM > 0 {
			filtersApplied["max_distance_km"] = filter.MaxDistanceKM
		}
	}
	if filter.City != "" {
		filtersApplied["city"] = filter.City
	}
	if filter.Gender != "" {
		filtersApplied["gender"] = filter.Gender
	}
	if filter.MinAge > 0 {
		filtersApplied["min_age"] = filter.MinAge
	}
	if filter.MaxAge > 0 {
		filtersApplied["max_age"] = filter.MaxAge
	}
	if filter.SortBy != "" {
		filtersApplied["sort_by"] = filter.SortBy
	}

	return &dto.ModelListResponse{
		Count: len(cards),
		Pagination: dto.PaginationMeta{
			CurrentPage: page,
			Limit:       limit,
			TotalCount:  totalCount,
			TotalPages:  totalPages,
			HasNext:     page < totalPages,
			HasPrev:     page > 1,
		},
		FiltersApplied: filtersApplied,
		Models:         cards,
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

