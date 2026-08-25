package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"Connect/internal/domain"
	"Connect/internal/repository"
	"Connect/internal/usecase"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Signaling message types
const (
	TypeCallRequest          = "CALL_REQUEST"
	TypeIncomingCall         = "INCOMING_CALL"
	TypeCallAccept           = "CALL_ACCEPT"
	TypeCallActive           = "CALL_ACTIVE"
	TypeCallReject           = "CALL_REJECT"
	TypeCallRejected         = "CALL_REJECTED"
	TypeCallEnd              = "CALL_END"
	TypeCallEnded            = "CALL_ENDED"
	TypeCallBusy             = "CALL_BUSY"
	TypeCallOffline          = "CALL_OFFLINE"
	TypeInsufficientBalance  = "CALL_INSUFFICIENT_BALANCE"
	TypeBalanceLowWarning    = "BALANCE_LOW_WARNING"
	TypeBalanceExhausted     = "CALL_ENDED_BALANCE_EXHAUSTED"
	TypeCallTick             = "CALL_TICK"
	TypePresenceUpdate       = "PRESENCE_UPDATE"
	TypeSessionTerminated    = "SESSION_TERMINATED"

	// Group Call
	TypeGroupCreate        = "GROUP_CREATE"
	TypeGroupJoin          = "GROUP_JOIN"
	TypeGroupLeave         = "GROUP_LEAVE"
	TypeGroupUserJoined    = "GROUP_USER_JOINED"
	TypeGroupUserLeft      = "GROUP_USER_LEFT"
	TypeGroupTick          = "GROUP_TICK"
	TypeGroupKickExhausted = "GROUP_KICK_BALANCE_EXHAUSTED"

	// WebRTC Forwarding
	TypeWebRTCOffer        = "WEBRTC_OFFER"
	TypeWebRTCAnswer       = "WEBRTC_ANSWER"
	TypeWebRTCICECandidate = "WEBRTC_ICE_CANDIDATE"

	// Ephemeral Chat
	TypeChatMessage  = "CHAT_MESSAGE"
	TypeChatReceived = "CHAT_RECEIVED"
	TypeChatError    = "CHAT_ERROR"
)

