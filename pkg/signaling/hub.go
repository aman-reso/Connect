package signaling

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"Connect/pkg/models"
	"Connect/pkg/store"

	"github.com/google/uuid"
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
	TypeCallRequest         = "CALL_REQUEST"
	TypeIncomingCall        = "INCOMING_CALL"
	TypeCallAccept          = "CALL_ACCEPT"
	TypeCallActive          = "CALL_ACTIVE"
	TypeCallReject          = "CALL_REJECT"
	TypeCallRejected        = "CALL_REJECTED"
	TypeCallEnd             = "CALL_END"
	TypeCallEnded           = "CALL_ENDED"
	TypeCallBusy            = "CALL_BUSY"
	TypeCallOffline         = "CALL_OFFLINE"
	TypeInsufficientBalance = "CALL_INSUFFICIENT_BALANCE"
	TypeBalanceLowWarning   = "BALANCE_LOW_WARNING"
	TypeBalanceExhausted    = "CALL_ENDED_BALANCE_EXHAUSTED"
	TypeCallTick            = "CALL_TICK"
	TypePresenceUpdate      = "PRESENCE_UPDATE"
	TypeSessionTerminated   = "SESSION_TERMINATED"

	// Group Call / Room Signaling
	TypeGroupCreate        = "GROUP_CREATE"
	TypeGroupJoin          = "GROUP_JOIN"
	TypeGroupLeave         = "GROUP_LEAVE"
	TypeGroupUserJoined    = "GROUP_USER_JOINED"
	TypeGroupUserLeft      = "GROUP_USER_LEFT"
	TypeGroupTick          = "GROUP_TICK"
	TypeGroupKickExhausted = "GROUP_KICK_BALANCE_EXHAUSTED"
	TypeGroupClosed        = "GROUP_CLOSED"

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
	User   *models.User
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

type ActiveGroupSession struct {
	RoomID         string
	HostID         string
	RatePerMin     float64
	StopTickerChan chan struct{}
}

type Hub struct {
	store        *store.Store
	mu           sync.RWMutex
	userClients  map[string]map[string]*Client  // userID -> map[clientID]*Client (multi-tab support)
	activeCalls  map[string]*ActiveCallSession  // callID -> ActiveCallSession
	activeGroups map[string]*ActiveGroupSession // roomID -> ActiveGroupSession
	userCalls    map[string]string              // userID -> callID
	userRooms    map[string]string              // userID -> roomID
}

func NewHub(st *store.Store) *Hub {
	return &Hub{
		store:        st,
		userClients:  make(map[string]map[string]*Client),
		activeCalls:  make(map[string]*ActiveCallSession),
		activeGroups: make(map[string]*ActiveGroupSession),
		userCalls:    make(map[string]string),
		userRooms:    make(map[string]string),
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
		return
	}

	user, err := h.store.GetUserByToken(token)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket Upgrade Error: %v", err)
		return
	}

	clientID := uuid.New().String()
	client := &Client{
		ID:     clientID,
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

	// 1. Single Active Device Enforcement: Evict any existing session on another device
	if tabs, ok := h.userClients[c.UserID]; ok {
		for id, oldClient := range tabs {
			termMsg := &WSMessage{
				Type:   TypeSessionTerminated,
				Reason: "You have been logged in from a new device/session. This previous session has been closed.",
			}
			data, _ := json.Marshal(termMsg)
			select {
			case oldClient.Send <- data:
			default:
			}
			oldClient.Conn.Close()
			delete(tabs, id)
		}
	} else {
		h.userClients[c.UserID] = make(map[string]*Client)
	}

	h.userClients[c.UserID][c.ID] = c
	h.store.SetUserPresence(c.UserID, true, false)
	h.mu.Unlock()

	log.Printf("Client connected: %s (%s) [Session ID: %s]", c.User.Name, c.UserID, c.ID[:8])
	h.broadcastPresence(c.UserID, true)
}

