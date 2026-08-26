package ws

import (
	"encoding/json"
	"log"
)

// MessageSender sends signaling messages to connected users.
type MessageSender interface {
	SendToUser(userID string, msg *SignalMessage) bool
	BroadcastToUsers(userIDs []string, msg *SignalMessage)
}

// SendToClient SendJSON encodes and delivers a message directly to a client channel.
func SendToClient(client *Client, msg *SignalMessage) {
	if client == nil {
		return
	}
	bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error serializing message: %v", err)
		return
	}
	select {
	case client.Send <- bytes:
	default:
		log.Printf("Send buffer full for user: %s", client.UserID)
	}
}
