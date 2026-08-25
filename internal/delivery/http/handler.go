package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/usecase"
)

type HTTPHandler struct {
	authUC     *usecase.AuthUseCase
	walletUC   *usecase.WalletUseCase
	callUC     *usecase.CallUseCase
	roomUC     *usecase.RoomUseCase
	paymentUC  *usecase.PaymentUseCase
	onboardUC  *usecase.ModelOnboardingUseCase
	reportUC   *usecase.ReportUseCase
	favoriteUC *usecase.FavoriteUseCase
}

func NewHTTPHandler(auth *usecase.AuthUseCase, wallet *usecase.WalletUseCase, call *usecase.CallUseCase, room *usecase.RoomUseCase, payment *usecase.PaymentUseCase, onboard *usecase.ModelOnboardingUseCase, report *usecase.ReportUseCase, favorite *usecase.FavoriteUseCase) *HTTPHandler {
	return &HTTPHandler{
		authUC:     auth,
		walletUC:   wallet,
		callUC:     call,
		roomUC:     room,
		paymentUC:  payment,
		onboardUC:  onboard,
		reportUC:   report,
		favoriteUC: favorite,
	}
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, msg string) {
	sendJSON(w, status, map[string]string{"error": msg})
}

// 1. Auth Endpoint
func (h *HTTPHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.authUC.RegisterOrLogin(&req)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

// 2. Models Listing
func (h *HTTPHandler) HandleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resp, err := h.authUC.ListModels()
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

// 3. Group Audio Rooms
func (h *HTTPHandler) HandleRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		resp, err := h.roomUC.ListRooms()
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sendJSON(w, http.StatusOK, resp)
		return
	}

	if r.Method == http.MethodPost {
		user := h.authenticate(r)
		if user == nil {
			sendError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		var req dto.CreateRoomRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		room, err := h.roomUC.CreateRoom(user, req.Title, req.RatePerMin)
		if err != nil {
			sendError(w, http.StatusBadRequest, err.Error())
			return
		}
		sendJSON(w, http.StatusOK, map[string]interface{}{"message": "Room created", "room": room})
		return
	}

	sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// 4. Wallet Endpoint
func (h *HTTPHandler) HandleWallet(w http.ResponseWriter, r *http.Request) {
	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if r.Method == http.MethodGet {
		resp, err := h.walletUC.GetWallet(user.ID)
		if err != nil {
			sendError(w, http.StatusNotFound, err.Error())
			return
		}
		sendJSON(w, http.StatusOK, resp)
		return
	}

	if r.Method == http.MethodPost {
		var req dto.RechargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
			sendError(w, http.StatusBadRequest, "Invalid recharge amount")
			return
		}

		resp, err := h.walletUC.Recharge(user.ID, req.Amount)
		if err != nil {
			sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		sendJSON(w, http.StatusOK, resp)
		return
	}

	sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// 5. Call History Endpoint (METADATA ONLY - ZERO AUDIO RECORDINGS)
func (h *HTTPHandler) HandleHistory(w http.ResponseWriter, r *http.Request) {
	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	resp, err := h.callUC.GetHistory(user.ID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) authenticate(r *http.Request) *domain.User {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	token = strings.TrimPrefix(token, "Bearer ")

	user, err := h.authUC.ValidateToken(token)
	if err != nil {
		return nil
	}
	return user
}

// 6. Payment Endpoints
func (h *HTTPHandler) HandleCreatePaymentOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.CreatePaymentOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.paymentUC.CreateOrder(user.ID, &req)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req dto.ProcessPaymentCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.paymentUC.ProcessCallback(&req)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandlePaymentRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.RetryPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FailedPaymentID == "" {
		sendError(w, http.StatusBadRequest, "Failed payment ID is required for retry")
		return
	}

	resp, err := h.paymentUC.RetryPayment(req.FailedPaymentID, user.ID)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandlePaymentRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.InitiateRefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PaymentID == "" {
		sendError(w, http.StatusBadRequest, "Payment ID is required for refund")
		return
	}

	resp, err := h.paymentUC.InitiateRefund(req.PaymentID, req.Reason)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandlePaymentTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	paymentID := r.URL.Query().Get("payment_id")
	if paymentID == "" {
		sendError(w, http.StatusBadRequest, "payment_id query parameter is required")
		return
	}

	resp, err := h.paymentUC.GetPaymentTimeline(paymentID)
	if err != nil {
		sendError(w, http.StatusNotFound, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

// 7. Model Onboarding Endpoints
func (h *HTTPHandler) HandleModelOnboarding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.ModelOnboardingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.onboardUC.SubmitOnboarding(user, &req)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandleGetModelOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	resp, err := h.onboardUC.GetOnboardingStatus(user.ID)
	if err != nil {
		sendError(w, http.StatusNotFound, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

// 8. User Safety & Model Report Endpoints
func (h *HTTPHandler) HandleCreateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.reportUC.CreateReport(user, &req)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandleGetModelReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	modelID := r.URL.Query().Get("model_id")
	if modelID == "" {
		sendError(w, http.StatusBadRequest, "model_id parameter is required")
		return
	}

	resp, err := h.reportUC.GetReportsForModel(modelID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

// 8. Model Favorites Endpoints
func (h *HTTPHandler) HandleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req dto.ToggleFavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.favoriteUC.ToggleFavorite(user.ID, req.ModelID)
	if err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	resp, err := h.favoriteUC.GetFavorites(user.ID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, resp)
}

func (h *HTTPHandler) HandleGetFavoriteIDs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	user := h.authenticate(r)
	if user == nil {
		sendError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ids, err := h.favoriteUC.GetFavoriteIDs(user.ID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"count":        len(ids),
		"favorite_ids": ids,
	})
}