func (h *Hub) unregisterClient(c *Client) {
	h.mu.Lock()
	if tabs, ok := h.userClients[c.UserID]; ok {
		delete(tabs, c.ID)
		if len(tabs) == 0 {
			delete(h.userClients, c.UserID)
			h.store.SetUserPresence(c.UserID, false, false)

			// If last tab closed during call, terminate call
			if callID, inCall := h.userCalls[c.UserID]; inCall {
				go h.endCall(callID, "User disconnected", c.UserID)
			}
			if roomID, inRoom := h.userRooms[c.UserID]; inRoom {
				go h.handleLeaveGroupRoom(c.UserID, roomID, "User disconnected")
			}
		}
	}
	h.mu.Unlock()

	log.Printf("Client tab disconnected: %s (%s)", c.User.Name, c.UserID)
	h.broadcastPresence(c.UserID, h.isUserOnline(c.UserID))
}

func (h *Hub) isUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	tabs, ok := h.userClients[userID]
	return ok && len(tabs) > 0
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

	c.Conn.SetReadLimit(1024 * 1024)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

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
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
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

	// Group Calls
	case TypeGroupCreate:
		h.handleCreateGroupRoom(client, msg)
	case TypeGroupJoin:
		h.handleJoinGroupRoom(client, msg)
	case TypeGroupLeave:
		h.handleLeaveGroupRoom(client.UserID, msg.RoomID, "User left group room")

	// WebRTC Forwarding
	case TypeWebRTCOffer, TypeWebRTCAnswer, TypeWebRTCICECandidate:
		h.handleWebRTCSignaling(client, msg)

	case TypeChatMessage:
		h.handleChatMessage(client, msg)
	}
}

// 1-ON-1 CALL HANDLERS
func (h *Hub) handleCallRequest(caller *Client, msg *WSMessage) {
	receiverID := msg.ReceiverID
	receiver, err := h.store.GetUserByID(receiverID)
	if err != nil {
		h.sendToUser(caller.UserID, &WSMessage{
			Type:   TypeCallOffline,
			Reason: "Model not found",
		})
		return
	}

	// 1. Check Caller Balance (Must have at least 1 minute of balance)
	wallet, err := h.store.GetWallet(caller.UserID)
	if err != nil || wallet.Balance < receiver.VoiceRatePerMin {
		h.sendToUser(caller.UserID, &WSMessage{
			Type:   TypeInsufficientBalance,
			Reason: fmt.Sprintf("Insufficient balance. Minimum required for 1 min is ₹%.2f. Current balance: ₹%.2f", receiver.VoiceRatePerMin, wallet.Balance),
		})
		return
	}

	// 2. Check if Receiver is Online & Available
	h.mu.RLock()
	tabs, isOnline := h.userClients[receiverID]
	_, isBusy := h.userCalls[receiverID]
	h.mu.RUnlock()

	if !isOnline || len(tabs) == 0 {
		h.sendToUser(caller.UserID, &WSMessage{
			Type:   TypeCallOffline,
			Reason: fmt.Sprintf("%s is currently offline (Open Model tab in another window)", receiver.Name),
		})
		return
	}

	if isBusy || receiver.IsBusy {
		h.sendToUser(caller.UserID, &WSMessage{
			Type:   TypeCallBusy,
			Reason: fmt.Sprintf("%s is currently busy on another call", receiver.Name),
		})
		return
	}

	callID := "call_" + uuid.New().String()[:8]
	record := &models.CallRecord{
		ID:           callID,
		CallerID:     caller.UserID,
		CallerName:   caller.User.Name,
		ReceiverID:   receiverID,
		ReceiverName: receiver.Name,
		CallType:     "voice",
		Status:       models.CallStatusRinging,
		RatePerMin:   receiver.VoiceRatePerMin,
		CreatedAt:    time.Now(),
	}
	h.store.CreateCallRecord(record)

	h.mu.Lock()
	h.userCalls[caller.UserID] = callID
	h.userCalls[receiverID] = callID
	h.store.SetUserPresence(receiverID, true, true)
	h.mu.Unlock()

	h.sendToUser(receiverID, &WSMessage{
		Type:       TypeIncomingCall,
		CallID:     callID,
		CallerID:   caller.UserID,
		ReceiverID: receiverID,
		RatePerMin: receiver.VoiceRatePerMin,
		Payload:    json.RawMessage(fmt.Sprintf(`{"caller_name":"%s","caller_avatar":"%s"}`, caller.User.Name, caller.User.AvatarURL)),
	})
}

