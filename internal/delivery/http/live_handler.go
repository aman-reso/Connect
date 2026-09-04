package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LiveStreamInfo struct {
	StreamID       string    `json:"stream_id"`
	HostID         string    `json:"host_id"`
	HostName       string    `json:"host_name"`
	HostAvatar     string    `json:"host_avatar"`
	Title          string    `json:"title"`
	ViewerCount    int       `json:"viewer_count"`
	TotalEarned    float64   `json:"total_earned"`
	StartedAt      time.Time `json:"started_at"`
	IsActive       bool      `json:"is_active"`
	IsPaidMode     bool      `json:"is_paid_mode"`
	CoinRatePerMin float64   `json:"coin_rate_per_min"`
}

type LiveCommentDto struct {
	ID        string    `json:"id"`
	StreamID  string    `json:"stream_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// In-memory live stream registry (synchronized)
var (
	liveStreamsMu sync.RWMutex
	liveStreams   = make(map[string]*LiveStreamInfo)
)

// POST /api/live/start
// Model goes live
func (h *HTTPHandler) HandleStartLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.authUC.ValidateToken(strings.TrimSpace(tokenStr))
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		Title          string  `json:"title"`
		IsPaidMode     bool    `json:"is_paid_mode"`
		CoinRatePerMin float64 `json:"coin_rate_per_min"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Title == "" {
		req.Title = fmt.Sprintf("%s's Live Room", user.Name)
	}
	if req.CoinRatePerMin <= 0 {
		req.CoinRatePerMin = 10.0
	}

	streamID := fmt.Sprintf("live_%s_%d", user.ID, time.Now().Unix())
	stream := &LiveStreamInfo{
		StreamID:       streamID,
		HostID:         user.ID,
		HostName:       user.Name,
		HostAvatar:     user.AvatarURL,
		Title:          req.Title,
		ViewerCount:    1,
		TotalEarned:    0,
		StartedAt:      time.Now(),
		IsActive:       true,
		IsPaidMode:     req.IsPaidMode,
		CoinRatePerMin: req.CoinRatePerMin,
	}

	liveStreamsMu.Lock()
	liveStreams[streamID] = stream
	liveStreamsMu.Unlock()

	apiKey := os.Getenv("LIVEKIT_API_KEY")
	if apiKey == "" {
		apiKey = "APImr59LGqwEVuj"
	}
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	if apiSecret == "" {
		apiSecret = "cvdsoq3pKQusl4HfAHPxSeGXvHcM5atVOWQ2WozyxF2"
	}
	livekitURL := os.Getenv("LIVEKIT_URL")
	if livekitURL == "" {
		livekitURL = "wss://connecto-7sxi06vp.livekit.cloud"
	}

	// Host Publisher Token
	token, err := createLiveBroadcastToken(apiKey, apiSecret, user.ID, user.Name, streamID, true)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to generate host token: "+err.Error())
		return
	}

	SendJSON(w, http.StatusOK, "Live stream started", map[string]any{
		"stream":      stream,
		"livekit_url": livekitURL,
		"token":       token,
	})
}

// POST /api/live/end
func (h *HTTPHandler) HandleEndLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.authUC.ValidateToken(strings.TrimSpace(tokenStr))
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		StreamID string `json:"stream_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	liveStreamsMu.Lock()
	if req.StreamID != "" {
		if stream, exists := liveStreams[req.StreamID]; exists {
			stream.IsActive = false
			delete(liveStreams, req.StreamID)
		}
	}
	// Also remove any active stream where HostID matches this user
	for id, stream := range liveStreams {
		if stream.HostID == user.ID {
			stream.IsActive = false
			delete(liveStreams, id)
		}
	}
	liveStreamsMu.Unlock()

	SendJSON(w, http.StatusOK, "Live stream ended", map[string]any{
		"success": true,
	})
}

func (h *HTTPHandler) HandleLiveList(w http.ResponseWriter, r *http.Request) {
	liveStreamsMu.RLock()
	var list []*LiveStreamInfo = make([]*LiveStreamInfo, 0)
	for _, s := range liveStreams {
		if s.IsActive {
			list = append(list, s)
		}
	}
	liveStreamsMu.RUnlock()

	SendJSON(w, http.StatusOK, "Active live streams", map[string]any{
		"streams": list,
	})
}

// POST /api/live/join
func (h *HTTPHandler) HandleJoinLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.authUC.ValidateToken(strings.TrimSpace(tokenStr))
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		StreamID string `json:"stream_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	liveStreamsMu.Lock()
	if stream, exists := liveStreams[req.StreamID]; exists {
		stream.ViewerCount++
	}
	liveStreamsMu.Unlock()

	apiKey := os.Getenv("LIVEKIT_API_KEY")
	if apiKey == "" {
		apiKey = "APImr59LGqwEVuj"
	}
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	if apiSecret == "" {
		apiSecret = "cvdsoq3pKQusl4HfAHPxSeGXvHcM5atVOWQ2WozyxF2"
	}
	livekitURL := os.Getenv("LIVEKIT_URL")
	if livekitURL == "" {
		livekitURL = "wss://connecto-7sxi06vp.livekit.cloud"
	}

	// Viewer Subscriber Token (canPublish: false)
	token, err := createLiveBroadcastToken(apiKey, apiSecret, user.ID, user.Name, req.StreamID, false)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to generate viewer token: "+err.Error())
		return
	}

	SendJSON(w, http.StatusOK, "Joined live stream", map[string]any{
		"livekit_url": livekitURL,
		"token":       token,
		"room_name":   req.StreamID,
	})
}

