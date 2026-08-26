package http

import (
	"encoding/json"
	"net/http"

	"Connect/internal/dto"
)

// HandleToggleFavorite toggles a model's favorite status for a user.
func (h *HTTPHandler) HandleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req dto.ToggleFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "user_default"
	}
	resp, err := h.favoriteUC.ToggleFavorite(userID, req.ModelID)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Favorite status updated", resp)
}

// HandleGetFavorites retrieves favorite models for a given user.
func (h *HTTPHandler) HandleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		SendError(w, http.StatusBadRequest, "Missing user_id parameter")
		return
	}
	models, err := h.favoriteUC.GetFavorites(userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Favorites retrieved", models)
}

// HandleGetFavoriteIDs retrieves a slice of favorite model IDs for a user.
func (h *HTTPHandler) HandleGetFavoriteIDs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		SendError(w, http.StatusBadRequest, "Missing user_id parameter")
		return
	}
	ids, err := h.favoriteUC.GetFavoriteIDs(userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Favorite IDs retrieved", ids)
}
