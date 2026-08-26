package http

import (
	"encoding/json"
	"net/http"

	"Connect/internal/dto"
)

// HandleWallet routes wallet retrieval and deposit requests.
func (h *HTTPHandler) HandleWallet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			SendError(w, http.StatusBadRequest, "Missing user_id parameter")
			return
		}
		wallet, err := h.walletUC.GetWallet(userID)
		if err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		SendJSON(w, http.StatusOK, "Wallet retrieved successfully", wallet)
	case http.MethodPost:
		var req dto.RechargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			SendError(w, http.StatusBadRequest, "Missing user_id parameter")
			return
		}
		wallet, err := h.walletUC.Recharge(userID, req.Amount)
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error())
			return
		}
		SendJSON(w, http.StatusOK, "Recharge successful", wallet)
	default:
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
