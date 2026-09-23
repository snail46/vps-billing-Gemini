package httputil

import (
	"encoding/json"
	"net/http"

	"vps-billing/internal/logger"
)

type APIResponse struct {
	Success   bool      `json:"success"`
	Data      any       `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
	RequestID string    `json:"request_id"`
}

type APIError struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
	Details    any    `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	resp := APIResponse{
		Success:   true,
		Data:      data,
		RequestID: logger.GetRequestID(r.Context()),
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func Error(w http.ResponseWriter, r *http.Request, statusCode int, code, messageKey string, details any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	resp := APIResponse{
		Success: false,
		Error: &APIError{
			Code:       code,
			MessageKey: messageKey,
			Details:    details,
		},
		RequestID: logger.GetRequestID(r.Context()),
	}

	_ = json.NewEncoder(w).Encode(resp)
}
