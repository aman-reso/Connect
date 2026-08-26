package ws

import (
	"log"
	"net/http"

	"Connect/pkg/common"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Hub manages active WebSocket connections.
type Hub struct {
	clients *common.Registry[string, *Client]
	router  *Router
}

// NewHub creates a new streamlined WebSocket Hub.
func NewHub(router *Router) *Hub {
	return &Hub{
		clients: common.NewRegistry[string, *Client](),
		router:  router,
	}
}

// ServeWS upgrades the HTTP connection and starts client pumps.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("token")
	if userID == "" {
		userID = r.URL.Query().Get("user_id")
	}
	if userID == "" {
		http.Error(w, "Unauthorized: missing user token", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed: %v", err)
		return
	}
	client := &Client{Hub: h, Conn: conn, Send: make(chan []byte, 256), UserID: userID}
	h.clients.Set(userID, client)
	go client.WritePump()
	go client.ReadPump()
}

// UnregisterClient removes a client on disconnect.
func (h *Hub) UnregisterClient(c *Client) {
	h.clients.Delete(c.UserID)
}

// Dispatch delegates messages to the router.
func (h *Hub) Dispatch(client *Client, raw []byte) {
	h.router.Route(client, raw)
}

// SendToUser delivers a signal message to a specific user.
func (h *Hub) SendToUser(userID string, msg *SignalMessage) bool {
	if client, ok := h.clients.Get(userID); ok {
		SendToClient(client, msg)
		return true
	}
	return false
}

// BroadcastToUsers sends a message to multiple users.
func (h *Hub) BroadcastToUsers(userIDs []string, msg *SignalMessage) {
	for _, uid := range userIDs {
		h.SendToUser(uid, msg)
	}
}
