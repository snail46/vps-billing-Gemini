package server

import (
	"net/http"
	"vps-billing/internal/httputil"
)

type APIResponse = httputil.APIResponse
type APIError = httputil.APIError

func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data any) {
	httputil.JSON(w, r, statusCode, data)
}

func Error(w http.ResponseWriter, r *http.Request, statusCode int, code, messageKey string, details any) {
	httputil.Error(w, r, statusCode, code, messageKey, details)
}
