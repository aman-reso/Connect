package http

import (
	"encoding/json"
	"net/http"

	"Connect/pkg/common"
	"Connect/pkg/proto"
)

// SendJSON writes a generic ApiResponse to the client.
func SendJSON[T any](w http.ResponseWriter, status int, msg string, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := common.SuccessResponse(status, msg, data)
	_ = json.NewEncoder(w).Encode(resp)
}

// SendError writes a generic error response.
func SendError(w http.ResponseWriter, status int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := common.ErrorResponse[any](status, errMsg)
	_ = json.NewEncoder(w).Encode(resp)
}

// SendProto writes a protobuf-compatible response.
func SendProto(w http.ResponseWriter, status int, msg string, data any) {
	pResp, err := proto.EncodeApiResponseProto(status, msg, data)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Protobuf encoding error")
		return
	}
	proto.SendProtoResponse(w, status, pResp)
}
