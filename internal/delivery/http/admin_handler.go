package http

import (
	"net/http"
)

// HandleCleanMockData allows administrative deletion of mock/fake seed models from the database.
func (h *HTTPHandler) HandleCleanMockData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	secret := r.Header.Get("X-Admin-Secret")
	if secret == "" {
		secret = r.URL.Query().Get("secret")
	}

	if secret != "connect_admin_secret_2026" {
		SendError(w, http.StatusUnauthorized, "Unauthorized: Invalid or missing admin secret")
		return
	}

	if err := h.authUC.DeleteMockModels(); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, http.StatusOK, "Mock models and test seed accounts successfully deleted from database", map[string]interface{}{
		"deleted_models": []string{"model-1", "model-2", "model-3", "model-4", "user-1", "user-2", "user-3"},
		"status":         "clean",
	})
}
