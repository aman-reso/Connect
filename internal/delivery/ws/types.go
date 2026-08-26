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

// SignalMessage is the envelope for WebSocket signaling messages.
type SignalMessage struct {
	Type          string  `json:"type"`
	FromUserID    string  `json:"from_user_id,omitempty"`
	ToUserID      string  `json:"to_user_id,omitempty"`
	RoomID        string  `json:"room_id,omitempty"`
	CallType      string  `json:"call_type,omitempty"`
	SDP           any     `json:"sdp,omitempty"`
	Candidate     any     `json:"candidate,omitempty"`
	SDPMid        string  `json:"sdp_mid,omitempty"`
	SDPMLineIndex int     `json:"sdp_m_line_index,omitempty"`
	Reason        string  `json:"reason,omitempty"`
	Content       string  `json:"content,omitempty"`
	Timestamp     int64   `json:"timestamp,omitempty"`
	Balance       float64 `json:"balance,omitempty"`
	Rate          float64 `json:"rate,omitempty"`
	DurationSec   int64   `json:"duration_sec,omitempty"`
	TotalCost     float64 `json:"total_cost,omitempty"`
}
