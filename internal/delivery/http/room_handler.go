package http

import (
	"encoding/json"
	"net/http"

	"Connect/internal/domain"
	"Connect/internal/dto"
)

// HandleRooms routes room listing and creation requests.
func (h *HTTPHandler) HandleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rooms, err := h.roomUC.ListRooms()
		if err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		SendJSON(w, http.StatusOK, "Active rooms retrieved", rooms)
	case http.MethodPost:
		var req dto.CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "host_default"
		}
		host := &domain.User{
			ID:   userID,
			Role: domain.RoleModel,
			Name: "Host",
		}
		room, err := h.roomUC.CreateRoom(host, req.Title, req.RatePerMin)
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error())
			return
		}
		SendJSON(w, http.StatusCreated, "Room created successfully", room)
	default:
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
