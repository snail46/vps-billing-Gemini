package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const (
	RequestIDKey   contextKey = "request_id"
	TraceIDKey     contextKey = "trace_id"
	ActorKey       contextKey = "actor"
	ActionKey      contextKey = "action"
	ResourceKey    contextKey = "resource"
	OperationIDKey contextKey = "operation_id"
	ProviderKey    contextKey = "provider"
	NodeKey        contextKey = "node"
)

type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
			r.AddAttrs(slog.String("request_id", reqID))
		}
		if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
			r.AddAttrs(slog.String("trace_id", traceID))
		}
		if actor, ok := ctx.Value(ActorKey).(string); ok && actor != "" {
			r.AddAttrs(slog.String("actor", actor))
		}
		if action, ok := ctx.Value(ActionKey).(string); ok && action != "" {
			r.AddAttrs(slog.String("action", action))
		}
		if res, ok := ctx.Value(ResourceKey).(string); ok && res != "" {
			r.AddAttrs(slog.String("resource", res))
		}
		if opID, ok := ctx.Value(OperationIDKey).(string); ok && opID != "" {
			r.AddAttrs(slog.String("operation_id", opID))
		}
		if prov, ok := ctx.Value(ProviderKey).(string); ok && prov != "" {
			r.AddAttrs(slog.String("provider", prov))
		}
		if node, ok := ctx.Value(NodeKey).(string); ok && node != "" {
			r.AddAttrs(slog.String("node", node))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func Init(levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	baseHandler := slog.NewJSONHandler(os.Stdout, opts)
	ctxHandler := &ContextHandler{Handler: baseHandler}
	logger := slog.New(ctxHandler)
	slog.SetDefault(logger)

	return logger
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDKey).(string); ok {
		return val
	}
	return ""
}

func GetTraceID(ctx context.Context) string {
	if val, ok := ctx.Value(TraceIDKey).(string); ok {
		return val
	}
	return ""
}