// POST /api/live/tip
func (h *HTTPHandler) HandleTipLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.authUC.ValidateToken(strings.TrimSpace(tokenStr))
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		StreamID string  `json:"stream_id"`
		Amount   float64 `json:"amount"`
		GiftName string  `json:"gift_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		SendError(w, http.StatusBadRequest, "Invalid tip amount")
		return
	}

	// Deduct user wallet & credit model wallet
	walletResp, err := h.walletUC.GetWallet(user.ID)
	if err != nil || walletResp == nil || walletResp.Wallet == nil || walletResp.Wallet.Balance < req.Amount {
		SendError(w, http.StatusBadRequest, "Insufficient coin balance in wallet to send tip")
		return
	}

	liveStreamsMu.Lock()
	if stream, exists := liveStreams[req.StreamID]; exists {
		stream.TotalEarned += req.Amount
		// Credit model
		_, _ = h.walletUC.Recharge(stream.HostID, req.Amount)
	}
	liveStreamsMu.Unlock()

	SendJSON(w, http.StatusOK, "Tip sent successfully", map[string]any{
		"amount":    req.Amount,
		"gift_name": req.GiftName,
		"sender":    user.Name,
	})
}

// POST /api/live/paid_mode
func (h *HTTPHandler) HandleTogglePaidMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.authUC.ValidateToken(strings.TrimSpace(tokenStr))
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		StreamID       string  `json:"stream_id"`
		IsPaidMode     bool    `json:"is_paid_mode"`
		CoinRatePerMin float64 `json:"coin_rate_per_min"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.CoinRatePerMin <= 0 {
		req.CoinRatePerMin = 10.0
	}

	liveStreamsMu.Lock()
	stream, exists := liveStreams[req.StreamID]
	if !exists {
		for _, s := range liveStreams {
			if s.HostID == user.ID && s.IsActive {
				stream = s
				exists = true
				break
			}
		}
	}

	if !exists || stream == nil {
		liveStreamsMu.Unlock()
		SendError(w, http.StatusNotFound, "Live stream not found")
		return
	}

	if stream.HostID != user.ID {
		liveStreamsMu.Unlock()
		SendError(w, http.StatusForbidden, "Only the host can configure paid mode")
		return
	}

	stream.IsPaidMode = req.IsPaidMode
	stream.CoinRatePerMin = req.CoinRatePerMin
	copiedStream := *stream
	liveStreamsMu.Unlock()

	SendJSON(w, http.StatusOK, "Paid mode updated successfully", map[string]any{
		"stream": &copiedStream,
	})
}

// GET /api/live/status
func (h *HTTPHandler) HandleLiveStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	streamID := r.URL.Query().Get("stream_id")
	if streamID == "" {
		SendError(w, http.StatusBadRequest, "stream_id is required")
		return
	}

	liveStreamsMu.RLock()
	stream, exists := liveStreams[streamID]
	if !exists || !stream.IsActive {
		liveStreamsMu.RUnlock()
		SendJSON(w, http.StatusOK, "Stream not found or inactive", map[string]any{
			"is_active": false,
		})
		return
	}
	copiedStream := *stream
	liveStreamsMu.RUnlock()

	SendJSON(w, http.StatusOK, "Stream status fetched", map[string]any{
		"stream": &copiedStream,
	})
}

