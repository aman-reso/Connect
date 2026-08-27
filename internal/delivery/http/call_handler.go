package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"Connect/internal/domain"
	"Connect/internal/dto"
)

// HandleCallHistory retrieves call logs for a user.
func (h *HTTPHandler) HandleCallHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if userID == "" && token != "" {
		if u, err := h.authUC.ValidateToken(token); err == nil && u != nil {
			userID = u.ID
		}
	}

	if userID == "" {
		SendError(w, http.StatusBadRequest, "Missing user_id parameter or authorization token")
		return
	}
	calls, err := h.callUC.GetHistory(userID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Call history retrieved", calls)
}

// HandleHistory aliases HandleCallHistory for routing compatibility.
func (h *HTTPHandler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	h.HandleCallHistory(w, r)
}

// HandleCheckCallBalance validates if caller has sufficient balance to place a call to a model.
func (h *HTTPHandler) HandleCheckCallBalance(w http.ResponseWriter, r *http.Request) {
	callerID := r.URL.Query().Get("caller_id")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)
	if token != "" {
		if u, err := h.authUC.ValidateToken(token); err == nil && u != nil {
			callerID = u.ID
			if u.Role == domain.RoleModel {
				SendJSON(w, http.StatusOK, "Creators cannot place calls", &dto.CheckCallBalanceResponse{
					CanCall: false,
					Message: "Creator accounts cannot place outgoing calls to other creators.",
				})
				return
			}
		}
	}

	if callerID == "" {
		SendError(w, http.StatusUnauthorized, "Authentication token required")
		return
	}

	modelID := r.URL.Query().Get("model_id")
	callType := r.URL.Query().Get("call_type")
	if callType == "" {
		callType = "voice"
	}

	if r.Method == http.MethodPost {
		var req dto.CheckCallBalanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.ModelID != "" {
				modelID = req.ModelID
			}
			if req.CallType != "" {
				callType = req.CallType
			}
		}
	}

	if modelID == "" {
		SendError(w, http.StatusBadRequest, "model_id parameter is required")
		return
	}

	res, err := h.callUC.CheckCallBalance(callerID, modelID, callType)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}

	SendJSON(w, http.StatusOK, "Balance check completed", res)
}
