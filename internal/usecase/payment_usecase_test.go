package usecase_test

import (
	"testing"

	"Connect/internal/domain"
	"Connect/internal/dto"
	"Connect/internal/mapper"
	"Connect/internal/repository/memory"
	"Connect/internal/usecase"
)

func TestPaymentOrderCreationAndSuccess(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	walletUC := usecase.NewWalletUseCase(store.Wallets, m)
	paymentUC := usecase.NewPaymentUseCase(store.Payments, store.Wallets, m)

	userResp, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9888800001", Name: "PayTester", Role: domain.RoleUser})
	initialWallet, _ := walletUC.GetWallet(userResp.User.ID)
	initialBalance := initialWallet.Wallet.Balance // ₹50 welcome bonus

	// 1. Initiate Order
	orderResp, err := paymentUC.CreateOrder(userResp.User.ID, &dto.CreatePaymentOrderRequest{
		Amount:   100.0,
		Currency: "INR",
	})
	if err != nil {
		t.Fatalf("Failed to create payment order: %v", err)
	}

	if orderResp.Order.Status != domain.PaymentStatusInProgress {
		t.Fatalf("Expected order status 'inprogress', got '%s'", orderResp.Order.Status)
	}

	// 2. Gateway Success Callback
	callbackResp, err := paymentUC.ProcessCallback(&dto.ProcessPaymentCallbackRequest{
		PaymentID:        orderResp.Order.ID,
		GatewayPaymentID: "pay_gw_12345",
		Status:           domain.PaymentStatusSuccessful,
	})
	if err != nil {
		t.Fatalf("Payment callback failed: %v", err)
	}

	if callbackResp.Order.Status != domain.PaymentStatusSuccessful {
		t.Fatalf("Expected order status 'successful', got '%s'", callbackResp.Order.Status)
	}

	// 3. Verify Wallet Balance
	updatedWallet, _ := walletUC.GetWallet(userResp.User.ID)
	expectedBalance := initialBalance + 100.0
	if updatedWallet.Wallet.Balance != expectedBalance {
		t.Fatalf("Expected wallet balance ₹%.2f, got ₹%.2f", expectedBalance, updatedWallet.Wallet.Balance)
	}

	// 4. Verify Step-by-Step Timeline Audit Logs
	timelineResp, err := paymentUC.GetPaymentTimeline(orderResp.Order.ID)
	if err != nil {
		t.Fatalf("Failed to get payment timeline: %v", err)
	}

	if timelineResp.Count < 3 {
		t.Fatalf("Expected at least 3 audit steps, got %d", timelineResp.Count)
	}
}

