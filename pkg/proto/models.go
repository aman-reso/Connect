package proto

// SignalMessageProto represents real-time signaling payload.
type SignalMessageProto struct {
	Type          string  `json:"type"`
	FromUserID    string  `json:"from_user_id,omitempty"`
	ToUserID      string  `json:"to_user_id,omitempty"`
	RoomID        string  `json:"room_id,omitempty"`
	CallType      string  `json:"call_type,omitempty"`
	SDP           string  `json:"sdp,omitempty"`
	Candidate     string  `json:"candidate,omitempty"`
	SDPMid        string  `json:"sdp_mid,omitempty"`
	SDPMLineIndex int32   `json:"sdp_m_line_index,omitempty"`
	Reason        string  `json:"reason,omitempty"`
	Content       string  `json:"content,omitempty"`
	Timestamp     int64   `json:"timestamp,omitempty"`
	Balance       float64 `json:"balance,omitempty"`
	Rate          float64 `json:"rate,omitempty"`
	DurationSec   int64   `json:"duration_sec,omitempty"`
	TotalCost     float64 `json:"total_cost,omitempty"`
}

// ApiResponseProto is the protobuf representation of API responses.
type ApiResponseProto struct {
	Success     bool   `json:"success"`
	StatusCode  int32  `json:"status_code"`
	Message     string `json:"message,omitempty"`
	DataPayload []byte `json:"data_payload,omitempty"`
	Error       string `json:"error,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}