func (h *Hub) handleCallAccept(modelClient *Client, msg *WSMessage) {
	callID := msg.CallID
	record, err := h.store.GetCallRecord(callID)
	if err != nil || record.Status != models.CallStatusRinging {
		return
	}

	now := time.Now()
	record.Status = models.CallStatusActive
	record.StartedAt = &now
	h.store.UpdateCallRecord(record)

	session := &ActiveCallSession{
		CallID:         callID,
		CallerID:       record.CallerID,
		ReceiverID:     record.ReceiverID,
		RatePerMin:     record.RatePerMin,
		StartedAt:      now,
		StopTickerChan: make(chan struct{}),
	}

	h.mu.Lock()
	h.activeCalls[callID] = session
	h.mu.Unlock()

	activeMsg := &WSMessage{
		Type:       TypeCallActive,
		CallID:     callID,
		CallerID:   record.CallerID,
		ReceiverID: record.ReceiverID,
		RatePerMin: record.RatePerMin,
	}
	h.sendToUser(record.CallerID, activeMsg)
	h.sendToUser(record.ReceiverID, activeMsg)

	go h.startCallTicker(session)
}

func (h *Hub) handleCallReject(modelClient *Client, msg *WSMessage) {
	callID := msg.CallID
	record, err := h.store.GetCallRecord(callID)
	if err != nil {
		return
	}

	record.Status = models.CallStatusRejected
	record.EndReason = "Model declined the call"
	h.store.UpdateCallRecord(record)

	h.mu.Lock()
	delete(h.userCalls, record.CallerID)
	delete(h.userCalls, record.ReceiverID)
	h.store.SetUserPresence(record.ReceiverID, true, false)
	h.mu.Unlock()

	h.sendToUser(record.CallerID, &WSMessage{
		Type:   TypeCallRejected,
		CallID: callID,
		Reason: "Call was declined",
	})
}

func (h *Hub) handleCallEnd(client *Client, msg *WSMessage) {
	callID := msg.CallID
	if callID == "" {
		h.mu.RLock()
		callID = h.userCalls[client.UserID]
		h.mu.RUnlock()
	}
	if callID != "" {
		h.endCall(callID, "Call ended by user", client.UserID)
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

			wallet, err := h.store.GetWallet(session.CallerID)
			if err != nil {
				h.endCall(session.CallID, "Wallet error", "system")
				return
			}

			ratePerSec := session.RatePerMin / 60.0
			costSoFar := float64(durationSec) * ratePerSec
			remainingBalance := wallet.Balance - costSoFar

			remainingSec := max(int(remainingBalance/ratePerSec), 0)

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
					Reason:       "Low balance! Call will auto-disconnect in less than 30 seconds.",
				})
			}

			if remainingBalance <= 0 || remainingSec <= 0 {
				h.endCall(session.CallID, "Balance exhausted", "billing_engine")
				return
			}
		}
	}
}

func (h *Hub) endCall(callID, reason, triggeredBy string) {
	h.mu.Lock()
	session, exists := h.activeCalls[callID]
	if exists {
		close(session.StopTickerChan)
		delete(h.activeCalls, callID)
	}

	record, err := h.store.GetCallRecord(callID)
	if err != nil {
		h.mu.Unlock()
		return
	}

	delete(h.userCalls, record.CallerID)
	delete(h.userCalls, record.ReceiverID)
	h.store.SetUserPresence(record.ReceiverID, true, false)
	h.mu.Unlock()

	now := time.Now()
	record.EndedAt = &now
	durationSec := 0
	if record.StartedAt != nil {
		durationSec = int(now.Sub(*record.StartedAt).Seconds())
	}
	record.DurationSeconds = durationSec

	if reason == "Balance exhausted" {
		record.Status = models.CallStatusExhausted
	} else if record.Status == models.CallStatusActive {
		record.Status = models.CallStatusCompleted
	}
	record.EndReason = reason

	cost, _ := h.store.ProcessCallFinancials(record.CallerID, record.ReceiverID, callID, durationSec, record.RatePerMin, reason)
	record.TotalCost = cost
	h.store.UpdateCallRecord(record)

	endMsgType := TypeCallEnded
	if reason == "Balance exhausted" {
		endMsgType = TypeBalanceExhausted
	}

	endMsg := &WSMessage{
		Type:        endMsgType,
		CallID:      callID,
		DurationSec: durationSec,
		Cost:        cost,
		Reason:      reason,
	}
	h.sendToUser(record.CallerID, endMsg)
	h.sendToUser(record.ReceiverID, endMsg)
}

