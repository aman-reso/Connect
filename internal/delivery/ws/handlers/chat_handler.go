package handlers

import (
	"time"

	"Connect/internal/delivery/ws"
)

// ChatHandler handles ephemeral text chat signaling.
type ChatHandler struct {
	sender ws.MessageSender
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(sender ws.MessageSender) *ChatHandler {
	return &ChatHandler{sender: sender}
}

// SupportedTypes returns the message types for chat.
func (h *ChatHandler) SupportedTypes() []string {
	return []string{ws.TypeChatMessage}
}

// Handle forwards the chat message and acknowledges receipt.
func (h *ChatHandler) Handle(client *ws.Client, msg *ws.SignalMessage) error {
	msg.Timestamp = time.Now().Unix()
	if msg.ToUserID != "" {
		h.sender.SendToUser(msg.ToUserID, msg)
	}
	ws.SendToClient(client, &ws.SignalMessage{
		Type:      ws.TypeChatReceived,
		ToUserID:  msg.ToUserID,
		Timestamp: msg.Timestamp,
	})
	return nil
}
