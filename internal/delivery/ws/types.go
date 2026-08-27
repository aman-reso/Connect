package ws

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
	TypeCallError           = "CALL_ERROR"
	TypeInsufficientBalance = "CALL_INSUFFICIENT_BALANCE"
	TypeBalanceLowWarning   = "BALANCE_LOW_WARNING"
	TypeBalanceExhausted    = "CALL_ENDED_BALANCE_EXHAUSTED"
	TypeCallTick            = "CALL_TICK"
	TypePresenceUpdate      = "PRESENCE_UPDATE"
	TypeSessionTerminated   = "SESSION_TERMINATED"
	TypeGroupCreate         = "GROUP_CREATE"
	TypeGroupJoin           = "GROUP_JOIN"
	TypeGroupLeave          = "GROUP_LEAVE"
	TypeGroupUserJoined     = "GROUP_USER_JOINED"
	TypeGroupUserLeft       = "GROUP_USER_LEFT"
	TypeGroupTick           = "GROUP_TICK"
	TypeGroupKickExhausted  = "GROUP_KICK_BALANCE_EXHAUSTED"
	TypeWebRTCOffer         = "WEBRTC_OFFER"
	TypeWebRTCAnswer        = "WEBRTC_ANSWER"
	TypeWebRTCICECandidate  = "WEBRTC_ICE_CANDIDATE"
	TypeChatMessage         = "CHAT_MESSAGE"
	TypeChatReceived        = "CHAT_RECEIVED"
)

// SignalMessage is the envelope for WebSocket signaling messages supporting Android and Web clients.
type SignalMessage struct {
	Type          string  `json:"type"`
	CallID        string  `json:"call_id,omitempty"`
	RoomID        string  `json:"room_id,omitempty"`
	CallerID      string  `json:"caller_id,omitempty"`
	ReceiverID    string  `json:"receiver_id,omitempty"`
	FromUserID    string  `json:"from_user_id,omitempty"`
	ToUserID      string  `json:"to_user_id,omitempty"`
	CallType      string  `json:"call_type,omitempty"`
	SDP           any     `json:"sdp,omitempty"`
	Candidate     any     `json:"candidate,omitempty"`
	SDPMid        string  `json:"sdp_mid,omitempty"`
	SDPMLineIndex int     `json:"sdp_m_line_index,omitempty"`
	RatePerMin    float64 `json:"rate_per_min,omitempty"`
	Rate          float64 `json:"rate,omitempty"`
	DurationSec   int64   `json:"duration_sec,omitempty"`
	RemainingSec  int     `json:"remaining_sec,omitempty"`
	Cost          float64 `json:"cost,omitempty"`
	TotalCost     float64 `json:"total_cost,omitempty"`
	Balance       float64 `json:"balance,omitempty"`
	Reason        string  `json:"reason,omitempty"`
	Content       string  `json:"content,omitempty"`
	Timestamp     int64   `json:"timestamp,omitempty"`
	Payload       any     `json:"payload,omitempty"`
}

// GetTargetUserID returns receiver ID or to_user_id.
func (m *SignalMessage) GetTargetUserID() string {
	if m.ReceiverID != "" {
		return m.ReceiverID
	}
	return m.ToUserID
}

// GetSenderUserID returns caller ID or from_user_id.
func (m *SignalMessage) GetSenderUserID() string {
	if m.CallerID != "" {
		return m.CallerID
	}
	return m.FromUserID
}

// GetCallID returns the call ID.
func (m *SignalMessage) GetCallID() string {
	return m.CallID
}