// -------------------------------------------------------------
// GROUP CALL / MULTI-PARTICIPANT AUDIO ROOM HANDLERS
// -------------------------------------------------------------

func (h *Hub) handleCreateGroupRoom(hostClient *Client, msg *WSMessage) {
	var payload struct {
		Title      string  `json:"title"`
		RatePerMin float64 `json:"rate_per_min"`
	}
	_ = json.Unmarshal(msg.Payload, &payload)
	if payload.Title == "" {
		payload.Title = hostClient.User.Name + "'s Audio Lounge"
	}

	room, err := h.store.CreateGroupRoom(hostClient.User, payload.Title, payload.RatePerMin)
	if err != nil {
		return
	}

	session := &ActiveGroupSession{
		RoomID:         room.ID,
		HostID:         hostClient.UserID,
		RatePerMin:     room.RatePerMin,
		StopTickerChan: make(chan struct{}),
	}

	h.mu.Lock()
	h.activeGroups[room.ID] = session
	h.userRooms[hostClient.UserID] = room.ID
	h.mu.Unlock()

	roomData, _ := json.Marshal(room)
	h.sendToUser(hostClient.UserID, &WSMessage{
		Type:       TypeGroupCreate,
		RoomID:     room.ID,
		RatePerMin: room.RatePerMin,
		Payload:    roomData,
	})

	go h.startGroupRoomTicker(session)
}

func (h *Hub) handleJoinGroupRoom(client *Client, msg *WSMessage) {
	roomID := msg.RoomID
	participant, err := h.store.JoinGroupRoom(roomID, client.User)
	if err != nil {
		h.sendToUser(client.UserID, &WSMessage{
			Type:   TypeInsufficientBalance,
			Reason: err.Error(),
		})
		return
	}

	h.mu.Lock()
	h.userRooms[client.UserID] = roomID
	h.mu.Unlock()

	room, _ := h.store.GetGroupRoom(roomID)
	roomData, _ := json.Marshal(room)

	h.sendToUser(client.UserID, &WSMessage{
		Type:       TypeGroupJoin,
		RoomID:     roomID,
		RatePerMin: room.RatePerMin,
		Payload:    roomData,
	})

	participantData, _ := json.Marshal(participant)
	h.broadcastToRoom(roomID, client.UserID, &WSMessage{
		Type:    TypeGroupUserJoined,
		RoomID:  roomID,
		Payload: participantData,
	})
}

func (h *Hub) handleLeaveGroupRoom(userID, roomID, reason string) {
	if roomID == "" {
		h.mu.RLock()
		roomID = h.userRooms[userID]
		h.mu.RUnlock()
	}
	if roomID == "" {
		return
	}

	cost, durationSec, err := h.store.LeaveGroupRoom(roomID, userID, reason)
	if err != nil {
		return
	}

	h.mu.Lock()
	delete(h.userRooms, userID)
	h.mu.Unlock()

	h.sendToUser(userID, &WSMessage{
		Type:        TypeGroupLeave,
		RoomID:      roomID,
		DurationSec: durationSec,
		Cost:        cost,
		Reason:      reason,
	})

	h.broadcastToRoom(roomID, userID, &WSMessage{
		Type:     TypeGroupUserLeft,
		RoomID:   roomID,
		CallerID: userID,
		Reason:   reason,
	})
}

