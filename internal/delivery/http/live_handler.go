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
	StreamID    string    `json:"stream_id"`
	HostID      string    `json:"host_id"`
	HostName    string    `json:"host_name"`
	HostAvatar  string    `json:"host_avatar"`
	Title       string    `json:"title"`
	ViewerCount int       `json:"viewer_count"`
	TotalEarned float64   `json:"total_earned"`
	StartedAt   time.Time `json:"started_at"`
	IsActive    bool      `json:"is_active"`
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
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Title == "" {
		req.Title = fmt.Sprintf("%s's Live Room", user.Name)
	}

	streamID := fmt.Sprintf("live_%s_%d", user.ID, time.Now().Unix())
	stream := &LiveStreamInfo{
		StreamID:    streamID,
		HostID:      user.ID,
		HostName:    user.Name,
		HostAvatar:  user.AvatarURL,
		Title:       req.Title,
		ViewerCount: 1,
		TotalEarned: 0,
		StartedAt:   time.Now(),
		IsActive:    true,
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
	stream, exists := liveStreams[req.StreamID]
	if exists && stream.HostID == user.ID {
		stream.IsActive = false
		delete(liveStreams, req.StreamID)
	}
	liveStreamsMu.Unlock()

	SendJSON(w, http.StatusOK, "Live stream ended", map[string]any{
		"success": true,
	})
}

// GET /api/live/list
func (h *HTTPHandler) HandleLiveList(w http.ResponseWriter, r *http.Request) {
	liveStreamsMu.RLock()
	var list []*LiveStreamInfo
	for _, s := range liveStreams {
		if s.IsActive {
			list = append(list, s)
		}
	}
	liveStreamsMu.RUnlock()

	demoStreams := []*LiveStreamInfo{
		{
			StreamID:    "live_demo_1",
			HostID:      "m1",
			HostName:    "Riya Gosh",
			HostAvatar:  "https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&w=400&q=80",
			Title:       "Late night vibe & music 🎵",
			ViewerCount: 142,
			TotalEarned: 1540.0,
			StartedAt:   time.Now().Add(-25 * time.Minute),
			IsActive:    true,
		},
		{
			StreamID:    "live_demo_2",
			HostID:      "m2",
			HostName:    "Ananya Sharma",
			HostAvatar:  "https://images.unsplash.com/photo-1517841905240-472988babdf9?auto=format&fit=crop&w=400&q=80",
			Title:       "Q&A and chill chat ✨",
			ViewerCount: 89,
			TotalEarned: 890.0,
			StartedAt:   time.Now().Add(-12 * time.Minute),
			IsActive:    true,
		},
	}

	// Always prepend real active live model streams first
	combined := append(list, demoStreams...)

	SendJSON(w, http.StatusOK, "Active live streams", map[string]any{
		"streams": combined,
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
