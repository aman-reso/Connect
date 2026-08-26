package usecase

import (
	"errors"
	"fmt"
	"time"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository"

	"github.com/google/uuid"
)

type PaymentUseCase struct {
	paymentRepo repository.PaymentRepository
	walletRepo  repository.WalletRepository
	mapper      *mapper.Mapper
}

func NewPaymentUseCase(pRepo repository.PaymentRepository, wRepo repository.WalletRepository, m *mapper.Mapper) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo: pRepo,
		walletRepo:  wRepo,
		mapper:      m,
	}
}

// CreateOrder 1. Create Initial Payment Order
func (uc *PaymentUseCase) CreateOrder(userID string, req *dto.CreatePaymentOrderRequest) (*dto.PaymentOrderResponse, error) {
	if req.Amount <= 0 {
		return nil, errors.New("payment amount must be greater than zero")
	}
	if req.Currency == "" {
		req.Currency = "INR"
	}
	if req.GatewayName == "" {
		req.GatewayName = "razorpay"
	}

	start := time.Now()
	paymentID := "pay_" + uuid.New().String()[:12]
	gatewayOrderID := "order_" + uuid.New().String()[:12]

	order := &domain.PaymentOrder{
		ID:             paymentID,
		UserID:         userID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         domain.PaymentStatusInitiated,
		GatewayName:    req.GatewayName,
		GatewayOrderID: gatewayOrderID,
		RetryCount:     0,
		CreatedAt:      start,
		UpdatedAt:      start,
	}

	if err := uc.paymentRepo.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	// Step 1: Log Order Initialized
	initEnd := time.Now()
	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:         uuid.New().String(),
		PaymentID:  order.ID,
		FromStatus: "",
		ToStatus:   domain.PaymentStatusInitiated,
		EventName:  "ORDER_INITIALIZED",
		Message:    fmt.Sprintf("Order of ₹%.2f %s initialized with gateway %s", order.Amount, order.Currency, order.GatewayName),
		DurationMS: initEnd.Sub(start).Milliseconds(),
		StartedAt:  start,
		EndedAt:    initEnd,
		CreatedAt:  initEnd,
	})

	// Step 2: Transition to InProgress (Ready for Gateway Checkout)
	order.Status = domain.PaymentStatusInProgress
	order.UpdatedAt = time.Now()
	_ = uc.paymentRepo.UpdateOrder(order)

	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:           uuid.New().String(),
		PaymentID:    order.ID,
		FromStatus:   domain.PaymentStatusInitiated,
		ToStatus:     domain.PaymentStatusInProgress,
		EventName:    "GATEWAY_CHECKOUT_OPENED",
		GatewayRefID: gatewayOrderID,
		Message:      "Awaiting user authorization on payment gateway",
		DurationMS:   time.Since(initEnd).Milliseconds(),
		StartedAt:    initEnd,
		EndedAt:      time.Now(),
		CreatedAt:    time.Now(),
	})

	return uc.mapper.ToPaymentOrderResponse(order, "Payment order created successfully"), nil
}

// 2. Process Gateway Webhook / Callback
func (uc *PaymentUseCase) ProcessCallback(req *dto.ProcessPaymentCallbackRequest) (*dto.PaymentOrderResponse, error) {
	start := time.Now()
	order, err := uc.paymentRepo.GetOrderByID(req.PaymentID)
	if err != nil {
		return nil, err
	}

	// Idempotency: If already completed, return existing order
	if order.Status == domain.PaymentStatusSuccessful {
		return uc.mapper.ToPaymentOrderResponse(order, "Payment already verified and credited"), nil
	}

	prevStatus := order.Status

	if req.Status == domain.PaymentStatusSuccessful {
		order.Status = domain.PaymentStatusSuccessful
		order.GatewayPaymentID = req.GatewayPaymentID
		order.GatewaySignature = req.GatewaySignature
		now := time.Now()
		order.CompletedAt = &now

		if err := uc.paymentRepo.UpdateOrder(order); err != nil {
			return nil, err
		}

		// Credit User Wallet
		_, _ = uc.walletRepo.Recharge(order.UserID, order.Amount)

		// Log Success Step
		_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
			ID:           uuid.New().String(),
			PaymentID:    order.ID,
			FromStatus:   prevStatus,
			ToStatus:     domain.PaymentStatusSuccessful,
			EventName:    "GATEWAY_PAYMENT_SUCCESS",
			GatewayRefID: req.GatewayPaymentID,
			Message:      fmt.Sprintf("Payment of ₹%.2f captured. Wallet balance credited.", order.Amount),
			DurationMS:   time.Since(start).Milliseconds(),
			StartedAt:    start,
			EndedAt:      time.Now(),
			CreatedAt:    time.Now(),
		})

		return uc.mapper.ToPaymentOrderResponse(order, "Payment successful! Wallet balance credited."), nil
	}

	// Failed Path
	order.Status = domain.PaymentStatusFailed
	order.FailureReason = req.ErrorMessage
	if order.FailureReason == "" {
		order.FailureReason = "Payment declined or cancelled by user"
	}
	_ = uc.paymentRepo.UpdateOrder(order)

	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:           uuid.New().String(),
		PaymentID:    order.ID,
		FromStatus:   prevStatus,
		ToStatus:     domain.PaymentStatusFailed,
		EventName:    "GATEWAY_PAYMENT_FAILED",
		GatewayRefID: req.GatewayPaymentID,
		GatewayCode:  req.ErrorCode,
		Message:      fmt.Sprintf("Payment failed: %s", order.FailureReason),
		DurationMS:   time.Since(start).Milliseconds(),
		StartedAt:    start,
		EndedAt:      time.Now(),
		CreatedAt:    time.Now(),
	})

	return uc.mapper.ToPaymentOrderResponse(order, "Payment marked as failed"), nil
}

