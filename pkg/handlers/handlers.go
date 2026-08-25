package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"Connect/pkg/models"
	"Connect/pkg/store"
)

type APIHandler struct {
	store *store.Store
}

func NewAPIHandler(st *store.Store) *APIHandler {
	return &APIHandler{store: st}
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{"error": message})
}

// 1. Auth: Register / Login
func (h *APIHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Phone string          `json:"phone"`
		Name  string          `json:"name"`
		Role  models.UserRole `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Phone == "" {
		sendError(w, http.StatusBadRequest, "Phone number is required")
		return
	}
	if req.Name == "" {
		req.Name = "User_" + req.Phone[len(req.Phone)-4:]
	}
	if req.Role == "" {
		req.Role = models.RoleUser
	}

	user, token, isNew := h.store.CreateOrLoginUser(req.Phone, req.Name, req.Role)
	wallet, _ := h.store.GetWallet(user.ID)

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"user":        user,
		"token":       token,
		"is_new_user": isNew,
		"wallet":      wallet,
		"message": func() string {
			if isNew && user.Role == models.RoleUser {
				return "Welcome! ₹50 bonus incentive added to your wallet!"
			}
			return "Login successful"
		}(),
	})
}

// 2. Models Directory Listing
func (h *APIHandler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	modelsList := h.store.ListModels()
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count":  len(modelsList),
		"models": modelsList,
	})
}

// 3. Group Audio Rooms
func (h *APIHandler) HandleRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		rooms := h.store.ListActiveRooms()
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"count": len(rooms),
			"rooms": rooms,
		})
		return
	}

	if r.Method == http.MethodPost {
		user := h.authenticate(r)
		if user == nil {
			sendError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		if user.Role != models.RoleModel {
			sendError(w, http.StatusForbidden, "Only models can host a group room")
			return
		}

		var req struct {
			Title      string  `json:"title"`
			RatePerMin float64 `json:"rate_per_min"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		room, err := h.store.CreateGroupRoom(user, req.Title, req.RatePerMin)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Group Audio Room created",
			"room":    room,
		})
		return
	}

	sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// 4. User Wallet & Balance
func (h *APIHandler) HandleWallet(w http.ResponseWriter, r *http.Request) {
	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if r.Method == http.MethodGet {
		wallet, err := h.store.GetWallet(user.ID)
		if err != nil {
			sendError(w, http.StatusNotFound, "Wallet not found")
			return
		}
		transactions := h.store.GetTransactions(user.ID)

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"wallet":       wallet,
			"transactions": transactions,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Amount float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
			sendError(w, http.StatusBadRequest, "Invalid recharge amount")
			return
		}

		wallet, err := h.store.RechargeWallet(user.ID, req.Amount)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Recharge successful",
			"wallet":  wallet,
		})
		return
	}

	sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// 5. Audio Call History (ZERO AUDIO STORED - METADATA ONLY)
func (h *APIHandler) HandleCallHistory(w http.ResponseWriter, r *http.Request) {
	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	calls := h.store.GetUserCallHistory(user.ID)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(calls),
		"calls": calls,
		"privacy_notice": "Calls are End-to-End Encrypted. No audio recordings or voice files are stored on any servers.",
	})
}

// 6. Ephemeral Messages for a partner
func (h *APIHandler) HandleChatMessages(w http.ResponseWriter, r *http.Request) {
	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	partnerID := r.URL.Query().Get("partner_id")
	if partnerID == "" {
		sendError(w, http.StatusBadRequest, "partner_id query parameter required")
		return
	}

	messages := h.store.GetEphemeralMessages(user.ID, partnerID)
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"partner_id": partnerID,
		"messages":   messages,
		"notice":     "Messages are ephemeral and automatically self-destruct within 60s of delivery.",
	})
}

func (h *APIHandler) authenticate(r *http.Request) *models.User {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		token := r.URL.Query().Get("token")
		if token != "" {
			u, err := h.store.GetUserByToken(token)
			if err == nil {
				return u
			}
		}
		return nil
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	u, err := h.store.GetUserByToken(token)
	if err != nil {
		return nil
	}
	return u
}