func (h *Hub) startGroupRoomTicker(session *ActiveGroupSession) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-session.StopTickerChan:
			return
		case now := <-ticker.C:
			room, err := h.store.GetGroupRoom(session.RoomID)
			if err != nil || !room.IsActive {
				return
			}

			ratePerSec := session.RatePerMin / 60.0

			for userID, p := range room.Participants {
				if p.IsHost {
					continue
				}

				durationSec := int(now.Sub(p.JoinedAt).Seconds())
				wallet, err := h.store.GetWallet(userID)
				if err != nil {
					continue
				}

				costSoFar := float64(durationSec) * ratePerSec
				remainingBalance := wallet.Balance - costSoFar

				remainingSec := int(remainingBalance / ratePerSec)
				if remainingSec < 0 {
					remainingSec = 0
				}

				h.sendToUser(userID, &WSMessage{
					Type:         TypeGroupTick,
					RoomID:       session.RoomID,
					DurationSec:  durationSec,
					RemainingSec: remainingSec,
					Cost:         costSoFar,
				})

				if remainingBalance <= 0 || remainingSec <= 0 {
					go func(uID, rID string) {
						h.sendToUser(uID, &WSMessage{
							Type:   TypeGroupKickExhausted,
							RoomID: rID,
							Reason: "Your wallet balance was exhausted. You have been removed from the group audio room.",
						})
						h.handleLeaveGroupRoom(uID, rID, "Balance exhausted in group call")
					}(userID, session.RoomID)
				}
			}
		}
	}
}

func (h *Hub) broadcastToRoom(roomID string, excludeUserID string, msg *WSMessage) {
	room, err := h.store.GetGroupRoom(roomID)
	if err != nil {
		return
	}

	for uid := range room.Participants {
		if uid != excludeUserID {
			h.sendToUser(uid, msg)
		}
	}
}

func (h *Hub) handleWebRTCSignaling(client *Client, msg *WSMessage) {
	if msg.ReceiverID != "" {
		h.sendToUser(msg.ReceiverID, msg)
		return
	}

	if msg.RoomID != "" {
		h.broadcastToRoom(msg.RoomID, client.UserID, msg)
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
	var chatContent struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Payload, &chatContent); err != nil {
		return
	}

	if msg.RoomID != "" {
		ephemeralMsg := &models.EphemeralMessage{
			ID:        "msg_" + uuid.New().String()[:8],
			SenderID:  sender.UserID,
			RoomID:    msg.RoomID,
			Content:   chatContent.Text,
			Cost:      0,
			ExpiresAt: time.Now().Add(60 * time.Second),
			CreatedAt: time.Now(),
		}
		h.store.SaveEphemeralMessage(ephemeralMsg)
		payloadBytes, _ := json.Marshal(ephemeralMsg)

		h.broadcastToRoom(msg.RoomID, "", &WSMessage{
			Type:    TypeChatMessage,
			RoomID:  msg.RoomID,
			Payload: payloadBytes,
		})
		return
	}

	receiverID := msg.ReceiverID
	receiver, err := h.store.GetUserByID(receiverID)
	if err != nil {
		return
	}

	msgCost := 0.0
	if sender.User.Role == models.RoleUser {
		msgCost = receiver.ChatRatePerMsg
		if err := h.store.DeductChatFee(sender.UserID, receiverID, msgCost); err != nil {
			h.sendToUser(sender.UserID, &WSMessage{
				Type:   TypeChatError,
				Reason: "Insufficient balance to send message (Requires ₹" + fmt.Sprintf("%.2f", msgCost) + ")",
			})
			return
		}
	}

	ephemeralMsg := &models.EphemeralMessage{
		ID:         "msg_" + uuid.New().String()[:8],
		SenderID:   sender.UserID,
		ReceiverID: receiverID,
		Content:    chatContent.Text,
		Cost:       msgCost,
		ExpiresAt:  time.Now().Add(60 * time.Second),
		CreatedAt:  time.Now(),
	}
	h.store.SaveEphemeralMessage(ephemeralMsg)

	payloadBytes, _ := json.Marshal(ephemeralMsg)

	h.sendToUser(receiverID, &WSMessage{
		Type:       TypeChatMessage,
		ReceiverID: receiverID,
		Payload:    payloadBytes,
	})

	wallet, _ := h.store.GetWallet(sender.UserID)
	h.sendToUser(sender.UserID, &WSMessage{
		Type:    TypeChatReceived,
		Payload: payloadBytes,
		Cost:    wallet.Balance,
	})
}
