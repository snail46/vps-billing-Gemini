package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"vps-billing/internal/logger"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := string(debug.Stack())
				ctx := r.Context()
				slog.ErrorContext(ctx, "panic recovered in http handler",
					slog.Any("error", rec),
					slog.String("stack", stack),
				)

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)

				resp := map[string]any{
					"success": false,
					"error": map[string]any{
						"code":        "INTERNAL_ERROR",
						"message_key": "errors.internal_error",
						"details":     fmt.Sprintf("%v", rec),
					},
					"request_id": logger.GetRequestID(ctx),
				}
				_ = json.NewEncoder(w).Encode(resp)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
