package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LiveKitTokenResponse represents the payload sent back to the Android client
type LiveKitTokenResponse struct {
	Token      string `json:"token"`
	LivekitURL string `json:"livekit_url"`
	RoomName   string `json:"room_name"`
}

// HandleCallToken generates a LiveKit participant token for voice or video calls.
// POST /api/calls/token
// Header: Authorization: Bearer <app_jwt>
// Body: { "remote_user_id": "<target_id>", "call_type": "video" | "voice" }
func (h *HTTPHandler) HandleCallToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	user, err := h.authUC.ValidateToken(tokenStr)
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	var req struct {
		RemoteUserID string `json:"remote_user_id"`
		CallType     string `json:"call_type"` // "voice" or "video"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RemoteUserID == "" {
		SendError(w, http.StatusBadRequest, "remote_user_id is required")
		return
	}

	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	livekitURL := os.Getenv("LIVEKIT_URL")

	if apiKey == "" {
		apiKey = "APImr59LGqwEVuj"
	}
	if apiSecret == "" {
		apiSecret = "cvdsoq3pKQusl4HfAHPxSeGXvHcM5atVOWQ2WozyxF2"
	}
	if livekitURL == "" {
		livekitURL = "wss://connecto-7sxi06vp.livekit.cloud"
	}

	roomName := buildRoomName(user.ID, req.RemoteUserID)
	token, err := createLiveKitToken(apiKey, apiSecret, user.ID, roomName)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to generate call token: "+err.Error())
		return
	}

	SendJSON(w, http.StatusOK, "Call token generated successfully", LiveKitTokenResponse{
		Token:      token,
		LivekitURL: livekitURL,
		RoomName:   roomName,
	})
}

// buildRoomName ensures deterministic room names for two participants
func buildRoomName(userA, userB string) string {
	if userA < userB {
		return fmt.Sprintf("call_%s_%s", userA, userB)
	}
	return fmt.Sprintf("call_%s_%s", userB, userA)
}

// createLiveKitToken generates a signed JWT compatible with LiveKit SFU
func createLiveKitToken(apiKey, apiSecret, identity, roomName string) (string, error) {
	type VideoGrant struct {
		RoomJoin     bool   `json:"roomJoin"`
		Room         string `json:"room"`
		CanPublish   bool   `json:"canPublish"`
		CanSubscribe bool   `json:"canSubscribe"`
	}

	claims := jwt.MapClaims{
		"iss":  apiKey,
		"sub":  identity,
		"name": identity,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(6 * time.Hour).Unix(),
		"nbf":  time.Now().Unix(),
		"video": VideoGrant{
			RoomJoin:     true,
			Room:         roomName,
			CanPublish:   true,
			CanSubscribe: true,
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(apiSecret))
}
