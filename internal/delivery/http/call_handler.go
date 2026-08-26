package http

import (
	"net/http"
)

// HandleCallHistory retrieves call logs for a user.
func (h *HTTPHandler) HandleCallHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		SendError(w, http.StatusBadRequest, "Missing user_id parameter")
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
