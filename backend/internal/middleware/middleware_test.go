package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vps-billing/internal/logger"
)

func TestRequestIDMiddleware(t *testing.T) {
	var capturedReqID, capturedTraceID string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReqID = logger.GetRequestID(r.Context())
		capturedTraceID = logger.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Without pre-existing headers
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	respReqID := rr.Header().Get("X-Request-ID")
	respTraceID := rr.Header().Get("X-Trace-ID")

	if respReqID == "" || respTraceID == "" {
		t.Errorf("expected generated X-Request-ID and X-Trace-ID")
	}
	if capturedReqID != respReqID {
		t.Errorf("context reqID %s != header reqID %s", capturedReqID, respReqID)
	}
	if capturedTraceID != respTraceID {
		t.Errorf("context traceID %s != header traceID %s", capturedTraceID, respTraceID)
	}

	// Case 2: Propagate existing headers
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Request-ID", "custom-req-123")
	req2.Header.Set("X-Trace-ID", "custom-trace-456")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Header().Get("X-Request-ID") != "custom-req-123" {
		t.Errorf("expected propagated X-Request-ID, got %s", rr2.Header().Get("X-Request-ID"))
	}
	if rr2.Header().Get("X-Trace-ID") != "custom-trace-456" {
		t.Errorf("expected propagated X-Trace-ID, got %s", rr2.Header().Get("X-Trace-ID"))
	}
}

func TestCORSMiddleware(t *testing.T) {
	corsHandler := CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Preflight OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for OPTIONS, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin to be set")
	}

	// Disallowed origin
	reqBad := httptest.NewRequest("GET", "/test", nil)
	reqBad.Header.Set("Origin", "http://evil.com")
	rrBad := httptest.NewRecorder()
	corsHandler.ServeHTTP(rrBad, reqBad)

	if rrBad.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected Access-Control-Allow-Origin to be empty for disallowed origin")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 Internal Server Error, got %d", rr.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["success"] != false {
		t.Errorf("expected success: false")
	}
}
