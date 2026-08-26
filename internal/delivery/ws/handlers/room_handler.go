package handlers

import (
	"sync"

	"Connect/internal/delivery/ws"
	"Connect/internal/usecase"
)

// RoomHandler handles group room participant joins, leaves, and notifications.
type RoomHandler struct {
	sender  ws.MessageSender
	roomUC  *usecase.RoomUseCase
	mu      sync.RWMutex
	members map[string]map[string]bool
}

// NewRoomHandler creates a new RoomHandler.
func NewRoomHandler(sender ws.MessageSender, roomUC *usecase.RoomUseCase) *RoomHandler {
	return &RoomHandler{
		sender:  sender,
		roomUC:  roomUC,
		members: make(map[string]map[string]bool),
	}
}

// SupportedTypes returns supported room message types.
func (h *RoomHandler) SupportedTypes() []string {
	return []string{ws.TypeGroupJoin, ws.TypeGroupLeave}
}

// Handle manages group room state transitions.
func (h *RoomHandler) Handle(client *ws.Client, msg *ws.SignalMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if msg.RoomID == "" {
		return nil
	}
	if msg.Type == ws.TypeGroupJoin {
		if _, ok := h.members[msg.RoomID]; !ok {
			h.members[msg.RoomID] = make(map[string]bool)
		}
		h.members[msg.RoomID][client.UserID] = true
	} else if msg.Type == ws.TypeGroupLeave {
		if _, ok := h.members[msg.RoomID]; ok {
			delete(h.members[msg.RoomID], client.UserID)
		}
	}
	return nil
}
