package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"vps-billing/internal/logger"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriterWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)
		ctx := r.Context()

		slog.LogAttrs(
			ctx,
			slog.LevelInfo,
			"http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("status", wrapper.statusCode),
			slog.Int64("latency_ms", duration.Milliseconds()),
			slog.Int("bytes", wrapper.bytesWritten),
			slog.String("request_id", logger.GetRequestID(ctx)),
			slog.String("trace_id", logger.GetTraceID(ctx)),
		)
	})
}
