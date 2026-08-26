package proto

import (
	"encoding/json"
	"net/http"
)

// SendProtoResponse sends a protobuf-compatible JSON or Binary response.
func SendProtoResponse(w http.ResponseWriter, status int, resp *ApiResponseProto) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// EncodeApiResponseProto converts generic data into an ApiResponseProto.
func EncodeApiResponseProto(status int, msg string, data any) (*ApiResponseProto, error) {
	var payload []byte
	var err error
	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	return &ApiResponseProto{
		Success:     status >= 200 && status < 300,
		StatusCode:  int32(status),
		Message:     msg,
		DataPayload: payload,
	}, nil
}
