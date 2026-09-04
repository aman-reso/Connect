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
	targetUserID := msg.GetTargetUserID()
	if targetUserID != "" {
		msg.ToUserID = targetUserID
		msg.ReceiverID = targetUserID
		if msg.FromUserID == "" && client != nil && client.UserID != "" {
			msg.FromUserID = client.UserID
			msg.CallerID = client.UserID
		}
		h.sender.SendToUser(targetUserID, msg)
	}
	ws.SendToClient(client, &ws.SignalMessage{
		Type:      ws.TypeChatReceived,
		ToUserID:  targetUserID,
		Timestamp: msg.Timestamp,
	})
	return nil
}
