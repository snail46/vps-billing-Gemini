package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMemoryRateLimiter(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	ctx := context.Background()
	key := "test-ip:127.0.0.1"

	// Limit to 3 requests per 200ms
	limit := 3
	window := 200 * time.Millisecond

	for i := 1; i <= limit; i++ {
		allowed, err := limiter.Allow(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should have been allowed", i)
		}
	}

	// 4th request should be rejected
	allowed, err := limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatalf("request 4 should have been rejected")
	}

	// Wait for window to expire
	time.Sleep(250 * time.Millisecond)

	// Now it should be allowed again
	allowed, err = limiter.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatalf("request after window should be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limit := 2
	window := 100 * time.Millisecond

	handler := RateLimit(limiter, limit, window)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	// 1st request -> 200
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}

	// 2nd request -> 200
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	// 3rd request -> 429
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req)
	if rec3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec3.Code)
	}
	if rec3.Header().Get("Retry-After") == "" {
		t.Errorf("expected Retry-After header to be set")
	}
}
