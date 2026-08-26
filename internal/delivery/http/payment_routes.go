package http

import (
	"encoding/json"
	"net/http"

	"Connect/internal/dto"
)

// HandleCreatePaymentOrder initiates a payment order.
func (h *HTTPHandler) HandleCreatePaymentOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req dto.CreatePaymentOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "user_default"
	}
	resp, err := h.paymentUC.CreateOrder(userID, &req)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Payment order created", resp)
}

// HandlePaymentCallback processes webhook payment callbacks.
func (h *HTTPHandler) HandlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req dto.ProcessPaymentCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	resp, err := h.paymentUC.ProcessCallback(&req)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Payment callback processed", resp)
}

// HandlePaymentRetry retries a failed payment order.
func (h *HTTPHandler) HandlePaymentRetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	orderID := r.URL.Query().Get("order_id")
	userID := r.URL.Query().Get("user_id")
	if orderID == "" {
		SendError(w, http.StatusBadRequest, "Missing order_id")
		return
	}
	if userID == "" {
		userID = "user_default"
	}
	resp, err := h.paymentUC.RetryPayment(orderID, userID)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Payment retried", resp)
}

// HandlePaymentRefund processes refunds.
func (h *HTTPHandler) HandlePaymentRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	type RefundReq struct {
		PaymentID string `json:"payment_id"`
		Reason    string `json:"reason"`
	}
	var req RefundReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	resp, err := h.paymentUC.InitiateRefund(req.PaymentID, req.Reason)
	if err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Refund processed", resp)
}

// HandlePaymentTimeline gets audit timeline.
func (h *HTTPHandler) HandlePaymentTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	orderID := r.URL.Query().Get("order_id")
	if orderID == "" {
		SendError(w, http.StatusBadRequest, "Missing order_id")
		return
	}
	timeline, err := h.paymentUC.GetPaymentTimeline(orderID)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, http.StatusOK, "Timeline retrieved", timeline)
}
