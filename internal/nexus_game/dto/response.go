package dto

import "time"

type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	Meta    *APIMeta  `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIMeta struct {
	Timestamp string `json:"timestamp"`
}

func OK(data any) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Meta:    &APIMeta{Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
}

func Fail(code string, message string) APIResponse {
	return APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
		Meta:    &APIMeta{Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
}
