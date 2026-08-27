package ws

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"Connect/internal/domain"
	"Connect/internal/usecase"
	"Connect/pkg/common"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ActiveCallSession tracks live 1:1 call state and the billing ticker.
type ActiveCallSession struct {
	CallID         string
	CallerID       string
	ReceiverID     string
	RatePerMin     float64
	CallType       string
	StartedAt      time.Time
	StopTickerChan chan struct{}
	once           sync.Once
}

// Stop safely stops the billing ticker once.
func (s *ActiveCallSession) Stop() {
	s.once.Do(func() {
		close(s.StopTickerChan)
	})
}

// Hub manages active WebSocket connections, presence, and call sessions.
type Hub struct {
	clients     *common.Registry[string, *Client]
	router      *Router
	authUC      *usecase.AuthUseCase
	mu          sync.RWMutex
	activeCalls map[string]*ActiveCallSession // callID -> session
	userCalls   map[string]string             // userID -> callID
}

// NewHub creates a new streamlined WebSocket Hub with full call session state.
func NewHub(router *Router, authUC ...*usecase.AuthUseCase) *Hub {
	var auc *usecase.AuthUseCase
	if len(authUC) > 0 {
		auc = authUC[0]
	}
	return &Hub{
		clients:     common.NewRegistry[string, *Client](),
		router:      router,
		authUC:      auc,
		activeCalls: make(map[string]*ActiveCallSession),
		userCalls:   make(map[string]string),
	}
}

// SetAuthUseCase injects the auth usecase if configured after router construction.
func (h *Hub) SetAuthUseCase(authUC *usecase.AuthUseCase) {
	h.authUC = authUC
}

// ServeWS upgrades the HTTP connection and starts client pumps.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.URL.Query().Get("user_id")
	}
	if token == "" {
		http.Error(w, "Unauthorized: missing user token", http.StatusUnauthorized)
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	var user *domain.User
	userID := token

	if h.authUC != nil {
		if u, err := h.authUC.ValidateToken(token); err == nil && u != nil {
			user = u
			userID = u.ID
		} else if u, err := h.authUC.GetModelByID(token); err == nil && u != nil {
			user = &domain.User{ID: u.ID, Name: u.Name, Role: domain.UserRole(u.Role), AvatarURL: u.AvatarURL}
			userID = u.ID
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed: %v", err)
		return
	}

	// Single Device Eviction: Terminate previous connection for this userID if present
	if oldClient, ok := h.clients.Get(userID); ok {
		SendToClient(oldClient, &SignalMessage{
			Type:   TypeSessionTerminated,
			Reason: "Logged in from another device/session.",
		})
		_ = oldClient.Conn.Close()
		h.clients.Delete(userID)
	}

	client := &Client{Hub: h, Conn: conn, Send: make(chan []byte, 256), UserID: userID, User: user}
	h.clients.Set(userID, client)

	log.Printf("🔌 WS Client connected: UserID=%s", userID)
	if h.authUC != nil {
		_ = h.authUC.SetPresence(userID, true, false)
	}
	h.BroadcastPresence(userID, true)

	go client.WritePump()
	go client.ReadPump()
}

// UnregisterClient removes a client on disconnect.
func (h *Hub) UnregisterClient(c *Client) {
	h.clients.Delete(c.UserID)
	log.Printf("🔌 WS Client disconnected: UserID=%s", c.UserID)

	if h.authUC != nil {
		_ = h.authUC.SetPresence(c.UserID, false, false)
	}

	// Clean up user call if active
	h.mu.RLock()
	callID, inCall := h.userCalls[c.UserID]
	h.mu.RUnlock()

	if inCall && callID != "" {
		// Forward disconnect termination
		h.mu.RLock()
		session, ok := h.activeCalls[callID]
		h.mu.RUnlock()
		if ok && session != nil {
			peerID := session.ReceiverID
			if c.UserID == session.ReceiverID {
				peerID = session.CallerID
			}
			h.SendToUser(peerID, &SignalMessage{
				Type:     TypeCallEnded,
				CallID:   callID,
				CallerID: c.UserID,
				Reason:   "Peer disconnected",
			})
			h.EndCallSession(callID)
		}
	}

	h.BroadcastPresence(c.UserID, false)
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

// IsUserOnline returns true if user has an active WS connection.
func (h *Hub) IsUserOnline(userID string) bool {
	_, ok := h.clients.Get(userID)
	return ok
}

// BroadcastPresence notifies all connected clients of presence updates.
func (h *Hub) BroadcastPresence(userID string, isOnline bool) {
	msg := &SignalMessage{
		Type:     TypePresenceUpdate,
		CallerID: userID,
		Reason:   fmt.Sprintf("%v", isOnline),
	}
	h.clients.Range(func(uid string, client *Client) bool {
		SendToClient(client, msg)
		return true
	})
}

// RegisterCallSession stores active call session state.
func (h *Hub) RegisterCallSession(session *ActiveCallSession) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeCalls[session.CallID] = session
	h.userCalls[session.CallerID] = session.CallID
	h.userCalls[session.ReceiverID] = session.CallID
}

// GetCallSession retrieves an active call session by callID.
func (h *Hub) GetCallSession(callID string) (*ActiveCallSession, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.activeCalls[callID]
	return s, ok
}

// GetUserCallID returns the call ID the user is currently participating in.
func (h *Hub) GetUserCallID(userID string) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cid, ok := h.userCalls[userID]
	return cid, ok
}

// EndCallSession terminates and removes an active call session.
func (h *Hub) EndCallSession(callID string) (*ActiveCallSession, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, exists := h.activeCalls[callID]
	if exists && session != nil {
		session.Stop()
		delete(h.activeCalls, callID)
		delete(h.userCalls, session.CallerID)
		delete(h.userCalls, session.ReceiverID)
	}
	return session, exists
}