type WSMessage struct {
	Type         string          `json:"type"`
	CallID       string          `json:"call_id,omitempty"`
	RoomID       string          `json:"room_id,omitempty"`
	CallerID     string          `json:"caller_id,omitempty"`
	ReceiverID   string          `json:"receiver_id,omitempty"`
	RatePerMin   float64         `json:"rate_per_min,omitempty"`
	DurationSec  int             `json:"duration_sec,omitempty"`
	RemainingSec int             `json:"remaining_sec,omitempty"`
	Cost         float64         `json:"cost,omitempty"`
	Reason       string          `json:"reason,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
}

type Client struct {
	ID     string
	UserID string
	User   *domain.User
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
}

type ActiveCallSession struct {
	CallID         string
	CallerID       string
	ReceiverID     string
	RatePerMin     float64
	StartedAt      time.Time
	StopTickerChan chan struct{}
}

type Hub struct {
	authUC       *usecase.AuthUseCase
	callUC       *usecase.CallUseCase
	roomUC       *usecase.RoomUseCase
	walletRepo   repository.WalletRepository
	callRepo     repository.CallRepository
	mu           sync.RWMutex
	userClients  map[string]map[string]*Client
	activeCalls  map[string]*ActiveCallSession
	userCalls    map[string]string
	userRooms    map[string]string
}

func NewHub(auth *usecase.AuthUseCase, call *usecase.CallUseCase, room *usecase.RoomUseCase, wRepo repository.WalletRepository, cRepo repository.CallRepository) *Hub {
	return &Hub{
		authUC:      auth,
		callUC:      call,
		roomUC:      room,
		walletRepo:  wRepo,
		callRepo:    cRepo,
		userClients: make(map[string]map[string]*Client),
		activeCalls: make(map[string]*ActiveCallSession),
		userCalls:   make(map[string]string),
		userRooms:   make(map[string]string),
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	user, err := h.authUC.ValidateToken(token)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS Upgrade Error: %v", err)
		return
	}

	client := &Client{
		ID:     token,
		UserID: user.ID,
		User:   user,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    h,
	}

	h.registerClient(client)
	go client.writePump()
	go client.readPump()
}

func (h *Hub) registerClient(c *Client) {
	h.mu.Lock()

	// Single Device Session Enforcement: Evict previous device session
	if tabs, ok := h.userClients[c.UserID]; ok {
		for id, old := range tabs {
			termMsg := &WSMessage{
				Type:   TypeSessionTerminated,
				Reason: "You have been logged in from another active device. This session has been deactivated.",
			}
			data, _ := json.Marshal(termMsg)
			select {
			case old.Send <- data:
			default:
			}
			old.Conn.Close()
			delete(tabs, id)
		}
	} else {
		h.userClients[c.UserID] = make(map[string]*Client)
	}

	h.userClients[c.UserID][c.ID] = c
	h.mu.Unlock()

	log.Printf("Client connected: %s (%s)", c.User.Name, c.UserID)
	h.broadcastPresence(c.UserID, true)
}

func (h *Hub) unregisterClient(c *Client) {
	h.mu.Lock()
	if tabs, ok := h.userClients[c.UserID]; ok {
		delete(tabs, c.ID)
		if len(tabs) == 0 {
			delete(h.userClients, c.UserID)
			if callID, inCall := h.userCalls[c.UserID]; inCall {
				go h.endCall(callID, "User disconnected")
			}
		}
	}
	h.mu.Unlock()
	h.broadcastPresence(c.UserID, false)
}

func (h *Hub) broadcastPresence(userID string, isOnline bool) {
	msg := &WSMessage{
		Type:     TypePresenceUpdate,
		CallerID: userID,
		Reason:   fmt.Sprintf("%v", isOnline),
	}
	data, _ := json.Marshal(msg)

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, tabs := range h.userClients {
		for _, client := range tabs {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregisterClient(c)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		c.Hub.handleMessage(c, &msg)
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

func (h *Hub) sendToUser(userID string, msg *WSMessage) bool {
	h.mu.RLock()
	tabs, ok := h.userClients[userID]
	h.mu.RUnlock()

	if !ok || len(tabs) == 0 {
		return false
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}

	sent := false
	for _, client := range tabs {
		select {
		case client.Send <- data:
			sent = true
		default:
		}
	}
	return sent
}

func (h *Hub) handleMessage(client *Client, msg *WSMessage) {
	switch msg.Type {
	case TypeCallRequest:
		h.handleCallRequest(client, msg)
	case TypeCallAccept:
		h.handleCallAccept(client, msg)
	case TypeCallReject:
		h.handleCallReject(client, msg)
	case TypeCallEnd:
		h.handleCallEnd(client, msg)
	case TypeGroupCreate, TypeGroupJoin, TypeGroupLeave:
		h.handleGroupSignaling(client, msg)
	case TypeWebRTCOffer, TypeWebRTCAnswer, TypeWebRTCICECandidate:
		h.handleWebRTCSignaling(client, msg)
	case TypeChatMessage:
		h.handleChatMessage(client, msg)
	}
}

func (h *Hub) handleCallRequest(caller *Client, msg *WSMessage) {
	record, err := h.callUC.InitiateCall(caller.User, msg.ReceiverID)
	if err != nil {
		h.sendToUser(caller.UserID, &WSMessage{Type: TypeInsufficientBalance, Reason: err.Error()})
		return
	}

	h.mu.Lock()
	h.userCalls[caller.UserID] = record.ID
	h.userCalls[record.ReceiverID] = record.ID
	h.mu.Unlock()

	h.sendToUser(record.ReceiverID, &WSMessage{
		Type:       TypeIncomingCall,
		CallID:     record.ID,
		CallerID:   caller.UserID,
		ReceiverID: record.ReceiverID,
		RatePerMin: record.RatePerMin,
		Payload:    json.RawMessage(fmt.Sprintf(`{"caller_name":"%s","caller_avatar":"%s"}`, caller.User.Name, caller.User.AvatarURL)),
	})
}

func (h *Hub) handleCallAccept(modelClient *Client, msg *WSMessage) {
	record, err := h.callUC.AcceptCall(msg.CallID)
	if err != nil {
		return
	}

	session := &ActiveCallSession{
		CallID:         record.ID,
		CallerID:       record.CallerID,
		ReceiverID:     record.ReceiverID,
		RatePerMin:     record.RatePerMin,
		StartedAt:      time.Now(),
		StopTickerChan: make(chan struct{}),
	}

	h.mu.Lock()
	h.activeCalls[record.ID] = session
	h.mu.Unlock()

	activeMsg := &WSMessage{
		Type:       TypeCallActive,
		CallID:     record.ID,
		CallerID:   record.CallerID,
		ReceiverID: record.ReceiverID,
		RatePerMin: record.RatePerMin,
	}
	h.sendToUser(record.CallerID, activeMsg)
	h.sendToUser(record.ReceiverID, activeMsg)

	go h.startCallTicker(session)
}

func (h *Hub) handleCallReject(modelClient *Client, msg *WSMessage) {
	_ = h.callUC.RejectCall(msg.CallID)

	h.mu.Lock()
	delete(h.userCalls, modelClient.UserID)
	delete(h.userCalls, msg.CallerID)
	h.mu.Unlock()

	h.sendToUser(msg.CallerID, &WSMessage{Type: TypeCallRejected, CallID: msg.CallID, Reason: "Declined"})
	h.broadcastPresence(modelClient.UserID, true)
	h.broadcastPresence(msg.CallerID, true)
}

func (h *Hub) handleCallEnd(client *Client, msg *WSMessage) {
	callID := msg.CallID
	if callID == "" {
		h.mu.RLock()
		callID = h.userCalls[client.UserID]
		h.mu.RUnlock()
	}
	if callID != "" {
		h.endCall(callID, "Call ended by user")
	}
}

func (h *Hub) startCallTicker(session *ActiveCallSession) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	warningSent := false
	for {
		select {
		case <-session.StopTickerChan:
			return
		case now := <-ticker.C:
			durationSec := int(now.Sub(session.StartedAt).Seconds())

			// Update PostgreSQL heartbeat checkpoint every 10s
			if durationSec%10 == 0 {
				_ = h.callRepo.UpdateHeartbeat(session.CallID)
			}

			wallet, err := h.walletRepo.GetWallet(session.CallerID)
			if err != nil {
				h.endCall(session.CallID, "Wallet error")
				return
			}

			ratePerSec := session.RatePerMin / 60.0
			costSoFar := float64(durationSec) * ratePerSec
			remainingBalance := wallet.Balance - costSoFar
			remainingSec := int(remainingBalance / ratePerSec)
			if remainingSec < 0 {
				remainingSec = 0
			}

			tickMsg := &WSMessage{
				Type:         TypeCallTick,
				CallID:       session.CallID,
				DurationSec:  durationSec,
				RemainingSec: remainingSec,
				Cost:         costSoFar,
			}
			h.sendToUser(session.CallerID, tickMsg)
			h.sendToUser(session.ReceiverID, tickMsg)

			if remainingSec <= 30 && !warningSent {
				warningSent = true
				h.sendToUser(session.CallerID, &WSMessage{
					Type:         TypeBalanceLowWarning,
					CallID:       session.CallID,
					RemainingSec: remainingSec,
					Reason:       "Low balance warning",
				})
			}

			if remainingBalance <= 0 || remainingSec <= 0 {
				h.endCall(session.CallID, "Balance exhausted")
				return
			}
		}
	}
}

func (h *Hub) endCall(callID, reason string) {
	h.mu.Lock()
	if session, exists := h.activeCalls[callID]; exists {
		close(session.StopTickerChan)
		delete(h.activeCalls, callID)
	}
	h.mu.Unlock()

	record, err := h.callRepo.GetByID(callID)
	if err != nil {
		return
	}
	callerID := record.CallerID
	receiverID := record.ReceiverID

	cost, durationSec, _ := h.callUC.EndCall(callID, reason)

	h.mu.Lock()
	delete(h.userCalls, callerID)
	delete(h.userCalls, receiverID)
	h.mu.Unlock()

	endMsgType := TypeCallEnded
	if reason == "Balance exhausted" {
		endMsgType = TypeBalanceExhausted
	}

	endMsg := &WSMessage{
		Type:        endMsgType,
		CallID:      callID,
		CallerID:    callerID,
		ReceiverID:  receiverID,
		DurationSec: durationSec,
		Cost:        cost,
		Reason:      reason,
	}
	h.sendToUser(callerID, endMsg)
	h.sendToUser(receiverID, endMsg)

	h.broadcastPresence(callerID, true)
	h.broadcastPresence(receiverID, true)
}

func (h *Hub) handleGroupSignaling(client *Client, msg *WSMessage) {
	switch msg.Type {
	case TypeGroupCreate:
		var payload struct {
			Title      string  `json:"title"`
			RatePerMin float64 `json:"rate_per_min"`
		}
		_ = json.Unmarshal(msg.Payload, &payload)
		room, err := h.roomUC.CreateRoom(client.User, payload.Title, payload.RatePerMin)
		if err != nil {
			return
		}
		roomData, _ := json.Marshal(room)
		h.sendToUser(client.UserID, &WSMessage{Type: TypeGroupCreate, RoomID: room.ID, RatePerMin: room.RatePerMin, Payload: roomData})

	case TypeGroupJoin:
		p, err := h.roomUC.JoinRoom(msg.RoomID, client.User)
		if err != nil {
			h.sendToUser(client.UserID, &WSMessage{Type: TypeInsufficientBalance, Reason: err.Error()})
			return
		}
		pData, _ := json.Marshal(p)
		h.sendToUser(client.UserID, &WSMessage{Type: TypeGroupJoin, RoomID: msg.RoomID, Payload: pData})

	case TypeGroupLeave:
		cost, dur, _ := h.roomUC.LeaveRoom(msg.RoomID, client.UserID, "User left")
		h.sendToUser(client.UserID, &WSMessage{Type: TypeGroupLeave, RoomID: msg.RoomID, DurationSec: dur, Cost: cost})
	}
}

func (h *Hub) handleWebRTCSignaling(client *Client, msg *WSMessage) {
	if msg.ReceiverID != "" {
		h.sendToUser(msg.ReceiverID, msg)
		return
	}
	if msg.CallID != "" {
		h.mu.RLock()
		if session, ok := h.activeCalls[msg.CallID]; ok {
			targetID := session.ReceiverID
			if client.UserID == session.ReceiverID {
				targetID = session.CallerID
			}
			h.sendToUser(targetID, msg)
		}
		h.mu.RUnlock()
	}
}

func (h *Hub) handleChatMessage(sender *Client, msg *WSMessage) {
	_ = h.walletRepo.DeductChatFee(sender.UserID, msg.ReceiverID, 1.0)

	forwardMsg := &WSMessage{
		Type:       TypeChatMessage,
		CallerID:   sender.UserID,
		ReceiverID: msg.ReceiverID,
		Payload:    msg.Payload,
	}
	h.sendToUser(msg.ReceiverID, forwardMsg)

	wallet, _ := h.walletRepo.GetWallet(sender.UserID)
	balance := 0.0
	if wallet != nil {
		balance = wallet.Balance
	}
	h.sendToUser(sender.UserID, &WSMessage{
		Type: TypeChatReceived,
		Cost: balance,
	})
}
