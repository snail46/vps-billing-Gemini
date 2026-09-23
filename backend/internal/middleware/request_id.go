package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"vps-billing/internal/logger"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			if id, err := uuid.NewV7(); err == nil {
				reqID = id.String()
			} else {
				reqID = uuid.New().String()
			}
		}

		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			if id, err := uuid.NewV7(); err == nil {
				traceID = id.String()
			} else {
				traceID = uuid.New().String()
			}
		}

		w.Header().Set("X-Request-ID", reqID)
		w.Header().Set("X-Trace-ID", traceID)

		ctx := r.Context()
		ctx = logger.WithRequestID(ctx, reqID)
		ctx = logger.WithTraceID(ctx, traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
