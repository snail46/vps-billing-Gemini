package logger

import (
	"context"
	"testing"
)

func TestLoggerContextValues(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithTraceID(ctx, "trace-456")

	if GetRequestID(ctx) != "req-123" {
		t.Errorf("expected req-123, got %s", GetRequestID(ctx))
	}
	if GetTraceID(ctx) != "trace-456" {
		t.Errorf("expected trace-456, got %s", GetTraceID(ctx))
	}

	l := Init("debug")
	if l == nil {
		t.Errorf("expected initialized logger")
	}
}
