package handlers

import (
	"fmt"
	"log"
	"time"

	"Connect/internal/delivery/ws"
	"Connect/internal/domain"
	"Connect/internal/usecase"
)

// CallHandler handles 1:1 call requests, balance enforcement, live billing, and termination.
type CallHandler struct {
	hub      *ws.Hub
	callUC   *usecase.CallUseCase
	walletUC *usecase.WalletUseCase
}

// NewCallHandler creates a new CallHandler instance.
func NewCallHandler(hub *ws.Hub, callUC *usecase.CallUseCase, walletUC *usecase.WalletUseCase) *CallHandler {
	return &CallHandler{
		hub:      hub,
		callUC:   callUC,
		walletUC: walletUC,
	}
}

// SupportedTypes returns the signaling types for 1:1 calls.
func (h *CallHandler) SupportedTypes() []string {
	return []string{
		ws.TypeCallRequest,
		ws.TypeCallAccept,
		ws.TypeCallReject,
		ws.TypeCallEnd,
	}
}

// Handle processes incoming 1:1 call events.
func (h *CallHandler) Handle(client *ws.Client, msg *ws.SignalMessage) error {
	switch msg.Type {
	case ws.TypeCallRequest:
		h.handleCallRequest(client, msg)
	case ws.TypeCallAccept:
		h.handleCallAccept(client, msg)
	case ws.TypeCallReject:
		h.handleCallReject(client, msg)
	case ws.TypeCallEnd:
		h.handleCallEnd(client, msg)
	}
	return nil
}

func (h *CallHandler) handleCallRequest(client *ws.Client, msg *ws.SignalMessage) {
	receiverID := msg.GetTargetUserID()
	if receiverID == "" {
		ws.SendToClient(client, &ws.SignalMessage{
			Type:   ws.TypeCallOffline,
			Reason: "Target User ID missing",
		})
		return
	}

	callType := msg.CallType
	if callType == "" {
		callType = "voice"
	}

	// 1. Mandatory Pre-Call Balance Check
	check, err := h.callUC.CheckCallBalance(client.UserID, receiverID, callType)
	if err != nil || !check.CanCall {
		reason := "Insufficient balance for call"
		var minReq, bal float64
		if check != nil {
			reason = check.Message
			minReq = check.MinRequired
			bal = check.Balance
		}
		log.Printf("⛔ Call blocked (Insufficient Balance) Caller=%s, Model=%s, Balance=%.2f, Required=%.2f",
			client.UserID, receiverID, bal, minReq)

		ws.SendToClient(client, &ws.SignalMessage{
			Type:         ws.TypeInsufficientBalance,
			CallerID:     client.UserID,
			ReceiverID:   receiverID,
			RatePerMin:   minReq,
			Balance:      bal,
			Reason:       reason,
			RemainingSec: 0,
		})
		return
	}

	// 2. Check Receiver Online Presence in WebSocket Hub
	if !h.hub.IsUserOnline(receiverID) {
		ws.SendToClient(client, &ws.SignalMessage{
			Type:       ws.TypeCallOffline,
			CallerID:   client.UserID,
			ReceiverID: receiverID,
			Reason:     fmt.Sprintf("%s is currently offline", check.ModelName),
		})
		return
	}

	// 3. Check if Receiver or Caller is Busy on another call
	if _, isBusy := h.hub.GetUserCallID(receiverID); isBusy {
		ws.SendToClient(client, &ws.SignalMessage{
			Type:       ws.TypeCallBusy,
			CallerID:   client.UserID,
			ReceiverID: receiverID,
			Reason:     fmt.Sprintf("%s is currently busy on another call", check.ModelName),
		})
		return
	}

	// 4. Create Call Record in Database/Repository
	callerUser := client.User
	if callerUser == nil {
		callerUser = &domain.User{ID: client.UserID, Name: "User_" + client.UserID}
	}
	record, err := h.callUC.InitiateCall(callerUser, receiverID, callType)
	if err != nil {
		ws.SendToClient(client, &ws.SignalMessage{
			Type:   ws.TypeInsufficientBalance,
			Reason: err.Error(),
		})
		return
	}

	// 5. Register Active Session in Hub
	session := &ws.ActiveCallSession{
		CallID:         record.ID,
		CallerID:       record.CallerID,
		ReceiverID:     record.ReceiverID,
		RatePerMin:     record.RatePerMin,
		CallType:       record.CallType,
		StartedAt:      time.Now(),
		StopTickerChan: make(chan struct{}),
	}
	h.hub.RegisterCallSession(session)

	// 6. Deliver INCOMING_CALL to Receiver
	log.Printf("📞 Routing INCOMING_CALL %s (%s) -> %s (Rate: ₹%.2f/min, CallID=%s)",
		client.UserID, callerUser.Name, receiverID, record.RatePerMin, record.ID)

	h.hub.SendToUser(receiverID, &ws.SignalMessage{
		Type:         ws.TypeIncomingCall,
		CallID:       record.ID,
		CallerID:     client.UserID,
		CallerName:   callerUser.Name,
		CallerAvatar: callerUser.AvatarURL,
		ReceiverID:   receiverID,
		CallType:     record.CallType,
		RatePerMin:   record.RatePerMin,
		Rate:         record.RatePerMin,
		DurationSec:  0,
	})
}