// POST /api/live/deduct
func (h *HTTPHandler) HandleDeductLiveCoins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	user, err := h.authUC.ValidateToken(strings.TrimSpace(tokenStr))
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req struct {
		StreamID        string `json:"stream_id"`
		DurationSeconds int    `json:"duration_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.DurationSeconds <= 0 {
		req.DurationSeconds = 60
	}

	liveStreamsMu.Lock()
	stream, exists := liveStreams[req.StreamID]
	if !exists || !stream.IsActive {
		liveStreamsMu.Unlock()
		SendError(w, http.StatusNotFound, "Live stream is no longer active")
		return
	}

	if !stream.IsPaidMode {
		liveStreamsMu.Unlock()
		walletResp, _ := h.walletUC.GetWallet(user.ID)
		balance := 0.0
		if walletResp != nil && walletResp.Wallet != nil {
			balance = walletResp.Wallet.Balance
		}
		SendJSON(w, http.StatusOK, "Stream is free", map[string]any{
			"success":           true,
			"deducted":          0.0,
			"balance":           balance,
			"is_paid_mode":      false,
			"coin_rate_per_min": stream.CoinRatePerMin,
		})
		return
	}

	ratePerMin := stream.CoinRatePerMin
	if ratePerMin <= 0 {
		ratePerMin = 10.0
	}
	cost := (ratePerMin / 60.0) * float64(req.DurationSeconds)
	if cost < 0.1 {
		cost = 0.1
	}

	// Check balance first
	walletResp, err := h.walletUC.GetWallet(user.ID)
	if err != nil || walletResp == nil || walletResp.Wallet == nil || walletResp.Wallet.Balance < cost {
		currentBal := 0.0
		if walletResp != nil && walletResp.Wallet != nil {
			currentBal = walletResp.Wallet.Balance
		}
		liveStreamsMu.Unlock()
		SendJSON(w, http.StatusOK, "Insufficient coin balance", map[string]any{
			"success":           false,
			"error":             "insufficient_balance",
			"balance":           currentBal,
			"required":          cost,
			"is_paid_mode":      true,
			"coin_rate_per_min": ratePerMin,
		})
		return
	}

	hostID := stream.HostID
	stream.TotalEarned += cost
	liveStreamsMu.Unlock()

	// Deduct via wallet usecase
	desc := fmt.Sprintf("Live stream view (%ds @ %.0f coins/min)", req.DurationSeconds, ratePerMin)
	updatedWallet, err := h.walletUC.DeductLiveFee(user.ID, hostID, cost, desc)
	newBalance := 0.0
	if err == nil && updatedWallet != nil && updatedWallet.Wallet != nil {
		newBalance = updatedWallet.Wallet.Balance
	} else if updatedWallet != nil && updatedWallet.Wallet != nil {
		newBalance = updatedWallet.Wallet.Balance
	} else {
		w2, _ := h.walletUC.GetWallet(user.ID)
		if w2 != nil && w2.Wallet != nil {
			newBalance = w2.Wallet.Balance
		}
	}

	SendJSON(w, http.StatusOK, "Live stream coins deducted successfully", map[string]any{
		"success":           true,
		"deducted":          cost,
		"balance":           newBalance,
		"is_paid_mode":      true,
		"coin_rate_per_min": ratePerMin,
	})
}

// createLiveBroadcastToken issues LiveKit JWT with Publisher or Subscriber video grants
func createLiveBroadcastToken(apiKey, apiSecret, identity, name, roomName string, isPublisher bool) (string, error) {
	type VideoGrant struct {
		RoomJoin     bool   `json:"roomJoin"`
		Room         string `json:"room"`
		CanPublish   bool   `json:"canPublish"`
		CanSubscribe bool   `json:"canSubscribe"`
	}

	claims := jwt.MapClaims{
		"iss":  apiKey,
		"sub":  identity,
		"name": name,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(6 * time.Hour).Unix(),
		"nbf":  time.Now().Unix(),
		"video": VideoGrant{
			RoomJoin:     true,
			Room:         roomName,
			CanPublish:   isPublisher,
			CanSubscribe: true,
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(apiSecret))
}
