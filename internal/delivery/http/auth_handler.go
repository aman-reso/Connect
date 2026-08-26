package http

import (
	"encoding/json"
	"net/http"

	"Connect/internal/dto"
)

// HandleAuth processes user registration and authentication requests.
func (h *HTTPHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	resp, err := h.authUC.RegisterOrLogin(&req)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Authentication successful", resp)
}
