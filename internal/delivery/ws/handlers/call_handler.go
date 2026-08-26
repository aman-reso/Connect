package handlers

import (
	"Connect/internal/delivery/ws"
	"Connect/internal/usecase"
)

// CallHandler handles 1:1 call requests, responses, and termination events.
type CallHandler struct {
	sender ws.MessageSender
	callUC *usecase.CallUseCase
}

// NewCallHandler creates a new CallHandler instance.
func NewCallHandler(sender ws.MessageSender, callUC *usecase.CallUseCase) *CallHandler {
	return &CallHandler{sender: sender, callUC: callUC}
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
		sent := h.sender.SendToUser(msg.ToUserID, &ws.SignalMessage{
			Type:       ws.TypeIncomingCall,
			FromUserID: client.UserID,
			CallType:   msg.CallType,
		})
		if !sent {
			ws.SendToClient(client, &ws.SignalMessage{
				Type:     ws.TypeCallOffline,
				ToUserID: msg.ToUserID,
			})
		}
	case ws.TypeCallAccept, ws.TypeCallReject, ws.TypeCallEnd:
		if msg.ToUserID != "" {
			h.sender.SendToUser(msg.ToUserID, msg)
		}
	}
	return nil
}
