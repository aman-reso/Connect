package common

import "time"

// ApiResponse is a generic wrapper for API responses.
type ApiResponse[T any] struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Message    string `json:"message,omitempty"`
	Data       T      `json:"data,omitempty"`
	Error      string `json:"error,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

// SuccessResponse creates a successful generic ApiResponse.
func SuccessResponse[T any](status int, msg string, data T) ApiResponse[T] {
	return ApiResponse[T]{
		Success:    true,
		StatusCode: status,
		Message:    msg,
		Data:       data,
		Timestamp:  time.Now().Unix(),
	}
}

// ErrorResponse creates a generic error response.
func ErrorResponse[T any](status int, err string) ApiResponse[T] {
	return ApiResponse[T]{
		Success:    false,
		StatusCode: status,
		Error:      err,
		Timestamp:  time.Now().Unix(),
	}
}

// PaginatedResponse is a generic wrapper for paginated lists.
type PaginatedResponse[T any] struct {
	Items      []T  `json:"items"`
	TotalCount int  `json:"total_count"`
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	HasNext    bool `json:"has_next"`
}
