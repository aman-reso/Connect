package ws

import (
	"encoding/json"
	"log"
)

// MessageHandler is the interface that all modular signaling handlers implement.
type MessageHandler interface {
	SupportedTypes() []string
	Handle(client *Client, msg *SignalMessage) error
}

// Router dispatches messages to registered handlers.
type Router struct {
	handlers map[string]MessageHandler
}

// NewRouter creates a new signaling router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]MessageHandler),
	}
}

// Register adds a handler for its supported message types.
func (r *Router) Register(h MessageHandler) {
	for _, msgType := range h.SupportedTypes() {
		r.handlers[msgType] = h
	}
}

// Route directs an incoming raw byte frame to the appropriate handler.
func (r *Router) Route(client *Client, raw []byte) {
	var msg SignalMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("Failed to unmarshal signaling message: %v", err)
		return
	}
	msg.FromUserID = client.UserID
	handler, exists := r.handlers[msg.Type]
	if !exists {
		log.Printf("No handler registered for message type: %s", msg.Type)
		return
	}
	if err := handler.Handle(client, &msg); err != nil {
		log.Printf("Handler error for %s: %v", msg.Type, err)
	}
}
