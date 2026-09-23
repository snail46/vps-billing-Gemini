package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vps-billing/internal/config"
)

func TestHealthLive(t *testing.T) {
	handler := NewHealthHandler(nil, nil)

	req := httptest.NewRequest("GET", "/health/live", nil)
	rr := httptest.NewRecorder()

	handler.Live(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success to be true")
	}

	dataMap, ok := resp.Data.(map[string]any)
	if !ok || dataMap["status"] != "alive" {
		t.Errorf("expected status 'alive', got %v", resp.Data)
	}
}

func TestHealthReadyUnconfigured(t *testing.T) {
	handler := NewHealthHandler(nil, nil)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	rr := httptest.NewRecorder()

	handler.Ready(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable when DB is nil, got %d", rr.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dataMap, ok := resp.Data.(map[string]any)
	if !ok || dataMap["status"] != "unready" {
		t.Errorf("expected status 'unready', got %v", resp.Data)
	}
}

func TestRouterRoutes(t *testing.T) {
	cfg := &config.Config{
		UserWebOrigin:  "http://localhost:3000",
		AdminWebOrigin: "http://localhost:3001",
	}
	router := NewRouter(cfg, nil, nil)

	// Test GET /health/live
	reqLive := httptest.NewRequest("GET", "/health/live", nil)
	rrLive := httptest.NewRecorder()
	router.ServeHTTP(rrLive, reqLive)
	if rrLive.Code != http.StatusOK {
		t.Errorf("expected /health/live 200, got %d", rrLive.Code)
	}

	// Test GET /api/v1/health/live
	reqApiLive := httptest.NewRequest("GET", "/api/v1/health/live", nil)
	rrApiLive := httptest.NewRecorder()
	router.ServeHTTP(rrApiLive, reqApiLive)
	if rrApiLive.Code != http.StatusOK {
		t.Errorf("expected /api/v1/health/live 200, got %d", rrApiLive.Code)
	}

	// Test 404 for unknown endpoint
	reqNotFound := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	rrNotFound := httptest.NewRecorder()
	router.ServeHTTP(rrNotFound, reqNotFound)
	if rrNotFound.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", rrNotFound.Code)
	}
}