func TestPaymentFailureAndRetryWithNewPaymentID(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	walletUC := usecase.NewWalletUseCase(store.Wallets, m)
	paymentUC := usecase.NewPaymentUseCase(store.Payments, store.Wallets, m)

	userResp, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9888800002", Name: "RetryTester", Role: domain.RoleUser})

	// 1. Create Initial Order
	orderResp, _ := paymentUC.CreateOrder(userResp.User.ID, &dto.CreatePaymentOrderRequest{
		Amount:   250.0,
		Currency: "INR",
	})
	failedOrderID := orderResp.Order.ID

	// 2. Gateway Fails
	failResp, err := paymentUC.ProcessCallback(&dto.ProcessPaymentCallbackRequest{
		PaymentID:    failedOrderID,
		Status:       domain.PaymentStatusFailed,
		ErrorCode:    "BANK_DECLINED",
		ErrorMessage: "Insufficient funds in bank account",
	})
	if err != nil {
		t.Fatalf("Failed to process fail callback: %v", err)
	}

	if failResp.Order.Status != domain.PaymentStatusFailed {
		t.Fatalf("Expected status 'failed', got '%s'", failResp.Order.Status)
	}

	// 3. RETRY: Create New Payment ID from Existing Failed Data
	retryResp, err := paymentUC.RetryPayment(failedOrderID, userResp.User.ID)
	if err != nil {
		t.Fatalf("Retry payment failed: %v", err)
	}

	newPaymentID := retryResp.Order.ID
	if newPaymentID == failedOrderID {
		t.Fatalf("Retry should create a NEW payment ID, but got same ID '%s'", newPaymentID)
	}

	if retryResp.Order.OriginalPaymentID != failedOrderID {
		t.Fatalf("Expected OriginalPaymentID '%s', got '%s'", failedOrderID, retryResp.Order.OriginalPaymentID)
	}

	if retryResp.Order.Amount != 250.0 {
		t.Fatalf("Expected amount ₹250.00 preserved on retry, got ₹%.2f", retryResp.Order.Amount)
	}

	if retryResp.Order.RetryCount != 1 {
		t.Fatalf("Expected retry_count = 1, got %d", retryResp.Order.RetryCount)
	}

	// 4. Successful callback on the new payment ID
	successResp, err := paymentUC.ProcessCallback(&dto.ProcessPaymentCallbackRequest{
		PaymentID:        newPaymentID,
		GatewayPaymentID: "pay_gw_retry_999",
		Status:           domain.PaymentStatusSuccessful,
	})
	if err != nil || successResp.Order.Status != domain.PaymentStatusSuccessful {
		t.Fatalf("Failed to complete retried payment: %v", err)
	}

	wallet, _ := walletUC.GetWallet(userResp.User.ID)
	if wallet.Wallet.Balance != 300.0 { // ₹50 welcome + ₹250 recharge
		t.Fatalf("Expected ₹300.00 balance, got ₹%.2f", wallet.Wallet.Balance)
	}

	// 5. Verify Parent Order Timeline contains retry event
	oldTimeline, _ := paymentUC.GetPaymentTimeline(failedOrderID)
	hasRetrySpawnLog := false
	for _, l := range oldTimeline.Logs {
		if l.EventName == "ORDER_RETRIED_NEW_ID_SPAWNED" {
			hasRetrySpawnLog = true
		}
	}
	if !hasRetrySpawnLog {
		t.Fatalf("Parent order timeline did not log ORDER_RETRIED_NEW_ID_SPAWNED")
	}
}

func TestPaymentRefundLifecycle(t *testing.T) {
	store := memory.NewMemoryStore()
	m := mapper.NewMapper()
	authUC := usecase.NewAuthUseCase(store.Users, store.Wallets, m)
	paymentUC := usecase.NewPaymentUseCase(store.Payments, store.Wallets, m)

	userResp, _ := authUC.RegisterOrLogin(&dto.RegisterRequest{Phone: "9888800003", Name: "RefundTester", Role: domain.RoleUser})

	orderResp, _ := paymentUC.CreateOrder(userResp.User.ID, &dto.CreatePaymentOrderRequest{Amount: 500.0})
	_, _ = paymentUC.ProcessCallback(&dto.ProcessPaymentCallbackRequest{
		PaymentID: orderResp.Order.ID,
		Status:    domain.PaymentStatusSuccessful,
	})

	// Initiate & Complete Refund
	refundResp, err := paymentUC.InitiateRefund(orderResp.Order.ID, "User requested cancellation")
	if err != nil {
		t.Fatalf("Refund failed: %v", err)
	}

	if refundResp.Order.Status != domain.PaymentStatusRefundDone {
		t.Fatalf("Expected status 'refund_done', got '%s'", refundResp.Order.Status)
	}

	if refundResp.Order.RefundID == "" {
		t.Fatalf("Expected non-empty RefundID")
	}

	timeline, _ := paymentUC.GetPaymentTimeline(orderResp.Order.ID)
	hasRefundLog := false
	for _, l := range timeline.Logs {
		if l.EventName == "REFUND_COMPLETED" {
			hasRefundLog = true
		}
	}
	if !hasRefundLog {
		t.Fatalf("Timeline did not record REFUND_COMPLETED")
	}
}