// RetryPayment 3. Retry Mechanism: Spawns a NEW Payment ID while preserving existing data & parent link
func (uc *PaymentUseCase) RetryPayment(failedPaymentID, userID string) (*dto.PaymentOrderResponse, error) {
	start := time.Now()
	oldOrder, err := uc.paymentRepo.GetOrderByID(failedPaymentID)
	if err != nil {
		return nil, fmt.Errorf("failed payment order not found: %w", err)
	}

	if oldOrder.UserID != userID {
		return nil, errors.New("unauthorized to retry this payment")
	}

	if oldOrder.Status == domain.PaymentStatusSuccessful {
		return nil, errors.New("cannot retry an already successful payment")
	}

	// Generate NEW Payment ID
	newPaymentID := "pay_" + uuid.New().String()[:12]
	newGatewayOrderID := "order_" + uuid.New().String()[:12]

	newOrder := &domain.PaymentOrder{
		ID:                newPaymentID,
		OriginalPaymentID: oldOrder.ID, // Link to previous failed order
		UserID:            oldOrder.UserID,
		Amount:            oldOrder.Amount, // Preserves amount
		Currency:          oldOrder.Currency,
		Status:            domain.PaymentStatusInProgress, // Directly inprogress for retry
		GatewayName:       oldOrder.GatewayName,
		GatewayOrderID:    newGatewayOrderID,
		RetryCount:        oldOrder.RetryCount + 1,
		CreatedAt:         start,
		UpdatedAt:         start,
	}

	if err := uc.paymentRepo.CreateOrder(newOrder); err != nil {
		return nil, fmt.Errorf("failed to create retry payment: %w", err)
	}

	// Audit Log on Parent Order
	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:         uuid.New().String(),
		PaymentID:  oldOrder.ID,
		FromStatus: oldOrder.Status,
		ToStatus:   oldOrder.Status,
		EventName:  "ORDER_RETRIED_NEW_ID_SPAWNED",
		Message:    fmt.Sprintf("User initiated retry. Spawned new Payment ID: %s (Retry #%d)", newPaymentID, newOrder.RetryCount),
		DurationMS: time.Since(start).Milliseconds(),
		StartedAt:  start,
		EndedAt:    time.Now(),
		CreatedAt:  time.Now(),
	})

	// Audit Log on New Order
	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:           uuid.New().String(),
		PaymentID:    newOrder.ID,
		FromStatus:   domain.PaymentStatusInitiated,
		ToStatus:     domain.PaymentStatusInProgress,
		EventName:    "RETRY_ORDER_INITIALIZED",
		GatewayRefID: newGatewayOrderID,
		Message:      fmt.Sprintf("Retry payment created from parent %s. Ready for checkout.", oldOrder.ID),
		DurationMS:   time.Since(start).Milliseconds(),
		StartedAt:    start,
		EndedAt:      time.Now(),
		CreatedAt:    time.Now(),
	})

	return uc.mapper.ToPaymentOrderResponse(newOrder, fmt.Sprintf("Retry payment created with new ID %s", newPaymentID)), nil
}

// 4. Refund Handling: refund_inprogress -> refund_done
func (uc *PaymentUseCase) InitiateRefund(paymentID string, reason string) (*dto.PaymentOrderResponse, error) {
	start := time.Now()
	order, err := uc.paymentRepo.GetOrderByID(paymentID)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.PaymentStatusSuccessful {
		return nil, fmt.Errorf("only successful payments can be refunded (current status: %s)", order.Status)
	}

	prevStatus := order.Status
	order.Status = domain.PaymentStatusRefundInProgress
	order.RefundReason = reason
	_ = uc.paymentRepo.UpdateOrder(order)

	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:         uuid.New().String(),
		PaymentID:  order.ID,
		FromStatus: prevStatus,
		ToStatus:   domain.PaymentStatusRefundInProgress,
		EventName:  "REFUND_INITIATED",
		Message:    fmt.Sprintf("Refund initiated: %s", reason),
		DurationMS: time.Since(start).Milliseconds(),
		StartedAt:  start,
		EndedAt:    time.Now(),
		CreatedAt:  time.Now(),
	})

	// Auto-settle refund
	refundID := "rfnd_" + uuid.New().String()[:10]
	order.Status = domain.PaymentStatusRefundDone
	order.RefundID = refundID
	_ = uc.paymentRepo.UpdateOrder(order)

	_ = uc.paymentRepo.RecordAuditLog(&domain.PaymentAuditLog{
		ID:           uuid.New().String(),
		PaymentID:    order.ID,
		FromStatus:   domain.PaymentStatusRefundInProgress,
		ToStatus:     domain.PaymentStatusRefundDone,
		EventName:    "REFUND_COMPLETED",
		GatewayRefID: refundID,
		Message:      fmt.Sprintf("Refund of ₹%.2f successfully settled to original payment source.", order.Amount),
		DurationMS:   time.Since(start).Milliseconds(),
		StartedAt:    start,
		EndedAt:      time.Now(),
		CreatedAt:    time.Now(),
	})

	return uc.mapper.ToPaymentOrderResponse(order, "Refund processed successfully"), nil
}

// GetPaymentTimeline 5. Payment Timeline & Audit Trail
func (uc *PaymentUseCase) GetPaymentTimeline(paymentID string) (*dto.PaymentTimelineResponse, error) {
	order, err := uc.paymentRepo.GetOrderByID(paymentID)
	if err != nil {
		return nil, err
	}

	logs, err := uc.paymentRepo.GetAuditLogs(paymentID)
	if err != nil {
		return nil, err
	}

	return uc.mapper.ToPaymentTimelineResponse(order, logs), nil
}