func (h *CallHandler) handleCallAccept(client *ws.Client, msg *ws.SignalMessage) {
	callID := msg.GetCallID()
	if callID == "" {
		if cid, ok := h.hub.GetUserCallID(client.UserID); ok {
			callID = cid
		}
	}

	if callID == "" {
		return
	}

	record, err := h.callUC.AcceptCall(callID)
	if err != nil {
		log.Printf("Failed to accept call %s: %v", callID, err)
		return
	}

	session, ok := h.hub.GetCallSession(callID)
	if !ok || session == nil {
		session = &ws.ActiveCallSession{
			CallID:         record.ID,
			CallerID:       record.CallerID,
			ReceiverID:     record.ReceiverID,
			RatePerMin:     record.RatePerMin,
			CallType:       record.CallType,
			StartedAt:      time.Now(),
			StopTickerChan: make(chan struct{}),
		}
		h.hub.RegisterCallSession(session)
	} else {
		session.StartedAt = time.Now()
	}

	log.Printf("🟢 Call Connected: %s (Caller=%s, Receiver=%s, Rate=₹%.2f/min)",
		callID, session.CallerID, session.ReceiverID, session.RatePerMin)

	activeMsg := &ws.SignalMessage{
		Type:        ws.TypeCallActive,
		CallID:      callID,
		CallerID:    session.CallerID,
		ReceiverID:  session.ReceiverID,
		CallType:    session.CallType,
		RatePerMin:  session.RatePerMin,
		Rate:        session.RatePerMin,
		DurationSec: 0,
	}

	h.hub.SendToUser(session.CallerID, activeMsg)
	h.hub.SendToUser(session.ReceiverID, activeMsg)

	// Launch live real-time billing ticker
	go h.startCallTicker(session)
}

func (h *CallHandler) handleCallReject(client *ws.Client, msg *ws.SignalMessage) {
	callID := msg.GetCallID()
	if callID == "" {
		if cid, ok := h.hub.GetUserCallID(client.UserID); ok {
			callID = cid
		}
	}

	if callID != "" {
		_ = h.callUC.RejectCall(callID)
		session, _ := h.hub.EndCallSession(callID)
		if session != nil {
			h.hub.SendToUser(session.CallerID, &ws.SignalMessage{
				Type:       ws.TypeCallRejected,
				CallID:     callID,
				CallerID:   session.CallerID,
				ReceiverID: session.ReceiverID,
				Reason:     "Call was declined",
			})
		}
	}
}

func (h *CallHandler) handleCallEnd(client *ws.Client, msg *ws.SignalMessage) {
	callID := msg.GetCallID()
	if callID == "" {
		if cid, ok := h.hub.GetUserCallID(client.UserID); ok {
			callID = cid
		}
	}

	if callID != "" {
		h.endCall(callID, "Call ended by user", client.UserID)
	}
}

func (h *CallHandler) startCallTicker(session *ws.ActiveCallSession) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	warningSent := false

	for {
		select {
		case <-session.StopTickerChan:
			return
		case now := <-ticker.C:
			durationSec := int(now.Sub(session.StartedAt).Seconds())

			walletResp, err := h.walletUC.GetWallet(session.CallerID)
			if err != nil || walletResp == nil || walletResp.Wallet == nil {
				h.endCall(session.CallID, "Wallet verification error", "system")
				return
			}

			ratePerSec := session.RatePerMin / 60.0
			costSoFar := float64(durationSec) * ratePerSec
			remainingBalance := walletResp.Wallet.Balance - costSoFar

			remainingSec := 0
			if ratePerSec > 0 {
				remainingSec = int(remainingBalance / ratePerSec)
			}
			if remainingSec < 0 {
				remainingSec = 0
			}

			tickMsg := &ws.SignalMessage{
				Type:         ws.TypeCallTick,
				CallID:       session.CallID,
				CallerID:     session.CallerID,
				ReceiverID:   session.ReceiverID,
				DurationSec:  int64(durationSec),
				RemainingSec: remainingSec,
				Cost:         costSoFar,
				TotalCost:    costSoFar,
				RatePerMin:   session.RatePerMin,
				Balance:      remainingBalance,
			}

			h.hub.SendToUser(session.CallerID, tickMsg)
			h.hub.SendToUser(session.ReceiverID, tickMsg)

			if remainingSec <= 30 && !warningSent {
				warningSent = true
				h.hub.SendToUser(session.CallerID, &ws.SignalMessage{
					Type:         ws.TypeBalanceLowWarning,
					CallID:       session.CallID,
					RemainingSec: remainingSec,
					Reason:       "Low balance! Call will auto-disconnect in less than 30 seconds.",
				})
			}

			// Automatic termination on balance exhaustion
			if remainingBalance <= 0 || remainingSec <= 0 {
				log.Printf("⚠️ Balance Exhausted for call %s (Caller: %s) -> Auto Terminating",
					session.CallID, session.CallerID)
				h.endCall(session.CallID, "Balance exhausted", "billing_engine")
				return
			}
		}
	}
}

func (h *CallHandler) endCall(callID, reason, triggeredBy string) {
	session, exists := h.hub.EndCallSession(callID)
	if !exists || session == nil {
		return
	}

	cost, durationSec, err := h.callUC.EndCall(callID, reason)
	if err != nil {
		log.Printf("Error settling call %s: %v", callID, err)
	}

	log.Printf("🔴 Call Settled: %s (Duration=%ds, Cost=₹%.2f, Reason='%s')",
		callID, durationSec, cost, reason)

	endMsgType := ws.TypeCallEnded
	if reason == "Balance exhausted" {
		endMsgType = ws.TypeBalanceExhausted
	}

	endMsg := &ws.SignalMessage{
		Type:        endMsgType,
		CallID:      callID,
		CallerID:    session.CallerID,
		ReceiverID:  session.ReceiverID,
		DurationSec: int64(durationSec),
		Cost:        cost,
		TotalCost:   cost,
		Reason:      reason,
	}

	h.hub.SendToUser(session.CallerID, endMsg)
	h.hub.SendToUser(session.ReceiverID, endMsg)
}

