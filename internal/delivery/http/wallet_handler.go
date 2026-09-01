package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"Connect/internal/dto"
)

// HandleWallet routes wallet retrieval and deposit requests.
func (h *HTTPHandler) HandleWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")

	// Also extract user from token if available
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if userID == "" && token != "" {
		if u, err := h.authUC.ValidateToken(token); err == nil && u != nil {
			userID = u.ID
		}
	}

	if userID == "" {
		SendError(w, http.StatusBadRequest, "Authentication token or user_id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
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
		if req.Amount <= 0 {
			req.Amount = 100 // default minimum direct top-up
		}
		walletRes, err := h.walletUC.Recharge(userID, req.Amount)
		if err != nil {
			SendError(w, http.StatusBadRequest, err.Error())
			return
		}
		SendJSON(w, http.StatusOK, "Recharge successful", walletRes.Wallet)
	default:
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// HandleWalletPackages returns available top-up packages.
func (h *HTTPHandler) HandleWalletPackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	packs := h.walletUC.GetWalletPacks()
	SendJSON(w, http.StatusOK, "Wallet packs fetched successfully", packs)
}
