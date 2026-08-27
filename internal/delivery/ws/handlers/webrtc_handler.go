package handlers

import (
	"log"

	"Connect/internal/delivery/ws"
)

// WebRTCHandler handles SDP offer, answer, and ICE candidate forwarding.
type WebRTCHandler struct {
	hub *ws.Hub
}

// NewWebRTCHandler creates a new WebRTCHandler.
func NewWebRTCHandler(hub *ws.Hub) *WebRTCHandler {
	return &WebRTCHandler{hub: hub}
}

// SupportedTypes returns the message types this handler processes.
func (h *WebRTCHandler) SupportedTypes() []string {
	return []string{
		ws.TypeWebRTCOffer,
		ws.TypeWebRTCAnswer,
		ws.TypeWebRTCICECandidate,
	}
}

// Handle forwards WebRTC negotiation frames directly to the target peer.
func (h *WebRTCHandler) Handle(client *ws.Client, msg *ws.SignalMessage) error {
	targetUserID := msg.GetTargetUserID()

	// If target ID is not directly in message, infer from active call session
	if targetUserID == "" {
		callID := msg.GetCallID()
		if callID == "" {
			if cid, ok := h.hub.GetUserCallID(client.UserID); ok {
				callID = cid
			}
		}
		if callID != "" {
			if session, ok := h.hub.GetCallSession(callID); ok && session != nil {
				if client.UserID == session.CallerID {
					targetUserID = session.ReceiverID
				} else {
					targetUserID = session.CallerID
				}
			}
		}
	}

	if targetUserID == "" {
		log.Printf("⚠️ WebRTC %s dropped: Target user ID unknown from sender %s", msg.Type, client.UserID)
		return nil
	}

	msg.CallerID = client.UserID
	msg.FromUserID = client.UserID
	msg.ReceiverID = targetUserID
	msg.ToUserID = targetUserID

	h.hub.SendToUser(targetUserID, msg)
	return nil
}

