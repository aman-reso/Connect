package handlers

import (
	"Connect/internal/delivery/ws"
)

// WebRTCHandler handles SDP offer, answer, and ICE candidate forwarding.
type WebRTCHandler struct {
	sender ws.MessageSender
}

// NewWebRTCHandler creates a new WebRTCHandler.
func NewWebRTCHandler(sender ws.MessageSender) *WebRTCHandler {
	return &WebRTCHandler{sender: sender}
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
	if msg.ToUserID == "" {
		return nil
	}
	h.sender.SendToUser(msg.ToUserID, msg)
	return nil
}
