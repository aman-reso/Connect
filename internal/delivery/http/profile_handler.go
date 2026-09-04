package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

type UpdateProfileRequest struct {
	Name      string `json:"name,omitempty"`
	Bio       string `json:"bio,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// HandleUserProfile manages fetching and updating authenticated user profile.
func (h *HTTPHandler) HandleUserProfile(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	user, err := h.authUC.ValidateToken(token)
	if err != nil {
		SendError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	if r.Method == http.MethodGet {
		SendJSON(w, http.StatusOK, "Profile retrieved successfully", user)
		return
	}

	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		var req UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		if req.Name != "" {
			user.Name = req.Name
		}
		if req.Bio != "" {
			user.Bio = req.Bio
		}
		if req.AvatarURL != "" {
			user.AvatarURL = req.AvatarURL
		}
		SendJSON(w, http.StatusOK, "Profile updated successfully", user)
		return
	}

	SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// HandleUserBusyStatus allows users or models to set their status as busy or available.
// POST /api/user/busy
// Body: {"is_busy": true|false}
func (h *HTTPHandler) HandleUserBusyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	user, err := h.authUC.ValidateToken(token)
	if err != nil || user == nil {
		SendError(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	var req struct {
		IsBusy bool `json:"is_busy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	_ = h.callUC.SetPresence(user.ID, true, req.IsBusy)
	user.IsBusy = req.IsBusy
	SendJSON(w, http.StatusOK, "Busy status updated successfully", map[string]any{
		"user_id": user.ID,
		"is_busy": req.IsBusy,
	})
}
